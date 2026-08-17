package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWikiFlagOnPathCommands는 위치 인자로 위키를 받던 넷이 --wiki 로도
// 같은 위키를 가리키는지 본다. 회귀 대상은 ADR 0053이 고친 결함이다.
// 커맨드 스물이 --wiki 를 받는데 이 넷만 안 받아서 에이전트가 막혔다.
func TestWikiFlagOnPathCommands(t *testing.T) {
	// reindex 는 위키에 .engram/ 을 쓰므로 검사 대상에서 분리한다.
	for _, name := range []string{"lint", "status", "doctor", "reindex"} {
		t.Run(name+"은 --wiki 와 위치 인자가 같은 결과를 냅니다", func(t *testing.T) {
			dir := makeWiki(t)

			byArg, argErr := runRoot(t, name, dir)
			byFlag, flagErr := runRoot(t, name, "--wiki", dir)

			if (argErr == nil) != (flagErr == nil) {
				t.Fatalf("종료 상태가 다름: 위치 인자 %v, --wiki %v", argErr, flagErr)
			}
			if byArg != byFlag {
				t.Errorf("출력이 다름\n위치 인자:\n%s\n--wiki:\n%s", byArg, byFlag)
			}
		})

		t.Run(name+"은 파일을 주면 무엇이 잘못됐는지 알립니다", func(t *testing.T) {
			dir := makeWiki(t)
			doc := filepath.Join(dir, "context", "a.md")

			out, err := runRoot(t, name, doc)
			if err == nil {
				t.Fatal("위키가 아니라 파일을 주면 실패해야 함")
			}
			// engram.yaml 을 파일 아래에서 찾다 깨지는 것이 아니라
			// 파일이라는 사실 자체를 알려야 한다.
			msg := err.Error() + out
			if !strings.Contains(msg, "파일") || strings.Contains(msg, "engram.yaml") {
				t.Errorf("파일이라는 안내가 아님: %v\n%s", err, out)
			}
		})

		t.Run(name+"은 둘 다 주면 거절합니다", func(t *testing.T) {
			dir := makeWiki(t)

			out, err := runRoot(t, name, dir, "--wiki", dir)
			if err == nil {
				t.Fatal("위치 인자와 --wiki 를 함께 주면 실패해야 함")
			}
			if !strings.Contains(err.Error()+out, "하나만") {
				t.Errorf("어느 쪽을 남기라는 안내가 없음: %v\n%s", err, out)
			}
		})
	}
}

// TestResolveDocPathInsideWiki는 셸의 작업 디렉토리가 위키 루트일 때
// 위키 루트 상대 경로와 절대 --wiki 를 함께 줘도 동작하는지 본다.
// 그 조합에서 filepath.Rel 이 절대 경로와 상대 경로를 섞어 받아 깨졌다.
func TestResolveDocPathInsideWiki(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"inbox", "context"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rel := filepath.Join("inbox", "memo.md")
	if err := os.WriteFile(filepath.Join(dir, rel), []byte("메모\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("작업 디렉토리가 위키 루트여도 절대 경로를 낸다", func(t *testing.T) {
		prev, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(prev) })

		got, err := resolveDocPath(dir, rel)
		if err != nil {
			t.Fatalf("경로를 못 찾음: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Fatalf("루트가 절대 경로면 문서 경로도 절대여야 함: %q", got)
		}
		if _, err := filepath.Rel(dir, got); err != nil {
			t.Errorf("위키 루트 기준 상대 경로를 못 냄: %v", err)
		}
	})
}

// TestSplitRelated는 --related 가 쉼표로 나뉘는지 본다. 나누지 않으면
// "a,b" 한 덩어리가 슬러그가 되어 [[a,b]] 라는 깨진 링크가 박힌다.
func TestSplitRelated(t *testing.T) {
	got := splitRelated([]string{"a,b", " c ", "", "d"})
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("나눈 결과 = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("나눈 결과 = %v, want %v", got, want)
		}
	}
}
