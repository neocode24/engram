package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeConfig는 임시 디렉토리에 설정 파일을 쓰고 그 경로를 반환한다.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("설정 파일 작성 실패: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string // 빈 문자열이면 파일을 만들지 않는다
		check   func(t *testing.T, c Config)
		wantErr string // 빈 문자열이면 에러 없음
	}{
		{
			name: "설정 파일이 없으면 기본값으로 동작한다",
			check: func(t *testing.T, c Config) {
				if c.Preset != DefaultPreset {
					t.Errorf("프리셋 = %q, want %q", c.Preset, DefaultPreset)
				}
				if want := (Thresholds{MinWikilinks: 2, StaleDays: 30, MaxLines: 1000, BroadTopicPct: 25, BridgeWordMin: DefaultBridgeWordMin, BridgeEmbedMin: DefaultBridgeEmbedMin}); c.Thresholds != want {
					t.Errorf("임계값 = %+v, want %+v", c.Thresholds, want)
				}
				if want := []string{"inbox", "sources", "context", "archive"}; !reflect.DeepEqual(c.PageDirs, want) {
					t.Errorf("page_dirs = %v, want %v", c.PageDirs, want)
				}
				if want := []string{"index.md"}; !reflect.DeepEqual(c.RootFiles, want) {
					t.Errorf("root_files = %v, want %v", c.RootFiles, want)
				}
				if want := []string{"README.md"}; !reflect.DeepEqual(c.IgnoreFiles, want) {
					t.Errorf("ignore_files = %v, want %v", c.IgnoreFiles, want)
				}
				if len(c.UnknownKeys) != 0 {
					t.Errorf("미정의 키가 있음: %v", c.UnknownKeys)
				}
			},
		},
		{
			name: "personal 프리셋은 하위 축만 켠다",
			yaml: "preset: minimal",
			check: func(t *testing.T, c Config) {
				off := []Axis{AxisSourceChannel, AxisDerivedContext, AxisScope, AxisSensitivity, AxisTriggerMode, AxisWorkflow}
				for _, a := range off {
					if c.Axes[a] {
						t.Errorf("축 %s는 꺼져 있어야 함", a)
					}
				}
				if !c.Axes[AxisType] || !c.Axes[AxisRelated] {
					t.Error("type과 related는 켜져 있어야 함")
				}
			},
		},
		{
			name: "personal 프리셋은 source_channel과 derived_context를 추가로 켠다",
			yaml: "preset: personal",
			check: func(t *testing.T, c Config) {
				if !c.Axes[AxisSourceChannel] || !c.Axes[AxisDerivedContext] {
					t.Error("source_channel과 derived_context는 켜져 있어야 함")
				}
				if c.Axes[AxisScope] || c.Axes[AxisWorkflow] {
					t.Error("scope와 workflow는 꺼져 있어야 함")
				}
			},
		},
		{
			name: "team 프리셋은 14축 전부를 켠다",
			yaml: "preset: team",
			check: func(t *testing.T, c Config) {
				for _, a := range allAxes() {
					if !c.Axes[a] {
						t.Errorf("축 %s가 꺼져 있음", a)
					}
				}
			},
		},
		{
			name: "파일 병합이 기본값을 덮고 없는 키는 기본값이 남는다",
			yaml: "min_wikilinks: 5\nstale_days: 60\n",
			check: func(t *testing.T, c Config) {
				if c.Thresholds.MinWikilinks != 5 || c.Thresholds.StaleDays != 60 {
					t.Errorf("임계값 = %+v, want min_wikilinks 5, stale_days 60", c.Thresholds)
				}
				if c.Thresholds.MaxLines != 1000 || c.Thresholds.BroadTopicPct != 25 {
					t.Errorf("파일에 없는 임계값이 기본값이 아님: %+v", c.Thresholds)
				}
			},
		},
		{
			name: "bridge 하한 둘은 각각 덮어쓴다",
			yaml: "bridge_word_min: 0.18\nbridge_embed_min: 0.75\n",
			check: func(t *testing.T, c Config) {
				if c.Thresholds.BridgeWordMin != 0.18 || c.Thresholds.BridgeEmbedMin != 0.75 {
					t.Errorf("bridge 하한 = %+v, want 0.18, 0.75", c.Thresholds)
				}
				if c.Origins["bridge_word_min"] != OriginFile || c.Origins["bridge_embed_min"] != OriginFile {
					t.Errorf("출처 = %+v", c.Origins)
				}
			},
		},
		{
			name:    "bridge 하한은 코사인 범위 밖이면 거절한다",
			yaml:    "bridge_embed_min: 1.5\n",
			wantErr: "범위 밖",
		},
		{
			name:    "bridge 하한은 실수가 아니면 거절한다",
			yaml:    "bridge_word_min: 많음\n",
			wantErr: "실수가 아님",
		},
		{
			name: "프리셋 위에서 축을 개별적으로 켠다",
			yaml: "preset: minimal\naxes:\n  scope: true\n",
			check: func(t *testing.T, c Config) {
				if !c.Axes[AxisScope] {
					t.Error("scope를 켰는데 꺼져 있음")
				}
				if c.Axes[AxisSensitivity] {
					t.Error("sensitivity는 여전히 꺼져 있어야 함")
				}
			},
		},
		{
			name: "알 수 없는 키는 에러가 아니라 수집한다",
			yaml: "min_wiklink: 3\naxes:\n  scame: true\n",
			check: func(t *testing.T, c Config) {
				want := []string{"axes.scame", "min_wiklink"}
				if !reflect.DeepEqual(c.UnknownKeys, want) {
					t.Errorf("미정의 키 = %v, want %v", c.UnknownKeys, want)
				}
				if c.Thresholds.MinWikilinks != 2 {
					t.Error("오타 키는 기본값을 건드리지 않아야 함")
				}
			},
		},
		{
			name:    "잘못된 프리셋은 허용값과 함께 거절한다",
			yaml:    "preset: hobby\n",
			wantErr: "minimal, personal, team",
		},
		{
			name:    "min_wikilinks 음수는 거절한다",
			yaml:    "min_wikilinks: -1\n",
			wantErr: "min_wikilinks",
		},
		{
			name: "min_wikilinks 0은 게이트 끔으로 허용한다",
			yaml: "min_wikilinks: 0\n",
			check: func(t *testing.T, c Config) {
				if c.Thresholds.MinWikilinks != 0 {
					t.Errorf("min_wikilinks = %d, want 0", c.Thresholds.MinWikilinks)
				}
			},
		},
		{
			name: "types topics forms page_dirs를 파일이 정의한다",
			yaml: "types: [note]\ntopics: [go, cli]\nforms: [memo]\npage_dirs: [in, out]\n",
			check: func(t *testing.T, c Config) {
				if want := []string{"note"}; !reflect.DeepEqual(c.Schema.Types, want) {
					t.Errorf("types = %v, want %v", c.Schema.Types, want)
				}
				if want := []string{"go", "cli"}; !reflect.DeepEqual(c.Schema.Taxonomy.Topics.Values, want) {
					t.Errorf("topics = %v, want %v", c.Schema.Taxonomy.Topics.Values, want)
				}
				if want := []string{"memo"}; !reflect.DeepEqual(c.Schema.Taxonomy.Forms.Values, want) {
					t.Errorf("forms = %v, want %v", c.Schema.Taxonomy.Forms.Values, want)
				}
				if want := []string{"in", "out"}; !reflect.DeepEqual(c.PageDirs, want) {
					t.Errorf("page_dirs = %v, want %v", c.PageDirs, want)
				}
			},
		},
		{
			name: "ignore_files 를 파일이 정의하고 비울 수 있다",
			yaml: "ignore_files: []\n",
			check: func(t *testing.T, c Config) {
				if len(c.IgnoreFiles) != 0 {
					t.Errorf("ignore_files = %v, want 빈 목록", c.IgnoreFiles)
				}
				if len(c.UnknownKeys) != 0 {
					t.Errorf("ignore_files 가 미정의 키로 수집됨: %v", c.UnknownKeys)
				}
				if got := c.Origins["ignore_files"]; got != OriginFile {
					t.Errorf("ignore_files 출처 = %q, want file", got)
				}
			},
		},
		{
			name:    "ignore_files 값이 목록이 아니면 거절한다",
			yaml:    "ignore_files: README.md\n",
			wantErr: "ignore_files",
		},
		{
			name: "값의 출처를 기본값 프리셋 파일로 구분해 추적한다",
			yaml: "preset: team\nmin_wikilinks: 4\naxes:\n  scope: false\n",
			check: func(t *testing.T, c Config) {
				if got := c.Origins["preset"]; got != OriginFile {
					t.Errorf("preset 출처 = %q, want file", got)
				}
				if got := c.Origins["min_wikilinks"]; got != OriginFile {
					t.Errorf("min_wikilinks 출처 = %q, want file", got)
				}
				if got := c.Origins["axes.scope"]; got != OriginFile {
					t.Errorf("axes.scope 출처 = %q, want file", got)
				}
				if got := c.Origins["axes.type"]; got != OriginPreset {
					t.Errorf("axes.type 출처 = %q, want preset", got)
				}
				if got := c.Origins["stale_days"]; got != OriginDefault {
					t.Errorf("stale_days 출처 = %q, want default", got)
				}
				if got := c.Origins["ignore_files"]; got != OriginDefault {
					t.Errorf("ignore_files 출처 = %q, want default", got)
				}
			},
		},
		{
			name: "created는 연월 형식을 함께 허용한다",
			check: func(t *testing.T, c Config) {
				layouts := c.Schema.Dates.Created.Layouts
				if len(layouts) != 2 || layouts[0] != "2006-01-02" || layouts[1] != "2006-01" {
					t.Errorf("created 형식 = %v, want [2006-01-02 2006-01]", layouts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Config
			var err error
			if tt.yaml == "" {
				c, err = Load(t.TempDir())
			} else {
				c, err = LoadFile(writeConfig(t, tt.yaml))
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("에러가 없음. want %q 포함 에러", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("에러 메시지 = %q, want %q 포함", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("에러 없이 로드되어야 함: %v", err)
			}
			tt.check(t, c)
		})
	}
}

// TestDefaultStaleDays는 노후 기준일 기본값을 못 박는다. 실운영 위키
// 308문서 측정에서 90이면 resurface 후보가 0건이었다(ADR 0067).
func TestDefaultStaleDays(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("에러 없이 로드되어야 함: %v", err)
	}
	if c.Thresholds.StaleDays != 30 {
		t.Errorf("stale_days 기본값 = %d, want 30", c.Thresholds.StaleDays)
	}
}
