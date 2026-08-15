package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/neocode24/engram/internal/doctor"
	"github.com/spf13/cobra"
)

// newDoctorCmd는 환경과 위키 설정을 진단하는 doctor 커맨드를 반환한다.
// 진단만 하고 고치지 않는다. ok 가 아닌 항목에는 조치를 함께 출력한다.
func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [경로]",
		Short: "환경과 위키 설정을 진단합니다",
		Long: `대상 경로의 환경과 위키 설정을 진단합니다. 경로를 생략하면 현재 디렉토리입니다.

위키가 아닌 디렉토리에서는 환경 항목만 검사합니다.
각 항목은 상태(ok, warn, fail, skip)와 관측값을 한 줄로 출력하고
ok 가 아닌 항목에는 조치를 이어서 출력합니다.

fail 항목이 하나라도 있으면 종료 코드 1로 끝납니다. warn은 0입니다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			res := doctor.Run(root)
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					return err
				}
			} else {
				printDoctorText(cmd.OutOrStdout(), res)
			}
			if res.HasFail() {
				// 진단은 이미 보고했으므로 에러 문자열은 다시 인쇄하지 않는다.
				// 종료 코드 1만 내는 것이 목적이다.
				cmd.SilenceErrors = true
				return errors.New("진단 실패 항목이 있습니다")
			}
			return nil
		},
	}
	return cmd
}

// printDoctorText는 항목별 상태와 조치, 마지막에 요약을 인쇄한다.
func printDoctorText(w io.Writer, res doctor.Result) {
	for _, f := range res.Findings {
		fmt.Fprintf(w, "[%s] %s %s\n", f.Status, f.ID, f.Detail)
		if f.Status != doctor.StatusOK && f.Fix != "" {
			fmt.Fprintf(w, "    조치: %s\n", f.Fix)
		}
	}
	s := res.Summary
	fmt.Fprintf(w, "항목 %d개, ok %d, warn %d, fail %d, skip %d\n",
		s.Items, s.OK, s.Warn, s.Fail, s.Skip)
}
