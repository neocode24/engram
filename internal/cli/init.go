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
	"github.com/neocode24/engram/internal/pin"
	"github.com/spf13/cobra"
)

// flagPreset는 init 커맨드의 프리셋 플래그 이름이다.
const flagPreset = "preset"

// gitignoreEntry는 init이 .gitignore에 넣는 유일한 항목이다. ADR 0010.
const gitignoreEntry = ".engram/"

// initResult는 init 결과 요약이다. --json 출력에 그대로 쓰인다.
//
// Fresh 가 거짓이면 이미 있던 위키에 다시 실행한 것이다. 그때 Created 는 이번에 채운 것이고 Existing 은
// 이미 있어서 바꾸지 않은 것이다. PresetKept 는 다른 프리셋을 달라고 했으나 바꾸지 않았을 때의 현재 프리셋이다.
type initResult struct {
	Root       string   `json:"root"`
	Preset     string   `json:"preset"`
	Dirs       []string `json:"dirs"`
	Files      []string `json:"files"`
	Fresh      bool     `json:"fresh"`
	Created    []string `json:"created"`
	Existing   []string `json:"existing"`
	PresetKept string   `json:"preset_kept,omitempty"`
}

// newInitCmd는 위키를 만들거나 이미 있는 위키의 빠진 것을 채우는 init 커맨드를 반환한다.
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init " + i18n.T("usage.args.path_opt"),
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
			res, err := runInit(dir, preset, cmd.Flags().Changed(flagPreset), Now(cmd))
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

// runInit는 위키 루트에 초기 파일을 만든다. 이미 위키(engram.yaml 이 있음)이면 거절하지 않고 빠진 것만
// 채운다. 대상은 pin/, 설정이 요구하는 디렉토리, .gitignore 의 .engram/ 항목이다. 있는 파일은
// 바꾸지 않는다. 프리셋은 새로 만들 때만 뜻이 있고 presetGiven 이 참이어도 있는 위키의 프리셋은
// 바꾸지 않는다(바꾸려면 migrate). ADR 0108.
func runInit(dir string, preset config.Preset, presetGiven bool, now time.Time) (initResult, error) {
	root := filepath.Clean(dir)
	cfgPath := filepath.Join(root, config.ConfigFileName)
	fresh := false
	if _, err := os.Stat(cfgPath); errors.Is(err, fs.ErrNotExist) {
		fresh = true
	} else if err != nil {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.path_check_fail"), err)
	}
	var res initResult
	if fresh {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.root_mkdir_fail"), err)
		}
		if err := writeFileIfAbsent(cfgPath, []byte(configYAML(preset))); err != nil {
			return initResult{}, err
		}
		res.Created = append(res.Created, config.ConfigFileName)
	} else {
		res.Existing = append(res.Existing, config.ConfigFileName)
	}
	// 방금 쓴(또는 이미 있는) 설정을 읽는다. init이 쓰는 파일이 곧 제품이 읽는 파일이므로
	// 생성물은 항상 이 로드 결과에서 파생한다.
	cfg, err := config.Load(root)
	if err != nil {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.config_load_fail"), err)
	}
	if !fresh && presetGiven && string(cfg.Preset) != string(preset) {
		res.PresetKept = string(cfg.Preset)
	}
	for _, d := range cfg.PageDirs {
		full := filepath.Join(root, d)
		if _, err := os.Stat(full); err == nil {
			res.Existing = append(res.Existing, d+"/")
			continue
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.dir_mkdir_fail", d), err)
		}
		res.Created = append(res.Created, d+"/")
	}
	// 있는 위키의 index.md 는 채우지 않는다. 없으면 doctor 가 알린다.
	if fresh {
		if err := writeFileIfAbsent(filepath.Join(root, "index.md"), []byte(indexMD(cfg, now))); err != nil {
			return initResult{}, err
		}
		res.Created = append(res.Created, "index.md")
	} else if _, err := os.Stat(filepath.Join(root, "index.md")); err == nil {
		res.Existing = append(res.Existing, "index.md")
	}
	changed, err := ensureGitignore(root)
	if err != nil {
		return initResult{}, err
	}
	if changed {
		res.Created = append(res.Created, ".gitignore ("+gitignoreEntry+")")
	} else {
		res.Existing = append(res.Existing, ".gitignore ("+gitignoreEntry+")")
	}
	hadPin := pin.Present(root)
	made, err := pin.Init(root)
	if err != nil {
		return initResult{}, fmt.Errorf("%s: %w", i18n.T("cli.init.pin_fail"), err)
	}
	switch {
	case !hadPin:
		res.Created = append(res.Created, "pin/")
	case len(made) > 0:
		res.Created = append(res.Created, "pin/ ("+strings.Join(made, ", ")+")")
	default:
		res.Existing = append(res.Existing, "pin/")
	}
	res.Root, res.Preset, res.Dirs, res.Fresh = root, string(cfg.Preset), cfg.PageDirs, fresh
	res.Files = []string{config.ConfigFileName, "index.md", ".gitignore"}
	if res.Created == nil {
		res.Created = []string{}
	}
	if res.Existing == nil {
		res.Existing = []string{}
	}
	return res, nil
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

// ensureGitignore는 .gitignore에 .engram/ 항목이 없을 때만 덧붙인다. 바꿨으면 참이다.
func ensureGitignore(root string) (bool, error) {
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, writeFileIfAbsent(path, []byte(gitignoreEntry+"\n"))
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("cli.init.gitignore_read_fail"), err)
	}
	if hasLine(data, gitignoreEntry) {
		return false, nil
	}
	var b strings.Builder
	b.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString(gitignoreEntry)
	b.WriteByte('\n')
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("cli.init.gitignore_write_fail"), err)
	}
	return true, nil
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
// 이미 있던 위키에는 채운 것과 이미 있던 것을 가려 적는다.
func printOnboarding(w io.Writer, res initResult) {
	if !res.Fresh {
		printRefill(w, res)
		return
	}
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
	fmt.Fprintf(w, "  %-12s %s\n", "pin/", i18n.T("cli.init.dir_pin"))
	fmt.Fprintln(w, "\n"+i18n.T("cli.init.files_header"))
	for _, f := range res.Files {
		fmt.Fprintf(w, "  %-12s %s\n", f, fileGuide[f])
	}
	fmt.Fprintln(w, "\n"+i18n.T("cli.init.next_header"))
	fmt.Fprintln(w, "  1. "+i18n.T("cli.init.step_inbox"))
	fmt.Fprint(w, "  2. "+i18n.T("cli.init.step_config", config.ConfigFileName)+"\n")
	fmt.Fprintln(w, "  3. "+i18n.T("cli.init.step_fill_index"))
}

// printRefill은 이미 있던 위키에 init 을 다시 실행한 결과를 인쇄한다.
func printRefill(w io.Writer, res initResult) {
	fmt.Fprintln(w, i18n.T("cli.init.refill_head", res.Root, res.Preset))
	if len(res.Created) == 0 {
		fmt.Fprintln(w, i18n.T("cli.init.refill_nothing"))
	} else {
		fmt.Fprintln(w, "\n"+i18n.T("cli.init.refill_created"))
		for _, c := range res.Created {
			fmt.Fprintln(w, "  "+c)
		}
	}
	if len(res.Existing) > 0 {
		fmt.Fprintln(w, "\n"+i18n.T("cli.init.refill_existing"))
		for _, e := range res.Existing {
			fmt.Fprintln(w, "  "+e)
		}
	}
	if res.PresetKept != "" {
		fmt.Fprintln(w, "\n"+i18n.T("cli.init.preset_kept", res.PresetKept))
	}
}
