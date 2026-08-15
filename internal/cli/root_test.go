package cli

import (
	"testing"
	"time"
)

func TestParseNow(t *testing.T) {
	t.Run("유효한 RFC3339 문자열은 그 시각을 반환합니다", func(t *testing.T) {
		got, err := parseNow("2026-01-01T00:00:00Z")
		if err != nil {
			t.Fatalf("에러 없이 파싱되어야 함: %v", err)
		}
		if want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("빈 값이면 실제 시계를 씁니다", func(t *testing.T) {
		before := time.Now()
		got, err := parseNow("")
		if err != nil {
			t.Fatalf("빈 값은 에러가 아니어야 함: %v", err)
		}
		if got.Before(before) {
			t.Errorf("빈 값의 결과 %v가 호출 시점 %s보다 과거임", got, before)
		}
	})

	t.Run("잘못된 형식은 에러를 반환합니다", func(t *testing.T) {
		got, err := parseNow("nonsense")
		if err == nil {
			t.Fatal("잘못된 형식은 에러여야 함")
		}
		if !got.IsZero() {
			t.Errorf("에러 시 반환값은 영값이어야 함: %v", got)
		}
	})
}
