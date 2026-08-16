package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/serve"
	"github.com/spf13/cobra"
)

// runServe는 serve 커맨드를 루트 등록 없이 시험한다. 커맨드 등록은
// coordinator 가 root.go 에서 하므로 여기서는 전역 플래그를 테스트용 부모
// 커맨드에 붙여 조립한다. 실제 포트를 여는 경로에는 들어가지 않는 인자만
// 준다. 서버를 띄우는 것은 손으로 확인할 일이다.
func runServe(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.AddCommand(newServeCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeServeWiki는 시험용 위키를 만든다. 문서는 없어도 되고 설정 파일이
// 있으면 위키다.
func makeServeWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte("preset: team\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestServeCmdRejectsBadInput(t *testing.T) {
	t.Run("위키가 아니면 거절하고 init 을 안내합니다", func(t *testing.T) {
		out, err := runServe(t, "serve", "--wiki", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 합니다")
		}
		if !strings.Contains(err.Error(), "engram init") {
			t.Errorf("안내에 engram init 이 없습니다: %v\n%s", err, out)
		}
	})

	t.Run("포트 범위 밖이면 거절합니다", func(t *testing.T) {
		_, err := runServe(t, "serve", "--wiki", makeServeWiki(t), "--port", "70000")
		if err == nil {
			t.Fatal("포트 범위 밖이면 에러여야 합니다")
		}
		if !strings.Contains(err.Error(), "포트 범위") {
			t.Errorf("에러 문구가 다릅니다: %v", err)
		}
	})

	t.Run("인자를 받지 않습니다", func(t *testing.T) {
		if _, err := runServe(t, "serve", "어딘가"); err == nil {
			t.Fatal("위치 인자를 받으면 에러여야 합니다")
		}
	})
}

func TestServeNotice(t *testing.T) {
	base := serve.Exposure{
		Context: 12, Index: 1, ExcludedInbox: 3, ExcludedSources: 2, ExcludedArchive: 4,
		ExcludedSensitive: 2, SensitivityOn: true,
	}

	t.Run("무엇이 노출되고 무엇이 제외되는지 알립니다", func(t *testing.T) {
		var out bytes.Buffer
		printServeNotice(&out, "/tmp/wiki", "127.0.0.1", "127.0.0.1:8420", base)
		got := out.String()
		for _, want := range []string{
			"http://127.0.0.1:8420",
			"/tmp/wiki",
			"context 문서 12개",
			"색인 문서 1개",
			"inbox 3개",
			"sources 2개",
			"archive 4개(--include-archive 로 엽니다)",
			"private-local-only 와 restricted 문서 2개를 제외했습니다",
			"쓰기 경로가 없습니다",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("안내에 %q 가 없습니다:\n%s", want, got)
			}
		}
		if strings.Contains(got, "경고:") {
			t.Errorf("루프백 바인딩에 경고가 붙었습니다:\n%s", got)
		}
	})

	t.Run("외부 노출이면 인증이 없다고 경고합니다", func(t *testing.T) {
		var out bytes.Buffer
		printServeNotice(&out, "/tmp/wiki", "0.0.0.0", "0.0.0.0:8420", base)
		got := out.String()
		if !strings.Contains(got, "경고:") || !strings.Contains(got, "인증이 없으므로") {
			t.Errorf("외부 노출 경고가 없습니다:\n%s", got)
		}
	})

	t.Run("축이 꺼진 위키는 거를 값이 없다고 알립니다", func(t *testing.T) {
		var out bytes.Buffer
		e := base
		e.SensitivityOn = false
		e.ExcludedSensitive = 0
		printServeNotice(&out, "/tmp/wiki", "localhost", "127.0.0.1:8420", e)
		got := out.String()
		if !strings.Contains(got, "sensitivity 축이 꺼져 있어") {
			t.Errorf("축이 꺼진 안내가 없습니다:\n%s", got)
		}
		if strings.Contains(got, "경고:") {
			t.Errorf("localhost 바인딩에 경고가 붙었습니다:\n%s", got)
		}
	})

	t.Run("archive 를 열면 노출에 함께 셉니다", func(t *testing.T) {
		var out bytes.Buffer
		e := base
		e.IncludeArchive = true
		e.Archive = 4
		e.ExcludedArchive = 0
		printServeNotice(&out, "/tmp/wiki", "127.0.0.1", "127.0.0.1:8420", e)
		got := out.String()
		if !strings.Contains(got, "archive 문서 4개") {
			t.Errorf("archive 노출 수가 없습니다:\n%s", got)
		}
		if strings.Contains(got, "--include-archive 로 엽니다") {
			t.Errorf("이미 열었는데 여는 법을 안내합니다:\n%s", got)
		}
	})
}

func TestLoopbackHost(t *testing.T) {
	for _, tc := range []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"localhost", true},
		{"0.0.0.0", false},
		{"192.168.0.10", false},
		{"", false},
		{"example.internal", false},
	} {
		if got := loopbackHost(tc.host); got != tc.want {
			t.Errorf("loopbackHost(%q) = %v, 기대값 %v", tc.host, got, tc.want)
		}
	}
}
