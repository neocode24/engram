package pin_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startPinServer는 engram 을 빌드하고 init 으로 위키를 만들어 serve 를 띄운다. 위키에는 선언 하나로 제안 둘이 들어 있다.
// 돌려주는 것은 서버 주소(http://포함 안 함)와 서버 프로세스다. 프로세스는 시험이 끝나면 정리된다.
func startPinServer(t *testing.T) (addr string, srv *exec.Cmd) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "engram")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/engram").CombinedOutput(); err != nil {
		t.Fatalf("빌드 실패: %v\n%s", err, out)
	}
	wiki := filepath.Join(t.TempDir(), "wiki")
	run := func(stdin string, args ...string) string {
		cmd := exec.Command(bin, args...)
		cmd.Stdin = strings.NewReader(stdin)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run("", "init", wiki)
	decl := filepath.Join(t.TempDir(), "decl.json")
	body := `{"requests":[` +
		`{"op":"add","args":{"title":"견적서 보내기","evidence":[{"section":"","quote":"김대리가 금요일까지 견적서를 보낸다."}]}},` +
		`{"op":"add","args":{"title":"설치 초안 만들기","evidence":[{"section":"","quote":"우리는 설치 초안을 만든다."}]}}]}`
	if err := os.WriteFile(decl, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	run("# 킥오프\n\n김대리가 금요일까지 견적서를 보낸다.\n\n우리는 설치 초안을 만든다.\n", "capture", "--wiki", wiki, "--title", "킥오프", "--pin", decl)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	srv = exec.CommandContext(ctx, bin, "serve", "--wiki", wiki, "--port", "0")
	stdout, err := srv.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Process.Kill(); _ = srv.Wait() })

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if _, rest, ok := strings.Cut(sc.Text(), "Pin 화면: http://"); ok {
			addr = strings.Fields(rest)[0]
			addr = strings.TrimSuffix(addr, "/pin")
			break
		}
	}
	if addr == "" {
		t.Fatal("serve 의 시작 안내에서 /pin 주소를 찾지 못했습니다")
	}

	return addr, srv
}

// TestPinPageInChrome은 engram serve 가 낸 /pin 화면을 크롬으로 열어 확인한다.
// 대역 서버가 없다. 확인하는 것은 browser/probe.mjs 첫머리에 적었다.
//
// 환경변수가 모자라면 건너뛴다(CI 는 크롬과 playwright-core 를 두지 않는다).
//
//	ENGRAM_CHROME            크롬 실행 파일 경로
//	ENGRAM_PLAYWRIGHT_CORE   playwright-core 패키지 디렉터리
func TestPinPageInChrome(t *testing.T) {
	chrome, pw := os.Getenv("ENGRAM_CHROME"), os.Getenv("ENGRAM_PLAYWRIGHT_CORE")
	if chrome == "" || pw == "" {
		t.Skip("ENGRAM_CHROME 과 ENGRAM_PLAYWRIGHT_CORE 가 없어 건너뜁니다")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node 가 없어 건너뜁니다")
	}
	addr, srv := startPinServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	probe, err := filepath.Abs("browser/probe.mjs")
	if err != nil {
		t.Fatal(err)
	}
	// 서버를 내린 뒤 기다리는 시간을 줄여 시험을 짧게 한다. 자동 다시 불러오기가 도는 주기보다 길게 잡는다.
	out, err := exec.CommandContext(ctx, node, probe, "http://"+addr, pw, chrome, strconv.Itoa(srv.Process.Pid), "40").CombinedOutput()
	t.Logf("\n%s", out)
	if err != nil || !strings.Contains(string(out), "PASS") {
		t.Fatalf("크롬 확인이 실패했습니다: %v", err)
	}
}
