package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/neocode24/engram/internal/resurface"
	"github.com/spf13/cobra"
)

// newResurfaceCmd는 오래 안 본 context 문서를 다시 꺼내는 resurface
// 커맨드를 반환한다. 재발견 커맨드 중 유일하게 상태를 쓰므로 --now 가
// 결과를 결정한다. ADR 0028.
func newResurfaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resurface",
		Short: "오래 안 본 context 문서를 다시 꺼냅니다",
		Long: `stale_days를 넘긴 context 문서를 골라 다시 보여줍니다.

제시 이력을 <위키>/.engram/resurface.json에 남겨 최근에 보여준 문서가
먼저 나오지 않게 합니다. 이 파일은 gitignore 대상이고 없어도 빈 이력으로
동작하므로 지워도 도구가 멈추지 않습니다.
한 번도 제시하지 않은 문서를 먼저 내고, 그다음은 마지막 제시가 오래된
순서입니다. 상태를 쓰는 유일한 조회 커맨드라 실행마다 결과가 달라지므로
--now로 기준 시각을 고정할 수 있습니다.
--dry-run은 이력을 기록하지 않습니다.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagLimit, err)
			}
			dryRun, err := cmd.Flags().GetBool(flagDryRun)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagDryRun, err)
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
	cmd.Flags().Int(flagLimit, 5, "낼 문서 수")
	cmd.Flags().Bool(flagDryRun, false, "제시 이력을 기록하지 않습니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// printResurface는 사람용 결과를 인쇄한다. 후보마다 슬러그, 제목,
// 경과일, 마지막 제시 시각을 낸다.
func printResurface(w io.Writer, res resurface.Result, dryRun bool) {
	if len(res.Candidates) == 0 {
		fmt.Fprintf(w, "다시 꺼낼 문서가 없습니다\n")
		fmt.Fprintf(w, "  이유: %s\n", res.Reason)
	} else {
		fmt.Fprintf(w, "다시 꺼낼 문서 %d개 (stale_days %d일, 기준 %s)\n",
			len(res.Candidates), res.StaleDays, res.Now.Format("2006-01-02"))
		for _, c := range res.Candidates {
			last := "제시한 적 없음"
			if c.LastShown != nil {
				last = "마지막 제시 " + c.LastShown.Format("2006-01-02")
			}
			title := ""
			if c.Title != "" {
				title = " (" + c.Title + ")"
			}
			fmt.Fprintf(w, "  - %s%s: 마지막 갱신 %d일 전, %s\n", c.Slug, title, c.AgeDays, last)
		}
	}
	if res.SkippedNoDate > 0 {
		fmt.Fprintf(w, "기준 날짜를 알 수 없는 context 문서 %d개는 대상에서 뺐습니다\n", res.SkippedNoDate)
	}
	if dryRun {
		fmt.Fprintf(w, "--dry-run이라 제시 이력을 기록하지 않았습니다\n")
	}
}
