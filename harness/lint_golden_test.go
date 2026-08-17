// Package harness는 ADR 0005의 골든 위키 비교 러너를 담는다.
// 지금 덮는 비교 축은 lint 하나다. 러너는 커맨드 계층을 그대로 호출해
// 바이너리를 exec 하지 않는다. exec 는 빌드 산출물 위치에 의존해서 CI 에서 깨진다.
package harness

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/cli"
)

// update는 go test ./harness -update 로 스냅샷을 재생성할 때 켜지는 플래그다.
var update = flag.Bool("update", false, "골든 스냅샷을 재생성한다")

// goldenDir는 스냅샷 디렉토리다. 테스트는 패키지 디렉토리(harness/)에서
// 돌므로 테스트 파일 기준 상대 경로로만 잡는다. 절대 경로를 쓰지 않는다.
const goldenDir = "golden"

// wikiRoot는 고정 입력 위키의 상대 경로다. filepath.Join 은 상수식이 아니므로
// var 로 둔다.
var wikiRoot = filepath.Join("fixtures", "golden-wiki")

// snapshots는 비교 축 하나와 스냅샷 파일 둘의 대응이다.
var snapshots = []struct {
	name string   // 하위 테스트 이름
	file string   // 스냅샷 파일 이름
	args []string // lint 커맨드에 넘길 인자
}{
	{name: "텍스트 출력", file: "lint.txt", args: []string{"lint", wikiRoot}},
	{name: "JSON 출력", file: "lint.json", args: []string{"lint", "--json", wikiRoot}},
}

// runLint는 커맨드 계층을 호출해 lint 를 돌리고 표준 출력과 종료 코드를 반환한다.
// cli.Execute 는 내부에서 newRootCmd().Execute() 를 돌리므로 인자와 출력을
// os.Args 와 os.Stdout 교체로 주입한다. golden-wiki 에는 승급을 막는 위반이
// 있으므로 종료 코드 1이 정상이다.
func runLint(t *testing.T, args []string) string {
	t.Helper()

	oldArgs := os.Args
	oldStdout := os.Stdout
	os.Args = append([]string{"engram"}, args...)
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = outW

	// 파이프를 읽는 쪽을 먼저 띄운다. 쓰기가 끝난 뒤에 읽으면 출력이
	// 파이프 버퍼보다 클 때 쓰기가 블록되어 영원히 멈춘다. 버퍼 크기는
	// 플랫폼마다 다르고 Windows 가 작아서, 이 하니스는 Windows CI 에서만
	// 10분 타임아웃으로 죽고 있었다. 골든 출력이 커질수록 다른 플랫폼도
	// 같은 자리에 닿는다.
	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(outR)
		done <- b
	}()

	code := cli.Execute()

	os.Args = oldArgs
	os.Stdout = oldStdout
	if err := outW.Close(); err != nil {
		t.Fatal(err)
	}
	out := <-done
	if err := outR.Close(); err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Errorf("golden-wiki 는 error/reject 위반이 있으므로 종료 코드 1이어야 한다, got %d", code)
	}
	return string(out)
}

// normalize는 플랫폼 차이를 없앤다. 줄바꿈은 스냅샷 파일을 git 이
// 체크아웃 설정에 따라 CRLF 로 풀 수 있으므로 LF 로 통일하고,
// 경로 구분자는 Windows 에서 역슬래시로 나올 수 있으므로 슬래시로 통일한다.
// 정규화는 이 골든 러너가 일어나는 유일한 지점이다. lint 본체는 위반 경로를
// 이미 슬래시로 내놓지만 방어적으로 여기서 한 번 더 통일한다.
func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\\", "/")
}

// TestLintGolden은 golden-wiki 를 검사한 출력이 스냅샷과 바이트까지 같은지 본다.
// 차이가 나면 줄 단위 diff 를 보여준다. "다르다" 만 내는 실패 메시지는 쓸모없다.
func TestLintGolden(t *testing.T) {
	for _, s := range snapshots {
		t.Run(s.name, func(t *testing.T) {
			got := normalize(runLint(t, s.args))
			path := filepath.Join(goldenDir, s.file)

			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("스냅샷 쓰기 실패: %v", err)
				}
				return
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("스냅샷 읽기 실패(%s). go test ./harness -update 로 재생성한다: %v", path, err)
			}
			want := normalize(string(raw))
			if got == want {
				return
			}
			t.Fatalf("%s 출력이 스냅샷과 다르다:\n%s", path, diffLines(want, got))
		})
	}
}

// diffLines는 두 텍스트의 줄 단위 차이를 읽을 수 있는 형태로 낸다.
// 표준 라이브러리만 쓰는 제약 안에서 단순 LCS 로 줄을 맞춘다.
// 스냅샷 규모가 수백 줄 이하라는 전제로 충분하다.
func diffLines(want, got string) string {
	a := strings.Split(want, "\n")
	b := strings.Split(got, "\n")

	// lcs[i][j] 는 a[i:] 와 b[j:] 의 최장 공통 부분수열 길이다.
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("--- 스냅샷\n+++ 실제 출력\n")
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			fmt.Fprintf(&sb, "- %s\n", a[i])
			i++
		default:
			fmt.Fprintf(&sb, "+ %s\n", b[j])
			j++
		}
	}
	for ; i < len(a); i++ {
		fmt.Fprintf(&sb, "- %s\n", a[i])
	}
	for ; j < len(b); j++ {
		fmt.Fprintf(&sb, "+ %s\n", b[j])
	}
	return sb.String()
}
