// Package digest는 기간 안의 위키 변화를 집계한다. 상태를 남기지
// 않으므로 같은 위키에 같은 기준 시각을 주면 몇 번을 돌려도 같은 결과가
// 나온다. ADR 0028.
package digest

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/resurface"
	"github.com/neocode24/engram/internal/walk"
)

// Result는 digest 집계 결과다. 슬러그 목록은 항목별로 정렬되어 있다.
// 승급 집계는 없다. promote 가 승급 시각을 프론트매터에 남기지 않으므로
// 시각이 없는 집계는 지어낼 수밖에 없어 뺐다.
type Result struct {
	Days      int      `json:"days"`
	Since     string   `json:"since"`
	Until     string   `json:"until"`
	StaleDays int      `json:"staleDays"`
	Created   []string `json:"created"`
	Stale     []string `json:"stale"`
	Orphans   []string `json:"orphans"`
}

// Run은 기간 안의 변화를 집계한다. 창은 [now - days일, now] 다.
// 노후 판정은 resurface.IsStale 이 단일 진실원이므로 두 커맨드가 같은
// 문서를 노후라고 한다. 고아 판정은 lint 의 graph.orphan 규칙에서
// 가져온다. 판정이 두 벌이 되면 커맨드끼리 다른 수를 말하게 된다.
func Run(wikiRoot string, cfg config.Config, now time.Time, days int) (Result, error) {
	since := now.AddDate(0, 0, -days)
	res := Result{
		Days:      days,
		Since:     since.Format(time.RFC3339),
		Until:     now.Format(time.RFC3339),
		StaleDays: cfg.Thresholds.StaleDays,
		Created:   []string{},
		Stale:     []string{},
		Orphans:   []string{},
	}

	walked, err := walk.Files(wikiRoot, cfg)
	if err != nil {
		return res, fmt.Errorf("위키를 순회할 수 없음: %w", err)
	}
	for _, w := range walked {
		if w.Err != nil || !w.Parsed.HasFrontmatter {
			continue
		}
		slug := slugOf(w.Rel)
		// 신규는 모든 단계에서 창 안에 만들어진 문서다.
		if t, ok := dateField(w.Parsed, "created"); ok && !t.Before(since) && !t.After(now) {
			res.Created = append(res.Created, slug)
		}
		if stageOfDir(w.Rel) == "context" && resurface.IsStale(w.Parsed, now, cfg.Thresholds.StaleDays) {
			res.Stale = append(res.Stale, slug)
		}
	}

	// 고아는 lint 결과의 graph.orphan 위반 그대로다. 여기서 링크를 다시
	// 세면 lint 와 digest 가 다른 수를 말하게 된다. lint.OrphanCount 도
	// 같은 위반에서 수를 낸다.
	lintRes, err := lint.Run(wikiRoot, cfg)
	if err != nil {
		return res, fmt.Errorf("lint 를 실행할 수 없음: %w", err)
	}
	for _, v := range lintRes.Violations {
		if v.Rule == "graph.orphan" {
			res.Orphans = append(res.Orphans, slugOf(v.Path))
		}
	}

	sort.Strings(res.Created)
	sort.Strings(res.Stale)
	sort.Strings(res.Orphans)
	return res, nil
}

// stageOfDir는 문서 위치의 첫 디렉토리로 단계를 잡는다. resurface 도
// 같은 세는 법을 쓰므로 노후 대상이 어긋나지 않는다.
func stageOfDir(rel string) string {
	seg, _, _ := strings.Cut(rel, "/")
	return seg
}

// slugOf는 문서 경로에서 슬러그를 낸다. lint 의 bySlug 와 같은 법이다.
func slugOf(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), ".md")
}

// dateField는 문서의 날짜 필드를 파싱한다. 하루 단위와 연월 단위 두
// 형식을 받는다. 노후 판정의 기준 날짜는 resurface.BaseDate 가 있으므로
// 여기는 신규 집계의 created 용도만 담는다.
func dateField(d doc.Doc, key string) (time.Time, bool) {
	for _, f := range d.Fields {
		if f.Key == key && f.Str != "" {
			for _, layout := range []string{"2006-01-02", "2006-01"} {
				if t, err := time.Parse(layout, f.Str); err == nil {
					return t, true
				}
			}
		}
	}
	return time.Time{}, false
}
