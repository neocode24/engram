package pin

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"net/url"
)

// 상태 생성. upstream 의 derive.build_state 를 옮겼다. 원장을 읽기만 한다.
//
// upstream 과 달리하는 점(ADR 0106, docs/spec-map.md)
//   - 상태의 주소를 지우지 않는다(scrub_urls). 상태가 이 기계를 떠나지 않는다.
//   - 문서 링크는 serve 가 이미 보여 주는 context 문서에만 단다. inbox 와 sources 는
//     제목, 경로, 절, 인용을 글자로만 보인다.
//   - 화면에 쓰지 않는 표현 검사를 하지 않는다.
//   - 미리알림, 캘린더, Hermes 질문은 만들지 않는다. 해당 칸은 upstream 과 같은 값으로 둔다.

var (
	frontTitleRe = regexp.MustCompile(`(?m)^title:[ \t]*(.+?)[ \t]*$`)
	bodyH1Re     = regexp.MustCompile(`(?m)^#[ \t]+(.+?)[ \t]*$`)
	datePrefixRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	itemIDInText = regexp.MustCompile(`P\d{4}`)
	swapRe       = regexp.MustCompile(`(P\d{4})([과와이가을를은는])?`)
)

func validUTF8(b []byte) bool { return utf8.Valid(b) }

// docTitle은 문서 제목을 낸다. 파일 frontmatter 의 title, 없으면 frontmatter 다음의
// 첫 `# ` 제목줄(앞 4096자 안), 그것도 없으면 파일 이름에서 날짜 접두사와 확장자를 뗀 것이다.
func docTitle(wiki, path string) string {
	name := datePrefixRe.ReplaceAllString(filepath.Base(path), "")
	name = strings.TrimSuffix(name, ".md")
	var title, heading string
	if data, err := os.ReadFile(filepath.Join(wiki, filepath.FromSlash(path))); err == nil && utf8.Valid(data) {
		head := []rune(string(data))
		if len(head) > 4096 {
			head = head[:4096]
		}
		text := string(head)
		body := text
		if strings.HasPrefix(text, "---") {
			block, rest, _ := strings.Cut(text[3:], "\n---")
			_, after, _ := strings.Cut(rest, "\n")
			body = after
			if m := frontTitleRe.FindStringSubmatch(block); m != nil {
				title = normalizeSpace(strings.Trim(m[1], "\"'"))
			}
		}
		if m := bodyH1Re.FindStringSubmatch(body); m != nil {
			heading = normalizeSpace(strings.TrimSpace(m[1]))
		}
	}
	for _, c := range []string{title, heading, name} {
		if c != "" {
			return c
		}
	}
	return "제목 생략"
}

// docURL은 serve 가 보여 주는 문서(context 의 마크다운)에만 링크 주소를 낸다.
func docURL(path string) any {
	if unsafePath(path) || !strings.HasSuffix(path, ".md") || !strings.HasPrefix(path, "context/") {
		return nil
	}
	return (&url.URL{Path: "/doc/" + path}).String()
}

func evidenceOrder(ev obj) int {
	if strings.HasPrefix(str(ev["path"]), "sources/transcripts/") {
		return 1
	}
	return 0
}

func mentionKind(path string) string {
	if strings.HasPrefix(path, ReminderPathPrefix) {
		return "reminder"
	}
	if strings.HasPrefix(path, NotePathPrefix) {
		return "note"
	}
	return "wiki"
}

type docRef struct {
	title string
	url   any
}

func refOf(wiki, path string, cache map[string]docRef) docRef {
	if r, ok := cache[path]; ok {
		return r
	}
	r := docRef{title: docTitle(wiki, path), url: docURL(path)}
	cache[path] = r
	return r
}

func lastNoteOf(item obj) any {
	var notes []obj
	for _, m := range asList(item["mentions"]) {
		if mo := asObj(m); mentionKind(str(mo["path"])) == "note" {
			notes = append(notes, mo)
		}
	}
	if len(notes) == 0 {
		return nil
	}
	n := notes[len(notes)-1]
	return obj{"date": n["date"], "quote": n["quote"]}
}

func daysBetween(today time.Time, d any) any {
	s, ok := d.(string)
	if !ok {
		return nil
	}
	return int(today.Sub(parseDate(s)).Hours() / 24)
}

func daysInt(today time.Time, d string) int {
	return int(today.Sub(parseDate(d)).Hours() / 24)
}

func ownerLabel(owner obj) string {
	who, basis := str(owner["who"]), str(owner["basis"])
	switch {
	case who == "me" && basis == "stated":
		return "내 할 일"
	case who == "me":
		return "내 할 일(추정)"
	case who == "unknown":
		return "담당자 미정"
	case basis == "stated":
		return who
	}
	return who + "(추정)"
}

func dueInfo(item obj, today time.Time, within int) any {
	due := asObj(item["due"])
	if due == nil {
		return nil
	}
	left := daysInt(today, str(due["date"]))
	left = -left // today 가 앞이면 days_between 이 양수이므로 부호를 뒤집어 기한까지 남은 날로 둔다
	var state string
	switch {
	case str(item["status"]) == "done":
		state = "closed"
	case left < 0:
		state = "overdue"
	case left <= within:
		state = "soon"
	default:
		state = "later"
	}
	return obj{"date": due["date"], "basis": due["basis"], "days_left": left, "state": state}
}

func sigInt(sig obj, key string) int {
	n, _ := toInt(sig[key])
	return n
}

// weekEnd는 이번 주의 끝(일요일)이다. 오늘 탭의 '이번 주 해야 할 일'과 같은 범위다.
func weekEnd(today time.Time) time.Time {
	return today.AddDate(0, 0, 6-(int(today.Weekday())+6)%7)
}

// laterFrom은 예정(upstream derive.later_from)이다. 할 일의 시작일이 오늘 뒤면 그
// 날짜를 돌려주고, 아니면 nil 이다. 시작일은 사람이 정한 start, 없으면 기한의
// later_days 일 전(자동)이다. 진행 중, 대기, 보류, 기한 없는 일은 예정이 아니다.
func laterFrom(item obj, today time.Time, sig obj) any {
	due := asObj(item["due"])
	if str(item["status"]) != "todo" || due == nil {
		return nil
	}
	var start time.Time
	if s := str(item["start"]); s != "" {
		start = parseDate(s)
	} else {
		days := defaultLaterDays
		if v, ok := sig["later_days"]; ok {
			days, _ = toInt(v)
		}
		start = parseDate(str(due["date"])).AddDate(0, 0, -days)
	}
	if start.After(today) {
		return start.Format("2006-01-02")
	}
	return nil
}

func itemSignalReasons(item obj, today time.Time, sig obj) (risk, caution []string) {
	riskDays := defaultDueRiskDays
	if v, ok := sig["due_risk_days"]; ok {
		riskDays, _ = toInt(v)
	}
	status := str(item["status"])
	if due := asObj(item["due"]); (status == "todo" || status == "doing" || status == "waiting") && due != nil {
		dueDate := parseDate(str(due["date"]))
		left := -daysInt(today, str(due["date"]))
		switch {
		case left < 0:
			risk = append(risk, str(item["id"])+" 기한이 "+itoa(-left)+"일 지났습니다")
		case left <= riskDays && (status == "todo" || status == "waiting"):
			risk = append(risk, str(item["id"])+" 기한이 "+itoa(left)+"일 남았습니다")
		case !dueDate.After(weekEnd(today)):
			caution = append(caution, str(item["id"])+" 기한이 "+itoa(left)+"일 남았습니다(이번 주)")
		}
	}
	if status == "waiting" {
		waited := daysInt(today, str(item["since"]))
		if waited > sigInt(sig, "waiting_over_days") {
			caution = append(caution, str(item["id"])+" 대기가 "+itoa(waited)+"일째입니다")
		}
	}
	return
}

func workSignal(items []obj, today time.Time, sig obj) obj {
	var allOpen, open []obj
	for _, i := range items {
		if str(i["status"]) != "done" {
			allOpen = append(allOpen, i)
			if laterFrom(i, today, sig) == nil { // 예정인 일은 신호에서 뺀다
				open = append(open, i)
			}
		}
	}
	var risk, caution []string
	for _, item := range open {
		r, c := itemSignalReasons(item, today, sig)
		risk = append(risk, r...)
		caution = append(caution, c...)
	}
	var last any
	lastStr := ""
	for _, i := range items {
		if laterFrom(i, today, sig) != nil {
			continue
		}
		if d := str(asObj(i["last_record"])["date"]); d != "" && (lastStr == "" || d > lastStr) {
			lastStr = d
		}
	}
	if lastStr != "" {
		last = lastStr
	}
	since := daysBetween(today, last)
	stale := len(open) > 0 && since != nil && since.(int) > sigInt(sig, "work_stale_days")
	if stale {
		caution = append(caution, "프로젝트의 마지막 기록이 "+itoa(since.(int))+"일 전입니다")
	}
	level := "normal"
	switch {
	case len(allOpen) == 0:
		level = "none"
	case len(risk) > 0:
		level = "risk"
	case len(caution) > 0:
		level = "caution"
	}
	reasons := caution
	if level == "risk" {
		reasons = risk
	}
	return obj{
		"signal": level, "signal_label": signalLabels[level], "reasons": toAnyList(reasons),
		"last_record": last, "days_since_record": since, "open_count": len(allOpen), "later_count": len(allOpen) - len(open),
	}
}

func (l *Ledger) titleOf(id string) string {
	if it, ok := l.items[id]; ok {
		return str(it["title"])
	}
	if it, ok := l.archived[id]; ok {
		return str(it["title"])
	}
	return id
}

func (l *Ledger) allTitles() map[string]string {
	out := map[string]string{}
	for id, it := range l.items {
		out[id] = str(it["title"])
	}
	for id, it := range l.archived {
		out[id] = str(it["title"])
	}
	return out
}

func (l *Ledger) relationsOf(itemID string) []any {
	titles := l.allTitles()
	var out []obj
	for _, rel := range l.relations {
		if itemID != str(rel["from"]) && itemID != str(rel["to"]) {
			continue
		}
		side, other := "to", str(rel["from"])
		if str(rel["from"]) == itemID {
			side, other = "from", str(rel["to"])
		}
		out = append(out, obj{
			"other": other, "other_title": titles[other], "kind": rel["kind"],
			"label": relationLabel(str(rel["kind"]), side), "reason": rel["reason"],
			"basis": rel["basis"], "confirmed": rel["confirmed"],
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := indexOf(RelationKind, str(out[i]["kind"])), indexOf(RelationKind, str(out[j]["kind"]))
		if ki != kj {
			return ki < kj
		}
		return str(out[i]["other"]) < str(out[j]["other"])
	})
	list := make([]any, len(out))
	for i, o := range out {
		list[i] = o
	}
	return list
}

func indexOf(list []string, s string) int {
	for i, e := range list {
		if e == s {
			return i
		}
	}
	return -1
}

// titledQuestion은 제안 질문에 항목 번호(P0003 꼴)만 나열돼 있으면 「제목」으로 바꾼다.
func (l *Ledger) titledQuestion(q string) string {
	if !itemIDInText.MatchString(q) {
		return q
	}
	pairs := map[string]string{"과": "과/와", "와": "과/와", "이": "이/가", "가": "이/가", "을": "을/를", "를": "을/를", "은": "은/는", "는": "은/는"}
	return swapRe.ReplaceAllStringFunc(q, func(m string) string {
		sub := swapRe.FindStringSubmatch(m)
		id, tail := sub[1], sub[2]
		title := shortTitle(l.titleOf(id))
		pair, ok := pairs[tail]
		if !ok {
			return "「" + title + "」" + tail
		}
		j, _ := josa(title, pair)
		return "「" + title + "」" + j
	})
}

func (l *Ledger) proposalQuestion(p obj) string {
	if str(p["kind"]) == "relation" {
		a := asObj(p["args"])
		return relationQuestion(str(a["kind"]), l.titleOf(str(a["from"])), l.titleOf(str(a["to"])))
	}
	return l.titledQuestion(str(p["question"]))
}

var classifyActions = []any{"attach", "new", "none"}

func (l *Ledger) buildAsks(today time.Time, items []obj) []any {
	staleDays, _ := toInt(l.config["confirm_stale_days"])
	var asks []any
	// 새 일 제안이 열려 있는 일은 담당자 확인을 따로 묻지 않는다. 제안 카드의 담당자 줄이 묻는다.
	proposedNew := map[string]bool{}
	for _, p := range l.proposals {
		if str(p["kind"]) == "new_item" && p["item"] != nil {
			proposedNew[str(p["item"])] = true
		}
	}
	for _, p := range l.proposals {
		args := asObj(p["args"])
		if str(p["kind"]) == "work" {
			var its []any
			for _, i := range asList(args["items"]) {
				if _, ok := l.items[str(i)]; ok {
					its = append(its, i)
				}
			}
			if its == nil {
				its = []any{}
			}
			desc := args["desc"]
			if desc == nil {
				desc = ""
			}
			asks = append(asks, obj{
				"id": "proposal:" + str(p["id"]), "kind": "classify", "source_kind": "wiki", "proposal": p["id"], "proposal_kind": "work",
				"items": its, "question": p["question"], "reason": p["reason"],
				"actions": classifyActions, "item_title": nil, "default_name": args["name"], "default_desc": desc,
			})
			continue
		}
		var target obj
		if it := p["item"]; it != nil {
			target = l.items[str(it)]
		}
		var its []any
		switch {
		case p["item"] != nil:
			its = []any{p["item"]}
		case str(p["kind"]) == "relation":
			its = []any{args["from"], args["to"]}
		default:
			its = []any{}
		}
		var title any
		if target != nil {
			title = target["title"]
		}
		asks = append(asks, obj{
			"id": "proposal:" + str(p["id"]), "kind": "proposal", "proposal": p["id"], "proposal_kind": p["kind"],
			"items": its, "question": l.proposalQuestion(p), "reason": p["reason"],
			"actions": []any{"accept", "reject"}, "item_title": title,
		})
	}
	for _, item := range items {
		if str(item["status"]) == "done" {
			continue
		}
		n, hasN := item["days_since_record"].(int)
		if str(item["status"]) != "hold" && item["later"] == nil && hasN && n > staleDays { // 예정인 일은 기록이 끊겨도 묻지 않는다
			lr := asObj(item["last_record"])
			asks = append(asks, obj{
				"id": "stale:" + str(item["id"]), "kind": "stale", "items": []any{item["id"]},
				"question": str(item["title"]) + ": 마지막 기록이 " + itoa(n) + "일 전입니다. 지금 어떤 상태입니까?",
				"reason":   "마지막 기록 " + str(lr["date"]) + "(" + recordLabels[str(lr["kind"])] + ")",
				"actions":  []any{"touch", "done", "hold"},
			})
		}
		if str(asObj(item["owner"])["basis"]) == "inferred" && !proposedNew[str(item["id"])] {
			asks = append(asks, obj{
				"id": "owner:" + str(item["id"]), "kind": "owner", "items": []any{item["id"]},
				"question": str(item["title"]) + ": 담당자가 정해지지 않았습니다",
				"reason":   "문서에 담당자가 적혀 있지 않아 추정했습니다",
				"actions":  []any{"owner_me", "owner_other", "owner_unknown"},
			})
		}
	}
	// 선행 완료로 대기에서 할 일로 바뀐 일마다 카드 하나. 확인(mark_seen)이나 그 일에 사람이 손대면 사라진다.
	for _, iid := range sortedKeys(l.items) {
		it := l.items[iid]
		fb := asObj(it["freed_by"])
		if fb == nil || str(it["status"]) == "done" {
			continue
		}
		by := str(fb["by"])
		asks = append(asks, obj{
			"id": "freed:" + iid, "kind": "freed", "items": []any{iid}, "prior": by,
			"question": str(it["title"]) + ": 선행 " + by + "이 완료되어 대기에서 할 일로 바뀌었습니다",
			"reason":   "선행 " + by + "이 완료되어 대기에서 할 일로 바뀌었습니다",
			"actions":  []any{"seen", "doing"},
		})
	}
	return asks
}

func (l *Ledger) doneDaily(today time.Time) []any {
	counts := map[string]int{}
	var days []string
	for n := doneDailyDays - 1; n >= 0; n-- {
		d := today.AddDate(0, 0, -n).Format("2006-01-02")
		days = append(days, d)
		counts[d] = 0
	}
	for _, it := range l.items {
		if str(it["status"]) == "done" {
			if _, ok := counts[str(it["closed_at"])]; ok {
				counts[str(it["closed_at"])]++
			}
		}
	}
	out := make([]any, len(days))
	for i, d := range days {
		out[i] = obj{"date": d, "count": counts[d]}
	}
	return out
}

func (l *Ledger) changeOf(r obj) obj {
	return obj{
		"id": r["id"], "result": r["result"], "code": r["code"], "message": r["message"],
		"item": r["item"], "op": r["op"], "via": r["via"], "at": r["applied_at"],
	}
}

func toObjAny(list []obj) []any {
	out := make([]any, len(list))
	for i, o := range list {
		out[i] = o
	}
	return out
}

// buildState는 화면이 읽는 상태를 만든다. today 는 now 의 날짜이고 nowISO 는 generated_at 이다.
func (l *Ledger) buildState(today time.Time, nowISO string) obj {
	cfg := l.config
	sig := obj{"due_risk_days": defaultDueRiskDays}
	for k, v := range asObj(cfg["signal"]) {
		sig[k] = v
	}
	proposed := map[string]bool{}
	for _, p := range l.proposals {
		if str(p["kind"]) == "new_item" && p["item"] != nil {
			proposed[str(p["item"])] = true
		}
	}
	children := map[string][]string{}
	for _, id := range sortedKeys(l.items) {
		if parent := l.items[id]["parent"]; parent != nil {
			children[str(parent)] = append(children[str(parent)], id)
		}
	}
	titles := l.allTitles()
	refs := map[string]docRef{}
	visibleDays, _ := toInt(cfg["archive_after_days"])
	staleDays, _ := toInt(cfg["confirm_stale_days"])
	within := sigInt(sig, "due_within_days")

	var itemsOut []obj
	for _, iid := range sortedKeys(l.items) {
		it := l.items[iid]
		status := str(it["status"])
		if status == "done" && daysInt(today, str(it["closed_at"])) > visibleDays {
			continue
		}
		var waitOut any
		if w := asObj(it["wait"]); w != nil {
			on := w["on"]
			var onTitle any
			if s, ok := on.(string); ok {
				if t, found := titles[s]; found {
					onTitle = t
				}
			}
			waitOut = obj{"reason": w["reason"], "reason_label": waitLabels[str(w["reason"])], "on": on, "on_title": onTitle}
		}
		lr := asObj(it["last_record"])
		n := daysBetween(today, lr["date"])
		var evs []obj
		for _, e := range asList(it["evidence"]) {
			evs = append(evs, asObj(e))
		}
		sort.SliceStable(evs, func(i, j int) bool { return evidenceOrder(evs[i]) < evidenceOrder(evs[j]) })
		evOut := make([]any, len(evs))
		for i, e := range evs {
			ref := refOf(l.Wiki, str(e["path"]), refs)
			o := cloneObj(e)
			o["title"], o["url"] = ref.title, ref.url
			evOut[i] = o
		}
		mnOut := []any{}
		for _, m := range asList(it["mentions"]) {
			mo := cloneObj(asObj(m))
			mo["kind"] = mentionKind(str(mo["path"]))
			if mo["kind"] == "wiki" {
				ref := refOf(l.Wiki, str(mo["path"]), refs)
				mo["title"], mo["url"] = ref.title, ref.url
			} else {
				mo["title"], mo["url"] = nil, nil
			}
			mnOut = append(mnOut, mo)
		}
		ch := children[iid]
		if ch == nil {
			ch = []string{}
		}
		changes := asList(it["changes"])
		if changes == nil {
			changes = []any{}
		}
		ctx := asObj(it["context"])
		var reminder any
		if rem := asObj(it["reminder"]); rem != nil {
			r := cloneObj(rem)
			r["notes"], r["notes_truncated"], r["due"], r["due_note"], r["subtasks"], r["tags"] = "", false, nil, nil, 0, []any{}
			reminder = r
		}
		later := laterFrom(it, today, sig)
		needs := status != "done" && status != "hold" && later == nil && n != nil && n.(int) > staleDays
		lockedList := asList(it["locked"])
		if lockedList == nil {
			lockedList = []any{}
		}
		itemsOut = append(itemsOut, obj{
			"id": iid, "url": DefaultPageURL + "#" + iid, "rev": it["rev"], "title": it["title"], "work": it["work"],
			"source": it["source"], "source_label": sourceLabels[str(it["source"])],
			"owner":  obj{"who": asObj(it["owner"])["who"], "basis": asObj(it["owner"])["basis"], "label": ownerLabel(asObj(it["owner"]))},
			"status": status, "status_label": statusLabels[status], "wait": waitOut,
			"since": it["since"], "days_in_status": daysBetween(today, it["since"]),
			"due":    dueInfo(it, today, within),
			"parent": it["parent"], "children": toAnyList(ch),
			"reminder": reminder,
			"changes":  changes,
			"evidence": evOut, "mentions": mnOut,
			"context":     obj{"why": ctx["why"], "now": ctx["now"], "next": ctx["next"]},
			"last_note":   lastNoteOf(it),
			"last_record": lr, "last_record_label": recordLabels[str(lr["kind"])],
			"days_since_record": n,
			"later":             later,
			"start":             it["start"],
			"needs_check":       needs,
			"proposed":          proposed[iid], "locked": lockedList, "created": it["created"], "closed_at": it["closed_at"],
			"unseen":    it["unseen"] == true,
			"freed_by":  it["freed_by"],
			"relations": l.relationsOf(iid),
		})
	}

	var worksOut []obj
	allWorks := append(append([]any{}, asList(cfg["works"])...), obj{"id": Unsorted, "name": "미분류", "words": []any{}})
	for _, w := range allWorks {
		wo := asObj(w)
		var its []obj
		for _, id := range sortedKeys(l.items) {
			if str(l.items[id]["work"]) == str(wo["id"]) {
				its = append(its, l.items[id])
			}
		}
		if len(its) == 0 && str(wo["id"]) == Unsorted {
			continue
		}
		info := workSignal(its, today, sig)
		desc := wo["desc"]
		if desc == nil {
			desc = ""
		}
		counts := obj{}
		for _, s := range Statuses {
			c := 0
			for _, i := range its {
				if str(i["status"]) == s {
					c++
				}
			}
			counts[s] = c
		}
		info["id"], info["name"], info["desc"], info["counts"] = wo["id"], wo["name"], desc, counts
		worksOut = append(worksOut, info)
	}

	var logTail []obj
	for _, r := range l.logRecords {
		if res := str(r["result"]); res == "applied" || res == "rejected" {
			logTail = append(logTail, r)
		}
	}
	if len(logTail) > 50 {
		logTail = logTail[len(logTail)-50:]
	}
	recent := make([]any, len(logTail))
	for i, r := range logTail {
		recent[i] = l.changeOf(r)
	}
	workOf := map[string]string{}
	for _, i := range itemsOut {
		workOf[str(i["id"])] = str(i["work"])
	}
	perWork := map[string][]any{}
	for i := len(l.logRecords) - 1; i >= 0; i-- {
		r := l.logRecords[i]
		wid, ok := workOf[str(r["item"])]
		if str(r["result"]) == "applied" && ok && len(perWork[wid]) < workRecentLimit {
			perWork[wid] = append(perWork[wid], l.changeOf(r))
		}
	}
	worksAny := make([]any, len(worksOut))
	for i, info := range worksOut {
		rec := perWork[str(info["id"])]
		if rec == nil {
			rec = []any{}
		}
		info["recent"] = rec
		worksAny[i] = info
	}

	asks := l.buildAsks(today, itemsOut)
	open := 0
	for _, i := range itemsOut {
		if str(i["status"]) != "done" {
			open++
		}
	}
	thresholds := obj{}
	for k, v := range sig {
		thresholds[k] = v
	}
	thresholds["confirm_stale_days"] = cfg["confirm_stale_days"]
	signalRules := obj{}
	for _, k := range []string{"due_risk_days", "due_within_days", "waiting_over_days", "work_stale_days"} {
		signalRules[k] = sig[k]
	}
	signalRules["later_days"] = defaultLaterDays
	if v, ok := sig["later_days"]; ok {
		signalRules["later_days"] = v
	}
	reminderLists := []any{}
	for _, n := range asList(cfg["reminder_lists"]) {
		if isStr(n) {
			reminderLists = append(reminderLists, n)
		}
	}
	return obj{
		"schema": StateSchema, "generated_at": nowISO, "today": today.Format("2006-01-02"), "page_url": DefaultPageURL,
		"thresholds": thresholds, "signal_rules": signalRules,
		"labels":         obj{"status": stringMapAny(statusLabels), "wait": stringMapAny(waitLabels), "source": stringMapAny(sourceLabels)},
		"reminder_lists": reminderLists,
		"works":          worksAny, "items": toObjAny(itemsOut), "asks": asks,
		"recent_changes": recent,
		"calendar_write": obj{"enabled": false, "calendar": "Pin", "only_human_due": true},
		"done_daily":     l.doneDaily(today),
		"counts":         obj{"items": len(itemsOut), "open": open, "asks": len(asks), "proposals": len(l.proposals)},
	}
}

func stringMapAny(m map[string]string) obj {
	out := obj{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
