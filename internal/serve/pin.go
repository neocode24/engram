package serve

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/expose"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/pin"
)

// pinBodyMax는 변경 요청 본문의 상한이다.
const pinBodyMax = 8 << 10

// 경로 상수. 화면 파일(internal/pin/web/template.html)이 API 를 /pin/api/ 로 부른다.
const (
	pinPage    = "/pin"
	pinState   = "/pin/api/state"
	pinChanges = "/pin/api/changes"
)

// PinEnabled는 /pin 경로를 등록할지 판정한다. 위키에 원장이 있고 리스너가 loopback 일 때만 참이다.
// 그 밖의 주소에 묶이면 경로를 등록하지 않는다. 인증 없는 쓰기와 미검수 문서의 인용이 네트워크에
// 나가기 때문이다. ADR 0044 가 막은 바로 그것이다.
func (s *Server) PinEnabled() bool { return s.opts.Loopback && pin.Present(s.root) }

// registerPin은 Pin 경로를 등록한다. /pin 은 화면, 나머지 셋은 API 다.
func (s *Server) registerPin(mux *http.ServeMux) {
	if !s.PinEnabled() {
		return
	}
	mux.HandleFunc(pinPage, s.handlePinPage)
	mux.HandleFunc(pinState, s.handlePinState)
	mux.HandleFunc(pinChanges, s.handlePinChanges)
}

// isPinPath는 요청 경로가 Pin 경로인지 본다. 보안 정책과 메서드 판정이 쓴다.
func isPinPath(p string) bool {
	return p == pinPage || p == pinState || p == pinChanges
}

// pinWritable은 readOnly 가 열어 줄 요청이다. POST /pin/api/changes 한 경로뿐이다.
func (s *Server) pinWritable(r *http.Request) bool {
	return r.Method == http.MethodPost && r.URL.Path == pinChanges && s.PinEnabled()
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

func (s *Server) handlePinPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	noStore(w)
	// 이 경로의 보안 정책은 다른 경로와 다르다. 화면 파일의 인라인 script 와 style 을 해시로 허용한다.
	w.Header().Set("Content-Security-Policy", pin.PageCSP())
	_, _ = w.Write(pin.Page())
}

func (s *Server) handlePinState(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	data, err := pin.State(s.root, time.Now())
	if err != nil {
		s.pinFail(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

// handlePinChanges는 GET 에는 빈 목록을 낸다. 대기열이 없으므로 화면이 기다릴 반영 대기가 없다.
// POST 는 받는 즉시 잠금을 잡고 게이트를 돌려 {ok, id, at} 으로 답한다.
func (s *Server) handlePinChanges(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"changes":[],"has_more":false}`)
		return
	}
	if msg := s.checkPinWrite(r); msg != "" {
		s.pinFailText(w, http.StatusForbidden, msg)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, pinBodyMax))
	if err != nil {
		s.pinFailText(w, http.StatusRequestEntityTooLarge, i18n.T("core.serve.pin_too_large", pinBodyMax))
		return
	}
	id, at, err := pin.ApplyWeb(s.root, body, time.Now())
	var bad *pin.BadRequest
	switch {
	case errors.As(err, &bad):
		s.pinFailText(w, http.StatusBadRequest, bad.Msg)
	case err != nil:
		s.pinFail(w, http.StatusInternalServerError, err)
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, pin.JSONText(map[string]any{"ok": true, "id": id, "at": at}))
	}
}

// checkPinWrite는 쓰기 요청을 검사하고 거절 사유를 낸다. 통과면 빈 글이다.
//   - Host 가 loopback 인가: DNS 를 바꿔 치는 공격(rebinding)은 여기서 걸린다.
//   - Origin 이 있으면 자기 출처인가: 다른 사이트가 브라우저를 빌려 쓰는 요청은 여기서 걸린다.
//   - X-Pin: 1 과 Content-Type: application/json: 단순 요청(폼)으로는 보낼 수 없는 헤더다.
//
// 역방향 프록시나 터널이 Host 를 고쳐 loopback 으로 넘기면 이 검사를 통과한다. 원격 접속은 지원하지
// 않는다(ADR 0106).
func (s *Server) checkPinWrite(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if !expose.LoopbackHost(host) {
		return i18n.T("core.serve.pin_bad_host", r.Host)
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			return i18n.T("core.serve.pin_bad_origin")
		}
	}
	if r.Header.Get("X-Pin") != "1" {
		return i18n.T("core.serve.pin_no_header")
	}
	if ct := strings.ToLower(strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])); ct != "application/json" {
		return i18n.T("core.serve.pin_bad_type")
	}
	return ""
}

func (s *Server) pinFail(w http.ResponseWriter, code int, err error) {
	if s.opts.ErrorLog != nil {
		_, _ = io.WriteString(s.opts.ErrorLog, i18n.T("core.serve.fail_log", err)+"\n")
	}
	s.pinFailText(w, code, i18n.T("core.serve.fail_read"))
}

// pinFailText는 화면이 읽는 모양({error})으로 오류를 낸다.
func (s *Server) pinFailText(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = io.WriteString(w, pin.JSONText(map[string]string{"error": msg}))
}
