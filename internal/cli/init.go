package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/spf13/cobra"
)

// flagPreset는 init 커맨드의 프리셋 플래그 이름이다.
const flagPreset = "preset"

// gitignoreEntry는 init이 .gitignore에 넣는 유일한 항목이다. ADR 0010.
const gitignoreEntry = ".engram/"

// initResult는 init 결과 요약이다. --json 출력에 그대로 쓰인다.
type initResult struct {
	Root   string   `json:"root"`
	Preset string   `json:"preset"`
	Dirs   []string `json:"dirs"`
	Files  []string `json:"files"`
}

// newInitCmd는 새 위키를 만드는 init 커맨드를 반환한다.
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [경로]",
		Short: "새 위키를 만듭니다",
		Long: `지정한 경로에 새 위키를 만듭니다. 경로를 생략하면 현재 디렉토리입니다.

디렉토리 구성, engram.yaml 설정, 첫 문서 index.md, .gitignore를 만듭니다.
이미 engram.yaml이 있으면 기존 위키를 보존하기 위해 거절합니다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			raw, err := cmd.Flags().GetString(flagPreset)
			if err != nil {
				return err
			}
			preset, err := parsePreset(raw)
			if err != nil {
				return err
			}
			res, err := runInit(dir, preset, Now(cmd))
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printOnboarding(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagPreset, string(config.DefaultPreset),
		"스키마 프리셋. personal, education, team 중 하나")
	return cmd
}

// parsePreset는 프리셋 플래그 값을 검증한다.
func parsePreset(raw string) (config.Preset, error) {
	p := config.Preset(raw)
	switch p {
	case config.PresetPersonal, config.PresetEducation, config.PresetTeam:
		return p, nil
	}
	return "", fmt.Errorf("--preset 값이 허용값이 아님: %q (허용값: personal, education, team)", raw)
}

// runInit는 위키 루트에 초기 파일을 만든다. 기존 파일은 덮어쓰지 않으므로
// 일부만 있던 대상에서도 나머지를 채워 완성한다.
func runInit(dir string, preset config.Preset, now time.Time) (initResult, error) {
	root := filepath.Clean(dir)
	cfgPath := filepath.Join(root, config.ConfigFileName)
	if _, err := os.Stat(cfgPath); err == nil {
		return initResult{}, fmt.Errorf("대상이 이미 engram 위키입니다: %s\n기존 위키를 덮어쓰지 않습니다. 다른 경로를 지정하거나 기존 %s을 손으로 고치세요",
			cfgPath, config.ConfigFileName)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return initResult{}, fmt.Errorf("대상 경로를 확인할 수 없음: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return initResult{}, fmt.Errorf("위키 루트를 만들 수 없음: %w", err)
	}
	if err := writeFileIfAbsent(cfgPath, []byte(configYAML(preset))); err != nil {
		return initResult{}, err
	}
	// 방금 쓴 설정을 다시 읽는다. init이 쓰는 파일이 곧 제품이 읽는 파일이므로
	// 생성물은 항상 이 로드 결과에서 파생한다.
	cfg, err := config.Load(root)
	if err != nil {
		return initResult{}, fmt.Errorf("초기 설정을 읽을 수 없음: %w", err)
	}
	for _, d := range cfg.PageDirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return initResult{}, fmt.Errorf("디렉토리를 만들 수 없음: %s: %w", d, err)
		}
	}
	if err := writeFileIfAbsent(filepath.Join(root, "index.md"), []byte(indexMD(cfg, now))); err != nil {
		return initResult{}, err
	}
	if err := ensureGitignore(root); err != nil {
		return initResult{}, err
	}
	return initResult{
		Root:   root,
		Preset: string(cfg.Preset),
		Dirs:   cfg.PageDirs,
		Files:  []string{config.ConfigFileName, "index.md", ".gitignore"},
	}, nil
}

// writeFileIfAbsent는 경로에 파일이 없을 때만 쓴다. 있으면 그대로 둔다.
func writeFileIfAbsent(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("파일을 만들 수 없음: %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("파일을 쓸 수 없음: %s: %w", path, err)
	}
	return f.Close()
}

// ensureGitignore는 .gitignore에 .engram/ 항목이 없을 때만 덧붙인다.
func ensureGitignore(root string) error {
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return writeFileIfAbsent(path, []byte(gitignoreEntry+"\n"))
	}
	if err != nil {
		return fmt.Errorf(".gitignore을 읽을 수 없음: %w", err)
	}
	if hasLine(data, gitignoreEntry) {
		return nil
	}
	var b strings.Builder
	b.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString(gitignoreEntry)
	b.WriteByte('\n')
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf(".gitignore을 갱신할 수 없음: %w", err)
	}
	return nil
}

// hasLine은 데이터에 해당 줄이 이미 있는지 검사한다.
func hasLine(data []byte, line string) bool {
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

// configYAML은 init이 쓰는 engram.yaml 본문이다. 주석이 무엇을 고칠 수
// 있는지를 알려준다. 임계값과 디렉토리는 기본값을 그대로 박는다.
func configYAML(preset config.Preset) string {
	return fmt.Sprintf(`# engram 위키 설정. 스키마 축, 임계값, 디렉토리 매핑을 정의합니다.
preset: %s

# 스키마 축. 프리셋(personal < education < team)이 시작점이며
# 개별 축을 아래에서 따로 켜고 끌 수 있습니다.
# 사용 가능한 축: type, artifact_stage, status, indexable, tags, source_refs,
# derived_from, related, source_channel, derived_context, scope, sensitivity,
# trigger_mode, workflow
# axes:
#   scope: true

# 문서 종류(type 축의 허용값). 위키에 맞게 추가합니다.
# types: [concept, project, system, decision, procedure, incident,
#   meeting-summary, agent-workflow, source-summary, inbox-note]

# taxonomy. topics는 개방 집합이고 forms는 폐쇄 집합입니다.
# topics: [go, cli]
# forms: [memo, report]

# 임계값. min_wikilinks만 승급 거절 사유이고 나머지는 경고에 쓰입니다.
min_wikilinks: 2    # promote 게이트. 0으로 두면 게이트가 꺼집니다
stale_days: 90      # 재발견 대상 판정 기준 일수
max_lines: 1000     # 문서 길이 경고 상한
broad_topic_pct: 25 # 광범위 주제 비율 경고 상한(퍼센트)

# 문서가 놓이는 디렉토리와 루트에 있어야 하는 파일
page_dirs: [inbox, sources, context, archive]
root_files: [index.md]
`, preset)
}

// indexAxisOrder는 index.md 프론트매터에 축을 놓는 순서다.
func indexAxisOrder() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// indexMD는 첫 문서 index.md 본문을 만든다. 프론트매터에는 켜진 축만
// 나오고 값은 index 문서에 맞게 채운다. 날짜는 기준 시각에서 온다.
func indexMD(cfg config.Config, now time.Time) string {
	values := map[config.Axis]string{
		config.AxisType:           "system",
		config.AxisArtifactStage:  "context",
		config.AxisStatus:         "promoted",
		config.AxisIndexable:      "true",
		config.AxisTags:           "[]",
		config.AxisSourceRefs:     "[]",
		config.AxisDerivedFrom:    "[]",
		config.AxisRelated:        "[]",
		config.AxisSourceChannel:  "manual-prompt",
		config.AxisDerivedContext: "[]",
		config.AxisScope:          "mixed",
		config.AxisSensitivity:    "internal",
		config.AxisTriggerMode:    "manual-prompt",
		config.AxisWorkflow:       "manual",
	}
	var b strings.Builder
	b.WriteString("---\n")
	for _, ax := range indexAxisOrder() {
		if !cfg.Axes[ax] {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", ax, values[ax])
	}
	date := now.Format("2006-01-02")
	fmt.Fprintf(&b, "created: %s\n", date)
	fmt.Fprintf(&b, "sourced_at: %s\n", date)
	fmt.Fprintf(&b, "updated: %s\n", date)
	b.WriteString("---\n\n")
	b.WriteString("# engram 위키\n\n")
	b.WriteString("이 문서는 위키의 첫 문서입니다. 위키를 소개하는 안내로 바꿉니다.\n\n")
	for _, d := range cfg.PageDirs {
		fmt.Fprintf(&b, "- %s/\n", d)
	}
	b.WriteString("\n새 자료는 inbox에 넣고 승급 파이프라인을 따라 context로 옮깁니다.\n")
	return b.String()
}

// printOnboarding은 무엇이 만들어졌고 다음에 무엇을 하면 되는지를 인쇄한다.
func printOnboarding(w io.Writer, res initResult) {
	dirGuide := map[string]string{
		"inbox":   "새 자료가 들어오는 곳",
		"sources": "원본을 보존하는 곳",
		"context": "정리된 문서가 사는 곳",
		"archive": "승급에서 물러난 문서가 가는 곳",
	}
	fileGuide := map[string]string{
		config.ConfigFileName: "위키 설정. 축과 임계값을 여기서 조정하세요",
		"index.md":            "첫 문서. 위키 소개로 채우세요",
		".gitignore":          ".engram/ 캐시 디렉토리를 git에서 제외합니다",
	}
	fmt.Fprintf(w, "위키를 초기화했습니다: %s (프리셋: %s)\n\n", res.Root, res.Preset)
	fmt.Fprintln(w, "디렉토리:")
	for _, d := range res.Dirs {
		guide, ok := dirGuide[d]
		if !ok {
			guide = "문서 디렉토리"
		}
		fmt.Fprintf(w, "  %-12s %s\n", d+"/", guide)
	}
	fmt.Fprintln(w, "\n파일:")
	for _, f := range res.Files {
		fmt.Fprintf(w, "  %-12s %s\n", f, fileGuide[f])
	}
	fmt.Fprintln(w, "\n다음 단계:")
	fmt.Fprintln(w, "  1. inbox에 첫 자료를 넣으세요")
	fmt.Fprintf(w, "  2. %s을 열어 축과 임계값을 위키에 맞게 조정하세요\n", config.ConfigFileName)
	fmt.Fprintln(w, "  3. index.md를 위키 소개로 채우세요")
}
