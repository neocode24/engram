package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/neocode24/engram/internal/status"
	"github.com/spf13/cobra"
)

// newStatusCmd는 위키 현황과 inbox 적체 압력을 보여주는 status 커맨드를 반환한다.
// status 는 진단이지 판정이 아니다. 정상 종료 코드는 항상 0 이고 판정은 lint 가 한다.
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [경로]",
		Short: "위키 현황과 inbox 적체 압력을 보여줍니다",
		Long: `대상 경로의 위키 현황과 inbox 적체 압력을 보여줍니다. 경로를 생략하면 현재 디렉토리입니다.

현황, 적체 압력, 다음 행동 세 구획으로 나눠 냅니다.
나이 계산의 기준 시각은 전역 --now 다.

위키가 아닌 디렉토리에서는 거절하고 engram init 을 안내합니다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
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
	return cmd
}

// printStatusText는 사람용 보고를 인쇄한다. 구획을 나누고 표는 쓰지 않는다.
func printStatusText(w io.Writer, res status.Result, now time.Time) {
	st := res.Stages
	fmt.Fprintf(w, "현황\n")
	fmt.Fprintf(w, "  inbox %d, source %d, context %d, archive %d 문서\n",
		st.Inbox, st.Source, st.Context, st.Archive)
	fmt.Fprintf(w, "  위키링크 %d개, 고아 문서 %d개\n", res.Links, res.Orphans)
	fmt.Fprintf(w, "  lint: 파일 %d개, error %d, warn %d, reject %d (상세는 engram lint)\n",
		res.Lint.Files, res.Lint.Error, res.Lint.Warn, res.Lint.Reject)

	b := res.Backlog
	fmt.Fprintf(w, "\n적체 압력 (기준 %s)\n", now.Format("2006-01-02"))
	switch {
	case b.OldestDays == nil && b.UnknownAge == 0:
		fmt.Fprintf(w, "  inbox 문서 %d개\n", b.Inbox)
	case b.OldestDays == nil:
		fmt.Fprintf(w, "  inbox 문서 %d개, 가장 오래된 것 알 수 없음\n", b.Inbox)
	default:
		fmt.Fprintf(w, "  inbox 문서 %d개, 가장 오래된 것 %d일\n", b.Inbox, *b.OldestDays)
	}
	if b.UnknownAge > 0 {
		fmt.Fprintf(w, "  날짜를 알 수 없는 inbox 문서 %d개\n", b.UnknownAge)
	}
	if b.Stale > 0 {
		fmt.Fprintf(w, "  stale_days를 넘긴 context 문서 %d개\n", b.Stale)
	}
	fmt.Fprintf(w, "  지금 승급할 수 있는 문서 %d개\n", b.Promotable)

	fmt.Fprintf(w, "\n다음 행동\n")
	if len(res.Suggestions) == 0 {
		fmt.Fprintf(w, "  제안 없음\n")
	}
	for _, s := range res.Suggestions {
		fmt.Fprintf(w, "  - %s\n", s.Action)
		fmt.Fprintf(w, "    %s\n", s.Detail)
	}
}
