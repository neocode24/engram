package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/digest"
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
		Short: "기간 안의 위키 변화를 집계합니다",
		Long: `기간 안의 위키 변화를 집계합니다. 상태를 남기지 않으므로 몇 번을 돌려도 같은 결과가 나옵니다.

--days로 기간을 정합니다. 창은 [기준 시각 - N일, 기준 시각]이고 기준
시각은 전역 --now 다. 신규는 created가 창 안에 있는 문서, 노후는
stale_days를 넘긴 context 문서, 고아는 링크가 0개인 문서입니다.
승급 집계는 promote가 승급 시각을 프론트매터에 남기지 않아 여기에
없습니다.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			days, err := cmd.Flags().GetInt(flagDays)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagDays, err)
			}
			if days < 0 {
				return fmt.Errorf("--%s 값은 0 이상이어야 합니다: %d", flagDays, days)
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
	cmd.Flags().Int(flagDays, 30, "집계 기간(일)")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// printDigest는 사람용 집계를 인쇄한다. 항목별로 건수와 슬러그 목록을
// 낸다. 목록은 상한을 넘으면 앞의 것과 개수로 줄인다.
func printDigest(w io.Writer, res digest.Result) {
	fmt.Fprintf(w, "기간 집계 (%s부터 %s까지, %d일)\n",
		dateOnly(res.Since), dateOnly(res.Until), res.Days)
	fmt.Fprintf(w, "  신규 %d개%s\n", len(res.Created), slugList(res.Created))
	fmt.Fprintf(w, "  노후 %d개%s\n", len(res.Stale), slugList(res.Stale))
	fmt.Fprintf(w, "  고아 %d개%s\n", len(res.Orphans), slugList(res.Orphans))
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
	return ": " + strings.Join(slugs[:max], ", ") + fmt.Sprintf(" 외 %d개", len(slugs)-max)
}

// dateOnly는 RFC3339 시각 문자열에서 날짜까지만 낸다. 사람용 출력에
// 쓴다. Result 의 Since/Until 은 항상 RFC3339 다.
func dateOnly(rfc3339 string) string {
	if len(rfc3339) >= 10 {
		return rfc3339[:10]
	}
	return rfc3339
}
