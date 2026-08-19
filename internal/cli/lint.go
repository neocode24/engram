package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// flagIncludeInbox는 lint 커맨드의 inbox 범위 플래그 이름이다.
const flagIncludeInbox = "include-inbox"

// newLintCmd는 위키의 스키마와 링크 무결성을 검사하는 lint 커맨드를 반환한다.
func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint " + i18n.T("usage.args.path_opt"),
		Short: i18n.T("cli.lint.short"),
		Long:  i18n.T("cli.lint.long"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := pathOrWikiFlag(cmd, args)
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.lint.config_read_fail"), err)
			}
			includeInbox, err := cmd.Flags().GetBool(flagIncludeInbox)
			if err != nil {
				return err
			}
			res, err := lint.Run(root, cfg, lint.Options{IncludeInbox: includeInbox})
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					return err
				}
			} else {
				printLintText(cmd.OutOrStdout(), res)
			}
			if res.HasBlocking() {
				// 위반은 이미 보고했으므로 에러 문자열은 다시 인쇄하지 않는다.
				// 종료 코드 1만 내는 것이 목적이다.
				cmd.SilenceErrors = true
				return errors.New(i18n.T("cli.lint.blocking_violation"))
			}
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	cmd.Flags().Bool(flagIncludeInbox, false, i18n.T("cli.lint.flag_include_inbox"))
	return cmd
}

// maxListedPaths는 위키 단위 진단에서 문서 목록을 화면에 보이는 상한이다.
const maxListedPaths = 5

// printLintText는 사람용 보고를 인쇄한다. 파일별로 묶고 등급을 앞에 둔다.
// 위키 단위 진단은 파일 묶음 바깥에 따로 둔다.
func printLintText(w io.Writer, res lint.Result) {
	current := ""
	for _, v := range res.Violations {
		if v.Path != current {
			current = v.Path
			fmt.Fprintf(w, "%s\n", v.Path)
		}
		fmt.Fprintf(w, "  [%s] %d %s\n", v.Severity, v.Line, v.Rule)
		fmt.Fprintf(w, "    %s\n", v.Message)
		fmt.Fprintln(w, i18n.T("cli.lint.fix", v.Fix))
	}
	if len(res.WikiFindings) > 0 {
		fmt.Fprintln(w, i18n.T("cli.lint.wiki_findings_header"))
		for _, f := range res.WikiFindings {
			fmt.Fprintln(w, i18n.T("cli.lint.wiki_finding_line", f.Severity, f.Rule, f.Topic))
			fmt.Fprintln(w, i18n.T("cli.lint.wiki_finding_ratio", f.Percent, len(f.Paths), f.Total, f.Threshold))
			fmt.Fprintln(w, i18n.T("cli.lint.wiki_finding_paths", previewPaths(f.Paths)))
			fmt.Fprintln(w, i18n.T("cli.lint.fix", f.Fix))
		}
	}
	s := res.Summary
	if len(res.Violations) == 0 && len(res.WikiFindings) == 0 {
		if s.SkippedInbox > 0 {
			fmt.Fprintln(w, i18n.T("cli.lint.summary_clean_inbox_skipped", s.Files, s.SkippedInbox))
		} else {
			fmt.Fprintln(w, i18n.T("cli.lint.summary_clean", s.Files))
		}
		return
	}
	if s.SkippedInbox > 0 {
		fmt.Fprintln(w, i18n.T("cli.lint.summary_inbox_skipped", s.Files, s.SkippedInbox, s.Error, s.Warn, s.Reject))
	} else {
		fmt.Fprintln(w, i18n.T("cli.lint.summary", s.Files, s.Error, s.Warn, s.Reject))
	}
}

// previewPaths는 문서 목록을 보기 좋게 줄인다. 많으면 앞의 몇 개만 보이고
// 나머지는 개수로 남긴다.
func previewPaths(paths []string) string {
	if len(paths) <= maxListedPaths {
		return strings.Join(paths, ", ")
	}
	return strings.Join(paths[:maxListedPaths], ", ") + i18n.T("cli.lint.more_paths", len(paths)-maxListedPaths)
}
