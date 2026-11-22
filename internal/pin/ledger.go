package pin

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Reject는 변경 요청을 적용하지 않는 이유다. Code는 기계용이고 Message는
// 화면에 그대로 보인다. 코드 이름은 upstream 의 거절 코드와 같다.
type Reject struct {
	Code    string
	Message string
}

func (r *Reject) Error() string { return r.Message }

func reject(code, format string, args ...any) *Reject {
	return &Reject{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Ledger는 `pin/` 폴더 전체를 메모리에 올린 것이다. 저장은 save 한 곳에서만 한다.
type Ledger struct {
	Dir  string // pin/ 의 경로
	Wiki string // 위키 루트
	// docs는 아직 디스크에 없는 문서의 본문이다. 문서를 쓰기 전에 선언을 미리
	// 돌릴 때 인용문 대조가 이 본문을 본다.
	docs map[string]string

	config    obj
	items     map[string]obj
	archived  map[string]obj
	relations []obj
	propSeq   int
	proposals []obj

	rejections    []obj
	newRejections []obj
	logRecords    []obj
	newLog        []obj
	logIDs        map[string]bool

	configDirty  bool
	dirtyItems   map[string]bool
	deletedItems map[string]bool
	dirtyArchive map[string]bool

	actingVia    string
	actingClient any
	detail       obj
}

func newLedger(dir, wiki string) *Ledger {
	return &Ledger{
		Dir: dir, Wiki: wiki, docs: map[string]string{},
		items: map[string]obj{}, archived: map[string]obj{}, logIDs: map[string]bool{},
		dirtyItems: map[string]bool{}, deletedItems: map[string]bool{}, dirtyArchive: map[string]bool{},
		actingVia: "command",
	}
}

// ConfigPath는 위키에서 원장의 설정 파일 경로다. 이 파일이 있어야 Pin이 켜진 위키다.
func ConfigPath(wiki string) string { return filepath.Join(wiki, "pin", "config.json") }

// Enabled는 위키가 Pin을 켰는지 본다. 켜지 않은 위키에서 세 명령은 1.0과 같게 돈다.
func Enabled(wiki string) bool {
	st, err := os.Stat(ConfigPath(wiki))
	return err == nil && !st.IsDir()
}

func readJSONFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v, err := decodeJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return normalizeNumbers(v), nil
}

// load는 원장을 읽는다. config.json이 없으면 init을 안내하는 에러다.
func load(wiki string) (*Ledger, error) {
	dir := filepath.Join(wiki, "pin")
	led := newLedger(dir, wiki)
	cfg, err := readJSONFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s가 없습니다. `engram pin init`으로 먼저 켜 주세요", filepath.Join(dir, "config.json"))
	}
	if err != nil {
		return nil, err
	}
	if led.config = asObj(cfg); led.config == nil {
		return nil, fmt.Errorf("%s가 객체가 아닙니다", filepath.Join(dir, "config.json"))
	}
	itemFiles, _ := filepath.Glob(filepath.Join(dir, "items", "P*.json"))
	sort.Strings(itemFiles)
	for _, p := range itemFiles {
		v, err := readJSONFile(p)
		if err != nil {
			return nil, err
		}
		led.items[strings.TrimSuffix(filepath.Base(p), ".json")] = asObj(v)
	}
	archFiles, _ := filepath.Glob(filepath.Join(dir, "archive", "*", "P*.json"))
	sort.Strings(archFiles)
	for _, p := range archFiles {
		v, err := readJSONFile(p)
		if err != nil {
			return nil, err
		}
		led.archived[strings.TrimSuffix(filepath.Base(p), ".json")] = asObj(v)
	}
	led.relations = nil
	if v, err := readJSONFile(filepath.Join(dir, "relations.json")); err == nil {
		for _, r := range asList(v) {
			led.relations = append(led.relations, asObj(r))
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if v, err := readJSONFile(filepath.Join(dir, "proposals.json")); err == nil {
		po := asObj(v)
		led.propSeq, _ = toInt(po["seq"])
		for _, p := range asList(po["proposals"]) {
			led.proposals = append(led.proposals, asObj(p))
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if data, err := os.ReadFile(filepath.Join(dir, "rejections.jsonl")); err == nil {
		recs, err := parseJSONL(data)
		if err != nil {
			return nil, err
		}
		led.rejections = recs
	}
	logFiles, _ := filepath.Glob(filepath.Join(dir, "log", "*.jsonl"))
	sort.Strings(logFiles)
	for _, p := range logFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		recs, err := parseJSONL(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		for _, r := range recs {
			led.indexLog(r, false)
		}
	}
	return led, nil
}

func parseJSONL(data []byte) ([]obj, error) {
	var out []obj
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		v, err := decodeJSON([]byte(line))
		if err != nil {
			return nil, err
		}
		out = append(out, asObj(normalizeNumbers(v)))
	}
	return out, nil
}

func (l *Ledger) indexLog(rec obj, isNew bool) {
	l.logRecords = append(l.logRecords, rec)
	l.logIDs[str(rec["id"])] = true
	if isNew {
		l.newLog = append(l.newLog, rec)
	}
}

func (l *Ledger) addLog(rec obj) { l.indexLog(rec, true) }

func (l *Ledger) allItemIDs() map[string]bool {
	out := map[string]bool{}
	for id := range l.items {
		out[id] = true
	}
	for id := range l.archived {
		out[id] = true
	}
	return out
}

func itemNum(id string) int {
	n := 0
	for _, c := range id[1:] {
		n = n*10 + int(c-'0')
	}
	return n
}

func (l *Ledger) nextItemID() string {
	max := 0
	for id := range l.allItemIDs() {
		if itemIDRe.MatchString(id) && itemNum(id) > max {
			max = itemNum(id)
		}
	}
	for _, r := range l.logRecords {
		if id := str(r["item"]); itemIDRe.MatchString(id) && itemNum(id) > max {
			max = itemNum(id)
		}
	}
	return fmt.Sprintf("P%04d", max+1)
}

func (l *Ledger) nextProposalID() string {
	l.propSeq++
	return fmt.Sprintf("Q%04d", l.propSeq)
}

func (l *Ledger) findProposal(id string) obj {
	for _, p := range l.proposals {
		if str(p["id"]) == id {
			return p
		}
	}
	return nil
}

func (l *Ledger) removeProposal(p obj) {
	for i, e := range l.proposals {
		if str(e["id"]) == str(p["id"]) {
			l.proposals = append(l.proposals[:i:i], l.proposals[i+1:]...)
			return
		}
	}
}

func (l *Ledger) workIDs() map[string]bool {
	out := map[string]bool{Unsorted: true}
	for _, w := range asList(l.config["works"]) {
		out[str(asObj(w)["id"])] = true
	}
	return out
}

func (l *Ledger) mark(id string) {
	l.dirtyItems[id] = true
	delete(l.deletedItems, id)
}

// snapshot은 요청 하나를 되돌리기 위한 복사본이다.
type snapshot struct {
	items, archived                     map[string]obj
	relations, proposals, newRejections []obj
	propSeq                             int
	dirtyItems, deletedItems            map[string]bool
	dirtyArchive                        map[string]bool
	config                              obj
	configDirty                         bool
}

func cloneObjMap(m map[string]obj) map[string]obj {
	out := make(map[string]obj, len(m))
	for k, v := range m {
		out[k] = cloneObj(v)
	}
	return out
}

func cloneObjs(l []obj) []obj {
	out := make([]obj, len(l))
	for i, v := range l {
		out[i] = cloneObj(v)
	}
	return out
}

func cloneSet(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (l *Ledger) snapshot() snapshot {
	return snapshot{
		items: cloneObjMap(l.items), archived: cloneObjMap(l.archived),
		relations: cloneObjs(l.relations), proposals: cloneObjs(l.proposals), newRejections: cloneObjs(l.newRejections),
		propSeq:    l.propSeq,
		dirtyItems: cloneSet(l.dirtyItems), deletedItems: cloneSet(l.deletedItems), dirtyArchive: cloneSet(l.dirtyArchive),
		config: cloneObj(l.config), configDirty: l.configDirty,
	}
}

func (l *Ledger) restore(s snapshot) {
	l.items, l.archived = s.items, s.archived
	l.relations, l.proposals, l.newRejections = s.relations, s.proposals, s.newRejections
	l.propSeq = s.propSeq
	l.dirtyItems, l.deletedItems, l.dirtyArchive = s.dirtyItems, s.deletedItems, s.dirtyArchive
	l.config, l.configDirty = s.config, s.configDirty
}

// writeIfChanged는 내용이 다를 때만 쓴다. 임시 파일에 쓴 뒤 바꿔 끼워 반쪽짜리
// 파일이 남지 않게 한다.
func writeIfChanged(path, text string) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && string(old) == text {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, nil
}

func appendLines(path string, recs []obj) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, r := range recs {
		if _, err := f.WriteString(dump(r, false) + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func relationSortKey(r obj) string {
	return str(r["from"]) + "\x00" + str(r["to"]) + "\x00" + str(r["kind"])
}

func (l *Ledger) relationsAny() []any {
	rels := append([]obj(nil), l.relations...)
	sort.SliceStable(rels, func(i, j int) bool { return relationSortKey(rels[i]) < relationSortKey(rels[j]) })
	out := make([]any, len(rels))
	for i, r := range rels {
		out[i] = r
	}
	return out
}

func (l *Ledger) proposalsObj() obj {
	list := make([]any, len(l.proposals))
	for i, p := range l.proposals {
		list[i] = p
	}
	return obj{"proposals": list, "seq": l.propSeq}
}

// save는 바뀐 것만 디스크에 쓰고 쓴 파일 이름을 낸다. 순서와 파일 모양은 upstream 과 같다.
func (l *Ledger) save() ([]string, error) {
	var written []string
	put := func(path, text, name string) error {
		changed, err := writeIfChanged(path, text)
		if err != nil {
			return err
		}
		if changed {
			written = append(written, name)
		}
		return nil
	}
	for _, id := range sortedKeys(l.dirtyItems) {
		if it, ok := l.items[id]; ok {
			if err := put(filepath.Join(l.Dir, "items", id+".json"), dump(it, true), "items/"+id+".json"); err != nil {
				return written, err
			}
		}
	}
	for _, id := range sortedKeys(l.deletedItems) {
		p := filepath.Join(l.Dir, "items", id+".json")
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return written, err
			}
			written = append(written, "items/"+id+".json (삭제)")
		}
	}
	for _, id := range sortedKeys(l.dirtyArchive) {
		it := l.archived[id]
		year := str(it["closed_at"])[:4]
		if err := put(filepath.Join(l.Dir, "archive", year, id+".json"), dump(it, true), "archive/"+year+"/"+id+".json"); err != nil {
			return written, err
		}
	}
	if l.configDirty {
		if err := put(filepath.Join(l.Dir, "config.json"), dump(l.config, true), "config.json"); err != nil {
			return written, err
		}
		l.configDirty = false
	}
	if err := put(filepath.Join(l.Dir, "relations.json"), dump(l.relationsAny(), true), "relations.json"); err != nil {
		return written, err
	}
	if err := put(filepath.Join(l.Dir, "proposals.json"), dump(l.proposalsObj(), true), "proposals.json"); err != nil {
		return written, err
	}
	if len(l.newRejections) > 0 {
		if err := appendLines(filepath.Join(l.Dir, "rejections.jsonl"), l.newRejections); err != nil {
			return written, err
		}
		l.rejections = append(l.rejections, l.newRejections...)
		l.newRejections = nil
		written = append(written, "rejections.jsonl")
	}
	byMonth := map[string][]obj{}
	for _, r := range l.newLog {
		m := str(r["applied_at"])[:7]
		byMonth[m] = append(byMonth[m], r)
	}
	for _, m := range sortedKeys(byMonth) {
		if err := appendLines(filepath.Join(l.Dir, "log", m+".jsonl"), byMonth[m]); err != nil {
			return written, err
		}
		written = append(written, "log/"+m+".jsonl")
	}
	l.newLog = nil
	l.dirtyItems, l.deletedItems, l.dirtyArchive = map[string]bool{}, map[string]bool{}, map[string]bool{}
	return written, nil
}

// DefaultConfig는 `pin init`이 쓰는 설정이다. upstream 의 설정에서 사내 프로젝트와
// 미리알림, Hermes, 캘린더 칸을 뺐다. 프로젝트는 비워 두고 사람이 정한다.
func DefaultConfig() obj {
	return obj{
		"works":              []any{},
		"smart_lists":        obj{},
		"reminder_lists":     []any{},
		"signal":             obj{"due_within_days": 14, "due_risk_days": 3, "waiting_over_days": 14, "work_stale_days": 10},
		"confirm_stale_days": 10,
		"archive_after_days": 30,
	}
}

// Init은 `pin/` 뼈대를 만들고 새로 만든 파일 이름을 낸다. 이미 있는 파일은 건드리지 않는다.
func Init(wiki string) ([]string, error) {
	dir := filepath.Join(wiki, "pin")
	var created []string
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	for _, name := range emptyDirs {
		d := filepath.Join(dir, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return nil, err
		}
		keep := filepath.Join(d, ".gitkeep")
		other := false
		for _, e := range entries {
			if e.Name() != ".gitkeep" {
				other = true
			}
		}
		if _, err := os.Stat(keep); err != nil && !other {
			if err := os.WriteFile(keep, nil, 0o644); err != nil {
				return nil, err
			}
			created = append(created, name+"/.gitkeep")
		}
	}
	files := []struct{ name, text string }{
		{"config.json", dump(DefaultConfig(), true)},
		{"relations.json", "[]\n"},
		{"proposals.json", dump(obj{"proposals": []any{}, "seq": 0}, true)},
		{"rejections.jsonl", ""},
	}
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := os.WriteFile(p, []byte(f.text), 0o644); err != nil {
			return nil, err
		}
		created = append(created, f.name)
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// 날짜와 글 도우미
// ---------------------------------------------------------------------------

func isDate(v any) bool {
	s, ok := v.(string)
	if !ok || !dateRe.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

var spaceRe = regexp.MustCompile(`\s+`)

func normalizeSpace(s string) string { return strings.TrimSpace(spaceRe.ReplaceAllString(s, " ")) }

// dateOfAt은 변경 요청의 at(시간대가 붙은 ISO 시각)에서 날짜를 꺼낸다.
func dateOfAt(at any) (string, error) {
	s, ok := at.(string)
	if !ok {
		return "", errors.New("at이 문자열이 아닙니다")
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return "", fmt.Errorf("at 형식이 시간대가 붙은 ISO 시각이 아닙니다: %q", s)
	}
	return s[:10], nil
}

// shortTitle은 제목을 20자로 줄인다.
func shortTitle(title string) string {
	text := normalizeSpace(title)
	r := []rune(text)
	if len(r) <= 20 {
		return text
	}
	return strings.TrimRight(string(r[:20]), " \t\n") + "..."
}
