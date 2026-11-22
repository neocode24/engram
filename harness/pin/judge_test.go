package pin

// 판정 규칙을 가짜 upstream 으로 고정하는 시험이다(ADR 0107). 0104 때와 같은 방식이다.
// 임시 git 저장소에 가짜 upstream 을 만들고, 임시 디렉터리에 upstream-sync.py 와 견본을 복사한 가짜 engram 루트를 만든다.
// 실제 저장소의 harness/upstream.lock 과 private/deltas 는 건드리지 않는다. ENGRAM_UPSTREAM 이 필요 없으므로 CI 에서도 돈다.
// python3 나 git 이 없으면 건너뛴다.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fakeAgents = "# 가짜 upstream\n\n## meta 계약 변경 로그\n\n계약 파일(`pin-rules.md`)을 바꾸면 CHANGELOG 에 적는다.\n\n## 다른 절\n"

	fakeChangelog = "# 변경 로그\n\n## 2026-11-01\n\n### pin-rules\n- 첫 줄\n"

	fakePinlib = `"""시험용 가짜 upstream pin 라이브러리. harness/pin/driver.py 가 부르는 이름만 갖는다."""
import json

OPS_BY_VIA = {"web": frozenset({"set_status"}), "command": frozenset({"set_status"}), "extract": frozenset({"add"})}
STATUSES = ("todo", "done")
WAIT_REASONS = ("reply",)
SOURCES = ("wiki",)
STAGES = ("inbox", "sources")
BASES = ("stated",)
ITEM_KEYS = ("id", "title")
EVIDENCE_KEYS = ("path", "quote")
STATE_EXTRA = %s


def init_pin(pin_dir):
    pin_dir.mkdir(parents=True, exist_ok=True)
    (pin_dir / "items.json").write_text("[]", encoding="utf-8")
    return ["items.json"]


def state(items):
    out = {"schema": 1, "items": [{"id": i["id"], "title": i["title"]} for i in items]}
    if STATE_EXTRA:
        out["recent"] = []
    return out


def dump(obj):
    return json.dumps(obj, ensure_ascii=False, indent=1) + "\n"
`

	fakeApply = `"""시험용 가짜 upstream apply. add 는 항목을 더하고 그 밖의 연산은 op_not_allowed 로 거절한다."""
import json

import pinlib

%s
def load_request_files(paths):
    out = []
    for p in paths:
        out += json.loads(p.read_text(encoding="utf-8"))["requests"]
    return out


def run(wiki, reqs, now, state_path, use_intake=True):
    items_path = wiki / "pin" / "items.json"
    items = json.loads(items_path.read_text(encoding="utf-8"))
    results = []
    for r in reqs:
        if r.get("op") == "add" and r.get("via") == "extract":
            items.append({"id": "P%%04d" %% (len(items) + 1), "title": r["args"]["title"]})
            results.append({"id": r["id"], "result": "applied", "code": "ok", "message": "새 항목"})
        else:
            results.append({"id": r["id"], "result": "rejected", "code": "op_not_allowed", "message": "%s"})
    items_path.write_text(pinlib.dump(items), encoding="utf-8")
    state_path.write_text(pinlib.dump(pinlib.state(items)), encoding="utf-8")
    return {"results": results, "archived": [], "written": ["items.json"]}
`

	fakeRules = `# Pin 할 일 원장 규칙

## 위키 문서에서 할 일을 찾을 때 (` + "`source: wiki`" + `)

- 위키 규칙 하나.

## 미리알림에서 들어온 일 (` + "`source: reminder`" + `)

%s

## 직접 입력과 원장 쓰기

직접 입력 규칙 하나.

## 항목·관계·제안

항목 규칙 하나.

## 삭제

삭제의 첫 문단.

삭제의 둘째 문단.
`
)

// fakeState는 가짜 upstream 의 한 시점이다. 필드 하나가 표면 하나에 대응한다.
type fakeState struct {
	page      string // template.html 전체
	stateKeys bool   // 상태에 키를 더한다(상태 형태)
	rejectMsg string // 거절 메시지(게이트)
	reminder  string // 미리알림 절 본문(규칙 문서에서 engram 이 기대지 않는 절)
	wikiRule  string // 위키 절 본문 앞에 붙는 글(규칙 문서에서 engram 이 기대는 절). 비면 그대로
	crash     bool   // apply.py 가 import 때 예외를 낸다(돌지 못함)
}

const goodPage = `<!doctype html><html><head><meta charset="utf-8"><title>Pin</title><!-- pin:site begin --><link rel="icon" href="https://example.invalid/icon.png"><!-- pin:site end --></head>
<body><main id="view"></main>
<script>
(function(){
// pin:config begin
// 설정 구역
var API='/e/p/api/', SOURCES=['wiki','reminder'], AUTH='access';
// pin:config end
function send(){return fetch(API+'changes',{method:'POST',body:JSON.stringify({op:'set_status'})})}
function load(){return fetch(API+'state')}
})();
</script>%s</body></html>
`

func basePage(extra string) string { return fmt.Sprintf(goodPage, extra) }

func baseState() fakeState {
	return fakeState{page: basePage(""), rejectMsg: "허용되지 않은 연산", reminder: "미리알림 규칙 하나."}
}

func (s fakeState) write(t *testing.T, dir string) {
	t.Helper()
	crash := ""
	if s.crash {
		crash = "raise RuntimeError('가짜 upstream 이 깨졌다')\n"
	}
	rules := fmt.Sprintf(fakeRules, s.reminder)
	if s.wikiRule != "" {
		rules = strings.Replace(rules, "- 위키 규칙 하나.", "- 위키 규칙 하나. "+s.wikiRule, 1)
	}
	files := map[string]string{
		"AGENTS.md":                 fakeAgents,
		"meta/CHANGELOG.md":         fakeChangelog,
		"meta/pin-rules.md":         rules,
		"scripts/pin/pinlib.py":     fmt.Sprintf(fakePinlib, map[bool]string{true: "True", false: "False"}[s.stateKeys]),
		"scripts/pin/apply.py":      fmt.Sprintf(fakeApply, crash, s.rejectMsg),
		"scripts/pin/template.html": s.page,
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runIn(t *testing.T, dir string, name string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return out.String(), code
}

func commit(t *testing.T, dir string, s fakeState, msg string) string {
	t.Helper()
	s.write(t, dir)
	if out, code := runIn(t, dir, "git", "add", "-A"); code != 0 {
		t.Fatalf("git add: %s", out)
	}
	if out, code := runIn(t, dir, "git", "commit", "-q", "--allow-empty", "-m", msg); code != 0 {
		t.Fatalf("git commit: %s", out)
	}
	out, _ := runIn(t, dir, "git", "rev-parse", "HEAD")
	return strings.TrimSpace(out)
}

// judgeEnv는 가짜 upstream(두 커밋)과 가짜 engram 루트를 만든다. 루트의 lock 은 첫 커밋을 가리킨다.
type judgeEnv struct {
	up, root string
	base     string
}

func newJudgeEnv(t *testing.T, first fakeState) *judgeEnv {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 이 없어 건너뜁니다")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 가 없어 건너뜁니다")
	}
	e := &judgeEnv{up: t.TempDir(), root: t.TempDir()}
	if out, code := runIn(t, e.up, "git", "init", "-q"); code != 0 {
		t.Fatalf("git init: %s", out)
	}
	e.base = commit(t, e.up, first, "첫 시점")

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"scripts/upstream-sync.py", "scripts/check-boundary.py", "harness/pin/driver.py"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(e.root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyTree(t, filepath.Join("fixtures"), filepath.Join(e.root, "harness", "pin", "fixtures"))
	e.writeLock(t, e.base)
	return e
}

func (e *judgeEnv) writeLock(t *testing.T, commit string) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"commit": commit, "files": []string{"pin-rules.md"}, "synced_at": "2026-11-01T00:00:00"})
	if err := os.WriteFile(filepath.Join(e.root, "harness", "upstream.lock"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (e *judgeEnv) sync(t *testing.T, args ...string) (string, int) {
	t.Helper()
	return runIn(t, e.root, "python3", append([]string{"scripts/upstream-sync.py", "--upstream", e.up}, args...)...)
}

func (e *judgeEnv) embedded() (string, bool) {
	data, err := os.ReadFile(filepath.Join(e.root, "internal", "pin", "web", "template.html"))
	return string(data), err == nil
}

func (e *judgeEnv) lockCommit(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.root, "harness", "upstream.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var l struct {
		Commit string `json:"commit"`
	}
	if err := json.Unmarshal(data, &l); err != nil {
		t.Fatal(err)
	}
	return l.Commit
}

func verdictOf(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "pin 판정: "); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// 표면 줄을 한 줄씩 읽는다. 잡은 `갱신 pin <표면>` 으로 무엇이 바뀌었는지 안다.
func changedSurfaces(out string) []string {
	var got []string
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "갱신 pin "); ok {
			got = append(got, strings.Fields(rest)[0]+firstSecond(rest))
		}
	}
	return got
}

func firstSecond(rest string) string {
	f := strings.Fields(rest)
	if len(f) > 1 && !strings.HasPrefix(f[1], "(") {
		return " " + f[1]
	}
	return ""
}

func TestJudgeScreenOnlyPullsThePage(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	next := baseState()
	next.page = basePage("<p>새 안내 문구</p>")
	commit(t, e.up, next, "화면 문구만 바뀜")

	out, code := e.sync(t, "--check")
	if code != 0 || verdictOf(out) != "화면만" {
		t.Fatalf("화면만 바뀐 회차의 판정이 아니다 (종료 %d)\n%s", code, out)
	}
	if got := changedSurfaces(out); len(got) != 1 || got[0] != "화면" {
		t.Errorf("바뀐 표면이 화면 하나여야 한다: %v\n%s", got, out)
	}
	if _, ok := e.embedded(); ok {
		t.Error("--check 가 화면 파일을 썼다")
	}

	out, code = e.sync(t)
	if code != 0 || verdictOf(out) != "화면만" {
		t.Fatalf("실제 실행이 실패했다 (종료 %d)\n%s", code, out)
	}
	page, ok := e.embedded()
	if !ok {
		t.Fatalf("화면을 가져오지 않았다\n%s", out)
	}
	want := strings.NewReplacer(
		"var API='/e/p/api/', SOURCES=['wiki','reminder'], AUTH='access';", "var API='/pin/api/', SOURCES=['wiki'], AUTH='none';",
		`<!-- pin:site begin --><link rel="icon" href="https://example.invalid/icon.png"><!-- pin:site end -->`, "",
	).Replace(basePage("<p>새 안내 문구</p>"))
	if page != want {
		t.Errorf("가져온 화면이 기대와 다르다\n--- 기대\n%s\n--- 실제\n%s", want, page)
	}
	if strings.Contains(page, "example.invalid") || strings.Contains(page, "SOURCES=['wiki','reminder']") {
		t.Error("사이트 구역이나 upstream 설정 값이 남았다")
	}
	if e.lockCommit(t) == e.base {
		t.Error("lock 이 새 커밋으로 가지 않았다")
	}
}

func TestJudgeOtherSurfacesGoToPR(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*fakeState)
		surface string
	}{
		{"상태 형태", func(s *fakeState) { s.stateKeys = true }, "상태 형태"},
		{"게이트", func(s *fakeState) { s.rejectMsg = "연산이 허용되지 않았습니다" }, "게이트"},
		{"화면 요청", func(s *fakeState) { s.page = strings.Replace(s.page, "op:'set_status'", "op:'set_owner'", 1) }, "화면 요청"},
		{"규칙 문서", func(s *fakeState) { s.wikiRule = "문장을 더했다." }, "규칙"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newJudgeEnv(t, baseState())
			next := baseState()
			next.page = basePage("<p>화면도 함께 바뀜</p>")
			c.mutate(&next)
			commit(t, e.up, next, c.name+" 이 바뀜")

			out, code := e.sync(t, "--check")
			if code != 0 || verdictOf(out) != "PR" {
				t.Fatalf("PR 대상이어야 한다 (종료 %d)\n%s", code, out)
			}
			found := false
			for _, s := range changedSurfaces(out) {
				if strings.HasPrefix(s, c.surface) {
					found = true
				}
			}
			if !found {
				t.Errorf("바뀐 표면에 %q 가 없다: %v\n%s", c.surface, changedSurfaces(out), out)
			}
			if !strings.Contains(out, "화면은 가져오지 않습니다") {
				t.Errorf("화면을 가져오지 않는다는 줄이 없다\n%s", out)
			}

			out, code = e.sync(t)
			if code != 0 || verdictOf(out) != "PR" {
				t.Fatalf("실제 실행의 판정이 PR 이 아니다 (종료 %d)\n%s", code, out)
			}
			if _, ok := e.embedded(); ok {
				t.Error("다른 표면이 바뀐 회차에 화면을 가져왔다")
			}
		})
	}
}

func TestJudgeNothingChanged(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	commit(t, e.up, baseState(), "아무것도 안 바뀜")
	out, code := e.sync(t, "--check")
	if code != 0 || verdictOf(out) != "유지" {
		t.Fatalf("유지여야 한다 (종료 %d)\n%s", code, out)
	}
	if got := changedSurfaces(out); len(got) != 0 {
		t.Errorf("바뀐 표면이 없어야 한다: %v", got)
	}
	out, code = e.sync(t)
	if code != 0 || verdictOf(out) != "유지" {
		t.Fatalf("실제 실행이 유지가 아니다 (종료 %d)\n%s", code, out)
	}
	if _, ok := e.embedded(); ok {
		t.Error("유지인데 화면을 썼다")
	}
}

// 미리알림 절처럼 engram 이 기대지 않는 절만 바뀐 주는 유지다. 0104 가 걱정한 볼 것 없는 PR 을 거른다.
func TestJudgeUnexpectedSectionIsIgnored(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	next := baseState()
	next.reminder = "미리알림 규칙을 통째로 고쳤다."
	commit(t, e.up, next, "미리알림 절만 바뀜")
	out, code := e.sync(t, "--check")
	if code != 0 || verdictOf(out) != "유지" {
		t.Fatalf("기대하지 않는 절의 변화는 유지여야 한다 (종료 %d)\n%s", code, out)
	}
}

func TestJudgePageWithoutMarksFails(t *testing.T) {
	cases := map[string]string{
		"표시 없음":  strings.Replace(basePage("<p>x</p>"), "// pin:config begin\n", "", 1),
		"표시 둘":   basePage("<!-- pin:site begin --><!-- pin:site end -->"),
		"값 줄 없음": strings.Replace(basePage("<p>x</p>"), "var API='/e/p/api/', SOURCES=['wiki','reminder'], AUTH='access';", "var API=location.pathname;", 1),
	}
	for name, page := range cases {
		t.Run(name, func(t *testing.T) {
			e := newJudgeEnv(t, baseState())
			next := baseState()
			next.page = page
			commit(t, e.up, next, "화면 구조가 바뀜")
			out, code := e.sync(t)
			if code == 0 || !strings.Contains(out, "화면을 가져오지 않습니다") {
				t.Fatalf("가져오지 않고 실패해야 한다 (종료 %d)\n%s", code, out)
			}
			if _, ok := e.embedded(); ok {
				t.Error("실패했는데 화면을 썼다")
			}
			if e.lockCommit(t) != e.base {
				t.Error("실패했는데 lock 을 옮겼다")
			}
		})
	}
}

// 경계 검사에 걸리는 화면은 가져오지 않고 이전 화면을 되돌린다. 입력 예시의 실명 같은 경우를 여기서 잡는다.
func TestJudgeBoundaryHitRollsBack(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	old := "<!-- 이전 화면 -->\n"
	pageDir := filepath.Join(e.root, "internal", "pin", "web")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pageDir, "template.html"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(e.root, "private"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.root, "private", "boundary-patterns.txt"), []byte("금지된실명\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	next := baseState()
	next.page = basePage("<p>예: 금지된실명 (비워도 됩니다)</p>")
	commit(t, e.up, next, "화면에 금지된 글자가 들어옴")

	out, code := e.sync(t)
	if code == 0 || !strings.Contains(out, "경계 검사 적중") {
		t.Fatalf("경계 검사에서 실패해야 한다 (종료 %d)\n%s", code, out)
	}
	if strings.Contains(out, "금지된실명") {
		t.Error("걸린 글자를 출력했다")
	}
	if page, _ := e.embedded(); page != old {
		t.Errorf("이전 화면으로 되돌리지 않았다: %q", page)
	}
	if e.lockCommit(t) != e.base {
		t.Error("실패했는데 lock 을 옮겼다")
	}
}

// 견본에 돌지 못한 회차는 PR 이다. 가져오지 않는다.
func TestJudgeCrashGoesToPR(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	next := baseState()
	next.crash = true
	next.page = basePage("<p>화면도 바뀜</p>")
	commit(t, e.up, next, "upstream 이 깨짐")
	out, code := e.sync(t)
	if code != 0 || verdictOf(out) != "PR" || !strings.Contains(out, "돌지 못함") {
		t.Fatalf("돌지 못함은 PR 이어야 한다 (종료 %d)\n%s", code, out)
	}
	if _, ok := e.embedded(); ok {
		t.Error("돌지 못한 회차에 화면을 가져왔다")
	}
}

// --pull-page 는 PR 판정이어도 화면을 가져온다. 사람이 PR 브랜치에서 Go 구현을 고친 뒤 쓴다.
func TestJudgePullPageSwitch(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	next := baseState()
	next.stateKeys = true
	next.page = basePage("<p>새 문구</p>")
	commit(t, e.up, next, "상태 형태와 화면이 바뀜")
	out, code := e.sync(t, "--pull-page")
	if code != 0 || verdictOf(out) != "PR" {
		t.Fatalf("판정은 PR 이어야 한다 (종료 %d)\n%s", code, out)
	}
	if page, ok := e.embedded(); !ok || !strings.Contains(page, "새 문구") {
		t.Errorf("--pull-page 가 화면을 가져오지 않았다\n%s", out)
	}
}

// HELD 가 비었으므로 pin-rules.md 는 기대하는 절만 사본으로 들어온다.
func TestJudgePinRulesExcerpt(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	if _, code := e.sync(t); code != 0 {
		t.Fatal("sync 가 실패했다")
	}
	data, err := os.ReadFile(filepath.Join(e.root, "harness", "upstream", "pin-rules.md"))
	if err != nil {
		t.Fatalf("pin-rules.md 사본이 없다: %v", err)
	}
	text := string(data)
	for _, want := range []string{"위키 규칙 하나.", "직접 입력 규칙 하나.", "항목 규칙 하나.", "삭제의 첫 문단."} {
		if !strings.Contains(text, want) {
			t.Errorf("기대하는 절 %q 가 사본에 없다", want)
		}
	}
	for _, bad := range []string{"미리알림 규칙 하나.", "삭제의 둘째 문단."} {
		if strings.Contains(text, bad) {
			t.Errorf("기대하지 않는 글 %q 가 사본에 있다", bad)
		}
	}
}

// upstream 이 기대하는 절의 이름을 바꾸면 조용히 빈 사본을 만들지 않고 실패한다.
func TestJudgePinRulesSectionRenamedFails(t *testing.T) {
	e := newJudgeEnv(t, baseState())
	next := baseState()
	p := filepath.Join(e.up, "meta", "pin-rules.md")
	commit(t, e.up, next, "같은 내용")
	data, _ := os.ReadFile(p)
	renamed := strings.Replace(string(data), "## 직접 입력과 원장 쓰기", "## 직접 입력", 1)
	if err := os.WriteFile(p, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	runIn(t, e.up, "git", "add", "-A")
	runIn(t, e.up, "git", "commit", "-q", "-m", "절 이름이 바뀜")
	out, code := e.sync(t)
	if code == 0 || !strings.Contains(out, "기대하는 절") {
		t.Fatalf("절 이름이 바뀌면 실패해야 한다 (종료 %d)\n%s", code, out)
	}
}
