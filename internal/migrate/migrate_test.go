package migrate

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/wiki"
)

// makeWiki는 파일 맵으로 임시 위키를 만든다. engram.yaml 을 주지 않으면
// 기본값(education 프리셋)으로 돈다.
func makeWiki(t *testing.T, files map[string]string) string {
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

// readDoc는 위키 안 상대 경로의 파일을 읽는다.
func readDoc(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// load는 위키 설정을 읽는다.
func load(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// run은 migrate.Run 을 돌아 실패를 테스트 오류로 바꾼다.
func run(t *testing.T, root string, cfg config.Config, opts Options) Report {
	t.Helper()
	rep, err := Run(root, cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestRun(t *testing.T) {
	t.Run("켜진 축의 필수 필드가 없으면 단계별 초기값으로 채운다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\n---\n\n본문입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		if rep.Changed != 1 || rep.Written != 1 {
			t.Fatalf("문서 1개가 바뀌어야 한다. changed %d, written %d", rep.Changed, rep.Written)
		}
		got := readDoc(t, root, "context/a.md")
		for _, want := range []string{"related: []", "tags: []", "source_refs: []", "source_channel:"} {
			if !strings.Contains(got, want) {
				t.Fatalf("채워진 필드가 없다: %s\n실제:\n%s", want, got)
			}
		}
		var fill []string
		for _, c := range rep.Documents[0].Changes {
			if c.Kind == KindFill {
				fill = append(fill, c.Field)
			}
		}
		if len(fill) == 0 {
			t.Fatalf("채우기 변경이 보고되지 않았다: %+v", rep.Documents[0].Changes)
		}
	})

	t.Run("꺼진 축의 빈 필드는 force 없이 지운다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\nrelated: []\nsensitivity:\n---\n\n본문입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		got := readDoc(t, root, "context/a.md")
		if strings.Contains(got, "sensitivity") {
			t.Fatalf("빈 꺼진 축 필드가 남아 있다:\n%s", got)
		}
		if rep.Blocked != 0 {
			t.Fatalf("빈 필드는 보류 대상이 아니다. blocked %d", rep.Blocked)
		}
	})

	t.Run("꺼진 축의 값 있는 필드는 force 없이는 보류하고 알린다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\nrelated: []\nsensitivity: internal\n---\n\n본문입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		if rep.Blocked != 1 {
			t.Fatalf("값 있는 꺼진 축 필드 하나가 보류되어야 한다. blocked %d", rep.Blocked)
		}
		b := rep.Documents[0].Blocked[0]
		if b.Field != "sensitivity" || b.Old != "internal" {
			t.Fatalf("보류 변경이 옳지 않다: %+v", b)
		}
		if got := readDoc(t, root, "context/a.md"); !strings.Contains(got, "sensitivity: internal") {
			t.Fatalf("보류된 필드가 지워졌다:\n%s", got)
		}
	})

	t.Run("force 를 주면 값 있는 필드도 지운다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\nrelated: []\nsensitivity: internal\n---\n\n본문입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true, Force: true})
		if rep.Blocked != 0 {
			t.Fatalf("force 로도 보류가 남았다. blocked %d", rep.Blocked)
		}
		if got := readDoc(t, root, "context/a.md"); strings.Contains(got, "sensitivity") {
			t.Fatalf("force 로 지웠는데 필드가 남아 있다:\n%s", got)
		}
	})

	t.Run("archive 디렉토리 문서의 단계를 archive 로 고친다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"archive/old.md": "---\ntype: concept\nartifact_stage: inbox\n" +
				"status: inbox\nindexable: false\n---\n\n오래된 문서입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		got := readDoc(t, root, "archive/old.md")
		if !strings.Contains(got, "artifact_stage: archive") {
			t.Fatalf("위치에 맞게 단계가 archive 로 고쳐지지 않았다:\n%s", got)
		}
		if rep.Documents[0].Changes[0].Kind != KindStage {
			t.Fatalf("첫 변경이 단계 고침이어야 한다: %+v", rep.Documents[0].Changes)
		}
	})

	t.Run("inbox 에 있으면서 context 로 선언한 문서는 선언이 내려가고 파일은 옮겨지지 않는다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"inbox/2026-01-05-memo.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\nrelated: []\n---\n\n승급 게이트를 지나지 않은 문서입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		got := readDoc(t, root, "inbox/2026-01-05-memo.md")
		if !strings.Contains(got, "artifact_stage: inbox") {
			t.Fatalf("선언이 위치에 맞게 inbox 로 내려가지 않았다:\n%s", got)
		}
		// 파일이 context 로 옮겨졌는지 본다. 옮겨지면 게이트 우회다.
		if _, err := os.Stat(filepath.Join(root, "context", "2026-01-05-memo.md")); err == nil {
			t.Fatal("문서가 context 디렉토리로 옮겨졌다. migrate 는 파일을 옮기지 않는다")
		}
		if _, err := os.Stat(filepath.Join(root, "context", "memo.md")); err == nil {
			t.Fatal("문서가 context 디렉토리로 옮겨졌다. migrate 는 파일을 옮기지 않는다")
		}
		var demoted bool
		for _, c := range rep.Documents[0].Changes {
			if c.Kind == KindStage && c.New == "inbox" {
				demoted = true
			}
		}
		if !demoted {
			t.Fatalf("강등 변경이 보고되지 않았다: %+v", rep.Documents[0].Changes)
		}
	})

	t.Run("기본은 시험 실행이라 파일을 쓰지 않는다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\n---\n\n본문입니다.\n",
		})
		before := readDoc(t, root, "context/a.md")
		rep := run(t, root, load(t, root), Options{})
		if rep.Written != 0 {
			t.Fatalf("시험 실행은 쓰면 안 된다. written %d", rep.Written)
		}
		if rep.Changed != 1 {
			t.Fatalf("시험 실행도 변경 예정을 계산해야 한다. changed %d", rep.Changed)
		}
		if after := readDoc(t, root, "context/a.md"); after != before {
			t.Fatalf("시험 실행이 파일을 바꿨다:\n%s", after)
		}
	})

	t.Run("두 번 적용하면 두 번째는 바뀌는 문서가 없다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\n" +
				"status: promoted\nindexable: true\n---\n\n본문입니다.\n",
			"archive/old.md": "---\ntype: concept\nartifact_stage: inbox\n" +
				"status: inbox\nindexable: false\n---\n\n오래된 문서입니다.\n",
		})
		cfg := load(t, root)
		first := run(t, root, cfg, Options{Apply: true})
		if first.Changed == 0 {
			t.Fatal("첫 적용은 변경이 있어야 한다")
		}
		second := run(t, root, cfg, Options{Apply: true})
		if second.Changed != 0 || second.Written != 0 {
			t.Fatalf("두 번째 적용은 변경이 없어야 한다. changed %d, written %d", second.Changed, second.Written)
		}
	})

	t.Run("프론트매터 키 순서를 보존한다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\ntags: []\nstatus: promoted\n" +
				"artifact_stage: context\nindexable: true\n---\n\n본문입니다.\n",
		})
		run(t, root, load(t, root), Options{Apply: true})
		got := readDoc(t, root, "context/a.md")
		order := []string{"type:", "tags:", "status:", "artifact_stage:", "indexable:", "related:"}
		last := -1
		for _, key := range order {
			i := strings.Index(got, "\n"+key)
			if i < 0 {
				i = strings.Index(got, key)
			}
			if i < 0 {
				t.Fatalf("키가 없다: %s\n실제:\n%s", key, got)
			}
			if i < last {
				t.Fatalf("키 순서가 바뀌었다. %s 가 이전 키보다 앞에 있다:\n%s", key, got)
			}
			last = i
		}
	})

	t.Run("게이트 위반 문서는 고치지 않고 보고만 한다", func(t *testing.T) {
		full := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated:\n  - \"[[b]]\"\n  - \"[[c]]\"\n---\n\n본문입니다.\n"
		loner := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated: []\n---\n\n링크가 없는 문서입니다.\n"
		root := makeWiki(t, map[string]string{
			"context/a.md": full,
			"context/b.md": full,
			"context/c.md": loner,
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		var gate []string
		for _, a := range rep.Advisories {
			if a.Rule == "gate.min-wikilinks" {
				gate = append(gate, a.Path)
			}
		}
		if len(gate) != 1 || gate[0] != "context/c.md" {
			t.Fatalf("게이트 위반이 c 하나로 보고되어야 한다. got %v, advisories %+v", gate, rep.Advisories)
		}
		// 고쳤다면 링크가 채워졌을 것이다. 링크를 지어내지 않는지 본다.
		if got := readDoc(t, root, "context/c.md"); strings.Count(got, "[[") != 0 {
			t.Fatalf("migrate 가 링크를 지어냈다:\n%s", got)
		}
	})

	t.Run("프론트매터를 읽지 못한 문서는 목록에 남는다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/a.md": "프론트매터가 없는 문서입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{})
		if len(rep.Unparsed) != 1 || rep.Unparsed[0] != "context/a.md" {
			t.Fatalf("읽지 못한 문서가 보고되지 않았다: %+v", rep.Unparsed)
		}
	})

	t.Run("sources 디렉토리의 inbox 선언 문서는 단계가 source 가 되고 created 를 파일명에서 채운다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"sources/2026-04-01-talk.md": "---\ntype: source-summary\nartifact_stage: inbox\n" +
				"status: inbox\nindexable: false\nsourced_at: 2026-04-02\n---\n\n원본입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		got := readDoc(t, root, "sources/2026-04-01-talk.md")
		if !strings.Contains(got, "artifact_stage: source") {
			t.Fatalf("단계가 source 로 고쳐지지 않았다:\n%s", got)
		}
		if !strings.Contains(got, "created: 2026-04-01") {
			t.Fatalf("created 가 파일명 접두사에서 채워지지 않았다:\n%s", got)
		}
		if rs := rep.Documents[0].Remainders; len(rs) != 0 {
			t.Fatalf("채울 수 있는 필드가 남은 것으로 보고되었다: %+v", rs)
		}
	})

	t.Run("sourced_at 은 채우지 않고 남은 것으로 보고한다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"sources/2026-04-01-talk.md": "---\ntype: source-summary\nartifact_stage: source\n" +
				"status: sourced\nindexable: false\ncreated: 2026-04-01\n" +
				"tags: []\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel:\nderived_context: []\n---\n\n원본입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		rs := rep.Documents[0].Remainders
		if len(rs) != 1 || rs[0].Field != "sourced_at" {
			t.Fatalf("sourced_at 하나가 남은 것으로 보고되어야 한다: %+v", rs)
		}
		if rep.NonConforming != 1 || rep.Partial != 1 || rep.Changed != 0 || rep.Written != 0 {
			t.Fatalf("남은 것이 있으면 맞지 않은 문서로 세야 한다: %+v", rep)
		}
		if got := readDoc(t, root, "sources/2026-04-01-talk.md"); strings.Contains(got, "sourced_at") {
			t.Fatalf("sourced_at 값을 지어냈다:\n%s", got)
		}
	})

	t.Run("파일명에 날짜 접두사가 없으면 created 를 못 채우고 남긴다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"sources/talk.md": "---\ntype: source-summary\nartifact_stage: source\n" +
				"status: sourced\nindexable: false\nsourced_at: 2026-04-02\n---\n\n원본입니다.\n",
		})
		rep := run(t, root, load(t, root), Options{Apply: true})
		rs := rep.Documents[0].Remainders
		if len(rs) != 1 || rs[0].Field != "created" {
			t.Fatalf("created 하나가 남은 것으로 보고되어야 한다: %+v", rs)
		}
		if got := readDoc(t, root, "sources/talk.md"); strings.Contains(got, "created:") {
			t.Fatalf("접두사가 없는데 created 를 지어냈다:\n%s", got)
		}
	})
}

// TestRequiredFieldsMatchLint는 migrate 가 채우는 필수 필드 목록과 lint 가
// 요구하는 목록이 어긋나지 않는지 행동으로 대조한다. lint.requiredFields 가
// 패키지 밖으로 나와 있지 않아 목록을 직접 비교할 수 없다. 그래서 필드를
// 전부 뺀 문서를 migrate 로 채운 뒤 lint 를 돌려, 남는 위반이 문서화된
// 예외(채울 진실원이 migrate 밖에 있는 날짜 필드)뿐인지 본다. lint 가 요구
// 필드를 늘리면 이 테스트가 먼저 실패한다.
func TestRequiredFieldsMatchLint(t *testing.T) {
	cases := []struct {
		name     string
		preset   string
		path     string
		expected []string // migrate 가 채우지 못해 lint 가 계속 잡는 필드
	}{
		{"inbox", "preset: education\n", "inbox/2026-01-01-note.md", nil},
		{"context", "preset: education\n", "context/note.md", nil},
		{"archive", "preset: education\n", "archive/note.md", nil},
		{"source 접두사 있음", "preset: education\n", "sources/2026-01-01-note.md", []string{"sourced_at"}},
		{"source 접두사 없음", "preset: education\n", "sources/note.md", []string{"created", "sourced_at"}},
		{"team 프리셋 source", "preset: team\n", "sources/2026-01-01-note.md", []string{"sourced_at"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			top := strings.Split(filepath.ToSlash(tc.path), "/")[0]
			stage, ok := wiki.StageForDir(top)
			if !ok {
				t.Fatalf("단계 디렉토리가 아니다: %s", tc.path)
			}
			root := makeWiki(t, map[string]string{
				"engram.yaml": tc.preset,
				tc.path:       "---\nartifact_stage: " + string(stage) + "\n---\n\n본문입니다.\n",
			})
			cfg := load(t, root)
			run(t, root, cfg, Options{Apply: true})
			res, err := lint.Run(root, cfg)
			if err != nil {
				t.Fatal(err)
			}
			var left []string
			for _, v := range res.Violations {
				if v.Rule != "frontmatter.missing-field" {
					continue
				}
				m := regexp.MustCompile(`필수 필드 ([a-z_]+)가 없습니다`).FindStringSubmatch(v.Message)
				if m == nil {
					t.Fatalf("위반 메시지에서 필드 이름을 못 꺼낸다: %s", v.Message)
				}
				left = append(left, m[1])
			}
			// 같은 줄, 같은 규칙의 위반은 정렬 순서가 정해지지 않으므로
			// 비교 전에 필드 이름순으로 묶는다.
			sort.Strings(left)
			if !reflect.DeepEqual(left, tc.expected) {
				t.Fatalf("migrate 뒤에도 필수 필드가 남는다. expected %v, got %v.\nmigrate 가 다루는 목록과 lint 가 요구하는 목록이 어긋났다.",
					tc.expected, left)
			}
		})
	}
}
