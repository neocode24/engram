// Package lint는 위키를 순회하며 스키마와 링크 무결성을 검사한다.
//
// 규칙 ID는 점 표기 소문자이고 ADR 0005의 parity 비교 키가 되므로
// 한 번 정하면 바꾸지 않는다는 전제로 지었다.
package lint

import (
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
)

// Severity는 위반의 등급이다.
type Severity string

const (
	SevError  Severity = "error"  // 승급을 막는다
	SevWarn   Severity = "warn"   // 통과시키되 알린다
	SevReject Severity = "reject" // 게이트 거절
)

// Rule은 lint 규칙 하나의 메타데이터다. ID 는 동등성 검증의 정규화
// 표가 짝짓는 문자열 그대로다.
type Rule struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // error, warn, reject. 조건에 따라 갈리면 그 사실을 적는다
	Desc     string `json:"desc"`     // 무엇을 판정하는지 한 줄
}

// registry는 정의된 규칙 전부다. newRule 이 정의 시점에 채운다.
// 선언 순서가 곧 Rules 의 나열 순서다.
var registry []Rule

// ruleIndex는 등록된 규칙 ID 의 색인이다. newViolation 이 등록 여부를
// 확인하는 데 쓴다.
var ruleIndex = map[string]bool{}

// newRule는 규칙을 정의하고 등록한다. 규칙은 이 함수로만 만들며
// 위반을 만드는 자리는 여기서 만든 Rule 값만 받는다. 정의와 등록이
// 한몸이므로 메타데이터 없는 규칙이 존재할 수 없고 목록이 규칙과
// 어긋날 수 없다. 소스 텍스트를 긁어 대조하는 방식은 새 접두어를
// 놓친 적이 있어 쓰지 않는다.
func newRule(id, severity, desc string) Rule {
	if ruleIndex[id] {
		panic("규칙 ID가 중복 정의되었습니다: " + id)
	}
	r := Rule{ID: id, Severity: severity, Desc: desc}
	ruleIndex[id] = true
	registry = append(registry, r)
	return r
}

// Rules는 이 바이너리가 검사하는 규칙 전부를 반환한다. 규칙 메타데이터의
// 진실원은 이 목록 하나다. rules show 가 여기서 읽고 계획된 eject 도
// 여기서 읽는다(ADR 0039).
func Rules() []Rule {
	out := make([]Rule, len(registry))
	copy(out, registry)
	return out
}

// newViolation은 등록된 규칙의 위반을 만든다. 등록되지 않은 규칙이
// 여기 오면 프로그램 결함이므로 바로 죽인다. 문자열을 직접 짓는
// 경로를 닫아 규칙과 메타데이터가 어긋나는 것을 막는다.
func newViolation(sev Severity, r Rule, path string, line int, msg, fix string) Violation {
	if !ruleIndex[r.ID] {
		panic("등록되지 않은 규칙: " + r.ID)
	}
	return Violation{Rule: r.ID, Severity: sev, Path: path, Line: line, Message: msg, Fix: fix}
}

// 규칙 정의. 나열 순서는 게이트의 거절 사유를 먼저 읽히는 순서다.
var (
	ruleGateMinWikilinks = newRule("gate.min-wikilinks", "reject",
		i18n.T("lint.rule.gate_min_wikilinks"))
	ruleFrontmatterMissing = newRule("frontmatter.missing", "error",
		i18n.T("lint.rule.frontmatter_missing"))
	ruleFrontmatterUnclosed = newRule("frontmatter.unclosed", "error",
		i18n.T("lint.rule.frontmatter_unclosed"))
	ruleFrontmatterYAML = newRule("frontmatter.yaml", "error",
		i18n.T("lint.rule.frontmatter_yaml"))
	ruleFrontmatterMissingField = newRule("frontmatter.missing-field", "error",
		i18n.T("lint.rule.frontmatter_missing_field"))
	ruleSchemaAllowedValue = newRule("schema.allowed-value", "error",
		i18n.T("lint.rule.schema_allowed_value"))
	ruleSchemaAxisOff = newRule("schema.axis-off", "error",
		i18n.T("lint.rule.schema_axis_off"))
	ruleLocationStageAgreement = newRule("location.stage-agreement", "error 또는 warn",
		i18n.T("lint.rule.location_stage_agreement"))
	ruleTaxonomyForms = newRule("taxonomy.forms", "error",
		i18n.T("lint.rule.taxonomy_forms"))
	ruleSourcesUpdated = newRule("sources.updated", "warn",
		i18n.T("lint.rule.sources_updated"))
	ruleTaxonomyTopics = newRule("taxonomy.topics", "warn",
		i18n.T("lint.rule.taxonomy_topics"))
	ruleBodyMaxLines = newRule("body.max-lines", "warn",
		i18n.T("lint.rule.body_max_lines"))
	ruleLinkBroken = newRule("link.broken", "warn",
		i18n.T("lint.rule.link_broken"))
	ruleGraphOrphan = newRule("graph.orphan", "warn",
		i18n.T("lint.rule.graph_orphan"))
	ruleGateDeferred = newRule("gate.deferred", "warn",
		i18n.T("lint.rule.gate_deferred"))
	ruleWikiBroadTopic = newRule("wiki.broad-topic", "warn",
		i18n.T("lint.rule.wiki_broad_topic"))
)

// Violation은 위반 하나다. 모든 위반은 경로와 줄, 무엇이 잘못됐는지,
// 어떻게 고치는지를 담는다. ADR 0009의 메시지 품질 요구다.
type Violation struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix"`
}

// Summary는 등급별 위반 수와 검사한 파일 수다.
type Summary struct {
	Files  int `json:"files"`
	Error  int `json:"error"`
	Warn   int `json:"warn"`
	Reject int `json:"reject"`
}

// Result는 lint 실행 결과다.
type Result struct {
	Violations   []Violation   `json:"violations"`
	WikiFindings []WikiFinding `json:"wikiFindings"`
	Summary      Summary       `json:"summary"`
}

// HasBlocking은 승급을 막는 위반(error 또는 reject)이 있는지를 반환한다.
func (r Result) HasBlocking() bool {
	return r.Summary.Error > 0 || r.Summary.Reject > 0
}

// WikiFinding은 파일이 아니라 위키 전체의 통계로 판정되는 진단이다.
// 파일 위반 배열에 섞지 않고 별도로 보고한다. 대표 규칙은 wiki.broad-topic이다.
type WikiFinding struct {
	Rule      string   `json:"rule"`
	Severity  Severity `json:"severity"`
	Topic     string   `json:"topic"`
	Percent   int      `json:"percent"`
	Threshold int      `json:"threshold"`
	Total     int      `json:"total"`
	Paths     []string `json:"paths"`
	Fix       string   `json:"fix"`
}

// scannedDoc는 검사한 문서 하나의 파싱 결과와 파생 값이다.
type scannedDoc struct {
	rel       string
	content   string
	d         doc.Doc
	stage     string
	fields    map[string]doc.Field
	related   []doc.Link
	bodyLinks []doc.Link
}

// sourcesDirName은 원본 보존 계층의 디렉토리 이름이다. 이 계층 문서는
// updated 필드를 쓰지 않는다. ADR 0009.
const sourcesDirName = "sources"

// Run은 위키 루트를 순회해 위반 목록을 반환한다.
// 순회와 파싱은 internal/walk 가 담당한다. 순회는 경로 기준 정렬,
// 파일 안은 줄 번호와 규칙 ID 기준 정렬이므로 같은 위키에 대한 결과는
// 항상 바이트까지 같다.
func Run(wikiRoot string, cfg config.Config) (Result, error) {
	walked, err := walk.Files(wikiRoot, cfg)
	if err != nil {
		return Result{}, err
	}
	var docs []scannedDoc
	var violations []Violation
	for _, w := range walked {
		if w.Err != nil {
			violations = append(violations, parseViolations(w)...)
			continue
		}
		// 프론트매터가 아예 없는 문서는 문서 단위 규칙을 적용할 수 없으므로
		// 위반만 남기고 그래프 판정 대상에서 빠진다.
		if !w.Parsed.HasFrontmatter {
			violations = append(violations, newViolation(SevError, ruleFrontmatterMissing, w.Rel, 1,
				i18n.T("lint.violation.frontmatter_missing.message"),
				i18n.T("lint.violation.frontmatter_missing.fix")))
			continue
		}
		sd, vs := scanDoc(w, cfg)
		docs = append(docs, sd)
		violations = append(violations, vs...)
	}
	violations = append(violations, graphRules(docs, walked, cfg)...)
	sortViolations(violations)
	findings := broadTopicFindings(docs, cfg)
	res := Result{Violations: violations, WikiFindings: findings}
	if res.Violations == nil {
		res.Violations = []Violation{}
	}
	if res.WikiFindings == nil {
		res.WikiFindings = []WikiFinding{}
	}
	// 검사한 파일 수는 파싱에 실패한 문서도 포함한다. 위반은 있지만
	// 문서로는 세지 못한 파일이 숨지 않게 하기 위해서다.
	res.Summary.Files = len(walked)
	for _, v := range res.Violations {
		switch v.Severity {
		case SevError:
			res.Summary.Error++
		case SevWarn:
			res.Summary.Warn++
		case SevReject:
			res.Summary.Reject++
		}
	}
	// 위키 단위 진단도 요약 카운트에 포함한다.
	for _, f := range res.WikiFindings {
		if f.Severity == SevWarn {
			res.Summary.Warn++
		}
	}
	return res, nil
}

// parseViolations는 파싱에 실패한 문서 하나의 위반을 만든다.
// 오류 종류는 walk 의 센티널로 구분한다. 에러 문자열 매칭을 쓰지 않는다.
func parseViolations(w walk.Doc) []Violation {
	if errors.Is(w.Err, walk.ErrUnclosed) {
		return []Violation{newViolation(SevError, ruleFrontmatterUnclosed, w.Rel, 1,
			i18n.T("lint.violation.frontmatter_unclosed.message"),
			i18n.T("lint.violation.frontmatter_unclosed.fix"))}
	}
	return []Violation{newViolation(SevError, ruleFrontmatterYAML, w.Rel,
		yamlErrorLine(w.Err.Error()),
		i18n.T("lint.violation.frontmatter_yaml.message", w.Err.Error()),
		i18n.T("lint.violation.frontmatter_yaml.fix"))}
}

// scanDoc는 파싱된 문서 하나에 문서 단위 규칙을 적용한다.
// 순회와 파싱은 internal/walk 가 이미 마쳤다.
func scanDoc(w walk.Doc, cfg config.Config) (scannedDoc, []Violation) {
	rel := w.Rel
	var vs []Violation
	add := func(sev Severity, r Rule, line int, msg, fix string) {
		vs = append(vs, newViolation(sev, r, rel, line, msg, fix))
	}

	d := w.Parsed
	sd := scannedDoc{
		rel:       rel,
		content:   w.Content,
		d:         d,
		fields:    fieldMap(d),
		related:   d.FrontmatterLinks(),
		bodyLinks: d.BodyLinks(),
	}
	if f, ok := sd.fields["artifact_stage"]; ok && f.Kind == doc.KindString {
		sd.stage = f.Str
	}

	sd.checkRequiredFields(cfg, add)
	sd.checkAllowedValues(cfg, add)
	sd.checkStageAgreement(cfg, add)
	sd.checkAxisOff(cfg, add)
	sd.checkTaxonomy(cfg, add)
	sd.checkSourcesUpdated(add)
	sd.checkMaxLines(cfg, add)
	return sd, vs
}

// fieldMap는 필드 순서 목록을 조회용 맵으로 바꾼다.
func fieldMap(d doc.Doc) map[string]doc.Field {
	m := make(map[string]doc.Field, len(d.Fields))
	for _, f := range d.Fields {
		m[f.Key] = f
	}
	return m
}

// yamlErrorLine은 파싱 에러 메시지에서 줄 번호를 꺼낸다. 없으면 프론트매터
// 첫 줄인 2를 쓴다.
func yamlErrorLine(msg string) int {
	m := regexp.MustCompile(`줄 (\d+)`).FindStringSubmatch(msg)
	if m != nil {
		n := 0
		for _, c := range m[1] {
			n = n*10 + int(c-'0')
		}
		if n > 0 {
			return n
		}
	}
	return 2
}

// axisFields는 속성 이름과 프론트매터 키가 같은 14종을 반환한다.
func axisFields() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// RequiredFields는 단계별 필수 필드를 반환한다. upstream 계약
// meta/frontmatter-schema.md의 단계별 필수 정의에서 왔고, 꺼진 속성의
// 필수성은 사라진다.
func RequiredFields(stage string, cfg config.Config) []string {
	req := []string{"type", "artifact_stage", "status", "indexable"}
	for _, f := range []string{"scope", "sensitivity", "source_channel", "trigger_mode", "workflow"} {
		if cfg.Axes[config.Axis(f)] {
			req = append(req, f)
		}
	}
	on := func(field string) bool { return cfg.Axes[config.Axis(field)] }
	switch stage {
	case "source":
		for _, f := range []string{"source_refs", "derived_from", "derived_context"} {
			if on(f) {
				req = append(req, f)
			}
		}
		req = append(req, "created", "sourced_at")
	case "context":
		for _, f := range []string{"source_refs", "derived_from", "related"} {
			if on(f) {
				req = append(req, f)
			}
		}
	}
	return req
}

// checkRequiredFields는 단계별 필수 필드 누락을 검사한다.
func (s *scannedDoc) checkRequiredFields(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	if _, ok := s.fields["artifact_stage"]; !ok {
		// artifact_stage 는 단계 판정의 입력이다. 없으면 그 자체가
		// 오류이다(ADR 0040). 어느 단계인지 모르므로 다른 필수 필드는
		// 보고하지 않는다. 값을 채우면 다음 실행이 나머지를 본다.
		if cfg.Axes[config.AxisArtifactStage] {
			add(SevError, ruleFrontmatterMissingField, lineOfKey(s.content, "artifact_stage"),
				i18n.T("lint.violation.stage_missing.message"),
				i18n.T("lint.violation.stage_missing.fix"))
		}
		return
	}
	if s.stage == "" {
		return // 값이 있어도 문자열이 아니면 단계를 알 수 없다
	}
	for _, f := range RequiredFields(s.stage, cfg) {
		if _, ok := s.fields[f]; !ok {
			add(SevError, ruleFrontmatterMissingField, lineOfKey(s.content, "artifact_stage"),
				i18n.T("lint.violation.required_missing.message", s.stage, f),
				i18n.T("lint.violation.required_missing.fix", f))
		}
	}
}

// valueFields는 허용값 검사 대상 필드와 그 속성, 허용값 집합을 묶은 것이다.
type valueField struct {
	key  string
	axis config.Axis
	set  func(config.Config) []string
}

// valueFields는 허용값 검사 대상이다. 검사 순서가 곧 위반 정렬의 기준
// 하나이므로 고정 순서를 유지한다.
func valueFields() []valueField {
	get := func(pick func(config.Schema) config.ClosedSet) func(config.Config) []string {
		return func(c config.Config) []string { return pick(c.Schema).Values }
	}
	return []valueField{
		{"artifact_stage", config.AxisArtifactStage, get(func(s config.Schema) config.ClosedSet { return s.ArtifactStages })},
		{"status", config.AxisStatus, get(func(s config.Schema) config.ClosedSet { return s.Statuses })},
		{"scope", config.AxisScope, get(func(s config.Schema) config.ClosedSet { return s.Scopes })},
		{"sensitivity", config.AxisSensitivity, get(func(s config.Schema) config.ClosedSet { return s.Sensitivities })},
		{"trigger_mode", config.AxisTriggerMode, get(func(s config.Schema) config.ClosedSet { return s.TriggerModes })},
	}
}

// checkAllowedValues는 허용값 밖의 값을 검사한다. 속성이 꺼져 있으면
// checkAxisOff가 이미 잡으므로 여기서는 보지 않는다.
func (s *scannedDoc) checkAllowedValues(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	for _, vf := range valueFields() {
		if !cfg.Axes[vf.axis] {
			continue
		}
		f, ok := s.fields[vf.key]
		if !ok || f.Kind != doc.KindString || f.Str == "" {
			continue
		}
		allowed := vf.set(cfg)
		if !contains(allowed, f.Str) {
			add(SevError, ruleSchemaAllowedValue, lineOfKey(s.content, vf.key),
				i18n.T("lint.violation.allowed_value.message", vf.key, f.Str, strings.Join(allowed, ", ")),
				i18n.T("lint.violation.allowed_value.fix", vf.key))
		}
	}
}

// checkStageAgreement는 문서가 놓인 최상위 디렉토리와 artifact_stage
// 값의 일치를 검사한다(ADR 0031). 단계와 디렉토리의 대응은 wiki의
// stageDirs 표가 단일 진실원이므로 여기에 두지 않는다. artifact_stage가
// 없으면 frontmatter.missing-field가 이미 잡으므로 건너뛴다. 값이
// 허용 집합 밖이어도 비교한다. 허용값 위반과 위치 불일치는 다른 결함이다.
// 등급은 방향으로 나눈다(ADR 0035). context를 선언했는데 context
// 디렉토리에 없는 문서는 게이트를 우회하므로 error고, 그 밖의 불일치는
// 느슨한 필수 필드 검사로 그치므로 warn이다.
func (s *scannedDoc) checkStageAgreement(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	if s.stage == "" {
		return
	}
	if isRootFile(s.rel, cfg) {
		return // 색인은 위키 루트에 있어 비교할 디렉토리가 없다(ADR 0019)
	}
	top, _, _ := strings.Cut(s.rel, "/")
	expected, ok := wiki.StageForDir(top)
	if !ok {
		return // 단계에 대응하지 않는 디렉토리는 비교 기준이 없다
	}
	if s.stage != string(expected) {
		sev := SevWarn
		if s.stage == string(wiki.StageContext) {
			// context를 선언한 문서가 여기 걸렸으면 context 디렉토리 밖에
			// 있다는 뜻이다. 검수된 지식의 필드 집합과 색인 자격을 훔친다.
			sev = SevError
		}
		add(sev, ruleLocationStageAgreement, lineOfKey(s.content, "artifact_stage"),
			i18n.T("lint.violation.stage_agreement.message", top, s.stage),
			i18n.T("lint.violation.stage_agreement.fix", expected))
	}
}

// checkAxisOff는 설정이 끈 속성이 문서에 있는지 검사한다.
func (s *scannedDoc) checkAxisOff(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	for _, ax := range axisFields() {
		if cfg.Axes[ax] {
			continue
		}
		if _, ok := s.fields[string(ax)]; ok {
			add(SevError, ruleSchemaAxisOff, lineOfKey(s.content, string(ax)),
				i18n.T("lint.violation.axis_off.message", ax, cfg.Preset),
				i18n.T("lint.violation.axis_off.fix", ax, ax))
		}
	}
}

// checkTaxonomy는 form 폐쇄 집합과 topics 개방 집합을 검사한다.
func (s *scannedDoc) checkTaxonomy(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	if f, ok := s.fields["form"]; ok && f.Kind == doc.KindString && f.Str != "" {
		forms := cfg.Schema.Taxonomy.Forms.Values
		if len(forms) > 0 && !contains(forms, f.Str) {
			add(SevError, ruleTaxonomyForms, lineOfKey(s.content, "form"),
				i18n.T("lint.violation.forms.message", f.Str, strings.Join(forms, ", ")),
				i18n.T("lint.violation.forms.fix"))
		}
	}
	if f, ok := s.fields["topics"]; ok && (f.Kind == doc.KindStringList) {
		topics := cfg.Schema.Taxonomy.Topics.Values
		for _, v := range f.List {
			if !contains(topics, v) {
				add(SevWarn, ruleTaxonomyTopics, lineOfKey(s.content, "topics"),
					i18n.T("lint.violation.topics.message", v),
					i18n.T("lint.violation.topics.fix", v))
			}
		}
	}
}

// checkSourcesUpdated는 sources 계층 문서의 updated 필드를 검사한다.
func (s *scannedDoc) checkSourcesUpdated(add func(Severity, Rule, int, string, string)) {
	if _, ok := s.fields["updated"]; !ok {
		return
	}
	if s.rel == sourcesDirName || strings.HasPrefix(s.rel, sourcesDirName+"/") {
		add(SevWarn, ruleSourcesUpdated, lineOfKey(s.content, "updated"),
			i18n.T("lint.violation.sources_updated.message"),
			i18n.T("lint.violation.sources_updated.fix"))
	}
}

// checkMaxLines는 문서 줄 수 상한을 검사한다.
func (s *scannedDoc) checkMaxLines(cfg config.Config, add func(Severity, Rule, int, string, string)) {
	if n := lineCount(s.content); n > cfg.Thresholds.MaxLines {
		add(SevWarn, ruleBodyMaxLines, lineOfKey(s.content, "artifact_stage"),
			i18n.T("lint.violation.max_lines.message", n, cfg.Thresholds.MaxLines),
			i18n.T("lint.violation.max_lines.fix"))
	}
}

// GateResult는 승급 게이트 판정 결과다.
type GateResult struct {
	Passed   bool // 게이트를 통과했는가. 유예와 게이트 오프도 통과다
	Deferred bool // 링크 대상이 부족해 게이트를 적용하지 않았는가 (ADR 0021)
	Links    int  // 그 문서의 고유 위키링크 수
	Targets  int  // 자신을 뺀 링크 가능 문서 수
	Min      int  // min_wikilinks
}

// EvaluateGate는 문서 하나의 승급 게이트를 판정한다. links 는 그 문서의
// 고유 위키링크 수, targets 는 자신을 뺀 링크 가능 문서 수다.
// lint, promote, new 가 같은 판정을 써야 커맨드로 통과한 문서를 lint 가
// 거절하지 않는다. min_wikilinks 가 0 이면 게이트 자체가 꺼져 있어
// 유예 표시도 하지 않는다.
func EvaluateGate(links, targets, minWikilinks int) GateResult {
	g := GateResult{Links: links, Targets: targets, Min: minWikilinks}
	switch {
	case minWikilinks <= 0:
		g.Passed = true
	case targets < minWikilinks:
		// 링크 대상이 물리적으로 없는 상태는 고립이 아니라 시작이다.
		g.Passed, g.Deferred = true, true
	default:
		g.Passed = links >= minWikilinks
	}
	return g
}

// OrphanCount는 lint 결과에서 고아 문서 수를 낸다. status 같은 다른
// 지표가 graph.orphan 과 같은 판정을 쓰게 하는 통로다. 고아의 정의는
// graph.orphan 규칙이 단일 진실원이므로 여기서 다시 세지 않는다.
func OrphanCount(res Result) int {
	n := 0
	for _, v := range res.Violations {
		if v.Rule == ruleGraphOrphan.ID {
			n++
		}
	}
	return n
}

// Linkable은 문서가 게이트의 유효한 연결 대상인지를 판정한다.
// inbox 단계 문서는 promote 되면 파일이 옮겨지며 슬러그가 날짜 접두사가
// 떨어진 형태로 바뀌어 가리키던 링크가 깨지므로 뺀다(ADR 0022).
// 단계를 읽을 수 없는 문서(파싱 실패, 프론트매터 없음, 필드 없음)도
// 그 자리에 남으리라 보장할 수 없어 뺀다. 색인(root_files)은 승급
// 대상이 아니라 그 자리에 남으므로 포함한다.
// EvaluateGate 를 부르는 모든 곳이 이 판정을 공유해야 대상 수가 갈라지지 않는다.
func Linkable(w walk.Doc) bool {
	if w.Root {
		return true
	}
	if w.Err != nil || !w.Parsed.HasFrontmatter {
		return false
	}
	for _, f := range w.Parsed.Fields {
		if f.Key == string(config.AxisArtifactStage) {
			return f.Kind == doc.KindString && f.Str != "inbox"
		}
	}
	return false
}

// LinkableTargets는 판정 대상 문서 self 를 뺀 유효 연결 대상 수를 센다.
func LinkableTargets(walked []walk.Doc, self string) int {
	n := 0
	for _, w := range walked {
		if w.Rel != self && Linkable(w) {
			n++
		}
	}
	return n
}

// graphRules는 전체 문서를 모은 뒤에야 판정할 수 있는 규칙을 적용한다.
// 깨진 링크, 고아 문서, 승급 게이트다. walked 는 순회 결과 전체로
// 게이트 유예의 대상 수를 잰다.
func graphRules(docs []scannedDoc, walked []walk.Doc, cfg config.Config) []Violation {
	var vs []Violation
	add := func(sev Severity, r Rule, rel string, line int, msg, fix string) {
		vs = append(vs, newViolation(sev, r, rel, line, msg, fix))
	}

	// 게이트의 발동 조건은 문서가 놓인 디렉토리다(ADR 0040). 승급은 파일을
	// 옮기는 행위이므로 위치가 운영의 진실원이고 resurface 와 status 도
	// 위치로 센다. 선언을 보면 값을 비우거나 낮춰 우회할 수 있다.
	contextDir, dirErr := wiki.DirFor(cfg, wiki.StageContext)
	gateOn := dirErr == nil && cfg.Thresholds.MinWikilinks > 0

	bySlug := map[string]string{}
	for _, s := range docs {
		slug := strings.TrimSuffix(filepath.Base(s.rel), ".md")
		if _, ok := bySlug[slug]; !ok {
			bySlug[slug] = s.rel
		}
	}

	// 고아 판정의 들어오는 관계는 위키링크와 관계 필드를 같은 기준으로 센다.
	// 키는 날짜 접두사를 뺀 슬러그로 정규화한다. 경로 형태 값과 슬러그 형태
	// 값이 같은 문서를 가리키도록 하기 위해서다.
	incoming := map[string]int{}
	for _, s := range docs {
		self := relationSlug(s.rel)
		for _, l := range append(append([]doc.Link{}, s.related...), s.bodyLinks...) {
			if key := relationSlug(l.Slug); key != self {
				incoming[key]++
			}
			if _, ok := bySlug[l.Slug]; !ok {
				add(SevWarn, ruleLinkBroken, s.rel, l.Line,
					i18n.T("lint.violation.link_broken.message", l.Slug),
					i18n.T("lint.violation.link_broken.fix", l.Slug))
			}
		}
		for _, v := range relationValues(s) {
			if key := relationSlug(v); key != "" && key != self {
				incoming[key]++
			}
		}
	}

	for _, s := range docs {
		slug := strings.TrimSuffix(filepath.Base(s.rel), ".md")
		// 색인 문서는 승급 게이트와 고아 판정에서 뺀다. 게이트는 지식 노드가
		// 고립된 채 context 로 올라가는 것을 막는 장치인데 색인은 승급 대상이
		// 아니라 위키의 구조 자체다. 갓 만든 위키의 색인이 가리킬 문서가 없는
		// 것은 결함이 아니라 초기 상태라서 도구가 자기 산출물을 거절하는 꼴이 된다.
		// 스키마, taxonomy, 프론트매터 형식 규칙은 색인에도 그대로 적용된다.
		if isRootFile(s.rel, cfg) {
			continue
		}
		// 고아의 정의는 어떤 관계도 없음이다. 관계는 related, 본문 위키링크,
		// 그리고 derived_from, derived_context, source_refs 다. 관계 필드는
		// 위키링크가 아니므로 존재만 보고 대상 문서를 해석하지 않는다.
		// 승급 파이프라인이 양방향 관계를 기록하므로(ADR 0022) 파이프라인의
		// 산출물을 검사기가 못 보면 안 된다.
		// 게이트(gate.min-wikilinks)는 이 판정과 달리 위키링크만 센다.
		// ADR 0009 가 게이트 대상을 위키링크로 규정했고 관계 필드는 도구가
		// 채우므로 게이트에 포함하면 게이트가 무력해진다.
		// 고아 여부는 그 문서의 관계 유무로 정해지고 고치는 행위도 그 문서에서
		// 일어나므로 broad-topic과 달리 파일 단위 위반이 맞다.
		outgoing := uniqueSlugs(s)
		related := len(relationValues(s)) > 0
		if len(outgoing) == 0 && !related && incoming[relationSlug(s.rel)] == 0 {
			add(SevWarn, ruleGraphOrphan, s.rel, 1,
				i18n.T("lint.violation.orphan.message"),
				i18n.T("lint.violation.orphan.fix", slug))
		}
		if gateOn && strings.HasPrefix(s.rel, contextDir+"/") {
			n := len(outgoing)
			g := EvaluateGate(n, LinkableTargets(walked, s.rel), cfg.Thresholds.MinWikilinks)
			// 유예는 링크가 부족해 게이트에 걸렸을 문서만 알린다.
			// 링크가 기준을 채운 문서는 유예와 무관하게 스스로 통과한다.
			if g.Deferred && n < cfg.Thresholds.MinWikilinks {
				add(SevWarn, ruleGateDeferred, s.rel, lineOfKey(s.content, "related"),
					i18n.T("lint.violation.gate_deferred.message", g.Targets, g.Min, g.Min),
					i18n.T("lint.violation.gate_deferred.fix"))
				continue
			}
			if !g.Passed {
				add(SevReject, ruleGateMinWikilinks, s.rel, lineOfKey(s.content, "related"),
					i18n.T("lint.violation.gate_reject.message", n, cfg.Thresholds.MinWikilinks),
					i18n.T("lint.violation.gate_reject.fix", cfg.Thresholds.MinWikilinks-n))
			}
		}
	}
	return vs
}

// broadTopicFindings는 한 주제가 전체 문서의 일정 비율을 넘게 붙었는지
// 검사해 주제당 하나의 위키 단위 진단을 만든다. 주제 비율은 위키 전체의
// 통계이지 그 문서의 결함이 아니므로 파일 위반에 넣지 않는다.
// 진단은 비율 내림차순, 같으면 주제 이름순으로 정렬한다.
func broadTopicFindings(docs []scannedDoc, cfg config.Config) []WikiFinding {
	if len(docs) == 0 || cfg.Thresholds.BroadTopicPct <= 0 {
		return nil
	}
	total := len(docs)
	count := map[string]int{}
	paths := map[string][]string{}
	for _, s := range docs {
		f, ok := s.fields["topics"]
		if !ok || f.Kind != doc.KindStringList {
			continue
		}
		// 한 문서가 같은 주제를 거듭 쓰면 문서 수 기준으로 한 번만 센다.
		seen := map[string]bool{}
		for _, v := range f.List {
			if seen[v] {
				continue
			}
			seen[v] = true
			count[v]++
			paths[v] = append(paths[v], s.rel)
		}
	}
	var out []WikiFinding
	for topic, n := range count {
		pct := 100 * n / total
		if pct <= cfg.Thresholds.BroadTopicPct {
			continue
		}
		out = append(out, WikiFinding{
			Rule:      ruleWikiBroadTopic.ID,
			Severity:  SevWarn,
			Topic:     topic,
			Percent:   pct,
			Threshold: cfg.Thresholds.BroadTopicPct,
			Total:     total,
			Paths:     paths[topic],
			Fix:       i18n.T("lint.wiki.broad_topic.fix"),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Percent != out[j].Percent {
			return out[i].Percent > out[j].Percent
		}
		return out[i].Topic < out[j].Topic
	})
	return out
}

// isRootFile은 문서가 config 의 root_files(기본값 index.md)에 해당하는
// 색인 문서인지를 반환한다. root_files 는 위키 루트에 두는 파일 목록이므로
// 비교는 경로 전체로 한다.
func isRootFile(rel string, cfg config.Config) bool {
	for _, f := range cfg.RootFiles {
		if f == rel {
			return true
		}
	}
	return false
}

// uniqueSlugs는 문서가 가진 위키링크 슬러그를 중복 없이 모은다.
func uniqueSlugs(s scannedDoc) map[string]bool {
	out := map[string]bool{}
	for _, l := range append(append([]doc.Link{}, s.related...), s.bodyLinks...) {
		out[l.Slug] = true
	}
	return out
}

// relationFieldKeys는 고아 판정에서 관계로 세는 관계 필드다.
// related 와 본문 위키링크는 링크 추출 단계에서 이미 다뤄진다.
func relationFieldKeys() []string {
	return []string{"derived_from", "derived_context", "source_refs"}
}

// relationValues는 문서의 관계 필드 값을 모은다. 값이 경로 형태든
// 슬러그 형태든 관계가 있다는 사실은 같으므로 존재만 확인한다.
func relationValues(s scannedDoc) []string {
	var out []string
	for _, key := range relationFieldKeys() {
		f, ok := s.fields[key]
		if !ok {
			continue
		}
		switch f.Kind {
		case doc.KindStringList:
			out = append(out, f.List...)
		case doc.KindString:
			if f.Str != "" {
				out = append(out, f.Str)
			}
		}
	}
	return out
}

// relationSlug는 관계 값이나 문서 경로를 고아 판정의 비교 키로 정규화한다.
// 위키링크 껍데기를 벗기고, 경로면 파일명만 남기고, 확장자와 날짜 접두사
// (YYYY-MM-DD- 또는 YYYY-MM-)를 뗀다. promote 가 문서를 옮기며 날짜
// 접두사를 떼기 때문에(ADR 0022) 접두사가 붙은 이름과 떨어진 이름이 같은
// 문서를 가리킨다.
func relationSlug(v string) string {
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

// sortViolations는 경로, 줄 번호, 규칙 ID 순으로 위반을 정렬한다.
func sortViolations(vs []Violation) {
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].Path != vs[j].Path {
			return vs[i].Path < vs[j].Path
		}
		if vs[i].Line != vs[j].Line {
			return vs[i].Line < vs[j].Line
		}
		return vs[i].Rule < vs[j].Rule
	})
}

// lineOfKey는 원문에서 "키:" 줄을 찾아 1 기반 줄 번호를 반환한다.
// 못 찾으면 프론트매터 시작 줄인 1을 쓴다.
func lineOfKey(content, key string) int {
	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), key+":") {
			return i + 1
		}
	}
	return 1
}

// lineCount는 파일의 줄 수를 센다.
func lineCount(content string) int {
	if content == "" {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}

// contains는 값이 목록에 있는지 검사한다.
func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
