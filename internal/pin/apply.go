package pin

import (
	"crypto/sha1" //nolint:gosec // 요청 id를 만드는 비암호 해시다. upstream 과 같은 값을 내려고 sha1을 쓴다.
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func itoa(n int) string { return strconv.Itoa(n) }

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// pyRepr는 파이썬의 repr(str)과 같은 글자를 낸다. 거절 메시지가 upstream 과 같은
// 글이 되게 하려는 것이다.
func pyRepr(s string) string {
	quote := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		case !unicode.IsPrint(r) && r > 0x7f:
			if r > 0xffff {
				fmt.Fprintf(&b, `\U%08x`, r)
			} else {
				fmt.Fprintf(&b, `\u%04x`, r)
			}
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

func pyList(items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = pyRepr(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func pyTuple(items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = pyRepr(s)
	}
	if len(parts) == 1 {
		return "(" + parts[0] + ",)"
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func has(o obj, key string) bool { _, ok := o[key]; return ok }

// Result는 요청 하나를 적용한 결과다. 로그(`pin/log/`)의 한 줄과 같은 내용이다.
type Result struct {
	ID      string `json:"id"`
	Result  string `json:"result"` // applied, rejected, skipped
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Item    string `json:"item,omitempty"`
	Noop    bool   `json:"noop,omitempty"`
}

type outcome struct {
	item string // 없으면 빈 글
	msg  string
	noop bool
}

const dayOfAtErr = "bad_request"

func dayOf(req obj) string {
	d, _ := dateOfAt(req["at"])
	return d
}

// ---------------------------------------------------------------------------
// 요청 형식
// ---------------------------------------------------------------------------

func validateRequest(raw any) (obj, *Reject) {
	req, ok := raw.(map[string]any)
	if !ok {
		return nil, reject("bad_request", "변경 요청이 객체가 아닙니다")
	}
	allowed := setOf(requestKeys...)
	var extra []string
	for k := range req {
		if !allowed[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return nil, reject("bad_request", "알 수 없는 칸: %s", pyList(extra))
	}
	for _, key := range []string{"id", "at", "via", "op"} {
		if s, isS := req[key].(string); !isS || s == "" {
			return nil, reject("bad_request", "%s가 없습니다", key)
		}
	}
	if c := req["client"]; c != nil {
		cs, isS := c.(string)
		if !(str(req["via"]) == "web" && isS && clientRe.MatchString(cs)) {
			return nil, reject("bad_request", "client는 화면 요청에만 쓰고 영문, 숫자, _, - 8~64자여야 합니다")
		}
	}
	if _, err := dateOfAt(req["at"]); err != nil {
		return nil, reject("bad_request", "%s", err.Error())
	}
	via, op := str(req["via"]), str(req["op"])
	if !contains(Vias, via) {
		return nil, reject("bad_via", "via %s는 %s 중 하나여야 합니다", pyRepr(via), pyTuple(Vias))
	}
	if !opsByVia[via][op] {
		return nil, reject("op_not_allowed", "%s는 op %s를 낼 수 없습니다", via, op)
	}
	if a, ok := req["args"]; ok {
		if _, isObj := a.(map[string]any); !isObj {
			return nil, reject("bad_request", "args가 객체가 아닙니다")
		}
	}
	if it := req["item"]; it != nil && !isStr(it) {
		return nil, reject("bad_request", "item이 글이 아닙니다")
	}
	if b := req["base_rev"]; b != nil && !isInt(b) {
		return nil, reject("bad_request", "base_rev가 정수가 아닙니다")
	}
	if itemRequiredOps[op] && !truthy(req["item"]) {
		return nil, reject("bad_request", "op %s에는 item이 필요합니다", op)
	}
	return req, nil
}

// ---------------------------------------------------------------------------
// 원장 조작 도우미
// ---------------------------------------------------------------------------

func bumpRecord(item obj, day, kind string) {
	last := asObj(item["last_record"])
	if last["date"] == nil || day >= str(last["date"]) {
		item["last_record"] = obj{"date": day, "kind": kind}
	}
}

func (l *Ledger) changed(item obj) {
	prevRev, _ := toInt(item["rev"])
	item["rev"] = prevRev + 1
	prev := asObj(item["rev_by"])
	run := l.actingVia == "web" && l.actingClient != nil && prev != nil && str(prev["via"]) == "web" && prev["client"] == l.actingClient
	from := prevRev
	if run {
		from, _ = toInt(prev["from_rev"])
	}
	item["rev_by"] = obj{"via": l.actingVia, "client": l.actingClient, "from_rev": from}
	l.mark(str(item["id"]))
}

func statusLabel(status string, wait obj) string {
	if status == "waiting" && wait != nil {
		return statusLabels[status] + "(" + waitLabels[str(wait["reason"])] + ")"
	}
	return statusLabels[status]
}

func ownerWord(owner obj) string {
	switch who := str(owner["who"]); who {
	case "me":
		return "나"
	case "unknown":
		return "모름"
	default:
		return who
	}
}

func (l *Ledger) workLabel(work string) string {
	for _, w := range asList(l.config["works"]) {
		if wo := asObj(w); str(wo["id"]) == work {
			return str(wo["name"])
		}
	}
	return "미분류"
}

// recordChange는 값이 실제로 바뀐 때만 부른다. from, to는 그때의 화면 글이라 프로젝트 이름이
// 나중에 바뀌어도 이미 쓴 줄은 그대로다.
func recordChange(item, req obj, field string, before, after any, note any) {
	if before == after {
		return
	}
	entry := obj{"date": dayOf(req), "field": field, "from": before, "to": after, "note": note, "via": req["via"]}
	list, _ := item["changes"].([]any)
	item["changes"] = append(list, entry)
}

func lock(item obj, field string) {
	locked := strList(item["locked"])
	if contains(locked, field) {
		return
	}
	locked = append(locked, field)
	sort.Strings(locked)
	item["locked"] = toAnyList(locked)
}

func isLocked(item obj, field string) bool {
	locked := strList(item["locked"])
	return contains(locked, field) || (strings.HasPrefix(field, "context.") && contains(locked, "context"))
}

func (l *Ledger) requireItem(req obj) (obj, *Reject) {
	id := str(req["item"])
	if it, ok := l.items[id]; ok && isStr(req["item"]) {
		return it, nil
	}
	if _, ok := l.archived[id]; ok {
		return nil, reject("archived", "%s 일은 보관되었습니다", id)
	}
	return nil, reject("no_item", "%s 일이 없습니다", id)
}

func checkRev(item, req obj) *Reject {
	base := req["base_rev"]
	if base == nil {
		if str(req["via"]) == "web" && baseRevOps[str(req["op"])] {
			return reject("base_rev_missing", "화면 요청에는 base_rev가 필요합니다")
		}
		return nil
	}
	b, _ := toInt(base)
	rev, _ := toInt(item["rev"])
	if b != rev && !sameScreenRun(item, req, b) {
		return reject("rev_conflict", "%s 일이 그 사이 변경되었습니다(화면 rev %d, 원장 rev %d)", str(item["id"]), b, rev)
	}
	return nil
}

// sameScreenRun은 base_rev 뒤로 rev를 올린 요청이 모두 이 요청과 같은 화면(client)에서
// 온 것이면 충돌이 아니라는 예외다.
func sameScreenRun(item, req obj, base int) bool {
	client := req["client"]
	by := asObj(item["rev_by"])
	if str(req["via"]) != "web" || client == nil || by == nil || str(by["via"]) != "web" || by["client"] != client {
		return false
	}
	from, _ := toInt(by["from_rev"])
	rev, _ := toInt(item["rev"])
	return from <= base && base < rev
}

func cleanOwner(raw any, human bool) (obj, *Reject) {
	ro := asObj(raw)
	if ro == nil || !isStr(ro["who"]) || strings.TrimSpace(str(ro["who"])) == "" {
		return nil, reject("bad_args", "owner에 who가 필요합니다")
	}
	basis := "stated"
	if !human {
		basis = "inferred"
		if b, ok := ro["basis"]; ok {
			basis, _ = b.(string)
		}
	}
	if !contains(Bases, basis) {
		return nil, reject("bad_args", "owner.basis는 %s 중 하나입니다", pyTuple(Bases))
	}
	return obj{"who": strings.TrimSpace(str(ro["who"])), "basis": basis}, nil
}

func cleanDue(raw any) (any, *Reject) {
	if raw == nil {
		return nil, nil
	}
	ro := asObj(raw)
	if ro == nil || !isDate(ro["date"]) {
		return nil, reject("bad_args", "due.date가 YYYY-MM-DD가 아닙니다")
	}
	if !isStr(ro["basis"]) {
		return nil, reject("bad_args", "due.basis가 글이 아닙니다")
	}
	return obj{"date": ro["date"], "basis": ro["basis"]}, nil
}

// dueFromArgs는 set_due의 args 세 가지 형식을 읽는다. 화면 {date, basis}, 지우기 {clear: true},
// 명령과 추출 {due: {date, basis} | null}.
func dueFromArgs(args obj) (any, *Reject) {
	if has(args, "clear") {
		if args["clear"] != true || has(args, "date") || has(args, "due") {
			return nil, reject("bad_args", "clear는 true 하나만 쓰고 date, due와 함께 쓸 수 없습니다")
		}
		return nil, nil
	}
	if has(args, "due") {
		return cleanDue(args["due"])
	}
	if has(args, "date") {
		return cleanDue(args)
	}
	return nil, reject("bad_args", "set_due에는 date와 basis, due, clear 중 하나가 필요합니다")
}

func (l *Ledger) cleanWait(status string, raw any, itemID string) (any, *Reject) {
	if status != "waiting" {
		if truthy(raw) {
			return nil, reject("bad_args", "대기가 아닌 상태에는 wait를 쓸 수 없습니다")
		}
		return nil, nil
	}
	ro := asObj(raw)
	if ro == nil || !contains(WaitReasons, str(ro["reason"])) {
		return nil, reject("bad_args", "대기에는 wait.reason(%s)이 필요합니다", pyTuple(WaitReasons))
	}
	out := obj{"reason": ro["reason"]}
	on := ro["on"]
	if str(ro["reason"]) == "prior" {
		if !l.allItemIDs()[str(on)] || str(on) == itemID {
			return nil, reject("bad_args", "선행 대기는 다른 항목의 id를 on에 적어야 합니다")
		}
	} else if on != nil && !isStr(on) {
		return nil, reject("bad_args", "wait.on이 글이 아닙니다")
	}
	if on != nil {
		out["on"] = on
	}
	return out, nil
}

func cleanEvidence(raw any) ([]obj, *Reject) {
	list, ok := raw.([]any)
	if !ok {
		return nil, reject("bad_args", "evidence가 목록이 아닙니다")
	}
	var out []obj
	for _, e := range list {
		eo := asObj(e)
		if eo == nil || !keysEqual(eo, evidenceKeys) {
			return nil, reject("bad_args", "evidence 칸이 %s가 아닙니다", pyTuple(evidenceKeys))
		}
		c := obj{}
		for _, k := range evidenceKeys {
			c[k] = eo[k]
		}
		out = append(out, c)
	}
	return out, nil
}

func cleanMentions(raw any) ([]obj, *Reject) {
	list, ok := raw.([]any)
	if !ok {
		return nil, reject("bad_args", "mentions가 목록이 아닙니다")
	}
	var out []obj
	for _, m := range list {
		mo := asObj(m)
		if mo == nil || !keysEqual(mo, mentionKeys) {
			return nil, reject("bad_args", "mentions 칸이 %s가 아닙니다", pyTuple(mentionKeys))
		}
		c := obj{}
		for _, k := range mentionKeys {
			c[k] = mo[k]
		}
		out = append(out, c)
	}
	return out, nil
}

func (l *Ledger) mergeEvidenceAndMentions(item obj, evidence, mentions []obj) {
	haveEv := map[string]bool{}
	for _, e := range asList(item["evidence"]) {
		eo := asObj(e)
		haveEv[str(eo["path"])+"\x00"+str(eo["section"])+"\x00"+str(eo["quote"])] = true
	}
	for _, ev := range evidence {
		key := str(ev["path"]) + "\x00" + str(ev["section"]) + "\x00" + str(ev["quote"])
		if !haveEv[key] {
			item["evidence"] = append(asList(item["evidence"]), ev)
		}
	}
	haveMn := map[string]bool{}
	for _, m := range asList(item["mentions"]) {
		mo := asObj(m)
		haveMn[str(mo["date"])+"\x00"+str(mo["path"])+"\x00"+str(mo["quote"])] = true
	}
	for _, mn := range mentions {
		key := str(mn["date"]) + "\x00" + str(mn["path"]) + "\x00" + str(mn["quote"])
		if !haveMn[key] {
			item["mentions"] = append(asList(item["mentions"]), mn)
		}
	}
	sortMentions(item)
	kinds := map[string]string{"reminder": "reminder", "note": "pin", "wiki": "wiki"}
	for _, m := range asList(item["mentions"]) {
		mo := asObj(m)
		bumpRecord(item, str(mo["date"]), kinds[mentionKind(str(mo["path"]))])
	}
	for _, e := range asList(item["evidence"]) {
		bumpRecord(item, str(asObj(e)["date"]), "wiki")
	}
}

func sortMentions(item obj) {
	ms := asList(item["mentions"])
	sort.SliceStable(ms, func(i, j int) bool {
		a, b := asObj(ms[i]), asObj(ms[j])
		if str(a["date"]) != str(b["date"]) {
			return str(a["date"]) < str(b["date"])
		}
		return str(a["path"]) < str(b["path"])
	})
	item["mentions"] = ms
}

func (l *Ledger) verifyEvidence(evidence, mentions []obj, label string) *Reject {
	var evs, mns []quoteRef
	for _, e := range evidence {
		evs = append(evs, quoteRef{str(e["path"]), str(e["quote"])})
	}
	for _, m := range mentions {
		mns = append(mns, quoteRef{str(m["path"]), str(m["quote"])})
	}
	if r := l.verifyQuotes(evs, label+" evidence"); r != nil {
		return r
	}
	return l.verifyQuotes(mns, label+" mentions")
}

func (l *Ledger) rejectedBefore(match obj) bool {
	for _, pool := range [][]obj{l.rejections, l.newRejections} {
		for _, r := range pool {
			all := true
			for k, v := range match {
				if r[k] != v {
					all = false
					break
				}
			}
			if all {
				return true
			}
		}
	}
	return false
}

func (l *Ledger) rememberRejection(req obj, fields obj) {
	rec := obj{"at": req["at"], "via": req["via"]}
	for k, v := range fields {
		rec[k] = v
	}
	l.newRejections = append(l.newRejections, rec)
}

func humanVia(via string) bool { return via == "web" || via == "command" }

// cleanNote는 set_status와 touch의 args.note(한 줄 메모, 500자 이하)를 읽는다. 메모는 사람이
// 남기는 것이라 선언(extract)에는 쓸 수 없다.
func cleanNote(req obj) (any, *Reject) {
	args := asObj(req["args"])
	raw, ok := args["note"]
	if !ok || raw == nil {
		return nil, nil
	}
	if !humanVia(str(req["via"])) {
		return nil, reject("bad_args", "메모는 사람이 남기는 것이라 %s 요청에는 쓸 수 없습니다", str(req["via"]))
	}
	s, isS := raw.(string)
	if !isS {
		return nil, reject("bad_args", "note가 글이 아닙니다")
	}
	note := normalizeSpace(s)
	if n := runeLen(note); n > noteMax {
		return nil, reject("bad_args", "메모는 %d자 이하여야 합니다(지금 %d자)", noteMax, n)
	}
	if note == "" {
		return nil, nil
	}
	return note, nil
}

func addNote(item, req obj, note string) {
	day := dayOf(req)
	line := obj{"date": day, "path": NotePathPrefix + str(req["via"]), "quote": note}
	found := false
	for _, m := range asList(item["mentions"]) {
		if jsonEqual(m, line) {
			found = true
		}
	}
	if !found {
		item["mentions"] = append(asList(item["mentions"]), line)
		sortMentions(item)
	}
	bumpRecord(item, day, "pin")
}

func (l *Ledger) findOpenTitle(title, exclude string) string {
	norm := normalizeSpace(title)
	for _, id := range sortedKeys(l.items) {
		it := l.items[id]
		if id != exclude && str(it["status"]) != "done" && normalizeSpace(str(it["title"])) == norm {
			return id
		}
	}
	return ""
}

func (l *Ledger) newProposal(req obj, kind string, item any, question, reason string, args obj) obj {
	p := obj{"id": l.nextProposalID(), "kind": kind, "item": item, "question": question, "reason": reason,
		"args": args, "by": req["via"], "at": req["at"]}
	l.proposals = append(l.proposals, p)
	return p
}

func (l *Ledger) acceptNewItemProposal(itemID string) bool {
	for _, p := range l.proposals {
		if str(p["kind"]) == "new_item" && str(p["item"]) == itemID {
			l.removeProposal(p)
			return true
		}
	}
	return false
}

func (l *Ledger) addRelation(args obj, by string, confirmed bool) (obj, *Reject) {
	keys := []string{"basis", "from", "kind", "reason", "to"}
	for _, k := range keys {
		if !has(args, k) {
			return nil, reject("bad_args", "관계에는 %s가 필요합니다", pyList(keys))
		}
	}
	frm, to, kind := str(args["from"]), str(args["to"]), str(args["kind"])
	if !contains(RelationKind, kind) {
		return nil, reject("bad_args", "관계 kind는 %s 중 하나입니다", pyTuple(RelationKind))
	}
	all := l.allItemIDs()
	if !all[frm] || !all[to] || frm == to {
		return nil, reject("bad_args", "관계 양쪽이 서로 다른 실재 항목이어야 합니다")
	}
	if !contains(Bases, str(args["basis"])) {
		return nil, reject("bad_args", "관계 basis는 %s 중 하나입니다", pyTuple(Bases))
	}
	reason := args["reason"]
	// 자동화(추출, 판단)가 맺는 관련은 이유가 있어야 한다. 사람(web, command)이 맺는 관련은 이유가 선택이다.
	if kind == "related" && !humanVia(by) && !(isStr(reason) && strings.TrimSpace(str(reason)) != "") {
		return nil, reject("relation_reason", "관련에는 이유가 필요합니다(같은 산출물, 결정, 약속 중 하나)")
	}
	for _, rel := range l.relations {
		a, b := str(rel["from"]), str(rel["to"])
		if (a == frm && b == to) || (a == to && b == frm) {
			return nil, reject("relation_exists", "%s와 %s 사이에는 이미 %s 관계가 있습니다", frm, to, str(rel["kind"]))
		}
	}
	if l.rejectedBefore(obj{"kind": "relation", "frm": frm, "to": to, "rel": kind}) || l.rejectedBefore(obj{"kind": "relation", "frm": to, "to": frm, "rel": kind}) {
		return nil, reject("previously_rejected", "사람이 버린 관계입니다")
	}
	rr := reason
	if !truthy(rr) {
		rr = ""
	}
	rel := obj{"from": frm, "to": to, "kind": kind, "reason": rr, "basis": args["basis"], "by": by, "confirmed": confirmed}
	l.relations = append(l.relations, rel)
	return rel, nil
}

func (l *Ledger) itemRejectedBefore(title string, evidence []obj) bool {
	norm := normalizeSpace(title)
	quotes := map[string]bool{}
	for _, e := range evidence {
		quotes[str(e["path"])+"\x00"+normalizeSpace(str(e["quote"]))] = true
	}
	for _, pool := range [][]obj{l.rejections, l.newRejections} {
		for _, rec := range pool {
			if str(rec["kind"]) != "item" {
				continue
			}
			if rec["title"] == norm {
				return true
			}
			for _, e := range asList(rec["evidence"]) {
				eo := asObj(e)
				if quotes[str(eo["path"])+"\x00"+str(eo["quote"])] {
					return true
				}
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// op 처리. 각 함수는 outcome 을 돌려주거나 *Reject 를 돌려준다.
// ---------------------------------------------------------------------------

func newItemShape(id string) obj {
	return obj{
		"id": id, "rev": 1, "source": "command", "title": "", "work": Unsorted,
		"owner": obj{"who": "unknown", "basis": "inferred"}, "status": "todo", "wait": nil, "since": nil,
		"due": nil, "parent": nil, "reminder": nil, "evidence": []any{}, "mentions": []any{},
		"context":     obj{"why": nil, "now": nil, "next": nil, "written_at": nil, "from_commit": nil},
		"last_record": obj{"date": nil, "kind": "pin"}, "locked": []any{}, "created": nil, "closed_at": nil,
	}
}

func (l *Ledger) opAdd(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	via := str(req["via"])
	human := humanVia(via)
	title := args["title"]
	if !isStr(title) || strings.TrimSpace(str(title)) == "" {
		return outcome{}, reject("bad_args", "title이 없습니다")
	}
	if via == "extract" && has(args, "status") {
		return outcome{}, reject("ai_status", "AI는 항목 상태를 정하지 않습니다(새 항목은 할 일로 들어갑니다)")
	}
	if dup := l.findOpenTitle(str(title), ""); dup != "" {
		return outcome{}, reject("duplicate_title", "같은 제목의 완료가 아닌 일이 있습니다: %s", dup)
	}
	rawEv := args["evidence"]
	if !has(args, "evidence") {
		rawEv = []any{}
	}
	evidence, rj := cleanEvidence(rawEv)
	if rj != nil {
		return outcome{}, rj
	}
	if via == "extract" && l.itemRejectedBefore(str(title), evidence) {
		return outcome{}, reject("previously_rejected", "사람이 삭제했거나 아님으로 답한 일입니다")
	}
	day := dayOf(req)
	item := newItemShape(l.nextItemID())
	item["title"] = strings.TrimSpace(str(title))
	if via == "extract" {
		item["source"] = "wiki"
	} else {
		item["source"] = "command"
	}
	item["created"], item["since"] = day, day
	work := Unsorted
	if has(args, "work") {
		w, isS := args["work"].(string)
		if !isS || !l.workIDs()[w] {
			if human {
				return outcome{}, reject("bad_args", "프로젝트 %s가 config에 없습니다", pyRepr(w))
			}
			w = Unsorted
		}
		work = w
	}
	item["work"] = work
	if has(args, "owner") {
		o, rj := cleanOwner(args["owner"], human)
		if rj != nil {
			return outcome{}, rj
		}
		item["owner"] = o
	} else if human {
		item["owner"] = obj{"who": "me", "basis": "stated"}
	}
	if has(args, "due") {
		d, rj := cleanDue(args["due"])
		if rj != nil {
			return outcome{}, rj
		}
		item["due"] = d
	}
	if p := args["parent"]; p != nil {
		if !l.allItemIDs()[str(p)] {
			return outcome{}, reject("bad_args", "parent가 실재하지 않습니다")
		}
		item["parent"] = p
	}
	if human {
		for _, f := range []string{"owner", "work", "due"} {
			if has(args, f) {
				lock(item, f)
			}
		}
	}
	rawMn := args["mentions"]
	if !has(args, "mentions") {
		rawMn = []any{}
	}
	mentions, rj := cleanMentions(rawMn)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := l.verifyEvidence(evidence, mentions, "신규 항목"); rj != nil {
		return outcome{}, rj
	}
	if ctx := args["context"]; ctx != nil {
		co := asObj(ctx)
		if co == nil || !keysSubset(co, contextKeys) {
			return outcome{}, reject("bad_args", "context 칸 오류")
		}
		ic := asObj(item["context"])
		for k, v := range co {
			ic[k] = v
		}
	}
	if via == "extract" {
		if len(evidence) == 0 {
			return outcome{}, reject("no_evidence", "위키에서 찾은 새 항목에는 근거(evidence)가 필요합니다")
		}
	} else {
		item["last_record"] = obj{"date": day, "kind": "pin"}
	}
	l.mergeEvidenceAndMentions(item, evidence, mentions)
	id := str(item["id"])
	l.items[id] = item
	l.mark(id)
	msg := "새 항목 " + id
	if via == "extract" {
		where := "근거 문서"
		if len(evidence) > 0 {
			where = str(evidence[0]["path"])
		}
		l.newProposal(req, "new_item", id, "「"+str(item["title"])+"」을 새 일로 둘까요?", "위키에서 찾았습니다: "+where, obj{})
		msg += " (제안 표시)"
	}
	return outcome{item: id, msg: msg}, nil
}

func (l *Ledger) opSetStatus(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	args := asObj(req["args"])
	status := str(args["status"])
	if !contains(Statuses, status) || !isStr(args["status"]) {
		return outcome{}, reject("bad_args", "status는 %s 중 하나입니다", pyTuple(Statuses))
	}
	note, rj := cleanNote(req)
	if rj != nil {
		return outcome{}, rj
	}
	id := str(item["id"])
	if status == "done" && str(item["status"]) == "done" {
		if note == nil {
			return outcome{item: id, msg: "이미 완료입니다", noop: true}, nil
		}
		addNote(item, req, note.(string))
		l.changed(item)
		return outcome{item: id, msg: "이미 완료입니다. 메모만 남김"}, nil
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	waitAny, rj := l.cleanWait(status, args["wait"], id)
	if rj != nil {
		return outcome{}, rj
	}
	wait := asObj(waitAny)
	if status == str(item["status"]) && jsonEqual(waitAny, item["wait"]) {
		if note == nil {
			return outcome{item: id, msg: "이미 같은 상태입니다", noop: true}, nil
		}
		addNote(item, req, note.(string))
		l.changed(item)
		return outcome{item: id, msg: "이미 같은 상태입니다. 메모만 남김"}, nil
	}
	day := dayOf(req)
	wasDone := str(item["status"]) == "done"
	beforeLabel := statusLabel(str(item["status"]), asObj(item["wait"]))
	if status != str(item["status"]) {
		item["since"] = day
	}
	item["status"], item["wait"] = status, waitAny
	recordChange(item, req, "status", beforeLabel, statusLabel(status, wait), note)
	if status == "done" {
		item["closed_at"] = day
	} else {
		item["closed_at"] = nil
	}
	bumpRecord(item, day, "pin")
	if note != nil {
		addNote(item, req, note.(string))
	}
	l.changed(item)
	msg := "상태 " + status
	if str(req["via"]) != "extract" && l.acceptNewItemProposal(id) {
		msg += ", 상태 변경으로 제안 수락"
	}
	if status == "done" && !wasDone {
		if freed := l.releaseWaiters(item, req); len(freed) > 0 {
			msg += ", 대기 해제 " + strings.Join(freed, ", ")
		}
	}
	return outcome{item: id, msg: msg}, nil
}

func (l *Ledger) opTouch(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	note, rj := cleanNote(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	bumpRecord(item, dayOf(req), "pin")
	if note != nil {
		addNote(item, req, note.(string))
	}
	l.changed(item)
	msg := "마지막 기록 갱신"
	if note != nil {
		msg += ", 메모 남김"
	}
	return outcome{item: str(item["id"]), msg: msg}, nil
}

func (l *Ledger) opSetTitle(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	human := humanVia(str(req["via"]))
	if !human && str(item["source"]) != "reminder" {
		return outcome{}, reject("bad_args", "제목을 따라가는 것은 미리알림에서 온 항목뿐입니다")
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	title := asObj(req["args"])["title"]
	if !isStr(title) || strings.TrimSpace(str(title)) == "" {
		return outcome{}, reject("bad_args", "title이 없습니다")
	}
	nt := strings.TrimSpace(str(title))
	if human {
		if n := runeLen(nt); n > maxTitle {
			return outcome{}, reject("bad_args", "제목은 %d자까지입니다(지금 %d자)", maxTitle, n)
		}
		if strings.ContainsAny(nt, "\r\n") {
			return outcome{}, reject("bad_args", "제목은 한 줄이어야 합니다")
		}
	}
	id := str(item["id"])
	if nt == str(item["title"]) {
		return outcome{item: id, msg: "제목이 같습니다", noop: true}, nil
	}
	recordChange(item, req, "title", item["title"], nt, nil)
	item["title"] = nt
	if human {
		lock(item, "title")
	}
	bumpRecord(item, dayOf(req), "pin")
	l.changed(item)
	return outcome{item: id, msg: "제목 고침"}, nil
}

func lockedGuard(item, req obj, field string) *Reject {
	if str(req["via"]) == "extract" && isLocked(item, field) {
		return reject("locked", "사람이 고친 칸이라 AI가 다시 쓰지 않습니다: %s", field)
	}
	return nil
}

func (l *Ledger) opSetOwner(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	if rj := lockedGuard(item, req, "owner"); rj != nil {
		return outcome{}, rj
	}
	human := humanVia(str(req["via"]))
	nw, rj := cleanOwner(req["args"], human)
	if rj != nil {
		return outcome{}, rj
	}
	id := str(item["id"])
	if jsonEqual(nw, item["owner"]) && (!human || contains(strList(item["locked"]), "owner")) {
		return outcome{item: id, msg: "담당이 같습니다", noop: true}, nil
	}
	recordChange(item, req, "owner", ownerWord(asObj(item["owner"])), ownerWord(nw), nil)
	item["owner"] = nw
	if human {
		lock(item, "owner")
		bumpRecord(item, dayOf(req), "pin")
	}
	l.changed(item)
	return outcome{item: id, msg: "담당 " + str(nw["who"])}, nil
}

func foldName(name any) string {
	s, ok := name.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(normalizeSpace(s)), ""))
}

func (l *Ledger) workByName(name string) string {
	key := foldName(name)
	for _, w := range asList(l.config["works"]) {
		if wo := asObj(w); foldName(wo["name"]) == key {
			return str(wo["id"])
		}
	}
	return ""
}

var slugRe = regexp.MustCompile(`[a-z0-9]+`)

func workSlug(name string, taken []string) string {
	base := strings.Join(slugRe.FindAllString(strings.ToLower(name), -1), "")
	if len(base) > 16 {
		base = base[:16]
	}
	if base == "" {
		base = "work"
	}
	used := setOf(append(append([]string{}, taken...), Unsorted)...)
	if !used[base] {
		return base
	}
	n := 2
	for used[base+itoa(n)] {
		n++
	}
	return base + itoa(n)
}

func cleanWorkName(raw any) (string, *Reject) {
	name := ""
	if s, ok := raw.(string); ok {
		name = normalizeSpace(s)
	}
	if name == "" || runeLen(name) > maxWorkName {
		return "", reject("bad_args", "프로젝트 이름은 1~%d자 글이어야 합니다", maxWorkName)
	}
	return name, nil
}

func cleanWorkDesc(raw any) (string, *Reject) {
	if raw == nil || raw == "" {
		return "", nil
	}
	desc := ""
	if s, ok := raw.(string); ok {
		desc = normalizeSpace(s)
	}
	if desc == "" || runeLen(desc) > maxWorkDesc {
		return "", reject("bad_args", "프로젝트 설명은 1~%d자 글이어야 합니다", maxWorkDesc)
	}
	return desc, nil
}

func (l *Ledger) addWork(name string, words []string, desc string) (string, *Reject) {
	var ids []string
	for _, w := range asList(l.config["works"]) {
		ids = append(ids, str(asObj(w)["id"]))
	}
	wid := workSlug(name, ids)
	for _, w := range asList(l.config["works"]) {
		if normalizeSpace(str(asObj(w)["name"])) == name {
			return "", reject("work_exists", "같은 이름의 프로젝트가 이미 있습니다: %s. 기존 프로젝트에 붙이기를 고르십시오", name)
		}
	}
	entry := obj{"id": wid, "name": name, "words": toAnyList(words)}
	if desc != "" {
		entry["desc"] = desc
	}
	l.config["works"] = append(asList(l.config["works"]), entry)
	l.configDirty = true
	return wid, nil
}

func (l *Ledger) opSetWork(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	if rj := lockedGuard(item, req, "work"); rj != nil {
		return outcome{}, rj
	}
	human := humanVia(str(req["via"]))
	args := asObj(req["args"])
	created := ""
	var work string
	if has(args, "new") {
		if !human {
			return outcome{}, reject("bad_args", "새 프로젝트는 사람이 만듭니다. %s 요청에는 쓸 수 없습니다", str(req["via"]))
		}
		if has(args, "work") {
			return outcome{}, reject("bad_args", "work와 new는 함께 쓸 수 없습니다")
		}
		nw := asObj(args["new"])
		if nw == nil || !keysSubset(nw, []string{"name", "desc"}) {
			return outcome{}, reject("bad_args", "new는 name과 desc만 가진 객체여야 합니다")
		}
		name, rj := cleanWorkName(nw["name"])
		if rj != nil {
			return outcome{}, rj
		}
		desc, rj := cleanWorkDesc(nw["desc"])
		if rj != nil {
			return outcome{}, rj
		}
		work = l.workByName(name)
		if work == "" {
			work, rj = l.addWork(name, []string{name}, desc)
			if rj != nil {
				return outcome{}, rj
			}
			created = name
		}
	} else {
		w, _ := args["work"].(string)
		work = w
	}
	if !l.workIDs()[work] {
		return outcome{}, reject("bad_args", "프로젝트 %s가 config에 없습니다. 프로젝트는 사람이 config에 먼저 적습니다", pyRepr(work))
	}
	id := str(item["id"])
	if work == str(item["work"]) && (!human || contains(strList(item["locked"]), "work")) && created == "" {
		return outcome{item: id, msg: "프로젝트가 같습니다", noop: true}, nil
	}
	recordChange(item, req, "work", l.workLabel(str(item["work"])), l.workLabel(work), nil)
	item["work"] = work
	if human {
		lock(item, "work")
		bumpRecord(item, dayOf(req), "pin")
	}
	l.changed(item)
	if created != "" {
		return outcome{item: id, msg: "새 프로젝트 " + created + " 만들고 이동: " + work}, nil
	}
	return outcome{item: id, msg: "프로젝트 " + work}, nil
}

func (l *Ledger) opSetWorkOrder(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	var extra []string
	for k := range args {
		if k != "order" {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return outcome{}, reject("bad_args", "set_work_order는 order만 받습니다: %s", pyList(extra))
	}
	rawOrder, isList := args["order"].([]any)
	if !isList {
		return outcome{}, reject("bad_args", "order가 프로젝트 id 목록이 아닙니다")
	}
	var order []string
	for _, o := range rawOrder {
		if !isStr(o) {
			return outcome{}, reject("bad_args", "order가 프로젝트 id 목록이 아닙니다")
		}
		order = append(order, str(o))
	}
	var have []string
	for _, w := range asList(l.config["works"]) {
		have = append(have, str(asObj(w)["id"]))
	}
	if len(setOf(order...)) != len(order) {
		return outcome{}, reject("bad_args", "order에 같은 프로젝트가 두 번 있습니다")
	}
	haveSet, orderSet := setOf(have...), setOf(order...)
	var unknown, missing []string
	for _, o := range order {
		if !haveSet[o] {
			unknown = append(unknown, o)
		}
	}
	if len(unknown) > 0 {
		return outcome{}, reject("bad_args", "order에 config에 없는 프로젝트가 있습니다: %s", pyList(unknown))
	}
	for _, h := range have {
		if !orderSet[h] {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return outcome{}, reject("bad_args", "order에 빠진 프로젝트가 있습니다: %s", pyList(missing))
	}
	if strings.Join(order, "\x00") == strings.Join(have, "\x00") {
		return outcome{msg: "프로젝트 순서가 같습니다", noop: true}, nil
	}
	byID := map[string]any{}
	for _, w := range asList(l.config["works"]) {
		byID[str(asObj(w)["id"])] = w
	}
	var works []any
	for _, o := range order {
		works = append(works, byID[o])
	}
	l.config["works"] = works
	l.configDirty = true
	return outcome{msg: fmt.Sprintf("프로젝트 순서 %d개를 바꿈", len(order))}, nil
}

func (l *Ledger) opMarkSeen(req obj) (outcome, *Reject) {
	if req["item"] != nil || req["base_rev"] != nil {
		return outcome{}, reject("bad_args", "mark_seen에는 item과 base_rev를 쓰지 않습니다")
	}
	args := asObj(req["args"])
	var extra []string
	for k := range args {
		if k != "items" {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return outcome{}, reject("bad_args", "mark_seen은 items만 받습니다: %s", pyList(extra))
	}
	list, isList := args["items"].([]any)
	ok := isList && len(list) >= 1 && len(list) <= markSeenMax
	for _, e := range list {
		if !isStr(e) {
			ok = false
		}
	}
	if !ok {
		return outcome{}, reject("bad_args", "items는 항목 번호 1~%d개의 목록이어야 합니다", markSeenMax)
	}
	var cleared []string
	seen := map[string]bool{}
	for _, e := range list {
		id := str(e)
		if seen[id] {
			continue
		}
		seen[id] = true
		it, found := l.items[id]
		hit := false
		for _, key := range []string{"unseen", "freed_by"} { // 선행 완료로 풀린 일의 확인 카드도 같은 요청으로 닫는다
			if found && has(it, key) {
				delete(it, key)
				hit = true
			}
		}
		if hit {
			l.mark(id)
			cleared = append(cleared, id)
		}
	}
	if len(cleared) == 0 {
		return outcome{msg: "확인할 새 일이 없습니다", noop: true}, nil
	}
	return outcome{msg: fmt.Sprintf("새 일 %d건 확인: %s", len(cleared), strings.Join(cleared, ", "))}, nil
}

func (l *Ledger) clearUnseenByHuman(req obj, itemID string) {
	if !humanVia(str(req["via"])) || str(req["op"]) == "mark_seen" {
		return
	}
	args := asObj(req["args"])
	touched := []string{itemID, str(req["item"])}
	if op := str(req["op"]); op == "add_relation" || op == "remove_relation" || op == "confirm_relation" {
		touched = append(touched, str(args["from"]), str(args["to"]))
	}
	for _, id := range touched {
		for _, key := range []string{"unseen", "freed_by"} { // 선행 완료로 풀린 표시도 같은 규칙으로 지운다
			if it, ok := l.items[id]; ok && has(it, key) {
				delete(it, key)
				l.mark(id)
			}
		}
	}
}

// releaseWaiters는 prior 를 선행으로 기다리던 일을 할 일로 바꾸고 freed_by 표시를 붙인다.
// prior 를 다시 열어도(완료 취소) 풀린 일은 대기로 되돌리지 않는다.
func (l *Ledger) releaseWaiters(prior, req obj) []string {
	day := dayOf(req)
	pid := str(prior["id"])
	var out []string
	for _, oid := range sortedKeys(l.items) {
		other := l.items[oid]
		wait := asObj(other["wait"])
		if oid == pid || str(other["status"]) != "waiting" || wait == nil || str(wait["reason"]) != "prior" || str(wait["on"]) != pid {
			continue
		}
		before := statusLabel(str(other["status"]), wait)
		other["status"], other["wait"], other["since"] = "todo", nil, day
		recordChange(other, req, "status", before, statusLabels["todo"], "선행 "+pid+" 완료로 대기 해제")
		other["freed_by"] = obj{"by": pid, "date": day}
		bumpRecord(other, day, "pin")
		l.changed(other)
		out = append(out, oid)
	}
	return out
}

// releaseDonePriors는 요청과 무관하게, 이미 완료된(또는 보관된) 일을 선행으로 기다리는
// 일이 남아 있으면 푼다. 규칙이 생기기 전의 대기와 다른 경로의 완료를 맞춘다.
func (l *Ledger) releaseDonePriors(nowISO string) []string {
	req := obj{"via": "command", "at": nowISO, "client": nil}
	l.actingVia, l.actingClient = "command", nil
	priors := map[string]bool{}
	for _, it := range l.items {
		if w := asObj(it["wait"]); str(it["status"]) == "waiting" && w != nil && w["on"] != nil {
			priors[str(w["on"])] = true
		}
	}
	var out []string
	for _, pid := range sortedKeys(priors) {
		prior, ok := l.items[pid]
		if !ok {
			prior, ok = l.archived[pid]
		}
		if ok && str(prior["status"]) == "done" {
			out = append(out, l.releaseWaiters(prior, req)...)
		}
	}
	return out
}

func (l *Ledger) opSetWorkDesc(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	var extra []string
	for k := range args {
		if k != "work" && k != "desc" {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return outcome{}, reject("bad_args", "set_work_desc는 work와 desc만 받습니다: %s", pyList(extra))
	}
	desc, rj := cleanWorkDesc(args["desc"])
	if rj != nil {
		return outcome{}, rj
	}
	if desc == "" {
		return outcome{}, reject("bad_args", "프로젝트 설명이 비었습니다")
	}
	for _, w := range asList(l.config["works"]) {
		wo := asObj(w)
		if str(wo["id"]) == str(args["work"]) {
			if wo["desc"] == desc {
				return outcome{msg: "프로젝트 설명이 같습니다", noop: true}, nil
			}
			wo["desc"] = desc
			l.configDirty = true
			return outcome{msg: "프로젝트 " + str(wo["id"]) + " 설명 고침"}, nil
		}
	}
	return outcome{}, reject("bad_args", "프로젝트 %s가 config에 없습니다", pyRepr(str(args["work"])))
}

func (l *Ledger) opSetDue(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	if rj := lockedGuard(item, req, "due"); rj != nil {
		return outcome{}, rj
	}
	due, rj := dueFromArgs(asObj(req["args"]))
	if rj != nil {
		return outcome{}, rj
	}
	human := humanVia(str(req["via"]))
	id := str(item["id"])
	if jsonEqual(due, item["due"]) && (!human || contains(strList(item["locked"]), "due")) {
		return outcome{item: id, msg: "기한이 같습니다", noop: true}, nil
	}
	var before, after any
	if d := asObj(item["due"]); d != nil {
		before = d["date"]
	}
	if d := asObj(due); d != nil {
		after = d["date"]
	}
	recordChange(item, req, "due", before, after, nil)
	item["due"] = due
	if st := str(item["start"]); has(item, "start") && (due == nil || st > str(asObj(due)["date"])) { // 시작은 기한이 있을 때만, 기한보다 늦지 않게. 어긋나면 자동 시작으로 돌아간다
		recordChange(item, req, "start", st, nil, nil)
		delete(item, "start")
	}
	if human {
		lock(item, "due")
		bumpRecord(item, dayOf(req), "pin")
	}
	l.changed(item)
	if after != nil {
		return outcome{item: id, msg: "기한 " + str(after)}, nil
	}
	return outcome{item: id, msg: "기한 없음"}, nil
}

// opSetStart는 시작일이다. args {date: YYYY-MM-DD} 또는 {clear: true}(자동 시작으로).
// 기한이 있어야 하고 기한보다 늦을 수 없다.
func (l *Ledger) opSetStart(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	args := asObj(req["args"])
	var start any
	switch {
	case len(args) == 1 && args["clear"] == true:
		start = nil
	case len(args) == 1 && has(args, "date") && isDate(args["date"]):
		start = args["date"]
	default:
		return outcome{}, reject("bad_args", "set_start에는 date(YYYY-MM-DD) 또는 clear: true 하나만 씁니다")
	}
	id := str(item["id"])
	if start != nil {
		due := asObj(item["due"])
		if due == nil {
			return outcome{}, reject("bad_args", "시작은 기한이 있을 때만 정합니다")
		}
		if str(start) > str(due["date"]) {
			return outcome{}, reject("bad_args", "시작이 기한보다 늦습니다")
		}
	}
	var cur any
	if has(item, "start") {
		cur = item["start"]
	}
	if cur == start {
		return outcome{item: id, msg: "시작이 같습니다", noop: true}, nil
	}
	recordChange(item, req, "start", cur, start, nil)
	if start == nil {
		delete(item, "start")
	} else {
		item["start"] = start
	}
	bumpRecord(item, dayOf(req), "pin")
	l.changed(item)
	if start == nil {
		return outcome{item: id, msg: "시작 자동"}, nil
	}
	return outcome{item: id, msg: "시작 " + str(start)}, nil
}

func (l *Ledger) setParent(item obj, parent any, req obj) *Reject {
	if parent != nil {
		p := str(parent)
		if !l.allItemIDs()[p] || p == str(item["id"]) {
			return reject("bad_args", "parent가 실재하지 않거나 자기 자신입니다")
		}
		cursor := p
		for cursor != "" {
			if cursor == str(item["id"]) {
				return reject("bad_args", "parent가 순환합니다")
			}
			next := ""
			if it, ok := l.items[cursor]; ok {
				next = str(it["parent"])
			} else if it, ok := l.archived[cursor]; ok {
				next = str(it["parent"])
			}
			cursor = next
		}
	}
	item["parent"] = parent
	bumpRecord(item, dayOf(req), "pin")
	l.changed(item)
	return nil
}

func (l *Ledger) opSetParent(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	parent := asObj(req["args"])["parent"]
	id := str(item["id"])
	if parent == item["parent"] {
		return outcome{item: id, msg: "상위 항목이 같습니다", noop: true}, nil
	}
	if rj := l.setParent(item, parent, req); rj != nil {
		return outcome{}, rj
	}
	return outcome{item: id, msg: parentMessage(parent)}, nil
}

// parentMessage는 상위 항목을 바꾼 로그 문장이다. upstream 은 상위를 비울 때 파이썬 str(None)인
// "상위 None"을 쓴다. 사용자가 읽는 글이라 여기서는 "상위 없음"으로 쓰고, 하네스 견본은 상위를 비우는
// 요청을 싣지 않는다(harness/pin/pin_parity_test.go).
func parentMessage(parent any) string {
	if parent == nil {
		return "상위 없음"
	}
	return "상위 " + str(parent)
}

func (l *Ledger) opUpdateAI(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	args := asObj(req["args"])
	if !keysSubset(args, []string{"context", "mentions", "evidence"}) {
		var bad []string
		for k := range args {
			if k != "context" && k != "mentions" && k != "evidence" {
				bad = append(bad, k)
			}
		}
		sort.Strings(bad)
		return outcome{}, reject("bad_args", "update_ai는 context, mentions, evidence만 바꿉니다: %s", pyList(bad))
	}
	rawEv, rawMn := args["evidence"], args["mentions"]
	if !has(args, "evidence") {
		rawEv = []any{}
	}
	if !has(args, "mentions") {
		rawMn = []any{}
	}
	evidence, rj := cleanEvidence(rawEv)
	if rj != nil {
		return outcome{}, rj
	}
	mentions, rj := cleanMentions(rawMn)
	if rj != nil {
		return outcome{}, rj
	}
	id := str(item["id"])
	if rj := l.verifyEvidence(evidence, mentions, id); rj != nil {
		return outcome{}, rj
	}
	var skipped []string
	before := dump(item, false)
	if ctx := args["context"]; ctx != nil {
		co := asObj(ctx)
		if co == nil || !keysSubset(co, contextKeys) {
			return outcome{}, reject("bad_args", "context 칸 오류")
		}
		ic := asObj(item["context"])
		for _, key := range contextKeys {
			value, ok := co[key]
			if !ok {
				continue
			}
			switch {
			case (key == "why" || key == "now" || key == "next") && isLocked(item, "context."+key):
				skipped = append(skipped, key)
			case !(value == nil || isStr(value)):
				return outcome{}, reject("bad_args", "context.%s가 글이 아닙니다", key)
			default:
				ic[key] = value
			}
		}
	}
	l.mergeEvidenceAndMentions(item, evidence, mentions)
	if dump(item, false) == before {
		return outcome{item: id, msg: "바뀐 것이 없습니다", noop: true}, nil
	}
	l.changed(item)
	msg := "맥락, 언급된 곳, 근거 갱신"
	if len(skipped) > 0 {
		msg += " (잠긴 칸 건너뜀: " + strings.Join(skipped, ",") + ")"
	}
	return outcome{item: id, msg: msg}, nil
}

func stageOfPath(path string) string {
	head, _, _ := strings.Cut(path, "/")
	if contains(Stages, head) {
		return head
	}
	return ""
}

// opMovePath는 위키 문서 이름이 바뀌면 기존 항목의 근거와 언급된 곳의 경로를 따라 고친다.
// 내용을 판단하지 않는 기계적 변경이다. upstream 과 달리 옮긴 문서에서 인용문을 다시 대조하지
// 않는다. update 가 inbox 본문을 바꾸는 것이 정상 절차라 대조하면 승급 준비를 거친 문서의
// 경로 이동이 전부 거절된다(ADR 0106).
func (l *Ledger) opMovePath(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	old, nw := str(args["from"]), str(args["to"])
	if !(isStr(args["from"]) && isStr(args["to"]) && old != "" && nw != "" && old != nw) {
		return outcome{}, reject("bad_args", "move_path에는 서로 다른 from, to 경로가 필요합니다")
	}
	for _, p := range []string{old, nw} {
		if unsafePath(p) || strings.HasPrefix(p, ReminderPathPrefix) {
			return outcome{}, reject("quote_path", "경로가 저장소 밖을 가리키거나 미리알림 경로입니다: %s", p)
		}
	}
	newStage := stageOfPath(nw)
	if newStage == "" {
		return outcome{}, reject("bad_args", "to 경로가 %s 중 한 곳에서 시작해야 합니다: %s", pyTuple(Stages), nw)
	}
	var holders []obj
	for _, bucket := range []map[string]obj{l.items, l.archived} {
		for _, id := range sortedKeys(bucket) {
			it := bucket[id]
			hit := false
			for _, e := range asList(it["evidence"]) {
				if str(asObj(e)["path"]) == old {
					hit = true
				}
			}
			for _, m := range asList(it["mentions"]) {
				if str(asObj(m)["path"]) == old {
					hit = true
				}
			}
			if hit {
				holders = append(holders, it)
			}
		}
	}
	if len(holders) == 0 {
		return outcome{msg: "이 경로를 근거로 둔 항목이 없습니다", noop: true}, nil
	}
	if !l.pathExists(nw) {
		return outcome{}, reject("quote_path", "근거 경로 이동: 근거 문서가 없습니다: %s", nw)
	}
	for _, it := range holders {
		for _, e := range asList(it["evidence"]) {
			if eo := asObj(e); str(eo["path"]) == old {
				eo["path"], eo["stage"] = nw, newStage
			}
		}
		for _, m := range asList(it["mentions"]) {
			if mo := asObj(m); str(mo["path"]) == old {
				mo["path"] = nw
			}
		}
		seenEv := map[string]bool{}
		var evs []any
		for _, e := range asList(it["evidence"]) {
			eo := asObj(e)
			key := str(eo["path"]) + "\x00" + str(eo["section"]) + "\x00" + str(eo["quote"])
			if !seenEv[key] {
				seenEv[key] = true
				evs = append(evs, e)
			}
		}
		if evs == nil {
			evs = []any{}
		}
		it["evidence"] = evs
		seenMn := map[string]bool{}
		var mns []any
		for _, m := range asList(it["mentions"]) {
			mo := asObj(m)
			key := str(mo["date"]) + "\x00" + str(mo["path"]) + "\x00" + str(mo["quote"])
			if !seenMn[key] {
				seenMn[key] = true
				mns = append(mns, m)
			}
		}
		if mns == nil {
			mns = []any{}
		}
		it["mentions"] = mns
		// 경로만 따라가는 변경이라 rev는 올리지 않는다(올리면 그 사이 화면에서 누른 단추가 rev 충돌로 거절된다).
		if _, ok := l.items[str(it["id"])]; ok {
			l.mark(str(it["id"]))
		} else {
			l.dirtyArchive[str(it["id"])] = true
		}
	}
	// 새 일 제안 카드의 근거 문장은 add 때 경로를 박아 둔 것이라 함께 따라 고친다(그대로 두면 카드가 옛 경로를 보인다).
	moved := map[string]bool{}
	for _, it := range holders {
		moved[str(it["id"])] = true
	}
	oldReason, newReason := "위키에서 찾았습니다: "+old, "위키에서 찾았습니다: "+nw
	for _, prop := range l.proposals {
		if str(prop["kind"]) == "new_item" && moved[str(prop["item"])] && prop["reason"] == oldReason {
			prop["reason"] = newReason
		}
	}
	return outcome{item: str(holders[0]["id"]), msg: fmt.Sprintf("근거 경로 %s>%s: 항목 %d건", old, nw, len(holders))}, nil
}

func relationWord(kind string) string {
	return map[string]string{"before": "선행", "related": "관련", "duplicate": "중복"}[kind]
}

func (l *Ledger) opAddRelation(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	human := humanVia(str(req["via"]))
	if !human && args["basis"] == "inferred" {
		// 추정 관계는 제안으로 낸다. 사람이 수락하면 들어간다.
		for _, k := range []string{"basis", "from", "kind", "reason", "to"} {
			if !has(args, k) {
				return outcome{}, reject("bad_args", "관계에는 %s가 필요합니다", k)
			}
		}
		for _, ex := range l.proposals {
			if str(ex["kind"]) == "relation" && jsonEqual(ex["args"], args) {
				return outcome{msg: "이미 제안했습니다 " + str(ex["id"]), noop: true}, nil
			}
		}
		if _, rj := l.addRelation(args, str(req["via"]), false); rj != nil {
			return outcome{}, rj
		}
		l.relations = l.relations[:len(l.relations)-1]
		titles := l.allTitles()
		prop := l.newProposal(req, "relation", nil, relationQuestion(str(args["kind"]), titles[str(args["from"])], titles[str(args["to"])]),
			str(args["reason"]), cloneObj(args))
		return outcome{msg: "관계 제안 " + str(prop["id"])}, nil
	}
	if args["kind"] == "related" && !human {
		for _, pid := range []any{args["from"], args["to"]} {
			n := 0
			for _, r := range l.relations {
				if str(r["kind"]) == "related" && (r["from"] == pid || r["to"] == pid) {
					n++
				}
			}
			if n >= relatedLimit {
				return outcome{}, reject("related_limit", "%s의 관련이 이미 %d건입니다", str(pid), relatedLimit)
			}
		}
	}
	rel, rj := l.addRelation(args, str(req["via"]), human)
	if rj != nil {
		return outcome{}, rj
	}
	for _, pid := range []string{str(rel["from"]), str(rel["to"])} {
		if _, ok := l.items[pid]; ok {
			l.mark(pid)
		}
	}
	return outcome{item: str(rel["from"]), msg: fmt.Sprintf("관계 %s>%s %s", str(rel["from"]), str(rel["to"]), relationWord(str(rel["kind"])))}, nil
}

func (l *Ledger) opRemoveRelation(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	for _, k := range []string{"from", "to", "kind"} {
		if !has(args, k) {
			return outcome{}, reject("bad_args", "remove_relation에는 from, to, kind가 필요합니다")
		}
	}
	i := l.findRelation(args["from"], args["to"], args["kind"])
	if i < 0 {
		return outcome{msg: "그런 관계가 없습니다", noop: true}, nil
	}
	rel := l.relations[i]
	l.relations = append(l.relations[:i:i], l.relations[i+1:]...)
	l.rememberRejection(req, obj{"kind": "relation", "frm": rel["from"], "to": rel["to"], "rel": rel["kind"]})
	return outcome{item: str(rel["from"]), msg: "관계 지움"}, nil
}

// findRelation은 (from, to, kind)가 같은 관계의 위치를 낸다. 없으면 -1 이다. upstream find_relation 과 같다.
// 선행(before)은 방향이 뜻이라 from, to 가 그대로 같아야 하고, 연관과 중복은 방향이 없어 두 끝이 같으면 된다.
// 화면은 연관과 중복의 저장 방향을 모르므로 사람이 끊고 확인하는 요청도 두 끝이 맞으면 찾는다.
func (l *Ledger) findRelation(frm, to, kind any) int {
	for i, rel := range l.relations {
		if rel["kind"] != kind {
			continue
		}
		if (rel["from"] == frm && rel["to"] == to) || (kind != "before" && rel["from"] == to && rel["to"] == frm) {
			return i
		}
	}
	return -1
}

// opConfirmRelation은 사람이 추정 관계를 맞다고 하는 것이다. 그 관계의 confirmed 만 true 로 바꾸고
// basis 는 그대로 둔다. args 는 from, to, kind 정확히 셋이다. 관계 목록이 대상이라 항목 rev 를 올리지
// 않고 양 끝 항목만 mark 한다. 이미 확인했거나 그런 관계가 없으면 바뀐 것 없음이다. 거절이 아니다.
func (l *Ledger) opConfirmRelation(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	if len(args) != 3 || !has(args, "from") || !has(args, "to") || !has(args, "kind") {
		return outcome{}, reject("bad_args", "confirm_relation은 from, to, kind만 받습니다: %s", pyList(sortedKeys(args)))
	}
	i := l.findRelation(args["from"], args["to"], args["kind"])
	if i < 0 {
		return outcome{msg: "그런 관계가 없습니다", noop: true}, nil
	}
	rel := l.relations[i]
	if rel["confirmed"] == true {
		return outcome{msg: "이미 확인한 관계입니다", noop: true}, nil
	}
	rel["confirmed"] = true
	for _, k := range []string{"from", "to"} {
		if pid := str(rel[k]); pid != "" {
			if _, ok := l.items[pid]; ok {
				l.mark(pid)
			}
		}
	}
	return outcome{item: str(rel["from"]), msg: fmt.Sprintf("관계 확인 %s>%s %s", str(rel["from"]), str(rel["to"]), relationWord(str(rel["kind"])))}, nil
}

func (l *Ledger) opPropose(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	kind := str(args["kind"])
	if !contains(ProposalKinds, kind) || kind == "new_item" {
		return outcome{}, reject("bad_args", "propose의 kind는 status, relation, parent, work, delete 중 하나입니다")
	}
	for _, k := range []string{"question", "reason"} {
		if !isStr(args[k]) || strings.TrimSpace(str(args[k])) == "" {
			return outcome{}, reject("bad_args", "%s가 없습니다", k)
		}
	}
	inner := obj{}
	if has(args, "args") {
		inner = asObj(args["args"])
		if inner == nil {
			return outcome{}, reject("bad_args", "제안의 args가 객체가 아닙니다")
		}
	}
	itemAny := req["item"]
	itemID := str(itemAny)
	if kind == "delete" {
		return outcome{}, reject("op_not_allowed", "삭제 제안은 미리알림 동기화만 냅니다")
	}
	if kind == "work" {
		return l.opProposeWork(req, args, inner)
	}
	if kind == "status" || kind == "parent" {
		it, ok := l.items[itemID]
		if !ok {
			return outcome{}, reject("no_item", "%s 일이 없습니다", itemID)
		}
		if kind == "status" && !(isStr(inner["status"]) && contains(Statuses, str(inner["status"]))) {
			return outcome{}, reject("bad_args", "제안의 status가 올바르지 않습니다")
		}
		if kind == "status" && str(inner["status"]) == str(it["status"]) {
			return outcome{item: itemID, msg: "이미 그 상태입니다", noop: true}, nil
		}
		if kind == "parent" && !l.allItemIDs()[str(inner["parent"])] {
			return outcome{}, reject("bad_args", "제안의 parent가 실재하지 않습니다")
		}
		if l.rejectedBefore(obj{"kind": kind, "item": itemAny, "value": dump(inner, false)}) {
			return outcome{}, reject("previously_rejected", "사람이 아님으로 버린 제안입니다")
		}
	}
	for _, ex := range l.proposals {
		if str(ex["kind"]) == kind && ex["item"] == itemAny && jsonEqual(ex["args"], inner) {
			return outcome{item: itemID, msg: "이미 제안했습니다 " + str(ex["id"]), noop: true}, nil
		}
	}
	prop := l.newProposal(req, kind, itemAny, str(args["question"]), str(args["reason"]), inner)
	return outcome{item: itemID, msg: "제안 " + str(prop["id"])}, nil
}

func (l *Ledger) opProposeWork(req, args, inner obj) (outcome, *Reject) {
	name, rj := cleanWorkName(inner["name"])
	if rj != nil {
		return outcome{}, rj
	}
	words := []any{name}
	if has(inner, "words") {
		words, _ = inner["words"].([]any)
	}
	items, isList := inner["items"].([]any)
	wordsOK := len(words) > 0
	for _, w := range words {
		if !isStr(w) || strings.TrimSpace(str(w)) == "" {
			wordsOK = false
		}
	}
	if !wordsOK {
		return outcome{}, reject("bad_args", "work 제안의 words는 글 목록이어야 합니다")
	}
	itemsOK := isList && len(items) > 0
	for _, i := range items {
		if _, ok := l.items[str(i)]; !ok || !isStr(i) {
			itemsOK = false
		}
	}
	if !itemsOK {
		return outcome{}, reject("bad_args", "work 제안의 items는 실재하는 열린 항목 id 목록이어야 합니다")
	}
	if l.rejectedBefore(obj{"kind": "work", "name": name}) {
		return outcome{}, reject("previously_rejected", "사람이 프로젝트 아님으로 버린 후보입니다")
	}
	var ws []any
	for _, w := range words {
		ws = append(ws, strings.TrimSpace(str(w)))
	}
	ids := map[string]bool{}
	for _, i := range items {
		ids[str(i)] = true
	}
	clean := obj{"name": name, "words": ws, "items": toAnyList(sortedKeys(ids))}
	draft, rj := cleanWorkDesc(inner["desc"])
	if rj != nil {
		return outcome{}, rj
	}
	if draft != "" {
		clean["desc"] = draft
	}
	for _, ex := range l.proposals {
		if str(ex["kind"]) == "work" && normalizeSpace(str(asObj(ex["args"])["name"])) == name {
			return outcome{msg: "이미 제안했습니다 " + str(ex["id"]), noop: true}, nil
		}
	}
	prop := l.newProposal(req, "work", nil, str(args["question"]), str(args["reason"]), clean)
	return outcome{msg: "프로젝트 후보 제안 " + str(prop["id"])}, nil
}

func (l *Ledger) dropProposal(pid any) (obj, *Reject) {
	var p obj
	if s, ok := pid.(string); ok {
		p = l.findProposal(s)
	}
	if p == nil {
		return nil, reject("no_proposal", "제안 %s가 없습니다", str(pid))
	}
	l.removeProposal(p)
	return p, nil
}

func (l *Ledger) acceptWorkProposal(req, prop obj) (outcome, *Reject) {
	inner := asObj(prop["args"])
	args := cloneObj(asObj(req["args"]))
	if args["choice"] == nil {
		args["choice"] = "new"
		if !has(args, "desc") && truthy(inner["desc"]) {
			args["desc"] = inner["desc"]
		}
	}
	work, rj := l.resolveChoice(args, str(inner["name"]), strList(inner["words"]))
	if rj != nil {
		return outcome{}, rj
	}
	if work == "" {
		l.rememberRejection(req, obj{"kind": "work", "name": inner["name"]})
		return outcome{msg: "프로젝트 아님: " + str(inner["name"]) + " 후보를 버립니다"}, nil
	}
	moved := 0
	for _, i := range asList(inner["items"]) {
		it, ok := l.items[str(i)]
		if !ok || str(it["work"]) == work {
			continue
		}
		it["work"] = work
		lock(it, "work")
		bumpRecord(it, dayOf(req), "pin")
		l.changed(it)
		moved++
	}
	return outcome{msg: fmt.Sprintf("프로젝트 %s: 항목 %d건", work, moved)}, nil
}

// resolveChoice는 답 하나를 프로젝트 id(없으면 빈 글, 프로젝트 아님)로 바꾼다.
func (l *Ledger) resolveChoice(args obj, defaultName string, words []string) (string, *Reject) {
	choice := str(args["choice"])
	if choice != "attach" && choice != "new" && choice != "none" {
		return "", reject("bad_args", "choice는 %s 중 하나입니다", pyTuple([]string{"attach", "new", "none"}))
	}
	switch choice {
	case "none":
		return "", nil
	case "attach":
		work := str(args["work"])
		found := false
		for _, w := range asList(l.config["works"]) {
			if str(asObj(w)["id"]) == work {
				found = true
			}
		}
		if !found {
			return "", reject("bad_args", "붙일 프로젝트 %s가 config에 없습니다", pyRepr(work))
		}
		return work, nil
	}
	rawName := args["name"]
	if rawName == nil || rawName == "" {
		rawName = defaultName
	}
	name, rj := cleanWorkName(rawName)
	if rj != nil {
		return "", rj
	}
	if len(words) == 0 {
		words = []string{name}
	}
	desc, rj := cleanWorkDesc(args["desc"])
	if rj != nil {
		return "", rj
	}
	return l.addWork(name, words, desc)
}

func (l *Ledger) removeItem(req, item obj, why, kind string) string {
	iid := str(item["id"])
	day := dayOf(req)
	detail := obj{"title": item["title"], "source": item["source"], "reminder": nil, "relations": 0,
		"unwaited": []any{}, "unparented": []any{}, "proposals": []any{}}
	var kept []obj
	for _, rel := range l.relations {
		if str(rel["from"]) == iid || str(rel["to"]) == iid {
			detail["relations"] = detail["relations"].(int) + 1
		} else {
			kept = append(kept, rel)
		}
	}
	l.relations = kept
	for _, oid := range sortedKeys(l.items) {
		other := l.items[oid]
		if oid == iid {
			continue
		}
		wait := asObj(other["wait"])
		if str(other["status"]) == "waiting" && wait != nil && str(wait["reason"]) == "prior" && str(wait["on"]) == iid {
			before := statusLabel(str(other["status"]), wait)
			other["status"], other["wait"], other["since"] = "todo", nil, day
			recordChange(other, req, "status", before, statusLabels["todo"],
				"선행 "+iid+" '"+shortTitle(str(item["title"]))+"'이 지워짐")
			l.changed(other)
			detail["unwaited"] = append(detail["unwaited"].([]any), oid)
		}
		if str(other["parent"]) == iid {
			other["parent"] = nil
			l.changed(other)
			detail["unparented"] = append(detail["unparented"].([]any), oid)
		}
	}
	for _, prop := range append([]obj(nil), l.proposals...) {
		args := asObj(prop["args"])
		if str(prop["kind"]) == "work" && containsAny(asList(args["items"]), iid) {
			var left []any
			for _, x := range asList(args["items"]) {
				if str(x) != iid {
					left = append(left, x)
				}
			}
			if left == nil {
				left = []any{}
			}
			args["items"] = left
			if len(left) > 0 {
				continue
			}
		} else if !(str(prop["item"]) == iid || (str(prop["kind"]) == "relation" && (str(args["from"]) == iid || str(args["to"]) == iid)) ||
			(str(prop["kind"]) == "parent" && str(args["parent"]) == iid)) {
			continue
		}
		l.removeProposal(prop)
		detail["proposals"] = append(detail["proposals"].([]any), prop["id"])
	}
	delete(l.items, iid)
	delete(l.dirtyItems, iid)
	l.deletedItems[iid] = true
	if kind == "rejected" || (len(asList(item["evidence"])) > 0 && humanVia(str(req["via"]))) {
		var evidence []any
		for _, e := range asList(item["evidence"]) {
			eo := asObj(e)
			evidence = append(evidence, obj{"path": eo["path"], "quote": normalizeSpace(str(eo["quote"]))})
		}
		if evidence == nil {
			evidence = []any{}
		}
		l.rememberRejection(req, obj{"kind": "item", "reason": kind, "item": iid, "title": normalizeSpace(str(item["title"])), "evidence": evidence})
	}
	l.detail = detail
	return "삭제 '" + str(item["title"]) + "': " + why
}

func containsAny(list []any, s string) bool {
	for _, e := range list {
		if str(e) == s {
			return true
		}
	}
	return false
}

func (l *Ledger) opDelete(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	if rj := checkRev(item, req); rj != nil {
		return outcome{}, rj
	}
	if item["reminder"] != nil {
		return outcome{}, reject("reminder_item", "미리알림과 이어진 일은 이 위키에서 지울 수 없습니다")
	}
	return outcome{item: str(item["id"]), msg: l.removeItem(req, item, "Pin에서 지움", "deleted")}, nil
}

func (l *Ledger) opCancelDelete(req obj) (outcome, *Reject) {
	item, rj := l.requireItem(req)
	if rj != nil {
		return outcome{}, rj
	}
	return outcome{item: str(item["id"]), msg: "미리알림에서 지우는 중이 아닙니다", noop: true}, nil
}

func (l *Ledger) opAccept(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	prop, rj := l.dropProposal(args["proposal"])
	if rj != nil {
		return outcome{}, rj
	}
	kind, inner := str(prop["kind"]), asObj(prop["args"])
	if kind == "work" {
		return l.acceptWorkProposal(req, prop)
	}
	sub := cloneObj(req)
	sub["op"], sub["args"], sub["item"], sub["base_rev"] = "", cloneObj(inner), prop["item"], nil
	switch kind {
	case "delete":
		target, ok := l.items[str(prop["item"])]
		if !ok {
			return outcome{item: str(prop["item"]), msg: "이미 없습니다", noop: true}, nil
		}
		return outcome{item: str(target["id"]), msg: l.removeItem(req, target, "미리알림에서 지워진 것을 Pin에서도 지움", "deleted")}, nil
	case "new_item":
		if it, ok := l.items[str(prop["item"])]; ok {
			l.changed(it) // 제안 표시만 뗀다. 일이 움직인 기록이 아니라 마지막 기록은 그대로다
		}
		return outcome{item: str(prop["item"]), msg: "맞음: 제안 표시를 뗍니다"}, nil
	case "status":
		sub["op"], sub["via"] = "set_status", req["via"]
		return l.opSetStatus(sub)
	case "parent":
		item, rj := l.requireItem(sub)
		if rj != nil {
			return outcome{}, rj
		}
		if rj := l.setParent(item, inner["parent"], req); rj != nil {
			return outcome{}, rj
		}
		return outcome{item: str(item["id"]), msg: parentMessage(inner["parent"])}, nil
	}
	rel, rj := l.addRelation(inner, str(prop["by"]), true)
	if rj != nil {
		return outcome{}, rj
	}
	for _, pid := range []string{str(rel["from"]), str(rel["to"])} {
		if _, ok := l.items[pid]; ok {
			l.mark(pid)
		}
	}
	return outcome{item: str(rel["from"]), msg: fmt.Sprintf("관계 %s>%s 수락", str(rel["from"]), str(rel["to"]))}, nil
}

func (l *Ledger) opReject(req obj) (outcome, *Reject) {
	args := asObj(req["args"])
	prop, rj := l.dropProposal(args["proposal"])
	if rj != nil {
		return outcome{}, rj
	}
	kind, inner := str(prop["kind"]), asObj(prop["args"])
	switch kind {
	case "work":
		l.rememberRejection(req, obj{"kind": "work", "name": inner["name"]})
		return outcome{msg: "아님: 프로젝트 후보를 버리고 다시 제안하지 않습니다"}, nil
	case "delete":
		target, ok := l.items[str(prop["item"])]
		if !ok || target["reminder"] == nil {
			return outcome{item: str(prop["item"]), msg: "이미 없습니다", noop: true}, nil
		}
		asObj(target["reminder"])["gone"] = true
		l.changed(target)
		return outcome{item: str(target["id"]), msg: "남김: 미리알림에서 지워진 항목으로 남깁니다"}, nil
	case "new_item":
		item, ok := l.items[str(prop["item"])]
		if !ok {
			return outcome{item: str(prop["item"]), msg: "이미 없습니다", noop: true}, nil
		}
		l.removeItem(req, item, "아님으로 버림", "rejected")
		return outcome{item: str(item["id"]), msg: "아님: 항목을 지우고 다시 제안하지 않습니다"}, nil
	case "relation":
		l.rememberRejection(req, obj{"kind": "relation", "frm": inner["from"], "to": inner["to"], "rel": inner["kind"]})
	default:
		l.rememberRejection(req, obj{"kind": kind, "item": prop["item"], "value": dump(inner, false)})
	}
	return outcome{item: str(prop["item"]), msg: "아님: 제안을 버립니다"}, nil
}

func (l *Ledger) handler(op string) func(obj) (outcome, *Reject) {
	return map[string]func(obj) (outcome, *Reject){
		"add": l.opAdd, "set_status": l.opSetStatus, "touch": l.opTouch, "set_title": l.opSetTitle, "set_owner": l.opSetOwner,
		"set_work": l.opSetWork, "set_work_order": l.opSetWorkOrder, "mark_seen": l.opMarkSeen, "set_work_desc": l.opSetWorkDesc,
		"set_due": l.opSetDue, "set_start": l.opSetStart, "set_parent": l.opSetParent, "update_ai": l.opUpdateAI,
		"add_relation": l.opAddRelation, "remove_relation": l.opRemoveRelation, "confirm_relation": l.opConfirmRelation, "propose": l.opPropose,
		"accept": l.opAccept, "reject": l.opReject, "move_path": l.opMovePath,
		"delete": l.opDelete, "cancel_delete": l.opCancelDelete,
	}[op]
}

// ---------------------------------------------------------------------------
// 한 건 적용
// ---------------------------------------------------------------------------

func requestID(raw any) string {
	if ro, ok := raw.(map[string]any); ok {
		if id, isS := ro["id"].(string); isS && id != "" {
			return id
		}
	}
	sum := sha1.Sum([]byte(dump(raw, false))) //nolint:gosec
	return "invalid:" + hex.EncodeToString(sum[:])[:10]
}

func (l *Ledger) applyOne(raw any, nowISO string) (res obj) {
	rid := requestID(raw)
	if l.logIDs[rid] {
		return obj{"id": rid, "result": "skipped", "code": "already_applied", "message": "log/에 이미 있는 id입니다"}
	}
	ro := asObj(raw)
	base := obj{"id": rid, "applied_at": nowISO, "at": ro["at"], "via": ro["via"], "op": ro["op"], "item": ro["item"]}
	if c, ok := ro["client"].(string); ok {
		base["client"] = c
	}
	snap := l.snapshot()
	var rec obj
	func() {
		defer func() {
			if r := recover(); r != nil {
				l.restore(snap)
				rec = mergeObj(base, obj{"result": "rejected", "code": "bad_args", "message": fmt.Sprintf("요청 형식 오류: %v", r)})
			}
		}()
		req, rj := validateRequest(raw)
		if rj != nil {
			l.restore(snap)
			rec = mergeObj(base, obj{"result": "rejected", "code": rj.Code, "message": rj.Message})
			return
		}
		if _, ok := req["args"]; !ok {
			req["args"] = obj{}
		}
		l.actingVia, l.actingClient = str(req["via"]), req["client"]
		l.detail = nil
		out, rj := l.handler(str(req["op"]))(req)
		if rj == nil {
			l.clearUnseenByHuman(req, out.item)
			if errs := checkLedger(l); len(errs) > 0 {
				if len(errs) > 3 {
					errs = errs[:3]
				}
				rj = reject("ledger_check", "커밋 전 검사 실패: %s", strings.Join(errs, "; "))
			}
		}
		if rj != nil {
			l.restore(snap)
			rec = mergeObj(base, obj{"result": "rejected", "code": rj.Code, "message": rj.Message})
			return
		}
		item := any(base["item"])
		if out.item != "" {
			item = out.item
		}
		code := "ok"
		if out.noop {
			code = "noop"
		}
		rec = mergeObj(base, obj{"item": item, "result": "applied", "code": code, "message": out.msg, "noop": out.noop})
		if l.detail != nil && !out.noop {
			rec["detail"] = l.detail
		}
		if it, ok := l.items[out.item]; ok && out.item != "" {
			rec["rev_after"] = it["rev"]
		}
	}()
	l.addLog(rec)
	return rec
}

func mergeObj(a, b obj) obj {
	out := obj{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func toResult(rec obj) Result {
	r := Result{ID: str(rec["id"]), Result: str(rec["result"]), Code: str(rec["code"]), Message: str(rec["message"])}
	r.Item = str(rec["item"])
	r.Noop = rec["noop"] == true
	return r
}
