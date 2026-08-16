package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/resurface"
	"github.com/spf13/cobra"
)

// runResurface는 resurface 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 가 root.go 에서 하므로 여기서는 전역 플래그를
// 테스트용 부모 커맨드에 붙여 조립한다. status_test 와 같은 모양이다.
func runResurface(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newResurfaceCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeResurfaceWiki는 오래된 context 문서 둘과 최근 문서 하나를 갖춘
// 위키를 만든다. 기준 시각 2026-08-16 에서 old-a 는 227일, old-b 는
// 258일 묵었다.
func makeResurfaceWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml":      "preset: education\n",
		"context/old-a.md": "---\ntype: concept\ncreated: 2026-01-01\n---\n\n# 첫 결정\n",
		"context/old-b.md": "---\ntype: concept\ncreated: 2025-12-01\n---\n\n# 더 오래된 결정\n",
		"context/fresh.md": "---\ntype: concept\ncreated: 2026-08-01\nupdated: 2026-08-10\n---\n\n본문\n",
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

func TestResurfaceCmd(t *testing.T) {
	t.Run("텍스트 출력은 후보와 근거를 낸다", func(t *testing.T) {
		wiki := makeResurfaceWiki(t)
		out, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki)
		if err != nil {
			t.Fatalf("resurface 오류: %v\n%s", err, out)
		}
		for _, want := range []string{
			"다시 꺼낼 문서 2개 (stale_days 90일, 기준 2026-08-16)",
			"old-b (더 오래된 결정): 마지막 갱신 258일 전, 제시한 적 없음",
			"old-a (첫 결정): 마지막 갱신 227일 전, 제시한 적 없음",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
		if strings.Contains(out, "fresh") {
			t.Error("최근 문서가 후보에 있습니다")
		}
		// 실행하면 이력이 기록된다.
		if _, err := os.Stat(filepath.Join(wiki, ".engram", "resurface.json")); err != nil {
			t.Fatalf("제시 이력이 기록되지 않았습니다: %v", err)
		}
	})

	t.Run("두 번째 실행은 마지막 제시 시각을 낸다", func(t *testing.T) {
		wiki := makeResurfaceWiki(t)
		if _, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki); err != nil {
			t.Fatal(err)
		}
		out, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "마지막 제시 2026-08-16") {
			t.Errorf("두 번째 출력에 마지막 제시가 없습니다:\n%s", out)
		}
	})

	t.Run("--dry-run 은 이력을 남기지 않고 알린다", func(t *testing.T) {
		wiki := makeResurfaceWiki(t)
		out, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--dry-run", "--wiki", wiki)
		if err != nil {
			t.Fatalf("resurface 오류: %v\n%s", err, out)
		}
		if !strings.Contains(out, "--dry-run이라 제시 이력을 기록하지 않았습니다") {
			t.Errorf("dry-run 안내가 없습니다:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(wiki, ".engram", "resurface.json")); !os.IsNotExist(err) {
			t.Fatalf("dry-run 이 이력을 남겼습니다: %v", err)
		}
	})

	t.Run("--limit 는 낼 문서 수를 제한한다", func(t *testing.T) {
		wiki := makeResurfaceWiki(t)
		out, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--limit", "1", "--wiki", wiki)
		if err != nil {
			t.Fatalf("resurface 오류: %v\n%s", err, out)
		}
		if strings.Contains(out, "old-a") {
			t.Errorf("limit 1 인데 old-a 가 나왔습니다:\n%s", out)
		}
	})

	t.Run("--json 은 후보와 기준값을 구조화해 낸다", func(t *testing.T) {
		wiki := makeResurfaceWiki(t)
		out, err := runResurface(t, "resurface", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki)
		if err != nil {
			t.Fatalf("resurface 오류: %v\n%s", err, out)
		}
		var res resurface.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.StaleDays != 90 || res.SkippedNoDate != 0 {
			t.Errorf("기준값: %+v", res)
		}
		if len(res.Candidates) != 2 || res.Candidates[0].Slug != "old-b" {
			t.Errorf("후보: %+v", res.Candidates)
		}
		if res.Candidates[0].LastShown != nil {
			t.Errorf("첫 실행에 LastShown 이 있습니다: %+v", res.Candidates[0])
		}
		if res.Candidates[0].AgeDays != 258 {
			t.Errorf("경과일: %d", res.Candidates[0].AgeDays)
		}
	})

	t.Run("후보가 없으면 이유를 낸다", func(t *testing.T) {
		// fresh 위키: context 문서가 하나뿐이고 최근이다.
		fresh := t.TempDir()
		for name, content := range map[string]string{
			"engram.yaml":      "preset: education\n",
			"context/fresh.md": "---\ntype: concept\ncreated: 2026-08-01\nupdated: 2026-08-10\n---\n\n본문\n",
		} {
			p := filepath.Join(fresh, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		out, err := runResurface(t, "resurface", "--now", "2026-08-16T12:00:00Z", "--wiki", fresh)
		if err != nil {
			t.Fatalf("resurface 오류: %v\n%s", err, out)
		}
		if !strings.Contains(out, "다시 꺼낼 문서가 없습니다") || !strings.Contains(out, "이유:") {
			t.Errorf("빈 결과 안내가 없습니다:\n%s", out)
		}
	})

	t.Run("위키가 아니면 거절하고 init 을 안내합니다", func(t *testing.T) {
		out, err := runResurface(t, "resurface", "--wiki", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 합니다")
		}
		if !strings.Contains(out, "engram init") {
			t.Errorf("안내에 engram init 이 없습니다: %s", out)
		}
	})
}
