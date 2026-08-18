// Package resurface는 오래 안 본 context 문서를 다시 꺼낸다.
// 재발견 커맨드 중 유일하게 상태를 읽고 쓴다. 제시 이력은
// .engram/resurface.json 에 두며 사라져도 중복 제시가 생길 뿐이므로
// 이력이 깨졌다고 도구가 멈추지 않는다. ADR 0028.
package resurface

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/walk"
)

// 제시 이력 파일의 위치와 형식 버전이다.
const (
	stateDir       = ".engram"
	historyName    = "resurface.json"
	historyVersion = 1
)

// CooldownDays는 한 번 제시한 문서를 후보에서 빼 두는 기간이다.
// 설정으로 노출하지 않는다(ADR 0010 과 같은 이유). upstream 과 같은 값이다.
const CooldownDays = 21

// Candidate는 다시 꺼낼 문서 하나다. 근거가 되는 경과일과 제시 이력을
// 함께 실는다. 문장은 만들지 않는다(design.md 재발견 절).
type Candidate struct {
	Slug      string     `json:"slug"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	AgeDays   int        `json:"daysSinceUpdate"`
	Inbound   int        `json:"inboundLinks"`
	Score     float64    `json:"score"`
	LastShown *time.Time `json:"lastShown"`
	Cooldown  bool       `json:"cooldown"`
}

// Unlinked는 아무도 가리키지 않는 context 문서다. 재발견의 고아 판정은
// 인바운드만 본다. lint 의 graph.orphan 과 다른 것을 재므로 형도 이름도
// 나눠 둔다(ADR 0066).
type Unlinked struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// Result는 resurface 실행 결과다. 후보가 없으면 Reason 에 이유가 들어간다.
type Result struct {
	StaleDays      int         `json:"staleDays"`
	Now            time.Time   `json:"now"`
	SkippedNoDate  int         `json:"skippedNoDate"`
	Candidates     []Candidate `json:"candidates"`
	CooldownDays   int         `json:"cooldownDays"`
	CooldownFilled int         `json:"cooldownFilled"`
	NoInbound      []Unlinked  `json:"noInbound"`
	Reason         string      `json:"reason,omitempty"`
}

// BaseDate는 문서의 기준 날짜를 반환한다. updated 를 우선하고 없으면
// created 를 쓴다. 둘 다 없으면 false 다. digest 의 노후 집계도 이 규칙을
// 쓰므로 노후 판정의 단일 진실원을 여기에 둔다.
func BaseDate(d doc.Doc) (time.Time, bool) {
	for _, key := range []string{"updated", "created"} {
		for _, f := range d.Fields {
			if f.Key == key && f.Str != "" {
				if t, ok := parseDate(f.Str); ok {
					return t, true
				}
			}
		}
	}
	return time.Time{}, false
}

// IsStale는 문서의 기준 날짜가 staleDays 를 넘었는지를 판정한다.
// 기준 날짜가 없는 문서는 노후가 아니라 대상 밖이다(개수는 따로 센다).
func IsStale(d doc.Doc, now time.Time, staleDays int) bool {
	base, ok := BaseDate(d)
	if !ok {
		return false
	}
	return days(now, base) > staleDays
}

// Run은 오래 안 본 context 문서를 골라 낸다. dryRun 이 아니면 낸 문서의
// 제시 시각을 now 값으로 기록한다. 상태를 쓰는 유일한 조회 커맨드라
// 같은 위키라도 실행마다 결과가 달라진다.
func Run(wikiRoot string, cfg config.Config, now time.Time, limit int, dryRun bool) (Result, error) {
	walked, err := walk.Files(wikiRoot, cfg)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", i18n.T("core.resurface.walk_fail"), err)
	}
	shown := LoadHistory(wikiRoot)
	inbound := inboundCounts(walked)

	res := Result{
		StaleDays:    cfg.Thresholds.StaleDays,
		Now:          now,
		Candidates:   []Candidate{},
		CooldownDays: CooldownDays,
		NoInbound:    []Unlinked{},
	}

	// 정렬에 제시 이력이 필요하므로 후보를 Candidate 로 바꾸는 단계에서
	// 이력을 함께 붙인다.
	type entry struct {
		cand  Candidate
		shown time.Time
		ever  bool
	}
	var entries []entry
	contextDocs := 0
	for _, w := range walked {
		if stageOfDir(w.Rel) != "context" {
			continue
		}
		contextDocs++
		slug := slugOf(w.Rel)
		// 재발견의 고아는 인바운드 0 하나로 정해진다. 노후 여부와
		// 무관하므로 날짜를 읽기 전에 센다.
		if inbound[slug] == 0 {
			res.NoInbound = append(res.NoInbound, Unlinked{
				Slug: slug, Title: titleOf(w.Parsed), Path: w.Rel})
		}
		// 파싱에 실패한 문서는 기준 날짜를 읽을 수 없으므로 날짜 없음으로 센다.
		if w.Err != nil || !w.Parsed.HasFrontmatter {
			res.SkippedNoDate++
			continue
		}
		base, ok := BaseDate(w.Parsed)
		if !ok {
			res.SkippedNoDate++
			continue
		}
		age := days(now, base)
		if age <= cfg.Thresholds.StaleDays {
			continue
		}
		e := entry{cand: Candidate{
			Slug:    slug,
			Title:   titleOf(w.Parsed),
			Path:    w.Rel,
			AgeDays: age,
			Inbound: inbound[slug],
			Score:   score(age, inbound[slug]),
		}}
		if t, ok := shown[slug]; ok {
			e.cand.LastShown = &t
			e.shown, e.ever = t, true
		}
		entries = append(entries, e)
	}
	sort.Slice(res.NoInbound, func(i, j int) bool { return res.NoInbound[i].Slug < res.NoInbound[j].Slug })

	// 제시 이력은 정렬 키가 아니라 제외 필터다. 마지막 제시를 정렬
	// 1차 키로 두면 점수가 순위를 바꾸지 못해 인바운드 가중치가 무의미해진다.
	// 쿨다운 안의 문서를 뺀다. 다만 제외가 후보를 고갈시키면 무시하고
	// 점수 순으로 도로 채운다. upstream 위키는 300 문서라 제외해도 후보가
	// 남지만 engram 의 대상에는 문서 열댓 개짜리 위키가 있다. 재발견이
	// 빈 결과를 내면 그 기능을 안 쓰게 되므로 빈 결과보다 반복 노출이 낫다.
	var fresh, cooled []entry
	for _, e := range entries {
		if e.ever && days(now, e.shown) < CooldownDays {
			e.cand.Cooldown = true
			cooled = append(cooled, e)
			continue
		}
		fresh = append(fresh, e)
	}
	// 정렬은 가중 점수 내림차순, 동점은 슬러그 오름차순 둘뿐이다.
	byScore := func(es []entry) {
		sort.Slice(es, func(i, j int) bool {
			a, b := es[i], es[j]
			if a.cand.Score != b.cand.Score {
				return a.cand.Score > b.cand.Score
			}
			return a.cand.Slug < b.cand.Slug
		})
	}
	byScore(fresh)
	byScore(cooled)
	entries = append(fresh, cooled...)
	if limit >= 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	for _, e := range entries {
		if e.cand.Cooldown {
			res.CooldownFilled++
		}
		res.Candidates = append(res.Candidates, e.cand)
	}

	if len(res.Candidates) == 0 {
		switch {
		case contextDocs == 0:
			res.Reason = i18n.T("core.resurface.no_context")
		case res.SkippedNoDate == contextDocs:
			res.Reason = i18n.T("core.resurface.no_dated")
		default:
			res.Reason = i18n.T("core.resurface.none_stale", cfg.Thresholds.StaleDays)
		}
	}

	if !dryRun && len(entries) > 0 {
		for _, e := range entries {
			shown[e.cand.Slug] = now
		}
		if err := SaveHistory(wikiRoot, shown); err != nil {
			return res, err
		}
	}
	return res, nil
}

// LoadHistory는 제시 이력을 읽는다. 파일이 없거나 파싱에 실패하거나
// 스키마 버전이 다르면 빈 이력으로 취급한다. 재생성 가능한 캐시가 깨졌다고
// 도구가 멈추면 안 된다.
func LoadHistory(wikiRoot string) map[string]time.Time {
	data, err := os.ReadFile(filepath.Join(wikiRoot, stateDir, historyName))
	if err != nil {
		return map[string]time.Time{}
	}
	var f struct {
		SchemaVersion int               `json:"schemaVersion"`
		Shown         map[string]string `json:"shown"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return map[string]time.Time{}
	}
	if f.SchemaVersion != historyVersion {
		return map[string]time.Time{}
	}
	out := make(map[string]time.Time, len(f.Shown))
	for slug, raw := range f.Shown {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			// 일부 항목이 깨져도 나머지 이력은 쓴다.
			continue
		}
		out[slug] = t
	}
	return out
}

// SaveHistory는 제시 이력을 .engram/resurface.json 에 쓴다. 슬러그 키는
// encoding/json 이 정렬하므로 같은 상태를 두 번 저장하면 바이트까지 같다.
func SaveHistory(wikiRoot string, shown map[string]time.Time) error {
	f := struct {
		SchemaVersion int               `json:"schemaVersion"`
		Shown         map[string]string `json:"shown"`
	}{SchemaVersion: historyVersion, Shown: make(map[string]string, len(shown))}
	for slug, t := range shown {
		f.Shown[slug] = t.UTC().Format(time.RFC3339)
	}
	dir := filepath.Join(wikiRoot, stateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("core.resurface.mkdir_fail"), err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("core.resurface.marshal_fail"), err)
	}
	data = append(data, '\n')
	path := filepath.Join(dir, historyName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("core.resurface.history_write_fail", path), err)
	}
	return nil
}

// score는 재발견 가중 점수다. 경과일에 인바운드 링크 수의 역수 가중을
// 곱한다. 링크가 적을수록 잊히기 쉬우므로 가중이 커진다. upstream 의
// 식과 같다(ADR 0066).
func score(ageDays, inbound int) float64 {
	return float64(ageDays) * (1.0 + 1.0/(1.0+float64(inbound)))
}

// inboundCounts는 슬러그마다 그것을 가리키는 다른 문서의 수를 센다.
// 같은 문서가 여러 번 걸어도 하나로 세고 자기 자신을 가리키는 링크는
// 세지 않는다. 잊힘의 강도는 남이 얼마나 가리키는지로 정해진다.
func inboundCounts(walked []walk.Doc) map[string]int {
	g := graph.Build(walked)
	out := map[string]int{}
	for _, w := range walked {
		if stageOfDir(w.Rel) != "context" {
			continue
		}
		slug := slugOf(w.Rel)
		froms := map[string]bool{}
		for _, l := range g.Backlinks(slug) {
			if l.From != w.Rel {
				froms[l.From] = true
			}
		}
		out[slug] = len(froms)
	}
	return out
}

// stageOfDir는 문서 위치의 첫 디렉토리로 단계를 잡는다. 승급은 파일을
// 옮기는 행위이므로 위치가 운영의 진실원이다(status 와 같은 세는 법).
func stageOfDir(rel string) string {
	seg, _, _ := strings.Cut(rel, "/")
	return seg
}

// slugOf는 문서 경로에서 슬러그를 낸다. context 문서는 날짜 접두사가
// 없으므로 파일명에서 확장자만 뗀 값이 슬러그다(lint 의 bySlug 와 같은 법).
func slugOf(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), ".md")
}

// titleOf는 본문 첫 머리글에서 제목을 낸다. 머리글이 없으면 빈 값이다.
func titleOf(d doc.Doc) string {
	for _, line := range strings.Split(d.Body, "\n") {
		t := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if t != "" {
			return t
		}
	}
	return ""
}

// parseDate는 하루 단위와 연월 단위 두 형식을 파싱한다. status 의 날짜
// 해석과 같은 두 형식이다.
func parseDate(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// days는 기준 시각에서 날짜까지의 경과일을 낸다.
func days(now, t time.Time) int {
	return int(now.Sub(t).Hours() / 24)
}
