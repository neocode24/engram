// Package eject는 사용자 위키로 내보낼 규칙 산출물을 만든다. 규칙 명세
// 문서, 문서 단위 규칙을 판정하는 Python 린터, 훅, 에이전트 계약 문서다.
// 판정값은 어디까지나 생성 시점의 config 와 lint 의 규칙 목록에서 온다.
// 이 패키지는 파일을 쓰지 않고 계획만 만든다. 쓰기와 충돌 판정은
// 커맨드 계층이 한다.
package eject

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/wiki"
)

// Artifact는 내보내는 파일 하나다. Path 는 위키 루트 기준 슬래시 경로다.
type Artifact struct {
	Path    string
	Mode    os.FileMode
	Content string
}

// axisOrder는 속성을 표기하는 고정 순서다. config 이 순서를 정의하지
// 않으므로 표기 순서만 여기서 정한다. 맵 순회에 의존하지 않기 위해서다.
func axisOrder() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// stageOrder는 단계를 표기하는 고정 순서다.
func stageOrder() []wiki.Stage {
	return []wiki.Stage{wiki.StageInbox, wiki.StageSource, wiki.StageContext, wiki.StageArchive}
}

// stageDirTable는 단계와 디렉토리의 대응을 wiki 에서 읽는다. 대응의
// 진실원이 wiki 이므로 여기에 표를 두지 않는다. page_dirs 에서 빠진
// 단계는 StageForDir 로 대응을 되짚는다.
func stageDirTable(cfg config.Config) map[string]string {
	out := map[string]string{}
	for _, stage := range stageOrder() {
		if d, err := wiki.DirFor(cfg, stage); err == nil {
			out[string(stage)] = d
			continue
		}
		for _, cand := range []string{string(stage), "sources"} {
			if st, ok := wiki.StageForDir(cand); ok && st == stage {
				out[string(stage)] = cand
				break
			}
		}
	}
	return out
}

// Plan은 위키 설정을 반영해 산출물 전부를 만든다. 같은 설정에 대해
// 항상 같은 순서로 같은 내용을 낸다. 맵을 순회하는 자리는 전부 정렬한다.
func Plan(cfg config.Config) []Artifact {
	dirs := stageDirTable(cfg)
	artifacts := []Artifact{
		{Path: "meta/frontmatter-schema.md", Mode: 0o644, Content: frontmatterSchemaDoc(cfg)},
		{Path: "meta/value-sets.md", Mode: 0o644, Content: valueSetsDoc(cfg)},
		{Path: "meta/promotion-rules.md", Mode: 0o644, Content: promotionRulesDoc(cfg, dirs)},
		{Path: "meta/lint-rules.md", Mode: 0o644, Content: lintRulesDoc(cfg)},
		{Path: "meta/wiki-layout.md", Mode: 0o644, Content: wikiLayoutDoc(cfg, dirs)},
		{Path: "meta/agent-contract.md", Mode: 0o644, Content: agentContractDoc()},
		{Path: "scripts/lint-frontmatter.py", Mode: 0o755, Content: linterScript(cfg, dirs)},
		{Path: ".githooks/pre-commit", Mode: 0o755, Content: preCommitHook()},
		{Path: "AGENTS.md", Mode: 0o644, Content: agentsDoc(cfg, dirs)},
		{Path: ".gitattributes", Mode: 0o644, Content: gitattributes()},
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return artifacts
}

// pyList는 문자열 목록을 Python 리스트 리터럴로 낸다.
func pyList(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Quote(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyInt는 정수를 Python 리터럴로 낸다.
func pyInt(n int) string {
	return strconv.Itoa(n)
}

// onAxes는 켜진 속성 이름을 고정 순서로 낸다.
func onAxes(cfg config.Config) []string {
	var out []string
	for _, ax := range axisOrder() {
		if cfg.Axes[ax] {
			out = append(out, string(ax))
		}
	}
	return out
}

// offAxes는 꺼진 속성 이름을 고정 순서로 낸다.
func offAxes(cfg config.Config) []string {
	var out []string
	for _, ax := range axisOrder() {
		if !cfg.Axes[ax] {
			out = append(out, string(ax))
		}
	}
	return out
}

// stageDirsSorted는 단계를 고정 순서로 (단계, 디렉토리) 쌍으로 낸다.
func stageDirsSorted(dirs map[string]string) [][2]string {
	out := make([][2]string, 0, len(dirs))
	for _, stage := range stageOrder() {
		if d, ok := dirs[string(stage)]; ok {
			out = append(out, [2]string{string(stage), d})
		}
	}
	return out
}

// excludedNote는 내보내지 않는 것과 그 이유다. 문서마다 같은 문구를
// 쓴다. 사용자가 왜 이 규칙은 없는지 헤매지 않게 하기 위해서다.
// 문구는 카탈로그에 있다. 산출물이 내보내는 시점 언어로 굳으므로(ADR 0049).
func excludedNote() string {
	return i18n.T("eject.excluded_note")
}

// frontmatterSchemaDoc은 속성과 단계별 필수 필드를 규정한다.
func frontmatterSchemaDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.schema_doc.heading"))
	b.WriteString(i18n.T("eject.schema_doc.intro", cfg.Preset))
	b.WriteString(i18n.T("eject.schema_doc.on_heading"))
	for _, ax := range onAxes(cfg) {
		b.WriteString("- " + ax + "\n")
	}
	if off := offAxes(cfg); len(off) > 0 {
		b.WriteString(i18n.T("eject.schema_doc.off_heading"))
		for _, ax := range off {
			b.WriteString("- " + ax + "\n")
		}
	}
	b.WriteString(i18n.T("eject.schema_doc.required_heading"))
	for _, stage := range stageOrder() {
		fields := lint.RequiredFields(string(stage), cfg)
		fmt.Fprintf(&b, "- %s: %s\n", stage, strings.Join(fields, ", "))
	}
	b.WriteString(i18n.T("eject.schema_doc.dates"))
	b.WriteString(excludedNote())
	b.WriteString("\n")
	return b.String()
}

// valueSetsDoc은 허용값 집합을 규정한다.
func valueSetsDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.values_doc.heading"))
	b.WriteString(i18n.T("eject.values_doc.intro"))
	b.WriteString(i18n.T("eject.values_doc.closed_heading"))
	fmt.Fprintf(&b, "- artifact_stage: %s\n", strings.Join(cfg.Schema.ArtifactStages.Values, ", "))
	fmt.Fprintf(&b, "- status: %s\n", strings.Join(cfg.Schema.Statuses.Values, ", "))
	fmt.Fprintf(&b, "- type: %s\n", strings.Join(cfg.Schema.Types, ", "))
	fmt.Fprintf(&b, "- form: %s\n", strings.Join(cfg.Schema.Taxonomy.Forms.Values, ", "))
	if cfg.Axes[config.AxisScope] {
		fmt.Fprintf(&b, "- scope: %s\n", strings.Join(cfg.Schema.Scopes.Values, ", "))
	}
	if cfg.Axes[config.AxisSensitivity] {
		fmt.Fprintf(&b, "- sensitivity: %s\n", strings.Join(cfg.Schema.Sensitivities.Values, ", "))
	}
	if cfg.Axes[config.AxisTriggerMode] {
		fmt.Fprintf(&b, "- trigger_mode: %s\n", strings.Join(cfg.Schema.TriggerModes.Values, ", "))
	}
	fmt.Fprintf(&b, i18n.T("eject.values_doc.open_heading"), strings.Join(cfg.Schema.Taxonomy.Topics.Values, ", "))
	b.WriteString(i18n.T("eject.values_doc.open_note"))
	b.WriteString(excludedNote())
	b.WriteString("\n")
	return b.String()
}

// promotionRulesDoc은 승급 게이트와 위치 규칙을 규정한다.
func promotionRulesDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.promotion_doc.heading"))
	b.WriteString(i18n.T("eject.promotion_doc.intro", cfg.Thresholds.MinWikilinks))
	b.WriteString(i18n.T("eject.promotion_doc.sole_reason"))
	b.WriteString(i18n.T("eject.promotion_doc.deferred"))
	b.WriteString(i18n.T("eject.promotion_doc.location_heading"))
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, i18n.T("eject.promotion_doc.location_row"), pair[0], pair[1])
	}
	b.WriteString(i18n.T("eject.promotion_doc.context_note"))
	b.WriteString(excludedNote())
	b.WriteString("\n")
	return b.String()
}

// lintRulesDoc은 규칙 목록 전체를 규정한다. 목록은 lint.Rules 에서 온다.
func lintRulesDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.lint_rules_doc.heading"))
	b.WriteString(i18n.T("eject.lint_rules_doc.severities"))
	b.WriteString(i18n.T("eject.lint_rules_doc.table_heading"))
	b.WriteString(i18n.T("eject.lint_rules_doc.table_header"))
	for _, r := range lint.Rules() {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.ID, r.Severity(), r.Desc())
	}
	b.WriteString(i18n.T("eject.lint_rules_doc.script_heading"))
	b.WriteString(i18n.T("eject.lint_rules_doc.script_note"))
	b.WriteString(excludedNote())
	b.WriteString("\n")
	return b.String()
}

// wikiLayoutDoc은 디렉토리 구조를 규정한다.
func wikiLayoutDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.layout_doc.heading"))
	b.WriteString(i18n.T("eject.layout_doc.stage_heading"))
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, i18n.T("eject.layout_doc.stage_row"), pair[1], pair[0])
	}
	fmt.Fprintf(&b, i18n.T("eject.layout_doc.root_files"), strings.Join(cfg.RootFiles, ", "))
	b.WriteString(i18n.T("eject.layout_doc.root_note"))
	fmt.Fprintf(&b, i18n.T("eject.layout_doc.ignore_files"), strings.Join(cfg.IgnoreFiles, ", "))
	b.WriteString(i18n.T("eject.layout_doc.ignore_note"))
	b.WriteString(excludedNote())
	b.WriteString("\n")
	return b.String()
}
