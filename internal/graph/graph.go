// Package graph는 위키 전체의 링크 관계를 계산한다.
// backlinks 가 찾는 것과 mv 가 고쳐야 하는 것이 같은 집합이므로 그
// 계산을 한 곳에 둔다. 링크가 어느 파일 어느 줄 어느 필드에 있는지를
// 반환해 mv가 그 자리를 고칠 수 있게 하는 것이 이 패키지의 계약이다.
package graph

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/walk"
)

// Kind는 링크가 문서의 어느 부분에서 왔는지 구분한다.
type Kind string

const (
	KindBody           Kind = "body"            // 본문의 위키링크
	KindRelated        Kind = "related"         // related 필드
	KindDerivedFrom    Kind = "derived_from"    // 파생 출처 필드
	KindDerivedContext Kind = "derived_context" // 파생 맥락 필드
	KindSourceRefs     Kind = "source_refs"     // 원본 출처 필드
)

// Label은 사람용 출력에 쓰는 링크 종류 이름이다.
func (k Kind) Label() string {
	switch k {
	case KindBody:
		return "본문 링크"
	case KindRelated:
		return "related 필드"
	case KindDerivedFrom:
		return "derived_from 필드"
	case KindDerivedContext:
		return "derived_context 필드"
	case KindSourceRefs:
		return "source_refs 필드"
	}
	return string(k)
}

// Link는 링크 하나다. 출처와 위치와 원본 값을 모두 담아 링크를 고치는
// 쪽(mv)이 글자 단위로 자리를 찾을 수 있게 한다.
type Link struct {
	From  string // 출처 문서의 위키 루트 상대 슬래시 경로
	Line  int    // 출처 줄 번호. 1 기반
	Field Kind   // 링크가 온 필드
	Slug  string // 정규화된 가리킴 슬러그
	Raw   string // 원본 값. 껍데기와 경로 형태를 그대로 담는다
}

// Graph는 위키 전체의 링크 관계다.
type Graph struct {
	links       []Link
	pathsBySlug map[string]string
}

// Build는 순회 결과로 링크 그래프를 만든다. 순회는 internal/walk 의
// 결과를 입력으로 받는다. 결과 링크는 출처 경로순, 같은 파일이면 줄
// 번호순으로 정렬된다.
func Build(walked []walk.Doc) *Graph {
	g := &Graph{pathsBySlug: map[string]string{}}
	for _, wd := range walked {
		g.pathsBySlug[Normalize(wd.Rel)] = wd.Rel
	}
	for _, wd := range walked {
		if wd.Err != nil {
			continue
		}
		for _, l := range wd.Parsed.BodyLinks() {
			g.add(Link{From: wd.Rel, Line: l.Line, Field: KindBody,
				Slug: Normalize(l.Slug), Raw: "[[" + l.Slug + "]]"})
		}
		g.addRelated(wd)
		for _, key := range []struct {
			field Kind
			name  string
		}{
			{KindDerivedFrom, "derived_from"},
			{KindDerivedContext, "derived_context"},
			{KindSourceRefs, "source_refs"},
		} {
			for _, raw := range fieldValues(wd.Parsed, key.name) {
				g.add(Link{From: wd.Rel, Line: lineOfKey(wd.Content, key.name),
					Field: key.field, Slug: Normalize(raw), Raw: raw})
			}
		}
	}
	sort.Slice(g.links, func(i, j int) bool {
		if g.links[i].From != g.links[j].From {
			return g.links[i].From < g.links[j].From
		}
		if g.links[i].Line != g.links[j].Line {
			return g.links[i].Line < g.links[j].Line
		}
		return g.links[i].Field < g.links[j].Field
	})
	return g
}

// add는 링크를 추가한다. 빈 슬러그는 링크가 아니다.
func (g *Graph) add(l Link) {
	if l.Slug != "" {
		g.links = append(g.links, l)
	}
}

// addRelated는 related 필드의 링크를 추가한다. 줄 번호는 프론트매터
// 링크 추출이 잡아 주는 항목 줄을 쓰고 못 잡으면 키 줄로 묶는다.
func (g *Graph) addRelated(wd walk.Doc) {
	f, ok := fieldOf(wd.Parsed, "related")
	if !ok {
		return
	}
	lines := wd.Parsed.FrontmatterLinks()
	for i, raw := range relatedValues(f) {
		line := lineOfKey(wd.Content, "related")
		if i < len(lines) {
			line = lines[i].Line
		}
		g.add(Link{From: wd.Rel, Line: line, Field: KindRelated,
			Slug: Normalize(raw), Raw: raw})
	}
}

// relatedValues는 related 필드의 값을 원본 그대로 낸다.
func relatedValues(f doc.Field) []string {
	switch f.Kind {
	case doc.KindStringList:
		return f.List
	case doc.KindString:
		if f.Str != "" {
			return []string{f.Str}
		}
	}
	return nil
}

// fieldOf는 프론트매터 필드 하나를 찾는다.
func fieldOf(d doc.Doc, key string) (doc.Field, bool) {
	for _, f := range d.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return doc.Field{}, false
}

// fieldValues는 관계 필드의 값을 목록이나 문자열에서 꺼낸다.
func fieldValues(d doc.Doc, key string) []string {
	f, ok := fieldOf(d, key)
	if !ok {
		return nil
	}
	switch f.Kind {
	case doc.KindStringList:
		return f.List
	case doc.KindString:
		if f.Str != "" {
			return []string{f.Str}
		}
	}
	return nil
}

// Backlinks는 슬러그를 가리키는 링크를 반환한다. 질의 슬러그도 같은
// 규칙으로 정규화하므로 경로 형태로 물어도 된다.
func (g *Graph) Backlinks(slug string) []Link {
	key := Normalize(slug)
	var out []Link
	for _, l := range g.links {
		if l.Slug == key {
			out = append(out, l)
		}
	}
	return out
}

// Outgoing은 한 문서가 거는 링크를 반환한다.
func (g *Graph) Outgoing(path string) []Link {
	var out []Link
	for _, l := range g.links {
		if l.From == path {
			out = append(out, l)
		}
	}
	return out
}

// Has는 슬러그에 해당하는 문서가 위키에 있는지를 반환한다.
// 백링크가 있어도 문서가 없으면 깨진 링크다.
func (g *Graph) Has(slug string) bool {
	_, ok := g.pathsBySlug[Normalize(slug)]
	return ok
}

// Normalize는 링크 대상을 비교 키로 정규화한다. internal/lint 의
// relationSlug 와 같은 규칙이어야 고아 판정과 백링크가 같은 문서 집합을
// 본다. 위키링크 껍데기를 벗기고, 경로면 파일명만 남기고, 확장자와
// 날짜 접두사(YYYY-MM-DD- 또는 YYYY-MM-)를 뗀다.
func Normalize(v string) string {
	v = strings.TrimSuffix(strings.TrimPrefix(v, "[["), "]]")
	base := strings.TrimSuffix(filepath.Base(v), ".md")
	if len(base) > 11 && isDayPrefix(base[:11]) {
		return base[11:]
	}
	if len(base) > 8 && isMonthPrefix(base[:8]) {
		return base[8:]
	}
	return base
}

// isDayPrefix는 앞 11글자가 YYYY-MM-DD- 형태인지 본다.
func isDayPrefix(s string) bool {
	return isDigits(s[:4]) && s[4] == '-' && isDigits(s[5:7]) && s[7] == '-' && isDigits(s[8:10]) && s[10] == '-'
}

// isMonthPrefix는 앞 8글자가 YYYY-MM- 형태인지 본다.
func isMonthPrefix(s string) bool {
	return isDigits(s[:4]) && s[4] == '-' && isDigits(s[5:7]) && s[7] == '-'
}

// isDigits는 문자열이 모두 숫자인지 본다. 호출자가 길이를 보장한다.
func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// lineOfKey는 원문에서 "키:" 줄을 찾아 1 기반 줄 번호를 반환한다.
// 못 찾으면 1을 쓴다.
func lineOfKey(content, key string) int {
	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), key+":") {
			return i + 1
		}
	}
	return 1
}
