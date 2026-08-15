package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

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
		Use:   "search <질의>",
		Short: "위키를 검색한다",
		Long: `검색 색인으로 위키를 검색한다.

색인이 신선하면 색인으로 검색한다. 낡았으면 낡은 색인 그대로 검색하되
경고를 낸다. 색인이 없거나 깨졌으면 이번 실행에서만 문서를 읽어 검색한다.
어느 쪽이든 색인 파일을 쓰지 않는다. 색인을 만드는 것은 reindex뿐이다.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagLimit, err)
			}

			// 색인 파일을 읽고 신선도를 판정한다. 조회는 파일을 쓰지 않는다.
			ix := index.Load(root)
			status := indexMissing
			if ix != nil {
				walked, err := walk.Files(root, cfg)
				if err != nil {
					return fmt.Errorf("위키를 순회할 수 없음: %w", err)
				}
				if ix.Fresh(walked, root) {
					status = indexFresh
				} else {
					status = indexStale
					fmt.Fprintf(cmd.ErrOrStderr(),
						"경고: 색인이 낡았다. 낡은 색인으로 검색한다. engram reindex로 갱신한다\n")
				}
			}
			if ix == nil {
				walked, err := walk.Files(root, cfg)
				if err != nil {
					return fmt.Errorf("위키를 순회할 수 없음: %w", err)
				}
				ix, err = index.Build(root, walked, index.DefaultWeights())
				if err != nil {
					return fmt.Errorf("즉석 색인을 만들 수 없음: %w", err)
				}
				fmt.Fprintf(cmd.ErrOrStderr(),
					"안내: 색인이 없어 이번 실행에서만 문서를 읽었다. engram reindex를 한 번 돌리면 검색이 빨라진다\n")
			}

			results := ix.Search(query, limit)
			if jsonOutput(cmd) {
				res := searchResponse{Query: query, IndexStatus: status, Results: make([]searchHit, 0, len(results))}
				for i, r := range results {
					res.Results = append(res.Results, searchHit{
						Rank: i + 1, Slug: r.Slug, Score: round2(r.Score), Path: r.Path,
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
	cmd.Flags().Int(flagLimit, 10, "결과 상한")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// printSearch는 사람용 검색 결과를 인쇄한다. 재료만 반환하고 요약을
// 만들지 않는다. ADR 0014.
func printSearch(w io.Writer, query string, results []index.SearchResult) {
	if len(results) == 0 {
		tokens := index.Tokenize(query)
		fmt.Fprintf(w, "결과가 없다\n")
		fmt.Fprintf(w, "질의 %q는 다음 토큰으로 검색했다: %s\n",
			query, strings.Join(tokens, ", "))
		return
	}
	for i, r := range results {
		fmt.Fprintf(w, "%3d  %.2f  %s  %s\n", i+1, r.Score, r.Slug, r.Path)
	}
}

// round2는 값을 소수점 둘째 자리까지 반올림한다.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
