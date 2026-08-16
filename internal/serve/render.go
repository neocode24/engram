package serve

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// wikiLinkPriority는 위키링크 파서의 우선순위다. 기본 링크 파서가 200
// 이므로 그보다 앞에 둬야 [[슬러그]] 를 먼저 본다.
const wikiLinkPriority = 190

// escapePriority는 원시 HTML 렌더러의 우선순위다. 기본 HTML 렌더러가
// 1000 이고 작은 값이 이기므로 이 렌더러가 원시 HTML 을 맡는다.
const escapePriority = 100

// renderMarkdown은 문서 본문을 HTML 로 만든다. resolve 는 위키링크를
// 노출 문서의 주소로 바꾸는 함수이며 실패하면 그 링크는 글자로 남는다.
//
// 반환값을 template.HTML 로 쓰는 근거는 둘이다. goldmark 의 기본 설정은
// 원시 HTML 을 내보내지 않고(html.WithUnsafe 를 켜지 않았다) 위험한 URL
// 스킴을 링크 대상에서 뺀다. 그 위에 원시 HTML 을 글자로 이스케이프하는
// 렌더러를 얹었다. 문서에 담긴 <script> 는 실행되지 않고 보인다.
func renderMarkdown(body string, resolve func(string) (string, bool)) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithInlineParsers(util.Prioritized(&wikiLinkParser{resolve: resolve}, wikiLinkPriority)),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(escapingRenderer{}, escapePriority)),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil //nolint:gosec // 위 주석의 근거로 안전하다
}

// wikiLinkParser는 [[슬러그]] 를 인라인 요소로 파싱한다. 코드 스팬과 코드
// 블록 안에서는 인라인 파서가 돌지 않으므로 링크 문법을 설명하는 문서가
// 링크로 바뀌지 않는다.
type wikiLinkParser struct {
	resolve func(slug string) (string, bool)
}

// Trigger는 여는 대괄호를 만나면 이 파서를 부르게 한다.
func (p *wikiLinkParser) Trigger() []byte {
	return []byte{'['}
}

// Parse는 [[슬러그]], [[슬러그|표시]], [[슬러그#헤딩]] 을 읽는다.
// 가리키는 문서가 노출 대상이면 링크로, 아니면 원문 그대로의 글자로
// 만든다. 제외된 문서로 가는 길을 남기지 않는 것이 이 함수의 계약이다.
func (p *wikiLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 5 || line[0] != '[' || line[1] != '[' {
		return nil
	}
	end := bytes.Index(line, []byte("]]"))
	if end < 3 {
		return nil
	}
	inner := line[2:end]
	if bytes.ContainsAny(inner, "[]") {
		return nil
	}
	raw := string(line[:end+2])
	block.Advance(end + 2)

	slug, display := splitWikiLink(string(inner))
	if slug == "" {
		// [[#헤딩]] 처럼 슬러그가 비면 같은 문서 안 참조다. 링크로 치지 않는다.
		return ast.NewString([]byte(raw))
	}
	dest, ok := p.resolve(slug)
	if !ok {
		return ast.NewString([]byte(raw))
	}
	link := ast.NewLink()
	link.Destination = []byte(dest)
	link.AppendChild(link, ast.NewString([]byte(display)))
	return link
}

// splitWikiLink는 링크 내용에서 가리킴 슬러그와 표시 문자를 가른다.
// 슬러그 추출 규칙은 internal/doc 의 링크 추출과 같다. 표시문자와 헤딩을
// 버리고 슬러그만 남긴다.
func splitWikiLink(inner string) (slug, display string) {
	display = strings.TrimSpace(inner)
	slug = inner
	if i := strings.Index(inner, "|"); i >= 0 {
		slug = inner[:i]
		display = strings.TrimSpace(inner[i+1:])
	}
	if i := strings.Index(slug, "#"); i >= 0 {
		slug = slug[:i]
	}
	slug = strings.TrimSpace(slug)
	if display == "" {
		display = slug
	}
	return slug, display
}

// escapingRenderer는 문서에 담긴 원시 HTML 을 글자로 이스케이프해 낸다.
// goldmark 의 기본은 원시 HTML 을 지우고 주석 한 줄로 바꾸는 것이라
// 안전하기는 하나 무엇이 적혀 있었는지가 사라진다. 위키 문서는 HTML
// 조각을 설명으로 담기도 하므로 지우는 대신 보이게 한다. 어느 쪽이든
// 브라우저가 실행하지 않는다.
type escapingRenderer struct{}

// RegisterFuncs는 원시 HTML 두 종류와 헤딩의 렌더링을 가져간다.
func (escapingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, renderRawHTMLEscaped)
	reg.Register(ast.KindHTMLBlock, renderHTMLBlockEscaped)
	reg.Register(ast.KindHeading, renderHeadingShifted)
}

// renderHeadingShifted는 본문 헤딩을 한 단계 낮춰 낸다. 문서 제목이 화면의
// h1 이므로 본문의 # 헤딩이 그대로 h1 이 되면 화면에 h1 이 둘이 된다.
// 보조 기술이 문서 구조를 읽는 순서가 흐트러지지 않게 한 단계씩 민다.
func renderHeadingShifted(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	level := n.Level + 1
	if level > 6 {
		level = 6
	}
	if entering {
		_, _ = w.WriteString("<h" + strconv.Itoa(level) + ">")
	} else {
		_, _ = w.WriteString("</h" + strconv.Itoa(level) + ">\n")
	}
	return ast.WalkContinue, nil
}

// renderRawHTMLEscaped는 본문 중간의 원시 HTML 조각을 이스케이프한다.
func renderRawHTMLEscaped(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		_, _ = w.Write(util.EscapeHTML(seg.Value(source)))
	}
	return ast.WalkSkipChildren, nil
}

// renderHTMLBlockEscaped는 HTML 블록을 코드 블록으로 이스케이프한다.
func renderHTMLBlockEscaped(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if entering {
		_, _ = w.WriteString("<pre><code>")
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			_, _ = w.Write(util.EscapeHTML(line.Value(source)))
		}
		return ast.WalkContinue, nil
	}
	if n.HasClosure() {
		_, _ = w.Write(util.EscapeHTML(n.ClosureLine.Value(source)))
	}
	_, _ = w.WriteString("</code></pre>\n")
	return ast.WalkContinue, nil
}
