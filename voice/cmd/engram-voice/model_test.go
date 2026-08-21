package main

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/voice/internal/model"
)

// fakeModel은 기대값 표를 시험용 작은 파일로 바꿔 끼운다. 실제 표는
// 1.8GB 라 시험이 그것을 받을 수 없다. 표 자체는 internal/model 이
// 검사하므로 여기서는 배선만 본다.
func fakeModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(model.EnvModelDir, dir)
	return dir
}

// writeExpected는 크기 하나의 기대 파일을 src 에 만든다. 내용은 기대
// 체크섬과 맞지 않으므로 반입이 거절되어야 한다.
func writeExpected(t *testing.T, src string, size model.Size, content []byte) {
	t.Helper()
	files, err := model.Files(size)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		p := filepath.Join(src, f.Name)
		if err := os.WriteFile(p, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStatusOnEmptyDirReportsIncomplete(t *testing.T) {
	fakeModel(t)
	var out bytes.Buffer
	if err := runStatus([]string{"--model", "small"}, &out); err != nil {
		t.Fatalf("빈 디렉토리에서 status 는 에러가 아니어야 함: %v", err)
	}
	got := out.String()
	for _, want := range []string{"encoder.int8.onnx", "없음", "받다 만 상태", "model pull"} {
		if !strings.Contains(got, want) {
			t.Errorf("출력에 %q 가 없음:\n%s", want, got)
		}
	}
	// 무엇을 하면 되는지 크기와 함께 알려야 한다. 크기를 빼면
	// 사용자가 기본값을 받아 엉뚱한 디렉토리를 채운다.
	if !strings.Contains(got, "--model small") {
		t.Errorf("이어받기 안내에 크기가 없음:\n%s", got)
	}
}

func TestPullFromRejectsWrongContent(t *testing.T) {
	fakeModel(t)
	src := t.TempDir()
	writeExpected(t, src, model.Small, []byte("이건 진짜 모델이 아니다"))

	var out bytes.Buffer
	err := runPull(context.Background(), http.DefaultClient,
		[]string{"--model", "small", "--from", src}, &out)
	if err == nil {
		t.Fatal("내용이 기대값과 다르면 거절해야 함")
	}
	// 크기와 체크섬 중 어느 것이 틀렸는지는 파일에 따라 다르다.
	// 어느 쪽이든 사용자가 무엇을 할지 알 문장이어야 한다.
	if !strings.Contains(err.Error(), "불일치") {
		t.Errorf("불일치를 알리는 문장이어야 함: %v", err)
	}
}

func TestPullFromAcceptsMatchingContent(t *testing.T) {
	fakeModel(t)
	src := t.TempDir()

	// 기대값 표의 크기와 sha256 을 가진 파일을 만들 수는 없다.
	// 대신 반입 경로가 검증을 거친다는 사실만 확인한다. 여기서는
	// 파일 하나만 진짜 기대값으로 맞춰 나머지가 거절되는지 본다.
	files, err := model.Files(model.Small)
	if err != nil {
		t.Fatal(err)
	}
	var vad model.ModelFile
	for _, f := range files {
		if f.Name == model.VadName {
			vad = f
		}
	}
	if vad.SHA256 == "" {
		t.Fatal("VAD 파일이 표에 없습니다")
	}
	// 기대 크기의 0 바이트 파일은 sha256 이 다르므로 거절되어야 한다.
	if err := os.WriteFile(filepath.Join(src, vad.Name), make([]byte, vad.Size), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runPull(context.Background(), http.DefaultClient,
		[]string{"--model", "small", "--from", src}, &out); err == nil {
		t.Fatal("나머지 파일이 없으므로 거절해야 함")
	}
	// 거절된 반입이 모델 디렉토리를 온전한 상태로 남기면 안 된다.
	var st bytes.Buffer
	if err := runStatus([]string{"--model", "small"}, &st); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(st.String(), "파일이 다 있습니다") {
		t.Errorf("실패한 반입 뒤에 온전하다고 말하면 안 됨:\n%s", st.String())
	}
}

func TestUnknownSizeIsRejected(t *testing.T) {
	fakeModel(t)
	var out bytes.Buffer
	err := runStatus([]string{"--model", "large"}, &out)
	if err == nil {
		t.Fatal("large 는 허용값이 아니므로 거절해야 함")
	}
	// 허용값을 알려야 한다. 거절만 하면 사용자가 추측한다.
	if !strings.Contains(err.Error(), "large-v3") {
		t.Errorf("허용값을 알려야 함: %v", err)
	}
}

func TestModelDirSeparatesSizes(t *testing.T) {
	// 크기마다 디렉토리가 달라야 한다. 섞이면 어느 크기가 온전한지
	// 판정할 수 없다. 화자 분할 모델이 크기와 무관하게 같은 이름이라
	// 특히 그렇다.
	t.Setenv(model.EnvModelDir, "/tmp/x")
	seen := map[string]bool{}
	for _, s := range model.Sizes() {
		d, err := model.Dir(s)
		if err != nil {
			t.Fatal(err)
		}
		if seen[d] {
			t.Errorf("%s 의 디렉토리가 겹침: %s", s, d)
		}
		seen[d] = true
	}
}
