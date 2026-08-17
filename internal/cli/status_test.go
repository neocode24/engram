package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/status"
	"github.com/spf13/cobra"
)

// runStatus는 status 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 가 root.go 에서 하므로 여기서는 전역 플래그를
// 테스트용 부모 커맨드에 붙여 조립한다. --now 파싱은 실제 루트와 같은
// PersistentPreRunE 로 흉내낸다.
func runStatus(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newStatusCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeStatusWiki는 임시 디렉토리에 승급 대기 문서 하나를 갖춘 위키를 만든다.
func makeStatusWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: personal\n",
		"inbox/2026-07-15-ready.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\ncreated: 2026-07-15\nrelated:\n  - \"[[hub]]\"\n---\n\n[[peer]] 도 봅니다\n",
		"inbox/peer.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel: manual\n---\n\n메모\n",
		"context/hub.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\nindexable: true\n" +
			"source_refs: []\nderived_from: []\nrelated:\n  - \"[[peer]]\"\nsource_channel: manual\nderived_context: []\n" +
			"created: 2026-01-01\nupdated: 2026-08-01\n---\n\n[[peer]]\n",
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

func TestStatusCmd(t *testing.T) {
	t.Run("텍스트 출력은 세 구획과 다음 행동을 냅니다", func(t *testing.T) {
		out, err := runStatus(t, "status", "--now", "2026-08-15T12:00:00Z", makeStatusWiki(t))
		if err != nil {
			t.Fatalf("status 는 항상 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		for _, want := range []string{"현황", "적체 압력 (기준 2026-08-15)", "다음 행동", "engram promote inbox/2026-07-15-ready.md"} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
		if strings.Contains(out, "\x1b[") {
			t.Error("색상 이스케이프 코드가 있습니다")
		}
	})

	t.Run("--json 은 지표와 제안을 구조화해 냅니다", func(t *testing.T) {
		out, err := runStatus(t, "status", "--json", "--now", "2026-08-15T12:00:00Z", makeStatusWiki(t))
		if err != nil {
			t.Fatalf("status 는 항상 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		var res status.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Stages.Inbox != 2 || res.Stages.Context != 1 {
			t.Errorf("단계 집계: %+v", res.Stages)
		}
		if res.Backlog.OldestDays == nil || *res.Backlog.OldestDays != 31 {
			t.Errorf("최고령 나이: %v", res.Backlog.OldestDays)
		}
		if len(res.Suggestions) == 0 || !strings.HasPrefix(res.Suggestions[0].Action, "engram promote") {
			t.Errorf("제안: %+v", res.Suggestions)
		}
	})

	t.Run("위키가 아니면 거절하고 init 을 안내합니다", func(t *testing.T) {
		out, err := runStatus(t, "status", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 합니다")
		}
		if !strings.Contains(out, "engram init") {
			t.Errorf("안내에 engram init 이 없습니다: %s", out)
		}
	})

	t.Run("--now 를 고정하면 두 번 실행에 출력이 같습니다", func(t *testing.T) {
		wiki := makeStatusWiki(t)
		first, err1 := runStatus(t, "status", "--json", "--now", "2026-08-15T12:00:00Z", wiki)
		second, err2 := runStatus(t, "status", "--json", "--now", "2026-08-15T12:00:00Z", wiki)
		if err1 != nil || err2 != nil {
			t.Fatalf("실행 에러: %v %v", err1, err2)
		}
		if first != second {
			t.Fatalf("두 실행의 출력이 다릅니다:\n%s\n===\n%s", first, second)
		}
	})
}
