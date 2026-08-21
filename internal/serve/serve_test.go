package serve

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
)

// contextDoc는 context 문서 하나의 원문을 만든다. sensitivity 를 빈
// 문자열로 주면 그 키를 넣지 않는다.
func contextDoc(title, sensitivity, body string) string {
	fm := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"
	if sensitivity != "" {
		fm += "sensitivity: " + sensitivity + "\n"
	}
	fm += "indexable: true\ncreated: 2026-01-01\nupdated: 2026-08-01\n---\n\n"
	return fm + "# " + title + "\n\n" + body + "\n"
}

// wikiFiles는 시험용 위키의 파일 집합이다. preset 은 호출자가 정한다.
// team 프리셋은 sensitivity 속성을 켜고 personal 은 끈다.
func wikiFiles(preset string) map[string]string {
	return map[string]string{
		"engram.yaml": "preset: " + preset + "\n",
		"index.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n---\n\n" +
			"# 위키 색인\n\n[[hub]] 부터 봅니다\n",
		"context/hub.md": contextDoc("허브 문서", "internal",
			"[[peer]] 로 이어집니다.\n\n제외 대상 링크입니다. [[secret]] 과 [[limited]] 와 [[2026-08-01-rough]] 입니다.\n\n"+
				"코드 안의 위키링크는 링크가 아닙니다: `[[peer]]`\n\n"+
				"<script>alert('침입')</script>\n\n인라인 원시 HTML 도 <img src=x onerror=\"alert(1)\"> 이렇게 막습니다.\n\n"+
				"검색용 낱말은 승급파이프라인입니다.\n"),
		"context/peer.md":    contextDoc("이웃 문서", "public-reference", "[[hub]] 를 가리킵니다.\n"),
		"context/secret.md":  contextDoc("로컬 전용 문서", "private-local-only", "[[hub]] 를 가리킵니다.\n비밀낱말은 살구입니다.\n"),
		"context/limited.md": contextDoc("제한 공개 문서", "restricted", "[[hub]] 를 가리킵니다.\n"),
		"archive/old.md":     contextDoc("보관 문서", "internal", "[[hub]] 를 가리킵니다.\n"),
		"inbox/2026-08-01-rough.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\ncreated: 2026-08-01\n---\n\n# 러프 메모\n\n[[hub]] 를 가리킵니다. 승급파이프라인 이야기입니다.\n",
		"sources/2026-08-01-src.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\ncreated: 2026-08-01\n---\n\n# 원본 요약\n\n승급파이프라인 원본입니다.\n",
	}
}

// makeWiki는 임시 디렉토리에 시험용 위키를 만든다.
func makeWiki(t *testing.T, preset string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range wikiFiles(preset) {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// newTestServer는 위키 하나를 노출하는 핸들러를 만든다. 실제 포트를 열지
// 않고 핸들러만 시험한다.
func newTestServer(t *testing.T, root string, opts Options) http.Handler {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("설정을 읽을 수 없습니다: %v", err)
	}
	return New(root, cfg, opts).Handler()
}

// get은 핸들러에 GET 요청을 보내고 상태 코드와 본문을 반환한다.
func get(t *testing.T, h http.Handler, target string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(body)
}

// docPath는 문서 상대 경로에 해당하는 요청 주소를 만든다.
func docPath(rel string) string {
	u := &url.URL{Path: "/doc/" + rel}
	return u.String()
}

func TestServeShowsOnlyVettedKnowledge(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})

	t.Run("목록에 context 문서와 색인 문서가 나옵니다", func(t *testing.T) {
		code, body := get(t, h, "/")
		if code != http.StatusOK {
			t.Fatalf("상태 코드 %d, 본문: %s", code, body)
		}
		for _, want := range []string{"허브 문서", "이웃 문서", "위키 색인", "context/hub.md"} {
			if !strings.Contains(body, want) {
				t.Errorf("목록에 %q 가 없습니다", want)
			}
		}
	})

	t.Run("목록에 inbox 와 sources 와 archive 와 민감 문서가 없습니다", func(t *testing.T) {
		_, body := get(t, h, "/")
		for _, hidden := range []string{
			"inbox/2026-08-01-rough.md", "러프 메모",
			"sources/2026-08-01-src.md", "원본 요약",
			"archive/old.md", "보관 문서",
			"context/secret.md", "로컬 전용 문서",
			"context/limited.md", "제한 공개 문서",
		} {
			if strings.Contains(body, hidden) {
				t.Errorf("목록에 제외 대상 %q 가 있습니다", hidden)
			}
		}
	})

	t.Run("제외된 문서는 URL 로도 404 입니다", func(t *testing.T) {
		for _, rel := range []string{
			"inbox/2026-08-01-rough.md",
			"sources/2026-08-01-src.md",
			"archive/old.md",
			"context/secret.md",
			"context/limited.md",
			"engram.yaml",
			".engram/index.json",
		} {
			code, _ := get(t, h, docPath(rel))
			if code != http.StatusNotFound {
				t.Errorf("%s 의 상태 코드가 %d 입니다. 404 여야 합니다", rel, code)
			}
		}
	})

	t.Run("위로 올라가는 경로는 문서를 내지 않습니다", func(t *testing.T) {
		// 라우터가 경로를 정리해 다른 주소로 보내기도 한다. 어디로
		// 가든 파일 내용이 나오지 않는 것이 계약이다.
		for _, target := range []string{"/doc/../engram.yaml", "/doc/context/../../engram.yaml"} {
			code, body := get(t, h, target)
			if code == http.StatusOK {
				t.Errorf("%s 가 200 을 냈습니다", target)
			}
			if strings.Contains(body, "preset:") {
				t.Errorf("%s 가 설정 파일 내용을 냈습니다", target)
			}
		}
	})

	t.Run("노출 문서는 URL 로 열립니다", func(t *testing.T) {
		code, body := get(t, h, docPath("context/hub.md"))
		if code != http.StatusOK {
			t.Fatalf("상태 코드 %d", code)
		}
		if !strings.Contains(body, "허브 문서") {
			t.Error("문서 제목이 없습니다")
		}
	})
}

func TestServeArchiveFlag(t *testing.T) {
	root := makeWiki(t, "team")

	t.Run("기본은 archive 를 감춥니다", func(t *testing.T) {
		h := newTestServer(t, root, Options{})
		code, _ := get(t, h, docPath("archive/old.md"))
		if code != http.StatusNotFound {
			t.Errorf("상태 코드 %d", code)
		}
	})

	t.Run("--include-archive 는 archive 를 엽니다", func(t *testing.T) {
		h := newTestServer(t, root, Options{IncludeArchive: true})
		code, body := get(t, h, docPath("archive/old.md"))
		if code != http.StatusOK {
			t.Fatalf("상태 코드 %d", code)
		}
		if !strings.Contains(body, "보관 문서") {
			t.Error("문서 제목이 없습니다")
		}
		_, list := get(t, h, "/")
		if !strings.Contains(list, "archive/old.md") {
			t.Error("목록에 archive 문서가 없습니다")
		}
	})
}

func TestServeSensitivityFilter(t *testing.T) {
	t.Run("축이 켜진 위키는 두 값을 뺍니다", func(t *testing.T) {
		h := newTestServer(t, makeWiki(t, "team"), Options{})
		for _, rel := range []string{"context/secret.md", "context/limited.md"} {
			code, _ := get(t, h, docPath(rel))
			if code != http.StatusNotFound {
				t.Errorf("%s 의 상태 코드가 %d 입니다", rel, code)
			}
		}
	})

	t.Run("축이 꺼진 위키는 그 필터가 걸리지 않습니다", func(t *testing.T) {
		root := makeWiki(t, "personal")
		cfg, err := config.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Axes[config.AxisSensitivity] {
			t.Fatal("personal 프리셋에서 sensitivity 속성이 켜져 있습니다. 시험 전제가 깨졌습니다")
		}
		h := New(root, cfg, Options{}).Handler()
		for _, rel := range []string{"context/secret.md", "context/limited.md"} {
			code, _ := get(t, h, docPath(rel))
			if code != http.StatusOK {
				t.Errorf("%s 의 상태 코드가 %d 입니다. 200 이어야 합니다", rel, code)
			}
		}
		_, list := get(t, h, "/")
		if !strings.Contains(list, "로컬 전용 문서") {
			t.Error("축이 꺼졌는데 목록에서 빠졌습니다")
		}
	})
}

// TestServeShowsInternalAndHidesUnvetted는 ADR 0063이 serve 에 준
// 기본값을 지킨다. internal 은 로컬 조회이므로 보이고, indexable 이
// false 이거나 status 가 superseded 인 문서는 보이지 않는다.
func TestServeShowsInternalAndHidesUnvetted(t *testing.T) {
	root := makeWiki(t, "team")
	extra := map[string]string{
		"context/색인제외.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: false\nsensitivity: public-reference\ncreated: 2026-01-01\n---\n\n# 색인 제외\n\n감춥니다.\n",
		"context/대체됨.md": "---\ntype: concept\nartifact_stage: context\nstatus: superseded\n" +
			"indexable: true\nsensitivity: public-reference\ncreated: 2026-01-01\n---\n\n# 대체된 문서\n\n감춥니다.\n",
	}
	for name, content := range extra {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := newTestServer(t, root, Options{})

	// hub 는 sensitivity 가 internal 이다. 자기 위키를 자기가 보는 것이므로
	// 감추지 않는다. 좁히는 플래그도 두지 않았다.
	if code, _ := get(t, h, docPath("context/hub.md")); code != http.StatusOK {
		t.Errorf("internal 문서의 상태 코드가 %d 입니다. 200 이어야 합니다", code)
	}
	for _, rel := range []string{"context/색인제외.md", "context/대체됨.md"} {
		if code, _ := get(t, h, docPath(rel)); code != http.StatusNotFound {
			t.Errorf("%s 의 상태 코드가 %d 입니다. 404 여야 합니다", rel, code)
		}
	}
	_, list := get(t, h, "/")
	if strings.Contains(list, "대체된 문서") || strings.Contains(list, "색인 제외") {
		t.Error("감춰야 할 문서가 목록에 있습니다")
	}
}

func TestServeWikiLinks(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	_, body := get(t, h, docPath("context/hub.md"))

	t.Run("노출 문서를 가리키는 링크는 링크가 됩니다", func(t *testing.T) {
		if !strings.Contains(body, `<a href="/doc/context/peer.md">peer</a>`) {
			t.Errorf("본문의 peer 링크가 없습니다:\n%s", body)
		}
	})

	t.Run("제외된 문서를 가리키는 링크는 링크가 되지 않습니다", func(t *testing.T) {
		for _, rel := range []string{"context/secret.md", "context/limited.md", "inbox/2026-08-01-rough.md"} {
			if strings.Contains(body, `<a href="`+docPath(rel)) {
				t.Errorf("제외된 %s 로 가는 링크가 있습니다", rel)
			}
		}
		for _, raw := range []string{"[[secret]]", "[[limited]]", "[[2026-08-01-rough]]"} {
			if !strings.Contains(body, raw) {
				t.Errorf("제외된 링크 %s 가 글자로 남지 않았습니다", raw)
			}
		}
	})

	t.Run("코드 안의 위키링크는 링크가 되지 않습니다", func(t *testing.T) {
		if !strings.Contains(body, "<code>[[peer]]</code>") {
			t.Errorf("코드 스팬이 링크로 바뀌었습니다:\n%s", body)
		}
	})
}

func TestServeEscapesRawHTML(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	_, body := get(t, h, docPath("context/hub.md"))
	if strings.Contains(body, "<script>") {
		t.Errorf("문서에 담긴 <script> 가 그대로 나갔습니다:\n%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("원시 HTML 이 이스케이프되지 않았습니다:\n%s", body)
	}
	if strings.Contains(body, "<img src=x") {
		t.Errorf("인라인 원시 HTML 이 그대로 나갔습니다:\n%s", body)
	}
	if !strings.Contains(body, "&lt;img src=x") {
		t.Errorf("인라인 원시 HTML 이 이스케이프되지 않았습니다:\n%s", body)
	}
	if strings.Count(body, "<h1") != 1 {
		t.Errorf("화면의 h1 이 하나가 아닙니다:\n%s", body)
	}
}

func TestServeSearch(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	code, body := get(t, h, "/search?q="+url.QueryEscape("승급파이프라인"))
	if code != http.StatusOK {
		t.Fatalf("상태 코드 %d", code)
	}
	if !strings.Contains(body, "context/hub.md") {
		t.Errorf("검색 결과에 노출 문서가 없습니다:\n%s", body)
	}
	for _, hidden := range []string{"inbox/2026-08-01-rough.md", "sources/2026-08-01-src.md"} {
		if strings.Contains(body, hidden) {
			t.Errorf("검색 결과에 제외 대상 %q 가 있습니다", hidden)
		}
	}

	// 제외된 문서에만 있는 낱말은 결과가 없어야 한다.
	_, body = get(t, h, "/search?q="+url.QueryEscape("살구"))
	if strings.Contains(body, "context/secret.md") {
		t.Error("민감 문서가 검색으로 샜습니다")
	}
}

func TestServeBacklinks(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	_, body := get(t, h, docPath("context/hub.md"))
	if !strings.Contains(body, "백링크") {
		t.Fatal("백링크 구획이 없습니다")
	}
	if !strings.Contains(body, `<a href="/doc/context/peer.md">이웃 문서`) {
		t.Errorf("노출 문서에서 오는 백링크가 없습니다:\n%s", body)
	}
	for _, hidden := range []string{"로컬 전용 문서", "제한 공개 문서", "러프 메모", "보관 문서"} {
		if strings.Contains(body, hidden) {
			t.Errorf("제외된 문서에서 오는 백링크 %q 가 있습니다", hidden)
		}
	}
}

func TestServeStatusPage(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	code, body := get(t, h, "/status")
	if code != http.StatusOK {
		t.Fatalf("상태 코드 %d, 본문: %s", code, body)
	}
	if !strings.Contains(body, "위키 현황") || !strings.Contains(body, "단계별 문서") {
		t.Errorf("현황 화면이 아닙니다:\n%s", body)
	}
	// 현황은 수치만 낸다. 제외 대상 문서의 경로가 화면에 있으면 안 된다.
	for _, leak := range []string{"inbox/2026-08-01-rough.md", "engram promote"} {
		if strings.Contains(body, leak) {
			t.Errorf("현황 화면에 %q 가 샜습니다", leak)
		}
	}
}

func TestServeRejectsWrites(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	targets := []string{"/", "/doc/context/hub.md", "/search", "/status", "/resurface", "/무엇이든"}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, target := range targets {
		for _, method := range methods {
			req := httptest.NewRequest(method, target, strings.NewReader("body=x"))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			code := rec.Result().StatusCode
			if code != http.StatusMethodNotAllowed && code != http.StatusNotFound {
				t.Errorf("%s %s 의 상태 코드가 %d 입니다. 405 나 404 여야 합니다", method, target, code)
			}
		}
	}
}

func TestServeDoesNotTouchWiki(t *testing.T) {
	root := makeWiki(t, "team")
	before := hashTree(t, root)
	h := newTestServer(t, root, Options{IncludeArchive: true})
	for _, target := range []string{
		// /resurface 가 여기 있는 것이 읽기 전용 계약의 시험이다. resurface 는
		// 원래 제시 이력을 쓰고 bridge 는 벡터 캐시를 쓴다. 화면이 그 둘을
		// 쓰지 않는지는 위키 전체 해시로만 확인된다(ADR 0076).
		"/", "/status", "/resurface", "/resurface",
		"/search?q=" + url.QueryEscape("승급파이프라인"), "/search",
		docPath("context/hub.md"), docPath("context/peer.md"), docPath("index.md"),
		docPath("archive/old.md"), docPath("inbox/2026-08-01-rough.md"),
		"/assets/engram.css", "/없는주소",
	} {
		get(t, h, target)
	}
	after := hashTree(t, root)
	if before != after {
		t.Errorf("요청을 처리하는 동안 위키 파일이 바뀌었습니다\n이전:\n%s\n이후:\n%s", before, after)
	}
}

// hashTree는 디렉토리 아래 모든 파일의 경로와 내용 해시를 문자열로 만든다.
func hashTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s %x", filepath.ToSlash(rel), sha256.Sum256(data)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestServeResurfaceHidesExcluded(t *testing.T) {
	// team 프리셋은 sensitivity 축을 켠다. private-local-only 와 restricted
	// 문서는 재발견 후보에도 오르지 못해야 한다. resurface 와 bridge 는
	// 위키 전체를 보므로 화면에서 다시 거르지 않으면 제목과 경로가 샌다.
	h := newTestServer(t, makeWiki(t, "team"), Options{})
	code, body := get(t, h, "/resurface")
	if code != http.StatusOK {
		t.Fatalf("상태 코드가 %d 입니다. 200 이어야 합니다", code)
	}
	for _, leak := range []string{
		"로컬 전용 문서", "제한 공개 문서", "보관 문서", "러프 메모", "원본 요약",
		"context/secret.md", "context/limited.md", "archive/old.md",
		"inbox/2026-08-01-rough.md", "sources/2026-08-01-src.md",
	} {
		if strings.Contains(body, leak) {
			t.Errorf("재발견 화면에 제외 대상 %q 가 나옵니다", leak)
		}
	}
	// 노출 문서는 나와야 한다. 전부 감추면 이 시험이 무의미해진다.
	if !strings.Contains(body, "재발견") {
		t.Error("재발견 화면의 제목이 없습니다")
	}
}

func TestServeResurfaceWithoutVectorsUsesWordAxisOnly(t *testing.T) {
	// 벡터 캐시가 없는 위키다. 의미 축이 없다는 사실을 화면이 밝혀야
	// 한다. 조용히 단어 축만 쓰면 읽는 사람이 의미 축까지 돌았다고
	// 믿는다(ADR 0076).
	h := newTestServer(t, makeWiki(t, "personal"), Options{})
	_, body := get(t, h, "/resurface")
	if !strings.Contains(body, "낱말 축만") {
		t.Error("의미 축을 쓰지 못했다는 안내가 없습니다")
	}
}

func TestServeAssetsAndHeaders(t *testing.T) {
	h := newTestServer(t, makeWiki(t, "team"), Options{})

	t.Run("스타일시트를 바이너리에서 냅니다", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/assets/engram.css", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("상태 코드 %d", res.StatusCode)
		}
		if got := res.Header.Get("Content-Type"); got != "text/css; charset=utf-8" {
			t.Errorf("Content-Type 이 %q 입니다", got)
		}
	})

	t.Run("바깥 자산을 부르지 않습니다", func(t *testing.T) {
		_, body := get(t, h, "/")
		for _, bad := range []string{"http://", "https://", "<script"} {
			if strings.Contains(body, bad) {
				t.Errorf("화면에 %q 가 있습니다. CDN 과 스크립트를 쓰지 않습니다", bad)
			}
		}
	})

	t.Run("응답에 보안 헤더가 붙습니다", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Result().Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") {
			t.Errorf("콘텐츠 보안 정책이 %q 입니다", got)
		}
	})
}

func TestServeExposureCounts(t *testing.T) {
	root := makeWiki(t, "team")
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(root, cfg, Options{}).Exposure()
	if err != nil {
		t.Fatal(err)
	}
	if e.Context != 2 || e.Index != 1 || e.Archive != 0 {
		t.Errorf("노출 집계가 다릅니다: %+v", e)
	}
	if e.ExcludedInbox != 1 || e.ExcludedSources != 1 || e.ExcludedArchive != 1 || e.ExcludedSensitive != 2 {
		t.Errorf("제외 집계가 다릅니다: %+v", e)
	}
	if !e.SensitivityOn || e.IncludeArchive {
		t.Errorf("축과 플래그 상태가 다릅니다: %+v", e)
	}
	if e.Visible() != 3 {
		t.Errorf("노출 합계가 %d 입니다", e.Visible())
	}
}
