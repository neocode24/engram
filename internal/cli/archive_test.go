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

// runArchive는 archive 커맨드를 루트 등록 없이 시험한다.
// 전역 플래그는 실제 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runArchive(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newArchiveCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeArchiveWiki는 링크로 이어진 context 문서 둘을 갖춘 위키를 만든다.
// archive 디렉토리는 만들지 않는다. 커맨드가 스스로 만드는지 함께 본다.
func makeArchiveWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: education\n",
		"context/hub.md": "---\n" +
			"type: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated:\n  - \"[[peer]]\"\n" +
			"created: 2026-01-01\nupdated: 2026-07-01\n" +
			"---\n\n# 허브\n\n모든 링크가 지나가는 문서입니다.\n",
		"context/peer.md": "---\n" +
			"type: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated: []\n" +
			"---\n\n# 피어\n\n[[hub]] 를 가리킵니다.\n",
		"context/solo.md": "---\n" +
			"type: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nrelated: []\n" +
			"---\n\n# 외톨이\n\n어디서도 링크하지 않는 문서입니다.\n",
		"inbox/2026-07-15-memo.md": "---\n" +
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\n---\n\n아직 승급 전입니다.\n",
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

func TestArchiveCmd(t *testing.T) {
	t.Run("context 문서를 슬러그 그대로 보관하고 프론트매터를 바꿉니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "context/hub.md")
		if err != nil {
			t.Fatalf("archive 실패: %v\n%s", err, out)
		}
		archived := filepath.Join(wiki, "archive", "hub.md")
		raw, err := os.ReadFile(archived)
		if err != nil {
			t.Fatalf("보관 파일이 없음: %v", err)
		}
		if _, err := os.Stat(filepath.Join(wiki, "context", "hub.md")); !os.IsNotExist(err) {
			t.Error("원본이 남아 있음")
		}
		text := string(raw)
		for _, want := range []string{"artifact_stage: archive", "status: archived", "updated: 2026-08-16"} {
			if !strings.Contains(text, want) {
				t.Errorf("보관 파일에 %q 없음:\n%s", want, text)
			}
		}
		// 본문과 그 외 필드는 그대로다.
		for _, want := range []string{"# 허브", "type: concept", "related:", "created: 2026-01-01"} {
			if !strings.Contains(text, want) {
				t.Errorf("보관 파일에 %q 없음:\n%s", want, text)
			}
		}
		if !strings.Contains(out, "보관했습니다") || !strings.Contains(out, "링크 1개") {
			t.Errorf("결과 안내가 잘못됨:\n%s", out)
		}
		// 링크를 건 문서는 그대로다. 슬러그가 유지되므로 링크도 유효하다.
		if _, err := os.Stat(filepath.Join(wiki, "context", "peer.md")); err != nil {
			t.Errorf("링크를 건 문서가 영향을 받음: %v", err)
		}
	})

	t.Run("--json 은 경로와 슬러그와 링크 수를 냅니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		out, err := runArchive(t, "archive", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "context/hub.md")
		if err != nil {
			t.Fatalf("archive 실패: %v\n%s", err, out)
		}
		var res archiveOutcome
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Slug != "hub" || res.IncomingLinks != 1 {
			t.Errorf("결과가 잘못됨: %+v", res)
		}
		if !strings.HasSuffix(filepath.ToSlash(res.Path), "archive/hub.md") {
			t.Errorf("경로가 잘못됨: %s", res.Path)
		}
	})

	t.Run("context 단계가 아니면 거절합니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "inbox/2026-07-15-memo.md")
		if err == nil {
			t.Fatal("거절되어야 합니다")
		}
		if !strings.Contains(out, "context 단계 문서만") || !strings.Contains(out, "engram demote") {
			t.Errorf("거절 안내가 잘못됨:\n%s", out)
		}
	})

	t.Run("이미 보관된 문서는 거절합니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		if _, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "context/hub.md"); err != nil {
			t.Fatal(err)
		}
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "archive/hub.md")
		if err == nil || !strings.Contains(out, "이미 보관 상태입니다") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
	})

	t.Run("도착지에 같은 이름이 있으면 거절합니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		dest := filepath.Join(wiki, "archive", "hub.md")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte("이미 있습니다\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "context/hub.md")
		if err == nil || !strings.Contains(out, "도착지에 이미 문서가 있습니다") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
		if raw, err := os.ReadFile(dest); err != nil || string(raw) != "이미 있습니다\n" {
			t.Errorf("기존 문서를 덮어썼음: %v %q", err, raw)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init 을 안내합니다", func(t *testing.T) {
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", t.TempDir(), "context/hub.md")
		if err == nil || !strings.Contains(out, "engram init") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
	})

	t.Run("링크가 없으면 링크 안내를 내지 않습니다", func(t *testing.T) {
		wiki := makeArchiveWiki(t)
		out, err := runArchive(t, "archive", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki, "context/solo.md")
		if err != nil {
			t.Fatalf("archive 실패: %v\n%s", err, out)
		}
		// 개수를 세는 안내만 없으면 된다. 링크 유지 원칙 안내는 늘 나온다.
		if strings.Contains(out, "개가") {
			t.Errorf("들어오는 링크가 없는데 개수 안내가 있음:\n%s", out)
		}
	})
}
