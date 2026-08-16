package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// rulesThresholds는 임계값 넷과 각각이 무엇을 판정하는지다.
type rulesThresholds struct {
	MinWikilinks  int `json:"minWikilinks"`
	StaleDays     int `json:"staleDays"`
	MaxLines      int `json:"maxLines"`
	BroadTopicPct int `json:"broadTopicPct"`
}

// rulesGate는 승급 게이트의 요약이다. 거절 사유가 하나뿐이라는 사실이
// 이 커맨드 출력에서 가장 중요한 줄이다.
type rulesGate struct {
	OnlyRejectRule string `json:"onlyRejectRule"`
	MinWikilinks   int    `json:"minWikilinks"`
}

// rulesDirs는 문서가 놓이는 곳의 규칙이다.
type rulesDirs struct {
	PageDirs    []string `json:"pageDirs"`
	RootFiles   []string `json:"rootFiles"`
	IgnoreFiles []string `json:"ignoreFiles"`
}

// rulesReport는 rules show가 내는 보고 전부다. --json 출력에 그대로 쓰인다.
type rulesReport struct {
	Preset         string              `json:"preset"`
	Axes           map[string]bool     `json:"axes"`
	RequiredFields map[string][]string `json:"requiredFields"`
	ClosedSets     map[string][]string `json:"closedSets"`
	OpenSets       map[string][]string `json:"openSets"`
	Thresholds     rulesThresholds     `json:"thresholds"`
	Gate           rulesGate           `json:"gate"`
	Rules          []lint.Rule         `json:"rules"`
	Dirs           rulesDirs           `json:"dirs"`
}

// displayAxes는 축을 표기 순서에 따라 낸다. 순서는 config 의 축 상수
// 순서를 따르며 맵 순회에 의존하지 않는다.
func displayAxes() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// newRulesCmd는 이 위키에 적용되는 규칙을 다루는 rules 커맨드를 반환한다.
// 하위 커맨드 없이 치면 사용법을 낸다.
func newRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "이 위키에 적용되는 규칙을 다룹니다",
		Long: `이 위키에 지금 적용되는 규칙을 다룹니다.

rules show가 규칙 전부를 읽기 전용으로 보여줍니다. eject 없이
규칙을 확인하려는 경우를 위한 커맨드입니다.`,
	}
	cmd.AddCommand(newRulesShowCmd())
	return cmd
}

// newRulesShowCmd는 규칙 전부를 읽기 전용으로 내는 show 커맨드를 반환한다.
// 강의의 클라이맥스에서 사람이 읽는 출력이므로 덤프가 아니라 읽히는
// 글이어야 한다. ADR 0013.
func newRulesShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "이 위키에 적용되는 규칙 전부를 보여줍니다",
		Long: `이 위키에 지금 적용되는 규칙 전부를 보여줍니다.

프리셋과 축, 단계별 필수 필드, 허용값, 임계값, 승급 게이트, lint 규칙,
디렉토리를 절로 나눠 보여줍니다. 위키의 engram.yaml을 읽어 프리셋과
사용자 설정이 합쳐진 결과를 냅니다. 제품 기본값이 아니라 이 위키의
값입니다.

무엇도 바꾸지 않습니다. 규칙은 읽는 것입니다.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			res := buildRulesReport(cfg)
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printRules(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// buildRulesReport는 설정에서 규칙 보고를 만든다. 어느 값도 여기서
// 정하지 않고 전부 설정에서 읽는다.
func buildRulesReport(cfg config.Config) rulesReport {
	axes := make(map[string]bool, len(displayAxes()))
	for _, ax := range displayAxes() {
		axes[string(ax)] = cfg.Axes[ax]
	}
	stages := []string{"inbox", "source", "context", "archive"}
	required := make(map[string][]string, len(stages))
	for _, st := range stages {
		required[st] = lint.RequiredFields(st, cfg)
	}
	res := rulesReport{
		Preset:         string(cfg.Preset),
		Axes:           axes,
		RequiredFields: required,
		ClosedSets: map[string][]string{
			"type":         cfg.Schema.Types,
			"status":       cfg.Schema.Statuses.Values,
			"scope":        cfg.Schema.Scopes.Values,
			"sensitivity":  cfg.Schema.Sensitivities.Values,
			"trigger_mode": cfg.Schema.TriggerModes.Values,
			"form":         cfg.Schema.Taxonomy.Forms.Values,
		},
		OpenSets: map[string][]string{
			"topics": cfg.Schema.Taxonomy.Topics.Values,
		},
		Thresholds: rulesThresholds{
			MinWikilinks:  cfg.Thresholds.MinWikilinks,
			StaleDays:     cfg.Thresholds.StaleDays,
			MaxLines:      cfg.Thresholds.MaxLines,
			BroadTopicPct: cfg.Thresholds.BroadTopicPct,
		},
		Gate:  rulesGate{OnlyRejectRule: "gate.min-wikilinks", MinWikilinks: cfg.Thresholds.MinWikilinks},
		Rules: lint.Rules(),
		Dirs: rulesDirs{
			PageDirs:    append([]string(nil), cfg.PageDirs...),
			RootFiles:   append([]string(nil), cfg.RootFiles...),
			IgnoreFiles: append([]string(nil), cfg.IgnoreFiles...),
		},
	}
	return res
}

// printRules는 사람용 보고를 인쇄한다. 절을 나누고 정렬해 터미널에서
// 읽히게 한다. 마크다운 표 기호는 쓰지 않는다.
func printRules(w io.Writer, res rulesReport) {
	fmt.Fprintf(w, "이 위키의 규칙 (프리셋: %s)\n", res.Preset)
	fmt.Fprintf(w, "프리셋과 engram.yaml 설정이 합쳐진 결과입니다. 제품 기본값이 아니라 이 위키의 값입니다.\n")

	axes := displayAxes()
	on, off := []string{}, []string{}
	for _, ax := range axes {
		if res.Axes[string(ax)] {
			on = append(on, string(ax))
		} else {
			off = append(off, string(ax))
		}
	}
	fmt.Fprintf(w, "\n축 %d종 (켜짐 %d, 꺼짐 %d)\n", len(axes), len(on), len(off))
	fmt.Fprintf(w, "  켜짐  %s\n", strings.Join(on, ", "))
	if len(off) > 0 {
		fmt.Fprintf(w, "  꺼짐  %s\n", strings.Join(off, ", "))
		fmt.Fprintf(w, "  꺼진 축의 필드는 필수가 아니고, 문서에 있으면 schema.axis-off가 잡습니다\n")
	}

	fmt.Fprintf(w, "\n단계별 필수 필드\n")
	stageWidth := 0
	for _, st := range []string{"inbox", "source", "context", "archive"} {
		if w := displayWidth(st); w > stageWidth {
			stageWidth = w
		}
	}
	for _, st := range []string{"inbox", "source", "context", "archive"} {
		fmt.Fprintf(w, "  %s  %s\n", padRight(st, stageWidth), strings.Join(res.RequiredFields[st], ", "))
	}

	fmt.Fprintf(w, "\n허용값. 폐쇄 집합은 정의되지 않은 값이 오류입니다\n")
	printValueSets(w, res.ClosedSets, "정의되지 않아 이 위키는 값을 검사하지 않습니다")
	fmt.Fprintf(w, "\n허용값. 개방 집합은 정의되지 않은 값이 경고입니다\n")
	printValueSets(w, res.OpenSets, "정의되지 않아 모든 값이 경고 대상입니다")

	fmt.Fprintf(w, "\n임계값\n")
	t := res.Thresholds
	thresholdRows := []struct {
		name string
		val  string
		desc string
	}{
		{"min_wikilinks", fmt.Sprintf("%d", t.MinWikilinks), "승급 게이트의 거절 기준입니다"},
		{"stale_days", fmt.Sprintf("%d", t.StaleDays), "resurface가 다시 꺼낼 문서를 고르는 기준 일수입니다"},
		{"max_lines", fmt.Sprintf("%d", t.MaxLines), "body.max-lines 경고의 상한 줄 수입니다"},
		{"broad_topic_pct", fmt.Sprintf("%d", t.BroadTopicPct), "wiki.broad-topic 진단의 상한 비율입니다"},
	}
	nameWidth, valWidth := 0, 0
	for _, r := range thresholdRows {
		if w := displayWidth(r.name); w > nameWidth {
			nameWidth = w
		}
		if w := displayWidth(r.val); w > valWidth {
			valWidth = w
		}
	}
	for _, r := range thresholdRows {
		fmt.Fprintf(w, "  %s %s  %s\n", padRight(r.name, nameWidth), padRight(r.val, valWidth), r.desc)
	}

	fmt.Fprintf(w, "\n승급 게이트\n")
	fmt.Fprintf(w, "  게이트가 승급을 거절하는 조건은 하나뿐입니다.\n")
	fmt.Fprintf(w, "  문서의 고유 위키링크 수가 min_wikilinks %d개에 못 미치는 것입니다.\n", res.Gate.MinWikilinks)
	fmt.Fprintf(w, "  promote와 new는 이것만 묻습니다. 길이도 형식도 주제도 막지 않습니다.\n")

	fmt.Fprintf(w, "\nlint 규칙 %d종\n", len(res.Rules))
	idWidth, sevWidth := 0, 0
	for _, r := range res.Rules {
		if w := displayWidth(r.ID); w > idWidth {
			idWidth = w
		}
		if w := displayWidth(r.Severity); w > sevWidth {
			sevWidth = w
		}
	}
	for _, r := range res.Rules {
		fmt.Fprintf(w, "  [%s] %s  %s\n", padRight(r.Severity, sevWidth), padRight(r.ID, idWidth), r.Desc)
	}
	fmt.Fprintf(w, "  등급에서 error와 reject는 lint를 종료 코드 1로 끝냅니다. warn은 알리고 통과시킵니다.\n")
	fmt.Fprintf(w, "  승급 시점에 문서를 거절하는 것은 reject 하나뿐입니다.\n")

	fmt.Fprintf(w, "\n디렉토리\n")
	fmt.Fprintf(w, "  page_dirs    %s\n", strings.Join(res.Dirs.PageDirs, ", "))
	fmt.Fprintf(w, "  root_files   %s\n", strings.Join(res.Dirs.RootFiles, ", "))
	fmt.Fprintf(w, "  ignore_files %s\n", strings.Join(res.Dirs.IgnoreFiles, ", "))
}

// printValueSets는 허용값 집합을 이름 폭에 맞춰 정렬해 인쇄한다.
// 맵을 정렬 없이 순회하면 출력이 흔들리므로 키를 정렬한다. 빈 집합은
// 공백 대신 emptyNote 로 뜻을 밝힌다. 폐쇄 집합이 비면 검사 자체가
// 일어나지 않고 개방 집합이 비면 모든 값이 경고 대상이 되어 뜻이 갈린다.
func printValueSets(w io.Writer, sets map[string][]string, emptyNote string) {
	keys := make([]string, 0, len(sets))
	for k := range sets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	width := 0
	for _, k := range keys {
		if w := displayWidth(k); w > width {
			width = w
		}
	}
	for _, k := range keys {
		vals := strings.Join(sets[k], ", ")
		if vals == "" {
			vals = "(" + emptyNote + ")"
		}
		fmt.Fprintf(w, "  %s  %s\n", padRight(k, width), vals)
	}
}
