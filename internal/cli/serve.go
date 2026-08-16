package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/neocode24/engram/internal/serve"
	"github.com/spf13/cobra"
)

// serve 커맨드의 플래그 이름이다.
const (
	flagHost           = "host"
	flagPort           = "port"
	flagIncludeArchive = "include-archive"
)

// defaultServeHost는 기본 바인딩 주소다. 기본이 전체 노출이면 사고가
// 나므로 루프백에 묶는다(ADR 0044). 외부 노출은 사용자가 --host 로
// 명시한다.
const defaultServeHost = "127.0.0.1"

// defaultServePort는 기본 포트다. 흔한 개발 서버 포트(3000, 5000, 8000,
// 8080, 8443)와 겹치면 다른 프로세스가 이미 쓰고 있을 확률이 높아 그
// 대역을 피했다. 등록 포트가 아닌 값이며 --port 로 바꾼다.
const defaultServePort = 8420

// shutdownGrace는 종료 신호를 받고 진행 중인 응답을 기다리는 시간이다.
// 읽기 전용이라 기다릴 것이 많지 않다.
const shutdownGrace = 3 * time.Second

// newServeCmd는 위키를 읽기 전용 웹 뷰어로 노출하는 serve 커맨드를 반환한다.
func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "위키를 읽기 전용 웹 뷰어로 노출합니다",
		Long: `위키를 읽기 전용 웹 뷰어로 노출합니다.

쓰기 경로가 없습니다. 제안 접수도 두지 않으므로 GET 과 HEAD 외의
요청은 전부 거절합니다.

검수를 지난 context 문서와 색인 문서만 보입니다. inbox 와 sources 는
목록에도 URL 에도 없습니다. archive 는 --include-archive 로 엽니다.
sensitivity 축이 켜진 위키에서는 private-local-only 와 restricted 문서를
빼며 이 제외를 뒤집는 플래그는 없습니다. 내보내야 하면 문서의 값을
고치세요.

기본 바인딩은 127.0.0.1 입니다. --host 로 외부에 노출하면 인증이 없으므로
접근할 수 있는 모두가 위 문서를 읽습니다.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			host, err := stringFlag(cmd, flagHost)
			if err != nil {
				return err
			}
			port, err := cmd.Flags().GetInt(flagPort)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagPort, err)
			}
			if port < 0 || port > 65535 {
				return fmt.Errorf("--%s 값이 포트 범위 밖입니다: %d (0부터 65535까지)", flagPort, port)
			}
			includeArchive, err := cmd.Flags().GetBool(flagIncludeArchive)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagIncludeArchive, err)
			}

			srv := serve.New(root, cfg, serve.Options{
				IncludeArchive: includeArchive,
				ErrorLog:       cmd.ErrOrStderr(),
			})
			// 노출 집계를 서버를 띄우기 전에 낸다. 무엇이 나가는지 모르고
			// 띄우는 일이 없어야 한다.
			exposure, err := srv.Exposure()
			if err != nil {
				return err
			}
			addr := net.JoinHostPort(host, strconv.Itoa(port))
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("주소 %s 를 열 수 없습니다: %w", addr, err)
			}
			printServeNotice(cmd.OutOrStdout(), root, host, listenAddr(host, ln), exposure)
			return runServer(cmd.Context(), ln, srv.Handler())
		},
	}
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	cmd.Flags().String(flagHost, defaultServeHost, "바인딩 주소")
	cmd.Flags().Int(flagPort, defaultServePort, "바인딩 포트")
	cmd.Flags().Bool(flagIncludeArchive, false, "archive 문서도 노출합니다")
	return cmd
}

// listenAddr는 안내에 낼 주소를 만든다. 포트는 실제로 열린 것을 쓴다.
// --port 0 을 주면 운영체제가 고른 포트를 알려 줘야 하기 때문이다.
// 주소는 사용자가 준 것을 그대로 쓴다. 0.0.0.0 에 묶으면 리스너가
// [::] 를 말하는데 그것은 사용자가 적은 것과 달라 헷갈린다.
func listenAddr(host string, ln net.Listener) string {
	if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
		return net.JoinHostPort(host, strconv.Itoa(tcp.Port))
	}
	return ln.Addr().String()
}

// runServer는 서버를 돌리고 context 가 끝나면 정리한다.
func runServer(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// printServeNotice는 시작 안내를 낸다. 어느 주소에서 뜨는지, 무엇이
// 노출되고 무엇이 제외되는지, 인증이 없는지를 알린다.
func printServeNotice(w io.Writer, root, host, addr string, e serve.Exposure) {
	fmt.Fprintf(w, "engram serve 를 시작합니다: http://%s\n", addr)
	fmt.Fprintf(w, "대상 위키: %s\n", root)
	fmt.Fprintf(w, "노출: context 문서 %d개, 색인 문서 %d개", e.Context, e.Index)
	if e.IncludeArchive {
		fmt.Fprintf(w, ", archive 문서 %d개", e.Archive)
	}
	fmt.Fprintf(w, " (모두 %d개)\n", e.Visible())

	fmt.Fprintf(w, "제외: inbox %d개, sources %d개", e.ExcludedInbox, e.ExcludedSources)
	if !e.IncludeArchive {
		fmt.Fprintf(w, ", archive %d개(--include-archive 로 엽니다)", e.ExcludedArchive)
	}
	fmt.Fprintf(w, "\n")
	if e.ExcludedOutside > 0 {
		fmt.Fprintf(w, "제외: 네 단계 밖에 있는 문서 %d개\n", e.ExcludedOutside)
	}
	if e.ExcludedUnparsed > 0 {
		fmt.Fprintf(w, "제외: 프론트매터를 읽을 수 없어 판정하지 못한 문서 %d개 (engram lint 로 확인하세요)\n",
			e.ExcludedUnparsed)
	}
	if e.SensitivityOn {
		fmt.Fprintf(w, "민감도: private-local-only 와 restricted 문서 %d개를 제외했습니다. 뒤집는 플래그는 없습니다\n",
			e.ExcludedSensitive)
	} else {
		fmt.Fprintf(w, "민감도: 이 위키는 sensitivity 축이 꺼져 있어 거를 값이 없습니다\n")
	}
	fmt.Fprintf(w, "쓰기 경로가 없습니다. GET 과 HEAD 외의 요청은 거절합니다\n")
	if !loopbackHost(host) {
		fmt.Fprintf(w, "경고: %s 에 바인딩했습니다. 인증이 없으므로 이 주소에 닿는 모두가 위 문서를 읽습니다\n", host)
	}
	fmt.Fprintf(w, "멈추려면 Ctrl+C 를 누르세요\n")
}

// loopbackHost는 바인딩 주소가 이 컴퓨터 밖으로 나가지 않는 주소인지 본다.
// 이름으로 준 주소는 여기서 풀지 않고 외부 노출로 본다. 경고를 빠뜨리는
// 쪽보다 더 내는 쪽이 안전하다.
func loopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
