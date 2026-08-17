// Package realdata 는 실운영 위키 사본에 engram 을 돌리는 스모크 테스트를
// 담는다.
//
// 골든 픽스처와 여정 테스트는 인공 데이터만 본다. 그 규모에서는 규칙이
// 규칙대로 도는지는 보이지만 모델이 현실과 맞는지는 안 보인다. 실제로
// 처음 실운영 위키에 돌려 보았을 때 인공 픽스처가 못 보던 위반이 즉시
// 나온 경험이 이 하니스의 이유다.
//
// 위반 건수는 단언하지 않는다. 실데이터의 위반 상당수는 upstream 위키
// 자체가 자기 명세에서 흘러내린 것이므로 정당하게 나온다. 건수를 고정하면
// upstream 이 문서를 하나 고칠 때마다 이 테스트가 깨진다. 측정값은
// t.Log 로만 남기고 단언은 불변식에만 건다.
//
// 공개 경계. upstream 은 비공개 사내 자료를 담는다. 이 파일에는 문서
// 경로, 슬러그, 제목, 본문을 박지 않는다. 질의는 일반 명사만 쓰고
// 슬러그 인자는 실행 시간에 사본에서 얻는다. 실패 메시지에 실경로가
// 실리는 것은 테스트 실행 로그로만 존재하고 저장소에 커밋되지 않는다.
package realdata

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// binaryPath 는 TestMain 이 빌드한 engram 바이너리 경로다. go 가 없으면
// 비어 있고 테스트는 skip 한다.
var binaryPath string

// repoRoot 는 모듈 루트다. 테스트는 harness/realdata 에서 돌므로 상대
// 경로로 잡는다.
var repoRoot = filepath.Join("..", "..")

// fixedNow 는 전 구간에 쓰는 기준 시각이다. 실제 시계를 쓰면 시간이
// 지나며 테스트가 깨진다.
const fixedNow = "2026-08-16T09:00:00Z"

// timeLimit 은 커맨드 하나의 상한이다. 실측이 0.1초 이하이므로 300배
// 여유다. 성능 회귀가 아니라 무한 루프와 지수 폭발을 잡는 상한이므로
// 조이지 않는다.
const timeLimit = 30 * time.Second

// TestMain 은 실제 바이너리를 만든다. 검증 대상이 종료 코드와 표준
// 출력이기 때문에 in-process 로 커맨드 계층을 부르지 않는다.
func TestMain(m *testing.M) {
	// 출력 언어를 못 박는다. 골든과 동등성 비교는 바이트 단위라
	// 개발자 환경의 ENGRAM_LANG 이 새어 들어오면 통째로 어긋난다.
	if err := os.Setenv("ENGRAM_LANG", "ko"); err != nil {
		panic(err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("go 가 PATH 에 없어 realdata 테스트를 건너뛴다")
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "engram-realdata-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "임시 디렉토리를 만들 수 없음: %v\n", err)
		os.Exit(1)
	}
	bin := filepath.Join(dir, "engram")
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

// TestRealdataSmoke 은 실운영 위키 사본에 대해 네 불변식을 단언한다.
// 커맨드가 죽지 않고, 조회가 위키를 바꾸지 않고, 판정이 결정론적이고,
// 규모에서 느려지지 않는 것. 위반 내용 자체는 단언하지 않고 로그로만
// 남긴다.
func TestRealdataSmoke(t *testing.T) {
	if binaryPath == "" {
		t.Skip("바이너리가 없어 실데이터 스모크를 돌지 못한다. go 가 PATH 에 있어야 한다")
	}
	root := expandTilde(os.Getenv("ENGRAM_UPSTREAM"))
	if root == "" {
		t.Skip("ENGRAM_UPSTREAM 이 없어 실데이터 스모크를 건너뛴다. ENGRAM_UPSTREAM=<llm-wiki 경로> 로 설정하면 돈다. CI 는 이 변수가 없으므로 항상 건너뛴다.")
	}
	if _, err := os.Stat(root); err != nil {
		t.Skipf("ENGRAM_UPSTREAM 경로를 확인할 수 없어 건너뛴다: %s: %v", root, err)
	}
	// 이 패키지도 바이너리를 exec 하므로 구현 소스를 읽어 테스트 결과
	// 캐시를 무력화한다. 자세한 근거는 harness/journey 의 touchSources
	// 주석에 있다. testlog 기록을 위해 테스트 함수 안에서 부른다.
	touchSources()

	// 사본을 만들고 init 으로 설정을 넣는다. upstream 을 직접 고치지
	// 않는다. index.md 는 이미 있으므로 init 이 덮어쓰지 않는다.
	wiki := t.TempDir()
	copyStages(t, root, wiki)
	r := exec2(t, "init", "--preset", "team", wiki)
	if r.code != 0 {
		t.Fatalf("사본 init 실패(종료 코드 %d)\n--- stdout ---\n%s--- stderr ---\n%s", r.code, r.stdout, r.stderr)
	}
	logDocCounts(t, wiki)

	// backlinks 인자는 소스에 식별자를 박지 않기 위해 실행 시간에 사본에서
	// 얻는다.
	slug := firstSlug(t, wiki)

	// (1) 죽지 않는다. lint 는 위반 있음이 정상이므로 종료 코드 1도
	// 받는다. resurface 는 쓰지 않는 --dry-run 으로만 부른다.
	// status, bridge, resurface, lint 의 JSON 출력은 아래 집계에 다시 쓴다.
	type spec struct {
		name    string
		args    []string
		exit1OK bool
	}
	specs := []spec{
		{"version", []string{"version"}, false},
		{"doctor", []string{"doctor", wiki}, false},
		{"status", []string{"status", "--json", wiki}, false},
		{"lint", []string{"lint", wiki}, true},
		{"reindex", []string{"reindex", wiki}, false},
		{"search", []string{"search", "--wiki", wiki, "문서"}, false},
		{"recall", []string{"recall", "--wiki", wiki, "위키"}, false},
		{"backlinks", []string{"backlinks", "--wiki", wiki, slug}, false},
		{"bridge", []string{"bridge", "--json", "--wiki", wiki}, false},
		{"digest", []string{"digest", "--wiki", wiki}, false},
		{"resurface", []string{"resurface", "--dry-run", "--json", "--wiki", wiki}, false},
	}
	ran := map[string]command{}
	for _, sp := range specs {
		r := exec2(t, sp.args...)
		bad := ""
		switch {
		case r.code == 1 && sp.exit1OK:
		case r.code == 0:
		default:
			bad = fmt.Sprintf("예상 밖의 종료 코드 %d", r.code)
		}
		if bad != "" {
			t.Fatalf("커맨드 %s: %s\n--- stdout ---\n%s--- stderr ---\n%s", sp.name, bad, r.stdout, r.stderr)
		}
		t.Logf("커맨드 %s: 종료 코드 %d, 소요 %s", sp.name, r.code, r.elapsed)
		ran[sp.name] = r
	}

	// (2) 조회 커맨드가 위키를 바꾸지 않는다. 경로와 내용 해시를 뜨고
	// 여덟 조회를 돌린 뒤 다시 떠서 완전히 같은지 본다. reindex 와
	// resurface 는 쓰는 것이 계약이므로 뺀다.
	before := snapshot(t, wiki)
	for _, args := range [][]string{
		{"lint", wiki},
		{"status", wiki},
		{"doctor", wiki},
		{"search", "--wiki", wiki, "문서"},
		{"recall", "--wiki", wiki, "위키"},
		{"backlinks", "--wiki", wiki, slug},
		{"bridge", "--wiki", wiki},
		{"digest", "--wiki", wiki},
	} {
		if r := exec2(t, args...); r.code > 1 || (r.code == 1 && args[0] != "lint") {
			t.Fatalf("조회 커맨드 %s 가 예상 밖의 종료 코드 %d 로 끝났다\n--- stdout ---\n%s--- stderr ---\n%s",
				args[0], r.code, r.stdout, r.stderr)
		}
	}
	added, removed, changed := diffSnapshots(before, snapshot(t, wiki))
	if len(added)+len(removed)+len(changed) > 0 {
		t.Fatalf("조회 커맨드가 위키를 바꿨다. 추가 %d, 삭제 %d, 변경 %d\n추가: %s\n삭제: %s\n변경: %s",
			len(added), len(removed), len(changed),
			strings.Join(added, ", "), strings.Join(removed, ", "), strings.Join(changed, ", "))
	}

	// (3) 판정이 결정론적이다. lint --json 을 두 번, search 를 같은 질의로
	// 두 번 돌려 출력 바이트가 같은지 본다.
	lintA := exec2(t, "lint", "--json", wiki)
	lintB := exec2(t, "lint", "--json", wiki)
	if lintA.stdout != lintB.stdout {
		t.Fatalf("lint --json 두 번의 출력이 다르다.\n--- 첫째 ---\n%s--- 둘째 ---\n%s", lintA.stdout, lintB.stdout)
	}
	searchA := exec2(t, "search", "--wiki", wiki, "문서")
	searchB := exec2(t, "search", "--wiki", wiki, "문서")
	if searchA.stdout != searchB.stdout {
		t.Fatalf("search 두 번의 순위가 다르다.\n--- 첫째 ---\n%s--- 둘째 ---\n%s", searchA.stdout, searchB.stdout)
	}

	// 아래는 측정값이다. 단언하지 않고 사람이 읽게 로그로만 남긴다.
	logLintSummary(t, lintA.stdout)
	logStatusSummary(t, ran["status"].stdout)
	logResurfaceSummary(t, ran["resurface"].stdout)
	logBridgeSummary(t, ran["bridge"].stdout)
}

// command 는 한 번의 실행 결과다.
type command struct {
	code           int
	stdout, stderr string
	elapsed        time.Duration
}

// exec2 는 바이너리를 실행한다. 종료 코드는 단언하지 않는다. 커맨드마다
// 걸린 시간을 재 상한을 넘으면 실패시키고, 표준 오류에 패닉 흔적이 있으면
// 실패시킨다. 패닉 흔적은 표준 오류에서만 본다. Go 런타임이 패닉
// 스택을 표준 오류로 내기 때문이고, 문서 본문을 인쇄하는 표준 출력의
// 단어를 재료로 거짓 실패를 만들지 않기 위해서다.
func exec2(t *testing.T, args ...string) command {
	t.Helper()
	full := append([]string{"--now", fixedNow}, args...)
	cmd := exec.Command(binaryPath, full...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	s := command{stdout: out.String(), stderr: errBuf.String(), elapsed: elapsed}
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("커맨드 실행 자체가 실패했다: %v\n인자: %s", err, strings.Join(args, " "))
		}
		s.code = ee.ExitCode()
	}
	if strings.Contains(s.stderr, "panic") || strings.Contains(s.stderr, "goroutine") {
		t.Fatalf("출력에 패닉 흔적이 있다. 인자: %s\n--- stdout ---\n%s--- stderr ---\n%s",
			strings.Join(args, " "), s.stdout, s.stderr)
	}
	if s.elapsed > timeLimit {
		t.Fatalf("커맨드가 %s 상한을 넘었다: %s. 인자: %s", timeLimit, s.elapsed, strings.Join(args, " "))
	}
	return s
}

// excludedDirs 는 사본에서 뺄 디렉토리 이름이다. 어느 깊이에서나 뺀다.
// raw-private 는 upstream 이 .gitignore 로 제외하는 sources 아래의 원본
// 비공개 계층이다. upstream 자신의 도구도 보지 않는 문서를 스모크가
// 보지 않게 한다. 이름으로 걸지만 이 경로에만 있는 이름이므로 목록에
// 둔다.
var excludedDirs = map[string]bool{
	".git": true, ".local": true, "private": true,
	".superpowers": true, ".codegraph": true, ".obsidian": true,
	"raw-private": true,
}

// copyStages 는 upstream 위키의 검사 대상 계층과 색인만 사본으로 만든다.
// 없는 계층은 건너뛴다. init 이 나머지 구조를 채운다.
func copyStages(t *testing.T, root, dst string) {
	t.Helper()
	for _, name := range []string{"inbox", "sources", "context", "archive"} {
		src := filepath.Join(root, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		copyTree(t, src, filepath.Join(dst, name))
	}
	if data, err := os.ReadFile(filepath.Join(root, "index.md")); err == nil {
		if err := os.WriteFile(filepath.Join(dst, "index.md"), data, 0o644); err != nil {
			t.Fatalf("색인 사본을 쓸 수 없음: %v", err)
		}
	}
}

// copyTree 는 디렉토리 트리를 파일 바이트 그대로 복사한다.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			if excludedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("위키 사본 만들기 실패: %v", err)
	}
}

// logDocCounts 는 단계별 문서 수와 총 문서 수를 로그로 남긴다.
func logDocCounts(t *testing.T, wiki string) {
	t.Helper()
	total := 0
	for _, stage := range []string{"inbox", "sources", "context", "archive"} {
		n := countMarkdown(filepath.Join(wiki, stage))
		total += n
		t.Logf("단계별 문서 수: %s %d", stage, n)
	}
	t.Logf("총 문서 수: %d (단계 합계, 색인 제외)", total)
}

// countMarkdown 은 디렉토리 아래 .md 파일 수를 센다. 숨김 디렉토리는
// 순회 규칙과 같게 뺀다.
func countMarkdown(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			n++
		}
		return nil
	})
	return n
}

// firstSlug 은 사본에서 문서 하나의 슬러그를 얻는다. context, sources,
// inbox 순으로 찾는다. backlinks 인자에 실제 슬러그를 쓰되 소스에
// 식별자를 박지 않기 위해 실행 시간에 얻는다.
func firstSlug(t *testing.T, wiki string) string {
	t.Helper()
	for _, stage := range []string{"context", "sources", "inbox"} {
		var found string
		filepath.WalkDir(filepath.Join(wiki, stage), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || found != "" {
				return nil
			}
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				found = strings.TrimSuffix(entry.Name(), ".md")
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	t.Fatal("사본에 문서가 없어 backlinks 인자를 얻을 수 없다")
	return ""
}

// snapshot 은 위키 전체 파일의 경로와 내용 해시를 모은다. 숨김 파일도
// 포함해 전부 뜬다. 조회 커맨드가 무엇이든 건드리지 않았는지 보는 게
// 목적이므로 제외를 두지 않는다.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		m[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("위키 스냅샷 실패: %v", err)
	}
	return m
}

// diffSnapshots 는 두 스냅샷의 차이를 추가, 삭제, 변경 목록으로 낸다.
func diffSnapshots(before, after map[string]string) (added, removed, changed []string) {
	for p := range after {
		if _, ok := before[p]; !ok {
			added = append(added, p)
		}
	}
	for p, h := range before {
		h2, ok := after[p]
		if !ok {
			removed = append(removed, p)
		} else if h != h2 {
			changed = append(changed, p)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	return added, removed, changed
}

// lintJSON 은 lint --json 출력에서 집계에 쓰는 부분만 담는다.
type lintJSON struct {
	Violations []struct {
		Severity string `json:"severity"`
		Rule     string `json:"rule"`
	} `json:"violations"`
	WikiFindings []struct {
		Rule string `json:"rule"`
	} `json:"wikiFindings"`
	Summary struct {
		Files  int `json:"files"`
		Error  int `json:"error"`
		Warn   int `json:"warn"`
		Reject int `json:"reject"`
	} `json:"summary"`
}

// logLintSummary 는 lint 위반을 등급별, 규칙별로 집계해 로그로 남긴다.
func logLintSummary(t *testing.T, out string) {
	t.Helper()
	var rep lintJSON
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("lint --json 출력을 파싱할 수 없다: %v\n%s", err, out)
	}
	s := rep.Summary
	t.Logf("lint: 파일 %d개, error %d, warn %d, reject %d, 위키 단위 진단 %d건",
		s.Files, s.Error, s.Warn, s.Reject, len(rep.WikiFindings))
	byRule := map[string]int{}
	for _, v := range rep.Violations {
		byRule[v.Rule+" ("+v.Severity+")"]++
	}
	keys := make([]string, 0, len(byRule))
	for k := range byRule {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("  규칙별 위반: %s %d건", k, byRule[k])
	}
}

// logStatusSummary 는 status --json 에서 위키링크, 고아, 승급 가능 수를
// 로그로 남긴다.
func logStatusSummary(t *testing.T, out string) {
	t.Helper()
	var rep struct {
		Links   int `json:"links"`
		Orphans int `json:"orphans"`
		Backlog struct {
			Promotable int `json:"promotable"`
			Stale      int `json:"stale"`
		} `json:"backlog"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("status --json 출력을 파싱할 수 없다: %v\n%s", err, out)
	}
	t.Logf("status: 위키링크 %d개, 고아 %d개, 승급 가능 %d개, stale context %d개",
		rep.Links, rep.Orphans, rep.Backlog.Promotable, rep.Backlog.Stale)
}

// logResurfaceSummary 는 resurface --dry-run 의 후보 수와 날짜 없음
// 제외 수를 로그로 남긴다.
func logResurfaceSummary(t *testing.T, out string) {
	t.Helper()
	var rep struct {
		SkippedNoDate int `json:"skippedNoDate"`
		Candidates    []struct {
			Slug string `json:"slug"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("resurface --json 출력을 파싱할 수 없다: %v\n%s", err, out)
	}
	t.Logf("resurface --dry-run: 후보 %d건, 날짜 없음 제외 %d건", len(rep.Candidates), rep.SkippedNoDate)
}

// logBridgeSummary 는 bridge 가 낸 쌍의 수를 로그로 남긴다.
func logBridgeSummary(t *testing.T, out string) {
	t.Helper()
	var rep struct {
		Pairs []struct {
			A string `json:"a"`
			B string `json:"b"`
		} `json:"pairs"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("bridge --json 출력을 파싱할 수 없다: %v\n%s", err, out)
	}
	t.Logf("bridge: 쌍 %d건", len(rep.Pairs))
}

// expandTilde 는 경로가 ~ 로 시작하면 홈 디렉토리로 펼친다. harness/parity
// 와 같은 방식이다.
func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// touchSources 는 구현 소스를 전부 읽어 go 의 테스트 결과 캐시를
// 무력화한다. 바이너리를 exec 하는 패키지는 소스 의존이 없어 캐시가
// 맞으면 internal/ 을 고쳐도 결과를 재사용한다. 패키지 디렉토리 밖
// 파일을 읽으면 그 사실이 캐시 키에 들어간다. testlog 기록을 위해
// 테스트 함수 안에서 부른다.
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
