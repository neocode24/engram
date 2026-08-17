package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/status"
	"github.com/spf13/cobra"
)

// newStatusCmd는 위키 현황과 밀린 것을 보여주는 status 커맨드를 반환한다.
// status 는 진단이지 판정이 아니다. 정상 종료 코드는 항상 0 이고 판정은 lint 가 한다.
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status " + i18n.T("usage.args.path_opt"),
		Short: i18n.T("cli.status.short"),
		Long:  i18n.T("cli.status.long"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := pathOrWikiFlag(cmd, args)
			if err != nil {
				return err
			}
			res, err := status.Run(root, Now(cmd))
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printStatusText(cmd.OutOrStdout(), res, Now(cmd))
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// printStatusText는 사람용 보고를 인쇄한다. 구획을 나누고 표는 쓰지 않는다.
func printStatusText(w io.Writer, res status.Result, now time.Time) {
	st := res.Stages
	fmt.Fprintln(w, i18n.T("cli.status.section_overview"))
	fmt.Fprintln(w, i18n.T("cli.status.overview_counts", st.Inbox, st.Source, st.Context, st.Archive))
	fmt.Fprintln(w, i18n.T("cli.status.overview_links", res.Links, res.Orphans))
	fmt.Fprintln(w, i18n.T("cli.status.overview_lint",
		res.Lint.Files, res.Lint.Error, res.Lint.Warn, res.Lint.Reject))

	b := res.Backlog
	fmt.Fprint(w, "\n"+i18n.T("cli.status.section_backlog", now.Format("2006-01-02"))+"\n")
	switch {
	case b.OldestDays == nil && b.UnknownAge == 0:
		fmt.Fprintln(w, i18n.T("cli.status.backlog_inbox", b.Inbox))
	case b.OldestDays == nil:
		fmt.Fprintln(w, i18n.T("cli.status.backlog_inbox_unknown", b.Inbox))
	case *b.OldestDays == 0:
		fmt.Fprintln(w, i18n.T("cli.status.backlog_inbox_today", b.Inbox))
	default:
		fmt.Fprintln(w, i18n.T("cli.status.backlog_inbox_oldest", b.Inbox, *b.OldestDays))
	}
	if b.UnknownAge > 0 {
		fmt.Fprintln(w, i18n.T("cli.status.backlog_unknown_age", b.UnknownAge))
	}
	if b.Stale > 0 {
		fmt.Fprintln(w, i18n.T("cli.status.backlog_stale", b.Stale))
	}
	fmt.Fprintln(w, i18n.T("cli.status.backlog_promotable", b.Promotable))

	fmt.Fprint(w, "\n"+i18n.T("cli.status.section_next")+"\n")
	if len(res.Suggestions) == 0 {
		fmt.Fprintln(w, i18n.T("cli.status.no_suggestions"))
	}
	for _, s := range res.Suggestions {
		fmt.Fprintf(w, "  - %s\n", s.Action)
		fmt.Fprintf(w, "    %s\n", s.Detail)
	}
}
