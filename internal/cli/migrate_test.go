package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runMigrate는 migrate 커맨드를 루트 등록 없이 시험한다.
// 전역 플래그는 실제 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runMigrate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		raw, err := cmd.Flags().GetString(flagNow)
		if err != nil {
			return err
		}
		parsed, err := parseNow(raw)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), nowKey{}, parsed))
		return nil
	}
	parent.AddCommand(newMigrateCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeMigrateWiki는 규칙과 어긋난 문서를 여럿 둔 위키를 만든다.
// 단계 불일치(inbox 와 archive), 필수 필드 누락(context 세 문서 중 하나),
// 소스 문서의 날짜 보존이 함께 걸린다.
func makeMigrateWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: education\n",
		"context/a.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated:\n  - \"[[b]]\"\n  - \"[[c]]\"\n" +
			"---\n\na 문서입니다.\n",
		"context/b.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated:\n  - \"[[a]]\"\n  - \"[[c]]\"\n" +
			"---\n\nb 문서입니다.\n",
		"context/c.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated:\n  - \"[[a]]\"\n  - \"[[b]]\"\n" +
			"---\n\nc 문서입니다.\n",
		"inbox/2026-03-01-memo.md": "---\ntype: concept\nartifact_stage: context\n" +
			"status: promoted\nindexable: true\nrelated: []\n---\n\n게이트를 지나지 않은 문서입니다.\n",
		"archive/2026-01-01-old.md": "---\ntype: concept\nartifact_stage: inbox\n" +
			"status: inbox\nindexable: false\n---\n\n보관된 문서입니다.\n",
		"sources/2026-02-01-talk.md": "---\ntype: source-summary\nartifact_stage: inbox\n" +
			"status: inbox\nindexable: false\nsourced_at: 2026-02-02\n" +
			"---\n\n원본 문서입니다.\n",
	}
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

func TestMigrateCmd(t *testing.T) {
	t.Run("기본은 시험 실행이라 파일을 쓰지 않습니다", func(t *testing.T) {
		root := makeMigrateWiki(t)
		archiveDoc := filepath.Join(root, "archive", "2026-01-01-old.md")
		before, err := os.ReadFile(archiveDoc)
		if err != nil {
			t.Fatal(err)
		}
		out, err := runMigrate(t, "migrate", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "시험 실행이므로 파일을 쓰지 않았습니다") {
			t.Errorf("시험 실행 안내가 없습니다:\n%s", out)
		}
		after, err := os.ReadFile(archiveDoc)
		if err != nil {
			t.Fatal(err)
		}
		if string(before) != string(after) {
			t.Errorf("시험 실행이 파일을 바꿨습니다:\n%s", after)
		}
	})

	t.Run("적용 뒤 lint 의 단계 불일치와 필수 필드 누락이 0이 됩니다", func(t *testing.T) {
		root := makeMigrateWiki(t)
		out, err := runMigrate(t, "migrate", "--apply", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 적용 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "고쳤습니다") {
			t.Errorf("적용 요약이 없습니다:\n%s", out)
		}
		// 강등 방향의 변경이 문서 이동이 아님을 확인한다.
		if _, err := os.Stat(filepath.Join(root, "context", "2026-03-01-memo.md")); err == nil {
			t.Fatal("inbox 문서가 context 로 옮겨졌습니다")
		}
		memo, err := os.ReadFile(filepath.Join(root, "inbox", "2026-03-01-memo.md"))
		if err != nil || !strings.Contains(string(memo), "artifact_stage: inbox") {
			t.Fatalf("선언이 inbox 로 내려가지 않았습니다: %v\n%s", err, memo)
		}

		// 이 커맨드의 존재 이유다. 적용 뒤 두 규칙이 사라져야 한다.
		lintOut, err := runRoot(t, "lint", "--json", root)
		if err != nil {
			t.Fatalf("lint 실행 실패: %v\n%s", err, out)
		}
		var res struct {
			Violations []struct {
				Rule string `json:"rule"`
			} `json:"violations"`
		}
		if err := json.Unmarshal([]byte(lintOut), &res); err != nil {
			t.Fatalf("lint --json 파싱 실패: %v\n%s", err, lintOut)
		}
		for _, v := range res.Violations {
			if v.Rule == "location.stage-agreement" || v.Rule == "frontmatter.missing-field" {
				t.Fatalf("적용 뒤에도 %s 위반이 남았습니다: %+v", v.Rule, res.Violations)
			}
		}
	})

	t.Run("JSON 출력은 보고서를 그대로 실웁니다", func(t *testing.T) {
		root := makeMigrateWiki(t)
		out, err := runMigrate(t, "migrate", "--json", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 실패: %v\n%s", err, out)
		}
		var rep struct {
			Applied bool `json:"applied"`
			Docs    int  `json:"docs"`
			Changed int  `json:"changed"`
			Written int  `json:"written"`
		}
		if err := json.Unmarshal([]byte(out), &rep); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if rep.Applied || rep.Written != 0 {
			t.Errorf("시험 실행 JSON 이 applied 나 written 를 담으면 안 됩니다: %+v", rep)
		}
		if rep.Docs != 6 || rep.Changed == 0 {
			t.Errorf("문서 수와 변경 수가 옳지 않습니다: %+v", rep)
		}
	})

	t.Run("값 있는 꺼진 축 필드는 안내와 함께 보류합니다", func(t *testing.T) {
		root := t.TempDir()
		doc := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated: []\nsensitivity: internal\n---\n\n본문입니다.\n"
		if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte("preset: education\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, "context", "a.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runMigrate(t, "migrate", "--apply", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "--force를 주세요") {
			t.Errorf("보류 안내가 없습니다:\n%s", out)
		}
		got, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(got), "sensitivity: internal") {
			t.Fatalf("값 있는 필드가 force 없이 지워졌습니다:\n%s", got)
		}
	})

	t.Run("적용 뒤 lint 의 필수 필드 누락이 늘지 않습니다", func(t *testing.T) {
		// 이번 결함의 재현이다. sources 아래 inbox 선언 문서를 단계만
		// source 로 고치고 그 단계가 요구하는 필드를 채우지 않으면
		// 위반이 늘어난다. 다른 문서의 개선이 이 증가를 덮지 못하게
		// 고장 시나리오만 담은 위키에서 잰다.
		root := t.TempDir()
		files := map[string]string{
			"engram.yaml": "preset: education\n",
			"context/a.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\nrelated:\n  - \"[[b]]\"\n" +
				"source_channel:\nderived_context: []\n---\n\na 문서입니다.\n",
			"context/b.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\nrelated:\n  - \"[[a]]\"\n" +
				"source_channel:\nderived_context: []\n---\n\nb 문서입니다.\n",
			"sources/2026-02-01-talk.md": "---\ntype: inbox-note\nartifact_stage: inbox\n" +
				"status: inbox\nindexable: false\nsource_channel:\nsourced_at: 2026-02-02\n" +
				"---\n\n원본 문서입니다.\n",
		}
		for name, content := range files {
			p := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		before := countMissingField(t, root)
		if before != 0 {
			t.Fatalf("기준 상태에서 필수 필드 누락이 이미 있습니다: %d", before)
		}
		out, err := runMigrate(t, "migrate", "--apply", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 적용 실패: %v\n%s", err, out)
		}
		if after := countMissingField(t, root); after > before {
			t.Fatalf("적용 뒤 필수 필드 누락이 늘었습니다. before %d, after %d\n%s", before, after, out)
		}
	})

	t.Run("채우지 못한 필드가 남으면 보고가 모두 맞았다고 말하지 않습니다", func(t *testing.T) {
		root := t.TempDir()
		files := map[string]string{
			"engram.yaml": "preset: education\n",
			"sources/2026-04-01-talk.md": "---\ntype: source-summary\nartifact_stage: source\n" +
				"status: sourced\nindexable: false\ncreated: 2026-04-01\n" +
				"tags: []\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel:\nderived_context: []\n---\n\n원본입니다.\n",
		}
		for name, content := range files {
			p := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		out, err := runMigrate(t, "migrate", "--apply", "--wiki", root)
		if err != nil {
			t.Fatalf("migrate 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "1개가 규칙에 맞지 않") || !strings.Contains(out, "완전히 맞추지 못") {
			t.Errorf("남은 필드가 있는데 보고가 정직하지 않습니다:\n%s", out)
		}
		if !strings.Contains(out, "sourced_at") || !strings.Contains(out, "engram sync가 채웁니다") {
			t.Errorf("남은 필드와 안내가 없습니다:\n%s", out)
		}
		if strings.Contains(out, "규칙에 맞습니다.\n") {
			t.Errorf("남은 것이 있는데 모두 맞았다고 말합니다:\n%s", out)
		}
	})
}

// countMissingField는 위키의 lint 위반 중 필수 필드 누락 수를 센다.
// lint 는 위반이 있으면 종료 코드 1로 끝난다. 세는 것이 목적이므로 오류는
// 무시하고 출력만 본다.
func countMissingField(t *testing.T, root string) int {
	t.Helper()
	out, _ := runRoot(t, "lint", "--json", root)
	var res struct {
		Violations []struct {
			Rule string `json:"rule"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("lint --json 파싱 실패: %v\n%s", err, out)
	}
	n := 0
	for _, v := range res.Violations {
		if v.Rule == "frontmatter.missing-field" {
			n++
		}
	}
	return n
}
