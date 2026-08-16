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
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/wiki"
)

// Artifact는 내보내는 파일 하나다. Path 는 위키 루트 기준 슬래시 경로다.
type Artifact struct {
	Path    string
	Mode    os.FileMode
	Content string
}

// axisOrder는 축을 표기하는 고정 순서다. config 이 순서를 정의하지
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

// onAxes는 켜진 축 이름을 고정 순서로 낸다.
func onAxes(cfg config.Config) []string {
	var out []string
	for _, ax := range axisOrder() {
		if cfg.Axes[ax] {
			out = append(out, string(ax))
		}
	}
	return out
}

// offAxes는 꺼진 축 이름을 고정 순서로 낸다.
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
const excludedNote = `이 문서와 린터가 내보내지 않는 것:

- wiki.broad-topic. 위키 전체의 통계로 판정하는 진단이라 문서 하나를 보는
  훅의 자리가 아니다. engram lint 가 계속 판정한다.
- 검색 색인, 재발견, 링크 그래프 계산, 다이제스트. 연산은 파일로 표현되지
  않는다. engram search, recall, resurface, bridge, digest, backlinks 가
  계속 수행한다.`

// frontmatterSchemaDoc은 축과 단계별 필수 필드를 규정한다.
func frontmatterSchemaDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString("# 프론트매터 스키마\n\n")
	fmt.Fprintf(&b, "이 위키는 %s 프리셋을 쓴다. 프리셋은 축의 시작점이고 engram.yaml 의 axes 로 개별 축을 켜고 끌 수 있다.\n\n", cfg.Preset)
	b.WriteString("## 켜진 축\n\n")
	for _, ax := range onAxes(cfg) {
		b.WriteString("- " + ax + "\n")
	}
	if off := offAxes(cfg); len(off) > 0 {
		b.WriteString("\n## 꺼진 축\n\n꺼진 축의 필드를 문서에 두면 위반이다.\n\n")
		for _, ax := range off {
			b.WriteString("- " + ax + "\n")
		}
	}
	b.WriteString("\n## 단계별 필수 필드\n\n")
	for _, stage := range stageOrder() {
		fields := lint.RequiredFields(string(stage), cfg)
		fmt.Fprintf(&b, "- %s: %s\n", stage, strings.Join(fields, ", "))
	}
	b.WriteString(`
## 날짜 필드

- created. 원본이 처음 기록된 날. 사람이나 인테이크가 채운다. 연월까지만
  알면 YYYY-MM 을 허용한다.
- sourced_at. 이 위키에 편입된 날. 도구가 채운다.
- updated. 마지막으로 내용이 갱신된 날. 도구가 채운다. 손으로 쓰지 않는다.
- ` + "`sources/`" + ` 계층 문서에는 updated 를 두지 않는다. 원본 보존 계층이라
  갱신되지 않으며 신선도를 오해하게 만든다.

`)
	b.WriteString(excludedNote)
	b.WriteString("\n")
	return b.String()
}

// valueSetsDoc은 허용값 집합을 규정한다.
func valueSetsDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString("# 값 집합\n\n")
	b.WriteString("닫힌 집합(집합 밖 값은 위반)과 열린 집합(집합 밖 값은 경고)을 구분한다.\n\n")
	b.WriteString("## 닫힌 집합\n\n")
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
	fmt.Fprintf(&b, "\n## 열린 집합\n\n- topics: %s\n", strings.Join(cfg.Schema.Taxonomy.Topics.Values, ", "))
	b.WriteString("\ntopics 에 정의되지 않은 값을 쓰면 경고다. 값 자체는 허용된다.\n\n")
	b.WriteString(excludedNote)
	b.WriteString("\n")
	return b.String()
}

// promotionRulesDoc은 승급 게이트와 위치 규칙을 규정한다.
func promotionRulesDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString("# 승급 게이트와 위치 규칙\n\n")
	fmt.Fprintf(&b, "min_wikilinks 는 %d다. context 디렉토리 아래 문서의 고유 위키링크 수가 이 값에 못 미치면 게이트가 문서를 거절한다. 0이면 게이트가 꺼진다. 게이트는 선언이 아니라 문서가 놓인 디렉토리로 발동한다.\n\n", cfg.Thresholds.MinWikilinks)
	b.WriteString("거절 사유는 gate.min-wikilinks 하나뿐이다. 게이트를 통과하려면 관련 문서에 위키링크로 연결되어 있어야 한다.\n\n")
	b.WriteString("링크 가능한 대상은 문서 수가 충분해야 센다. 대상 문서가 min_wikilinks 보다 적으면 게이트를 유예하고 경고만 낸다. 위키가 자라면 게이트가 다시 동작한다. inbox 단계 문서는 대상에서 뺀다. promote 되면 슬러그가 바뀌어 링크가 깨지기 때문이다.\n\n")
	b.WriteString("## 위치와 단계의 일치\n\n문서가 놓인 최상위 디렉토리와 artifact_stage 값이 일치해야 한다.\n\n")
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, "- %s 단계 문서는 %s/ 디렉토리에 둔다\n", pair[0], pair[1])
	}
	b.WriteString("\ncontext 를 선언했는데 context 디렉토리 밖에 있으면 오류이다. 검수된 지식의 필드 집합을 갖추고 색인 자격을 주장하며 파이프라인을 우회하기 때문이다. 그 밖의 불일치는 경고다.\n\n")
	b.WriteString(excludedNote)
	b.WriteString("\n")
	return b.String()
}

// lintRulesDoc은 규칙 목록 전체를 규정한다. 목록은 lint.Rules 에서 온다.
func lintRulesDoc(cfg config.Config) string {
	var b strings.Builder
	b.WriteString("# lint 규칙\n\n규칙 ID 는 점 표기 소문자다. 등급의 의미는 아래와 같다.\n\n")
	b.WriteString("- error: 승급을 막는다\n- warn: 통과시키되 알린다\n- reject: 승급 게이트 거절\n\n")
	b.WriteString("## 규칙 목록\n\n")
	b.WriteString("| 규칙 ID | 등급 | 판정 |\n|---|---|---|\n")
	for _, r := range lint.Rules() {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.ID, r.Severity, r.Desc)
	}
	b.WriteString("\n## scripts/lint-frontmatter.py 가 판정하는 규칙\n\n")
	b.WriteString("문서 단위 규칙과 링크 무결성, 고아 판정, 승급 게이트를 판정한다. 위 표에서 wiki.broad-topic 만 빠진다. 위키 전체의 통계로 판정하는 진단이라 문서 하나를 보는 훅의 자리가 아니다.\n\n")
	b.WriteString(excludedNote)
	b.WriteString("\n")
	return b.String()
}

// wikiLayoutDoc은 디렉토리 구조를 규정한다.
func wikiLayoutDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString("# 위키 배치\n\n")
	b.WriteString("## 단계 디렉토리\n\n")
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, "- %s/ (%s 단계)\n", pair[1], pair[0])
	}
	fmt.Fprintf(&b, "\n## 루트 파일\n\nroot_files 로 정의한다: %s\n", strings.Join(cfg.RootFiles, ", "))
	b.WriteString("\n루트 파일은 색인이다. 승급 게이트와 고아 판정에서 빠지고 스키마 검사는 그대로 받는다.\n\n")
	fmt.Fprintf(&b, "## 문서가 아닌 파일\n\nignore_files 로 정의한다: %s\n", strings.Join(cfg.IgnoreFiles, ", "))
	b.WriteString("\n같은 파일명이면 깊이와 무관하게 문서에서 뺀다. 디렉토리를 설명하는 README 같은 파일이 해당한다.\n\n")
	b.WriteString(excludedNote)
	b.WriteString("\n")
	return b.String()
}
