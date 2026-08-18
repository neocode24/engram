// Package export는 위키의 일부를 밖으로 내보낼 번들을 만든다.
//
// 무엇이 나가는지는 internal/expose 가 판정한다(ADR 0044). 여기서 규칙을
// 다시 만들지 않는다. 이 패키지가 더하는 것은 둘이다. 명시 슬러그로
// 좁히는 것과 사용자가 준 사전으로 식별자를 치환하는 것이다.
//
// 형태를 바꾸지 않는다. 마크다운이 마크다운으로 나가며 병합도 압축도
// 포맷 변환도 하지 않는다. 그것은 다른 도구의 일이다(ADR 0046).
package export

import (
	"errors"
	"path"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/expose"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
)

// Options는 반출 범위와 익명화 사전이다.
type Options struct {
	// IncludeArchive는 archive 문서도 반출할지 정한다.
	IncludeArchive bool
	// IncludeInternal은 sensitivity 가 internal 인 문서도 반출할지
	// 정한다. 기본은 거짓이다. 파일이 기계 밖으로 나가므로 기본이 전체
	// 노출이면 사고가 난다(ADR 0063).
	IncludeInternal bool
	// Slugs는 반출할 문서를 좁힌다. 비면 노출 규칙을 지난 전부다.
	// 링크를 따라가지 않는다. 지목한 것만 나간다.
	Slugs []string
	// Rules는 익명화 치환 규칙이다. 비면 치환하지 않는다.
	Rules []Rule
}

// Rule은 치환 규칙 하나다. To 는 비어도 된다. 지우는 규칙이 된다.
type Rule struct {
	From string
	To   string
}

// Hit은 규칙 하나의 적용 건수다. 0건인 규칙은 사전의 오타일 가능성이
// 높으므로 호출자가 따로 알린다.
type Hit struct {
	Rule  Rule
	Count int
}

// File은 번들에 들어갈 파일 하나다.
type File struct {
	// Rel은 번들 안에서의 상대 슬래시 경로다. 파일명에 치환이 적용된 뒤의
	// 이름이며 디렉토리 구조는 위키 그대로다.
	Rel string
	// Source는 위키 안에서의 원본 상대 경로다. 안내와 충돌 보고에 쓴다.
	Source string
	// Content는 치환을 마친 파일 전체다. 프론트매터를 포함한다.
	Content string
}

// Result는 반출 계획이다. 파일을 쓰기 전에 무엇이 나가는지 전부 담는다.
type Result struct {
	// Files는 번들에 들어갈 파일이다. 위키 순회 순서를 따르므로 경로순이다.
	Files []File
	// Exposure는 노출 규칙이 무엇을 걸렀는지의 집계다.
	Exposure expose.Exposure
	// ExcludedByFilter는 노출 규칙은 지났으나 명시 슬러그에 들지 않아
	// 빠진 문서 수다. 슬러그를 주지 않았으면 0이다.
	ExcludedByFilter int
	// Hits는 규칙별 치환 건수다. Options.Rules 의 적용 순서와 같다.
	Hits []Hit
	// DanglingLinks는 번들 밖을 가리키는 위키링크 수다. 본문을 고치지
	// 않으므로 링크는 그대로 나간다. 수만 알린다(ADR 0046).
	DanglingLinks int
	// DanglingSlugs는 그 링크들이 가리키는 슬러그다. 중복 없이 사전순이다.
	DanglingSlugs []string
}

// Replaced는 치환이 한 건이라도 일어났는지다.
func (r Result) Replaced() int {
	n := 0
	for _, h := range r.Hits {
		n += h.Count
	}
	return n
}

// UnusedRules는 한 번도 걸리지 않은 규칙이다. 사전에 오타가 있으면
// 조용히 통과하는데 그것이 익명화 실패의 가장 흔한 형태다.
func (r Result) UnusedRules() []Rule {
	var out []Rule
	for _, h := range r.Hits {
		if h.Count == 0 {
			out = append(out, h.Rule)
		}
	}
	return out
}

// Plan은 반출 계획을 만든다. 파일을 쓰지 않는다.
func Plan(root string, cfg config.Config, opts Options) (Result, error) {
	sel, err := expose.Select(root, cfg, exposeOptions(opts))
	if err != nil {
		return Result{}, err
	}
	res := Result{Exposure: sel.Exposure}

	chosen, err := narrow(sel, opts, cfg)
	if err != nil {
		return Result{}, err
	}
	res.ExcludedByFilter = len(sel.Docs) - len(chosen)

	// 번들에 드는 슬러그 집합이다. 링크가 번들 밖을 가리키는지 이것으로
	// 본다. 치환 전 슬러그로 세는 이유는 링크도 같은 치환을 받기 때문이다.
	inBundle := map[string]bool{}
	for _, ed := range chosen {
		inBundle[graph.Normalize(ed.Rel)] = true
	}

	rules := sortRules(opts.Rules)
	counts := make([]int, len(rules))
	dangling := map[string]bool{}

	for _, ed := range chosen {
		for _, l := range append(ed.Parsed.BodyLinks(), ed.Parsed.FrontmatterLinks()...) {
			if !inBundle[graph.Normalize(l.Slug)] {
				res.DanglingLinks++
				dangling[l.Slug] = true
			}
		}
		content, c1 := apply(ed.Content, rules)
		rel, c2, err := renameRel(ed.Rel, rules)
		if err != nil {
			return Result{}, err
		}
		addCounts(counts, c1)
		addCounts(counts, c2)
		res.Files = append(res.Files, File{Rel: rel, Source: ed.Rel, Content: content})
	}

	if err := checkCollisions(res.Files); err != nil {
		return Result{}, err
	}

	res.Hits = make([]Hit, len(rules))
	for i, r := range rules {
		res.Hits[i] = Hit{Rule: r, Count: counts[i]}
	}
	res.DanglingSlugs = sortedKeys(dangling)
	return res, nil
}

// exposeOptions는 반출 선택지를 노출 판정의 선택지로 옮긴다. 두 곳에서
// 손으로 옮기면 한쪽만 고쳐질 수 있어 한 자리에 둔다.
func exposeOptions(opts Options) expose.Options {
	return expose.Options{IncludeArchive: opts.IncludeArchive, IncludeInternal: opts.IncludeInternal}
}

// narrow는 명시 슬러그로 대상을 좁힌다. 제외된 문서를 지목하면 왜
// 제외되었는지 알리고 멈춘다. 슬러그 지정이 노출 규칙을 뚫는 경로가
// 되면 민감도 제외가 이름만 바꿔 무력해지기 때문이다(ADR 0046).
func narrow(sel expose.Result, opts Options, cfg config.Config) ([]expose.Doc, error) {
	if len(opts.Slugs) == 0 {
		return sel.Docs, nil
	}
	bySlug := map[string]expose.Doc{}
	for _, ed := range sel.Docs {
		s := graph.Normalize(ed.Rel)
		// 같은 슬러그가 둘이면 먼저 나온 문서를 쓴다. 순회가 경로순이라
		// 이 선택은 결정론적이다.
		if _, dup := bySlug[s]; !dup {
			bySlug[s] = ed
		}
	}
	picked := map[string]bool{}
	var out []expose.Doc
	for _, want := range opts.Slugs {
		s := graph.Normalize(want)
		ed, ok := bySlug[s]
		if !ok {
			return nil, notExportable(sel, want, s, cfg, exposeOptions(opts))
		}
		if picked[s] {
			continue
		}
		picked[s] = true
		out = append(out, ed)
	}
	// 순회 순서를 되살린다. 인자 순서가 번들의 파일 순서를 바꾸면
	// 같은 집합을 다르게 적은 두 실행의 결과가 달라진다.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// notExportable은 지목한 슬러그가 왜 나갈 수 없는지 설명하는 오류를 만든다.
func notExportable(sel expose.Result, want, norm string, cfg config.Config, opts expose.Options) error {
	for _, wd := range sel.Walked {
		if graph.Normalize(wd.Rel) != norm {
			continue
		}
		loc := expose.Locate(wd)
		reason := expose.Exclude(wd, loc, cfg, opts)
		return errors.New(i18n.T("core.export.not_exportable", want, expose.ReasonText(reason)))
	}
	return errors.New(i18n.T("core.export.no_such_doc", want))
}

// sortRules는 적용 순서를 정한다. 원문이 긴 규칙이 먼저다. 겹치는 규칙이
// 있을 때 긴 쪽이 걸려야 한다. 길이가 같으면 사전순이라 순서가 결정론이다.
func sortRules(rules []Rule) []Rule {
	out := append([]Rule(nil), rules...)
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].From) != len(out[j].From) {
			return len(out[i].From) > len(out[j].From)
		}
		return out[i].From < out[j].From
	})
	return out
}

// apply는 규칙을 한 번의 훑기로 적용한다. 치환 결과를 다시 치환하지
// 않으므로 A 를 B 로 바꾸는 규칙과 B 를 C 로 바꾸는 규칙이 함께 있어도
// A 가 C 가 되지 않는다.
//
// ponytail: 위치마다 규칙 전부를 훑으므로 O(문서길이 * 규칙수)다. 위키
// 규모에서는 문제가 아니다. 사전이 수천 줄이 되면 트라이로 바꾼다.
func apply(s string, rules []Rule) (string, []int) {
	counts := make([]int, len(rules))
	if len(rules) == 0 {
		return s, counts
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		matched := -1
		for ri, r := range rules {
			if strings.HasPrefix(s[i:], r.From) {
				matched = ri
				break
			}
		}
		if matched < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		b.WriteString(rules[matched].To)
		i += len(rules[matched].From)
		counts[matched]++
	}
	return b.String(), counts
}

// renameRel은 파일명에 치환을 적용한다. 디렉토리 경로는 건드리지 않는다.
// 단계 디렉토리 이름은 설정이 정하는 것이고 사용자 식별자가 아니다.
// 본문만 치환하고 파일명을 두면 이름에 박힌 식별자가 그대로 나간다.
func renameRel(rel string, rules []Rule) (string, []int, error) {
	dir, base := path.Split(rel)
	renamed, counts := apply(base, rules)
	if renamed == "" {
		return "", nil, errors.New(i18n.T("core.export.rename_empty", rel))
	}
	if strings.ContainsAny(renamed, `/\`) {
		return "", nil, errors.New(i18n.T("core.export.rename_path_sep", rel, renamed))
	}
	return dir + renamed, counts, nil
}

// checkCollisions는 치환 뒤 두 문서가 같은 이름이 되는지 본다. 조용히
// 덮으면 반출물에서 문서 하나가 사라진다.
func checkCollisions(files []File) error {
	seen := map[string]string{}
	for _, f := range files {
		if prev, dup := seen[f.Rel]; dup {
			return errors.New(i18n.T("core.export.collision", prev, f.Source, f.Rel))
		}
		seen[f.Rel] = f.Source
	}
	return nil
}

// addCounts는 규칙별 건수를 누적한다.
func addCounts(dst, src []int) {
	for i := range src {
		dst[i] += src[i]
	}
}

// sortedKeys는 집합을 사전순 목록으로 만든다.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
