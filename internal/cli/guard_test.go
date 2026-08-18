package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guardInboxDoc는 검사 대상 inbox 문서를 위키에 직접 쓴다. capture 가
// 만드는 초기값이 아니라 검사가 보는 값을 정확히 두기 위해서다.
func guardInboxDoc(t *testing.T, dir, name, frontmatter, body string) string {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\n"+frontmatter+"---\n\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	return name
}

// makeTeamWiki는 sensitivity 축이 켜진 team 프리셋 위키를 만든다.
func makeTeamWiki(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wiki")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "engram.yaml"), []byte("preset: team\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	addContextDocs(t, dir)
	return dir
}

func TestPromoteGuards(t *testing.T) {
	t.Run("민감도 restricted 문서의 context 승급을 거절합니다", func(t *testing.T) {
		dir := makeTeamWiki(t)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-민감.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n"+
				"indexable: false\nscope: unknown\nsensitivity: restricted\n"+
				"source_channel: manual\ntrigger_mode: \nworkflow: \nrelated: []\n",
			"# 민감 메모\n\n내용입니다.\n")
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err == nil || !strings.Contains(err.Error(), "restricted") {
			t.Fatalf("거절 오류 = %v", err)
		}
		if _, e := os.Stat(filepath.Join(dir, "context", "민감.md")); !os.IsNotExist(e) {
			t.Error("거절했는데 context 문서가 생겼음")
		}
	})

	t.Run("민감도 private-local-only 문서의 context 승급을 거절합니다", func(t *testing.T) {
		dir := makeTeamWiki(t)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-로컬.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n"+
				"indexable: false\nscope: unknown\nsensitivity: private-local-only\n"+
				"source_channel: manual\ntrigger_mode: \nworkflow: \nrelated: []\n",
			"# 로컬 메모\n\n내용입니다.\n")
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err == nil || !strings.Contains(err.Error(), "private-local-only") {
			t.Fatalf("거절 오류 = %v", err)
		}
	})

	t.Run("--to sources는 민감도를 보지 않습니다", func(t *testing.T) {
		dir := makeTeamWiki(t)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-민감.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n"+
				"indexable: false\nscope: unknown\nsensitivity: restricted\n"+
				"source_channel: manual\ntrigger_mode: \nworkflow: \nrelated: []\n",
			"# 민감 메모\n\n내용입니다.\n")
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, "--to", "sources",
			"--now", "2026-08-01T00:00:00Z", rel)
		if err != nil {
			t.Fatalf("--to sources 승급 실패: %v\n%s", err, out)
		}
		if _, e := os.Stat(filepath.Join(dir, "sources", "2026-08-01-민감.md")); e != nil {
			t.Errorf("sources 문서가 없음: %v", e)
		}
	})

	t.Run("sensitivity 축이 꺼진 위키에서는 민감도로 막지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		addContextDocs(t, dir)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-민감.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n"+
				"source_channel: manual\nderived_context: []\nrelated: []\n"+
				"sensitivity: restricted\n",
			"# 민감 메모\n\n내용입니다.\n")
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, "--related", "a-doc,b-doc", rel)
		if err != nil {
			t.Fatalf("축이 꺼졌으므로 통과해야 함: %v\n%s", err, out)
		}
	})

	t.Run("시크릿이 든 문서의 context 승급을 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-열쇠.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n"+
				"source_channel: manual\nderived_context: []\nrelated: []\n",
			"# 열쇠 메모\n\n토큰은 ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD 입니다.\n")
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err == nil || !strings.Contains(err.Error(), "github-token") {
			t.Fatalf("거절 오류 = %v", err)
		}
		// 값 자체는 거절 메시지에 남지 않는다. 줄 번호만 나간다.
		if strings.Contains(err.Error(), "ghp_") {
			t.Errorf("거절 메시지에 시크릿 값이 실림:\n%s", err)
		}
	})

	t.Run("시크릿이 든 문서는 --to sources도 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := guardInboxDoc(t, dir, "inbox/2026-08-01-열쇠.md",
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n"+
				"source_channel: manual\nderived_context: []\nrelated: []\n",
			"# 열쇠 메모\n\n토큰은 ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD 입니다.\n")
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, "--to", "sources", rel)
		if err == nil || !strings.Contains(err.Error(), "github-token") {
			t.Fatalf("거절 오류 = %v", err)
		}
		if _, e := os.Stat(filepath.Join(dir, "sources", "2026-08-01-열쇠.md")); !os.IsNotExist(e) {
			t.Error("거절했는데 sources 문서가 생겼음")
		}
		if _, e := os.Stat(filepath.Join(dir, rel)); e != nil {
			t.Error("원본 inbox 문서가 사라졌음")
		}
	})

	t.Run("new도 같은 검사를 받습니다", func(t *testing.T) {
		dir := initWiki(t)
		_, err := runPromoteRoot(t, "new", "--wiki", dir,
			"ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD 정리")
		if err == nil || !strings.Contains(err.Error(), "github-token") {
			t.Fatalf("거절 오류 = %v", err)
		}
		entries, e := os.ReadDir(filepath.Join(dir, "context"))
		if e != nil {
			t.Fatal(e)
		}
		if len(entries) != 0 {
			t.Errorf("거절했는데 context에 파일이 생김: %v", entries)
		}
	})
}
