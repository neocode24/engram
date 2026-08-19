package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/embed"
	"github.com/spf13/cobra"
)

// runModel는 model 커맨드를 루트 등록 없이 시험한다. 커맨드 등록은
// coordinator 가 root.go 에서 하므로 여기서는 전역 플래그를 테스트용
// 부모 커맨드에 붙여 조립한다. 모델 경로는 ENGRAM_MODEL_DIR 로 시험
// 디렉토리를 가리킨다.
func runModel(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.AddCommand(newModelCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// modelTestDir는 시험용 모델 디렉토리를 만들고 그 경로로 환경변수를
// 잡는다.
func modelTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(embed.EnvModelDir, dir)
	return dir
}

func TestModelStatusCmdReportsState(t *testing.T) {
	dir := modelTestDir(t)

	out, err := runModel(t, "model", "status")
	if err != nil {
		t.Fatalf("에러 없이 돌아야 함: %v\n%s", err, out)
	}
	for _, want := range []string{
		dir,
		embed.Revision,
		"model.onnx",
		"model.onnx_data",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("출력에 %q 가 없습니다:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "없음"); got != len(embed.ModelFiles()) {
		t.Errorf("빈 디렉토리에서 없음이 %d번 나와야 합니다:\n%s", len(embed.ModelFiles()), out)
	}
	if !strings.Contains(out, "파일 0/6개") {
		t.Errorf("파일 수 요약이 없습니다:\n%s", out)
	}
	if !strings.Contains(out, "온전하지 않습니다") {
		t.Errorf("불완전 안내가 없습니다:\n%s", out)
	}
}

func TestModelStatusCmdWithFiles(t *testing.T) {
	dir := modelTestDir(t)
	// 크기가 기대와 다른 파일을 하나 둔다. 기본 출력은 존재와 크기만
	// 보므로 해시 없이도 이 사실이 나와야 한다.
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runModel(t, "model", "status")
	if err != nil {
		t.Fatalf("에러 없이 돌아야 함: %v\n%s", err, out)
	}
	if !strings.Contains(out, "2 / 770 바이트") {
		t.Errorf("실측 크기가 출력에 없습니다:\n%s", out)
	}
	if !strings.Contains(out, "파일 1/6개") {
		t.Errorf("파일 수 요약이 다릅니다:\n%s", out)
	}
}

func TestModelStatusCmdJSON(t *testing.T) {
	dir := modelTestDir(t)

	out, err := runModel(t, "model", "status", "--json")
	if err != nil {
		t.Fatalf("에러 없이 돌아야 함: %v\n%s", err, out)
	}
	var res modelStatusJSON
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("JSON 을 해석할 수 없습니다: %v\n%s", err, out)
	}
	if res.Dir != dir {
		t.Errorf("dir 이 %s 여야 함: %s", dir, res.Dir)
	}
	if res.Revision != embed.Revision {
		t.Errorf("revision 이 다릅니다: %s", res.Revision)
	}
	if len(res.Files) != len(embed.ModelFiles()) {
		t.Errorf("파일이 %d개여야 함: %d", len(embed.ModelFiles()), len(res.Files))
	}
	if res.Complete || res.Present != 0 {
		t.Errorf("빈 디렉토리인데 완전하다고 나옵니다: %+v", res)
	}
	if res.Verified != nil {
		t.Error("--verify 를 주지 않았는데 검증 결과가 들어 있습니다")
	}
	if res.ExpectedBytes == 0 {
		t.Error("기대 합계가 비었습니다")
	}
}

func TestModelStatusVerifyJudgesCorruption(t *testing.T) {
	dir := modelTestDir(t)
	// 여섯 전부 크기는 맞고 내용은 틀리게 둔다. 존재와 크기 판정은
	// 통과하고 체크섬 판정만 실패하는 훼손 상태다.
	for _, f := range embed.ModelFiles() {
		body := bytes.Repeat([]byte("!"), int(f.Size))
		if err := os.WriteFile(filepath.Join(dir, f.Name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("텍스트 출력은 온전하지 않다고 판정합니다", func(t *testing.T) {
		out, err := runModel(t, "model", "status", "--verify")
		// 훼손은 실패로 낸다. 화면이 온전하지 않다고 말하는데 종료코드가
		// 0 이면 doctor 와 스크립트가 통과로 읽는다.
		if err == nil {
			t.Errorf("훼손된 모델인데 실패로 끝나지 않았습니다:\n%s", out)
		}
		if !strings.Contains(out, "체크섬 불일치") {
			t.Errorf("불일치가 출력에 없습니다:\n%s", out)
		}
		if !strings.Contains(out, "온전하지 않습니다") {
			t.Errorf("훼손된 모델을 온전하다고 합니다:\n%s", out)
		}
	})

	t.Run("검사하지 않았으면 온전하다고 말하지 않습니다", func(t *testing.T) {
		out, err := runModel(t, "model", "status")
		if err != nil {
			t.Fatalf("검사 없는 조회는 실패가 아니어야 함: %v\n%s", err, out)
		}
		if strings.Contains(out, "모델이 온전합니다") {
			t.Errorf("존재와 크기만 보고 온전하다고 합니다:\n%s", out)
		}
		if !strings.Contains(out, "검사하지 않았습니다") {
			t.Errorf("검사하지 않았다는 사실이 결론에 없습니다:\n%s", out)
		}
	})

	t.Run("JSON 의 complete 도 거짓이어야 합니다", func(t *testing.T) {
		out, _ := runModel(t, "model", "status", "--json", "--verify")
		var res modelStatusJSON
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("JSON 을 해석할 수 없습니다: %v\n%s", err, out)
		}
		if res.Complete {
			t.Errorf("훼손된 모델인데 complete 가 참입니다:\n%s", out)
		}
	})
}

func TestModelPullCmdImportErrors(t *testing.T) {
	t.Run("없는 경로를 가져오려 하면 실패합니다", func(t *testing.T) {
		modelTestDir(t)
		missing := filepath.Join(t.TempDir(), "없음")
		_, err := runModel(t, "model", "pull", "--from", missing)
		if err == nil {
			t.Fatal("없는 경로는 에러여야 합니다")
		}
		if !strings.Contains(err.Error(), "오프라인 자료를 가져올 수 없습니다") {
			t.Errorf("가져오기 실패 문구가 없습니다: %v", err)
		}
	})

	t.Run("위치 인자를 받지 않습니다", func(t *testing.T) {
		modelTestDir(t)
		if _, err := runModel(t, "model", "pull", "extra"); err == nil {
			t.Fatal("위치 인자를 받으면 에러여야 합니다")
		}
	})
}

func TestModelCmdGroup(t *testing.T) {
	t.Run("하위 커맨드 없이 부르면 도움말을 냅니다", func(t *testing.T) {
		modelTestDir(t)
		out, err := runModel(t, "model")
		if err != nil {
			t.Fatalf("에러 없이 돌아야 함: %v\n%s", err, out)
		}
		if !strings.Contains(out, "pull") || !strings.Contains(out, "status") {
			t.Errorf("하위 커맨드가 안내에 없습니다:\n%s", out)
		}
	})
}

func TestHumanBytes(t *testing.T) {
	for _, tc := range []struct {
		n    int64
		want string
	}{
		{512, "512 B"},
		{2048, "2.0 KiB"},
		{607298, "593.1 KiB"},
		{17082821, "16.3 MiB"},
		{2266820608, "2.1 GiB"},
	} {
		if got := humanBytes(tc.n); got != tc.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
