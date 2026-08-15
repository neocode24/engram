package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// runMv는 mv 커맨드를 루트 등록 없이 시험한다.
func runMv(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newMvCmd())
	var out strings.Builder
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// linkyWiki는 옛 슬러그 old 에 링크를 여러 종류로 건 위키를 만든다.
// 대상 문서는 inbox 에 날짜 접두사를 달아 둔다.
func linkyWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: education\nmin_wikilinks: 0\n",
		// mv 대상. context 문서라 슬러그가 곧 파일명이다.
		"context/old.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n대상 문서\n",
		// 날짜 접두사 보존 확인용 inbox 문서.
		"inbox/2026-02-20-prefixdoc.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\n---\n\n메모\n",
		// 본문 링크 세 형태와 코드 펜스, 인라인 코드를 함께 둔다.
		"context/body.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n" +
			"---\n\n[[old]] 를 봅니다. [[old|옛 문서]] 도 봅니다. [[old#절]] 도 봅니다.\n" +
			"```\n[[old]] 는 코드 펜스 안입니다\n```\n" +
			"인라인 `[[old]]` 도 코드입니다.\n",
		// related 필드 링크.
		"context/related.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[old]]\"\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n본문\n",
		// 관계 필드 경로 링크와 파생 맥락 슬러그 링크.
		"sources/2026-01-10-원본.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\nsource_refs:\n  - context/old.md\nderived_from: []\n" +
			"derived_context:\n  - old\nsource_channel: web\ncreated: 2026-01-10\nsourced_at: 2026-01-11\n" +
			"---\n\n원본\n",
		// 링크가 전혀 없는 문서.
		"context/alone.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n고립 문서\n",
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

func lintHasBroken(res lint.Result) bool {
	for _, v := range res.Violations {
		if v.Rule == "link.broken" {
			return true
		}
	}
	return false
}

func TestMvCmd(t *testing.T) {
	t.Run("모든 종류의 링크를 고치고 접두사를 유지합니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--wiki", root, "old", "renamed")
		if err != nil {
			t.Fatalf("mv 실패: %v\n%s", err, out)
		}
		// 파일이 옮겨졌는지.
		if _, err := os.Stat(filepath.Join(root, "context", "renamed.md")); err != nil {
			t.Fatalf("옮겨진 문서가 없습니다: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "context", "old.md")); !os.IsNotExist(err) {
			t.Error("옛 문서가 남아 있습니다")
		}
		// 본문 세 형태. 표시 문자열과 헤딩 보존.
		body := readWiki(t, root, "context/body.md")
		for _, want := range []string{"[[renamed]] 를 봅니다", "[[renamed|옛 문서]] 도 봅니다", "[[renamed#절]] 도 봅니다"} {
			if !strings.Contains(body, want) {
				t.Errorf("본문에 %q 없음:\n%s", want, body)
			}
		}
		// 코드 펜스와 인라인 코드는 그대로.
		for _, want := range []string{"[[old]] 는 코드 펜스 안입니다", "`[[old]]`"} {
			if !strings.Contains(body, want) {
				t.Errorf("코드 안 링크가 바뀌었습니다: %q 없음:\n%s", want, body)
			}
		}
		// related 갱신.
		if rel := readWiki(t, root, "context/related.md"); !strings.Contains(rel, "[[renamed]]") {
			t.Errorf("related 갱신이 없음:\n%s", rel)
		}
		// 관계 필드 갱신.
		src := readWiki(t, root, "sources/2026-01-10-원본.md")
		if !strings.Contains(src, "context/renamed.md") {
			t.Errorf("source_refs 경로 갱신이 없음:\n%s", src)
		}
		if !strings.Contains(src, "  - renamed") {
			t.Errorf("derived_context 슬러그 갱신이 없음:\n%s", src)
		}
	})

	t.Run("mv 뒤 lint 가 깨진 링크를 보고하지 않습니다", func(t *testing.T) {
		root := linkyWiki(t)
		if _, err := runMv(t, "mv", "--wiki", root, "old", "renamed"); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		res, err := lint.Run(root, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if lintHasBroken(res) {
			for _, v := range res.Violations {
				if v.Rule == "link.broken" {
					t.Errorf("깨진 링크: %+v", v)
				}
			}
		}
	})

	t.Run("링크 없는 문서는 파일만 옮깁니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--wiki", root, "alone", "solitary")
		if err != nil {
			t.Fatalf("mv 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "고칠 링크가 없습니다") {
			t.Errorf("링크 없음 안내가 없음:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(root, "context", "solitary.md")); err != nil {
			t.Fatalf("옮겨진 문서가 없습니다: %v", err)
		}
	})

	t.Run("--dry-run 은 아무것도 쓰지 않습니다", func(t *testing.T) {
		root := linkyWiki(t)
		before := readWiki(t, root, "context/body.md")
		out, err := runMv(t, "mv", "--dry-run", "--wiki", root, "old", "renamed")
		if err != nil {
			t.Fatalf("dry-run 실패: %v\n%s", err, out)
		}
		if after := readWiki(t, root, "context/body.md"); after != before {
			t.Error("dry-run 이 문서를 바꿨다")
		}
		if _, err := os.Stat(filepath.Join(root, "context", "old.md")); err != nil {
			t.Error("dry-run 이 파일을 옮겼다")
		}
		if !strings.Contains(out, "바꿀 예정") || !strings.Contains(out, "아무것도 쓰지 않았습니다") {
			t.Errorf("dry-run 안내가 없음:\n%s", out)
		}
	})

	t.Run("옛 슬러그가 없으면 거절합니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--wiki", root, "no-such-doc", "x")
		if err == nil {
			t.Fatal("거절이어야 합니다")
		}
		if !strings.Contains(out, "문서가 없습니다") {
			t.Errorf("거절 안내가 없음: %s", out)
		}
	})

	t.Run("새 슬러그가 이미 있으면 거절합니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--wiki", root, "old", "alone")
		if err == nil {
			t.Fatal("거절이어야 합니다")
		}
		if !strings.Contains(out, "이미 쓰이고 있습니다") {
			t.Errorf("거절 안내가 없음: %s", out)
		}
		if _, err := os.Stat(filepath.Join(root, "context", "old.md")); err != nil {
			t.Error("거절했는데 원본이 사라졌습니다")
		}
	})

	t.Run("--json 은 결과를 구조화해 냅니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--json", "--wiki", root, "old", "renamed")
		if err != nil {
			t.Fatalf("mv 실패: %v\n%s", err, out)
		}
		var res mvOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if res.Slug != "renamed" || !strings.HasSuffix(res.To, "renamed.md") {
			t.Errorf("결과가 틀리입니다: %+v", res)
		}
		total := 0
		for _, u := range res.Updated {
			total += u.Links
		}
		// 본문 3 + related 1 + source_refs 1 + derived_context 1 = 6.
		if total != 6 {
			t.Errorf("고친 링크 수 = %d, want 6: %+v", total, res.Updated)
		}
	})

	t.Run("위키가 아니면 거절합니다", func(t *testing.T) {
		out, err := runMv(t, "mv", "--wiki", t.TempDir(), "a", "b")
		if err == nil {
			t.Fatal("거절이어야 합니다")
		}
		if !strings.Contains(out, "engram init") {
			t.Errorf("init 안내가 없음: %s", out)
		}
	})
}

func TestMvCmdPrefix(t *testing.T) {
	t.Run("inbox 문서를 옮기면 날짜 접두사를 유지합니다", func(t *testing.T) {
		root := linkyWiki(t)
		out, err := runMv(t, "mv", "--wiki", root, "prefixdoc", "newname")
		if err != nil {
			t.Fatalf("mv 실패: %v\n%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(root, "inbox", "2026-02-20-newname.md")); err != nil {
			t.Fatalf("접두사가 유지된 파일이 없습니다: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "inbox", "2026-02-20-prefixdoc.md")); !os.IsNotExist(err) {
			t.Error("옛 문서가 남아 있습니다")
		}
	})
}
