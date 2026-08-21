package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/embed"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// flagLimit는 search 결과 상한 플래그 이름이다.
const flagLimit = "limit"

// flagSemantic는 단어 대신 의미로 순위를 매기는 플래그 이름이다.
const flagSemantic = "semantic"

// 검색이 쓴 축이다. --json의 axis 에 그대로 쓰인다.
const (
	axisTerm     = "term"     // BM25 단어 축
	axisSemantic = "semantic" // 임베딩 코사인 축
)

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

// searchResponse는 search의 --json 출력이다. Axis 는 순위를 매긴 축이며
// 점수의 뜻이 축마다 다르다. 단어 축은 BM25 점수이고 의미 축은 코사인
// 유사도(0에서 1)다. 소비자가 두 축의 점수를 섞어 비교하지 않도록 밝힌다.
type searchResponse struct {
	Query       string      `json:"query"`
	Axis        string      `json:"axis"`
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

			semantic, err := cmd.Flags().GetBool(flagSemantic)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.search.flag_read_fail", flagSemantic), err)
			}
			axis := axisTerm
			var results []index.SearchResult
			if semantic {
				walked, err := walk.Files(root, cfg)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.search.walk_fail"), err)
				}
				results, err = semanticSearch(root, query, ix, walked, limit)
				if err != nil {
					return err
				}
				axis = axisSemantic
			} else {
				results = ix.Search(query, limit)
			}

			if jsonOutput(cmd) {
				res := searchResponse{Query: query, Axis: axis, IndexStatus: status, Results: make([]searchHit, 0, len(results))}
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
			printSearch(cmd.OutOrStdout(), query, results, semantic)
			return nil
		},
	}
	cmd.Flags().Int(flagLimit, 10, i18n.T("cli.search.flag_limit"))
	cmd.Flags().Bool(flagSemantic, false, i18n.T("cli.search.flag_semantic"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.search.flag_wiki"))
	return cmd
}

// semanticSearch는 질의를 벡터로 만들어 문서 벡터와의 코사인 유사도로
// 순위를 매긴다. 단어 축과 섞지 않는다. 두 점수는 척도가 달라 하나로
// 합치려면 세 번째 순위 규칙이 필요하고, 그 규칙이 생기면 bridge 가 쓰는
// 두 축 두 하한과 어긋난다(ADR 0078).
//
// 문서 벡터는 캐시에서 읽기만 한다. 계산은 bridge 의 몫이다. 캐시가
// 비어 있으면 계산하지 않고 안내한다. 여기서 계산하면 검색 한 번이
// 수십 분이 된다.
//
// 대상은 context 문서뿐이다. 벡터를 만드는 자리가 bridge 이고 bridge 가
// context 만 보기 때문이다. sources 원본은 이 축에 잡히지 않으므로
// 이을 곳을 찾을 때는 단어 축도 함께 돌린다.
func semanticSearch(root, query string, ix *index.Index, walked []walk.Doc, limit int) ([]index.SearchResult, error) {
	docs := contextComputeDocs(ix, walked)
	vectors := embed.Cached(root, docs)
	if len(vectors) == 0 {
		return nil, errors.New(i18n.T("cli.search.semantic_no_vectors"))
	}

	enc, err := embed.Open()
	if errors.Is(err, embed.ErrNoModel) {
		return nil, errors.New(i18n.T("cli.search.semantic_no_model"))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.search.semantic_open_fail"), err)
	}
	defer func() { _ = enc.Close() }()

	vecs, err := enc.Encode(context.Background(), []string{embed.Truncate(query)})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.search.semantic_encode_fail"), err)
	}
	q := vecs[0]

	titles := map[string]index.DocEntry{}
	for _, e := range ix.Docs {
		titles[e.Path] = e
	}
	out := make([]index.SearchResult, 0, len(vectors))
	for path, v := range vectors {
		e, ok := titles[path]
		if !ok {
			continue
		}
		out = append(out, index.SearchResult{
			Path: path, Slug: e.Slug, Title: e.Title, Score: cosine(q, v),
		})
	}
	// 동점은 슬러그 순으로 고정한다. 같은 위키와 같은 질의가 언제나 같은
	// 순서를 내야 교재의 실측 출력이 성립한다(ADR 0028).
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Slug < out[j].Slug
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// cosine은 정규화된 벡터 둘의 코사인 유사도를 낸다. Encode 가 정규화해
// 내므로 내적이 곧 코사인이다. 길이가 다르면 0 이다.
func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

// printSearch는 사람용 검색 결과를 인쇄한다. 재료만 반환하고 요약을
// 만들지 않는다. ADR 0014.
func printSearch(w io.Writer, query string, results []index.SearchResult, semantic bool) {
	if len(results) == 0 {
		if semantic {
			fmt.Fprintln(w, i18n.T("cli.search.no_results"))
			return
		}
		tokens := index.Tokenize(query)
		fmt.Fprintln(w, i18n.T("cli.search.no_results"))
		fmt.Fprintln(w, i18n.T("cli.search.query_tokens", query, strings.Join(tokens, ", ")))
		return
	}
	if semantic {
		fmt.Fprintln(w, i18n.T("cli.search.semantic_header"))
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
