package model

import (
	"strings"
	"testing"
)

func TestFilesTable(t *testing.T) {
	// 크기와 sha256 은 실제로 받아 계산한 값이다. 상수 하나가 바뀌면
	// 이 시험이 잡는다(ADR 0081).
	want := map[Size]int64{
		Large:  1_810_671_849,
		Medium: 980_990_201,
		Small:  410_403_258,
	}
	for size, total := range want {
		got, err := TotalSize(size)
		if err != nil {
			t.Fatalf("%s: %v", size, err)
		}
		if got != total {
			t.Errorf("%s 합계가 %d 여야 함: %d", size, total, got)
		}
	}
}

func TestFilesAreFlatAndComplete(t *testing.T) {
	for _, size := range Sizes() {
		files, err := Files(size)
		if err != nil {
			t.Fatalf("%s: %v", size, err)
		}
		// whisper 셋, 화자 분할 둘, VAD 하나.
		if len(files) != 6 {
			t.Errorf("%s: 파일이 여섯이어야 함: %d", size, len(files))
		}
		seen := map[string]bool{}
		for _, f := range files {
			// 저장 이름은 평평해야 한다. 원격 경로에는 계층이 있다.
			if strings.Contains(f.Name, "/") {
				t.Errorf("%s: 저장 이름이 평평해야 함: %q", size, f.Name)
			}
			if seen[f.Name] {
				t.Errorf("%s: 저장 이름이 겹침: %q", size, f.Name)
			}
			seen[f.Name] = true
			// 호스트가 둘이므로 파일마다 Base 가 있어야 한다. 비면
			// 호출자의 기본 base 로 떨어져 엉뚱한 곳을 친다.
			if f.Base == "" {
				t.Errorf("%s: Base 가 비었음: %q", size, f.Name)
			}
			if f.SHA256 == "" || f.Size == 0 {
				t.Errorf("%s: 기대값이 비었음: %q", size, f.Name)
			}
		}
	}
}

func TestParseSizeRejectsUnknown(t *testing.T) {
	if _, err := ParseSize("large"); err == nil {
		t.Error("large 는 허용값이 아니므로 거절해야 함")
	}
	got, err := ParseSize("medium")
	if err != nil || got != Medium {
		t.Errorf("medium 을 받아야 함: %v %v", got, err)
	}
}
