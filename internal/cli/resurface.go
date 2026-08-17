package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/resurface"
	"github.com/spf13/cobra"
)

// newResurfaceCmd는 오래 안 본 context 문서를 다시 꺼내는 resurface
// 커맨드를 반환한다. 재발견 커맨드 중 유일하게 상태를 쓰므로 --now 가
// 결과를 결정한다. ADR 0028.
func newResurfaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resurface",
		Short: i18n.T("cli.resurface.short"),
		Long:  i18n.T("cli.resurface.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.resurface.flag_read_fail", flagLimit), err)
			}
			dryRun, err := cmd.Flags().GetBool(flagDryRun)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.resurface.flag_read_fail", flagDryRun), err)
			}
			res, err := resurface.Run(root, cfg, Now(cmd), limit, dryRun)
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printResurface(cmd.OutOrStdout(), res, dryRun)
			return nil
		},
	}
	cmd.Flags().Int(flagLimit, 5, i18n.T("cli.resurface.flag_limit"))
	cmd.Flags().Bool(flagDryRun, false, i18n.T("cli.resurface.flag_dry_run"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.resurface.flag_wiki"))
	return cmd
}

// printResurface는 사람용 결과를 인쇄한다. 후보마다 슬러그, 제목,
// 경과일, 마지막 제시 시각을 낸다.
func printResurface(w io.Writer, res resurface.Result, dryRun bool) {
	if len(res.Candidates) == 0 {
		fmt.Fprintln(w, i18n.T("cli.resurface.no_candidates"))
		fmt.Fprintln(w, i18n.T("cli.resurface.reason", res.Reason))
	} else {
		fmt.Fprintln(w, i18n.T("cli.resurface.header",
			len(res.Candidates), res.StaleDays, res.Now.Format("2006-01-02")))
		for _, c := range res.Candidates {
			last := i18n.T("cli.resurface.never_shown")
			if c.LastShown != nil {
				last = i18n.T("cli.resurface.last_shown", c.LastShown.Format("2006-01-02"))
			}
			title := ""
			if c.Title != "" {
				title = " (" + c.Title + ")"
			}
			fmt.Fprintln(w, i18n.T("cli.resurface.candidate_line", c.Slug, title, c.AgeDays, last))
		}
	}
	if res.SkippedNoDate > 0 {
		fmt.Fprintln(w, i18n.T("cli.resurface.skipped_no_date", res.SkippedNoDate))
	}
	if dryRun {
		fmt.Fprintln(w, i18n.T("cli.resurface.dry_run_note"))
	}
}
