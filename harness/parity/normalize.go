// Package parity는 upstream llm-wiki 스크립트와 engram lint 의 위반 목록을
// 공통 형태로 정규화해 비교한다. ADR 0029 가 정한 비교 축 중 lint 위반
// 목록 하나를 담당한다.
//
// 정규화는 차이를 감추지 않아야 한다. 규칙 개념이 필드 단위로 쪼개지는
// 규칙(필수 필드, 허용값)은 필드 이름까지 규칙 이름에 붙여, 한쪽만 잡은
// 위반이 같은 규칙으로 보이는 일을 막는다. 매핑 표에 없는 이름은
// "unmapped:" 접두사와 함께 결과에 그대로 남는다. 조용히 버리지 않는다.
package parity

import (
	"regexp"
	"strings"
)

// Violation은 비교의 단위다. 줄 번호는 넣지 않는다. 두 구현이 같은 줄을
// 가리키리라는 보장이 없기 때문이다.
type Violation struct {
	Path string // 위키 루트 기준 슬래시 경로
	Rule string // 정규화한 규칙 이름
}

// upstreamConstantRules는 upstream scripts/lint-frontmatter.sh 의
// record_failure, record_warning 메시지 접두사를 정규화 규칙 이름으로
// 옮기는 표다. 완전 일치가 아니라 접두사로 맞춘다. 메시지 뒤에 안내 문구가
// 붙는 규칙이 있기 때문이다. 처음 일치에서 끝내므로 한 항목이 다른 항목의
// 접두사가 되지 않게 관리한다.
var upstreamConstantRules = []struct{ prefix, rule string }{
	{"missing frontmatter block at top of file", "frontmatter.missing"},
	{"empty or malformed frontmatter block", "frontmatter.malformed"},
	{"quality_level is retired", "schema.retired-field:quality_level"},
	{"review_after is retired", "schema.retired-field:review_after"},
	{"source artifact requires created", "frontmatter.missing-field:created"},
	{"source artifact requires sourced_at", "frontmatter.missing-field:sourced_at"},
	{"source artifact must not carry updated", "sources.updated"},
	{"source artifact missing derived_context", "frontmatter.missing-field:derived_context"},
	{"context artifact missing title", "frontmatter.missing-field:title"},
	{"context artifact missing source_refs", "frontmatter.missing-field:source_refs"},
	{"context artifact missing related", "frontmatter.missing-field:related"},
	{"agent-workflow artifact missing source_refs", "frontmatter.missing-field:source_refs"},
	{"agent-workflow artifact missing related", "frontmatter.missing-field:related"},
	{"index artifact missing source_refs", "frontmatter.missing-field:source_refs"},
	{"index artifact missing related", "frontmatter.missing-field:related"},
	{"context/mocs documents must use artifact_stage index", "location.stage-agreement"},
	{"context/mocs documents must be type moc", "location.type-agreement"},
	{"context documents must use artifact_stage context", "location.stage-agreement"},
	{"agent workflows must use artifact_stage agent-workflow", "location.stage-agreement"},
	{"source artifacts must use artifact_stage source", "location.stage-agreement"},
	{"inbox documents must use artifact_stage inbox", "location.stage-agreement"},
	{"root index must be type moc", "location.type-agreement"},
	{"root index must use artifact_stage index", "location.stage-agreement"},
	{"inbox artifacts must be indexable false", "indexable.policy"},
	{"source artifacts are normally indexable false", "indexable.policy"},
	{"context artifacts are normally indexable true", "indexable.policy"},
}

// upstreamRule은 upstream 메시지 하나를 정규화 규칙 이름으로 옮긴다.
// 필드 이름이 메시지 안에 있는 두 규칙(필수 필드 누락, 허용값 밖 값)은
// 상수표 대신 필드를 꺼내 붙인다. 어느 쪽에도 속하지 않으면 unmapped 로
// 남긴다.
func upstreamRule(msg string) string {
	for _, m := range upstreamConstantRules {
		if strings.HasPrefix(msg, m.prefix) {
			return m.rule
		}
	}
	if rest, ok := strings.CutPrefix(msg, "missing required field '"); ok {
		if field, _, found := strings.Cut(rest, "'"); found {
			return "frontmatter.missing-field:" + field
		}
	}
	if rest, ok := strings.CutPrefix(msg, "invalid "); ok {
		if field, _, found := strings.Cut(rest, " '"); found {
			return "schema.allowed-value:" + field
		}
	}
	return "unmapped:" + msg
}

// upstreamLine은 lint-frontmatter.sh 출력의 위반 줄 형식이다. 경로에 콜론이
// 없다는 가정을 둔다. 골든 위키 픽스처가 그 조건을 지킨다.
var upstreamLine = regexp.MustCompile(`^(FAIL|WARN) ([^:]+): (.*)$`)

// ParseUpstreamOutput은 lint-frontmatter.sh 의 표준 출력에서 위반 줄을 꺼내
// 정규화한다. 요약 줄과 빈 줄은 무시한다. 등급(FAIL, WARN)은 비교 축에서
// 빼므로 여기서 버린다.
func ParseUpstreamOutput(out string) []Violation {
	var vs []Violation
	for _, line := range strings.Split(out, "\n") {
		m := upstreamLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		vs = append(vs, Violation{Path: m[2], Rule: upstreamRule(m[3])})
	}
	return vs
}

// engramFieldPatterns는 engram lint 위반 메시지에서 규칙 이름에 붙일 필드
// 이름을 꺼내는 정규식이다. 규칙 ID 혼자는 필드 구분을 못 하므로 메시지에서
// 꺼낸다. 메시지 형식은 harness/golden 스냅샷이 지키고 있다.
var (
	engramMissingField = regexp.MustCompile(`필수 필드 ([a-z_]+)가 없습니다`)
	engramAllowedValue = regexp.MustCompile(`^([a-z_]+) 값이 허용값 밖입니다`)
	engramAxisOff      = regexp.MustCompile(`필드가 문서에 있습니다: ([a-z_]+)`)
)

// NormalizeEngram은 engram lint 위반 하나를 정규화한다. rule 은 lint 의
// 규칙 ID, message 는 위반 메시지다. 필드 이름을 못 꺼내면 unmapped 로
// 남긴다.
func NormalizeEngram(rule, message string) string {
	switch rule {
	case "frontmatter.missing", "frontmatter.unclosed", "frontmatter.yaml",
		"taxonomy.forms", "taxonomy.topics", "sources.updated", "body.max-lines",
		"link.broken", "graph.orphan", "gate.deferred", "gate.min-wikilinks":
		return rule
	case "frontmatter.missing-field":
		if m := engramMissingField.FindStringSubmatch(message); m != nil {
			return "frontmatter.missing-field:" + m[1]
		}
	case "schema.allowed-value":
		if m := engramAllowedValue.FindStringSubmatch(message); m != nil {
			return "schema.allowed-value:" + m[1]
		}
	case "schema.axis-off":
		if m := engramAxisOff.FindStringSubmatch(message); m != nil {
			return "schema.axis-off:" + m[1]
		}
	}
	return "unmapped:" + rule
}
