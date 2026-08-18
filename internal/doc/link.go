package doc

import (
	"path"
	"regexp"
	"strings"
)

// Link는 추출된 링크 하나다. 중복 제거는 하지 않는다. 세는 것은 호출자 몫이다.
type Link struct {
	Slug string
	Line int // 원본 파일 기준 1 기반 줄 번호
}

// linkPattern은 두 문법을 한 번에 잡는다(ADR 0065). 첫 갈래가 위키링크
// [[슬러그]], [[슬러그|표시문자]], [[슬러그#헤딩]]이고 둘째 갈래가 마크다운
// 링크 [텍스트](경로)다. 하나로 묶어야 문서 안 등장 순서가 보존된다.
var linkPattern = regexp.MustCompile(`\[\[([^[\]]+)\]\]|\[[^\[\]]*\]\(([^()\s]+)\)`)

// schemePattern은 http:, https:, mailto: 같은 URI 스킴 접두를 잡는다.
var schemePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)

// BodyLinks는 본문에서 링크를 순서대로 추출한다.
// 코드 펜스(```)와 인라인 코드(백틱) 안의 것은 링크가 아니다.
// 문서에 링크 문법 자체를 설명해도 판정이 틀어지지 않게 하기 위해서다.
func (d Doc) BodyLinks() []Link {
	var links []Link
	inFence := false
	for i, line := range strings.Split(d.Body, "\n") {
		lineNo := d.BodyLine + i
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, seg := range outsideCode(line) {
			for _, m := range linkPattern.FindAllStringSubmatch(seg, -1) {
				slug := slugOf(m[1])
				if m[1] == "" {
					slug = slugOfMarkdownPath(m[2])
				}
				if slug != "" {
					links = append(links, Link{Slug: slug, Line: lineNo})
				}
			}
		}
	}
	return links
}

// slugOfMarkdownPath는 마크다운 링크 경로를 슬러그로 바꾼다.
// 위키 안 문서를 가리키지 않으면 빈 문자열을 낸다(ADR 0065).
// 디렉토리를 버리는 것은 위키 안에서 슬러그가 유일하다는 전제 때문이다.
func slugOfMarkdownPath(p string) string {
	if i := strings.IndexByte(p, '#'); i >= 0 {
		p = p[:i] // 문서 안 앵커는 관계가 아니다. #만 있으면 경로가 빈다
	}
	if !strings.HasSuffix(p, ".md") {
		return "" // 이미지와 외부 자료는 관계가 아니다
	}
	if schemePattern.MatchString(p) {
		return "" // 외부 링크는 위키 안의 관계가 아니다
	}
	return strings.TrimSuffix(path.Base(p), ".md")
}

// FrontmatterLinks는 프론트매터 related 필드의 링크를 반환한다.
// 링크 문법을 설명하는 문서에서 게이트가 셀 링크를 본문 것과 섞지 않도록 본문과 분리했다.
func (d Doc) FrontmatterLinks() []Link {
	var values []string
	for _, f := range d.Fields {
		if f.Key == "related" && f.Kind == KindStringList {
			values = f.List
		}
	}
	if len(values) == 0 {
		return nil
	}
	// 파싱된 값을 진실원으로 쓰고 줄 번호만 원문에서 잡는다.
	// 흐름 목록(related: [a, b])처럼 항목 줄을 못 잡으면 키가 있는 줄로 묶는다.
	keyLine, itemLines := relatedLines(d.fmLines)
	links := make([]Link, 0, len(values))
	for i, v := range values {
		line := keyLine
		if i < len(itemLines) {
			line = itemLines[i]
		}
		links = append(links, Link{Slug: stripBrackets(v), Line: line})
	}
	return links
}

// slugOf는 링크 내용에서 표시문자와 헤딩을 버리고 슬러그만 남긴다.
// [[#헤딩]]처럼 슬러그가 비면 같은 문서 안 참조이므로 링크로 치지 않는다.
func slugOf(content string) string {
	if i := strings.IndexAny(content, "#|"); i >= 0 {
		content = content[:i]
	}
	return strings.TrimSpace(content)
}

// stripBrackets는 related 항목이 [[슬러그]] 형태로 적혀 있을 경우 껍데기를 벗긴다.
func stripBrackets(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[[")
	return strings.TrimSuffix(v, "]]")
}

// outsideCode는 한 줄을 백틱 기준으로 나눠 코드가 아닌 구간만 남긴다.
func outsideCode(line string) []string {
	parts := strings.Split(line, "`")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		if i%2 == 0 {
			out = append(out, p)
		}
	}
	return out
}

// relatedLines는 프론트매터 원문에서 related 키 줄과 블록 목록 항목 줄의
// 원본 파일 줄 번호를 찾는다. 프론트매터 본문은 파일 2줄부터 시작한다.
func relatedLines(fmLines []string) (keyLine int, itemLines []int) {
	keyLine = 2
	inList := false
	for i, line := range fmLines {
		t := strings.TrimSpace(line)
		if !inList {
			if strings.HasPrefix(line, "related:") {
				keyLine = i + 2
				inList = true
			}
			continue
		}
		if strings.HasPrefix(t, "-") {
			itemLines = append(itemLines, i+2)
			continue
		}
		break
	}
	return keyLine, itemLines
}
