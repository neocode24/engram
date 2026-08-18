package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/digest"
	"github.com/spf13/cobra"
)

// runDigest는 digest 커맨드를 루트 등록 없이 시험한다. 조립 방식은
// status_test 와 같다.
func runDigest(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newDigestCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeDigestWiki는 집계 대상이 분명한 위키를 만든다. 기준 시각
// 2026-08-16, --days 30 의 창은 2026-07-17 부터다.
func makeDigestWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: personal\n",
		// 창 안 신규. old 와 서로 링크해 고아가 아니다.
		"context/new.md": "---\ntype: concept\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n\n[[old]]\n",
		// 창 밖 노후.
		"context/old.md": "---\ntype: concept\ncreated: 2025-01-01\nupdated: 2025-01-01\n---\n\n[[new]]\n",
		// 고아.
		"context/lonely.md": "---\ntype: concept\ncreated: 2026-07-20\nupdated: 2026-07-20\n---\n\n본문\n",
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

func TestDigestCmd(t *testing.T) {
	t.Run("텍스트 출력은 항목별 건수와 슬러그를 낸다", func(t *testing.T) {
		out, err := runDigest(t, "digest", "--now", "2026-08-16T12:00:00Z", "--wiki", makeDigestWiki(t))
		if err != nil {
			t.Fatalf("digest 오류: %v\n%s", err, out)
		}
		for _, want := range []string{
			"기간 집계 (2026-07-17부터 2026-08-16까지, 30일)",
			"신규 2개: lonely, new",
			"오래된 문서 1개: old",
			"고아 1개: lonely",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("--days 로 기간을 바꾼다", func(t *testing.T) {
		// 창을 20일로 줄이면 2026-07-20 문서는 창 밖이 되고 08-01 문서만 남는다.
		out, err := runDigest(t, "digest", "--now", "2026-08-16T12:00:00Z", "--days", "20", "--wiki", makeDigestWiki(t))
		if err != nil {
			t.Fatalf("digest 오류: %v\n%s", err, out)
		}
		if !strings.Contains(out, "신규 1개: new") {
			t.Errorf("기간 축소 반영 안 됨:\n%s", out)
		}
	})

	t.Run("--json 은 전체 목록을 낸다", func(t *testing.T) {
		out, err := runDigest(t, "digest", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", makeDigestWiki(t))
		if err != nil {
			t.Fatalf("digest 오류: %v\n%s", err, out)
		}
		var res digest.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if len(res.Created) != 2 || len(res.Stale) != 1 || len(res.Orphans) != 1 {
			t.Errorf("집계: %+v", res)
		}
		if res.Days != 30 || res.StaleDays != 30 {
			t.Errorf("기준값: %+v", res)
		}
	})

	t.Run("같은 --now 로 두 번 실행하면 출력이 같습니다", func(t *testing.T) {
		wiki := makeDigestWiki(t)
		first, err1 := runDigest(t, "digest", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki)
		second, err2 := runDigest(t, "digest", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", wiki)
		if err1 != nil || err2 != nil {
			t.Fatalf("실행 에러: %v %v", err1, err2)
		}
		if first != second {
			t.Fatalf("두 실행의 출력이 다릅니다:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("위키가 아니면 거절하고 init 을 안내합니다", func(t *testing.T) {
		out, err := runDigest(t, "digest", "--wiki", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 합니다")
		}
		if !strings.Contains(out, "engram init") {
			t.Errorf("안내에 engram init 이 없습니다: %s", out)
		}
	})

	t.Run("음수 --days 는 거절합니다", func(t *testing.T) {
		out, err := runDigest(t, "digest", "--days", "-1", "--wiki", makeDigestWiki(t))
		if err == nil {
			t.Fatal("음수 기간은 에러여야 합니다")
		}
		if !strings.Contains(out, "0 이상") {
			t.Errorf("안내에 조건이 없습니다: %s", out)
		}
	})
}

func TestDigestCmdManyNew(t *testing.T) {
	// 신규가 11개면 사람용 목록은 10개와 남은 수로 줄인다.
	root := t.TempDir()
	files := map[string]string{"engram.yaml": "preset: personal\n"}
	for i := 1; i <= 11; i++ {
		files[fmt.Sprintf("inbox/2026-08-01-note%02d.md", i)] =
			fmt.Sprintf("---\ntype: inbox-note\ncreated: 2026-08-01\n---\n\n본문 %d\n", i)
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
	out, err := runDigest(t, "digest", "--now", "2026-08-16T12:00:00Z", "--wiki", root)
	if err != nil {
		t.Fatalf("digest 오류: %v\n%s", err, out)
	}
	if !strings.Contains(out, "신규 11개: ") || !strings.Contains(out, "외 1개") {
		t.Errorf("상한 축소가 없습니다:\n%s", out)
	}
	// --json 은 전체 목록을 낸다.
	jsonOut, err := runDigest(t, "digest", "--json", "--now", "2026-08-16T12:00:00Z", "--wiki", root)
	if err != nil {
		t.Fatalf("digest 오류: %v\n%s", err, jsonOut)
	}
	var res digest.Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonOut)), &res); err != nil {
		t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, jsonOut)
	}
	if len(res.Created) != 11 {
		t.Errorf("JSON 신규 목록 = %d개, want 11", len(res.Created))
	}
}
