package pin

import (
	"crypto/sha1" //nolint:gosec // 요청 id를 만드는 비암호 해시다.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TimeFormat은 요청의 at 에 쓰는 시각 형식이다. 시간대는 항상 오프셋으로 쓴다(Z 를 쓰지 않는다).
// 시스템 파이썬 3.9 는 Z 로 끝나는 시각을 읽지 못해, 같은 원장을 upstream 스크립트로 읽을 때 걸린다.
const TimeFormat = "2006-01-02T15:04:05-07:00"

// ---------------------------------------------------------------------------
// 쓰기 잠금
// ---------------------------------------------------------------------------

const (
	lockName      = ".lock"
	lockStale     = 30 * time.Second
	lockWait      = 5 * time.Second
	lockPollEvery = 20 * time.Millisecond
)

// withLock은 `pin/.lock`을 배타 생성으로 잡고 fn 을 돌린다. 읽기, 검사, 저장이 모두 그 안에서
// 끝난다. 쓰는 프로세스가 둘이다(serve 의 화면 쓰기, CLI). 오래된 잠금은 시각으로 가려 치운다.
func withLock(wiki string, fn func() error) error {
	dir := filepath.Join(wiki, "pin")
	path := filepath.Join(dir, lockName)
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d %s\n", os.Getpid(), time.Now().Format(time.RFC3339))
			f.Close()
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("원장 쓰기 잠금을 만들지 못했습니다: %w", err)
		}
		if st, serr := os.Stat(path); serr == nil && time.Since(st.ModTime()) > lockStale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("다른 프로세스가 원장을 쓰고 있어 %d초 안에 잠금을 얻지 못했습니다. 잠시 뒤 다시 시도해 주세요", int(lockWait.Seconds()))
		}
		time.Sleep(lockPollEvery)
	}
	defer os.Remove(path)
	return fn()
}

// ---------------------------------------------------------------------------
// 요청 묶음 적용
// ---------------------------------------------------------------------------

// Summary는 요청 묶음을 적용한 결과다.
type Summary struct {
	Applied  int      `json:"applied"`
	Noop     int      `json:"noop"`
	Rejected int      `json:"rejected"`
	Skipped  int      `json:"skipped"`
	Archived []string `json:"archived"`
	Results  []Result `json:"results"`
	Written  []string `json:"written"`
}

func requestSortKey(raw any) string {
	ro, ok := raw.(map[string]any)
	if !ok {
		return "2\x00\x00"
	}
	at := ro["received_at"]
	if !truthy(at) {
		at = ro["at"]
	}
	return "1\x00" + str(at) + "\x00" + str(ro["id"])
}

func (l *Ledger) archiveOld(today time.Time) []string {
	days, _ := toInt(l.config["archive_after_days"])
	var moved []string
	for _, id := range sortedKeys(l.items) {
		it := l.items[id]
		if str(it["status"]) != "done" || int(today.Sub(parseDate(str(it["closed_at"]))).Hours()/24) <= days {
			continue
		}
		blocked := false
		for _, c := range l.items {
			if str(c["parent"]) == id && str(c["status"]) != "done" {
				blocked = true
			}
		}
		if blocked {
			continue
		}
		l.archived[id] = it
		delete(l.items, id)
		l.deletedItems[id] = true
		delete(l.dirtyItems, id)
		l.dirtyArchive[id] = true
		moved = append(moved, id)
	}
	return moved
}

// run은 잠금 안에서 원장을 읽고 요청을 적용해 저장한다. save 가 거짓이면 아무것도 쓰지 않는다.
// 앞 단계(commit)가 있으면 요청이 전부 통과한 뒤 저장 직전에 부른다. 저장이 실패하면 undo 를 부른다.
func (l *Ledger) run(raws []any, now time.Time) (*Summary, error) {
	if errs := checkLedger(l); len(errs) > 0 {
		if len(errs) > 10 {
			errs = errs[:10]
		}
		return nil, fmt.Errorf("적용 전 원장이 이미 검사에 실패합니다(적용하지 않음):\n  %s", strings.Join(errs, "\n  "))
	}
	sort.SliceStable(raws, func(i, j int) bool { return requestSortKey(raws[i]) < requestSortKey(raws[j]) })
	nowISO := now.Format(TimeFormat)
	s := &Summary{Archived: []string{}, Results: []Result{}, Written: []string{}}
	for _, raw := range raws {
		rec := l.applyOne(raw, nowISO)
		res := toResult(rec)
		s.Results = append(s.Results, res)
		switch {
		case res.Result == "applied" && !res.Noop:
			s.Applied++
		case res.Result == "applied":
			s.Noop++
		case res.Result == "rejected":
			s.Rejected++
		default:
			s.Skipped++
		}
	}
	s.Archived = append(s.Archived, l.archiveOld(now)...)
	if errs := checkLedger(l); len(errs) > 0 {
		if len(errs) > 10 {
			errs = errs[:10]
		}
		return nil, fmt.Errorf("저장 전 원장 검사 실패(저장하지 않음):\n  %s", strings.Join(errs, "\n  "))
	}
	return s, nil
}

// Apply는 요청 묶음을 원장에 적용한다. 요청마다 순서대로 적용하고 하나가 거절돼도 나머지는 계속한다.
// 거절은 log/ 에 남는다. via 가 없는 요청은 command 로 본다. id 와 at 이 없으면 채운다.
func Apply(wiki string, reqs []any, now time.Time) (*Summary, error) {
	raws := make([]any, len(reqs))
	for i, r := range reqs {
		raws[i] = fillCommandRequest(r, now, i)
	}
	var out *Summary
	err := withLock(wiki, func() error {
		led, err := load(wiki)
		if err != nil {
			return err
		}
		s, err := led.run(raws, now)
		if err != nil {
			return err
		}
		if s.Written, err = led.save(); err != nil {
			return err
		}
		out = s
		return nil
	})
	return out, err
}

func canonicalHash(v any, n int) string {
	sum := sha1.Sum([]byte(dump(v, false))) //nolint:gosec
	return hex.EncodeToString(sum[:])[:n]
}

// fillCommandRequest는 `pin apply`로 들어온 요청에 빠진 칸을 채운다.
func fillCommandRequest(raw any, now time.Time, n int) any {
	ro, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	out := cloneObj(ro)
	if !truthy(out["via"]) {
		out["via"] = "command"
	}
	if !truthy(out["at"]) {
		out["at"] = now.Format(TimeFormat)
	}
	if !truthy(out["id"]) {
		out["id"] = fmt.Sprintf("cmd-%03d-%s", n, canonicalHash(obj{"day": now.Format("2006-01-02"), "req": ro}, 12))
	}
	return out
}

// BadRequest는 화면이 보낸 요청이 형식부터 틀렸다는 에러다. 로그에 남기지 않고 400 으로 답한다.
type BadRequest struct{ Msg string }

func (e *BadRequest) Error() string { return e.Msg }

// ApplyWeb은 화면이 보낸 변경 요청 하나를 받는 즉시 게이트에 돌린다. body 는 화면이 보낸 JSON 이다.
// id 와 at 은 여기서 채우고 via 는 web 으로 고정한다. 거절된 요청도 log/ 에 남아 화면이 상태에서 읽는다.
func ApplyWeb(wiki string, body []byte, now time.Time) (id, at string, err error) {
	v, err := decodeJSON(body)
	if err != nil {
		return "", "", &BadRequest{fmt.Sprintf("요청이 JSON이 아닙니다: %v", err)}
	}
	ro, ok := normalizeNumbers(v).(map[string]any)
	if !ok {
		return "", "", &BadRequest{"요청이 객체가 아닙니다"}
	}
	for _, k := range []string{"id", "at", "via", "received_at"} {
		if _, has := ro[k]; has {
			return "", "", &BadRequest{fmt.Sprintf("요청에 %s를 실을 수 없습니다", k)}
		}
	}
	if op, isS := ro["op"].(string); !isS || !opsByVia["web"][op] {
		return "", "", &BadRequest{fmt.Sprintf("화면이 낼 수 없는 연산입니다: %s", str(ro["op"]))}
	}
	id = fmt.Sprintf("%d-%s", now.UnixMilli(), canonicalHash(ro, 8))
	at = now.Format(TimeFormat)
	ro["id"], ro["at"], ro["via"] = id, at, "web"
	err = withLock(wiki, func() error {
		led, lerr := load(wiki)
		if lerr != nil {
			return lerr
		}
		if _, rerr := led.run([]any{ro}, now); rerr != nil {
			return rerr
		}
		_, serr := led.save()
		return serr
	})
	return id, at, err
}

// ---------------------------------------------------------------------------
// 상태
// ---------------------------------------------------------------------------

// State는 원장에서 화면이 읽는 상태를 만든다. 상태 파일을 두지 않고 요청마다 원장을 읽어 만든다.
// CLI 가 쓴 것이 다음 새로 고침에 보인다.
func State(wiki string, now time.Time) ([]byte, error) {
	var out []byte
	err := withLock(wiki, func() error {
		led, err := load(wiki)
		if err != nil {
			return err
		}
		if errs := checkLedger(led); len(errs) > 0 {
			return fmt.Errorf("원장이 검사에 실패합니다: %s", strings.Join(errs[:min(3, len(errs))], "; "))
		}
		today := parseDate(now.Format("2006-01-02"))
		out = []byte(dump(led.buildState(today, now.Format(TimeFormat)), true))
		return nil
	})
	return out, err
}

// ---------------------------------------------------------------------------
// 선언
// ---------------------------------------------------------------------------

// Declaration은 `--pin` 선언이다. 판단했고 할 일이 없으면 None 이다.
type Declaration struct {
	None     bool
	Requests []obj
}

var declaredOps = []string{"add", "update_ai", "set_owner", "set_work", "set_due", "add_relation", "propose"}

// ParseDeclaration은 `--pin` 인자를 읽는다. `none` 이거나 JSON 선언(`{"requests":[...]}`)의 글이다.
// 선언은 변경 요청에서 명령이 아는 것을 뺀 모양이다. id, at, via, 근거의 path, stage, date 는 쓰지 않는다.
func ParseDeclaration(text string) (*Declaration, error) {
	text = strings.TrimSpace(text)
	if text == "none" {
		return &Declaration{None: true}, nil
	}
	v, err := decodeJSON([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("선언이 JSON이 아닙니다: %w", err)
	}
	top := asObj(normalizeNumbers(v))
	if top == nil || !keysSubset(top, []string{"requests"}) {
		return nil, errors.New(`선언은 {"requests":[{"op":..., "args":{...}}]} 모양이어야 합니다`)
	}
	list, ok := top["requests"].([]any)
	if !ok || len(list) == 0 {
		return nil, errors.New(`선언의 requests가 비었습니다. 할 일이 없으면 --pin none을 쓰세요`)
	}
	d := &Declaration{}
	for i, e := range list {
		ro := asObj(e)
		if ro == nil {
			return nil, fmt.Errorf("선언의 %d번째 요청이 객체가 아닙니다", i+1)
		}
		for k := range ro {
			if k == "id" || k == "at" || k == "via" || k == "received_at" || k == "client" {
				return nil, fmt.Errorf("선언의 %d번째 요청에 %s를 쓸 수 없습니다. 명령이 채웁니다", i+1, k)
			}
		}
		op := str(ro["op"])
		if !contains(declaredOps, op) {
			return nil, fmt.Errorf("선언의 %d번째 요청 op %q는 받지 않습니다(받는 것: %s)", i+1, op, strings.Join(declaredOps, ", "))
		}
		d.Requests = append(d.Requests, ro)
	}
	return d, nil
}

// Doc은 지금 쓰는 문서다. 선언의 근거는 이 문서 하나다.
type Doc struct {
	Path  string // 위키 루트 상대 슬래시 경로
	Stage string // inbox, sources
	Date  string // 근거의 날짜. 문서의 created
	Text  string // 쓸 파일 전문. 아직 디스크에 없으므로 인용문 대조가 이것을 본다
}

// Move는 문서가 옮겨진 것이다. 이 문서를 근거로 둔 항목의 경로와 단계를 따라 고친다.
type Move struct{ From, To string }

// Affected는 옮겨진 문서를 근거로 둔 열린 일이다. 명령이 출력에 알린다.
type Affected struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// Txn은 문서 쓰기와 원장 쓰기를 한 덩어리로 묶는 요청이다. 선언의 요청이 하나라도 거절되면 아무것도 쓰지 않는다.
type Txn struct {
	Wiki   string
	Now    time.Time
	Doc    Doc          // 선언이 있을 때만 쓴다
	Decl   *Declaration // nil 이면 선언 없음(이동만 따라간다)
	Moves  []Move       // 선언보다 먼저 적용한다
	DryRun bool         // 참이면 게이트만 돌리고 아무것도 쓰지 않는다
	// Commit은 게이트가 전부 통과한 뒤 문서를 쓴다. Rollback은 원장 저장이 실패했을 때 방금 쓴 것을 지운다.
	Commit   func() error
	Rollback func()
}

// Report는 Txn 의 결과다.
type Report struct {
	Results   []Result   `json:"results"`
	Items     []string   `json:"items,omitempty"`     // 선언으로 새로 생긴 항목 번호
	Proposals []string   `json:"proposals,omitempty"` // 선언으로 생긴 제안 번호
	Affected  []Affected `json:"affected,omitempty"`  // 옮겨진 문서를 근거로 둔 열린 일
	None      bool       `json:"none,omitempty"`
	DryRun    bool       `json:"dryRun,omitempty"`
}

// RejectedError는 선언의 요청이 게이트에서 거절됐다는 에러다. 문서도 원장도 쓰지 않았다.
type RejectedError struct{ Rejected []Result }

func (e *RejectedError) Error() string {
	var b strings.Builder
	b.WriteString("선언이 원장 검사에서 거절되어 문서도 원장도 쓰지 않았습니다.")
	for _, r := range e.Rejected {
		fmt.Fprintf(&b, "\n  %s: %s", r.Code, r.Message)
	}
	return b.String()
}

// fillDeclared는 선언의 요청에 명령이 아는 것(id, at, via, 근거의 경로와 단계와 날짜)을 채운다.
// 에이전트가 근거에 path 를 적어 오면 거절한다. 근거는 지금 쓰는 그 문서뿐이다.
func fillDeclared(d *Declaration, doc Doc, now time.Time) ([]any, error) {
	var out []any
	for i, req := range d.Requests {
		r := cloneObj(req)
		args := asObj(r["args"])
		if args == nil {
			args = obj{}
			r["args"] = args
		}
		for _, key := range []string{"evidence", "mentions"} {
			list, ok := args[key].([]any)
			if !ok {
				continue
			}
			for _, e := range list {
				eo := asObj(e)
				if eo == nil {
					return nil, fmt.Errorf("선언의 %d번째 요청 %s에 객체가 아닌 값이 있습니다", i+1, key)
				}
				for _, bad := range []string{"path", "stage", "date"} {
					if _, has := eo[bad]; has {
						return nil, fmt.Errorf("선언의 %d번째 요청 %s에 %s를 쓸 수 없습니다. 근거는 지금 쓰는 문서이고 명령이 채웁니다", i+1, key, bad)
					}
				}
				eo["path"], eo["date"] = doc.Path, doc.Date
				if key == "evidence" {
					eo["stage"] = doc.Stage
					if _, has := eo["section"]; !has {
						eo["section"] = ""
					}
				}
			}
		}
		if ctx := asObj(args["context"]); ctx != nil {
			if _, has := ctx["written_at"]; !has {
				ctx["written_at"] = doc.Date
			}
		}
		r["via"] = "extract"
		r["at"] = now.Format(TimeFormat)
		// 요청 순서가 id 로 뒤섞이지 않게 번호를 앞에 둔다. 같은 문서의 같은 선언은 같은 id 라 두 번 들어가지 않는다.
		r["id"] = fmt.Sprintf("ext-%03d-%s", i, canonicalHash(obj{"path": doc.Path, "req": req}, 12))
		out = append(out, r)
	}
	return out, nil
}

func moveRequests(moves []Move, now time.Time) []any {
	var out []any
	for _, m := range moves {
		out = append(out, obj{
			"id": "move-" + canonicalHash(obj{"from": m.From, "to": m.To}, 12), "at": now.Format(TimeFormat), "via": "extract",
			"op": "move_path", "args": obj{"from": m.From, "to": m.To},
		})
	}
	return out
}

func (l *Ledger) referencedPaths() map[string]bool {
	out := map[string]bool{}
	for _, bucket := range []map[string]obj{l.items, l.archived} {
		for _, it := range bucket {
			for _, e := range asList(it["evidence"]) {
				out[str(asObj(e)["path"])] = true
			}
			for _, m := range asList(it["mentions"]) {
				out[str(asObj(m)["path"])] = true
			}
		}
	}
	return out
}

// affectedBy는 옛 경로를 근거나 언급으로 둔 열린 일을 모은다.
func (l *Ledger) affectedBy(paths []string) []Affected {
	var out []Affected
	for _, id := range sortedKeys(l.items) {
		it := l.items[id]
		if str(it["status"]) == "done" {
			continue
		}
		hit := false
		for _, e := range asList(it["evidence"]) {
			if contains(paths, str(asObj(e)["path"])) {
				hit = true
			}
		}
		for _, m := range asList(it["mentions"]) {
			if contains(paths, str(asObj(m)["path"])) {
				hit = true
			}
		}
		if hit {
			out = append(out, Affected{ID: id, Title: str(it["title"]), Status: str(it["status"])})
		}
	}
	return out
}

// Run은 Txn 을 한 덩어리로 돌린다. 순서는 잠금, 읽기, 이동 따라가기, 선언 적용, 문서 쓰기, 원장 저장이다.
// 선언의 요청이 하나라도 거절되면 *RejectedError 를 내고 아무것도 쓰지 않는다. 원장 저장이 실패하면
// 방금 쓴 문서를 지운다.
func Run(tx Txn) (*Report, error) {
	var rep *Report
	err := withLock(tx.Wiki, func() error {
		led, err := load(tx.Wiki)
		if err != nil {
			return err
		}
		// 근거나 언급으로 쓰이지 않는 경로의 이동은 요청으로 만들지 않는다. 로그에 변화 없음 줄이 쌓이지 않게 한다.
		var moves []Move
		var oldPaths []string
		for _, m := range tx.Moves {
			oldPaths = append(oldPaths, m.From)
		}
		referenced := led.referencedPaths()
		for _, m := range tx.Moves {
			if referenced[m.From] {
				moves = append(moves, m)
				if !led.pathExists(m.To) {
					led.docs[m.To] = "" // 아직 쓰지 않은 도착 문서. 실재 검사만 통과시킨다
				}
			}
		}
		rep = &Report{Affected: led.affectedBy(oldPaths), DryRun: tx.DryRun}
		if tx.Decl != nil && tx.Decl.None {
			rep.None = true
		}
		var raws []any
		if tx.Decl != nil && !tx.Decl.None {
			led.docs[tx.Doc.Path] = tx.Doc.Text
			declared, err := fillDeclared(tx.Decl, tx.Doc, tx.Now)
			if err != nil {
				return err
			}
			raws = append(moveRequests(moves, tx.Now), declared...)
		} else {
			raws = moveRequests(moves, tx.Now)
		}
		// 이동은 선언보다 먼저 적용해야 한다. 정렬 키가 같은 at 이라 id 로 갈리므로 따로 두 번 돌린다.
		moveCount := len(moves)
		sum, err := led.run(raws[:moveCount], tx.Now)
		if err != nil {
			return err
		}
		var declSum *Summary
		if len(raws) > moveCount {
			if declSum, err = led.run(raws[moveCount:], tx.Now); err != nil {
				return err
			}
		}
		all := sum.Results
		if declSum != nil {
			all = append(append([]Result{}, all...), declSum.Results...)
		}
		rep.Results = all
		if declSum != nil {
			var rejected []Result
			for _, r := range declSum.Results {
				if r.Result == "rejected" {
					rejected = append(rejected, r)
				}
			}
			if len(rejected) > 0 {
				return &RejectedError{Rejected: rejected}
			}
		}
		for _, r := range sum.Results {
			if r.Result == "rejected" {
				return &RejectedError{Rejected: []Result{r}}
			}
		}
		newItems, newProps := newlyCreated(led, declSum)
		rep.Items, rep.Proposals = newItems, newProps
		if tx.Decl != nil && tx.Decl.None {
			led.addLog(obj{
				"id": "none-" + canonicalHash(obj{"path": tx.Doc.Path, "at": tx.Now.Format(TimeFormat)}, 12), "applied_at": tx.Now.Format(TimeFormat),
				"at": tx.Now.Format(TimeFormat), "via": "extract", "op": "none", "item": nil, "result": "declared_none",
				"code": "none", "message": "판단했고 할 일이 없습니다", "path": tx.Doc.Path,
			})
		}
		if tx.DryRun {
			return nil
		}
		if tx.Commit != nil {
			if err := tx.Commit(); err != nil {
				return err
			}
		}
		if _, err := led.save(); err != nil {
			if tx.Rollback != nil {
				tx.Rollback()
			}
			return fmt.Errorf("원장을 쓰지 못해 방금 쓴 문서를 지웠습니다: %w", err)
		}
		return nil
	})
	if err != nil {
		return rep, err
	}
	return rep, nil
}

// newlyCreated는 이번 선언으로 생긴 항목과 제안 번호를 로그에서 가려낸다.
func newlyCreated(l *Ledger, s *Summary) (items, props []string) {
	if s == nil {
		return nil, nil
	}
	ids := map[string]bool{}
	for _, r := range s.Results {
		ids[r.ID] = true
	}
	for _, rec := range l.logRecords {
		if !ids[str(rec["id"])] || str(rec["result"]) != "applied" || rec["noop"] == true {
			continue
		}
		msg := str(rec["message"])
		switch {
		case strings.HasPrefix(msg, "새 항목 "):
			items = append(items, str(rec["item"]))
		case strings.HasPrefix(msg, "제안 "), strings.HasPrefix(msg, "관계 제안 "), strings.HasPrefix(msg, "프로젝트 후보 제안 "):
			if f := strings.Fields(msg); len(f) > 0 {
				props = append(props, f[len(f)-1])
			}
		}
	}
	for _, p := range l.proposals {
		if str(p["kind"]) == "new_item" && contains(items, str(p["item"])) {
			props = append(props, str(p["id"]))
		}
	}
	props = uniqueSorted(props)
	return
}

func uniqueSorted(in []string) []string {
	m := setOf(in...)
	if len(m) == 0 {
		return nil
	}
	return sortedKeys(m)
}

// Follow는 문서를 옮긴 명령(promote, mv, demote, archive)이 근거 경로를 따라가게 한다. 선언이 없고
// 이동만 있다. 판단이 없는 기계적 변경이라 rev 를 올리지 않는다.
func Follow(wiki string, moves []Move, now time.Time) ([]Affected, error) {
	rep, err := Run(Txn{Wiki: wiki, Now: now, Moves: moves})
	if rep == nil {
		return nil, err
	}
	return rep.Affected, err
}

// ---------------------------------------------------------------------------
// 목록(에이전트가 판단 전에 읽는 색인)
// ---------------------------------------------------------------------------

// ListItem은 열린 일 한 줄이다.
type ListItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Work          string   `json:"work"`
	Status        string   `json:"status"`
	Owner         string   `json:"owner"`
	Locked        []string `json:"locked"`
	Proposed      bool     `json:"proposed"`
	EvidencePaths []string `json:"evidence_paths"`
	RecentNotes   []obj    `json:"recent_notes,omitempty"`
}

// ListWork는 프로젝트 한 줄이다.
type ListWork struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc,omitempty"`
}

// Listing은 `pin list` 의 결과다.
type Listing struct {
	Items    []ListItem `json:"items"`
	Works    []ListWork `json:"works"`
	Rejected []string   `json:"rejected"`
	// StaleEvidence는 인용문이 문서에서 사라진 근거의 수다. 승급 준비로 본문이 바뀐 문서가 여기에 든다.
	StaleEvidence int `json:"stale_evidence"`
}

// List는 열린 일, 프로젝트, 사람이 버린 것을 낸다. 에이전트가 문서에서 할 일을 찾을 때 이것과 견준다.
func List(wiki string) (*Listing, error) {
	var out *Listing
	err := withLock(wiki, func() error {
		led, err := load(wiki)
		if err != nil {
			return err
		}
		out = led.listing()
		return nil
	})
	return out, err
}

func (l *Ledger) listing() *Listing {
	proposed := map[string]bool{}
	for _, p := range l.proposals {
		if str(p["kind"]) == "new_item" && p["item"] != nil {
			proposed[str(p["item"])] = true
		}
	}
	out := &Listing{Items: []ListItem{}, Works: []ListWork{}, Rejected: []string{}}
	for _, id := range sortedKeys(l.items) {
		it := l.items[id]
		if str(it["status"]) == "done" {
			continue
		}
		paths := map[string]bool{}
		for _, e := range asList(it["evidence"]) {
			eo := asObj(e)
			paths[str(eo["path"])] = true
			if text, ok := l.docText(str(eo["path"])); ok && !strings.Contains(normalizeSpace(text), normalizeSpace(str(eo["quote"]))) {
				out.StaleEvidence++
			}
		}
		owner := asObj(it["owner"])
		basis := "추정"
		if str(owner["basis"]) == "stated" {
			basis = "적힘"
		}
		row := ListItem{
			ID: id, Title: str(it["title"]), Work: str(it["work"]), Status: str(it["status"]),
			Owner: str(owner["who"]) + "(" + basis + ")", Locked: strList(it["locked"]), Proposed: proposed[id],
			EvidencePaths: sortedKeys(paths),
		}
		if row.Locked == nil {
			row.Locked = []string{}
		}
		var notes []obj
		for _, m := range asList(it["mentions"]) {
			if mo := asObj(m); mentionKind(str(mo["path"])) == "note" {
				notes = append(notes, obj{"date": mo["date"], "text": mo["quote"]})
			}
		}
		if len(notes) > 3 {
			notes = notes[len(notes)-3:]
		}
		row.RecentNotes = notes
		out.Items = append(out.Items, row)
	}
	for _, w := range asList(l.config["works"]) {
		wo := asObj(w)
		out.Works = append(out.Works, ListWork{ID: str(wo["id"]), Name: str(wo["name"]), Desc: str(wo["desc"])})
	}
	var rows []obj
	for _, r := range l.rejections {
		switch str(r["kind"]) {
		case "item", "relation", "status", "parent", "work":
			rows = append(rows, r)
		}
	}
	if len(rows) > 30 {
		rows = rows[len(rows)-30:]
	}
	for _, r := range rows {
		switch str(r["kind"]) {
		case "item":
			out.Rejected = append(out.Rejected, "항목 아님: "+str(r["title"]))
		case "work":
			out.Rejected = append(out.Rejected, "프로젝트 후보 아님: "+str(r["name"]))
		case "relation":
			out.Rejected = append(out.Rejected, "관계 아님: "+str(r["frm"])+" "+str(r["rel"])+" "+str(r["to"]))
		default:
			out.Rejected = append(out.Rejected, str(r["kind"])+" 제안 아님: "+str(r["item"])+" "+str(r["value"]))
		}
	}
	return out
}

// ReadRequests는 `pin apply --requests` 파일을 읽는다. {"requests":[...]}, 목록, 객체 하나를 받는다.
func ReadRequests(data []byte) ([]any, error) {
	v, err := decodeJSON(data)
	if err != nil {
		return nil, fmt.Errorf("요청 파일이 JSON이 아닙니다: %w", err)
	}
	v = normalizeNumbers(v)
	if o, ok := v.(map[string]any); ok {
		if reqs, has := o["requests"]; has {
			l, isList := reqs.([]any)
			if !isList {
				return nil, errors.New("requests가 목록이 아닙니다")
			}
			return l, nil
		}
		return []any{o}, nil
	}
	if l, ok := v.([]any); ok {
		return l, nil
	}
	return nil, errors.New("요청 파일은 객체, 목록, 또는 {\"requests\":[...]} 여야 합니다")
}

// JSONText는 값을 한 줄 JSON 글로 쓴다. 출력과 시험이 쓴다.
func JSONText(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
