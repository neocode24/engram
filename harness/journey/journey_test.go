// Package journey는 engram 의 전체 사용 여정을 실제 바이너리로 한 번
// 돌아보는 통합 테스트를 담는다.
//
// 단위 테스트는 커맨드를 하나씩 본다. 과거에 난 결함 다섯 건은 전부
// 커맨드 사이에서 났다. "도구가 만든 산출물을 도구 자신의 검사가
// 거절한다" 계열이었다. 이 테스트가 지키는 불변식은 하나다.
// 도구가 만든 위키는 어느 시점에 lint 를 돌려도 error 가 0이다.
//
// 커맨드 계층을 in-process 로 부르지 않고 빌드한 바이너리를 exec 한다.
// 검증 대상이 종료 코드와 표준 출력이기 때문이다.
package journey

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// binaryPath는 TestMain 이 빌드한 engram 바이너리 경로다. go 가 없으면
// 비어 있고 테스트는 skip 한다.
var binaryPath string

// repoRoot는 모듈 루트다. 테스트는 harness/journey 에서 돌므로 상대
// 경로로 잡는다. 절대 경로를 쓰지 않는 이유는 골든 러너와 같다.
var repoRoot = filepath.Join("..", "..")

// TestMain은 실제 바이너리를 만든다. 커맨드 사이의 결함을 보려면 진짜
// 실행 파일이 필요하다. go build 실패는 테스트 실패다. 빌드가 깨졌는데
// 조용히 넘어가면 이 하니스는 아무것도 지키지 못한다.
func TestMain(m *testing.M) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("go 가 PATH 에 없어 journey 테스트를 건너뛴다")
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "engram-journey-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "임시 디렉토리를 만들 수 없음: %v\n", err)
		os.Exit(1)
	}
	// Windows 는 확장자 없는 파일을 실행하지 못한다. .exe 를 붙이지 않으면
	// exec 이 "executable file not found" 로 죽는다.
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

// touchSources는 구현 소스를 전부 읽어 go 의 테스트 결과 캐시를 무력화한다.
//
// 이 패키지는 바이너리를 exec 하므로 검사 대상 코드에 소스 의존이 없다.
// 블랭크 import 로 의존을 만들어도 부족하다. go 의 테스트 결과 캐시는
// 링크된 패키지가 바뀌었다고 결과를 버리지 않는다. 실측으로 확인했다.
// internal/config 를 고친 뒤에도 ok (cached) 가 나왔다.
//
// 반면 패키지 디렉토리 밖 파일을 읽으면 그 사실이 캐시 키에 들어가고, go 는
// 그런 실행의 결과를 재사용하지 않는다. 여기서 소스를 읽는 이유가 그것이다.
// 내용은 쓰지 않는다. 읽었다는 사실만 필요하다.
//
// 반드시 테스트 함수 안에서 부른다. 파일 접근을 기록하는 testlog 는
// m.Run 이 켜므로 TestMain 에서 읽으면 기록되지 않고 캐시가 그대로 맞는다.
//
// 캐시가 맞으면 이 하니스는 아무것도 지키지 못한다. 커맨드 사이의 결함을
// 잡으려고 만든 테스트가 소스를 고쳐도 안 도는 상태가 가장 나쁘다.
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

// TestJourney는 init 부터 reindex 까지 사용 여정 전 구간을 순서대로 돈다.
// 각 변경 단계 뒤에 lint 를 돌려 error 와 reject 가 0인지 단언한다.
// 게이트는 우회하지 않는다. 링크가 부족하면 본문에 실제 위키링크를
// 넣어 통과시킨다.
func TestJourney(t *testing.T) {
	if binaryPath == "" {
		t.Skip("바이너리가 없어 여정을 돌지 못한다. go 가 PATH 에 있어야 한다")
	}
	touchSources()
	wiki := t.TempDir()
	files := t.TempDir()
	w := walker{bin: binaryPath, wiki: wiki, now: "2026-05-10T09:00:00Z"}

	// 1. init. 만들어진 위키가 곧 첫 검사 대상이다.
	w.mustOK(t, w.run("init", "init", wiki))
	w.assertLintClean(t, "init")

	// 2. capture. inbox 문서 하나를 넣는다.
	w.mustOK(t, w.run("capture",
		"capture", "--wiki", wiki, "--title", "현장 메모", "--slug", "field-memo",
		"회의 중 적은 메모"))
	w.assertLintClean(t, "capture")

	// 3. source. 원본 문서를 sources 에 넣는다.
	w.mustOK(t, w.run("source",
		"source", "--wiki", wiki, "--title", "기술 강연 원본", "--slug", "tech-talk",
		"--created", "2026-04-01", "--ref", "https://example.com/talk", "강연 필사 원본"))
	w.assertLintClean(t, "source")

	// 4. new. context 문서 둘을 서로 위키링크로 잇는다. doc-a 를 만드는
	// 시점에 doc-b 는 아직 없다. 없는 슬러그는 경고만 나고 만들 수 있으므로
	// 이 순서로 상호 연결을 만든다.
	w.mustOK(t, w.run("new 문서A",
		"new", "문서 A", "--wiki", wiki, "--slug", "doc-a",
		"--related", "doc-b", "--related", "index"))
	w.assertLintClean(t, "new 문서A")
	w.mustOK(t, w.run("new 문서B",
		"new", "문서 B", "--wiki", wiki, "--slug", "doc-b",
		"--related", "doc-a", "--related", "index"))
	w.assertLintClean(t, "new 문서B")

	// 5. promote. inbox 문서를 context 로 올린다. capture 문서에는 위키링크가
	// 없으므로 게이트에 걸린다. 강제 플래그나 임계값 낮추기 대신 본문에
	// 실제 위키링크를 넣어 통과시킨다.
	inboxDoc := "inbox/2026-05-10-field-memo.md"
	bodyFile := filepath.Join(files, "field-memo-body.md")
	writeFile(t, bodyFile, "# 현장 메모\n\n회의 중 적은 메모다. 정리하면 [[doc-a]] 와 [[doc-b]] 로 나뉜다.\n")
	w.mustOK(t, w.run("update 게이트 채우기",
		"update", "--wiki", wiki, "--body-from", bodyFile, inboxDoc))
	w.assertLintClean(t, "update 게이트 채우기")
	w.mustOK(t, w.run("promote inbox",
		"promote", "--wiki", wiki, "--type", "meeting-summary", inboxDoc))
	w.assertLintClean(t, "promote inbox")
	// inbox 문서는 이동이므로 원본이 남지 않고 context 에 날짜 접두사 없는
	// 이름으로 도착한다.
	assertFileExists(t, wiki, "context/field-memo.md")
	assertFileAbsent(t, wiki, inboxDoc)
	// 승급 문서에 created 가 살아 있는지 그 자리에서 본다. promote 가 날짜
	// 접두사를 떼면서 created 까지 사라지면 resurface 와 digest 가 이 문서를
	// 대상에서 뺀다. 이 단계에서 잡으면 문서가 나중에 demote 나 archive 로
	// 옮겨도 놓치지 않는다.
	assertFileContains(t, wiki, "context/field-memo.md", "created: 2026-05-10")

	// 6. promote. sources 문서에서 파생을 만든다. 원본은 남는다.
	w.mustOK(t, w.run("promote sources",
		"promote", "--wiki", wiki, "--type", "procedure",
		"--related", "doc-a", "--related", "doc-b", "sources/2026-04-01-tech-talk.md"))
	w.assertLintClean(t, "promote sources")
	assertFileExists(t, wiki, "context/tech-talk.md")
	assertFileExists(t, wiki, "sources/2026-04-01-tech-talk.md")
	assertFileContains(t, wiki, "context/tech-talk.md", "created: 2026-04-01")

	// 7. update. 프론트매터와 본문을 함께 바꾼다.
	bodyFile2 := filepath.Join(files, "doc-a-body.md")
	writeFile(t, bodyFile2, "# 문서 A\n\n갱신한 본문이다. [[doc-b]] 와 짝을 이룬다.\n")
	w.mustOK(t, w.run("update",
		"update", "--wiki", wiki, "--set", "topics=testing",
		"--body-from", bodyFile2, "context/doc-a.md"))
	w.assertLintClean(t, "update")

	// 8. mv. 문서 하나의 이름을 바꾸고 들어오는 링크가 따라오는지 본다.
	w.mustOK(t, w.run("mv", "mv", "--wiki", wiki, "doc-b", "doc-b-renamed"))
	w.assertLintClean(t, "mv")
	assertFileExists(t, wiki, "context/doc-b-renamed.md")
	assertFileAbsent(t, wiki, "context/doc-b.md")
	// 관계 필드(doc-a)와 본문 링크(field-memo) 모두 새 슬러그를 가리켜야 한다.
	assertFileContains(t, wiki, "context/doc-a.md", "[[doc-b-renamed]]")
	assertFileContains(t, wiki, "context/field-memo.md", "[[doc-b-renamed]]")

	// 9. demote. context 문서 하나를 inbox 로 내린다.
	w.mustOK(t, w.run("demote", "demote", "--wiki", wiki, "context/field-memo.md"))
	w.assertLintClean(t, "demote")
	assertFileAbsent(t, wiki, "context/field-memo.md")
	demoted, err := filepath.Glob(filepath.Join(wiki, "inbox", "*field-memo*.md"))
	if err != nil || len(demoted) != 1 {
		t.Fatalf("단계 demote: inbox 에 되돌린 문서가 정확히 하나여야 한다. 찾은 수 %d, 오류 %v", len(demoted), err)
	}

	// 10. archive. context 문서 하나를 archive 로 보낸다.
	w.mustOK(t, w.run("archive", "archive", "--wiki", wiki, "context/doc-a.md"))
	w.assertLintClean(t, "archive")
	assertFileExists(t, wiki, "archive/doc-a.md")

	// 11. reindex. 검색 색인을 만든다.
	w.mustOK(t, w.run("reindex", "reindex", wiki))
	w.assertLintClean(t, "reindex")
	assertFileExists(t, wiki, ".engram/index.json")

	// 12. export. 도구가 만든 위키를 도구 자신의 반출 커맨드에 태운다.
	// 여정 끝 상태에서 context 에 남은 것은 doc-b-renamed 와 tech-talk 이고
	// field-memo 는 demote 로 inbox 에, doc-a 는 archive 에 있다. 번들에
	// inbox 문서가 섞이면 검수 경계가 반출에서 뚫린 것이다(ADR 0044, 0046).
	bundle := filepath.Join(files, "bundle")
	dict := filepath.Join(files, "repl.txt")
	writeFile(t, dict, "# 반출 사전\ntech-talk==>lecture\n")
	w.mustOK(t, w.run("export",
		"export", "--wiki", wiki, "--out", bundle, "--replacements", dict))
	w.assertLintClean(t, "export")
	assertBundle(t, bundle, []string{
		"context/doc-b-renamed.md",
		"context/lecture.md",
		"index.md",
	})
	// 치환은 본문에도 걸린다. 파일명만 바뀌고 본문에 원문이 남으면
	// 익명화가 반쪽이다.
	assertFileAbsent(t, bundle, "context/tech-talk.md")
	assertFileContains(t, bundle, "context/lecture.md", "lecture")

	// 승급 문서가 재발견 대상에 들어가는지 본다. 기준 시각을 stale_days
	// 기본값 90일보다 훨씬 뒤로 옮긴다. 승급 문서에 날짜가 없으면 후보가
	// 비거나 skippedNoDate 가 늘어난다.
	future := walker{bin: binaryPath, wiki: wiki, now: "2026-12-01T00:00:00Z"}
	rs := future.run("resurface 미래 시각", "resurface", "--json", "--wiki", wiki)
	future.mustOK(t, rs)
	var rep resurfaceReport
	if err := json.Unmarshal([]byte(rs.stdout), &rep); err != nil {
		future.fail(t, rs, fmt.Sprintf("resurface --json 출력을 파싱할 수 없다: %v", err))
	}
	// 후보 목록에 승급으로 생긴 문서가 실제로 들어 있는지 슬러그로 못 박아
	// 본다. new 로 만든 문서만 후보에 남아도 len(Candidates) > 0 은
	// 성립하므로 그 단언은 승급 문서의 날짜 소실을 못 잡는다. 여정 끝에
	// context 에 남는 승급 문서는 tech-talk 하나다.
	found := false
	var slugs strings.Builder
	for _, c := range rep.Candidates {
		fmt.Fprintf(&slugs, "%s (%s) ", c.Slug, c.Path)
		if c.Slug == "tech-talk" {
			found = true
		}
	}
	if !found || rep.SkippedNoDate != 0 {
		t.Fatalf("단계 resurface 미래 시각: 승급 문서 tech-talk 이 후보에 없다. 후보 %d건, 날짜 없음 제외 %d건.\n후보 목록: %s\n%s",
			len(rep.Candidates), rep.SkippedNoDate, slugs.String(), rs.stdout)
	}

	// 조회 커맨드가 여정 끝 상태의 실제 위키에서 죽지 않는지 본다.
	// 출력 내용은 단위 테스트가 이미 본다. 여기서 보는 것은 생존이다.
	for _, q := range [][]string{
		{"status", wiki},
		{"search", "--wiki", wiki, "문서"},
		{"recall", "--wiki", wiki, "문서"},
		{"backlinks", "--wiki", wiki, "doc-b-renamed"},
		{"bridge", "--wiki", wiki},
		{"digest", "--wiki", wiki},
		{"doctor", wiki},
		{"version"},
	} {
		w.mustOK(t, w.run("조회 "+q[0], q...))
	}
}

// walker는 여정 전체에서 같은 바이너리, 위키, 기준 시각을 들고 다닌다.
type walker struct {
	bin  string
	wiki string
	now  string
}

// step은 한 커맨드 실행의 종료 코드와 출력이다.
type step struct {
	name           string
	code           int
	stdout, stderr string
}

// run은 engram 커맨드를 실제 바이너리로 실행한다. --now 는 전 구간에서
// 고정한다. 실제 시계를 쓰면 시간이 지나며 테스트가 깨진다.
func (w walker) run(name string, args ...string) step {
	full := append([]string{"--now", w.now}, args...)
	cmd := exec.Command(w.bin, full...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	s := step{name: name, stdout: out.String(), stderr: errBuf.String()}
	if err == nil {
		return s
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		s.code = ee.ExitCode()
		return s
	}
	s.code = -1
	s.stderr += "\n실행 자체가 실패했다: " + err.Error()
	return s
}

// mustOK는 단계가 종료 코드 0으로 끝났는지 본다.
func (w walker) mustOK(t *testing.T, s step) {
	t.Helper()
	if s.code != 0 {
		w.fail(t, s, "종료 코드 0이 필요하다")
	}
}

// fail은 단계 이름과 그 커맨드의 실제 출력 전문을 담아 실패를 낸다.
// 진단이 안 되는 통합 테스트는 쓸모가 없다.
func (w walker) fail(t *testing.T, s step, why string) {
	t.Helper()
	t.Fatalf("단계 %s 실패: %s\n종료 코드: %d\n--- stdout ---\n%s--- stderr ---\n%s",
		s.name, why, s.code, s.stdout, s.stderr)
}

// lintReport는 lint --json 출력에서 이 테스트가 보는 부분이다. 위반
// 전문은 실패 안내에 그대로 실어 판단에 필요한 문장을 전부 보게 한다.
type lintReport struct {
	Violations []struct {
		Severity string `json:"severity"`
		Rule     string `json:"rule"`
		Path     string `json:"path"`
		Message  string `json:"message"`
	} `json:"violations"`
	Summary struct {
		Error  int `json:"error"`
		Warn   int `json:"warn"`
		Reject int `json:"reject"`
	} `json:"summary"`
}

// assertLintClean은 위키에 lint --json 을 돌려 error 와 reject 가 0인지
// 단언한다. 도구가 만든 산출물을 도구 자신의 검사가 거절하는 순간을 잡는
// 것이 이 테스트의 본체다. reject 도 보는 이유는 gate.min-wikilinks 가
// 승급을 막는 등급이라 여정 어느 시점에 나도 같은 결함이기 때문이다.
// 위반 목록 전문을 실은 실패를 낸다.
func (w walker) assertLintClean(t *testing.T, after string) {
	t.Helper()
	s := w.run("lint ("+after+" 뒤)", "lint", "--json", w.wiki)
	var rep lintReport
	if err := json.Unmarshal([]byte(s.stdout), &rep); err != nil {
		w.fail(t, s, fmt.Sprintf("lint --json 출력을 파싱할 수 없다: %v", err))
	}
	if rep.Summary.Error == 0 && rep.Summary.Reject == 0 {
		return
	}
	var b strings.Builder
	for _, v := range rep.Violations {
		fmt.Fprintf(&b, "  [%s] %s %s: %s\n", v.Severity, v.Rule, v.Path, v.Message)
	}
	t.Fatalf("단계 %s 뒤 lint 가 위키를 거절했다: error %d건, reject %d건 (경고 %d건)\n위반 목록:\n%s\nlint stdout:\n%s",
		after, rep.Summary.Error, rep.Summary.Reject, rep.Summary.Warn, b.String(), s.stdout)
}

// resurfaceReport는 resurface --json 출력에서 이 테스트가 보는 부분이다.
type resurfaceReport struct {
	SkippedNoDate int `json:"skippedNoDate"`
	Candidates    []struct {
		Slug string `json:"slug"`
		Path string `json:"path"`
	} `json:"candidates"`
}

// writeFile은 여정에서 본문으로 넣을 파일을 쓴다.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("본문 파일을 쓸 수 없음: %s: %v", path, err)
	}
}

// assertFileExists는 위키 안 상대 경로에 파일이 있는지 본다.
func assertFileExists(t *testing.T, wiki, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(wiki, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("파일이 있어야 한다: %s: %v", rel, err)
	}
}

// assertFileAbsent는 위키 안 상대 경로에 파일이 없는지 본다.
func assertFileAbsent(t *testing.T, wiki, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(wiki, filepath.FromSlash(rel))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("파일이 없어야 한다: %s (stat 오류: %v)", rel, err)
	}
}

// assertBundle은 반출 번들의 파일 목록이 기대와 정확히 같은지 본다.
// 부분 집합이 아니라 정확히 같은지 보는 이유는, 나가면 안 되는 문서가
// 하나 더 들었을 때를 잡는 것이 이 단언의 목적이기 때문이다.
func assertBundle(t *testing.T, dir string, want []string) {
	t.Helper()
	var got []string
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		got = append(got, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("번들을 읽을 수 없음: %s: %v", dir, err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("번들 내용이 다르다.\n기대: %v\n실제: %v", want, got)
	}
}

// assertFileContains는 위키 안 파일의 내용에 문자열이 있는지 본다.
func assertFileContains(t *testing.T, wiki, rel, want string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(wiki, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("파일을 읽을 수 없음: %s: %v", rel, err)
	}
	if !strings.Contains(string(raw), want) {
		t.Fatalf("파일 %s 가 다음 문자열을 담고 있어야 한다: %s\n실제 내용:\n%s", rel, want, raw)
	}
}
