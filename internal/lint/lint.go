// Package lint는 위키를 순회하며 스키마와 링크 무결성을 검사한다.
//
// 규칙 ID는 점 표기 소문자이고 ADR 0005의 parity 비교 키가 되므로
// 한 번 정하면 바꾸지 않는다는 전제로 지었다.
package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
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
// 순회는 경로 기준 정렬, 파일 안은 줄 번호와 규칙 ID 기준 정렬이므로
// 같은 위키에 대한 결과는 항상 바이트까지 같다.
func Run(wikiRoot string, cfg config.Config) (Result, error) {
	rels, err := scanFiles(wikiRoot, cfg)
	if err != nil {
		return Result{}, err
	}
	var docs []scannedDoc
	var violations []Violation
	for _, rel := range rels {
		sd, vs, err := scanDoc(wikiRoot, rel, cfg)
		if err != nil {
			return Result{}, err
		}
		if sd != nil {
			docs = append(docs, *sd)
		}
		violations = append(violations, vs...)
	}
	violations = append(violations, graphRules(docs, cfg)...)
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
	res.Summary.Files = len(rels)
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

// scanFiles는 검사 대상 .md 파일의 상대 경로를 정렬해 반환한다.
// page_dirs 아래와 루트 파일이 대상이고 숨김 디렉토리는 건너뛴다.
func scanFiles(wikiRoot string, cfg config.Config) ([]string, error) {
	seen := map[string]bool{}
	var rels []string
	add := func(rel string) {
		if !seen[rel] {
			seen[rel] = true
			rels = append(rels, rel)
		}
	}
	dirs := append([]string{}, cfg.PageDirs...)
	sort.Strings(dirs)
	for _, d := range dirs {
		if d == ".engram" || strings.HasPrefix(d, ".") {
			continue
		}
		err := filepath.WalkDir(filepath.Join(wikiRoot, d), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".md") {
				rel, err := filepath.Rel(wikiRoot, path)
				if err != nil {
					return err
				}
				add(filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, f := range cfg.RootFiles {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		if _, err := os.Stat(filepath.Join(wikiRoot, f)); err == nil {
			add(filepath.ToSlash(f))
		}
	}
	sort.Strings(rels)
	return rels, nil
}

// scanDoc는 문서 하나를 파싱하고 문서 단위 규칙을 적용한다.
// 파싱 자체가 실패한 문서는 nil 문서와 프론트매터 위반을 반환한다.
func scanDoc(wikiRoot, rel string, cfg config.Config) (*scannedDoc, []Violation, error) {
	raw, err := os.ReadFile(filepath.Join(wikiRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, nil, fmt.Errorf("문서를 읽을 수 없음: %s: %w", rel, err)
	}
	text := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	text = strings.ReplaceAll(text, "\r\n", "\n")

	var vs []Violation
	add := func(sev Severity, rule string, line int, msg, fix string) {
		vs = append(vs, Violation{Rule: rule, Severity: sev, Path: rel, Line: line, Message: msg, Fix: fix})
	}

	// 닫는 구분자 검사를 doc.Parse 앞에 둔다. 파싱 에러의 종류를
	// 에러 문자열 매칭 없이 확정하기 위해서다.
	lines := strings.Split(text, "\n")
	if strings.TrimRight(lines[0], " \t") == "---" {
		closed := false
		for _, l := range lines[1:] {
			if strings.TrimRight(l, " \t") == "---" {
				closed = true
				break
			}
		}
		if !closed {
			add(SevError, "frontmatter.unclosed", 1,
				"프론트매터가 닫는 --- 구분자 없이 끝났다",
				"프론트매터 끝에 --- 줄을 추가한다")
			return nil, vs, nil
		}
	}

	d, err := doc.Parse(rel, []byte(text))
	if err != nil {
		add(SevError, "frontmatter.yaml", yamlErrorLine(err.Error()),
			"프론트매터 YAML 파싱 실패: "+err.Error(),
			"프론트매터의 YAML 문법을 고친다")
		return nil, vs, nil
	}
	if !d.HasFrontmatter {
		add(SevError, "frontmatter.missing", 1,
			"프론트매터가 없다",
			"문서 첫 줄에 --- 로 여는 구분자를 두고 필드를 채운 뒤 --- 로 닫는다")
		return nil, vs, nil
	}

	sd := &scannedDoc{
		rel:       rel,
		content:   text,
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
	return sd, vs, nil
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
				fmt.Sprintf("단계 %s의 필수 필드 %s가 없다", s.stage, f),
				fmt.Sprintf("프론트매터에 %s 필드를 추가한다", f))
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
				fmt.Sprintf("%s 값이 허용값 밖이다: %q (허용값: %s)", vf.key, f.Str, strings.Join(allowed, ", ")),
				fmt.Sprintf("%s 값을 허용값 중 하나로 바꾼다", vf.key))
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
				fmt.Sprintf("설정에서 꺼진 축의 필드가 문서에 있다: %s (프리셋 %s)", ax, cfg.Preset),
				fmt.Sprintf("engram.yaml의 axes에서 %s를 켜거나 문서에서 %s 필드를 지운다", ax, ax))
		}
	}
}

// checkTaxonomy는 form 폐쇄 집합과 topics 개방 집합을 검사한다.
func (s *scannedDoc) checkTaxonomy(cfg config.Config, add func(Severity, string, int, string, string)) {
	if f, ok := s.fields["form"]; ok && f.Kind == doc.KindString && f.Str != "" {
		forms := cfg.Schema.Taxonomy.Forms.Values
		if len(forms) > 0 && !contains(forms, f.Str) {
			add(SevError, "taxonomy.forms", lineOfKey(s.content, "form"),
				fmt.Sprintf("form 값이 forms 폐쇄 집합에 없다: %q (허용값: %s)", f.Str, strings.Join(forms, ", ")),
				"form 값을 허용값 중 하나로 바꾼다")
		}
	}
	if f, ok := s.fields["topics"]; ok && (f.Kind == doc.KindStringList) {
		topics := cfg.Schema.Taxonomy.Topics.Values
		for _, v := range f.List {
			if !contains(topics, v) {
				add(SevWarn, "taxonomy.topics", lineOfKey(s.content, "topics"),
					fmt.Sprintf("topics 값이 설정에 정의되지 않았다: %q (topics는 개방 집합이다)", v),
					fmt.Sprintf("engram.yaml의 topics 목록에 %q를 추가한다", v))
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
			"sources 계층 문서에 updated 필드가 있다",
			"updated 필드를 지운다. sources는 원본 보존 계층이라 갱신하지 않는다")
	}
}

// checkMaxLines는 문서 줄 수 상한을 검사한다.
func (s *scannedDoc) checkMaxLines(cfg config.Config, add func(Severity, string, int, string, string)) {
	if n := lineCount(s.content); n > cfg.Thresholds.MaxLines {
		add(SevWarn, "body.max-lines", lineOfKey(s.content, "artifact_stage"),
			fmt.Sprintf("문서가 %d줄로 max_lines %d줄을 넘는다", n, cfg.Thresholds.MaxLines),
			"문서를 나눈다. 상한은 engram.yaml의 max_lines로 조정한다")
	}
}

// graphRules는 전체 문서를 모은 뒤에야 판정할 수 있는 규칙을 적용한다.
// 깨진 링크, 고아 문서, 승급 게이트다.
func graphRules(docs []scannedDoc, cfg config.Config) []Violation {
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

	incoming := map[string]int{}
	for _, s := range docs {
		self := strings.TrimSuffix(filepath.Base(s.rel), ".md")
		for _, l := range append(append([]doc.Link{}, s.related...), s.bodyLinks...) {
			if l.Slug != self {
				incoming[l.Slug]++
			}
			if _, ok := bySlug[l.Slug]; !ok {
				add(SevWarn, "link.broken", s.rel, l.Line,
					fmt.Sprintf("깨진 위키링크: [[%s]]에 해당하는 문서가 없다", l.Slug),
					fmt.Sprintf("슬러그를 고치거나 [[%s]] 문서를 만든다", l.Slug))
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
		// 고아 여부는 그 문서의 링크 유무로 정해지고 고치는 행위도 그 문서에서
		// 일어나므로 broad-topic과 달리 파일 단위 위반이 맞다.
		outgoing := uniqueSlugs(s)
		if len(outgoing) == 0 && incoming[slug] == 0 {
			add(SevWarn, "graph.orphan", s.rel, 1,
				"들어오는 링크와 나가는 링크가 모두 없다",
				fmt.Sprintf("다른 문서의 related나 본문에서 [[%s]]로 연결한다", slug))
		}
		if s.stage == "context" && cfg.Thresholds.MinWikilinks > 0 {
			if n := len(outgoing); n < cfg.Thresholds.MinWikilinks {
				add(SevReject, "gate.min-wikilinks", s.rel, lineOfKey(s.content, "related"),
					fmt.Sprintf("위키링크가 %d개로 min_wikilinks %d개에 못 미친다", n, cfg.Thresholds.MinWikilinks),
					fmt.Sprintf("related 필드나 본문에 위키링크를 %d개 더 추가한다", cfg.Thresholds.MinWikilinks-n))
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
			Fix:       "주제를 더 세분한다. 기준은 engram.yaml의 broad_topic_pct로 조정한다",
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
