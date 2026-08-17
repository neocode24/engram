package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/doctor"
	"github.com/spf13/cobra"
)

// runDoctor는 doctor 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 가 root.go 에서 하므로 여기서는 전역 플래그를
// 테스트용 부모 커맨드에 붙여 조립한다.
func runDoctor(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.AddCommand(newDoctorCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeDoctorWiki는 임시 디렉토리에 위키를 만든다.
func makeDoctorWiki(t *testing.T, engramYAML string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"inbox", "sources", "context", "archive"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range map[string]string{
		"engram.yaml": engramYAML,
		"index.md":    "# 색인\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestDoctorCmd(t *testing.T) {
	t.Run("정상 위키는 종료 코드 0 이고 요약 줄을 냅니다", func(t *testing.T) {
		out, err := runDoctor(t, "doctor", makeDoctorWiki(t, "preset: personal\n"))
		if err != nil {
			t.Fatalf("fail 이 없으면 에러가 아니어야 합니다: %v\n%s", err, out)
		}
		for _, want := range []string{"[ok] env.git", "항목 12개"} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("위키가 아닌 디렉토리도 환경 점검은 돕니다", func(t *testing.T) {
		out, err := runDoctor(t, "doctor", t.TempDir())
		if err != nil {
			t.Fatalf("환경 점검만으로는 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		if !strings.Contains(out, "[ok] env.git") {
			t.Errorf("환경 점검이 없음:\n%s", out)
		}
		if !strings.Contains(out, "[skip] wiki.config") {
			t.Errorf("위키 항목 skip 이 없음:\n%s", out)
		}
	})

	t.Run("fail 항목이 있으면 종료 코드 1 입니다", func(t *testing.T) {
		out, err := runDoctor(t, "doctor", makeDoctorWiki(t, "preset: [깨진\n"))
		if err == nil {
			t.Fatal("설정 파싱 실패는 종료 코드 1이어야 합니다")
		}
		if !strings.Contains(out, "wiki.config") {
			t.Errorf("설정 실패 항목이 없음:\n%s", out)
		}
	})

	t.Run("warn 항목에는 조치가 따라 나옵니다", func(t *testing.T) {
		out, err := runDoctor(t, "doctor", makeDoctorWiki(t, "min_wikilinks: 0\n"))
		if err != nil {
			t.Fatalf("warn 은 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		if !strings.Contains(out, "[warn] wiki.min-wikilinks") || !strings.Contains(out, "조치:") {
			t.Errorf("warn 과 조치가 없음:\n%s", out)
		}
	})

	t.Run("--json 은 항목 배열과 요약을 냅니다", func(t *testing.T) {
		out, err := runDoctor(t, "doctor", "--json", makeDoctorWiki(t, "min_wikilinks: 0\n"))
		if err != nil {
			t.Fatalf("warn 은 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		var res doctor.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if len(res.Findings) != 12 {
			t.Errorf("항목은 12개여야 합니다, got %d", len(res.Findings))
		}
		if res.Findings[0].ID != "env.git" || res.Findings[0].Status == "" {
			t.Errorf("첫 항목이 env.git 이어야 합니다: %+v", res.Findings[0])
		}
		if res.Summary.Items != 12 || res.Summary.Warn < 1 {
			t.Errorf("요약이 틀렸습니다: %+v", res.Summary)
		}
		for _, f := range res.Findings {
			if f.ID == "" || f.Detail == "" {
				t.Errorf("항목에 id 나 detail 이 비어 있음: %+v", f)
			}
			if (f.Status == doctor.StatusWarn || f.Status == doctor.StatusFail) && f.Fix == "" {
				t.Errorf("warn/fail 항목에 fix 없음: %+v", f)
			}
		}
	})
}
