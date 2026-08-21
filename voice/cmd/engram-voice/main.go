// engram-voice는 오디오를 전사 텍스트로 바꾼다.
//
// 위키에 쓰지 않는다. 전사 결과를 낼 뿐이고 위키로 넣는 것은
// engram capture 의 일이다(ADR 0079). 그래서 이 바이너리는 위키 규약을
// 알 필요가 없다.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// version은 릴리스 태그다. 빌드 때 ldflags 로 박는다. 심볼 경로가
// 틀리면 조용히 dev 로 남으므로 바꿀 때 주의한다. 루트의
// internal/cli.version 과 같은 자리다.
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		if errors.Is(err, context.Canceled) {
			// Ctrl-C 다. 이어받기가 되므로 다시 돌리면 된다.
			fmt.Fprintln(os.Stderr, "\n중단했습니다. 다시 실행하면 이어받습니다")
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("커맨드가 필요합니다")
	}
	switch args[0] {
	case "model":
		return runModel(ctx, http.DefaultClient, args[1:], os.Stdout)
	case "transcribe":
		return runTranscribe(args[1:], os.Stdout)
	case "version", "--version":
		fmt.Fprintln(os.Stdout, version)
		return nil
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("모르는 커맨드: %s", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `engram-voice는 오디오를 전사 텍스트로 바꿉니다.

위키에 쓰지 않습니다. 전사 결과를 표준 출력으로 내며 위키에 넣는 것은
engram capture 가 합니다.

사용법:
  engram-voice transcribe <오디오> [--speakers N] [--wiki <위키>] [--json]
  engram-voice model pull [--model <크기>] [--from <경로>]
  engram-voice model status [--model <크기>] [--verify]
  engram-voice version

크기는 large-v3(기본), medium, small 입니다.

화자 수를 아는 값이 있으면 --speakers 로 주세요. 생략하면 추정하는데
그 값은 믿을 수 없습니다.

--wiki 를 주면 그 위키의 meta/terminology.md 를 읽어 전사 뒤에 용어를
교정합니다. 사전은 위키가 소유하고 사람이 채웁니다.

전사 결과는 표준 출력으로 나가고 진행률은 표준 오류로 나갑니다.
그대로 위키에 넣으려면 이렇게 씁니다.

  engram-voice transcribe 회의.m4a --speakers 3 | engram capture --title "회의"
`)
}
