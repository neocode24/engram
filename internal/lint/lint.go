// Package lint는 위키를 순회하며 스키마와 링크 무결성을 검사한다.
//
// 규칙 ID는 점 표기 소문자이고 ADR 0005의 parity 비교 키가 되므로
// 한 번 정하면 바꾸지 않는다는 전제로 지었다.
package lint

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/walk"
)

// Severity는 위반의 등급이다.
type Severity string

const (
	SevError  Severity = "error"  // 승급을 막는다
	SevWarn   Severity = "warn"   // 통과시키되 알린다
	SevReject Severity = "reject" // 게이트 거절
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
			violations = append(violations, Violation{
				Rule: "frontmatter.missing", Severity: SevError, Path: w.Rel, Line: 1,
				Message: "프론트매터가 없습니다",
				Fix:     "문서 첫 줄에 --- 로 여는 구분자를 두고 필드를 채운 뒤 --- 로 닫으세요",
			})
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
		return []Violation{{
			Rule: "frontmatter.unclosed", Severity: SevError, Path: w.Rel, Line: 1,
			Message: "프론트매터가 닫는 --- 구분자 없이 끝났다",
			Fix:     "프론트매터 끝에 --- 줄을 추가하세요",
		}}
	}
	return []Violation{{
		Rule: "frontmatter.yaml", Severity: SevError, Path: w.Rel,
		Line:    yamlErrorLine(w.Err.Error()),
		Message: "프론트매터 YAML 파싱 실패: " + w.Err.Error(),
		Fix:     "프론트매터의 YAML 문법을 고치세요",
	}}
}

// scanDoc는 파싱된 문서 하나에 문서 단위 규칙을 적용한다.
// 순회와 파싱은 internal/walk 가 이미 마쳤다.
func scanDoc(w walk.Doc, cfg config.Config) (scannedDoc, []Violation) {
	rel := w.Rel
	var vs []Violation
	add := func(sev Severity, rule string, line int, msg, fix string) {
		vs = append(vs, Violation{Rule: rule, Severity: sev, Path: rel, Line: line, Message: msg, Fix: fix})
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

// axisFields는 축 이름과 프론트매터 키가 같은 14축을 반환한다.
func axisFields() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// requiredFields는 단계별 필수 필드를 반환한다. upstream 계약
// meta/frontmatter-schema.md의 단계별 필수 정의에서 왔고, 꺼진 축의
// 필수성은 사라진다.
func requiredFields(stage string, cfg config.Config) []string {
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
func (s *scannedDoc) checkRequiredFields(cfg config.Config, add func(Severity, string, int, string, string)) {
	if s.stage == "" {
		return // artifact_stage 자체의 문제는 다른 규칙이 잡는다
	}
	for _, f := range requiredFields(s.stage, cfg) {
		if _, ok := s.fields[f]; !ok {
			add(SevError, "frontmatter.missing-field", lineOfKey(s.content, "artifact_stage"),
				fmt.Sprintf("단계 %s의 필수 필드 %s가 없습니다", s.stage, f),
				fmt.Sprintf("프론트매터에 %s 필드를 추가하세요", f))
		}
	}
}

// valueFields는 허용값 검사 대상 필드와 그 축, 허용값 집합을 묶은 것이다.
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

// checkAllowedValues는 허용값 밖의 값을 검사한다. 축이 꺼져 있으면
// checkAxisOff가 이미 잡으므로 여기서는 보지 않는다.
func (s *scannedDoc) checkAllowedValues(cfg config.Config, add func(Severity, string, int, string, string)) {
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
			add(SevError, "schema.allowed-value", lineOfKey(s.content, vf.key),
				fmt.Sprintf("%s 값이 허용값 밖입니다: %q (허용값: %s)", vf.key, f.Str, strings.Join(allowed, ", ")),
				fmt.Sprintf("%s 값을 허용값 중 하나로 바꿉니다", vf.key))
		}
	}
}

// checkAxisOff는 설정이 끈 축의 필드가 문서에 있는지 검사한다.
func (s *scannedDoc) checkAxisOff(cfg config.Config, add func(Severity, string, int, string, string)) {
	for _, ax := range axisFields() {
		if cfg.Axes[ax] {
			continue
		}
		if _, ok := s.fields[string(ax)]; ok {
			add(SevError, "schema.axis-off", lineOfKey(s.content, string(ax)),
				fmt.Sprintf("설정에서 꺼진 축의 필드가 문서에 있습니다: %s (프리셋 %s)", ax, cfg.Preset),
				fmt.Sprintf("engram.yaml의 axes에서 %s를 켜거나 문서에서 %s 필드를 지웁니다", ax, ax))
		}
	}
}

// checkTaxonomy는 form 폐쇄 집합과 topics 개방 집합을 검사한다.
func (s *scannedDoc) checkTaxonomy(cfg config.Config, add func(Severity, string, int, string, string)) {
	if f, ok := s.fields["form"]; ok && f.Kind == doc.KindString && f.Str != "" {
		forms := cfg.Schema.Taxonomy.Forms.Values
		if len(forms) > 0 && !contains(forms, f.Str) {
			add(SevError, "taxonomy.forms", lineOfKey(s.content, "form"),
				fmt.Sprintf("form 값이 forms 폐쇄 집합에 없습니다: %q (허용값: %s)", f.Str, strings.Join(forms, ", ")),
				"form 값을 허용값 중 하나로 바꿉니다")
		}
	}
	if f, ok := s.fields["topics"]; ok && (f.Kind == doc.KindStringList) {
		topics := cfg.Schema.Taxonomy.Topics.Values
		for _, v := range f.List {
			if !contains(topics, v) {
				add(SevWarn, "taxonomy.topics", lineOfKey(s.content, "topics"),
					fmt.Sprintf("topics 값이 설정에 정의되지 않았습니다: %q (topics는 개방 집합입니다)", v),
					fmt.Sprintf("engram.yaml의 topics 목록에 %q를 추가하세요", v))
			}
		}
	}
}

// checkSourcesUpdated는 sources 계층 문서의 updated 필드를 검사한다.
func (s *scannedDoc) checkSourcesUpdated(add func(Severity, string, int, string, string)) {
	if _, ok := s.fields["updated"]; !ok {
		return
	}
	if s.rel == sourcesDirName || strings.HasPrefix(s.rel, sourcesDirName+"/") {
		add(SevWarn, "sources.updated", lineOfKey(s.content, "updated"),
			"sources 계층 문서에 updated 필드가 있습니다",
			"updated 필드를 지우세요. sources는 원본 보존 계층이라 갱신하지 않습니다")
	}
}

// checkMaxLines는 문서 줄 수 상한을 검사한다.
func (s *scannedDoc) checkMaxLines(cfg config.Config, add func(Severity, string, int, string, string)) {
	if n := lineCount(s.content); n > cfg.Thresholds.MaxLines {
		add(SevWarn, "body.max-lines", lineOfKey(s.content, "artifact_stage"),
			fmt.Sprintf("문서가 %d줄로 max_lines %d줄을 넘습니다", n, cfg.Thresholds.MaxLines),
			"문서를 나누세요. 상한은 engram.yaml의 max_lines로 조정하세요")
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
		if v.Rule == "graph.orphan" {
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
	add := func(sev Severity, rule, rel string, line int, msg, fix string) {
		vs = append(vs, Violation{Rule: rule, Severity: sev, Path: rel, Line: line, Message: msg, Fix: fix})
	}

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
				add(SevWarn, "link.broken", s.rel, l.Line,
					fmt.Sprintf("깨진 위키링크: [[%s]]에 해당하는 문서가 없습니다", l.Slug),
					fmt.Sprintf("슬러그를 고치거나 [[%s]] 문서를 만드세요", l.Slug))
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
			add(SevWarn, "graph.orphan", s.rel, 1,
				"들어오는 관계와 나가는 관계가 모두 없습니다",
				fmt.Sprintf("다른 문서의 related나 본문에서 [[%s]]로 연결하거나 관계 필드로 잇으세요", slug))
		}
		if s.stage == "context" && cfg.Thresholds.MinWikilinks > 0 {
			n := len(outgoing)
			g := EvaluateGate(n, LinkableTargets(walked, s.rel), cfg.Thresholds.MinWikilinks)
			// 유예는 링크가 부족해 게이트에 걸렸을 문서만 알린다.
			// 링크가 기준을 채운 문서는 유예와 무관하게 스스로 통과한다.
			if g.Deferred && n < cfg.Thresholds.MinWikilinks {
				add(SevWarn, "gate.deferred", s.rel, lineOfKey(s.content, "related"),
					fmt.Sprintf("링크 가능한 대상 문서가 %d개로 min_wikilinks %d개보다 적어 게이트를 유예합니다. 대상 문서가 %d개가 되면 게이트가 동작합니다", g.Targets, g.Min, g.Min),
					fmt.Sprintf("연결할 문서를 만들어 대상을 늘리세요. 기준은 engram.yaml의 min_wikilinks로 조정하세요"))
				continue
			}
			if !g.Passed {
				add(SevReject, "gate.min-wikilinks", s.rel, lineOfKey(s.content, "related"),
					fmt.Sprintf("위키링크가 %d개로 min_wikilinks %d개에 못 미칩니다", n, cfg.Thresholds.MinWikilinks),
					fmt.Sprintf("related 필드나 본문에 위키링크를 %d개 더 추가하세요", cfg.Thresholds.MinWikilinks-n))
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
			Rule:      "wiki.broad-topic",
			Severity:  SevWarn,
			Topic:     topic,
			Percent:   pct,
			Threshold: cfg.Thresholds.BroadTopicPct,
			Total:     total,
			Paths:     paths[topic],
			Fix:       "주제를 더 세분하세요. 기준은 engram.yaml의 broad_topic_pct로 조정하세요",
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
