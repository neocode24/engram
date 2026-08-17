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

	"github.com/neocode24/engram/internal/i18n"
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
		Short: i18n.T("cli.serve.short"),
		Long:  i18n.T("cli.serve.long"),
		Args:  cobra.NoArgs,
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
				return fmt.Errorf("%s: %w", i18n.T("cli.serve.flag_read_fail", flagPort), err)
			}
			if port < 0 || port > 65535 {
				return fmt.Errorf("%s", i18n.T("cli.serve.port_range", flagPort, port))
			}
			includeArchive, err := cmd.Flags().GetBool(flagIncludeArchive)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.serve.flag_read_fail", flagIncludeArchive), err)
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
				return fmt.Errorf("%s: %w", i18n.T("cli.serve.listen_fail", addr), err)
			}
			printServeNotice(cmd.OutOrStdout(), root, host, listenAddr(host, ln), exposure)
			return runServer(cmd.Context(), ln, srv.Handler())
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.serve.flag_wiki"))
	cmd.Flags().String(flagHost, defaultServeHost, i18n.T("cli.serve.flag_host"))
	cmd.Flags().Int(flagPort, defaultServePort, i18n.T("cli.serve.flag_port"))
	cmd.Flags().Bool(flagIncludeArchive, false, i18n.T("cli.serve.flag_include_archive"))
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
	fmt.Fprint(w, i18n.T("cli.serve.notice_start", addr)+"\n")
	fmt.Fprint(w, i18n.T("cli.serve.notice_root", root)+"\n")
	fmt.Fprint(w, i18n.T("cli.serve.notice_exposed", e.Context, e.Index))
	if e.IncludeArchive {
		fmt.Fprint(w, i18n.T("cli.serve.notice_exposed_archive", e.Archive))
	}
	fmt.Fprint(w, i18n.T("cli.serve.notice_exposed_total", e.Visible())+"\n")

	fmt.Fprint(w, i18n.T("cli.serve.notice_excluded", e.ExcludedInbox, e.ExcludedSources))
	if !e.IncludeArchive {
		fmt.Fprint(w, i18n.T("cli.serve.notice_excluded_archive", e.ExcludedArchive))
	}
	fmt.Fprintf(w, "\n")
	if e.ExcludedOutside > 0 {
		fmt.Fprint(w, i18n.T("cli.serve.notice_excluded_outside", e.ExcludedOutside)+"\n")
	}
	if e.ExcludedUnparsed > 0 {
		fmt.Fprint(w, i18n.T("cli.serve.notice_excluded_unparsed", e.ExcludedUnparsed)+"\n")
	}
	if e.SensitivityOn {
		fmt.Fprint(w, i18n.T("cli.serve.notice_sensitivity_on", e.ExcludedSensitive)+"\n")
	} else {
		fmt.Fprint(w, i18n.T("cli.serve.notice_sensitivity_off")+"\n")
	}
	fmt.Fprint(w, i18n.T("cli.serve.notice_readonly")+"\n")
	if !loopbackHost(host) {
		fmt.Fprint(w, i18n.T("cli.serve.notice_expose_warning", host)+"\n")
	}
	fmt.Fprint(w, i18n.T("cli.serve.notice_stop")+"\n")
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
