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
	"github.com/neocode24/engram/internal/i18n"
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
		Short: i18n.T("cli.init.short"),
		Long:  i18n.T("cli.init.long"),
		Args:  cobra.MaximumNArgs(1),
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
		i18n.T("cli.init.flag_preset"))
	return cmd
}

// parsePreset는 프리셋 플래그 값을 검증한다.
func parsePreset(raw string) (config.Preset, error) {
	p := config.Preset(raw)
	switch p {
	case config.PresetMinimal, config.PresetPersonal, config.PresetTeam:
		return p, nil
	}
	return "", fmt.Errorf("%s", i18n.T("cli.init.preset_invalid", raw))
}

// runInit는 위키 루트에 초기 파일을 만든다. 기존 파일은 덮어쓰지 않으므로
// 일부만 있던 대상에서도 나머지를 채워 완성한다.
func runInit(dir string, preset config.Preset, now time.Time) (initResult, error) {
	root := filepath.Clean(dir)
	cfgPath := filepath.Join(root, config.ConfigFileName)
	if _, err := os.Stat(cfgPath); err == nil {
		return initResult{}, fmt.Errorf("%s", i18n.T("cli.init.already_wiki", cfgPath, config.ConfigFileName))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.path_check_fail"), err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.root_mkdir_fail"), err)
	}
	if err := writeFileIfAbsent(cfgPath, []byte(configYAML(preset))); err != nil {
		return initResult{}, err
	}
	// 방금 쓴 설정을 다시 읽는다. init이 쓰는 파일이 곧 제품이 읽는 파일이므로
	// 생성물은 항상 이 로드 결과에서 파생한다.
	cfg, err := config.Load(root)
	if err != nil {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.config_load_fail"), err)
	}
	for _, d := range cfg.PageDirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.dir_mkdir_fail", d), err)
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
		return fmt.Errorf("%s: %w", i18n.T("cli.init.file_create_fail", path), err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("%s: %w", i18n.T("cli.init.file_write_fail", path), err)
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
		return fmt.Errorf("%s: %w", i18n.T("cli.init.gitignore_read_fail"), err)
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
		return fmt.Errorf("%s: %w", i18n.T("cli.init.gitignore_write_fail"), err)
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
	return i18n.T("cli.init.config_yaml", preset)
}

// indexAxisOrder는 index.md 프론트매터에 속성을 놓는 순서다.
func indexAxisOrder() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// indexMD는 첫 문서 index.md 본문을 만든다. 프론트매터에는 켜진 속성만
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
	b.WriteString(i18n.T("cli.init.index_title") + "\n\n")
	b.WriteString(i18n.T("cli.init.index_intro") + "\n\n")
	for _, d := range cfg.PageDirs {
		fmt.Fprintf(&b, "- %s/\n", d)
	}
	b.WriteString("\n" + i18n.T("cli.init.index_guide") + "\n")
	return b.String()
}

// printOnboarding은 무엇이 만들어졌고 다음에 무엇을 하면 되는지를 인쇄한다.
func printOnboarding(w io.Writer, res initResult) {
	dirGuide := map[string]string{
		"inbox":   i18n.T("cli.init.dir_inbox"),
		"sources": i18n.T("cli.init.dir_sources"),
		"context": i18n.T("cli.init.dir_context"),
		"archive": i18n.T("cli.init.dir_archive"),
	}
	fileGuide := map[string]string{
		config.ConfigFileName: i18n.T("cli.init.file_config"),
		"index.md":            i18n.T("cli.init.file_index"),
		".gitignore":          i18n.T("cli.init.file_gitignore"),
	}
	fmt.Fprint(w, i18n.T("cli.init.done", res.Root, res.Preset)+"\n\n")
	fmt.Fprintln(w, i18n.T("cli.init.dirs_header"))
	for _, d := range res.Dirs {
		guide, ok := dirGuide[d]
		if !ok {
			guide = i18n.T("cli.init.dir_other")
		}
		fmt.Fprintf(w, "  %-12s %s\n", d+"/", guide)
	}
	fmt.Fprintln(w, "\n"+i18n.T("cli.init.files_header"))
	for _, f := range res.Files {
		fmt.Fprintf(w, "  %-12s %s\n", f, fileGuide[f])
	}
	fmt.Fprintln(w, "\n"+i18n.T("cli.init.next_header"))
	fmt.Fprintln(w, "  1. "+i18n.T("cli.init.step_inbox"))
	fmt.Fprint(w, "  2. "+i18n.T("cli.init.step_config", config.ConfigFileName)+"\n")
	fmt.Fprintln(w, "  3. "+i18n.T("cli.init.step_fill_index"))
}
