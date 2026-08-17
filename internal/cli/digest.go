package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/digest"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// flagDays는 digest 의 기간 플래그 이름이다.
const flagDays = "days"

// newDigestCmd는 기간 안의 변화를 집계하는 digest 커맨드를 반환한다.
// 상태를 남기지 않으므로 같은 기준 시각에는 몇 번을 돌려도 같은 결과가
// 나온다. ADR 0028.
func newDigestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest",
		Short: i18n.T("cli.digest.short"),
		Long:  i18n.T("cli.digest.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			days, err := cmd.Flags().GetInt(flagDays)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.digest.flag_read_fail", flagDays), err)
			}
			if days < 0 {
				return errors.New(i18n.T("cli.digest.days_negative", flagDays, days))
			}
			res, err := digest.Run(root, cfg, Now(cmd), days)
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printDigest(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Int(flagDays, 30, i18n.T("cli.digest.flag_days"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.digest.flag_wiki"))
	return cmd
}

// printDigest는 사람용 집계를 인쇄한다. 항목별로 건수와 슬러그 목록을
// 낸다. 목록은 상한을 넘으면 앞의 것과 개수로 줄인다.
func printDigest(w io.Writer, res digest.Result) {
	fmt.Fprintln(w, i18n.T("cli.digest.header",
		dateOnly(res.Since), dateOnly(res.Until), res.Days))
	fmt.Fprintln(w, i18n.T("cli.digest.created", len(res.Created), slugList(res.Created)))
	fmt.Fprintln(w, i18n.T("cli.digest.stale", len(res.Stale), slugList(res.Stale)))
	fmt.Fprintln(w, i18n.T("cli.digest.orphans", len(res.Orphans), slugList(res.Orphans)))
}

// slugList는 슬러그 목록을 사람용 보기로 낸다. 상한은 10개이고 넘으면
// 앞의 10개와 남은 수로 줄인다. 비었으면 빈 문자열이다.
func slugList(slugs []string) string {
	if len(slugs) == 0 {
		return ""
	}
	max := 10
	if len(slugs) <= max {
		return ": " + strings.Join(slugs, ", ")
	}
	return ": " + strings.Join(slugs[:max], ", ") + i18n.T("cli.digest.more_slugs", len(slugs)-max)
}

// dateOnly는 RFC3339 시각 문자열에서 날짜까지만 낸다. 사람용 출력에
// 쓴다. Result 의 Since/Until 은 항상 RFC3339 다.
func dateOnly(rfc3339 string) string {
	if len(rfc3339) >= 10 {
		return rfc3339[:10]
	}
	return rfc3339
}
