// Package docs는 문서가 코드에 대해 말하는 정량 주장을 실행 결과와
// 대조한다.
//
// 이 저장소는 코드가 코드에게 하는 약속은 harness 여섯으로 지켜 왔으나,
// 문서가 코드에게 하는 약속은 전부 사람이 지켰다. AGENTS.md 가 "인용하는
// 커맨드 출력은 전부 실측값이다" 라고 선언해 놓고도 그것을 지키는 장치가
// 없었다. 그래서 커맨드가 스물아홉인데 문서 넷이 스물여덟이라 적고, lint
// 규칙이 스물인데 문서들이 열넷, 열일곱, 열아홉, 스물을 동시에 적는 상태가
// 되었다. 어느 쪽도 CI 가 잡지 못했다.
//
// 같은 저장소의 AGENTS.md 가 이미 원칙을 적어 두었다. 에이전트는 셸을
// 쥐고 있어 문서로 막을 수 없으므로 강제는 커맨드와 lint 에서 한다는
// 것이다. 그 원칙을 코드에만 적용하고 문서에는 적용하지 않은 것이
// 격차였다. 이 하니스가 그 자리를 메운다.
//
// 대조하는 것은 셋이다. 커맨드 목록과 그 수, lint 규칙 목록과 그 수,
// ADR 색인의 번호와 상태와 날짜다. 셋 다 바이너리나 파일에서 기계적으로
// 도출되는 값이라 사람이 옮겨 적을 이유가 없다.
package docs

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// binaryPath는 TestMain 이 빌드한 바이너리다. go 가 없으면 비어 있다.
var binaryPath string

// repoRoot는 모듈 루트다. 테스트가 harness/docs 에서 돈다.
var repoRoot = filepath.Join("..", "..")

func TestMain(m *testing.M) {
	// 출력 언어를 못 박는다. 개발자 환경의 ENGRAM_LANG 이 새어 들어오면
	// 커맨드 설명도 규칙 설명도 다른 언어로 나와 파싱이 어긋난다.
	if err := os.Setenv("ENGRAM_LANG", "ko"); err != nil {
		panic(err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("go 가 PATH 에 없어 docs 하니스를 건너뛴다")
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "engram-docs-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "임시 디렉토리를 만들 수 없음: %v\n", err)
		os.Exit(1)
	}
	name := "engram"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(dir, name)
	build := exec.Command(goBin, "build", "-o", bin, "./cmd/engram")
	build.Dir = repoRoot
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		fmt.Fprintf(os.Stderr, "engram 바이너리 빌드 실패: %v\n%s\n", err, out)
		os.Exit(1)
	}
	binaryPath = bin
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// touchSources는 구현 소스를 읽어 go 의 테스트 결과 캐시를 무력화한다.
// 이 패키지는 바이너리를 exec 하므로 소스 의존이 없다. 커맨드가 늘어도
// 캐시된 통과가 남아 있으면 이 하니스는 아무것도 지키지 못한다.
// harness/examples 와 같은 이유이고 같은 방법이다.
func touchSources() {
	for _, root := range []string{"cmd", "internal"} {
		filepath.WalkDir(filepath.Join(repoRoot, root), func(path string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			os.ReadFile(path)
			return nil
		})
	}
}

// ---------------------------------------------------------------------
// 살아 있는 문서와 시점 기록
// ---------------------------------------------------------------------

// recordMarker는 그 문서가 특정 시점의 기록임을 선언하는 표시다. 이
// 표시가 있는 문서는 지금의 코드와 어긋나도 회귀가 아니다. 검증 보고서와
// 대조 보고서가 그렇다. 표시가 없으면 살아 있는 문서로 보고 지금의
// 코드와 맞아야 한다.
//
// 표시를 문서 안에 두는 이유가 있다. 하니스 쪽에 파일 목록을 두면 새
// 보고서가 생길 때마다 하니스를 고쳐야 하고, 그 목록이 또 하나의 손으로
// 맞추는 파생 상태가 된다. 문서가 자기 성격을 스스로 선언하게 두면
// 목록이 필요 없다.
var recordMarker = regexp.MustCompile(`<!--\s*engram:record\s+as-of=(\d{4}-\d{2}-\d{2})\s*-->`)

// headBytes는 시점 표시를 찾는 범위다. 표시는 문서 첫머리에 두는 것이
// 규약이므로 앞쪽만 본다. 본문 전체를 뒤지면 이 규약을 설명하는 문서가
// 자기 예시 때문에 면제를 받아 버린다.
const headBytes = 512

// liveDocs는 지금의 코드와 맞아야 하는 문서 전부다.
//
// docs/decisions/ 는 뺀다. ADR 은 소급 수정하지 않는 것이 계약이라
// (AGENTS.md) 결정 당시의 수를 적은 것이 옳다. 그것을 지금 값으로
// 맞추라고 요구하면 계약이 깨진다.
func liveDocs(t *testing.T) []string {
	t.Helper()
	var out []string
	roots := []string{
		filepath.Join(repoRoot, "docs"),
		filepath.Join(repoRoot, "README.md"),
		filepath.Join(repoRoot, "AGENTS.md"),
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				// ADR 은 소급 수정 금지 대상이라 검사하지 않는다.
				if filepath.Base(p) == "decisions" {
					return fs.SkipDir
				}
				// 자산 디렉토리에는 산문이 없다.
				if filepath.Base(p) == "assets" {
					return fs.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext != ".md" && ext != ".html" {
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			head := raw
			if len(head) > headBytes {
				head = head[:headBytes]
			}
			if recordMarker.Match(head) {
				return nil
			}
			out = append(out, p)
			return nil
		})
		if err != nil {
			t.Fatalf("문서를 훑을 수 없음(%s): %v", root, err)
		}
	}
	sort.Strings(out)
	return out
}

// rel은 보고용 상대 경로다.
func rel(p string) string {
	r, err := filepath.Rel(repoRoot, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}

// TestRecordMarkersAreWellFormed는 시점 기록 표시가 제 모양인지 본다.
// 이 표시가 검사를 면제하므로, 오타 난 표시가 소리 없이 면제로 굳는 일을
// 막는다. 표시를 단 문서는 그 표시 아래에서 무엇이든 적을 수 있으니
// 표시 자체는 엄격해야 한다.
//
// 문서 머리만 본다. 표시는 첫머리에 두는 것이 규약이고, 본문 어디서나
// 잡으면 이 규약을 설명하는 문서가 자기 예시에 걸린다. ADR 0096 이
// 실제로 그랬다.
func TestRecordMarkersAreWellFormed(t *testing.T) {
	loose := regexp.MustCompile(`(?i)<!--\s*engram:record`)
	var bad []string
	for _, root := range []string{filepath.Join(repoRoot, "docs"), filepath.Join(repoRoot, "README.md"), filepath.Join(repoRoot, "AGENTS.md")} {
		filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext != ".md" && ext != ".html" {
				return nil
			}
			raw, _ := os.ReadFile(p)
			head := raw
			if len(head) > headBytes {
				head = head[:headBytes]
			}
			if loose.Match(head) && !recordMarker.Match(head) {
				bad = append(bad, rel(p))
			}
			return nil
		})
	}
	if len(bad) > 0 {
		t.Fatalf("시점 기록 표시의 모양이 어긋난 문서가 있다. 정확한 모양은 <!-- engram:record as-of=YYYY-MM-DD --> 이고 문서 첫머리에 둔다:\n  %s",
			strings.Join(bad, "\n  "))
	}
}

// ---------------------------------------------------------------------
// 한국어 수사
// ---------------------------------------------------------------------

// koTens와 koUnits는 문서가 쓰는 고유어 수사다. AGENTS.md 의 문체 규칙이
// 산문에서 수를 고유어로 적게 하므로 숫자만 봐서는 주장을 찾을 수 없다.
var koTens = map[string]int{
	"열": 10, "스물": 20, "서른": 30, "마흔": 40, "쉰": 50,
	"예순": 60, "일흔": 70, "여든": 80, "아흔": 90,
}

var koUnits = map[string]int{
	"하나": 1, "한": 1, "둘": 2, "두": 2, "셋": 3, "세": 3, "넷": 4, "네": 4,
	"다섯": 5, "여섯": 6, "일곱": 7, "여덟": 8, "아홉": 9,
}

// koNumeral은 고유어 수사 하나를 수로 옮긴다. 옮기지 못하면 ok 가 거짓이다.
func koNumeral(s string) (int, bool) {
	if v, ok := koUnits[s]; ok {
		return v, true
	}
	if v, ok := koTens[s]; ok {
		return v, true
	}
	// 스물여덟 같은 겹수사. 십의 자리를 먼저 떼고 나머지를 단위로 본다.
	for tens, tv := range koTens {
		if strings.HasPrefix(s, tens) {
			restStr := strings.TrimPrefix(s, tens)
			if restStr == "" {
				return tv, true
			}
			if uv, ok := koUnits[restStr]; ok {
				return tv + uv, true
			}
		}
	}
	return 0, false
}

// claim은 문서 한 곳이 말하는 수 하나다.
type claim struct {
	file string
	line int
	text string
	got  int
}

// pastMarker는 문장 하나가 지난 시점의 기록임을 밝히는 말이다.
//
// 시점 표시(recordMarker)는 문서 전체를 면제하지만, roadmap 처럼 살아
// 있는 문서가 완료 기록을 함께 품는 경우가 있다. "0.3 시점 README 는
// 당시 커맨드 스물 개를 정리했다" 는 지금도 참인 문장이고 지금 값으로
// 고치면 오히려 거짓이 된다. 그런 자리를 문장 단위로 면제한다.
//
// 면제의 범위를 좁게 둔다. 같은 줄에 이 말이 있어야 하고, 말 자체가
// 산문에서 눈에 띄므로 사람이 읽다가 남용을 알아챌 수 있다.
var pastMarker = regexp.MustCompile(`당시|시절|무렵|그때`)

// scanClaims는 살아 있는 문서에서 pattern 이 잡는 주장을 전부 모은다.
// pattern 의 첫 그룹이 수를 담아야 한다.
func scanClaims(t *testing.T, pattern *regexp.Regexp) []claim {
	t.Helper()
	var out []claim
	for _, p := range liveDocs(t) {
		f, err := os.Open(p)
		if err != nil {
			t.Fatalf("문서를 열 수 없음(%s): %v", rel(p), err)
		}
		sc := bufio.NewScanner(f)
		// 덱의 HTML 은 한 줄이 매우 길다. 기본 버퍼로는 잘린다.
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		n := 0
		for sc.Scan() {
			n++
			line := sc.Text()
			if pastMarker.MatchString(line) {
				continue
			}
			for _, m := range pattern.FindAllStringSubmatch(line, -1) {
				v, ok := parseCount(m[1])
				if !ok {
					continue
				}
				out = append(out, claim{file: rel(p), line: n, text: strings.TrimSpace(m[0]), got: v})
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			t.Fatalf("문서를 읽을 수 없음(%s): %v", rel(p), err)
		}
	}
	return out
}

// parseCount는 아라비아 숫자와 고유어 수사를 모두 받는다.
func parseCount(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if s[0] >= '0' && s[0] <= '9' {
		v := 0
		for _, r := range s {
			if r < '0' || r > '9' {
				return 0, false
			}
			v = v*10 + int(r-'0')
		}
		return v, true
	}
	return koNumeral(s)
}

// reportClaims는 어긋난 주장을 전부 모아 한 번에 낸다. 하나씩 고치게
// 하면 같은 검사를 열 번 돌려야 한다.
func reportClaims(t *testing.T, what string, want int, claims []claim) {
	t.Helper()
	if len(claims) == 0 {
		t.Fatalf("%s 를 말하는 문서가 하나도 없다. 주장을 찾는 정규식이 문체 변경으로 헛돌고 있을 수 있다", what)
	}
	var bad []claim
	for _, c := range claims {
		if c.got != want {
			bad = append(bad, c)
		}
	}
	if len(bad) == 0 {
		t.Logf("%s: %d, 주장 %d곳 일치", what, want, len(claims))
		return
	}
	var b strings.Builder
	for _, c := range bad {
		fmt.Fprintf(&b, "  %s:%d  적힌 값 %d  (실제 %d)  %q\n", c.file, c.line, c.got, want, c.text)
	}
	t.Fatalf("%s 가 실제와 다르게 적힌 자리가 %d곳이다. 실제 값은 %d 이다.\n%s",
		what, len(bad), want, b.String())
}

// ---------------------------------------------------------------------
// 커맨드
// ---------------------------------------------------------------------

// helpCommands는 --help 가 내는 커맨드 이름 전부다. AGENTS.md 가 이
// 목록을 진실원으로 선언했다.
//
// completion 과 help 는 뺀다. cobra 가 자동으로 붙이는 것이라 이
// 저장소가 설계한 커맨드가 아니고, 문서도 그것을 세지 않는다.
func helpCommands(t *testing.T) []string {
	t.Helper()
	out := runBinary(t, "--help")
	var names []string
	inSection := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Available Commands:") {
			inSection = true
			continue
		}
		if inSection {
			if strings.TrimSpace(line) == "" || !strings.HasPrefix(line, "  ") {
				break
			}
			f := strings.Fields(line)
			if len(f) == 0 {
				continue
			}
			if f[0] == "completion" || f[0] == "help" {
				continue
			}
			names = append(names, f[0])
		}
	}
	if len(names) == 0 {
		t.Fatalf("--help 에서 커맨드를 하나도 읽지 못했다. 출력 모양이 바뀌었을 수 있다:\n%s", out)
	}
	sort.Strings(names)
	return names
}

// runBinary는 커맨드를 돌리고 표준 출력과 표준 오류를 합쳐 낸다.
func runBinary(t *testing.T, args ...string) string {
	t.Helper()
	if binaryPath == "" {
		t.Skip("바이너리가 없어 대조하지 못한다. go 가 PATH 에 있어야 한다")
	}
	cmd := exec.Command(binaryPath, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("engram %s 실패: %v\n%s", strings.Join(args, " "), err, buf.String())
	}
	return buf.String()
}

// commandCountClaim은 커맨드 수를 말하는 자리를 잡는다.
//
// 뒤에 회, 번, 차가 오면 횟수이지 개수가 아니므로 뺀다. "조회 커맨드
// 5회 반복" 같은 문장이 실제로 있다.
var commandCountClaim = regexp.MustCompile(`커맨드(?:는|가|를)?\s+([0-9]+|[가-힣]{1,6}?)\s*(?:개|종)?(?:이며|이고|입니다|이다|은|는|을|를|,|\.|$)`)

// TestCommandCountClaimsMatchHelp는 문서가 적은 커맨드 수를 --help 와
// 대조한다.
func TestCommandCountClaimsMatchHelp(t *testing.T) {
	touchSources()
	cmds := helpCommands(t)
	claims := scanClaims(t, commandCountClaim)
	// 횟수를 세는 자리를 걸러낸다.
	var kept []claim
	for _, c := range claims {
		if strings.Contains(c.text, "회") || strings.Contains(c.text, "번") {
			continue
		}
		// 커맨드 하나, 커맨드 둘 같은 표현은 개수 주장이 아니라 문장의
		// 일부다. 전체 커맨드 수를 말하는 자리는 열 이상이다.
		if c.got < 10 {
			continue
		}
		kept = append(kept, c)
	}
	reportClaims(t, "커맨드 수", len(cmds), kept)
}

// TestCommandListInAgentsMatchesHelp는 AGENTS.md 가 나열한 커맨드
// 이름들이 실제 목록과 같은지 본다. 수만 맞고 이름이 틀리면 수 검사는
// 통과한다. model 이 빠졌을 때 실제로 그랬다.
func TestCommandListInAgentsMatchesHelp(t *testing.T) {
	touchSources()
	want := helpCommands(t)
	wantSet := map[string]bool{}
	for _, c := range want {
		wantSet[c] = true
	}

	path := filepath.Join(repoRoot, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("AGENTS.md 를 읽을 수 없음: %v", err)
	}
	// 커맨드 목록을 적은 줄을 찾는다. 갈래 표기가 그 줄의 표식이다.
	var listLine string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "engram --help") && strings.Contains(line, "`") {
			listLine = line
			break
		}
	}
	if listLine == "" {
		t.Fatalf("AGENTS.md 에서 커맨드 목록 줄을 찾지 못했다. `engram --help` 를 진실원으로 적은 줄이 있어야 한다")
	}

	tick := regexp.MustCompile("`([a-z][a-z ]*)`")
	gotSet := map[string]bool{}
	for _, m := range tick.FindAllStringSubmatch(listLine, -1) {
		name := strings.Fields(m[1])[0]
		if name == "engram" {
			continue
		}
		gotSet[name] = true
	}

	var missing, extra []string
	for c := range wantSet {
		if !gotSet[c] {
			missing = append(missing, c)
		}
	}
	for c := range gotSet {
		if !wantSet[c] {
			extra = append(extra, c)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("AGENTS.md 의 커맨드 목록이 --help 와 다르다.\n  목록에 빠진 것: %v\n  목록에만 있는 것: %v\n  실제 목록: %v",
			missing, extra, want)
	}
	t.Logf("AGENTS.md 커맨드 목록 %d개 일치", len(want))
}

// ---------------------------------------------------------------------
// lint 규칙
// ---------------------------------------------------------------------

// ruleIDLine은 rules show 의 규칙 줄에서 규칙 ID 를 뽑는다.
var ruleIDLine = regexp.MustCompile(`^\s*\[[^\]]+\]\s+([a-z][a-z0-9]*\.[a-z-]+)\b`)

// ruleHeader는 rules show 가 스스로 적는 규칙 수다.
var ruleHeader = regexp.MustCompile(`lint 규칙\s+([0-9]+|[가-힣]{1,6})종`)

// lintRules는 rules show 가 내는 규칙 ID 전부다.
//
// 규칙 목록은 위키를 요구하므로 임시 위키를 하나 만들어 돌린다. 프리셋은
// 기본값인 personal 을 쓴다.
func lintRules(t *testing.T) []string {
	t.Helper()
	if binaryPath == "" {
		t.Skip("바이너리가 없어 대조하지 못한다. go 가 PATH 에 있어야 한다")
	}
	wiki := filepath.Join(t.TempDir(), "w")
	runBinary(t, "init", wiki, "--preset", "personal")
	out := runBinary(t, "rules", "show", "--wiki", wiki)

	var ids []string
	for _, line := range strings.Split(out, "\n") {
		if m := ruleIDLine.FindStringSubmatch(line); m != nil {
			ids = append(ids, m[1])
		}
	}
	if len(ids) == 0 {
		t.Fatalf("rules show 에서 규칙 ID 를 하나도 읽지 못했다. 출력 모양이 바뀌었을 수 있다:\n%s", out)
	}

	// rules show 가 자기 머리글에 적은 수와 실제로 낸 줄 수가 어긋나면
	// 그것부터 구현 결함이다.
	if m := ruleHeader.FindStringSubmatch(out); m != nil {
		if v, ok := parseCount(m[1]); ok && v != len(ids) {
			t.Fatalf("rules show 의 머리글이 %d종이라 적었는데 실제로 낸 규칙은 %d개다", v, len(ids))
		}
	} else {
		t.Fatalf("rules show 출력에서 규칙 수 머리글을 찾지 못했다")
	}

	sort.Strings(ids)
	return ids
}

// ruleCountClaim은 lint 규칙 수를 말하는 자리를 잡는다. 규칙 명세 7종
// 처럼 다른 것을 세는 자리와 섞이지 않도록 lint 를 함께 요구한다.
var ruleCountClaim = regexp.MustCompile(`(?:lint )?규칙\s+([0-9]+|[가-힣]{1,6})종`)

// TestLintRuleCountClaimsMatchRulesShow는 문서가 적은 lint 규칙 수를
// rules show 와 대조한다.
func TestLintRuleCountClaimsMatchRulesShow(t *testing.T) {
	touchSources()
	rules := lintRules(t)
	claims := scanClaims(t, ruleCountClaim)
	var kept []claim
	for _, c := range claims {
		// 규칙 명세 7종은 upstream 문서의 수이지 lint 규칙이 아니다.
		if strings.Contains(c.text, "명세") {
			continue
		}
		kept = append(kept, c)
	}
	reportClaims(t, "lint 규칙 수", len(rules), kept)
}

// ---------------------------------------------------------------------
// ADR 색인
// ---------------------------------------------------------------------

// adrFrontmatter는 ADR 의 frontmatter 네 키를 읽는다.
var adrFrontmatter = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)

// adrIndexRow는 색인 목록표의 한 행이다.
var adrIndexRow = regexp.MustCompile(`^\|\s*\[(\d{4})\]\(([^)]+)\)\s*\|([^|]*)\|\s*([0-9-]+)\s*\|\s*(\w+)\s*\|`)

type adr struct {
	number string
	title  string
	date   string
	status string
	file   string
}

// readADRs는 docs/decisions 의 ADR 파일 전부를 읽는다.
func readADRs(t *testing.T) map[string]adr {
	t.Helper()
	dir := filepath.Join(repoRoot, "docs", "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ADR 디렉토리를 읽을 수 없음: %v", err)
	}
	out := map[string]adr{}
	kv := regexp.MustCompile(`(?m)^(\w+):\s*(.+)$`)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("ADR 을 읽을 수 없음(%s): %v", name, err)
		}
		m := adrFrontmatter.FindSubmatch(raw)
		if m == nil {
			t.Errorf("%s: frontmatter 가 없다", name)
			continue
		}
		fields := map[string]string{}
		for _, p := range kv.FindAllStringSubmatch(string(m[1]), -1) {
			fields[p[1]] = strings.TrimSpace(p[2])
		}
		num := fields["number"]
		for len(num) < 4 {
			num = "0" + num
		}
		if num != name[:4] {
			t.Errorf("%s: 파일명과 number 가 다르다(number=%s)", name, fields["number"])
		}
		out[num] = adr{
			number: num,
			title:  fields["title"],
			date:   fields["date"],
			status: fields["status"],
			file:   name,
		}
	}
	if len(out) == 0 {
		t.Fatalf("ADR 을 하나도 읽지 못했다")
	}
	return out
}

// TestADRIndexMatchesFiles는 색인의 목록표를 ADR 파일들과 대조한다.
//
// AGENTS.md 가 색인을 개정 관계의 단일 진실원으로 선언했으므로 색인
// 자체는 사람이 쓴다. 다만 번호, 날짜, 상태, 링크 대상은 파일에서
// 도출되는 값이라 어긋나면 색인이 틀린 것이다.
//
// 제목은 대조하지 않는다. 색인의 제목은 지금 어휘로 다듬어 적는 것이
// 이 저장소의 관행이고(0002, 0006 이 그렇다), ADR 본문은 소급 수정하지
// 않으므로 둘이 다른 것이 정상이다.
func TestADRIndexMatchesFiles(t *testing.T) {
	files := readADRs(t)

	idxPath := filepath.Join(repoRoot, "docs", "decisions", "README.md")
	raw, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("ADR 색인을 읽을 수 없음: %v", err)
	}

	listed := map[string]adr{}
	for n, line := range strings.Split(string(raw), "\n") {
		m := adrIndexRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if prev, dup := listed[m[1]]; dup {
			t.Errorf("색인에 %s 가 두 번 있다(%s, %d줄)", m[1], prev.file, n+1)
		}
		listed[m[1]] = adr{number: m[1], file: m[2], date: strings.TrimSpace(m[4]), status: m[5]}
	}

	var problems []string
	for num, f := range files {
		l, ok := listed[num]
		if !ok {
			problems = append(problems, fmt.Sprintf("색인에 없음: %s (%s)", num, f.file))
			continue
		}
		if l.status != f.status {
			problems = append(problems, fmt.Sprintf("%s 상태 불일치: 색인=%s 파일=%s", num, l.status, f.status))
		}
		if l.date != f.date {
			problems = append(problems, fmt.Sprintf("%s 날짜 불일치: 색인=%s 파일=%s", num, l.date, f.date))
		}
		if l.file != f.file {
			problems = append(problems, fmt.Sprintf("%s 링크 대상 불일치: 색인=%s 파일=%s", num, l.file, f.file))
		}
	}
	for num, l := range listed {
		if _, ok := files[num]; !ok {
			problems = append(problems, fmt.Sprintf("색인에만 있음: %s (%s)", num, l.file))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("ADR 색인이 파일과 어긋난다. docs/decisions/README.md 를 고친다.\n  %s",
			strings.Join(problems, "\n  "))
	}
	t.Logf("ADR %d건, 색인 %d행 일치", len(files), len(listed))
}

// TestADRStatusVocabulary는 상태 어휘를 확인한다. ADR 0015 가 넷으로
// 못 박았다.
func TestADRStatusVocabulary(t *testing.T) {
	vocab := map[string]bool{"accepted": true, "amended": true, "superseded": true, "proposed": true}
	for num, a := range readADRs(t) {
		if !vocab[a.status] {
			t.Errorf("%s: 상태 어휘 위반 %q (%s)", num, a.status, a.file)
		}
	}
}
