package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/neocode24/engram/internal/doctor"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// newDoctorCmd는 환경과 위키 설정을 진단하는 doctor 커맨드를 반환한다.
// 진단만 하고 고치지 않는다. ok 가 아닌 항목에는 조치를 함께 출력한다.
func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [경로]",
		Short: i18n.T("cli.doctor.short"),
		Long:  i18n.T("cli.doctor.long"),
		Args:  cobra.MaximumNArgs(1),
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
				return errors.New(i18n.T("cli.doctor.has_fail"))
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
			fmt.Fprintln(w, i18n.T("cli.doctor.fix", f.Fix))
		}
	}
	s := res.Summary
	fmt.Fprintln(w, i18n.T("cli.doctor.summary", s.Items, s.OK, s.Warn, s.Fail, s.Skip))
}
