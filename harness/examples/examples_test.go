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
	// 적체가 0인 위키는 status 가 무엇을 보는지 보여주지 못한다.
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

	fill("index.md", indexBody)
	fill("context/promotion-pipeline.md", pipelineBody)
	fill("context/wikilink-graph.md", graphBody)
	fill("context/km-primer.md", primerBody)

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

// 아래는 데모 문서의 본문이다. 조직 맥락 없이 지식관리 일반 개념만
// 다룬다. 이 저장소의 공개 산출물이므로 한국어 산문에 em dash, 화살표,
// 가운뎃점, 말줄임표를 쓰지 않는다(AGENTS.md).

const indexBody = `# 데모 위키

engram 이 만드는 위키의 모습을 보여주는 예제다. ` + "`engram init --preset personal`" + ` 으로
시작해 몇 개의 커맨드를 순서대로 돌린 결과이며, 손으로 고친 곳이 없다.

무엇부터 볼지는 [[promotion-pipeline]] 이 정리해 두었다. 문서끼리 어떻게
엮이는지는 [[wikilink-graph]] 에 있다.

## 지금 이 위키의 상태

| 단계 | 문서 | 뜻 |
|---|---|---|
| inbox | 1건 | 아직 처리하지 않은 메모 |
| sources | 1건 | 고치지 않고 보존하는 원본 |
| context | 4건 | 게이트를 지난 검수된 지식 |

` + "`engram status .`" + ` 를 돌리면 이 표를 도구가 직접 낸다. inbox 에 하나가
남아 있는 것은 실수가 아니라 의도다. 적체가 0인 위키로는 status 가
무엇을 보는지 보여줄 수 없다.
`

const pipelineBody = `# 지식 승급 파이프라인

## 결론

메모가 지식이 되려면 관문을 하나 지나야 한다. engram 은 그 관문을 코드로
강제한다. 이것이 다른 메모 도구와 갈리는 유일한 지점이다.

## 맥락

메모를 모으는 도구는 많고 대부분 잘 동작한다. 문제는 모은 다음이다.
검수를 거치지 않은 조각이 검수된 지식과 같은 자리에 쌓이면, 몇 달 뒤에는
무엇을 믿어도 되는지 아무도 모르게 된다.

## 현재 이해

문서는 세 단계를 지난다.

| 단계 | 무엇이 사는가 | 규칙 |
|---|---|---|
| inbox | 검증하지 않은 러프 캡처 | 아무거나 넣는다. 마찰이 없어야 한다 |
| sources | 원본 자료 | 한 번 들어오면 본문을 고치지 않는다 |
| context | 검수된 지식 | 게이트를 지나야 들어온다 |

` + "`inbox`" + ` 에서 ` + "`context`" + ` 로 가려면 ` + "`promote`" + ` 를 부르고, 그때 게이트가 돈다.
거절 사유는 하나뿐이다. 위키링크가 기준보다 적으면 거절한다. 나머지는
전부 경고다.

## 근거

거절 조건을 늘리면 사용자가 우회로를 찾기 시작하고, 우회로가 생기는
순간 관문이 관문이 아니게 된다. 링크 하나를 요구하는 것은 "이 문서가
기존 지식과 어디서 만나는가"를 쓰는 사람이 한 번은 생각하게 만드는
장치다. [[wikilink-graph]] 가 그 링크가 무엇을 만드는지 다룬다.

되돌리는 길도 있다. ` + "`demote`" + ` 가 승급을 취소한다. 관문을 강제하는 도구일수록
되돌리기가 신뢰를 만든다.

## 관련 링크

- [[wikilink-graph]]
- [[km-primer]]
`

const graphBody = `# 위키링크로 엮는 지식 그래프

## 결론

` + "`[[슬러그]]`" + ` 하나가 문서 사이의 관계를 만든다. 그 관계가 쌓이면 검색이
못 찾는 것을 찾아 주는 그래프가 된다.

## 맥락

폴더로 분류하면 문서 하나는 자리 하나를 갖는다. 그런데 실제 지식은 여러
갈래에 동시에 걸친다. 폴더를 아무리 잘 나눠도 "이 문서를 어디 둘까"에서
매번 막히는 이유다.

## 현재 이해

링크는 두 자리에 적는다. 본문에 ` + "`[[슬러그]]`" + ` 로 적거나, 프론트매터의
` + "`related`" + ` 에 넣는다. 둘 다 그래프에 들어가지만 쓰임이 다르다. 본문
링크는 문맥 안에서 자연스럽게 생기고, ` + "`related`" + ` 는 문맥 없이 관계만
선언한다.

이름을 바꿔야 할 때가 온다. ` + "`mv`" + ` 가 슬러그를 바꾸면서 그 문서를 가리키던
링크를 전부 함께 고친다. 이것은 선택 기능이 아니다. 링크를 따라가지
않는 이름 변경은 그래프를 그 자리에서 부순다.

## 근거

그래프가 있으면 도구가 사람 대신 볼 수 있는 것이 생긴다.

- ` + "`backlinks`" + ` 는 이 문서를 누가 가리키는지 보여준다
- ` + "`bridge`" + ` 는 내용이 비슷한데 링크가 없는 쌍을 찾아 준다
- ` + "`digest`" + ` 는 링크가 하나도 없는 고아 문서를 센다

셋 다 사람이 손으로는 못 하는 일이다. 문서가 백 개를 넘으면 머릿속에
그래프가 안 들어온다.

## 관련 링크

- [[promotion-pipeline]]
- [[km-primer]]
`

const primerBody = `# 지식관리 입문 자료

## 결론

원본을 보존하는 계층을 따로 두면, 요약이 틀렸을 때 되돌아갈 자리가 남는다.

## 맥락

이 문서는 ` + "`sources`" + ` 에 있는 원본에서 파생되었다. 원본은 지워지지 않고
제자리에 남아 있으며, 프론트매터의 ` + "`derived_from`" + ` 과 ` + "`derived_context`" + ` 가
두 문서를 양방향으로 잇는다.

## 현재 이해

` + "`promote`" + ` 는 출발지에 따라 다르게 동작한다.

| 출발지 | 동작 |
|---|---|
| inbox | 문서를 ` + "`context`" + ` 로 **옮긴다**. 원본이 남지 않는다 |
| sources | 파생 문서를 **새로 만든다**. 원본은 그대로 있다 |

원본 보존 계층에서 문서를 빼내면 보존이라는 약속이 그 순간 깨지기
때문이다. 같은 이유로 ` + "`sources`" + ` 문서에는 ` + "`updated`" + ` 필드를 쓰지 않는다.
오타 하나 고친 것이 자료의 신선도를 오해하게 만든다.

## 근거

요약은 시간이 지나면 틀린다. 무엇을 근거로 그렇게 요약했는지 돌아갈 수
있어야 고칠 수 있다. [[promotion-pipeline]] 의 게이트가 지키는 것이
"쌓이는 쪽"이라면, 원본 보존이 지키는 것은 "거슬러 올라가는 쪽"이다.

## 관련 링크

- [[promotion-pipeline]]
- [[wikilink-graph]]
`
