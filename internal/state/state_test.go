package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("파일이 없으면 에러가 아니어야 합니다: %v", err)
	}
	if len(s.BridgeRejections) != 0 {
		t.Errorf("빈 상태여야 합니다: %+v", s.BridgeRejections)
	}
}

func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	var s State
	for _, p := range [][2]string{{"가나다-문서", "라마바-문서"}, {"자차카-문서", "타파하-문서"}, {"a", "b"}} {
		if !s.Reject(p[0], p[1]) {
			t.Fatalf("새 쌍 %v 이 추가돼야 합니다", p)
		}
	}
	if err := s.Save(root); err != nil {
		t.Fatalf("저장 실패: %v", err)
	}

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("읽기 실패: %v", err)
	}
	if len(loaded.BridgeRejections) != len(s.BridgeRejections) {
		t.Fatalf("왕복 결과가 다릅니다: %+v vs %+v", loaded.BridgeRejections, s.BridgeRejections)
	}
	for i := range s.BridgeRejections {
		if loaded.BridgeRejections[i] != s.BridgeRejections[i] {
			t.Errorf("%d번째 쌍 왕복 불일치: %v vs %v", i, loaded.BridgeRejections[i], s.BridgeRejections[i])
		}
	}

	// 같은 상태를 두 번 저장하면 바이트까지 같아야 한다.
	if err := loaded.Save(root); err != nil {
		t.Fatalf("재저장 실패: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(root, StateFileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.Save(root); err != nil {
		t.Fatalf("삼저장 실패: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(root, StateFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("두 번 저장한 바이트가 다릅니다:\n%q\n%q", first, second)
	}
}

func TestSaveFormat(t *testing.T) {
	root := t.TempDir()
	var s State
	s.Reject("가나다-문서", "라마바-문서")
	if err := s.Save(root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, StateFileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "bridge_rejections:\n  - [가나다-문서, 라마바-문서]\n"
	if string(raw) != want {
		t.Errorf("저장 형식:\n got %q\nwant %q", raw, want)
	}
}

func TestPairOrderInvariant(t *testing.T) {
	var s State
	if !s.Reject("b", "a") {
		t.Fatal("새 쌍이어야 합니다")
	}
	if s.Reject("a", "b") {
		t.Error("순서만 다른 쌍은 중복이어야 합니다")
	}
	if got := s.BridgeRejections; len(got) != 1 || got[0] != [2]string{"a", "b"} {
		t.Errorf("쌍이 정렬돼 저장돼야 합니다: %+v", got)
	}
	if !s.IsRejected("b", "a") || !s.IsRejected("a", "b") {
		t.Error("순서와 무관하게 판정해야 합니다")
	}
}

func TestUnreject(t *testing.T) {
	var s State
	s.Reject("a", "b")
	if s.Unreject("a", "c") {
		t.Error("없는 쌍을 지웠다고 보고하면 안 됩니다")
	}
	if !s.Unreject("b", "a") {
		t.Error("순서와 무관하게 지워야 합니다")
	}
	if s.IsRejected("a", "b") {
		t.Error("지운 뒤에는 기각이 아니어야 합니다")
	}
}

func TestHandEditedFileIsCanonicalized(t *testing.T) {
	root := t.TempDir()
	// 손으로 편집해 쌍 순서와 목록 순서가 어긋나고 중복이 있는 파일.
	raw := "bridge_rejections:\n  - [z, a]\n  - [b, a]\n  - [a, b]\n"
	if err := os.WriteFile(filepath.Join(root, StateFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(root)
	if err != nil {
		t.Fatalf("읽기 실패: %v", err)
	}
	want := [][2]string{{"a", "b"}, {"a", "z"}}
	if len(s.BridgeRejections) != len(want) {
		t.Fatalf("정규화 결과: %+v", s.BridgeRejections)
	}
	for i := range want {
		if s.BridgeRejections[i] != want[i] {
			t.Errorf("%d번째: %+v, want %+v", i, s.BridgeRejections[i], want[i])
		}
	}
}

func TestBrokenFileIsError(t *testing.T) {
	cases := map[string]string{
		"문법 오류":   "bridge_rejections:\n  - [a\n",
		"원소 세 개":  "bridge_rejections:\n  - [a, b, c]\n",
		"모르는 키":   "bridge_rejctions:\n  - [a, b]\n",
		"원소 하나":   "bridge_rejections:\n  - [a]\n",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, StateFileName), []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(root)
			if err == nil {
				t.Fatal("파손된 파일은 에러여야 합니다")
			}
			if !strings.Contains(err.Error(), StateFileName) {
				t.Errorf("에러가 파일을 짚어야 합니다: %v", err)
			}
		})
	}
}
