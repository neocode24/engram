package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/chunk"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// recallChunk는 --json이 내는 청크 한 건이다. 원문을 그대로 실으므로
// 에이전트가 컨텍스트에 넣고 [[슬러그]]로 인용할 수 있다.
type recallChunk struct {
	Rank        int      `json:"rank"`
	Slug        string   `json:"slug"`
	Path        string   `json:"path"`
	Heading     string   `json:"heading"`
	HeadingPath []string `json:"headingPath"`
	StartLine   int      `json:"startLine"`
	EndLine     int      `json:"endLine"`
	Score       float64  `json:"score"`
	Body        string   `json:"body"`
}

// recallResponse는 recall의 --json 출력이다.
type recallResponse struct {
	Query       string        `json:"query"`
	IndexStatus string        `json:"indexStatus"`
	Chunks      []recallChunk `json:"chunks"`
}

// newRecallCmd는 질의에 맞는 원문 조각을 내는 recall 커맨드를 반환한다.
// search가 사람이 열어 볼 문서 목록을 내는 데 대해 recall은 에이전트가
// 인용할 조각을 낸다. 둘 다 요약을 만들지 않는다. ADR 0014, 0028.
func newRecallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall " + i18n.T("usage.args.query"),
		Short: i18n.T("cli.recall.short"),
		Long:  i18n.T("cli.recall.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.recall.flag_read_fail", flagLimit), err)
			}

			// 색인 파일이 없으면 즉석 색인을 만들지 않고 안내한다.
			// recall의 결과물이 색인 문서 집합에 의존하므로 search 와 달리
			// 색인이 필수다.
			ix := index.Load(root)
			if ix == nil {
				return errors.New(i18n.T("cli.recall.no_index", root))
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.recall.walk_fail"), err)
			}
			status := indexFresh
			if !ix.Fresh(walked, root) {
				status = indexStale
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.recall.warn_stale"))
			}

			scored, err := scoreChunks(root, ix.Search(query, limit), query)
			if err != nil {
				return err
			}
			sort.Slice(scored, func(i, j int) bool {
				if scored[i].score != scored[j].score {
					return scored[i].score > scored[j].score
				}
				if scored[i].slug != scored[j].slug {
					return scored[i].slug < scored[j].slug
				}
				return scored[i].startLine < scored[j].startLine
			})
			if limit > 0 && len(scored) > limit {
				scored = scored[:limit]
			}

			res := recallResponse{Query: query, IndexStatus: status,
				Chunks: make([]recallChunk, 0, len(scored))}
			for i, s := range scored {
				res.Chunks = append(res.Chunks, recallChunk{
					Rank: i + 1, Slug: s.slug, Path: s.path,
					Heading: s.heading, HeadingPath: s.headingPath,
					StartLine: s.fileStart, EndLine: s.fileEnd,
					Score: round2(float64(s.score)), Body: s.body,
				})
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printRecall(cmd.OutOrStdout(), query, res.Chunks)
			return nil
		},
	}
	cmd.Flags().Int(flagLimit, 5, i18n.T("cli.recall.flag_limit"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.recall.flag_wiki"))
	return cmd
}

// recallScored는 점수 계산이 끝난 조각 하나다.
type recallScored struct {
	slug, path, heading  string
	headingPath          []string
	startLine, fileStart int
	fileEnd              int
	body                 string
	score                int
}

// scoreChunks는 후보 문서들을 헤딩 단위로 자르고 질의 토큰이 조각마다
// 얼마나 걸리는지 센다. 토크나이저는 index.Tokenize 를 그대로 쓴다.
// search 와 recall 이 다른 토크나이저를 쓰면 두 커맨드의 결과가 어긋난다.
func scoreChunks(root string, candidates []index.SearchResult, query string) ([]recallScored, error) {
	qset := map[string]bool{}
	for _, t := range index.Tokenize(query) {
		qset[t] = true
	}
	var out []recallScored
	for _, r := range candidates {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.Path)))
		if err != nil {
			// 색인이 낡아 지워진 문서를 가리킬 수 있다. 이미 경고했으므로 건너뛴다.
			continue
		}
		d, err := doc.Parse(r.Path, raw)
		if err != nil {
			continue
		}
		for _, c := range chunk.Split(d.Body) {
			score := 0
			for _, t := range index.Tokenize(c.Body) {
				if qset[t] {
					score++
				}
			}
			if score == 0 {
				continue
			}
			out = append(out, recallScored{
				slug: r.Slug, path: r.Path, heading: c.Heading, headingPath: c.Path,
				// chunk 의 줄 번호는 본문 기준이므로 파일 기준으로 옮긴다.
				startLine: c.StartLine,
				fileStart: c.StartLine + d.BodyLine - 1,
				fileEnd:   c.EndLine + d.BodyLine - 1,
				body:      c.Body, score: score,
			})
		}
	}
	return out, nil
}

// printRecall는 사람용 조각 목록을 인쇄한다. 조각 사이에 구분선을 넣어
// 읽을 수 있게 한다. 재료만 반환하고 요약을 만들지 않는다. ADR 0014.
func printRecall(w io.Writer, query string, chunks []recallChunk) {
	if len(chunks) == 0 {
		fmt.Fprintln(w, i18n.T("cli.recall.no_results"))
		fmt.Fprintln(w, i18n.T("cli.recall.query_tokens", query, strings.Join(index.Tokenize(query), ", ")))
		return
	}
	for i, c := range chunks {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}
		fmt.Fprintf(w, "%d  %.2f  [[%s]]  %s:%d-%d\n", c.Rank, c.Score, c.Slug, c.Path, c.StartLine, c.EndLine)
		if loc := headingLabel(c); loc != "" {
			fmt.Fprintf(w, "%s\n", loc)
		}
		fmt.Fprintf(w, "%s\n", c.Body)
	}
}

// headingLabel는 조각의 헤딩 위치를 사람이 읽는 경로로 낸다.
// 상위 경로와 이 조각의 헤딩을 잇는다. 서두 조각은 빈 문자열이다.
func headingLabel(c recallChunk) string {
	parts := append([]string{}, c.HeadingPath...)
	if c.Heading != "" {
		parts = append(parts, c.Heading)
	}
	return strings.Join(parts, " > ")
}
