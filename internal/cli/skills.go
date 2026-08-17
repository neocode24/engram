package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/skills"
	"github.com/spf13/cobra"
)

// skills 커맨드의 플래그 이름. 다른 커맨드의 force, dry-run 과 값이
// 같지만 소속을 밝히는 이름을 쓴다.
const (
	flagSkillsDir    = "dir"
	flagSkillsForce  = "force"
	flagSkillsDryRun = "dry-run"
)

// newSkillsCmd는 에이전트 스킬을 다루는 skills 커맨드를 반환한다.
// 하위 커맨드 없이 치면 사용법을 낸다.
func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: i18n.T("cli.skills.short"),
		Long:  i18n.T("cli.skills.long"),
	}
	cmd.AddCommand(newSkillsInstallCmd())
	return cmd
}

// newSkillsInstallCmd는 임베드된 스킬 문서를 에이전트 스킬 디렉토리에
// 심는 install 커맨드를 반환한다.
func newSkillsInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: i18n.T("cli.skills.install_short"),
		Long:  i18n.T("cli.skills.install_long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirFlag, err := stringFlag(cmd, flagSkillsDir)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool(flagSkillsForce)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.skills.flag_read_fail", flagSkillsForce), err)
			}
			dryRun, err := cmd.Flags().GetBool(flagSkillsDryRun)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.skills.flag_read_fail", flagSkillsDryRun), err)
			}

			var roots []string
			if dirFlag != "" {
				info, err := os.Stat(dirFlag)
				if err != nil || !info.IsDir() {
					return fmt.Errorf("%s", i18n.T("cli.skills.dir_not_dir", dirFlag))
				}
				roots = []string{dirFlag}
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("%s: %w\n%s", i18n.T("cli.skills.home_fail"), err, i18n.T("cli.skills.home_fail_hint"))
				}
				roots = skills.Detect(home)
				if len(roots) == 0 {
					return fmt.Errorf("%s", i18n.T("cli.skills.detect_fail", strings.Join(skills.Sources(), "\n  ")))
				}
			}

			paths := make([]string, 0, len(roots))
			for _, root := range roots {
				paths = append(paths, skills.InstallPath(root))
			}

			// 충돌을 전부 찾은 뒤에야 쓰기 시작한다. 일부만 심고 멈추면
			// 어느 에이전트에만 스킬이 들어간 어중간한 상태가 된다.
			var conflicts []string
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					conflicts = append(conflicts, p)
				}
			}
			if len(conflicts) > 0 && !force {
				res := skillsOutcome{Conflicts: conflicts}
				if jsonOutput(cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				printSkillsConflicts(cmd.ErrOrStderr(), res)
				return errors.New(i18n.T("cli.skills.conflict"))
			}

			var written, overwritten []string
			if !dryRun {
				for _, p := range paths {
					if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
						return fmt.Errorf("%s: %w", i18n.T("cli.skills.dir_mkdir_fail"), err)
					}
					if err := os.WriteFile(p, []byte(skills.Doc()), 0o644); err != nil {
						return fmt.Errorf("%s: %w", i18n.T("cli.skills.doc_write_fail", p), err)
					}
					written = append(written, p)
					if containsString(conflicts, p) {
						overwritten = append(overwritten, p)
					}
				}
			}

			res := skillsOutcome{
				DryRun:      dryRun,
				Force:       force,
				Files:       paths,
				Written:     written,
				Overwritten: overwritten,
				Conflicts:   conflicts,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printSkillsInstall(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagSkillsDir, "", i18n.T("cli.skills.flag_dir"))
	cmd.Flags().Bool(flagSkillsForce, false, i18n.T("cli.skills.flag_force"))
	cmd.Flags().Bool(flagSkillsDryRun, false, i18n.T("cli.skills.flag_dry_run"))
	return cmd
}

// skillsOutcome은 skills install의 결과다. DryRun 이면 Files 는 심을
// 예정, 아니면 심은 목록이다.
type skillsOutcome struct {
	DryRun      bool     `json:"dryRun"`
	Force       bool     `json:"force"`
	Files       []string `json:"files"`
	Written     []string `json:"written,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

// printSkillsInstall는 심은 목록과 재시작 안내를 낸다.
func printSkillsInstall(w io.Writer, res skillsOutcome) {
	if res.DryRun {
		fmt.Fprint(w, i18n.T("cli.skills.dry_run", len(res.Files))+"\n")
	} else {
		fmt.Fprint(w, i18n.T("cli.skills.done", len(res.Files))+"\n")
	}
	for _, p := range res.Files {
		fmt.Fprintf(w, "  %s\n", p)
	}
	if len(res.Overwritten) > 0 {
		fmt.Fprint(w, i18n.T("cli.skills.overwritten", len(res.Overwritten))+"\n")
		for _, p := range res.Overwritten {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	fmt.Fprint(w, i18n.T("cli.skills.restart_note")+"\n")
}

// printSkillsConflicts는 충돌 목록을 알린다.
func printSkillsConflicts(w io.Writer, res skillsOutcome) {
	fmt.Fprint(w, i18n.T("cli.skills.conflicts_count", len(res.Conflicts))+"\n")
	for _, p := range res.Conflicts {
		fmt.Fprintf(w, "  %s\n", p)
	}
}
