// Package examples는 저장소의 examples/ 데모 위키를 생성하고 검증한다.
//
// examples/README.md 가 세 가지를 계약으로 걸었다. 생성물이라는 것,
// 손으로 고치지 않는다는 것, 재생성 결과가 커밋된 내용과 다르면 회귀로
// 본다는 것이다. 그 계약을 지키는 장치가 없어서 0.1 이후 placeholder 로
// 남아 있었다.
//
// 생성 주체는 init 하나가 아니라 커맨드 시퀀스다. init 만으로는 파일
// 셋짜리 빈 위키가 나와서 승급도 게이트도 링크도 보이지 않는다. 시퀀스로
// 넓혀도 계약은 그대로 성립한다. --now 를 고정하므로 결정론이다.
//
// go test ./harness/examples -update 로 재생성한다.
package examples

import (
	"bytes"
	"embed"
	"errors"
	"flag"
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

// update는 examples/ 를 재생성할 때 켜는 플래그다.
var update = flag.Bool("update", false, "examples/ 데모 위키를 재생성한다")

// binaryPath는 TestMain 이 빌드한 바이너리다. go 가 없으면 비어 있다.
var binaryPath string

// repoRoot는 모듈 루트다. 테스트가 harness/examples 에서 돌므로 상대
// 경로로 잡는다.
var repoRoot = filepath.Join("..", "..")

// demoDir는 데모 위키가 커밋되는 자리다.
var demoDir = filepath.Join(repoRoot, "examples", "personal")

// fixedNow는 생성 시각이다. 고정하지 않으면 재생성마다 날짜가 바뀌어
// 회귀 비교가 성립하지 않는다.
const fixedNow = "2026-03-02T09:00:00Z"

func TestMain(m *testing.M) {
	// 출력 언어를 못 박는다. 골든과 동등성 비교는 바이트 단위라
	// 개발자 환경의 ENGRAM_LANG 이 새어 들어오면 통째로 어긋난다.
	if err := os.Setenv("ENGRAM_LANG", "ko"); err != nil {
		panic(err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("go 가 PATH 에 없어 examples 테스트를 건너뛴다")
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "engram-examples-")
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

// touchSources는 구현 소스를 읽어 go 의 테스트 결과 캐시를 무력화한다.
// 이 패키지는 바이너리를 exec 하므로 소스 의존이 없다. journey 하니스와
// 같은 이유이고 같은 방법이다.
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

// TestExamplesAreReproducible는 데모 위키를 새로 만들어 커밋된 것과
// 대조한다. 어긋나면 회귀다. -update 를 주면 커밋본을 갱신한다.
func TestExamplesAreReproducible(t *testing.T) {
	if binaryPath == "" {
		t.Skip("바이너리가 없어 데모 위키를 만들지 못한다. go 가 PATH 에 있어야 한다")
	}
	touchSources()

	built := filepath.Join(t.TempDir(), "personal")
	buildDemo(t, built)

	if *update {
		if err := os.RemoveAll(demoDir); err != nil {
			t.Fatalf("이전 데모 위키를 지울 수 없음: %v", err)
		}
		if err := copyTree(built, demoDir); err != nil {
			t.Fatalf("데모 위키를 옮길 수 없음: %v", err)
		}
		t.Logf("examples/personal 을 재생성했다")
		return
	}

	want, err := readTree(demoDir)
	if err != nil {
		t.Fatalf("커밋된 데모 위키를 읽을 수 없음(%s). go test ./harness/examples -update 로 생성한다: %v", demoDir, err)
	}
	got, err := readTree(built)
	if err != nil {
		t.Fatalf("만든 데모 위키를 읽을 수 없음: %v", err)
	}
	diffTrees(t, want, got)
}

// buildDemo는 데모 위키를 만드는 커맨드 시퀀스다. 이 순서가 곧 교육
// 자료의 내용이다. 게이트를 우회하지 않는다.
func buildDemo(t *testing.T, dir string) {
	t.Helper()
	r := runner{t: t}

	// 1. 빈 위키. personal 프리셋이 기본값이다.
	r.run("init", dir, "--preset", "personal")

	// 2. context 문서 둘을 서로 잇는다. 게이트가 요구하는 링크를 관계
	// 필드로 채운다. 먼저 만드는 쪽은 상대가 아직 없으므로 경고가 난다.
	r.run("new", "지식 승급 파이프라인", "--wiki", dir, "--slug", "promotion-pipeline",
		"--related", "wikilink-graph", "--related", "index")
	r.run("new", "위키링크로 엮는 지식 그래프", "--wiki", dir, "--slug", "wikilink-graph",
		"--related", "promotion-pipeline", "--related", "index")

	// 3. inbox 에 미처리 메모 둘. 하나는 승급하고 하나는 남긴다.
	// 밀린 것이 0인 위키는 status 가 무엇을 보는지 보여주지 못한다.
	r.run("capture", "--wiki", dir, "--title", "읽을거리 메모", "--slug", "reading-note",
		"주간 회고에서 나온 이야기. 정리 전 상태다.")
	r.run("capture", "--wiki", dir, "--title", "강연 메모", "--slug", "talk-note",
		"강연을 들으며 적은 조각. 아직 지식이 아니다.")

	// 4. sources 에 원본 하나. 원본 보존 계층을 보여준다.
	r.run("source", "--wiki", dir, "--title", "지식관리 입문 자료", "--slug", "km-primer",
		"--created", "2026-02-10", "--ref", "https://example.com/km-primer",
		"외부 자료의 원문 발췌. 이 계층은 고치지 않는다.")

	// 5. inbox 문서 하나를 승급시킨다. 게이트가 위키링크를 요구하므로
	// 본문에 실제 링크를 넣어 통과시킨다. 강제 플래그를 쓰지 않는다.
	body := filepath.Join(t.TempDir(), "reading-note.md")
	writeFile(t, body, "# 읽을거리 메모\n\n"+
		"주간 회고에서 나온 이야기를 정리했다. 승급의 기준은 [[promotion-pipeline]] 에 있고, "+
		"문서를 어떻게 잇는지는 [[wikilink-graph]] 에 있다.\n")
	r.run("update", "--wiki", dir, "--body-from", body, "inbox/2026-03-02-reading-note.md")
	r.run("promote", "--wiki", dir, "--type", "concept", "inbox/2026-03-02-reading-note.md")

	// 6. sources 문서에서 파생을 만든다. 원본은 그 자리에 남는다.
	r.run("promote", "--wiki", dir, "--type", "concept",
		"--related", "promotion-pipeline", "--related", "wikilink-graph",
		"sources/2026-02-10-km-primer.md")

	// 7. 본문을 채운다. new 가 만드는 것은 빈 템플릿 헤딩이라 그대로 두면
	// search 도 recall 도 bridge 도 보여줄 것이 없다. 교육 자료는 읽을
	// 내용이 있어야 한다.
	tmp := t.TempDir()
	fill := func(rel, content string) {
		f := filepath.Join(tmp, strings.ReplaceAll(rel, "/", "_"))
		writeFile(t, f, content)
		r.run("update", "--wiki", dir, "--body-from", f, rel)
	}

	fill("index.md", docBody(t, "index.md"))
	fill("context/promotion-pipeline.md", docBody(t, "promotion-pipeline.md"))
	fill("context/wikilink-graph.md", docBody(t, "wikilink-graph.md"))
	fill("context/km-primer.md", docBody(t, "km-primer.md"))

	// 색인은 만들지 않는다. .engram/ 은 캐시이고 gitignore 대상이라
	// 커밋되지 않으므로 여기서 만들어도 사라진다. 데모를 받은 사람이
	// engram reindex 를 한 번 돌린다. examples/README.md 가 안내한다.

	// 만든 위키가 자기 검사를 통과하는지 그 자리에서 본다. 교육 자료가
	// lint 를 통과하지 못하면 그것부터 가르치는 꼴이 된다.
	r.mustLintClean(dir)

	// 캐시는 커밋 대상이 아니다. init 이 만든 .gitignore 가 이미 제외하지만
	// 비교 대상에서도 빼야 재생성 결과가 흔들리지 않는다.
	if err := os.RemoveAll(filepath.Join(dir, ".engram")); err != nil {
		t.Fatalf(".engram 캐시를 지울 수 없음: %v", err)
	}
}

// runner는 바이너리를 고정 시각으로 실행한다.
type runner struct{ t *testing.T }

// run은 커맨드 하나를 돌리고 실패하면 출력 전문과 함께 멈춘다.
func (r runner) run(args ...string) {
	r.t.Helper()
	full := append([]string{"--now", fixedNow}, args...)
	cmd := exec.Command(binaryPath, full...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("데모 생성 단계 실패: engram %s\n오류: %v\n--- stdout ---\n%s--- stderr ---\n%s",
			strings.Join(args, " "), err, out.String(), errBuf.String())
	}
}

// mustLintClean은 데모 위키에 lint 를 돌려 거절이 없는지 본다.
func (r runner) mustLintClean(dir string) {
	r.t.Helper()
	cmd := exec.Command(binaryPath, "--now", fixedNow, "lint", dir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("데모 위키가 lint 를 통과하지 못한다:\n%s", out.String())
	}
}

// readTree는 디렉토리의 모든 파일을 상대 경로에서 내용으로 읽는다.
func readTree(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		// CRLF 로 체크아웃된 기계에서도 같은 결과가 나와야 한다.
		out[filepath.ToSlash(rel)] = strings.ReplaceAll(string(raw), "\r\n", "\n")
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// copyTree는 디렉토리를 통째로 옮긴다. -update 경로에서만 쓴다.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
}

// diffTrees는 두 트리의 차이를 전부 보고한다. 첫 차이에서 멈추면
// 재생성이 몇 번이나 필요해진다.
func diffTrees(t *testing.T, want, got map[string]string) {
	t.Helper()
	var missing, extra, changed []string
	for path, w := range want {
		g, ok := got[path]
		if !ok {
			missing = append(missing, path)
			continue
		}
		if w != g {
			changed = append(changed, path)
		}
	}
	for path := range got {
		if _, ok := want[path]; !ok {
			extra = append(extra, path)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(changed)
	if len(missing) == 0 && len(extra) == 0 && len(changed) == 0 {
		return
	}
	var b strings.Builder
	for _, p := range missing {
		fmt.Fprintf(&b, "  없어짐: %s\n", p)
	}
	for _, p := range extra {
		fmt.Fprintf(&b, "  새로 생김: %s\n", p)
	}
	for _, p := range changed {
		fmt.Fprintf(&b, "  내용 바뀜: %s\n--- 커밋본 ---\n%s--- 재생성 ---\n%s\n", p, want[p], got[p])
	}
	t.Fatalf("examples/personal 이 재생성 결과와 다르다. 의도한 변경이면 go test ./harness/examples -update 로 갱신한다.\n%s", b.String())
}

// writeFile은 생성 중 필요한 임시 본문 파일을 쓴다.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("본문 파일을 쓸 수 없음: %s: %v", path, err)
	}
}

// ensure는 errors 임포트를 쓰는 자리다. 데모 디렉토리가 아예 없을 때와
// 읽기 실패를 구분해 안내를 다르게 낸다.
func init() {
	if _, err := os.Stat(demoDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "데모 디렉토리 확인 중 오류: %v\n", err)
	}
}

// bodies는 데모 문서의 본문이다. Go 문자열 상수 대신 파일로 둔다.
// 문서가 스물을 넘으면 상수로는 검토도 유지도 안 된다. 마크다운
// 파일이면 그대로 읽고 고칠 수 있다.
//
// 이 저장소의 공개 산출물이므로 한국어 산문에 em dash, 화살표,
// 가운뎃점, 말줄임표를 쓰지 않는다(AGENTS.md).
//
//go:embed bodies
var bodies embed.FS

// docBody는 bodies/ 아래의 본문 하나를 읽는다. 없으면 그 자리에서 멈춘다.
// 시퀀스가 가리키는 본문이 없는 것은 오타이지 건너뛸 일이 아니다.
func docBody(t *testing.T, name string) string {
	t.Helper()
	b, err := bodies.ReadFile("bodies/" + name)
	if err != nil {
		t.Fatalf("본문을 읽을 수 없음: %v", err)
	}
	return string(b)
}
