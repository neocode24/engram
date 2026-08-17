package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
)

// flagRelated는 promote와 new가 함께 쓰는 링크 추가 플래그 이름이다.
const flagRelated = "related"

// gateSummary는 --json이 내는 게이트 판정 요약이다.
type gateSummary struct {
	Passed   bool `json:"passed"`
	Deferred bool `json:"deferred"`
	Links    int  `json:"links"`
	Targets  int  `json:"targets"`
	Min      int  `json:"min"`
}

// writeOutcome은 promote와 new의 공통 결과다. --json 출력에 그대로 쓰인다.
type writeOutcome struct {
	Path  string      `json:"path"`
	Slug  string      `json:"slug"`
	Stage string      `json:"stage"`
	Gate  gateSummary `json:"gate"`
}

// gateOf는 판정 결과를 JSON 요약으로 바꾼다.
func gateOf(g lint.GateResult) gateSummary {
	return gateSummary{Passed: g.Passed, Deferred: g.Deferred, Links: g.Links, Targets: g.Targets, Min: g.Min}
}

// linkSlugs는 문서가 가진 위키링크의 고유 슬러그 집합을 반환한다.
// related 필드와 본문 링크를 합치고 중복은 한 번만 센다.
func linkSlugs(d doc.Doc) map[string]bool {
	out := map[string]bool{}
	for _, l := range d.FrontmatterLinks() {
		out[l.Slug] = true
	}
	for _, l := range d.BodyLinks() {
		out[l.Slug] = true
	}
	return out
}

// gateInputs는 게이트에 넣을 두 수를 낸다. 실제로 이어지는 링크 수와
// 링크 대상 수다. 자기 자신은 뺀다. 세 곳(lint, status, 이곳)이 같은
// 집계를 쓰지 않으면 커맨드로 통과한 문서를 lint 가 거절하는 상태가 생긴다.
func gateInputs(links map[string]bool, walked []walk.Doc, skipRel, skipSlug string) (int, int) {
	targets := lint.LinkableSlugs(walked, skipRel, skipSlug)
	return lint.ResolvedLinks(links, targets), len(targets)
}

// slugOfRel는 순회 경로에서 문서 슬러그를 낸다.
func slugOfRel(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), ".md")
}

// knownSlugs는 순회 결과에서 슬러그 집합을 만든다.
func knownSlugs(walked []walk.Doc) map[string]bool {
	out := map[string]bool{}
	for _, w := range walked {
		if w.Err == nil {
			out[slugOfRel(w.Rel)] = true
		}
	}
	return out
}

// gateRejectError는 게이트 거절 안내를 만든다. 무엇이 부족하고 어떻게
// 채우는지를 함께 낸다. 거절 메시지 품질이 제품 품질이다.
func gateRejectError(g lint.GateResult) error {
	need := g.Min - g.Links
	return errors.New(i18n.T("cli.gate.reject", g.Links, g.Min, need))
}

// warnDeferred는 게이트 유예 사실을 경고로 알린다. ADR 0021.
func warnDeferred(w io.Writer, g lint.GateResult) {
	fmt.Fprintln(w, i18n.T("cli.gate.deferred", g.Targets, g.Min))
}

// warnUnknownRelated는 위키에 없는 --related 슬러그를 알린다. 곧 만들
// 문서일 수 있으므로 막지 않는다.
func warnUnknownRelated(w io.Writer, related []string, known map[string]bool) {
	for _, s := range related {
		if !known[s] {
			fmt.Fprintln(w, i18n.T("cli.gate.unknown_related", s))
		}
	}
}

// createDocFile은 경로에 문서 바이트를 처음 쓴다. 디렉토리가 없으면 만들고
// 이미 파일이 있으면 기존 경로를 내며 거절한다.
func createDocFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.gate.mkdir_fail"), err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return errors.New(i18n.T("cli.gate.dest_exists", path))
	}
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.gate.create_fail", path), err)
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		return fmt.Errorf("%s: %w", i18n.T("cli.gate.write_fail", path), err)
	}
	return f.Close()
}
