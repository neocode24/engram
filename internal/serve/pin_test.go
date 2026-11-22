package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neocode24/engram/internal/pin"
)

// pinWiki는 Pin 을 켠 시험 위키다. 제안 하나가 들어 있다.
func pinWiki(t *testing.T) string {
	t.Helper()
	root := makeWiki(t, "personal")
	if _, err := pin.Init(root); err != nil {
		t.Fatal(err)
	}
	d, err := pin.ParseDeclaration(`{"requests":[{"op":"add","args":{"title":"허브 정리","evidence":[{"section":"","quote":"[[peer]] 로 이어집니다."}]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	// 근거는 노출되는 context 문서다. 인용문 대조는 디스크의 파일을 본다.
	rel := "context/hub.md"
	text, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pin.Run(pin.Txn{Wiki: root, Now: pinTime(), Decl: d,
		Doc: pin.Doc{Path: rel, Stage: "context", Date: "2026-08-01", Text: string(text)}, Commit: func() error { return nil }}); err != nil {
		t.Fatalf("시험 원장을 만들지 못했습니다: %v", err)
	}
	return root
}

func pinServer(t *testing.T, root string, loopback bool) http.Handler {
	t.Helper()
	return newTestServer(t, root, Options{Loopback: loopback})
}

type resp struct {
	code int
	body string
	hdr  http.Header
}

func send(t *testing.T, h http.Handler, method, target, body string, hdr map[string]string) resp {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Host = "127.0.0.1:8420"
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{res.StatusCode, string(b), res.Header}
}

var okWrite = map[string]string{"Content-Type": "application/json", "X-Pin": "1", "Origin": "http://127.0.0.1:8420"}

func TestPinRoutesExistOnlyForLoopbackWithPinEnabled(t *testing.T) {
	for name, tc := range map[string]struct {
		root     func(*testing.T) string
		loopback bool
		want     int
	}{
		"loopback 이고 Pin 이 켜짐": {pinWiki, true, http.StatusOK},
		"바깥 주소에 묶임":            {pinWiki, false, http.StatusNotFound},
		"Pin 을 켜지 않음":          {func(t *testing.T) string { return makeWiki(t, "personal") }, true, http.StatusNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			h := pinServer(t, tc.root(t), tc.loopback)
			for _, p := range []string{"/pin", "/pin/api/state", "/pin/api/changes"} {
				if r := send(t, h, http.MethodGet, p, "", nil); r.code != tc.want {
					t.Errorf("GET %s: %d, 기대 %d", p, r.code, tc.want)
				}
			}
			r := send(t, h, http.MethodPost, "/pin/api/changes", `{"op":"touch"}`, okWrite)
			if tc.want == http.StatusOK && r.code == http.StatusMethodNotAllowed {
				t.Errorf("Pin 을 연 서버가 POST 를 막았습니다")
			}
			if tc.want == http.StatusNotFound && r.code != http.StatusMethodNotAllowed {
				t.Errorf("Pin 을 열지 않은 서버의 POST 가 %d 입니다. 405 여야 합니다", r.code)
			}
		})
	}
}

func TestPinPageHeadersAndOtherPathsKeepPolicy(t *testing.T) {
	h := pinServer(t, pinWiki(t), true)
	page := send(t, h, http.MethodGet, "/pin", "", nil)
	csp := page.hdr.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'sha256-") || !strings.Contains(csp, "connect-src 'self'") || !strings.Contains(csp, "img-src 'none'") {
		t.Errorf("/pin 의 보안 정책: %s", csp)
	}
	if page.hdr.Get("Cache-Control") != "no-store" {
		t.Errorf("/pin 이 캐시됩니다")
	}
	if !strings.Contains(page.body, "var API='/pin/api/', SOURCES=['wiki'], AUTH='none';") {
		t.Errorf("화면이 engram 설정이 아닙니다")
	}
	// 다른 경로의 정책은 그대로다. 스크립트를 허용하지 않는다.
	other := send(t, h, http.MethodGet, "/", "", nil).hdr.Get("Content-Security-Policy")
	if strings.Contains(other, "script-src") || !strings.Contains(other, "default-src 'none'") {
		t.Errorf("다른 경로의 정책이 바뀌었습니다: %s", other)
	}
	// Pin 이 없는 서버의 /pin 404 도 다른 경로와 같은 정책을 받는다.
	off := pinServer(t, makeWiki(t, "personal"), true)
	got := send(t, off, http.MethodGet, "/pin", "", nil).hdr.Get("Content-Security-Policy")
	if strings.Contains(got, "script-src") || !strings.Contains(got, "default-src 'none'") {
		t.Errorf("Pin 이 꺼진 서버의 /pin 이 다른 경로와 같은 정책을 받지 못했습니다: %q", got)
	}
}

func TestPinWriteChecksRequestHeaders(t *testing.T) {
	h := pinServer(t, pinWiki(t), true)
	body := `{"op":"accept","args":{"proposal":"Q0001"}}`
	copyWith := func(over map[string]string, drop ...string) map[string]string {
		m := map[string]string{}
		for k, v := range okWrite {
			m[k] = v
		}
		for k, v := range over {
			m[k] = v
		}
		for _, k := range drop {
			delete(m, k)
		}
		return m
	}
	for name, tc := range map[string]struct {
		hdr  map[string]string
		want int
	}{
		"X-Pin 없음":                  {copyWith(nil, "X-Pin"), 403},
		"X-Pin 값이 1이 아님":            {copyWith(map[string]string{"X-Pin": "0"}), 403},
		"다른 출처":                     {copyWith(map[string]string{"Origin": "http://evil.example"}), 403},
		"출처 포트가 다름":                 {copyWith(map[string]string{"Origin": "http://127.0.0.1:9999"}), 403},
		"Host 가 바깥 이름":              {copyWith(map[string]string{"Host": "evil.example"}), 403},
		"Host 가 바깥 이름과 loopback 출처": {copyWith(map[string]string{"Host": "evil.example:8420", "Origin": "http://evil.example:8420"}), 403},
		"폼 전송 형식":                   {copyWith(map[string]string{"Content-Type": "application/x-www-form-urlencoded"}), 403},
		"text/plain":                {copyWith(map[string]string{"Content-Type": "text/plain"}), 403},
		"Content-Type 없음":           {copyWith(nil, "Content-Type"), 403},
	} {
		t.Run(name, func(t *testing.T) {
			if r := send(t, h, http.MethodPost, "/pin/api/changes", body, tc.hdr); r.code != tc.want {
				t.Errorf("%d, 기대 %d: %s", r.code, tc.want, r.body)
			}
		})
	}
	t.Run("Origin 없이 보낸 요청은 받는다", func(t *testing.T) {
		if r := send(t, h, http.MethodPost, "/pin/api/changes", body, copyWith(nil, "Origin")); r.code != http.StatusCreated {
			t.Errorf("%d: %s", r.code, r.body)
		}
	})
	t.Run("charset 이 붙은 JSON 형식", func(t *testing.T) {
		wiki := pinWiki(t)
		h := pinServer(t, wiki, true)
		if r := send(t, h, http.MethodPost, "/pin/api/changes", body, copyWith(map[string]string{"Content-Type": "application/json; charset=utf-8"})); r.code != http.StatusCreated {
			t.Errorf("%d: %s", r.code, r.body)
		}
	})
	t.Run("본문 8KB 초과", func(t *testing.T) {
		big := `{"op":"add","args":{"title":"` + strings.Repeat("가", 5000) + `"}}`
		if r := send(t, h, http.MethodPost, "/pin/api/changes", big, okWrite); r.code != http.StatusRequestEntityTooLarge {
			t.Errorf("%d", r.code)
		}
	})
	t.Run("화면이 낼 수 없는 연산", func(t *testing.T) {
		if r := send(t, h, http.MethodPost, "/pin/api/changes", `{"op":"move_path","args":{"from":"a","to":"b"}}`, okWrite); r.code != http.StatusBadRequest {
			t.Errorf("%d: %s", r.code, r.body)
		}
	})
}

func TestPinWriteAnswersAndShowsInState(t *testing.T) {
	h := pinServer(t, pinWiki(t), true)
	before := send(t, h, http.MethodGet, "/pin/api/state", "", nil)
	if before.code != 200 || !strings.Contains(before.body, `"proposed": true`) {
		t.Fatalf("제안이 상태에 없습니다: %d %s", before.code, before.body[:200])
	}
	r := send(t, h, http.MethodPost, "/pin/api/changes", `{"op":"accept","args":{"proposal":"Q0001"}}`, okWrite)
	if r.code != http.StatusCreated || !strings.Contains(r.body, `"ok":true`) || !strings.Contains(r.body, `"id"`) || !strings.Contains(r.body, `"at"`) {
		t.Fatalf("응답 모양이 {ok,id,at} 이 아닙니다: %d %s", r.code, r.body)
	}
	after := send(t, h, http.MethodGet, "/pin/api/state", "", nil)
	if !strings.Contains(after.body, `"proposed": false`) {
		t.Error("맞음이 다음 상태에 반영되지 않았습니다")
	}
	if !strings.Contains(after.body, `"url": "/doc/context/hub.md"`) {
		t.Error("context 문서 근거에 문서 링크가 달리지 않았습니다")
	}
	if c := send(t, h, http.MethodGet, "/pin/api/changes", "", nil); c.body != `{"changes":[],"has_more":false}` {
		t.Errorf("changes 는 항상 빈 목록이어야 합니다: %s", c.body)
	}
}

// 쓰기 요청이 pin/ 밖을 건드리지 않는다. 위키 전체를 해시로 견주고 pin/ 만 바뀐 것을 본다.
func TestPinWriteTouchesOnlyPinDirectory(t *testing.T) {
	root := pinWiki(t)
	h := pinServer(t, root, true)
	before := hashLines(t, root)
	for _, body := range []string{
		`{"op":"accept","args":{"proposal":"Q0001"}}`,
		`{"op":"set_status","item":"P0001","base_rev":2,"args":{"status":"doing","note":"시작"}}`,
		`{"op":"add","args":{"title":"직접 넣은 일"}}`,
		`{"op":"delete","item":"P0001","base_rev":3}`,
		`{"op":"set_work","item":"P0002","base_rev":1,"args":{"new":{"name":"새 프로젝트"}}}`,
		`{"op":"touch","item":"P9999","base_rev":1}`,
	} {
		send(t, h, http.MethodPost, "/pin/api/changes", body, okWrite)
	}
	for _, p := range []string{"/pin", "/pin/api/state", "/pin/api/changes"} {
		send(t, h, http.MethodGet, p, "", nil)
	}
	after := hashLines(t, root)
	var changed []string
	for path, hash := range after {
		if before[path] != hash && !strings.HasPrefix(path, "pin/") {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok && !strings.HasPrefix(path, "pin/") {
			changed = append(changed, path+" (삭제)")
		}
	}
	if len(changed) > 0 {
		t.Fatalf("pin/ 밖의 파일이 바뀌었습니다: %v", changed)
	}
	pinChanged := 0
	for path, hash := range after {
		if strings.HasPrefix(path, "pin/") && before[path] != hash {
			pinChanged++
		}
	}
	if pinChanged == 0 {
		t.Fatal("pin/ 도 바뀌지 않아 시험이 아무것도 보지 못했습니다")
	}
}

func hashLines(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range strings.Split(hashTree(t, root), "\n") {
		if f := strings.SplitN(line, " ", 2); len(f) == 2 && !strings.HasSuffix(f[0], pinLockName) {
			out[f[0]] = f[1]
		}
	}
	return out
}

const pinLockName = "pin/.lock"

func TestPinRequestsOtherMethodsStillBlocked(t *testing.T) {
	h := pinServer(t, pinWiki(t), true)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/pin"}, {http.MethodPost, "/pin/api/state"}, {http.MethodPut, "/pin/api/changes"},
		{http.MethodDelete, "/pin/api/changes"}, {http.MethodPatch, "/pin/api/changes"}, {http.MethodPost, "/search"},
		{http.MethodPost, "/doc/context/hub.md"},
	} {
		if r := send(t, h, tc.method, tc.path, "{}", okWrite); r.code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s 가 %d 입니다. 405 여야 합니다", tc.method, tc.path, r.code)
		}
	}
}

// Pin 화면은 inbox 와 sources 문서의 전문을 열지 않는다. 근거의 제목, 경로, 절, 인용만 나간다(ADR 0044 의 예외 범위).
func TestPinDoesNotOpenInboxOrSourcesDocuments(t *testing.T) {
	root := pinWiki(t)
	h := pinServer(t, root, true)
	for _, p := range []string{"/doc/inbox/2026-08-01-rough.md", "/doc/sources/2026-08-01-src.md"} {
		if r := send(t, h, http.MethodGet, p, "", nil); r.code != http.StatusNotFound {
			t.Errorf("%s 가 %d 입니다. 열리면 안 됩니다", p, r.code)
		}
	}
	d, err := pin.ParseDeclaration(`{"requests":[{"op":"add","args":{"title":"러프 메모의 일","evidence":[{"section":"","quote":"승급파이프라인 이야기입니다."}]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(filepath.Join(root, "inbox/2026-08-01-rough.md"))
	if _, err := pin.Run(pin.Txn{Wiki: root, Now: pinTime(), Decl: d,
		Doc: pin.Doc{Path: "inbox/2026-08-01-rough.md", Stage: "inbox", Date: "2026-08-01", Text: string(text)}, Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	state := send(t, h, http.MethodGet, "/pin/api/state", "", nil).body
	if !strings.Contains(state, "승급파이프라인 이야기입니다.") || !strings.Contains(state, "inbox/2026-08-01-rough.md") {
		t.Error("inbox 근거의 인용문과 경로가 화면 상태에 없습니다")
	}
	if strings.Contains(state, `"url": "/doc/inbox`) {
		t.Error("inbox 근거에 문서 링크가 달렸습니다")
	}
	if strings.Contains(state, "러프 메모\\n") || strings.Contains(state, "# 러프 메모") {
		t.Error("inbox 문서의 본문이 상태에 실렸습니다")
	}
	if u, _ := url.Parse("/doc/inbox/2026-08-01-rough.md"); u == nil {
		t.Fatal()
	}
}

func pinTime() time.Time { return time.Date(2026, 11, 22, 10, 0, 0, 0, time.FixedZone("KST", 9*3600)) }
