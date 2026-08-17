// Package config는 위키별 설정 파일(engram.yaml)을 읽어 프론트매터 속성, 임계값,
// 디렉토리 매핑이 확정된 구성을 반환한다.
//
// 병합 순서는 기본값, 프리셋, 파일이다. 파일에 없는 키는 기본값이 남는다.
// 설정 파일이 없는 것은 에러가 아니며 기본값으로 동작한다.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
)

// ConfigFileName은 위키 루트에 두는 설정 파일 이름이다. ADR 0017.
const ConfigFileName = "engram.yaml"

// Preset은 프론트매터 속성의 시작점이 되는 프리셋이다. 포함 관계는
// minimal이 personal에, personal이 team에 포함된다. ADR 0009, 0048.
type Preset string

const (
	PresetMinimal  Preset = "minimal"
	PresetPersonal Preset = "personal"
	PresetTeam     Preset = "team"
)

// DefaultPreset은 프리셋을 지정하지 않았을 때의 기본값이다.
const DefaultPreset = PresetPersonal

// Axis는 프론트매터 스키마의 속성이다. 한국어 표기는 "속성"이고 코드
// 식별자와 engram.yaml 키는 axis, axes 로 남는다. ADR 0048.
type Axis string

const (
	AxisType           Axis = "type"
	AxisArtifactStage  Axis = "artifact_stage"
	AxisStatus         Axis = "status"
	AxisIndexable      Axis = "indexable"
	AxisTags           Axis = "tags"
	AxisSourceRefs     Axis = "source_refs"
	AxisDerivedFrom    Axis = "derived_from"
	AxisRelated        Axis = "related"
	AxisSourceChannel  Axis = "source_channel"
	AxisDerivedContext Axis = "derived_context"
	AxisScope          Axis = "scope"
	AxisSensitivity    Axis = "sensitivity"
	AxisTriggerMode    Axis = "trigger_mode"
	AxisWorkflow       Axis = "workflow"
)

// allAxes는 스키마가 다루는 속성 14종을 반환한다.
func allAxes() []Axis {
	return []Axis{
		AxisType, AxisArtifactStage, AxisStatus, AxisIndexable, AxisTags,
		AxisSourceRefs, AxisDerivedFrom, AxisRelated,
		AxisSourceChannel, AxisDerivedContext,
		AxisScope, AxisSensitivity, AxisTriggerMode, AxisWorkflow,
	}
}

func knownAxis(a Axis) bool {
	for _, x := range allAxes() {
		if x == a {
			return true
		}
	}
	return false
}

func knownPreset(p Preset) bool {
	switch p {
	case PresetMinimal, PresetPersonal, PresetTeam:
		return true
	}
	return false
}

// presetAxes는 프리셋이 켜는 속성 집합을 새 맵으로 반환한다.
// 포함 관계를 누적으로 구현해 minimal이 personal의, personal이 team의
// 부분집합이 되도록 한다.
func presetAxes(p Preset) map[Axis]bool {
	on := make(map[Axis]bool, 14)
	for _, a := range []Axis{
		AxisType, AxisArtifactStage, AxisStatus, AxisIndexable, AxisTags,
		AxisSourceRefs, AxisDerivedFrom, AxisRelated,
	} {
		on[a] = true
	}
	if p == PresetMinimal {
		return on
	}
	on[AxisSourceChannel] = true
	on[AxisDerivedContext] = true
	if p == PresetPersonal {
		return on
	}
	on[AxisScope] = true
	on[AxisSensitivity] = true
	on[AxisTriggerMode] = true
	on[AxisWorkflow] = true
	return on
}

// Origin은 설정 값의 출처다. config list --origin이 이 값을 출력한다.
type Origin string

const (
	OriginDefault Origin = "default"
	OriginPreset  Origin = "preset"
	OriginFile    Origin = "file"
)

// Thresholds는 게이트와 경고의 임계값이다. min_wikilinks만 승급 거절
// 사유이고 나머지는 경고에 쓰인다.
type Thresholds struct {
	MinWikilinks  int
	StaleDays     int
	MaxLines      int
	BroadTopicPct int
}

// OpenSet은 미정의 값이 경고인 값 집합이다.
type OpenSet struct {
	Values []string
}

// ClosedSet은 미정의 값이 오류인 값 집합이다.
type ClosedSet struct {
	Values []string
}

// Taxonomy는 주제와 문서 형태 두 facet을 담는다. 미정의 값의 판정이
// 경고냐 오류냐로 갈리므로 서로 다른 타입으로 구분한다. 판정은 lint가 한다.
type Taxonomy struct {
	Topics OpenSet
	Forms  ClosedSet
}

// DateField는 날짜 필드 하나의 허용 형식이다. Layouts는 time.Parse 레이아웃이다.
type DateField struct {
	Name    string
	Layouts []string
}

// DateFields는 날짜 필드 셋이다. created는 정확도를 모르는 자료를 위해
// 연월 형식을 함께 허용한다. ADR 0009.
type DateFields struct {
	Created   DateField
	SourcedAt DateField
	Updated   DateField
}

// Schema는 lint가 문서 필드 값을 대조할 허용값 집합이다.
// source_channel과 workflow는 개방 집합이라 여기 담지 않는다.
type Schema struct {
	Types          []string
	ArtifactStages ClosedSet
	Statuses       ClosedSet
	Scopes         ClosedSet
	Sensitivities  ClosedSet
	TriggerModes   ClosedSet
	Dates          DateFields
	Taxonomy       Taxonomy
}

// Config는 한 위키의 확정된 구성이다.
type Config struct {
	Preset     Preset
	Axes       map[Axis]bool
	Schema     Schema
	Thresholds Thresholds
	PageDirs   []string
	RootFiles  []string
	// IgnoreFiles는 page_dirs 아래 어느 깊이에 있든 문서가 아닌 파일로
	// 순회에서 빼는 파일명이다. ADR 0036.
	IgnoreFiles []string
	// UnknownKeys는 설정 파일에 있었지만 스키마가 모르는 키다.
	// 오타를 조용히 삼키지 않기 위해 호출자에게 넘긴다.
	UnknownKeys []string
	// Origins는 키별 값 출처다. 속성 키는 "axes.<속성 이름>" 형태다.
	Origins map[string]Origin
}

// Load는 위키 루트의 engram.yaml을 읽어 구성을 반환한다.
// 파일이 없으면 기본값으로 동작한다.
func Load(wikiRoot string) (Config, error) {
	return LoadFile(filepath.Join(wikiRoot, ConfigFileName))
}

// LoadFile은 설정 파일 하나를 읽어 구성을 반환한다.
func LoadFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("설정 파일을 읽을 수 없음: %w", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Config{}, fmt.Errorf("설정 파일 파싱 실패: %w", err)
	}
	return build(doc)
}

// defaults는 프리셋과 파일 어디에도 없을 때의 기본 구성이다.
func defaults() Config {
	cfg := Config{
		Preset:      DefaultPreset,
		Axes:        presetAxes(DefaultPreset),
		Schema:      defaultSchema(),
		Thresholds:  Thresholds{MinWikilinks: 2, StaleDays: 90, MaxLines: 1000, BroadTopicPct: 25},
		PageDirs:    []string{"inbox", "sources", "context", "archive"},
		RootFiles:   []string{"index.md"},
		IgnoreFiles: []string{"README.md"},
		Origins:     make(map[string]Origin),
	}
	// 프리셋 키 자체는 기본값에서 왔다. 속성의 on/off는 프리셋이 정했다.
	cfg.Origins["preset"] = OriginDefault
	for _, k := range []string{"types", "topics", "forms", "min_wikilinks", "stale_days", "max_lines", "broad_topic_pct", "page_dirs", "root_files", "ignore_files"} {
		cfg.Origins[k] = OriginDefault
	}
	for _, a := range allAxes() {
		cfg.Origins["axes."+string(a)] = OriginPreset
	}
	return cfg
}

// defaultSchema는 허용값의 기본 집합을 반환한다.
// 고정 집합은 upstream 계약에서 왔고 type만 위키별로 덮어쓴다.
func defaultSchema() Schema {
	return Schema{
		Types: []string{
			"concept", "project", "system", "decision", "procedure", "incident",
			"meeting-summary", "agent-workflow", "source-summary", "inbox-note",
		},
		ArtifactStages: ClosedSet{Values: []string{"inbox", "source", "context", "archive"}},
		Statuses:       ClosedSet{Values: []string{"inbox", "sourced", "promoted", "archived", "superseded"}},
		Scopes:         ClosedSet{Values: []string{"work", "personal", "mixed", "unknown"}},
		Sensitivities:  ClosedSet{Values: []string{"public-reference", "internal", "restricted", "private-local-only"}},
		TriggerModes:   ClosedSet{Values: []string{"manual-prompt", "scheduled-automation", "file-drop", "mcp-fetch", "script-import"}},
		Dates: DateFields{
			Created:   DateField{Name: "created", Layouts: []string{"2006-01-02", "2006-01"}},
			SourcedAt: DateField{Name: "sourced_at", Layouts: []string{"2006-01-02"}},
			Updated:   DateField{Name: "updated", Layouts: []string{"2006-01-02"}},
		},
	}
}

// build는 파일 내용을 기본 구성 위에 덮어쓴다.
func build(doc map[string]any) (Config, error) {
	cfg := defaults()
	if doc == nil {
		return cfg, nil
	}

	// preset을 먼저 적용한다. 속성 개별 on/off가 나중에 덮어써야 하므로 순서가 필요하다.
	if raw, ok := doc["preset"]; ok {
		s, ok := raw.(string)
		if !ok {
			return Config{}, fmt.Errorf("preset 값이 문자열이 아님: %v", raw)
		}
		if !knownPreset(Preset(s)) {
			return Config{}, fmt.Errorf("preset 값이 허용값이 아님: %q (허용값: minimal, personal, team)", s)
		}
		cfg.Preset = Preset(s)
		cfg.Axes = presetAxes(cfg.Preset)
		cfg.Origins["preset"] = OriginFile
		for _, a := range allAxes() {
			cfg.Origins["axes."+string(a)] = OriginPreset
		}
		delete(doc, "preset")
	}

	for key, val := range doc {
		switch key {
		case "axes":
			m, ok := val.(map[string]any)
			if !ok {
				return Config{}, fmt.Errorf("axes 값이 매핑이 아님: %v", val)
			}
			for name, on := range m {
				ax := Axis(name)
				if !knownAxis(ax) {
					cfg.UnknownKeys = append(cfg.UnknownKeys, "axes."+name)
					continue
				}
				b, ok := on.(bool)
				if !ok {
					return Config{}, fmt.Errorf("axes.%s 값이 참/거짓이 아님: %v", name, on)
				}
				cfg.Axes[ax] = b
				cfg.Origins["axes."+name] = OriginFile
			}
		case "types":
			list, err := stringListOf("types", val)
			if err != nil {
				return Config{}, err
			}
			cfg.Schema.Types = list
			cfg.Origins["types"] = OriginFile
		case "topics":
			list, err := stringListOf("topics", val)
			if err != nil {
				return Config{}, err
			}
			cfg.Schema.Taxonomy.Topics = OpenSet{Values: list}
			cfg.Origins["topics"] = OriginFile
		case "forms":
			list, err := stringListOf("forms", val)
			if err != nil {
				return Config{}, err
			}
			cfg.Schema.Taxonomy.Forms = ClosedSet{Values: list}
			cfg.Origins["forms"] = OriginFile
		case "min_wikilinks":
			n, err := thresholdOf(key, val, 0)
			if err != nil {
				return Config{}, err
			}
			cfg.Thresholds.MinWikilinks = n
			cfg.Origins[key] = OriginFile
		case "stale_days":
			n, err := thresholdOf(key, val, 1)
			if err != nil {
				return Config{}, err
			}
			cfg.Thresholds.StaleDays = n
			cfg.Origins[key] = OriginFile
		case "max_lines":
			n, err := thresholdOf(key, val, 1)
			if err != nil {
				return Config{}, err
			}
			cfg.Thresholds.MaxLines = n
			cfg.Origins[key] = OriginFile
		case "broad_topic_pct":
			n, err := thresholdOf(key, val, 1)
			if err != nil {
				return Config{}, err
			}
			cfg.Thresholds.BroadTopicPct = n
			cfg.Origins[key] = OriginFile
		case "page_dirs":
			list, err := stringListOf("page_dirs", val)
			if err != nil {
				return Config{}, err
			}
			cfg.PageDirs = list
			cfg.Origins["page_dirs"] = OriginFile
		case "root_files":
			list, err := stringListOf("root_files", val)
			if err != nil {
				return Config{}, err
			}
			cfg.RootFiles = list
			cfg.Origins["root_files"] = OriginFile
		case "ignore_files":
			list, err := stringListOf("ignore_files", val)
			if err != nil {
				return Config{}, err
			}
			cfg.IgnoreFiles = list
			cfg.Origins["ignore_files"] = OriginFile
		default:
			cfg.UnknownKeys = append(cfg.UnknownKeys, key)
		}
	}
	sort.Strings(cfg.UnknownKeys)
	return cfg, nil
}

// stringListOf는 YAML 값을 문자열 슬라이스로 변환한다.
func stringListOf(key string, v any) ([]string, error) {
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s 값이 문자열 목록이 아님: %v", key, v)
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		s, ok := it.(string)
		if !ok {
			return nil, fmt.Errorf("%s 항목이 문자열이 아님: %v", key, it)
		}
		out = append(out, s)
	}
	return out, nil
}

// thresholdOf는 임계값 키를 정수로 변환하고 하한을 검사한다.
// min은 허용 최솟값이며 0을 주면 게이트 끔을 허용한다.
func thresholdOf(key string, v any, min int) (int, error) {
	n, ok := intValue(v)
	if !ok {
		return 0, fmt.Errorf("%s 값이 정수가 아님: %v", key, v)
	}
	if n < min {
		return 0, fmt.Errorf("%s 값이 허용 범위 밖임: %d (최소 %d)", key, n, min)
	}
	return n, nil
}

// intValue는 YAML 파서가 내놓을 수 있는 정수 계열을 int로 통일한다.
func intValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case uint64:
		return int(n), true
	case float64:
		return int(n), n == float64(int(n))
	}
	return 0, false
}
