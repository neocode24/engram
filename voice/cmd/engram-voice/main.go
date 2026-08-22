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
	"strings"
	"syscall"

	"github.com/neocode24/engram/internal/i18n"
	_ "github.com/neocode24/engram/voice/internal/vi18n"
)

// takeLang은 인자에서 --lang 을 떼어내고 쓸 언어를 정한다.
//
// 하위 커맨드마다 FlagSet 이 달라 --lang 을 각각 선언하면 네 군데에
// 같은 것을 두게 된다. 진입점에서 한 번 걷어내는 편이 낫다.
//
// 값이 없으면 ENGRAM_LANG 을 보고 그것도 없으면 한국어다. 본체와 같은
// 규칙이며 LANG 환경변수는 보지 않는다.
func takeLang(args []string) ([]string, i18n.Lang, error) {
	raw := ""
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lang" || a == "-lang":
			if i+1 < len(args) {
				i++
				raw = args[i]
			}
		case strings.HasPrefix(a, "--lang="):
			raw = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			raw = strings.TrimPrefix(a, "-lang=")
		default:
			rest = append(rest, a)
		}
	}
	lang, err := i18n.Resolve(raw)
	if err != nil {
		return nil, "", err
	}
	return rest, lang, nil
}

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
			fmt.Fprintln(os.Stderr, i18n.T("voice.cmd.interrupted"))
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	// 언어를 먼저 정한다. 그 뒤의 모든 출력이 이 값을 본다.
	// --lang 은 어느 자리에 와도 받는다. 커맨드 앞에 오는 것이
	// 자연스러운데 하위 커맨드의 FlagSet 은 그것을 모른다.
	args, lang, err := takeLang(args)
	if err != nil {
		return err
	}
	i18n.SetLang(lang)

	if len(args) == 0 {
		usage()
		return errors.New(i18n.T("voice.cmd.need_command"))
	}
	switch args[0] {
	case "model":
		return runModel(ctx, http.DefaultClient, args[1:], os.Stdout)
	case "transcribe":
		return runTranscribe(args[1:], os.Stdout)
	case "mcp":
		return runMCP(ctx, args[1:])
	case "version", "--version":
		fmt.Fprintln(os.Stdout, version)
		return nil
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf(i18n.T("voice.cmd.unknown"), args[0])
	}
}

// usage는 사용법을 낸다. 문구는 카탈로그에 있다.
func usage() {
	fmt.Fprint(os.Stderr, i18n.T("voice.usage"))
}
