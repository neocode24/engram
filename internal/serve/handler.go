package serve

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/status"
)

// 정적 자산은 바이너리에 임베드한다. 실행 시 파일도 네트워크도 필요하지
// 않아야 폐쇄망에서 쓸 수 있다(ADR 0044).
//
//go:embed templates/*.html
var templateFS embed.FS

//go:embed assets/engram.css
var styleSheet []byte

// searchLimit는 검색 결과 화면의 상한이다.
const searchLimit = 20

// 내비게이션 현재 위치 값이다.
const (
	secIndex  = "index"
	secSearch = "search"
	secStatus = "status"
	secDoc    = "doc"
)

// templates는 임베드한 템플릿을 파싱한다. 파싱 실패는 빌드에 실은 자산의
// 문제이므로 프로그램 오류로 다룬다.
func templates() *template.Template {
	return template.Must(template.New("serve").ParseFS(templateFS, "templates/*.html"))
}

// chrome은 모든 화면이 공유하는 껍데기 자료다.
type chrome struct {
	Title    string
	Section  string
	Query    string
	Exposure Exposure
}

// group은 문서 목록의 묶음 하나다.
type group struct {
	ID    string
	Label string
	Docs  []docView
}

// indexData는 문서 목록 화면의 자료다.
type indexData struct {
	Chrome chrome
	Groups []group
	Notes  []string
}

// backlink은 문서 화면에 내는 백링크 한 줄이다.
type backlink struct {
	Title string
	URL   string
	Field string
}

// docData는 문서 화면의 자료다.
type docData struct {
	Chrome    chrome
	Doc       docView
	Body      template.HTML
	Backlinks []backlink
}

// searchHit는 검색 결과 한 줄이다.
type searchHit struct {
	Title string
	URL   string
	Rel   string
}

// searchData는 검색 화면의 자료다.
type searchData struct {
	Chrome   chrome
	Searched bool
	Hits     []searchHit
	Note     string
}

// statusData는 위키 현황 화면의 자료다. status.Result 에는 승급 대기
// 문서의 경로와 그 경로를 담은 제안 문장이 있으나 그 경로는 inbox 문서를
// 가리키므로 여기로 옮기지 않는다. 수치만 낸다(ADR 0044).
type statusData struct {
	Chrome     chrome
	Inbox      int
	Sources    int
	Context    int
	Archive    int
	Links      int
	Orphans    int
	LintFiles  int
	LintError  int
	LintWarn   int
	LintReject int
	Stale      int
	Promotable int
}

// Handler는 서버의 HTTP 핸들러를 만든다. 실제 포트를 여는 것은 호출자의
// 몫이라 이 함수만으로 시험할 수 있다.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", s.handleIndex)
	mux.HandleFunc("/doc/", s.handleDoc)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/assets/engram.css", s.handleStyle)
	mux.HandleFunc("/", s.handleNotFound)
	return readOnly(withHeaders(mux))
}

// readOnly는 GET 과 HEAD 외의 메서드를 전부 막는다. 이 서버에는 쓰기
// 경로가 없으므로 라우팅에 닿기 전에 끝낸다. 핸들러마다 메서드를 따로
// 판단하면 언젠가 하나가 빠진다.
func readOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, i18n.T("core.serve.readonly")+"\n", http.StatusMethodNotAllowed)
		}
	})
}

// withHeaders는 응답 헤더를 붙인다. 콘텐츠 보안 정책은 스크립트를 막고
// 자산을 자기 자신으로 제한한다. 이 서버는 JavaScript 를 쓰지 않고
// CDN 도 쓰지 않으므로 정책이 화면을 깨뜨리지 않는다. 위키 문서가 부르는
// 바깥 이미지도 함께 막혀 폐쇄망에서 밖으로 나가는 요청이 생기지 않는다.
func withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'none'; style-src 'self'; img-src 'self' data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// handleIndex는 문서 목록을 낸다.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	v, err := s.load()
	if err != nil {
		s.fail(w, err)
		return
	}
	data := indexData{
		Chrome: chrome{Title: i18n.T("core.serve.title_docs"), Section: secIndex, Exposure: v.exposure},
		Groups: []group{
			{ID: locContext, Label: i18n.T("core.serve.group_context"), Docs: v.visibleDocs(locContext)},
			{ID: locIndex, Label: i18n.T("core.serve.group_index"), Docs: v.visibleDocs(locIndex)},
		},
	}
	if s.opts.IncludeArchive {
		data.Groups = append(data.Groups, group{
			ID: locArchive, Label: i18n.T("core.serve.group_archive"), Docs: v.visibleDocs(locArchive),
		})
	}
	data.Notes = notesOf(v.exposure)
	s.render(w, http.StatusOK, "index.html", data)
}

// notesOf는 목록 화면 아래에 붙일 안내를 만든다. 감춘 것이 있으면 감췄다고
// 알린다. 조용히 빼면 읽는 사람이 위키 전체를 봤다고 착각한다.
func notesOf(e Exposure) []string {
	var notes []string
	if !e.IncludeArchive && e.ExcludedArchive > 0 {
		notes = append(notes, i18n.T("core.serve.note_archive_hidden", e.ExcludedArchive))
	}
	if e.SensitivityOn && e.ExcludedSensitive > 0 {
		notes = append(notes, i18n.T("core.serve.note_sensitive_hidden", e.ExcludedSensitive))
	}
	if e.ExcludedUnparsed > 0 {
		notes = append(notes, i18n.T("core.serve.note_unparsed", e.ExcludedUnparsed))
	}
	return notes
}

// handleDoc은 문서 하나를 렌더한다. 제외된 문서와 없는 문서는 같은 404 를
// 낸다. 응답이 다르면 그 차이가 곧 문서의 존재를 알리는 통로가 된다.
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	v, err := s.load()
	if err != nil {
		s.fail(w, err)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/doc/")
	i, ok := v.byPath[rel]
	if !ok {
		s.notFound(w, v)
		return
	}
	d := v.docs[i]
	body, err := renderMarkdown(d.body, v.resolve)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "doc.html", docData{
		Chrome:    chrome{Title: d.Title, Section: secDoc, Exposure: v.exposure},
		Doc:       d,
		Body:      body,
		Backlinks: backlinksOf(v, d),
	})
}

// backlinksOf는 노출 문서에서 오는 백링크만 모은다. 제외된 문서에서 오는
// 링크를 내면 감춘 문서의 경로가 그대로 드러난다. 같은 문서의 같은 종류
// 링크는 한 줄로 묶는다.
func backlinksOf(v *view, d docView) []backlink {
	seen := map[string]bool{}
	var out []backlink
	for _, l := range v.graph.Backlinks(d.Slug) {
		if l.From == d.Rel {
			continue
		}
		i, ok := v.byPath[l.From]
		if !ok {
			continue
		}
		key := l.From + "\x00" + string(l.Field)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, backlink{
			Title: v.docs[i].Title,
			URL:   v.docs[i].URL,
			Field: l.Field.Label(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].Field < out[j].Field
	})
	return out
}

// handleSearch는 검색 폼과 결과를 낸다. JavaScript 없이 폼 제출로 돈다.
// 검색은 internal/index 를 그대로 부르고 결과에서 제외 문서를 걷어낸다.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	v, err := s.load()
	if err != nil {
		s.fail(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data := searchData{Chrome: chrome{Title: i18n.T("core.serve.title_search"), Section: secSearch, Query: q, Exposure: v.exposure}}
	if q == "" {
		s.render(w, http.StatusOK, "search.html", data)
		return
	}
	data.Searched = true

	// 조회는 색인 파일을 쓰지 않는다. 색인이 없으면 이번 요청에서만
	// 문서를 읽는다(ADR 0025 와 search 커맨드의 계약을 그대로 따른다).
	ix := index.Load(s.root)
	switch {
	case ix == nil:
		ix, err = index.Build(s.root, v.walked, index.DefaultWeights())
		if err != nil {
			s.fail(w, err)
			return
		}
		data.Note = i18n.T("core.serve.note_no_index")
	case !ix.Fresh(v.walked, s.root):
		data.Note = i18n.T("core.serve.note_stale_index")
	}

	// 상한을 검색 단계에서 걸지 않는다. 제외된 문서가 상위를 차지하면
	// 노출 문서가 상한 밖으로 밀려나기 때문이다.
	for _, hit := range ix.Search(q, 0) {
		i, ok := v.byPath[hit.Path]
		if !ok {
			continue
		}
		data.Hits = append(data.Hits, searchHit{
			Title: v.docs[i].Title, URL: v.docs[i].URL, Rel: v.docs[i].Rel,
		})
		if len(data.Hits) == searchLimit {
			break
		}
	}
	s.render(w, http.StatusOK, "search.html", data)
}

// handleStatus는 위키 현황을 낸다. 재발견 커맨드는 노출하지 않는다.
// 사람의 판단 기록을 건드리기 때문이다(ADR 0028, 0044).
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	v, err := s.load()
	if err != nil {
		s.fail(w, err)
		return
	}
	res, err := status.Run(s.root, time.Now())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, http.StatusOK, "status.html", statusData{
		Chrome:     chrome{Title: i18n.T("core.serve.title_status"), Section: secStatus, Exposure: v.exposure},
		Inbox:      res.Stages.Inbox,
		Sources:    res.Stages.Source,
		Context:    res.Stages.Context,
		Archive:    res.Stages.Archive,
		Links:      res.Links,
		Orphans:    res.Orphans,
		LintFiles:  res.Lint.Files,
		LintError:  res.Lint.Error,
		LintWarn:   res.Lint.Warn,
		LintReject: res.Lint.Reject,
		Stale:      res.Backlog.Stale,
		Promotable: res.Backlog.Promotable,
	})
}

// handleStyle은 임베드한 스타일시트를 낸다.
func (s *Server) handleStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(styleSheet)
}

// handleNotFound는 등록되지 않은 주소를 받는다.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	v, err := s.load()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.notFound(w, v)
}

// notFound는 404 화면을 낸다.
func (s *Server) notFound(w http.ResponseWriter, v *view) {
	s.render(w, http.StatusNotFound, "notfound.html", indexData{
		Chrome: chrome{Title: i18n.T("core.serve.title_notfound"), Section: secIndex, Exposure: v.exposure},
	})
}

// fail은 서버 쪽 오류를 알린다. 화면에는 위키 경로 같은 내부 사정을 싣지
// 않고 사유는 서버를 띄운 사람에게만 적는다.
func (s *Server) fail(w http.ResponseWriter, err error) {
	if s.opts.ErrorLog != nil {
		fmt.Fprintln(s.opts.ErrorLog, i18n.T("core.serve.fail_log", err))
	}
	http.Error(w, i18n.T("core.serve.fail_read")+"\n",
		http.StatusInternalServerError)
}

// render는 화면을 먼저 메모리에 만든 뒤 내보낸다. 템플릿이 중간에
// 실패해도 반쪽짜리 화면이 200 으로 나가지 않게 하기 위해서다.
func (s *Server) render(w http.ResponseWriter, code int, name string, data any) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, i18n.T("core.serve.render_fail")+"\n", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(buf.Bytes())
}
