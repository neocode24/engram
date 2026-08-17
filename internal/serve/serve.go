// Package serve는 위키를 읽기 전용 웹 뷰어로 노출한다.
//
// 노출 범위는 ADR 0044가 정한다. context 와 root_files 만 보이고 inbox 와
// sources 는 목록에도 URL 에도 없다. archive 는 기본으로 감추고 플래그로
// 연다. sensitivity 축이 켜진 위키에서는 private-local-only 와 restricted 를
// 뺀다. 그 제외를 뒤집는 플래그는 두지 않는다.
//
// 쓰기 경로가 없다. POST 계열은 라우팅에 닿기 전에 막힌다. 판정과 검색은
// internal 패키지를 그대로 부르므로 여기서 규칙을 다시 만들지 않는다.
package serve

import (
	"html/template"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/expose"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/walk"
)

// Options는 노출 범위를 넓히는 선택지다. 넓히는 쪽만 있고 좁히는 쪽은
// 기본값이다. 기본이 전체 노출이면 사고가 나기 때문이다.
type Options struct {
	// IncludeArchive는 archive 문서를 노출할지 정한다. 기본은 거짓이다.
	IncludeArchive bool
	// ErrorLog는 요청 처리 중 난 오류를 적을 곳이다. 화면에는 사정을
	// 싣지 않으므로 서버를 띄운 사람이 볼 자리가 필요하다. nil 이면 버린다.
	ErrorLog io.Writer
}

// Server는 위키 하나를 노출하는 웹 뷰어다. 상태를 갖지 않으며 요청마다
// 위키를 다시 읽는다. 서버가 뜬 뒤에 CLI 가 문서를 고쳐도 화면이 옛
// 내용을 보이지 않게 하기 위해서다.
type Server struct {
	root string
	cfg  config.Config
	opts Options
	tmpl *template.Template
}

// New는 위키 경로와 설정으로 서버를 만든다. 설정은 호출자가 읽어서 준다.
// 위키가 아닌 디렉토리 판정은 커맨드 계층의 몫이다.
func New(root string, cfg config.Config, opts Options) *Server {
	return &Server{root: root, cfg: cfg, opts: opts, tmpl: templates()}
}

// 문서의 위치 구분이다. 판정은 internal/expose 가 하며 여기서는 화면
// 묶음의 이름으로 쓴다.
const (
	locIndex   = expose.LocIndex
	locContext = expose.LocContext
	locArchive = expose.LocArchive
)

// Exposure는 무엇이 노출되고 무엇이 제외되는지의 집계다. 판정과 함께
// internal/expose 에 있으며 serve 는 그것을 시작 안내에 쓴다.
type Exposure = expose.Exposure

// docView는 노출 문서 하나다. 화면에 필요한 것만 담는다.
type docView struct {
	Rel    string  // 위키 루트 상대 슬래시 경로
	Slug   string  // 링크 해석용 정규화 슬러그
	Title  string  // 표시 제목
	Loc    string  // 위치 구분
	URL    string  // 문서 페이지 주소
	Fields []field // 표시할 프론트매터 요약
	body   string  // 마크다운 본문
}

// field는 화면에 내는 프론트매터 한 줄이다.
type field struct {
	Key   string
	Value string
}

// view는 한 요청이 보는 위키의 모습이다. 노출 문서 목록과 링크 그래프와
// 제외 집계를 함께 담는다.
type view struct {
	docs     []docView
	byPath   map[string]int // 상대 경로에서 docs 색인으로
	bySlug   map[string]int // 정규화 슬러그에서 docs 색인으로
	graph    *graph.Graph   // 위키 전체의 링크 그래프
	walked   []walk.Doc     // 순회 결과 원본. 색인 신선도 판정에 쓴다
	exposure Exposure
}

// load는 위키를 순회해 이번 요청이 볼 모습을 만든다. 제외된 문서는
// docs 에 남지 않으므로 목록에도 URL 로도 닿지 않는다.
func (s *Server) load() (*view, error) {
	sel, err := expose.Select(s.root, s.cfg, expose.Options{IncludeArchive: s.opts.IncludeArchive})
	if err != nil {
		return nil, err
	}
	v := &view{
		byPath: map[string]int{},
		bySlug: map[string]int{},
		// 그래프는 위키 전체로 만든다. 제외된 문서가 거는 링크를 세지
		// 않으려면 결과를 거르면 되고, 그래프 자체를 좁히면 mv 나
		// backlinks 와 다른 계산이 된다.
		graph:    graph.Build(sel.Walked),
		walked:   sel.Walked,
		exposure: sel.Exposure,
	}
	for _, ed := range sel.Docs {
		d := docView{
			Rel:    ed.Rel,
			Slug:   graph.Normalize(ed.Rel),
			Title:  titleOf(ed.Doc),
			Loc:    ed.Loc,
			URL:    docURL(ed.Rel),
			Fields: fieldsOf(ed.Parsed),
			body:   ed.Parsed.Body,
		}
		v.byPath[d.Rel] = len(v.docs)
		// 같은 슬러그가 둘이면 먼저 나온 문서가 링크 대상이 된다.
		// 순회가 경로순이므로 이 선택은 결정론적이다.
		if _, dup := v.bySlug[d.Slug]; !dup {
			v.bySlug[d.Slug] = len(v.docs)
		}
		v.docs = append(v.docs, d)
	}
	return v, nil
}

// Exposure는 지금 위키를 순회해 노출과 제외를 집계한다. 서버를 띄우기
// 전에 무엇이 보이는지 알리는 데 쓴다.
func (s *Server) Exposure() (Exposure, error) {
	v, err := s.load()
	if err != nil {
		return Exposure{}, err
	}
	return v.exposure, nil
}

// titleOf는 표시 제목을 정한다. 본문 첫 헤딩을 우선하고 없으면 슬러그에서
// 만든다. internal/index 의 색인 제목과 같은 규칙이라 검색 결과와 목록의
// 제목이 어긋나지 않는다.
func titleOf(wd walk.Doc) string {
	for _, line := range strings.Split(wd.Parsed.Body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(t, "#"))
			if heading != "" {
				return heading
			}
		}
	}
	slug := strings.TrimSuffix(filepath.Base(wd.Rel), ".md")
	return strings.ReplaceAll(slug, "-", " ")
}

// metaKeys는 문서 화면에 내는 프론트매터 키다. 축 전부를 늘어놓지 않고
// 읽는 사람에게 쓸모 있는 것만 doc.StandardKeys 순서로 낸다.
var metaKeys = []string{
	"type", "status", "sensitivity", "topics", "tags", "created", "updated",
}

// fieldsOf는 표시할 프론트매터를 순서대로 만든다. 값이 없는 키는 낸다고
// 알릴 것이 없으므로 건너뛴다.
func fieldsOf(d doc.Doc) []field {
	present := map[string]doc.Field{}
	for _, f := range d.Fields {
		present[f.Key] = f
	}
	var out []field
	for _, k := range doc.StandardKeys() {
		if !allowedMetaKey(k) {
			continue
		}
		f, ok := present[k]
		if !ok {
			continue
		}
		if v := fieldValue(f); v != "" {
			out = append(out, field{Key: k, Value: v})
		}
	}
	return out
}

// allowedMetaKey는 화면에 낼 키인지 본다.
func allowedMetaKey(k string) bool {
	for _, m := range metaKeys {
		if m == k {
			return true
		}
	}
	return false
}

// fieldValue는 프론트매터 값 하나를 표시 문자열로 만든다.
func fieldValue(f doc.Field) string {
	switch f.Kind {
	case doc.KindStringList:
		return strings.Join(f.List, ", ")
	case doc.KindBool:
		if f.Bool {
			return "true"
		}
		return "false"
	case doc.KindEmpty:
		return ""
	}
	return f.Str
}

// docURL은 문서의 주소를 만든다. 한글 파일명과 공백이 있는 파일명을 위해
// 경로를 이스케이프한다.
func docURL(rel string) string {
	u := &url.URL{Path: "/doc/" + rel}
	return u.String()
}

// resolve는 위키링크 슬러그를 노출 문서의 주소로 바꾼다. 제외된 문서와
// 없는 문서는 둘 다 실패한다. 실패한 링크는 링크가 되지 않는다.
// 경로가 새면 제외가 무의미해지기 때문이다(ADR 0044).
func (v *view) resolve(slug string) (string, bool) {
	i, ok := v.bySlug[graph.Normalize(slug)]
	if !ok {
		return "", false
	}
	return v.docs[i].URL, true
}

// visibleDocs는 위치별로 묶은 노출 문서를 낸다. 순회가 경로순이므로
// 묶음 안의 순서도 경로순이다.
func (v *view) visibleDocs(loc string) []docView {
	var out []docView
	for _, d := range v.docs {
		if d.Loc == loc {
			out = append(out, d)
		}
	}
	return out
}
