package embed

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTruncate는 자르기가 룬 기준인지 본다. 바이트 기준으로 자르면
// 한국어 한 글자가 3바이트라 upstream 이 보는 양의 3분의 1만 보게
// 되고, 경계에서 글자가 반토막 나 깨진 문자가 들어간다.
func TestTruncate(t *testing.T) {
	short := "짧은 글"
	if got := Truncate(short); got != short {
		t.Errorf("한도 안의 글은 그대로여야 한다: %q", got)
	}

	long := strings.Repeat("가", Chars+500)
	got := Truncate(long)
	if n := len([]rune(got)); n != Chars {
		t.Errorf("룬 %d개여야 하는데 %d개다", Chars, n)
	}
	if !strings.HasPrefix(long, got) {
		t.Error("자른 결과가 원문의 접두사가 아니다")
	}
	if strings.ContainsRune(got, '�') {
		t.Error("깨진 문자가 들어갔다. 바이트 기준으로 잘랐다는 뜻이다")
	}

	// 바이트로 잘랐다면 3배 짧아진다. 그 실수를 잡는 단언이다.
	if len(got) <= Chars {
		t.Errorf("한글 %d자의 바이트 길이가 %d다. 바이트 기준으로 자른 것으로 보인다", Chars, len(got))
	}
}

// TestKey는 캐시 키가 자른 뒤의 내용만 본다는 것을 확인한다. 한도
// 밖의 글자를 고쳐도 키가 같아야 인코딩을 다시 하지 않는다.
func TestKey(t *testing.T) {
	base := strings.Repeat("나", Chars)
	if Key(base+"뒤에 붙는 글") != Key(base+"다른 글") {
		t.Error("한도 밖의 차이가 키를 바꿨다")
	}
	if Key(base) == Key(strings.Repeat("다", Chars)) {
		t.Error("내용이 다른데 키가 같다")
	}
	if n := len(Key("아무거나")); n != 64 {
		t.Errorf("sha256 16진 표기는 64자여야 하는데 %d자다", n)
	}
}

// TestModelDir는 환경변수 덮어쓰기와 기본 경로를 본다.
func TestModelDir(t *testing.T) {
	t.Setenv(EnvModelDir, filepath.FromSlash("/tmp/어디든"))
	got, err := ModelDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.FromSlash("/tmp/어디든") {
		t.Errorf("환경변수를 안 따랐다: %q", got)
	}

	t.Setenv(EnvModelDir, "")
	got, err = ModelDir()
	if err != nil {
		t.Fatal(err)
	}
	// 위키 로컬 캐시가 아니라 사용자 전역 캐시여야 한다(ADR 0074).
	if strings.Contains(filepath.ToSlash(got), "/.engram/") {
		t.Errorf("모델이 위키 로컬 .engram 에 놓인다: %q", got)
	}
	if want := filepath.Join("engram", "models", ModelName); !strings.HasSuffix(got, want) {
		t.Errorf("기본 경로가 %q 로 끝나야 하는데 %q 다", want, got)
	}
}

// TestPresentWithoutModel은 모델이 없는 디렉토리에서 Present 가
// 거짓이고 Open 이 ErrNoModel 을 내는지 본다. 이 경로가 성립해야
// 시맨틱의 부재가 결손이 아니라 성능 저하가 된다(ADR 0007).
func TestPresentWithoutModel(t *testing.T) {
	t.Setenv(EnvModelDir, t.TempDir())
	if Present() {
		t.Fatal("빈 디렉토리인데 모델이 있다고 한다")
	}
	if _, err := Open(); err != ErrNoModel {
		t.Errorf("ErrNoModel 이 아니라 %v 를 냈다", err)
	}
}
