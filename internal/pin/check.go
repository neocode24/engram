package pin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 원장 검사. upstream 의 check_config, check_item, check_ledger 와 같은 검사를 한다.
// 달리한 점은 두 가지다(ADR 0106, docs/spec-map.md).
//  1. 프로젝트가 0개인 설정을 받는다. 새 위키에는 프로젝트가 없다.
//  2. 화면에 쓰지 않기로 한 표현 아홉을 검사하지 않는다. upstream 운영자의
//     화면 문체 규칙이라 "스팸 차단 목록 정리" 같은 정상 제목이 거절된다.
//
// 미리알림, Hermes, 캘린더 칸의 상세 검사는 옮기지 않는다. 이 패키지가 그 칸을
// 쓰지 않기 때문이다. origin과 guess는 객체인지만 본다.

func checkConfig(cfg obj) []string {
	var errs []string
	works := asList(cfg["works"])
	if _, ok := cfg["works"].([]any); !ok {
		return []string{"config.works가 목록이 아닙니다"}
	}
	ids := map[string]bool{}
	dup := false
	for _, w := range works {
		wo := asObj(w)
		id := str(wo["id"])
		if ids[id] {
			dup = true
		}
		ids[id] = true
		if !isStr(wo["id"]) || !isStr(wo["name"]) {
			errs = append(errs, fmt.Sprintf("config.works 항목 형식 오류: %s", dump(w, false)))
		} else if _, ok := wo["words"].([]any); !ok {
			errs = append(errs, fmt.Sprintf("config.works 항목 형식 오류: %s", dump(w, false)))
		}
		if d, ok := wo["desc"]; ok && !isStr(d) {
			errs = append(errs, fmt.Sprintf("config.works %q의 desc가 글이 아닙니다", id))
		}
		if id == Unsorted {
			errs = append(errs, "config.works에 예약어 unsorted를 쓸 수 없습니다")
		}
	}
	if dup {
		errs = append(errs, "config.works id 중복")
	}
	if _, ok := cfg["reminder_lists"].([]any); !ok {
		errs = append(errs, "config.reminder_lists가 목록이 아닙니다")
	}
	if sl, ok := cfg["smart_lists"]; ok {
		if _, isObj := sl.(map[string]any); !isObj {
			errs = append(errs, "config.smart_lists가 객체가 아닙니다")
		}
	}
	sig := asObj(cfg["signal"])
	for _, key := range []string{"due_within_days", "waiting_over_days", "work_stale_days"} {
		if !isInt(sig[key]) {
			errs = append(errs, fmt.Sprintf("config.signal.%s가 정수가 아닙니다", key))
		}
	}
	for _, key := range []string{"due_risk_days", "later_days"} {
		if v, ok := sig[key]; ok {
			if n, isI := toInt(v); !isI || n < 0 {
				errs = append(errs, "config.signal."+key+"가 0 이상의 정수가 아닙니다")
			}
		}
	}
	for _, key := range []string{"confirm_stale_days", "archive_after_days"} {
		if !isInt(cfg[key]) {
			errs = append(errs, fmt.Sprintf("config.%s가 정수가 아닙니다", key))
		}
	}
	return errs
}

func isBool(v any) bool { _, ok := v.(bool); return ok }

func keysEqual(o obj, want []string) bool {
	if len(o) != len(want) {
		return false
	}
	for _, k := range want {
		if _, ok := o[k]; !ok {
			return false
		}
	}
	return true
}

func keysSubset(o obj, allowed ...[]string) bool {
	ok := map[string]bool{}
	for _, l := range allowed {
		for _, k := range l {
			ok[k] = true
		}
	}
	for k := range o {
		if !ok[k] {
			return false
		}
	}
	return true
}

func checkItem(item obj, allIDs, works map[string]bool, filenameID string) []string {
	iid := str(item["id"])
	if iid == "" {
		iid = "?"
	}
	var errs []string
	errf := func(format string, args ...any) { errs = append(errs, iid+": "+fmt.Sprintf(format, args...)) }

	if !keysSubset(item, itemKeys, optionalItemKeys) || !hasAll(item, itemKeys) {
		errf("칸이 spec 2.2와 다릅니다(%v)", symmetricKeys(item))
		return errs
	}
	if rb, ok := item["rev_by"]; ok {
		rbo := asObj(rb)
		rev, _ := toInt(item["rev"])
		from, fromOK := toInt(rbo["from_rev"])
		client := rbo["client"]
		if !(rbo != nil && keysEqual(rbo, []string{"via", "client", "from_rev"}) && contains(knownVias, str(rbo["via"])) &&
			(client == nil || isStr(client)) && fromOK && from >= 0 && from < rev) {
			errf("rev_by 형식 오류")
		}
	}
	if u, ok := item["unseen"]; ok && u != true {
		errf("unseen은 true만 씁니다(확인되면 칸을 지웁니다)")
	}
	if fb, ok := item["freed_by"]; ok {
		fo := asObj(fb)
		if !(fo != nil && keysEqual(fo, []string{"by", "date"}) && isStr(fo["by"]) && itemIDRe.MatchString(str(fo["by"])) && isDate(fo["date"])) {
			errf("freed_by 형식 오류")
		}
	}
	if !isStr(item["id"]) || !itemIDRe.MatchString(iid) {
		errf("id 형식이 P0000이 아닙니다")
	}
	if filenameID != "" && iid != filenameID {
		errf("파일 이름 %s와 id가 다릅니다", filenameID)
	}
	if n, ok := toInt(item["rev"]); !ok || n < 1 {
		errf("rev가 1 이상의 정수가 아닙니다")
	}
	if !contains(Sources, str(item["source"])) {
		errf("source가 %v 중 하나가 아닙니다", Sources)
	}
	if !isStr(item["title"]) || strings.TrimSpace(str(item["title"])) == "" {
		errf("title이 비었습니다")
	}
	if !works[str(item["work"])] {
		errf("work %q가 config에 없습니다", str(item["work"]))
	}
	owner := asObj(item["owner"])
	if !(owner != nil && keysEqual(owner, []string{"who", "basis"}) && isStr(owner["who"]) && strings.TrimSpace(str(owner["who"])) != "" && contains(Bases, str(owner["basis"]))) {
		errf("owner 형식 오류")
	}
	status := str(item["status"])
	if !contains(Statuses, status) {
		errf("status가 %v 중 하나가 아닙니다", Statuses)
	}
	wait := item["wait"]
	if status == "waiting" {
		wo := asObj(wait)
		switch {
		case wo == nil || !contains(WaitReasons, str(wo["reason"])) || !keysSubset(wo, []string{"reason", "on"}):
			errf("대기인데 wait가 없거나 형식이 틀립니다")
		case str(wo["reason"]) == "prior" && !allIDs[str(wo["on"])]:
			errf("선행 대기의 on이 실재하는 항목 id가 아닙니다")
		default:
			if on, ok := wo["on"]; ok && on != nil && !isStr(on) {
				errf("wait.on이 글이 아닙니다")
			}
		}
	} else if wait != nil {
		errf("대기가 아닌데 wait가 있습니다")
	}
	for _, key := range []string{"since", "created"} {
		if !isDate(item[key]) {
			errf("%s가 날짜가 아닙니다", key)
		}
	}
	if due := item["due"]; due != nil {
		do := asObj(due)
		if !(do != nil && keysEqual(do, []string{"date", "basis"}) && isDate(do["date"]) && isStr(do["basis"])) {
			errf("due 형식 오류")
		}
	}
	if st, ok := item["start"]; ok {
		do := asObj(item["due"])
		switch {
		case !isDate(st):
			errf("start가 날짜가 아닙니다")
		case do == nil || !isDate(do["date"]):
			errf("start는 기한이 있을 때만 씁니다")
		case str(st) > str(do["date"]):
			errf("start가 기한보다 늦습니다")
		}
	}
	var changes []any
	if c, ok := item["changes"]; ok {
		if l, isList := c.([]any); isList {
			changes = l
		} else {
			changes = []any{nil}
		}
	}
	for _, ch := range changes {
		co := asObj(ch)
		ok := co != nil && keysEqual(co, changeKeys) && isDate(co["date"]) && contains(changeFields, str(co["field"])) && contains(knownVias, str(co["via"]))
		if ok {
			for _, k := range []string{"from", "to", "note"} {
				if v := co[k]; v != nil && !isStr(v) {
					ok = false
				}
			}
			if f := str(co["field"]); f == "due" || f == "start" {
				for _, k := range []string{"from", "to"} {
					if v := co[k]; v != nil && !isDate(v) {
						ok = false
					}
				}
			}
		}
		if !ok {
			errf("changes 형식 오류: %s", dump(ch, false))
		}
	}
	if parent := item["parent"]; parent != nil {
		if p := str(parent); !allIDs[p] || p == iid {
			errf("parent %q가 실재하지 않거나 자기 자신입니다", p)
		}
	}
	rem := item["reminder"]
	if rem != nil {
		ro := asObj(rem)
		if !(ro != nil && isStr(ro["id"]) && isStr(ro["list"]) && keysSubset(ro, []string{"id", "list", "completed_seen", "modified_seen", "write_pending", "gone"})) {
			errf("reminder 형식 오류")
		}
	}
	if str(item["source"]) == "reminder" && rem == nil {
		errf("source가 reminder인데 reminder 칸이 없습니다")
	}
	if o, ok := item["origin"]; ok && asObj(o) == nil {
		errf("origin 형식 오류")
	}
	if g, ok := item["guess"]; ok && asObj(g) == nil {
		errf("guess 형식 오류")
	}
	if _, hasOrigin := item["origin"]; (str(item["source"]) == "hermes") != hasOrigin {
		errf("origin은 source가 hermes인 항목에만 있고, hermes 항목에는 반드시 있어야 합니다")
	}
	for _, e := range asList(item["evidence"]) {
		eo := asObj(e)
		ok := eo != nil && keysEqual(eo, evidenceKeys) && contains(Stages, str(eo["stage"])) && isDate(eo["date"]) &&
			isStr(eo["path"]) && isStr(eo["section"]) && isStr(eo["quote"]) && strings.TrimSpace(str(eo["quote"])) != ""
		if !ok {
			errf("evidence 형식 오류: %s", dump(e, false))
		} else if first, _, _ := strings.Cut(str(eo["path"]), "/"); first != str(eo["stage"]) {
			errf("evidence stage %s가 경로 %s와 다릅니다", str(eo["stage"]), str(eo["path"]))
		}
	}
	var dates []string
	for _, m := range asList(item["mentions"]) {
		mo := asObj(m)
		ok := mo != nil && keysEqual(mo, mentionKeys) && isDate(mo["date"]) && isStr(mo["path"]) && isStr(mo["quote"]) && strings.TrimSpace(str(mo["quote"])) != ""
		switch {
		case !ok:
			errf("mentions 형식 오류: %s", dump(m, false))
		case strings.HasPrefix(str(mo["path"]), NotePathPrefix) && str(mo["path"]) != "note:web" && str(mo["path"]) != "note:command":
			errf("mentions 메모 경로가 [note:web note:command] 중 하나가 아닙니다: %q", str(mo["path"]))
		default:
			dates = append(dates, str(mo["date"]))
		}
	}
	if !sort.StringsAreSorted(dates) {
		errf("mentions가 시간순이 아닙니다")
	}
	ctx := asObj(item["context"])
	ctxOK := ctx != nil && keysEqual(ctx, contextKeys)
	if ctxOK {
		for _, v := range ctx {
			if v != nil && !isStr(v) {
				ctxOK = false
			}
		}
	}
	if !ctxOK {
		errf("context 형식 오류")
	}
	last := asObj(item["last_record"])
	if !(last != nil && keysEqual(last, []string{"date", "kind"}) && isDate(last["date"]) && contains(RecordKinds, str(last["kind"]))) {
		errf("last_record 형식 오류")
	}
	locked, isList := item["locked"].([]any)
	lockOK := isList
	seen := map[string]bool{}
	for _, x := range locked {
		s := str(x)
		if !contains(lockable, s) || seen[s] {
			lockOK = false
		}
		seen[s] = true
	}
	if !lockOK {
		errf("locked 형식 오류")
	}
	closed := item["closed_at"]
	if status == "done" {
		if !isDate(closed) {
			errf("완료인데 closed_at이 없습니다")
		}
	} else if closed != nil {
		errf("완료가 아닌데 closed_at이 있습니다")
	}
	return errs
}

// knownVias는 원장에 이미 적혀 있을 수 있는 요청 경로 이름이다. upstream 원장을
// 그대로 읽을 수 있도록 이 패키지가 받지 않는 경로 이름도 저장된 값으로는 인정한다.
var knownVias = []string{"web", "command", "reminders", "extract", "reaction", "judge"}

func hasAll(o obj, keys []string) bool {
	for _, k := range keys {
		if _, ok := o[k]; !ok {
			return false
		}
	}
	return true
}

func symmetricKeys(item obj) []string {
	have := map[string]bool{}
	for k := range item {
		have[k] = true
	}
	for _, k := range optionalItemKeys {
		delete(have, k)
	}
	want := setOf(itemKeys...)
	var out []string
	for k := range have {
		if !want[k] {
			out = append(out, k)
		}
	}
	for k := range want {
		if !have[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func relationKey(rel obj) string {
	if str(rel["kind"]) == "before" {
		return str(rel["from"]) + ">" + str(rel["to"]) + ">before"
	}
	a, b := str(rel["from"]), str(rel["to"])
	if b < a {
		a, b = b, a
	}
	return a + ">" + b + ">" + str(rel["kind"])
}

func checkLedger(l *Ledger) []string {
	errs := checkConfig(l.config)
	if len(errs) > 0 {
		return errs
	}
	allIDs := l.allItemIDs()
	works := l.workIDs()
	for _, id := range sortedKeys(l.items) {
		if _, both := l.archived[id]; both {
			errs = append(errs, fmt.Sprintf("%s: 항목과 보관에 id가 겹칩니다", id))
		}
	}
	for _, id := range sortedKeys(l.items) {
		errs = append(errs, checkItem(l.items[id], allIDs, works, id)...)
	}
	seen := map[string]bool{}
	for _, rel := range l.relations {
		tag := fmt.Sprintf("관계 %s>%s", str(rel["from"]), str(rel["to"]))
		if !keysEqual(rel, []string{"from", "to", "kind", "reason", "basis", "by", "confirmed"}) {
			errs = append(errs, tag+": 칸이 spec 2.2와 다릅니다")
			continue
		}
		kind := str(rel["kind"])
		if !contains(RelationKind, kind) {
			errs = append(errs, fmt.Sprintf("%s: kind %q", tag, kind))
			continue
		}
		if !allIDs[str(rel["from"])] || !allIDs[str(rel["to"])] {
			errs = append(errs, tag+": 양쪽 항목이 실재하지 않습니다")
		}
		if str(rel["from"]) == str(rel["to"]) {
			errs = append(errs, tag+": 자기 자신과의 관계")
		}
		if kind == "related" && !humanVia(str(rel["by"])) && !(isStr(rel["reason"]) && strings.TrimSpace(str(rel["reason"])) != "") {
			errs = append(errs, tag+": related에 이유가 없습니다")
		}
		if !contains(Bases, str(rel["basis"])) || !isBool(rel["confirmed"]) {
			errs = append(errs, tag+": basis 또는 confirmed 형식 오류")
		}
		key := relationKey(rel)
		if seen[key] {
			errs = append(errs, tag+": 같은 관계가 두 번 적혔습니다")
		}
		seen[key] = true
	}
	pids := map[string]bool{}
	for _, p := range l.proposals {
		id := str(p["id"])
		if pids[id] || !proposalIDRe.MatchString(id) {
			errs = append(errs, fmt.Sprintf("제안 id 오류 또는 중복: %q", id))
		}
		pids[id] = true
		if !contains(ProposalKinds, str(p["kind"])) {
			errs = append(errs, fmt.Sprintf("제안 %s: kind %q", id, str(p["kind"])))
		}
		if item := p["item"]; item != nil && !allIDs[str(item)] {
			errs = append(errs, fmt.Sprintf("제안 %s: 항목 %s가 실재하지 않습니다", id, str(item)))
		}
	}
	return errs
}

// docText는 근거 대조에 쓸 문서 글을 읽는다. 아직 디스크에 없는 문서는 overlay 의
// 본문을 쓴다. 경로가 폴더면 그 아래 텍스트 파일을 200개까지 이어 붙인다.
func (l *Ledger) docText(path string) (string, bool) {
	if text, ok := l.docs[path]; ok {
		return text, true
	}
	target := filepath.Join(l.Wiki, filepath.FromSlash(path))
	st, err := os.Stat(target)
	if err != nil {
		return "", false
	}
	var files []string
	if st.Mode().IsRegular() {
		files = []string{target}
	} else {
		_ = filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
			if err == nil && d.Type().IsRegular() {
				files = append(files, p)
			}
			return nil
		})
		sort.Strings(files)
	}
	if len(files) > 200 {
		files = files[:200]
	}
	var chunks []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil || !validUTF8(data) {
			continue
		}
		chunks = append(chunks, string(data))
	}
	return strings.Join(chunks, "\n"), true
}

// pathExists는 문서가 디스크나 overlay 에 있는지 본다.
func (l *Ledger) pathExists(path string) bool {
	if _, ok := l.docs[path]; ok {
		return true
	}
	_, err := os.Stat(filepath.Join(l.Wiki, filepath.FromSlash(path)))
	return err == nil
}

func unsafePath(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

type quoteRef struct{ path, quote string }

// verifyQuotes는 근거와 언급된 곳의 경로가 실재하고 인용문이 그 문서에 있는지 본다
// (공백만 정규화해 대조). 어긋나면 Reject 를 낸다. reminder: 와 note: 접두사는
// 문서 근거가 아니므로 거절한다. 이 패키지는 미리알림에서 온 요청을 받지 않는다.
func (l *Ledger) verifyQuotes(refs []quoteRef, label string) *Reject {
	cache := map[string]string{}
	for _, r := range refs {
		switch {
		case strings.HasPrefix(r.path, ReminderPathPrefix):
			return reject("quote_path", "%s: 미리알림 경로는 미리알림에서 온 요청만 쓸 수 있습니다: %s", label, r.path)
		case strings.HasPrefix(r.path, NotePathPrefix):
			return reject("quote_path", "%s: 메모 경로는 사람이 남긴 메모에만 쓰고 문서 근거로 쓸 수 없습니다: %s", label, r.path)
		case unsafePath(r.path):
			return reject("quote_path", "%s: 경로가 저장소 밖을 가리킵니다: %s", label, r.path)
		}
		text, ok := cache[r.path]
		if !ok {
			t, exists := l.docText(r.path)
			if !exists {
				return reject("quote_path", "%s: 근거 문서가 없습니다: %s", label, r.path)
			}
			text = normalizeSpace(t)
			cache[r.path] = text
		}
		if !strings.Contains(text, normalizeSpace(r.quote)) {
			q := []rune(r.quote)
			if len(q) > 40 {
				q = q[:40]
			}
			return reject("quote_missing", "%s: 인용문이 문서에 없습니다: %s \"%s\"", label, r.path, string(q))
		}
	}
	return nil
}
