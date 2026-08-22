// Package status는 위키 현황과 밀린 것을 계산한다.
// 숫자 나열이 아니라 지금 무엇을 해야 하는지가 보여야 하므로
// 지표에서 파생한 다음 행동 제안을 함께 낸다.
package status

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
)

// Stages는 단계별 문서 수다. 단계는 문서 위치(디렉토리) 기준으로 잡는다.
// 승급은 파일을 옮기는 행위이므로 운영의 진실원은 위치고,
// 프론트매터 값과 어긋나는 것은 lint 가 판정한다.
type Stages struct {
	Inbox   int `json:"inbox"`
	Source  int `json:"source"`
	Context int `json:"context"`
	Archive int `json:"archive"`
}

// LintSummary는 lint 결과의 요약이다. 상세는 내지 않는다.
type LintSummary struct {
	Files  int `json:"files"`
	Error  int `json:"error"`
	Warn   int `json:"warn"`
	Reject int `json:"reject"`
}

// Backlog은 아직 처리하지 않은 것의 지표다. OldestDays 는 날짜를 아는 inbox 문서가
// 없으면 nil 이다. 나이를 모르는 문서는 UnknownAge 로 따로 세고 0 으로 치지 않는다.
type Backlog struct {
	Inbox           int      `json:"inbox"`
	OldestDays      *int     `json:"oldestDays"`
	UnknownAge      int      `json:"unknownAge"`
	Stale           int      `json:"stale"`
	Promotable      int      `json:"promotable"`
	PromotablePaths []string `json:"promotablePaths"`
}

// Suggestion은 다음 행동 제안 하나다. Action 은 명령이나 행동, Detail 은 근거다.
type Suggestion struct {
	Action string `json:"action"`
	Detail string `json:"detail"`
}

// Result는 status 계산 결과 전부다.
type Result struct {
	Stages      Stages       `json:"stages"`
	Links       int          `json:"links"`
	Orphans     int          `json:"orphans"`
	Lint        LintSummary  `json:"lint"`
	Backlog     Backlog      `json:"backlog"`
	Suggestions []Suggestion `json:"suggestions"`
}

// scannedDoc는 문서 하나의 위치와 파싱 결과다.
type scannedDoc struct {
	rel    string
	root   bool // 색인(root_files) 문서인가
	parsed doc.Doc
	ok     bool // 프론트매터까지 파싱됐는가
}

// Run은 위키 현황을 계산한다. 나이 계산 기준 시각은 호출자가 준다.
// 전역 --now 파싱 결과를 받는 구조라 여기서 시계를 직접 읽지 않는다.
func Run(wikiRoot string, now time.Time) (Result, error) {
	if _, err := os.Stat(filepath.Join(wikiRoot, config.ConfigFileName)); err != nil {
		return Result{}, errors.New(i18n.T("core.status.not_wiki", wikiRoot, wikiRoot))
	}
	cfg, err := config.Load(wikiRoot)
	if err != nil {
		return Result{}, err
	}

	// lint 요약은 lint.Run 을 그대로 쓴다. 판정 로직이 두 벌이 되면 곧 갈라진다.
	lintRes, err := lint.Run(wikiRoot, cfg)
	if err != nil {
		return Result{}, err
	}

	// 순회는 internal/walk 가 담당한다. 여기서 다시 읽거나 파싱하지 않는다.
	walked, err := walk.Files(wikiRoot, cfg)
	if err != nil {
		return Result{}, err
	}
	docs := make([]scannedDoc, 0, len(walked))
	for _, w := range walked {
		docs = append(docs, scannedDoc{rel: w.Rel, root: w.Root, parsed: w.Parsed, ok: w.Err == nil})
	}
	return buildResult(docs, walked, cfg, lintRes, now), nil
}

// buildResult는 파싱된 문서에서 지표와 제안을 계산한다.
// walked 는 순회 결과 전체로 게이트 유예의 대상 수를 잰다.
// lintRes 는 lint 실행 결과 전체다. 고아 수는 lint 의 graph.orphan 판정을
// 그대로 가져온다. 자체 판정을 갖으면 정의가 갈라진다.
func buildResult(docs []scannedDoc, walked []walk.Doc, cfg config.Config, lintRes lint.Result, now time.Time) Result {
	sum := lintRes.Summary
	res := Result{
		Orphans: lint.OrphanCount(lintRes),
		Lint:    LintSummary{Files: sum.Files, Error: sum.Error, Warn: sum.Warn, Reject: sum.Reject},
	}
	res.Backlog.PromotablePaths = []string{}
	res.Suggestions = []Suggestion{}

	// 링크 통계는 lint 의 graphRules 와 같은 세는 법을 쓴다.
	// 문서 안에서 중복 슬러그는 한 번으로 센다.
	outgoing := map[string]map[string]bool{}
	for _, s := range docs {
		if !s.ok {
			continue
		}
		set := map[string]bool{}
		for _, l := range allLinks(s.parsed) {
			set[l.Slug] = true
		}
		outgoing[s.rel] = set
		res.Links += len(set)
	}

	// 승급 대기 판정용으로 오래된 순 문서를 모은다.
	type aged struct {
		rel   string
		age   int
		known bool
	}
	var promotable []aged
	oldest := -1
	unknownAge := 0
	stale := 0

	for _, s := range docs {
		switch stageOfDir(s.rel) {
		case "inbox":
			res.Stages.Inbox++
		case "source":
			res.Stages.Source++
		case "context":
			res.Stages.Context++
		case "archive":
			res.Stages.Archive++
		}

		if !s.ok {
			continue
		}

		if stageOfDir(s.rel) != "inbox" && stageOfDir(s.rel) != "context" {
			continue
		}
		created, createdKnown := docDate(s, "created")
		if stageOfDir(s.rel) == "inbox" {
			// 나이는 프론트매터 created 를 쓰고 없으면 파일명 날짜 접두사를 쓴다.
			if !createdKnown {
				created, createdKnown = dateFromName(s.rel)
			}
			if !createdKnown {
				// 나이를 알 수 없는 문서는 따로 센다. 0 으로 치지 않는다.
				unknownAge++
			} else {
				if age := days(now, created); age > oldest {
					oldest = age
				}
			}
			// 승급 가능 여부는 lint 의 게이트 판정과 같은 함수와 같은
			// 대상 집계로 본다. promote 가 유예로 통과시키는 문서를 여기서
			// 못 넘는다고 말하면 status 가 거짓 미리보기가 된다. ADR 0021.
			targets := lint.LinkableSlugs(walked, s.rel, "")
			resolved := lint.ResolvedLinks(outgoing[s.rel], targets)
			if lint.EvaluateGate(resolved, len(targets), cfg.Thresholds.MinWikilinks).Passed {
				promotable = append(promotable, aged{rel: s.rel, age: ageOf(now, created, createdKnown), known: createdKnown})
			}
		} else {
			// context 문서의 노후는 updated 를 쓰고 없으면 created 를 쓴다.
			updated, ok := docDate(s, "updated")
			if !ok {
				updated, ok = created, createdKnown
			}
			if ok && days(now, updated) > cfg.Thresholds.StaleDays {
				stale++
			}
		}
	}

	res.Backlog.Inbox = res.Stages.Inbox
	res.Backlog.UnknownAge = unknownAge
	res.Backlog.Stale = stale
	if oldest >= 0 {
		res.Backlog.OldestDays = &oldest
	}

	// 오래된 것부터. 나이를 모르는 문서는 끝으로, 같으면 경로순으로.
	sort.Slice(promotable, func(i, j int) bool {
		if promotable[i].known != promotable[j].known {
			return promotable[i].known
		}
		if promotable[i].known && promotable[i].age != promotable[j].age {
			return promotable[i].age > promotable[j].age
		}
		return promotable[i].rel < promotable[j].rel
	})
	res.Backlog.Promotable = len(promotable)
	for _, p := range promotable {
		res.Backlog.PromotablePaths = append(res.Backlog.PromotablePaths, p.rel)
	}

	res.Suggestions = suggest(res, cfg)
	return res
}

// suggest는 지표에서 다음 행동 제안을 만든다. 최대 3개.
// 제안 우선순위는 고정이라 같은 지표면 같은 제안이 나온다.
func suggest(res Result, cfg config.Config) []Suggestion {
	var out []Suggestion
	// reject 가 가장 앞이다. 가장 심한 등급이고, context 에 있으면 안 될
	// 문서가 이미 올라가 있다는 뜻이다. 게이트가 유예된 채 승급한 문서가
	// 위키가 자란 뒤 이 상태가 된다. 여기서 안 내면 사용자가 lint 를
	// 따로 돌려야만 알게 된다(ADR 0091).
	if res.Lint.Reject > 0 {
		out = append(out, Suggestion{
			Action: "engram lint",
			Detail: i18n.T("core.status.suggest_reject",
				res.Lint.Reject, cfg.Thresholds.MinWikilinks),
		})
	}
	if res.Backlog.Promotable > 0 {
		out = append(out, Suggestion{
			Action: "engram promote " + res.Backlog.PromotablePaths[0],
			Detail: i18n.T("core.status.suggest_promotable",
				res.Backlog.Promotable, previewPaths(res.Backlog.PromotablePaths, 3)),
		})
	}
	if res.Backlog.Stale > 0 {
		out = append(out, Suggestion{
			Action: i18n.T("core.status.action_review"),
			Detail: i18n.T("core.status.suggest_stale",
				cfg.Thresholds.StaleDays, res.Backlog.Stale),
		})
	}
	if res.Lint.Error > 0 {
		out = append(out, Suggestion{
			Action: "engram lint",
			Detail: i18n.T("core.status.suggest_lint", res.Lint.Error),
		})
	}
	// inbox 에 밀린 것이 있는데 하나도 못 올리는 상태를 잡는다. 게이트가
	// 링크 부족으로 전부 막은 경우인데, 이 가지가 없으면 "제안 없음" 이
	// 나와서 사용자는 도구가 할 말이 없다고 읽는다.
	if res.Backlog.Inbox > 0 && res.Backlog.Promotable == 0 {
		out = append(out, Suggestion{
			Action: i18n.T("core.status.action_link"),
			Detail: i18n.T("core.status.suggest_blocked",
				res.Backlog.Inbox, cfg.Thresholds.MinWikilinks),
		})
	}
	if res.Backlog.Inbox == 0 {
		out = append(out, Suggestion{
			Action: "engram capture",
			Detail: i18n.T("core.status.suggest_capture"),
		})
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// previewPaths는 목록을 보기 좋게 줄인다. 상한을 넘으면 앞의 몇 개와 개수로 남긴다.
func previewPaths(paths []string, max int) string {
	if len(paths) <= max {
		return strings.Join(paths, ", ")
	}
	return strings.Join(paths[:max], ", ") + i18n.T("core.status.more_paths", len(paths)-max)
}

// stageOfDir는 문서 위치의 첫 디렉토리로 단계를 잡는다.
// 네 단계 외의 위치는 단계 집계에 넣지 않는다.
func stageOfDir(rel string) string {
	seg, _, _ := strings.Cut(rel, "/")
	switch seg {
	case "inbox", "sources", "context", "archive":
		if seg == "sources" {
			return "source"
		}
		return seg
	}
	return ""
}

// allLinks는 문서의 위키링크 전부(related 와 본문)를 합친다.
func allLinks(d doc.Doc) []doc.Link {
	return append(append([]doc.Link{}, d.FrontmatterLinks()...), d.BodyLinks()...)
}

// docDate는 문서의 날짜 필드를 파싱한다. created 는 연월(YYYY-MM)이면
// 그 달 1일로 본다.
func docDate(s scannedDoc, key string) (time.Time, bool) {
	for _, f := range s.parsed.Fields {
		if f.Key == key && f.Str != "" {
			if t, ok := parseDate(f.Str); ok {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// dateFromName은 파일명 날짜 접두사를 파싱한다. inbox 와 sources 문서는
// <날짜>-<슬러그>.md 규칙을 쓴다(internal/wiki 의 FilePath 계약).
func dateFromName(rel string) (time.Time, bool) {
	base := strings.TrimSuffix(filepath.Base(rel), ".md")
	if len(base) >= 11 && base[10] == '-' {
		if t, ok := parseDate(base[:10]); ok {
			return t, true
		}
	}
	if len(base) >= 8 && base[7] == '-' {
		if t, ok := parseDate(base[:7]); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseDate는 하루 단위와 연월 단위 두 형식을 파싱한다.
func parseDate(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// days는 기준 시각에서 날짜까지의 나이를 일수로 낸다.
func days(now, t time.Time) int {
	return int(now.Sub(t).Hours() / 24)
}

// ageOf는 나이를 계산한다. 날짜를 모르면 0 을 낸다(호출자가 known 으로 구분한다).
func ageOf(now time.Time, t time.Time, known bool) int {
	if !known {
		return 0
	}
	return days(now, t)
}
