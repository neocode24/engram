package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/eject"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// eject 커맨드의 플래그 이름.
const (
	flagEjectForce  = "force"
	flagEjectDryRun = "dry-run"
)

// newEjectCmd는 내장 규칙을 사용자가 고칠 수 있는 파일로 내보내는 eject
// 커맨드를 반환한다. 규칙만 사용자 것이 되고 연산은 engram 에 남는다.
func newEjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eject",
		Short: i18n.T("cli.eject.short"),
		Long:  i18n.T("cli.eject.long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool(flagEjectForce)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.eject.flag_read_fail", flagEjectForce), err)
			}
			dryRun, err := cmd.Flags().GetBool(flagEjectDryRun)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.eject.flag_read_fail", flagEjectDryRun), err)
			}

			plan := eject.Plan(cfg)

			// 충돌을 전부 찾은 뒤에야 쓰기 시작한다. 일부만 쓰고 멈추면
			// 위키가 어중간해진다.
			var conflicts []string
			for _, a := range plan {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(a.Path))); err == nil {
					conflicts = append(conflicts, a.Path)
				}
			}
			if len(conflicts) > 0 && !force {
				res := ejectOutcome{Conflicts: conflicts}
				if jsonOutput(cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				printEjectConflicts(cmd.ErrOrStderr(), res)
				return errors.New(i18n.T("cli.eject.conflict"))
			}

			var written, overwritten []string
			if !dryRun {
				for _, a := range plan {
					if err := writeArtifact(root, a); err != nil {
						return err
					}
					written = append(written, a.Path)
					if containsString(conflicts, a.Path) {
						overwritten = append(overwritten, a.Path)
					}
				}
			}

			res := ejectOutcome{
				DryRun:      dryRun,
				Force:       force,
				Files:       planPaths(plan),
				Written:     written,
				Overwritten: overwritten,
				Conflicts:   conflicts,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printEject(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Bool(flagEjectForce, false, i18n.T("cli.eject.flag_force"))
	cmd.Flags().Bool(flagEjectDryRun, false, i18n.T("cli.eject.flag_dry_run"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.eject.flag_wiki"))
	return cmd
}

// ejectOutcome은 eject 의 결과다. DryRun 이면 Files 는 만들 예정,
// 아니면 만든 목록이다.
type ejectOutcome struct {
	DryRun      bool     `json:"dryRun"`
	Force       bool     `json:"force"`
	Files       []string `json:"files"`
	Written     []string `json:"written,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

// planPaths는 산출물 경로만 낸다.
func planPaths(plan []eject.Artifact) []string {
	out := make([]string, 0, len(plan))
	for _, a := range plan {
		out = append(out, a.Path)
	}
	return out
}

// writeArtifact는 산출물 하나를 위키에 쓴다. 훅은 실행 권한이 필요하다.
func writeArtifact(root string, a eject.Artifact) error {
	path := filepath.Join(root, filepath.FromSlash(a.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.eject.dir_mkdir_fail"), err)
	}
	if err := os.WriteFile(path, []byte(a.Content), a.Mode); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.eject.artifact_write_fail", path), err)
	}
	return nil
}

// printEject는 만든 목록과 세 가지 안내를 낸다.
func printEject(w io.Writer, res ejectOutcome) {
	if res.DryRun {
		fmt.Fprint(w, i18n.T("cli.eject.dry_run", len(res.Files))+"\n")
	} else {
		fmt.Fprint(w, i18n.T("cli.eject.done", len(res.Files))+"\n")
	}
	for _, p := range res.Files {
		fmt.Fprintf(w, "  %s\n", p)
	}
	if len(res.Overwritten) > 0 {
		fmt.Fprint(w, i18n.T("cli.eject.overwritten", len(res.Overwritten))+"\n")
		for _, p := range res.Overwritten {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	fmt.Fprint(w, "\n"+i18n.T("cli.eject.guide_header")+"\n")
	fmt.Fprint(w, "  "+i18n.T("cli.eject.hook_enable")+"\n")
	fmt.Fprint(w, "  "+i18n.T("cli.eject.still_works")+"\n")
	fmt.Fprint(w, "  "+i18n.T("cli.eject.python_note")+"\n")
}

// printEjectConflicts는 충돌 목록을 알린다.
func printEjectConflicts(w io.Writer, res ejectOutcome) {
	fmt.Fprint(w, i18n.T("cli.eject.conflicts_count", len(res.Conflicts))+"\n")
	for _, p := range res.Conflicts {
		fmt.Fprintf(w, "  %s\n", p)
	}
}
