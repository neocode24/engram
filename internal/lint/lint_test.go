package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
)

// writeWiki는 임시 디렉토리에 작은 위키를 만들고 그 루트를 반환한다.
func writeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runLint는 위키를 만들어 검사하고 결과를 반환한다.
func runLint(t *testing.T, files map[string]string) Result {
	t.Helper()
	dir := writeWiki(t, files)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("설정 로드 실패: %v", err)
	}
	res, err := Run(dir, cfg)
	if err != nil {
		t.Fatalf("lint 실행 실패: %v", err)
	}
	return res
}

// findByRule는 해당 규칙의 위반만 모은다.
func findByRule(res Result, rule string) []Violation {
	var out []Violation
	for _, v := range res.Violations {
		if v.Rule == rule {
			out = append(out, v)
		}
	}
	return out
}

// cleanContextDoc는 규칙을 통과하는 context 단계 문서다.
func cleanContextDoc(related, body string) string {
	return "---\n" +
		"type: procedure\n" +
		"artifact_stage: context\n" +
		"status: promoted\n" +
		"indexable: true\n" +
		"tags: []\n" +
		"source_refs: []\n" +
		"derived_from: []\n" +
		"related:\n  - \"[[" + related + "]]\"\n" +
		"source_channel: manual\n" +
		"derived_context: []\n" +
		"---\n\n본문 링크 [[" + body + "]]\n"
}

func TestRun(t *testing.T) {
	t.Run("링크가 서로 이어진 위키는 위반이 없다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "c"),
			"context/b.md": cleanContextDoc("a", "c"),
			"context/c.md": cleanContextDoc("a", "b"),
		})
		if len(res.Violations) != 0 {
			t.Fatalf("위반이 없어야 함: %+v", res.Violations)
		}
		if res.Summary.Files != 3 {
			t.Errorf("검사 파일 수 = %d, want 3", res.Summary.Files)
		}
	})

	t.Run("프론트매터가 없으면 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/plain.md": "# 제목만 있는 문서\n",
		})
		vs := findByRule(res, "frontmatter.missing")
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("frontmatter.missing error가 있어야 함: %+v", res.Violations)
		}
		if vs[0].Fix == "" {
			t.Error("고치는 법이 비어 있으면 안 됨")
		}
	})

	t.Run("닫는 구분자가 없으면 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/unclosed.md": "---\ntype: inbox-note\nartifact_stage: inbox\n",
		})
		vs := findByRule(res, "frontmatter.unclosed")
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("frontmatter.unclosed error가 있어야 함: %+v", res.Violations)
		}
	})

	t.Run("YAML 파싱 실패는 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/bad.md": "---\ntype: [안 닫힌 목록\n---\n",
		})
		vs := findByRule(res, "frontmatter.yaml")
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("frontmatter.yaml error가 있어야 함: %+v", res.Violations)
		}
	})

	t.Run("단계별 필수 필드 누락은 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/no-related.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"source_channel: manual\nderived_context: []\n---\n\n본문\n",
		})
		vs := findByRule(res, "frontmatter.missing-field")
		if len(vs) != 1 || vs[0].Message == "" {
			t.Fatalf("related 누락이 잡혀야 함: %+v", res.Violations)
		}
		if !strings.Contains(vs[0].Message, "related") {
			t.Errorf("메시지에 누락 필드가 있어야 함: %s", vs[0].Message)
		}
	})

	t.Run("artifact_stage가 없으면 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/hole.md": "---\n" +
				"type: concept\nstatus: promoted\n" +
				"---\n\n# 링크 없음\n",
		})
		vs := findByRule(res, "frontmatter.missing-field")
		// 누락 그 자체만 말한다. 어느 단계인지 모르므로 다른 필수 필드는
		// 보고하지 않는다(ADR 0040).
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("artifact_stage 누락이 error 한 건이어야 함: %+v", vs)
		}
		if !strings.Contains(vs[0].Message, "artifact_stage") {
			t.Errorf("메시지에 artifact_stage가 없음: %s", vs[0].Message)
		}
	})

	t.Run("artifact_stage 축이 꺼져 있으면 누락을 보고하지 않는다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml":   "axes:\n  artifact_stage: false\n",
			"inbox/note.md": "---\ntype: inbox-note\nstatus: inbox\nindexable: false\n---\n\n메모\n",
		})
		if got := findByRule(res, "frontmatter.missing-field"); len(got) != 0 {
			t.Fatalf("축이 꺼져 있으면 누락을 보고하지 않음: %+v", got)
		}
	})

	t.Run("context 디렉토리의 inbox 선언 문서에 게이트가 돈다", func(t *testing.T) {
		// 선언을 낮춰 게이트를 우회하던 경로다(ADR 0040).
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "c"),
			"context/b.md": cleanContextDoc("a", "c"),
			"context/c.md": cleanContextDoc("a", "b"),
			"context/underdeclared.md": "---\n" +
				"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel:\n" +
				"---\n\n링크 없는 메모\n",
		})
		gate := findByRule(res, "gate.min-wikilinks")
		if len(gate) != 1 || gate[0].Path != "context/underdeclared.md" || gate[0].Severity != SevReject {
			t.Fatalf("context 디렉토리의 문서가 게이트에 걸려야 함: %+v", gate)
		}
		loc := findByRule(res, "location.stage-agreement")
		if len(loc) != 1 || loc[0].Path != "context/underdeclared.md" || loc[0].Severity != SevWarn {
			t.Fatalf("선언이 낮은 방향은 location warn이어야 함: %+v", loc)
		}
	})

	t.Run("artifact_stage 없는 context 문서에도 게이트가 돈다", func(t *testing.T) {
		// 값을 비워 게이트를 우회하던 경로다(ADR 0040).
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "c"),
			"context/b.md": cleanContextDoc("a", "c"),
			"context/c.md": cleanContextDoc("a", "b"),
			"context/nostage.md": "---\n" +
				"type: concept\nstatus: promoted\n" +
				"---\n\n# 링크 없음\n",
		})
		gate := findByRule(res, "gate.min-wikilinks")
		if len(gate) != 1 || gate[0].Path != "context/nostage.md" {
			t.Fatalf("artifact_stage 가 없어도 게이트가 돌아야 함: %+v", gate)
		}
		if got := findByRule(res, "gate.deferred"); len(got) != 0 {
			t.Fatalf("대상이 충분하면 유예가 아니어야 함: %+v", got)
		}
	})

	t.Run("허용값 밖의 값은 허용값 목록과 함께 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/draft.md": "---\n" +
				"type: inbox-note\nartifact_stage: draft\nstatus: inbox\n" +
				"indexable: false\n---\n\n본문\n",
		})
		vs := findByRule(res, "schema.allowed-value")
		if len(vs) != 1 {
			t.Fatalf("schema.allowed-value가 있어야 함: %+v", res.Violations)
		}
		for _, want := range []string{"artifact_stage", `"draft"`, "inbox, source, context"} {
			if !strings.Contains(vs[0].Message, want) {
				t.Errorf("메시지에 %q 없음: %s", want, vs[0].Message)
			}
		}
	})

	t.Run("context를 선언한 문서가 context 밖에 있으면 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/misplaced.md": "---\n" +
				"type: inbox-note\nartifact_stage: context\nstatus: inbox\nindexable: false\n" +
				"---\n\n메모\n",
		})
		vs := findByRule(res, "location.stage-agreement")
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("선언이 위치보다 높은 방향은 error여야 함: %+v", res.Violations)
		}
		// 메시지는 어느 디렉토리에 있고 무엇이라 적혀 있는지를 낸다.
		for _, want := range []string{"inbox", `"context"`} {
			if !strings.Contains(vs[0].Message, want) {
				t.Errorf("메시지에 %q 없음: %s", want, vs[0].Message)
			}
		}
		// 고치는 법은 둘이므로 둘 다 알린다(ADR 0031).
		if !strings.Contains(vs[0].Fix, "옮기") || !strings.Contains(vs[0].Fix, "artifact_stage") {
			t.Errorf("고치는 법이 파일 이동과 값 수정을 모두 알려야 함: %s", vs[0].Fix)
		}
	})

	t.Run("archive에 있으면서 inbox를 선언하면 warn이다", func(t *testing.T) {
		// 선언이 위치보다 낮은 방향은 게이트를 우회하지 않는다(ADR 0035).
		res := runLint(t, map[string]string{
			"archive/old.md": "---\n" +
				"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n" +
				"---\n\n낡은 메모\n",
		})
		vs := findByRule(res, "location.stage-agreement")
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("선언이 위치보다 낮은 방향은 warn이어야 함: %+v", vs)
		}
	})

	t.Run("sources에 있으면서 inbox를 선언하면 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"sources/s.md": "---\n" +
				"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n" +
				"---\n\n원본\n",
		})
		vs := findByRule(res, "location.stage-agreement")
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("선언이 위치보다 낮은 방향은 warn이어야 함: %+v", vs)
		}
	})

	t.Run("sources 디렉토리의 source 단계 문서는 통과한다", func(t *testing.T) {
		// source만 단계 이름과 디렉토리 이름이 어긋나는 자리다.
		res := runLint(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			"sources/s.md": "---\n" +
				"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
				"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n" +
				"---\n\n원본\n",
		})
		if got := findByRule(res, "location.stage-agreement"); len(got) != 0 {
			t.Fatalf("sources 디렉토리의 source 문서는 통과해야 함: %+v", got)
		}
	})

	t.Run("하위 디렉토리는 최상위 디렉토리 기준으로 판정한다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml":           "min_wikilinks: 0\n",
			"context/sub/nested.md": cleanContextDoc("sub", "sub"),
		})
		if got := findByRule(res, "location.stage-agreement"); len(got) != 0 {
			t.Fatalf("하위 디렉토리는 단계를 바꾸지 않음: %+v", got)
		}
	})

	t.Run("허용값 밖의 단계 값도 위치 검사가 잡는다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/draft.md": "---\n" +
				"type: inbox-note\nartifact_stage: draft\nstatus: inbox\nindexable: false\n" +
				"---\n\n본문\n",
		})
		vs := findByRule(res, "location.stage-agreement")
		// context가 아닌 값의 불일치는 방향과 무관하게 warn이다(ADR 0035).
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("허용값 밖 값도 위치 불일치와 별개로 warn으로 잡혀야 함: %+v", vs)
		}
	})

	t.Run("artifact_stage가 없으면 위치 검사를 건너뛴다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"inbox/nostage.md": "---\ntype: inbox-note\nstatus: inbox\nindexable: false\n---\n\n메모\n",
		})
		if got := findByRule(res, "location.stage-agreement"); len(got) != 0 {
			t.Fatalf("값이 없으면 missing-field가 잡으므로 위치 검사는 하지 않음: %+v", got)
		}
	})

	t.Run("README.md는 frontmatter 검사와 고아 판정에서 빠진다", func(t *testing.T) {
		// README는 순회에서 이미 빠진다. 남아 있었다면 frontmatter.missing
		// 두 건과 고아 판정이 나온다(ADR 0036).
		res := runLint(t, map[string]string{
			"engram.yaml":        "preset: personal\n",
			"context/README.md":  "context 디렉토리 설명입니다\n",
			"inbox/모음/README.md": "모음 디렉토리 설명입니다\n",
		})
		if len(res.Violations) != 0 || res.Summary.Files != 0 {
			t.Fatalf("README가 순회에 남아 있음: %+v (파일 %d)", res.Violations, res.Summary.Files)
		}
	})

	t.Run("꺼진 축의 필드가 있으면 error다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/with-scope.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nscope: work\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[a]]\"\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n본문 [[b]] 링크\n",
		})
		vs := findByRule(res, "schema.axis-off")
		if len(vs) != 1 || !strings.Contains(vs[0].Message, "scope") {
			t.Fatalf("personal 프리셋에서 scope는 꺼진 속성 위반이어야 함: %+v", res.Violations)
		}
	})

	t.Run("forms 폐쇄 집합 위반은 error고 topics 미정의는 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "topics: [go]\nforms: [note]\n",
			"context/f.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[a]]\"\nsource_channel: manual\nderived_context: []\n" +
				"form: memo\ntopics:\n  - go\n  - kubernetes\n" +
				"---\n\n본문 [[b]] 링크\n",
		})
		fv := findByRule(res, "taxonomy.forms")
		if len(fv) != 1 || fv[0].Severity != SevError {
			t.Fatalf("form 위반이 error여야 함: %+v", res.Violations)
		}
		if !strings.Contains(fv[0].Message, "note") {
			t.Errorf("허용값 목록이 메시지에 있어야 함: %s", fv[0].Message)
		}
		tv := findByRule(res, "taxonomy.topics")
		if len(tv) != 1 || tv[0].Severity != SevWarn {
			t.Fatalf("kubernetes 미정의가 warn이어야 함: %+v", res.Violations)
		}
		if !strings.Contains(tv[0].Fix, "topics") {
			t.Errorf("고치는 법에 설정 추가 안내가 있어야 함: %s", tv[0].Fix)
		}
	})

	t.Run("max_lines 초과는 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "max_lines: 5\n",
			"inbox/long.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n---\n" +
				"줄\n줄\n줄\n줄\n줄\n줄\n줄\n",
		})
		vs := findByRule(res, "body.max-lines")
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("max_lines warn이 있어야 함: %+v", res.Violations)
		}
	})

	t.Run("주제가 여러 문서에 걸쳐도 진단은 주제당 1건이다", func(t *testing.T) {
		// 이 테스트의 대상은 주제 진단이므로 게이트는 꺼 둔다.
		res := runLint(t, map[string]string{
			"engram.yaml": "topics: [go]\nbroad_topic_pct: 25\nmin_wikilinks: 0\n",
			"context/a.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[b]]\"\nsource_channel: manual\nderived_context: []\n" +
				"topics:\n  - go\n---\n\n본문 [[b]] 링크\n",
			"context/b.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[a]]\"\nsource_channel: manual\nderived_context: []\n" +
				"topics:\n  - go\n---\n\n본문 [[a]] 링크\n",
		})
		if len(res.WikiFindings) != 1 {
			t.Fatalf("진단이 1건이어야 함: %+v", res.WikiFindings)
		}
		f := res.WikiFindings[0]
		if f.Rule != "wiki.broad-topic" || f.Severity != SevWarn || f.Topic != "go" {
			t.Fatalf("진단 내용이 잘못됨: %+v", f)
		}
		if f.Percent != 100 || f.Total != 2 || f.Threshold != 25 {
			t.Fatalf("비율 정보가 잘못됨: %+v", f)
		}
		if want := []string{"context/a.md", "context/b.md"}; !reflect.DeepEqual(f.Paths, want) {
			t.Errorf("해당 문서 목록 = %v, want %v", f.Paths, want)
		}
		if got := findByRule(res, "wiki.broad-topic"); len(got) != 0 {
			t.Errorf("위키 단위 진단이 파일 위반에 섞였음: %+v", got)
		}
		if res.Summary.Warn != 1 {
			t.Errorf("warn 카운트 = %d, want 1", res.Summary.Warn)
		}
	})

	t.Run("위키 진단은 비율 내림차순, 같으면 주제 이름순이다", func(t *testing.T) {
		doc := func(topics string) string {
			return "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel: manual\ntopics:\n" + topics +
				"---\n\n메모\n"
		}
		res := runLint(t, map[string]string{
			"engram.yaml": "topics: [alpha, beta, gamma]\nbroad_topic_pct: 25\n",
			"inbox/1.md":  doc("  - gamma\n  - alpha\n  - beta\n"),
			"inbox/2.md":  doc("  - gamma\n  - alpha\n  - beta\n"),
			"inbox/3.md":  doc("  - gamma\n  - alpha\n"),
			"inbox/4.md":  doc("  - gamma\n  - alpha\n"),
		})
		// alpha와 gamma는 4/4, beta는 2/4다. 같은 비율은 이름순이다.
		want := []string{"alpha", "gamma", "beta"}
		if len(res.WikiFindings) != len(want) {
			t.Fatalf("진단 %d건이어야 함: %+v", len(want), res.WikiFindings)
		}
		for i, topic := range want {
			if res.WikiFindings[i].Topic != topic {
				t.Fatalf("%d번째 진단 = %q, want %q: %+v", i, res.WikiFindings[i].Topic, topic, res.WikiFindings)
			}
		}
	})

	t.Run("한 문서가 같은 주제를 거듭 쓰면 문서 수로 한 번만 센다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "topics: [go]\nbroad_topic_pct: 25\n",
			"inbox/1.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel: manual\ntopics:\n  - go\n  - go\n" +
				"---\n\n메모\n",
			"inbox/2.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel: manual\n" +
				"---\n\n메모\n",
		})
		if len(res.WikiFindings) != 1 || res.WikiFindings[0].Percent != 50 {
			t.Fatalf("거듭 쓴 주제는 50퍼센트(1/2)로 한 번만 세야 함: %+v", res.WikiFindings)
		}
	})

	t.Run("깨진 위키링크는 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "없는문서"),
			"context/b.md": cleanContextDoc("a", "a"),
		})
		vs := findByRule(res, "link.broken")
		if len(vs) != 1 {
			t.Fatalf("깨진 링크 하나가 잡혀야 함: %+v", res.Violations)
		}
		if !strings.Contains(vs[0].Message, "없는문서") {
			t.Errorf("메시지에 슬러그가 있어야 함: %s", vs[0].Message)
		}
	})

	t.Run("sources 문서에 updated가 있으면 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"sources/s.md": "---\n" +
				"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
				"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\nupdated: 2026-01-03\n" +
				"---\n\n원본\n",
		})
		vs := findByRule(res, "sources.updated")
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("sources.updated warn이 있어야 함: %+v", res.Violations)
		}
	})

	t.Run("링크가 전혀 없는 문서는 고아 warn이다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			"inbox/alone.md": "---\n" +
				"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n" +
				"---\n\n링크 없는 메모\n",
		})
		vs := findByRule(res, "graph.orphan")
		if len(vs) != 1 || vs[0].Severity != SevWarn {
			t.Fatalf("고아 warn이 있어야 함: %+v", res.Violations)
		}
	})

	t.Run("위키링크가 부족한 context 문서는 reject다", func(t *testing.T) {
		// 문서가 3개면 대상이 2개로 min_wikilinks 기본값과 같아 게이트가 동작한다.
		res := runLint(t, map[string]string{
			"context/thin.md": cleanContextDoc("b", "b"),
			"context/b.md":    cleanContextDoc("thin", "c"),
			"context/c.md":    cleanContextDoc("thin", "b"),
		})
		vs := findByRule(res, "gate.min-wikilinks")
		if len(vs) != 1 || vs[0].Severity != SevReject {
			t.Fatalf("게이트 reject가 있어야 함: %+v", res.Violations)
		}
		if res.HasBlocking() != true {
			t.Error("reject는 승급을 막아야 함")
		}
		// b.md는 related와 본문이 서로 다른 슬러그를 가리켜 2개로 통과한다.
		for _, v := range vs {
			if v.Path != "context/thin.md" {
				t.Errorf("b.md도 reject 되었음: %+v", v)
			}
		}
	})

	t.Run("코드 펜스 안의 링크는 게이트와 링크 검사에서 세지 않는다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"context/fence.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[b]]\"\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n" +
				"```\n[[b]]\n```\n" +
				"인라인 `[[b]]` 도 링크가 아니다.\n",
			"context/b.md": cleanContextDoc("a", "fence"),
			"context/a.md": cleanContextDoc("fence", "b"),
		})
		if vs := findByRule(res, "gate.min-wikilinks"); len(vs) == 0 {
			t.Fatalf("펜스 안 [[b]]를 빼면 링크 1개라 reject여야 함: %+v", res.Violations)
		}
	})

	t.Run("min_wikilinks가 0이면 게이트가 돌지 않는다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			"context/nolink.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"source_channel: manual\nderived_context: []\n" +
				"---\n\n링크 없는 문서\n",
		})
		if vs := findByRule(res, "gate.min-wikilinks"); len(vs) != 0 {
			t.Fatalf("게이트가 꺼져 있어야 함: %+v", vs)
		}
		// 게이트가 꺼져 있으면 유예 경고도 내지 않는다.
		if vs := findByRule(res, "gate.deferred"); len(vs) != 0 {
			t.Fatalf("게이트 오프 상태에서 유예 경고가 나오면 안 됨: %+v", vs)
		}
		if res.Summary.Reject != 0 {
			t.Errorf("reject 수 = %d, want 0", res.Summary.Reject)
		}
	})

	t.Run("링크 대상이 부족하면 게이트를 유예하고 경고를 낸다", func(t *testing.T) {
		// 문서 2개. 대상이 1개뿐이라 min_wikilinks 2 를 채울 수 없다.
		res := runLint(t, map[string]string{
			"context/first.md": cleanContextDoc("second", "second"),
			"context/second.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\n" +
				"---\n\n링크 없는 문서\n",
		})
		vs := findByRule(res, "gate.deferred")
		if len(vs) != 2 {
			t.Fatalf("링크가 부족한 문서마다 유예 경고가 나와야 함: %+v", res.Violations)
		}
		for _, v := range vs {
			if v.Severity != SevWarn {
				t.Errorf("유예는 warn 이어야 함: %+v", v)
			}
			if !strings.Contains(v.Message, "유예") || !strings.Contains(v.Message, "동작합니다") {
				t.Errorf("유예 메시지는 원인과 게이트 동작 시점을 알려야 함: %s", v.Message)
			}
		}
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 0 {
			t.Fatalf("유예 중에는 reject 가 나오면 안 됨: %+v", got)
		}
		if res.HasBlocking() {
			t.Error("유예는 승급을 막지 않는다")
		}
	})

	t.Run("문서가 min_wikilinks 만큼 쌓이면 게이트가 동작한다", func(t *testing.T) {
		files := map[string]string{
			"context/thin.md": cleanContextDoc("b", "b"),
			"context/b.md":    cleanContextDoc("thin", "c"),
		}
		// 문서 2개에서는 유예다.
		before := runLint(t, files)
		if got := findByRule(before, "gate.deferred"); len(got) == 0 {
			t.Fatalf("대상 부족 시 유예 경고가 있어야 함: %+v", before.Violations)
		}
		// 세 번째 문서가 생기면 대상이 2개가 되어 게이트가 동작한다.
		files["context/c.md"] = cleanContextDoc("thin", "b")
		after := runLint(t, files)
		if got := findByRule(after, "gate.deferred"); len(got) != 0 {
			t.Fatalf("대상이 충분하면 유예 경고가 남으면 안 됨: %+v", got)
		}
		if got := findByRule(after, "gate.min-wikilinks"); len(got) != 1 {
			t.Fatalf("게이트가 동작해 링크 1개 문서가 reject 여야 함: %+v", after.Violations)
		}
	})

	t.Run("inbox 문서만 있는 위키에서는 게이트가 유예된다", func(t *testing.T) {
		// 문서 3개지만 링크 대상은 0개다. inbox 문서는 promote 되면
		// 슬러그가 바뀌어 링크가 깨지므로 대상이 아니다(ADR 0022).
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "b"),
			"inbox/b.md":   "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel: manual\n---\n\n메모\n",
			"inbox/c.md":   "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel: manual\n---\n\n메모\n",
		})
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 0 {
			t.Fatalf("inbox 문서를 대상으로 세면 게이트가 동작해 유예가 사라진다: %+v", got)
		}
		vs := findByRule(res, "gate.deferred")
		if len(vs) != 1 || !strings.Contains(vs[0].Message, "대상 문서가 0개") {
			t.Fatalf("대상 0개로 유예되어야 함: %+v", vs)
		}
	})

	t.Run("sources 문서는 링크 대상에 포함된다", func(t *testing.T) {
		// sources 는 promote 가 옮기지 않고 파생을 만들므로 그 자리에 남는다.
		source := "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
			"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n---\n\n원본\n"
		res := runLint(t, map[string]string{
			"context/a.md":  cleanContextDoc("b", "b"),
			"sources/s1.md": source,
			"sources/s2.md": source,
		})
		if got := findByRule(res, "gate.deferred"); len(got) != 0 {
			t.Fatalf("대상이 2개면 유예되면 안 됨: %+v", got)
		}
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 1 {
			t.Fatalf("대상 2개로 게이트가 동작해 reject 여야 함: %+v", res.Violations)
		}
	})

	t.Run("단계를 읽을 수 없는 문서는 링크 대상에서 빠진다", func(t *testing.T) {
		// 프론트매터 없는 문서를 포함해서 세면 대상이 2가 되어 게이트가
		// 동작한다. 대상 1개로 유예되는 것이 올바르다.
		res := runLint(t, map[string]string{
			"context/a.md": cleanContextDoc("b", "b"),
			"sources/s.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
				"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n---\n\n원본\n",
			"context/plain.md": "프론트매터 없는 문서\n",
		})
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 0 {
			t.Fatalf("단계를 모르는 문서를 대상으로 세면 안 됨: %+v", got)
		}
		vs := findByRule(res, "gate.deferred")
		if len(vs) != 1 || !strings.Contains(vs[0].Message, "대상 문서가 1개") {
			t.Fatalf("대상 1개로 유예되어야 함: %+v", vs)
		}
	})

	t.Run("관계 필드가 있는 문서는 고아가 아니다", func(t *testing.T) {
		// 위키링크 없이 관계 필드만 있는 문서 셋. 각각 고아가 아니어야 한다.
		res := runLint(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			// derived_context 만 있다.
			"sources/only-context.md": "---\n" +
				"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\n" +
				"derived_context:\n  - 파생문서\n" +
				"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n" +
				"---\n\n원본\n",
			// derived_from 만 있다.
			"context/only-from.md": "---\n" +
				"type: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\n" +
				"derived_from:\n  - sources/2026-02-원본.md\n" +
				"related: []\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n본문\n",
			// source_refs 만 있다.
			"inbox/only-refs.md": "---\n" +
				"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\n" +
				"source_refs:\n  - https://example.com/a\n" +
				"---\n\n메모\n",
		})
		vs := findByRule(res, "graph.orphan")
		for _, v := range vs {
			t.Errorf("관계 필드가 있는 문서가 고아로 판정됨: %+v", v)
		}
	})

	t.Run("관계 필드 경로 값은 날짜 접두사를 떼고 슬러그로 비교한다", func(t *testing.T) {
		// 원본은 위키링크가 없다. 파생 문서의 derived_from 이 경로로 이
		// 문서를 가리키므로 고아가 아니다. promote 가 문서를 옮기며 날짜
		// 접두사를 떼기 때문에 접두사를 정규화해야 연결된다(ADR 0022).
		res := runLint(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			"sources/2026-02-원본.md": "---\n" +
				"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
				"source_channel: web\ncreated: 2026-02-01\nsourced_at: 2026-02-02\n" +
				"---\n\n원본\n",
			"context/파생문서.md": "---\n" +
				"type: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\n" +
				"derived_from:\n  - sources/2026-02-원본.md\n" +
				"related: []\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n본문\n",
		})
		vs := findByRule(res, "graph.orphan")
		for _, v := range vs {
			if v.Path == "sources/2026-02-원본.md" {
				t.Errorf("경로 관계로 연결된 원본이 고아로 판정됨: %+v", v)
			}
		}
	})

	t.Run("관계 필드가 있어도 게이트는 위키링크만 센다", func(t *testing.T) {
		// 대상 2개(다른 context 문서 2개)로 게이트가 동작하는 상태에서
		// 관계 필드만 있고 위키링크가 없는 문서는 게이트에 걸린다.
		// 관계 필드는 도구가 채우므로 게이트에 넣으면 게이트가 무력해진다.
		res := runLint(t, map[string]string{
			"context/a.md": "---\n" +
				"type: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\n" +
				"derived_from:\n  - sources/2026-02-원본.md\n" +
				"related: []\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n본문\n",
			"context/b.md": cleanContextDoc("a", "c"),
			"context/c.md": cleanContextDoc("a", "b"),
		})
		if got := findByRule(res, "graph.orphan"); len(got) != 0 {
			t.Fatalf("관계 필드가 있는 문서가 고아로 판정됨: %+v", got)
		}
		vs := findByRule(res, "gate.min-wikilinks")
		if len(vs) != 1 || vs[0].Path != "context/a.md" {
			t.Fatalf("위키링크 없는 문서는 게이트 reject 여야 함: %+v", vs)
		}
	})

	t.Run("EvaluateGate 는 자기 자신을 대상 수에서 뺀다", func(t *testing.T) {
		// 문서가 자신 하나뿐이면 대상 0개. 유예다.
		g := EvaluateGate(2, 0, 2)
		if !g.Deferred || !g.Passed {
			t.Errorf("대상 0이면 유예 통과여야 함: %+v", g)
		}
		// 대상이 min 과 같아지면 유예 없이 판정한다.
		g = EvaluateGate(1, 2, 2)
		if g.Deferred || g.Passed {
			t.Errorf("대상 2, 링크 1이면 유예 없이 거절이어야 함: %+v", g)
		}
		g = EvaluateGate(2, 2, 2)
		if g.Deferred || !g.Passed {
			t.Errorf("대상 2, 링크 2면 통과여야 함: %+v", g)
		}
		// min_wikilinks 0 은 게이트 오프. 유예 표시도 없다.
		g = EvaluateGate(0, 0, 0)
		if !g.Passed || g.Deferred {
			t.Errorf("게이트 오프는 유예 없이 통과여야 함: %+v", g)
		}
	})

	t.Run("같은 위키를 두 번 검사하면 결과가 같다", func(t *testing.T) {
		files := map[string]string{
			"engram.yaml": "topics: [go]\nforms: [note]\n",
			"context/a.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[없는문서]]\"\nsource_channel: manual\nderived_context: []\n" +
				"form: memo\ntopics:\n  - go\n---\n\n본문\n",
			"inbox/b.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: weird\nindexable: false\nscope: work\n---\n\n메모\n",
		}
		first := runLint(t, files)
		second := runLint(t, files)
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("두 실행 결과가 다름:\n%+v\n%+v", first, second)
		}
	})

	t.Run("위반은 경로, 줄, 규칙 ID 순으로 정렬된다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "topics: [go]\nforms: [note]\n",
			"context/a.md": "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nscope: work\nsource_refs: []\nderived_from: []\n" +
				"related:\n  - \"[[b]]\"\nsource_channel: manual\nderived_context: []\n" +
				"form: memo\n---\n\n본문 [[b]]\n",
			"context/b.md": cleanContextDoc("a", "a"),
		})
		for i := 1; i < len(res.Violations); i++ {
			prev, cur := res.Violations[i-1], res.Violations[i]
			if prev.Path > cur.Path ||
				(prev.Path == cur.Path && prev.Line > cur.Line) ||
				(prev.Path == cur.Path && prev.Line == cur.Line && prev.Rule > cur.Rule) {
				t.Fatalf("정렬 깨짐: %+v 다음 %+v", prev, cur)
			}
		}
	})

	// indexDoc는 링크가 없는 색인 문서다. init 직후 위키의 초기 상태를 재현한다.
	indexDoc := "---\n" +
		"type: system\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
		"source_channel: manual\nderived_context: []\n" +
		"---\n\n# 위키 색인\n"

	t.Run("색인 문서는 게이트와 고아 판정에서 빠진다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"index.md": indexDoc,
		})
		if len(res.Violations) != 0 {
			t.Fatalf("링크 없는 색인만 있는 위키는 위반이 없어야 함: %+v", res.Violations)
		}
		if res.HasBlocking() {
			t.Error("색인만 있는 위키가 승급을 막는 판정을 받으면 안 됨")
		}
	})

	t.Run("색인 문서라도 스키마 위반은 잡힌다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "forms: [note]\n",
			"index.md": "---\n" +
				"type: system\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\nform: memo\n" +
				"---\n\n# 위키 색인\n",
		})
		vs := findByRule(res, "taxonomy.forms")
		if len(vs) != 1 || vs[0].Severity != SevError {
			t.Fatalf("색인의 스키마 위반이 잡혀야 함: %+v", res.Violations)
		}
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 0 {
			t.Errorf("스키마 위반이 있어도 게이트는 색인을 보지 않음: %+v", got)
		}
	})

	t.Run("root_files를 바꾸면 그 파일이 색인으로 제외된다", func(t *testing.T) {
		res := runLint(t, map[string]string{
			"engram.yaml": "root_files: [home.md]\n",
			"home.md":     indexDoc,
		})
		if len(res.Violations) != 0 {
			t.Fatalf("root_files로 지정한 색인은 게이트와 고아에서 빠져야 함: %+v", res.Violations)
		}
		// 제외는 root_files 소속 여부로 판정한다. page_dirs 안의 문서는
		// 이름이 무엇이든 게이트와 고아 판정 대상이다.
		// 문서 3개로 대상을 min_wikilinks 만큼 확보해 게이트가 동작하게 한다.
		res = runLint(t, map[string]string{
			"engram.yaml":  "root_files: [home.md]\n",
			"home.md":      indexDoc,
			"context/x.md": indexDoc,
			// y 와 z 가 서로를 가리켜 링크 수를 채운다. x 가 고아 판정에
			// 남도록 x 로는 들어오는 링크를 만들지 않는다. 실재하지 않는
			// 슬러그로 채우는 방법은 게이트가 막는다(ADR 0054).
			"context/y.md": cleanContextDoc("home", "z"),
			"context/z.md": cleanContextDoc("home", "y"),
		})
		if got := findByRule(res, "gate.min-wikilinks"); len(got) != 1 {
			t.Fatalf("page_dirs 안의 색인형 문서는 게이트 대상이어야 함: %+v", res.Violations)
		}
		if got := findByRule(res, "graph.orphan"); len(got) != 1 {
			t.Fatalf("page_dirs 안의 색인형 문서는 고아 판정 대상이어야 함: %+v", res.Violations)
		}
	})

	t.Run("색인 문서는 위치 검사에서 빠진다", func(t *testing.T) {
		// 색인은 위키 루트에 있어 비교할 디렉토리가 없다(ADR 0019).
		res := runLint(t, map[string]string{
			"index.md": indexDoc,
		})
		if got := findByRule(res, "location.stage-agreement"); len(got) != 0 {
			t.Fatalf("root_files는 위치 검사 대상이 아님: %+v", got)
		}
	})
}
