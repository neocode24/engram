package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// runSync는 sync 커맨드를 루트 등록 없이 시험한다. 전역 플래그는 실제
// 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runSync(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newSyncCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// syncGitRun은 임시 저장소에서 git 을 돌린다. date 가 있으면 커밋 날짜를
// 고정해 테스트가 실제 시계와 무관하게 돌게 한다.
func syncGitRun(t *testing.T, root, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if date != "" {
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE="+date+"T12:00:00+00:00",
			"GIT_COMMITTER_DATE="+date+"T12:00:00+00:00",
		)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v 실패: %v\n%s", args, err, out)
	}
}

// syncCommitAll은 날짜를 고정해 전체를 커밋한다.
func syncCommitAll(t *testing.T, root, date, msg string) {
	t.Helper()
	syncGitRun(t, root, date, "add", "-A")
	syncGitRun(t, root, date,
		"-c", "user.name=시험", "-c", "user.email=test@example.com",
		"commit", "-m", msg)
}

// makeSyncWiki는 커밋 이력이 둘 있는 git 저장소 위키를 만든다.
// context/note.md 는 두 번 커밋되고 나머지는 첫 커밋에만 들어간다.
// created 는 커밋 날짜와 다른 2020-01-01 이다.
func makeSyncWiki(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 이 PATH 에 없다")
	}
	root := t.TempDir()
	syncGitRun(t, root, "", "init")
	files := map[string]string{
		"engram.yaml": "preset: personal\nmin_wikilinks: 0\n",
		"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
		"context/note.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[index]]\"\nsource_channel: manual\nderived_context: []\n" +
			"created: 2020-01-01\n---\n\n본문\n",
		"sources/2026-01-01-원본.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
			"source_channel: web\ncreated: 2020-01-01\n---\n\n원본\n",
	}
	for name, content := range files {
		writeWikiFileAt(t, root, name, content)
	}
	syncCommitAll(t, root, "2026-01-01", "첫 커밋")
	writeWikiFileAt(t, root, "context/note.md", strings.Replace(
		files["context/note.md"], "본문\n", "본문\n고친 내용\n", 1))
	syncCommitAll(t, root, "2026-02-02", "둘째 커밋")
	return root
}

// writeWikiFileAt은 위키 루트 아래 파일 하나를 쓴다.
func writeWikiFileAt(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncCmd(t *testing.T) {
	t.Run("dry-run 이 기본이라 파일을 쓰지 않습니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		out, err := runSync(t, "sync", "--wiki", root)
		if err != nil {
			t.Fatalf("sync 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "dry-run") {
			t.Errorf("dry-run 임이 드러나야 함: %s", out)
		}
		content := readWiki(t, root, "context/note.md")
		if strings.Contains(content, "updated:") || strings.Contains(content, "sourced_at:") {
			t.Errorf("dry-run 이 파일을 썼음:\n%s", content)
		}
	})

	t.Run("--apply 는 updated 를 마지막 커밋 날짜로 씁니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		out, err := runSync(t, "sync", "--wiki", root, "--apply")
		if err != nil {
			t.Fatalf("sync 실패: %v\n%s", err, out)
		}
		// 두 번 커밋했으므로 마지막 커밋 날짜가 들어가야 한다.
		if !strings.Contains(readWiki(t, root, "context/note.md"), "updated: 2026-02-02") {
			t.Errorf("updated 가 마지막 커밋 날짜가 아님:\n%s", readWiki(t, root, "context/note.md"))
		}
	})

	t.Run("sourced_at 은 최초 커밋 날짜로 채웁니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		if _, err := runSync(t, "sync", "--wiki", root, "--apply"); err != nil {
			t.Fatal(err)
		}
		content := readWiki(t, root, "context/note.md")
		if !strings.Contains(content, "sourced_at: 2026-01-01") {
			t.Errorf("sourced_at 이 최초 커밋 날짜가 아님:\n%s", content)
		}
	})

	t.Run("sources 문서에 updated 가 생기지 않습니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		if _, err := runSync(t, "sync", "--wiki", root, "--apply"); err != nil {
			t.Fatal(err)
		}
		content := readWiki(t, root, "sources/2026-01-01-원본.md")
		if strings.Contains(content, "updated:") {
			t.Errorf("원본 보존 계층에 updated 가 생겼음:\n%s", content)
		}
		if !strings.Contains(content, "sourced_at: 2026-01-01") {
			t.Errorf("sourced_at 은 채워져야 함:\n%s", content)
		}
	})

	t.Run("created 는 바꾸지 않습니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		if _, err := runSync(t, "sync", "--wiki", root, "--apply"); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"context/note.md", "sources/2026-01-01-원본.md"} {
			if !strings.Contains(readWiki(t, root, name), "created: 2020-01-01") {
				t.Errorf("%s 의 created 가 바뀌었음:\n%s", name, readWiki(t, root, name))
			}
		}
	})

	t.Run("두 번 apply 하면 두 번째는 바뀐 문서가 0 입니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		if _, err := runSync(t, "sync", "--wiki", root, "--apply"); err != nil {
			t.Fatal(err)
		}
		out, err := runSync(t, "sync", "--wiki", root, "--apply", "--json")
		if err != nil {
			t.Fatalf("두 번째 sync 실패: %v\n%s", err, out)
		}
		var res syncOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if len(res.Changed) != 0 {
			t.Errorf("두 번째는 바뀐 문서가 없어야 함: %+v", res.Changed)
		}
	})

	t.Run("커밋되지 않은 문서는 건너뛰고 개수를 알립니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		writeWikiFileAt(t, root, "inbox/new.md",
			"---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n---\n\n메모\n")
		out, err := runSync(t, "sync", "--wiki", root)
		if err != nil {
			t.Fatalf("커밋되지 않은 파일이 있어도 실패가 아니어야 함: %v\n%s", err, out)
		}
		if !strings.Contains(out, "건너뛰었습니다") || !strings.Contains(out, "1개") {
			t.Errorf("건너뛴 개수 안내가 없음: %s", out)
		}
		if strings.Contains(out, "inbox/new.md") {
			t.Errorf("커밋되지 않은 문서가 정정 대상에 있음: %s", out)
		}
	})

	t.Run("git 저장소가 아니면 실패하고 안내가 나옵니다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runSync(t, "sync", "--wiki", root)
		if err == nil {
			t.Fatal("저장소가 아니면 에러여야 함")
		}
		if !strings.Contains(out, "git init") {
			t.Errorf("git init 안내가 없음: %s", out)
		}
	})

	t.Run("--apply 뒤 lint 의 error 와 warn 이 늘지 않습니다", func(t *testing.T) {
		root := makeSyncWiki(t)
		cfg, err := config.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		before, err := lint.Run(root, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runSync(t, "sync", "--wiki", root, "--apply"); err != nil {
			t.Fatal(err)
		}
		after, err := lint.Run(root, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if after.Summary.Error > before.Summary.Error || after.Summary.Warn > before.Summary.Warn {
			t.Errorf("lint 위반이 늘었음: before error %d warn %d, after error %d warn %d\n%s",
				before.Summary.Error, before.Summary.Warn, after.Summary.Error, after.Summary.Warn,
				formatViolations(after.Violations))
		}
	})
}

// formatViolations은 위반 목록을 읽을 수 있는 형태로 낸다.
func formatViolations(vs []lint.Violation) string {
	var b strings.Builder
	for _, v := range vs {
		fmt.Fprintf(&b, "%s [%s] %s\n", v.Path, v.Severity, v.Rule)
	}
	return b.String()
}
