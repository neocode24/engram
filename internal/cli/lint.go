package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// newLintCmd는 위키의 스키마와 링크 무결성을 검사하는 lint 커맨드를 반환한다.
func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [경로]",
		Short: "위키의 스키마와 링크 무결성을 검사한다",
		Long: `지정한 경로의 위키를 순회하며 스키마와 링크 무결성을 검사한다.
경로를 생략하면 현재 디렉토리다.

등급은 error, warn, reject 셋이다. error와 reject는 승급을 막아
종료 코드 1로 끝나고 warn은 통과시키되 알린다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			cfg, err := config.Load(root)
			if err != nil {
				return fmt.Errorf("위키 설정을 읽을 수 없음: %w", err)
			}
			res, err := lint.Run(root, cfg)
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
				return errors.New("승급을 막는 위반이 있다")
			}
			return nil
		},
	}
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
		fmt.Fprintf(w, "    고치는 법: %s\n", v.Fix)
	}
	if len(res.WikiFindings) > 0 {
		fmt.Fprintln(w, "위키 진단:")
		for _, f := range res.WikiFindings {
			fmt.Fprintf(w, "  [%s] %s 주제 %q\n", f.Severity, f.Rule, f.Topic)
			fmt.Fprintf(w, "    사용 비율이 %d%%(%d/%d 문서)로 broad_topic_pct %d를 넘는다\n",
				f.Percent, len(f.Paths), f.Total, f.Threshold)
			fmt.Fprintf(w, "    해당 문서: %s\n", previewPaths(f.Paths))
			fmt.Fprintf(w, "    고치는 법: %s\n", f.Fix)
		}
	}
	s := res.Summary
	if len(res.Violations) == 0 && len(res.WikiFindings) == 0 {
		fmt.Fprintf(w, "검사한 파일 %d개, 위반 없음\n", s.Files)
		return
	}
	fmt.Fprintf(w, "검사한 파일 %d개, error %d, warn %d, reject %d\n",
		s.Files, s.Error, s.Warn, s.Reject)
}

// previewPaths는 문서 목록을 보기 좋게 줄인다. 많으면 앞의 몇 개만 보이고
// 나머지는 개수로 남긴다.
func previewPaths(paths []string) string {
	if len(paths) <= maxListedPaths {
		return strings.Join(paths, ", ")
	}
	return strings.Join(paths[:maxListedPaths], ", ") + fmt.Sprintf(" 외 %d개", len(paths)-maxListedPaths)
}
