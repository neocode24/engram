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

// runDemoteUpdate는 demote와 update 커맨드를 루트 등록 없이 시험한다.
// 전역 플래그는 실제 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runDemoteUpdate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력한다")
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
	parent.AddCommand(newDemoteCmd())
	parent.AddCommand(newUpdateCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// contextDoc는 승급이 끝난 context 문서 형태를 만든다.
func contextDocFM(created, updated string) string {
	fm := "type: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
		"related:\n  - \"[[hub]]\"\nsource_channel: manual\nderived_context: []\n" +
		"topics:\n  - go\n"
	if created != "" {
		fm += "created: " + created + "\n"
	}
	if updated != "" {
		fm += "updated: " + updated + "\n"
	}
	return "---\n" + fm + "---\n\n본문 [[hub]] 링크\n"
}

// makeDemoteWiki는 context 문서 하나와 색인을 갖춘 위키를 만든다.
func makeDemoteWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml":     "preset: education\n",
		"index.md":        "---\ntype: system\nartifact_stage: context\nstatus: promoted\nindexable: true\nsource_refs: []\nderived_from: []\nrelated: []\nsource_channel: manual\nderived_context: []\n---\n\n# 색인\n",
		"context/note.md": contextDocFM("2026-01-15", "2026-02-01"),
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

func readWiki(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDemoteCmd(t *testing.T) {
	t.Run("기본 도착지는 inbox 다. created 날짜 접두사를 붙인다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "context/note.md")
		if err != nil {
			t.Fatalf("demote 실패: %v\n%s", err, out)
		}
		content := readWiki(t, root, "inbox/2026-01-15-note.md")
		for _, want := range []string{"artifact_stage: inbox", "status: inbox", "indexable: false"} {
			if !strings.Contains(content, want) {
				t.Errorf("프론트매터에 %q 없음:\n%s", want, content)
			}
		}
		if _, err := os.Stat(filepath.Join(root, "context", "note.md")); !os.IsNotExist(err) {
			t.Error("원본 context 문서가 남아 있다")
		}
		if !strings.Contains(out, "inbox로 내렸다") {
			t.Errorf("출력에 안내 없음:\n%s", out)
		}
	})

	t.Run("--to sources 로 내린다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "--to", "sources", "context/note.md")
		if err != nil {
			t.Fatalf("demote 실패: %v\n%s", err, out)
		}
		content := readWiki(t, root, "sources/2026-01-15-note.md")
		for _, want := range []string{"artifact_stage: source", "status: sourced"} {
			if !strings.Contains(content, want) {
				t.Errorf("프론트매터에 %q 없음:\n%s", want, content)
			}
		}
	})

	t.Run("created 가 없으면 기준 시각 날짜를 쓴다", func(t *testing.T) {
		root := t.TempDir()
		files := map[string]string{
			"engram.yaml":       "preset: education\n",
			"context/nodate.md": strings.Replace(contextDocFM("", ""), "created: \n", "", 1),
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
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "--now", "2026-08-15T12:00:00Z", "context/nodate.md")
		if err != nil {
			t.Fatalf("demote 실패: %v\n%s", err, out)
		}
		readWiki(t, root, "inbox/2026-08-15-nodate.md")
	})

	t.Run("context 아닌 문서는 거절한다", func(t *testing.T) {
		root := t.TempDir()
		inbox := "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel: manual\n---\n\n메모\n"
		if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte("preset: education\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "inbox", "memo.md"), []byte(inbox), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "inbox/memo.md")
		if err == nil {
			t.Fatal("context 아닌 문서는 거절이어야 한다")
		}
		if !strings.Contains(out, "context 단계 문서만") {
			t.Errorf("거절 안내가 없음: %s", out)
		}
	})

	t.Run("깨질 링크를 출처와 줄 번호로 경고한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		// 색인이 note 를 본문에서 가리키게 한다.
		linked := "---\ntype: system\nartifact_stage: context\nstatus: promoted\nindexable: true\n" +
			"source_refs: []\nderived_from: []\nrelated: []\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n# 색인\n\n[[note]] 문서를 본다\n"
		if err := os.WriteFile(filepath.Join(root, "index.md"), []byte(linked), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "context/note.md")
		if err != nil {
			t.Fatalf("경고는 진행을 막지 않는다: %v\n%s", err, out)
		}
		if !strings.Contains(out, "깨질 위키링크") || !strings.Contains(out, "index.md") {
			t.Errorf("깨질 링크 경고에 출처가 없음:\n%s", out)
		}
		if !strings.Contains(out, "engram mv") {
			t.Errorf("링크 고치기 안내가 없음:\n%s", out)
		}
	})

	t.Run("derived_from 이 있으면 원본을 경고로 알린다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		derived := strings.Replace(contextDocFM("2026-01-15", ""),
			"derived_from: []", "derived_from:\n  - sources/2026-01-10-note.md", 1)
		if err := os.WriteFile(filepath.Join(root, "context", "note.md"), []byte(derived), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runDemoteUpdate(t, "demote", "--wiki", root, "context/note.md")
		if err != nil {
			t.Fatalf("경고는 진행을 막지 않는다: %v\n%s", err, out)
		}
		if !strings.Contains(out, "sources/2026-01-10-note.md") || !strings.Contains(out, "derived_context") {
			t.Errorf("파생 원본 경고가 없음:\n%s", out)
		}
	})

	t.Run("--json 은 결과를 구조화해 낸다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "demote", "--json", "--wiki", root, "context/note.md")
		if err != nil {
			t.Fatalf("demote 실패: %v\n%s", err, out)
		}
		var res demoteOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Slug != "note" || res.Stage != "inbox" || res.Date != "2026-01-15" {
			t.Errorf("결과가 틀리다: %+v", res)
		}
	})

	t.Run("위키가 아니면 거절한다", func(t *testing.T) {
		out, err := runDemoteUpdate(t, "demote", "--wiki", t.TempDir(), "context/x.md")
		if err == nil {
			t.Fatal("거절이어야 한다")
		}
		if !strings.Contains(out, "engram init") {
			t.Errorf("init 안내가 없음: %s", out)
		}
	})
}
