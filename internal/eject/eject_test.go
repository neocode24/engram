package eject

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
)

// writeWiki는 임시 디렉토리에 위키 파일들을 만들고 그 루트를 반환한다.
func writeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// loadCfg는 임시 위키의 설정을 읽는다.
func loadCfg(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("설정 로드 실패: %v", err)
	}
	return cfg
}

// artifactByPath는 산출물 하나를 찾는다.
func artifactByPath(t *testing.T, plan []Artifact, path string) Artifact {
	t.Helper()
	for _, a := range plan {
		if a.Path == path {
			return a
		}
	}
	t.Fatalf("산출물이 없음: %s", path)
	return Artifact{}
}

// writePlan은 산출물 전부를 위키에 쓴다. 커맨드 계층과 같은 규칙으로
// 디렉토리를 만들고 모드를 지킨다.
func writePlan(t *testing.T, root string, plan []Artifact) {
	t.Helper()
	for _, a := range plan {
		p := filepath.Join(root, filepath.FromSlash(a.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(a.Content), a.Mode); err != nil {
			t.Fatal(err)
		}
	}
}

// needPython3는 python3 가 없으면 테스트를 건너뛴다.
func needPython3(t *testing.T) string {
	t.Helper()
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 이 PATH 에 없다")
	}
	return py
}

// runPython은 인자를 주어 python3 를 돌리고 출력과 종료 코드를 반환한다.
func runPython(t *testing.T, py string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(py, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("python3 실행 실패: %v", err)
	}
	return string(out), code
}

func TestPlan(t *testing.T) {
	t.Run("산출물 다섯 종류를 전부 만든다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		plan := Plan(loadCfg(t, root))
		var kinds int
		hasMeta, hasScript, hasHook, hasAgents, hasAttrs := false, false, false, false, false
		for _, a := range plan {
			switch {
			case strings.HasPrefix(a.Path, "meta/") && strings.HasSuffix(a.Path, ".md"):
				hasMeta = true
			case a.Path == "scripts/lint-frontmatter.py":
				hasScript = true
			case a.Path == ".githooks/pre-commit":
				hasHook = true
			case a.Path == "AGENTS.md":
				hasAgents = true
			case a.Path == ".gitattributes":
				hasAttrs = true
			}
		}
		for _, ok := range []bool{hasMeta, hasScript, hasHook, hasAgents, hasAttrs} {
			if ok {
				kinds++
			}
		}
		if kinds != 5 {
			t.Errorf("산출물 종류 = %d, want 5: %+v", kinds, plan)
		}
	})

	t.Run("두 번 만들면 산출물이 바이트까지 같다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		first := Plan(loadCfg(t, root))
		second := Plan(loadCfg(t, root))
		if !reflect.DeepEqual(first, second) {
			t.Fatal("같은 설정에 두 번 만든 산출물이 다름")
		}
	})

	t.Run("프리셋이 다르면 meta 문서와 린터가 다르다", func(t *testing.T) {
		edu := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		personal := writeWiki(t, map[string]string{"engram.yaml": "preset: minimal\n"})
		eduPlan := Plan(loadCfg(t, edu))
		personalPlan := Plan(loadCfg(t, personal))
		if artifactByPath(t, eduPlan, "meta/frontmatter-schema.md").Content ==
			artifactByPath(t, personalPlan, "meta/frontmatter-schema.md").Content {
			t.Error("프리셋이 다른데 meta 문서가 같음")
		}
		if artifactByPath(t, eduPlan, "scripts/lint-frontmatter.py").Content ==
			artifactByPath(t, personalPlan, "scripts/lint-frontmatter.py").Content {
			t.Error("프리셋이 다른데 린터 내용이 같음")
		}
	})

	t.Run("임계값을 생성물에 반영한다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		cfg := loadCfg(t, root)
		cfg.Thresholds.MinWikilinks = 7
		plan := Plan(cfg)
		if !strings.Contains(artifactByPath(t, plan, "meta/promotion-rules.md").Content, "7") {
			t.Error("min_wikilinks 7 이 게이트 문서에 없음")
		}
		if !strings.Contains(artifactByPath(t, plan, "scripts/lint-frontmatter.py").Content, `"min_wikilinks": 7`) {
			t.Error("min_wikilinks 7 이 린터 기본값에 없음")
		}
	})

	t.Run("훅과 린터에 실행 권한이 있다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		plan := Plan(loadCfg(t, root))
		for _, path := range []string{"scripts/lint-frontmatter.py", ".githooks/pre-commit"} {
			if a := artifactByPath(t, plan, path); a.Mode != 0o755 {
				t.Errorf("%s 모드 = %o, want 755", path, a.Mode)
			}
		}
	})

	t.Run("규칙 목록은 lint.Rules 에서 온다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		doc := artifactByPath(t, Plan(loadCfg(t, root)), "meta/lint-rules.md").Content
		for _, r := range lint.Rules() {
			if !strings.Contains(doc, r.ID) {
				t.Errorf("규칙 %s 가 문서에 없음", r.ID)
			}
		}
	})
}

// cleanWiki는 린터가 내는 위반이 없는 위키 파일이다. 링크가 서로 이어진
// context 문서 셋과 색인이다.
func cleanWiki() map[string]string {
	contextDoc := func(relations ...string) string {
		items := make([]string, 0, len(relations))
		for _, r := range relations {
			items = append(items, "  - \"[["+r+"]]\"")
		}
		list := strings.Join(items, "\n")
		return "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
			"related:\n" + list + "\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n본문\n"
	}
	return map[string]string{
		"engram.yaml": "preset: personal\n",
		"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
		"context/a.md": contextDoc("b", "c"),
		"context/b.md": contextDoc("a", "c"),
		"context/c.md": contextDoc("a", "b"),
	}
}

// violatingWiki는 문서 단위 위반이 있는 위키 파일이다. 전 문서가 같은
// 주제를 달아 wiki.broad-topic 조건도 만든다.
func violatingWiki() map[string]string {
	files := cleanWiki()
	files["inbox/misplaced.md"] = "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\nrelated: []\n" +
		"source_channel: manual\nderived_context: []\n---\n\n검수 단계를 선언하고 inbox 에 남은 문서\n"
	files["inbox/alone.md"] = "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
		"indexable: false\n---\n\n링크 없는 메모\n"
	files["context/broken.md"] = "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
		"related:\n  - \"[[없는문서]]\"\nsource_channel: manual\nderived_context: []\n" +
		"---\n\n본문 [[없는문서]] 링크\n"
	for name, content := range files {
		if strings.HasPrefix(name, "context/") || name == "index.md" {
			files[name] = strings.Replace(content, "tags: []", "tags: []\ntopics: [go]", 1)
		}
	}
	return files
}

func TestLinter(t *testing.T) {
	t.Run("python3 로 문법 오류 없이 로드된다", func(t *testing.T) {
		py := needPython3(t)
		root := writeWiki(t, map[string]string{"engram.yaml": "preset: personal\n"})
		writePlan(t, root, Plan(loadCfg(t, root)))
		// 인코딩을 명시한다. Windows 의 파이썬은 로케일 인코딩(cp1252 등)으로
		// 파일을 열어서, 한글 주석이 든 이 스크립트를 UnicodeDecodeError 로
		// 읽지 못한다.
		out, code := runPython(t, py, "-c",
			"import ast,sys; ast.parse(open(sys.argv[1], encoding='utf-8').read())",
			filepath.Join(root, "scripts", "lint-frontmatter.py"))
		if code != 0 {
			t.Fatalf("문법 오류: %s", out)
		}
	})

	t.Run("정상 위키에서 종료 코드 0", func(t *testing.T) {
		py := needPython3(t)
		root := writeWiki(t, cleanWiki())
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, code := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)
		if code != 0 {
			t.Fatalf("정상 위키가 종료 코드 %d: %s", code, out)
		}
	})

	t.Run("위반 있는 위키에서 종료 코드 1", func(t *testing.T) {
		py := needPython3(t)
		root := writeWiki(t, violatingWiki())
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, code := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)
		if code != 1 {
			t.Fatalf("위반 위키가 종료 코드 %d: %s", code, out)
		}
		for _, want := range []string{"location.stage-agreement", "link.broken", "graph.orphan"} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %s 없음:\n%s", want, out)
			}
		}
	})

	t.Run("경고만 있는 위키에서 종료 코드 0", func(t *testing.T) {
		py := needPython3(t)
		// 고아 문서 하나뿐인 위키다. 경고는 정상 상태이므로 커밋 훅이
		// 이 위키를 막으면 안 된다. engram lint 의 HasBlocking 과 같다.
		files := map[string]string{
			"engram.yaml": "preset: personal\n",
			"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
			"inbox/note.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel:\n---\n\n링크 없는 메모\n",
		}
		root := writeWiki(t, files)
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, code := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)
		if code != 0 {
			t.Fatalf("경고만 있는 위키가 종료 코드 %d:\n%s", code, out)
		}
		if !strings.Contains(out, "graph.orphan") {
			t.Errorf("경고가 출력에 없음:\n%s", out)
		}
	})

	t.Run("CRLF 와 BOM 과 한글 파일명을 정상 파싱한다", func(t *testing.T) {
		py := needPython3(t)
		files := cleanWiki()
		// CRLF 프론트매터. upstream 의 bash 린터가 인식하지 못하는 형태다.
		files["context/crlf.md"] = "\xEF\xBB\xBF---\r\ntype: concept\r\nartifact_stage: context\r\n" +
			"status: promoted\r\nindexable: true\r\ntags: []\r\nsource_refs: []\r\n" +
			"derived_from: []\r\nrelated:\r\n  - \"[[a]]\"\r\n  - \"[[b]]\"\r\n" +
			"source_channel: manual\r\nderived_context: []\r\n---\r\n\r\n본문 [[a]]\r\n"
		files["context/한글문서.md"] = "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[a]]\"\n  - \"[[b]]\"\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n한글 제목 문서. [[a]] 링크\n"
		root := writeWiki(t, files)
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, code := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)
		if code != 0 {
			t.Fatalf("CRLF/BOM/한글 문서에서 위반이 나옴(종료 코드 %d):\n%s", code, out)
		}
	})

	t.Run("wiki.broad-topic 을 내지 않는다", func(t *testing.T) {
		py := needPython3(t)
		root := writeWiki(t, violatingWiki())
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, _ := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "wiki.broad-topic") {
				t.Fatalf("wiki.broad-topic 이 출력에 있음: %s", line)
			}
		}
	})

	t.Run("engram lint 와 문서 단위 규칙 판정이 같다", func(t *testing.T) {
		py := needPython3(t)
		root := writeWiki(t, violatingWiki())
		writePlan(t, root, Plan(loadCfg(t, root)))
		out, _ := runPython(t, py, filepath.Join(root, "scripts", "lint-frontmatter.py"), root)

		goRes, err := lint.Run(root, loadCfg(t, root))
		if err != nil {
			t.Fatal(err)
		}
		// 위키 단위 진단(wiki.broad-topic)은 린터가 내보내지 않는 대상이므로
		// 파일 위반만 대조한다. 경로, 줄, 규칙, 등급을 전부 본다.
		want := make([]string, 0, len(goRes.Violations))
		for _, v := range goRes.Violations {
			want = append(want, fmt.Sprintf("%s %d [%s] %s", v.Path, v.Line, v.Severity, v.Rule))
		}
		sort.Strings(want)

		var got []string
		current := ""
		for _, line := range strings.Split(out, "\n") {
			switch {
			case strings.HasPrefix(line, "  ["):
				parts := strings.Fields(line)
				if len(parts) == 3 {
					got = append(got, fmt.Sprintf("%s %s %s %s", current, parts[1], parts[0], parts[2]))
				}
			case line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "검사한"):
				current = line
			}
		}
		sort.Strings(got)

		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("규칙 판정이 다름:\n python: %v\n engram: %v\n%s", got, want, out)
		}
	})
}
