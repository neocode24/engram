package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// flagLimit는 search 결과 상한 플래그 이름이다.
const flagLimit = "limit"

// 인덱스 상태를 나타내는 값이다. --json의 indexStatus 에 그대로 쓰인다.
const (
	indexFresh   = "fresh"  // 색인 파일이 현재 문서와 일치한다
	indexStale   = "stale"  // 색인 파일이 있지만 현재 문서와 어긋난다
	indexMissing = "memory" // 색인 파일이 없거나 깨져 이번 실행에서만 색인했다
)

// searchHit는 --json이 내는 결과 한 건이다.
type searchHit struct {
	Rank  int     `json:"rank"`
	Slug  string  `json:"slug"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
	Path  string  `json:"path"`
}

// searchResponse는 search의 --json 출력이다.
type searchResponse struct {
	Query       string      `json:"query"`
	IndexStatus string      `json:"indexStatus"`
	Results     []searchHit `json:"results"`
}

// newSearchCmd는 인덱스로 위키를 검색하는 search 커맨드를 반환한다.
// 조회 커맨드는 색인 파일을 갱신하지 않는다. ADR 0025.
func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search " + i18n.T("usage.args.query"),
		Short: i18n.T("cli.search.short"),
		Long:  i18n.T("cli.search.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.search.flag_read_fail", flagLimit), err)
			}

			// 색인 파일을 읽고 신선도를 판정한다. 조회는 파일을 쓰지 않는다.
			ix := index.Load(root)
			status := indexMissing
			if ix != nil {
				walked, err := walk.Files(root, cfg)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.search.walk_fail"), err)
				}
				if ix.Fresh(walked, root) {
					status = indexFresh
				} else {
					status = indexStale
					fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.search.warn_stale"))
				}
			}
			if ix == nil {
				walked, err := walk.Files(root, cfg)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.search.walk_fail"), err)
				}
				ix, err = index.Build(root, walked, index.DefaultWeights())
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.search.index_build_fail"), err)
				}
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.search.notice_memory_index"))
			}

			results := ix.Search(query, limit)
			if jsonOutput(cmd) {
				res := searchResponse{Query: query, IndexStatus: status, Results: make([]searchHit, 0, len(results))}
				for i, r := range results {
					res.Results = append(res.Results, searchHit{
						Rank: i + 1, Slug: r.Slug, Title: r.Title,
						Score: round2(r.Score), Path: r.Path,
					})
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printSearch(cmd.OutOrStdout(), query, results)
			return nil
		},
	}
	cmd.Flags().Int(flagLimit, 10, i18n.T("cli.search.flag_limit"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.search.flag_wiki"))
	return cmd
}

// printSearch는 사람용 검색 결과를 인쇄한다. 재료만 반환하고 요약을
// 만들지 않는다. ADR 0014.
func printSearch(w io.Writer, query string, results []index.SearchResult) {
	if len(results) == 0 {
		tokens := index.Tokenize(query)
		fmt.Fprintln(w, i18n.T("cli.search.no_results"))
		fmt.Fprintln(w, i18n.T("cli.search.query_tokens", query, strings.Join(tokens, ", ")))
		return
	}
	for i, r := range results {
		fmt.Fprintf(w, "%3d  %5.2f  %-8s %s\n", i+1, r.Score, stageOf(r.Path), r.Slug)
	}
}

// stageOf는 순회 경로에서 단계 이름을 낸다. 목록에서 경로 전체를 내면
// 슬러그와 겹쳐 같은 값을 두 번 보여 주는 꼴이 된다. 사람이 목록에서
// 실제로 쓰는 정보는 어느 단계에 있는가다(ADR 0060).
func stageOf(rel string) string {
	if i := strings.Index(rel, "/"); i > 0 {
		return rel[:i]
	}
	return i18n.T("cli.search.stage_root")
}

// round2는 값을 소수점 둘째 자리까지 반올림한다.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
