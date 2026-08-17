// Package eject는 내보낸 Python 린터와 engram lint 의 판정이 같은지
// 대조한다. ADR 0039 의 계약이다.
//
// 다른 동등성 축은 upstream 이라는 남과의 비교라 차이를 측정값으로만
// 남긴다. 이 축은 자기 산출물끼리의 비교다. "이것이 당신을 막던
// 규칙입니다"라고 주면서 다른 규칙을 주면 안 되므로 어긋나면 실패다.
// ENGRAM_UPSTREAM 과 무관하게 항상 돈다.
//
// 커맨드 계층이 아니라 빌드한 바이너리를 exec 한다. 검증 대상이 표준
// 출력과 종료 코드이기 때문이다. 종료 코드 결함은 출력만 보면 보이지
// 않는다.
package eject

import (
	"bytes"
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

// pythonPath는 TestMain 이 찾은 python3 경로다. 없으면 비어 있고 테스트는
// skip 한다. python3 부재가 이 하니스의 유일한 skip 조건이다.
var pythonPath string

// repoRoot는 모듈 루트다. 테스트는 harness/eject 에서 돈다.
var repoRoot = filepath.Join("..", "..")

// excludedRules는 Python 린터가 의도적으로 내보내지 않는 규칙이다.
// 대조에서 이 규칙의 출력 구간만 뺀다. 이 목록에 없는 규칙이 어긋나면
// 실패한다. 조용히 무시하는 자리를 만들지 않는다.
var excludedRules = []string{"wiki.broad-topic"}

// TestMain은 실제 바이너리를 만들고 python3 를 찾는다.
func TestMain(m *testing.M) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("go 가 PATH 에 없어 eject 대조 테스트를 건너뛴다")
		os.Exit(m.Run())
	}
	if pythonPath, err = exec.LookPath("python3"); err != nil {
		pythonPath = ""
	}
	dir, err := os.MkdirTemp("", "engram-eject-")
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
// 이 패키지는 바이너리를 exec 하므로 검사 대상 코드에 소스 의존이 없다.
// 파일을 읽었다는 사실만 캐시 키에 들어간다. 반드시 테스트 함수 안에서
// 부른다. 파일 접근을 기록하는 testlog 는 m.Run 이 켜므로 TestMain 에서
// 읽으면 기록되지 않는다. harness/journey 와 같은 이유, 같은 방법이다.
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

// run은 명령을 돌려 표준 출력과 종료 코드를 반환한다. 실패한 명령도
// 대조 대상이므로 에러가 아니라 결과로 돌려준다.
func run(t *testing.T, name string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("%s 실행 실패: %v", name, err)
	}
	return out.String(), code
}

// copyGoldenWiki는 골든 위키를 임시 디렉토리에 복사한다.
func copyGoldenWiki(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join(repoRoot, "harness", "fixtures", "golden-wiki")
	err := filepath.WalkDir(src, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("골든 위키 복사 실패: %v", err)
	}
	return dst
}

// setPreset은 위키 설정의 프리셋을 바꾼다.
func setPreset(t *testing.T, wiki, preset string) {
	t.Helper()
	path := filepath.Join(wiki, "engram.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	found := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "preset:") {
			lines = append(lines, "preset: "+preset)
			found = true
			continue
		}
		lines = append(lines, line)
	}
	if !found {
		t.Fatalf("프리셋 줄이 없음: %s", raw)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// canonicalize는 출력을 위반 단위로 쪼개 비교 가능한 형태로 낸다.
//
// 빼는 것은 둘뿐이다. excludedRules 의 위키 진단 구간과 요약 줄(경고 수에
// 위키 진단 건수가 포함되므로). 그 밖의 모든 줄은 그대로 남는다.
//
// 파일 안의 위반 단위는 정렬해 낸다. engram lint 의 정렬은 sort.Slice 로
// 안정적이지 않아 (경로, 줄, 규칙) 키가 같은 위반의 순서가 구현 사정을
// 따른다. 판정 내용은 그 순서와 무관하므로 단위 안에서만 정렬해 대조한다.
// 규칙이 다른 위반의 순서는 키 정렬이 이미 정하므로 이 정렬이 순서 차이를
// 숨기지 않는다.
func canonicalize(t *testing.T, out string) string {
	t.Helper()
	var b strings.Builder
	var path string
	var stanzas []string
	flush := func() {
		if path == "" {
			return
		}
		sort.Strings(stanzas)
		b.WriteString(path + "\n")
		for _, s := range stanzas {
			b.WriteString(s)
		}
		stanzas = nil
	}
	lines := strings.Split(out, "\n")
	inWikiFindings := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case line == "위키 진단:":
			flush()
			path = ""
			inWikiFindings = true
		case inWikiFindings:
			if strings.HasPrefix(line, "검사한 파일") {
				inWikiFindings = false
				continue
			}
			// 위키 진단 구간의 규칙이 제외 목록에 있는지 검사한다.
			// 목록 밖의 규칙이 여기 나오면 조용히 빼는 것이 아니라 실패다.
			trimmed := strings.TrimSpace(line)
			fields := strings.Fields(trimmed)
			if strings.HasPrefix(trimmed, "[") && len(fields) >= 2 && !contains(excludedRules, fields[1]) {
				t.Fatalf("제외 목록에 없는 위키 진단이 출력에 있음: %s (제외 목록: %v)", fields[1], excludedRules)
			}
		case strings.HasPrefix(line, "검사한 파일"):
			// 요약 줄은 경고 수에 위키 진단 건수가 포함되어 비교에서 뺀다.
			flush()
			path = ""
		case strings.HasPrefix(line, "  ["):
			stanza := line + "\n"
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    ") {
				i++
				stanza += lines[i] + "\n"
			}
			stanzas = append(stanzas, stanza)
		case line != "" && !strings.HasPrefix(line, " "):
			flush()
			path = line
		}
	}
	flush()
	return b.String()
}

// contains는 값이 목록에 있는지 검사한다.
func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// compareLinters는 위키 하나에 두 린터를 돌려 출력과 종료 코드를 대조한다.
// eject 를 먼저 돌려 산출물을 만들어 둔다.
func compareLinters(t *testing.T, wiki string) {
	t.Helper()
	if out, code := run(t, binaryPath, "eject", "--wiki", wiki); code != 0 {
		t.Fatalf("eject 실패(종료 코드 %d):\n%s", code, out)
	}

	engOut, engCode := run(t, binaryPath, "lint", wiki)
	pyOut, pyCode := run(t, pythonPath, filepath.Join(wiki, "scripts", "lint-frontmatter.py"), wiki)

	if engCode != pyCode {
		t.Fatalf("종료 코드가 다름: engram lint %d, python %d\n--- engram lint ---\n%s\n--- python ---\n%s",
			engCode, pyCode, engOut, pyOut)
	}
	if canonicalize(t, engOut) != canonicalize(t, pyOut) {
		t.Fatalf("출력이 다름:\n--- engram lint ---\n%s\n--- python ---\n%s", engOut, pyOut)
	}
}

func TestEjectMatchesEngramLint(t *testing.T) {
	if binaryPath == "" {
		t.Skip("바이너리가 없어 대조를 돌리지 못한다. go 가 PATH 에 있어야 한다")
	}
	if pythonPath == "" {
		t.Skip("python3 가 PATH 에 없다")
	}
	touchSources()

	t.Run("골든 위키는 규칙 경계를 때리는 문서 전부에서 같다", func(t *testing.T) {
		compareLinters(t, copyGoldenWiki(t))
	})

	for _, preset := range []string{"personal", "education", "team"} {
		t.Run("골든 위키를 "+preset+" 프리셋으로 검사하면 같다", func(t *testing.T) {
			wiki := copyGoldenWiki(t)
			setPreset(t, wiki, preset)
			compareLinters(t, wiki)
		})
	}

	t.Run("init 직후의 빈 위키는 종료 코드 0 이다", func(t *testing.T) {
		wiki := filepath.Join(t.TempDir(), "wiki")
		if out, code := run(t, binaryPath, "init", wiki); code != 0 {
			t.Fatalf("init 실패:\n%s", out)
		}
		compareLinters(t, wiki)
	})

	t.Run("경고만 있는 위키는 종료 코드 0 이다", func(t *testing.T) {
		wiki := warnOnlyWiki(t)
		compareLinters(t, wiki)
	})

	t.Run("오류가 있는 위키는 종료 코드 1 이다", func(t *testing.T) {
		wiki := warnOnlyWiki(t)
		writeFile(t, wiki, "inbox/claims-context.md",
			"---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"+
				"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\nrelated: []\n"+
				"source_channel: manual\nderived_context: []\n---\n\n검수 단계를 선언하고 inbox 에 남은 문서\n")
		compareLinters(t, wiki)
	})

	t.Run("게이트 거절이 있는 위키는 종료 코드 1 이다", func(t *testing.T) {
		wiki := warnOnlyWiki(t)
		writeFile(t, wiki, "context/thin.md",
			"---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"+
				"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n"+
				"related:\n  - \"[[index]]\"\nsource_channel: manual\nderived_context: []\n"+
				"---\n\n본문\n")
		compareLinters(t, wiki)
	})
}

// warnOnlyWiki는 고아 문서 하나로 경고만 나는 위키를 만든다. 경고는 정상
// 상태이므로 두 린터 모두 종료 코드 0 을 내야 한다.
func warnOnlyWiki(t *testing.T) string {
	t.Helper()
	wiki := t.TempDir()
	writeFile(t, wiki, "engram.yaml", "preset: education\n")
	writeFile(t, wiki, "index.md",
		"---\ntype: system\nartifact_stage: context\nstatus: promoted\n"+
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n"+
			"source_channel: manual\nderived_context: []\n---\n\n# 색인\n")
	writeFile(t, wiki, "inbox/note.md",
		"---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n"+
			"indexable: false\nsource_channel:\n---\n\n링크 없는 메모\n")
	return wiki
}

// writeFile은 위키 루트 아래 파일 하나를 만든다.
func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLinterSurvivesNonUTF8Console는 내보낸 Python 린터가 UTF-8 이 아닌
// 콘솔 인코딩에서도 한글 메시지를 내는지 본다.
//
// Windows 콘솔의 기본 인코딩은 cp1252 나 cp949 다. 파이썬은 그 인코딩으로
// 담을 수 없는 문자를 만나면 UnicodeEncodeError 로 죽는다. engram 본체는
// 콘솔 코드페이지를 UTF-8 로 바꿔 푸는데(ADR 0026) 내보낸 스크립트는 그
// 처리를 받지 못하므로 스스로 스트림을 다시 열어야 한다.
//
// 이 결함은 Windows CI 에서만 드러났다. cp1252 를 PYTHONIOENCODING 으로
// 주면 어느 플랫폼에서든 재현되므로 여기서 못 박는다.
func TestLinterSurvivesNonUTF8Console(t *testing.T) {
	if binaryPath == "" {
		t.Skip("바이너리가 없다")
	}
	if pythonPath == "" {
		t.Skip("python3 가 없다")
	}
	touchSources()

	wiki := t.TempDir()
	if _, code := run(t, binaryPath, "init", wiki); code != 0 {
		t.Fatalf("init 실패: 종료 코드 %d", code)
	}
	if _, code := run(t, binaryPath, "eject", "--wiki", wiki); code != 0 {
		t.Fatalf("eject 실패: 종료 코드 %d", code)
	}

	linter := filepath.Join(wiki, "scripts", "lint-frontmatter.py")
	cmd := exec.Command(pythonPath, linter, wiki)
	// cp1252 는 한글을 담지 못한다. 스트림을 다시 열지 않으면 여기서 죽는다.
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=cp1252")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cp1252 콘솔에서 린터가 죽었다: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "UnicodeEncodeError") {
		t.Fatalf("cp1252 콘솔에서 인코딩 오류가 났다:\n%s", out)
	}
	if !strings.Contains(string(out), "검사한 파일") {
		t.Errorf("한글 메시지가 온전히 나오지 않았다:\n%s", out)
	}
}
