package cli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neocode24/engram/internal/bridge"
	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/digest"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/mcpserver"
	"github.com/neocode24/engram/internal/resurface"
	"github.com/neocode24/engram/internal/state"
	"github.com/neocode24/engram/internal/status"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// newMCPCmd는 위키를 MCP 서버로 노출하는 mcp 커맨드를 반환한다.
// 전송은 stdio 하나다. stdout 은 프로토콜 전용이므로 어떤 안내도
// stdout 으로 내지 않는다. JSON-RPC 가 깨져 서버가 조용히 죽는 것을
// 막기 위해서다(ADR 0043).
func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "위키를 MCP 서버로 노출합니다",
		Long: `위키를 MCP 서버로 stdio 로 노출합니다.

도구는 열이고 쓰기는 capture 하나뿐이다. capture 는 inbox 에만 쓴다.
승급(promote, demote, archive)은 사람이 확정하므로 도구로 내보내지
않는다. 이것이 다른 PKM MCP 와의 차별점이다.

위키 경로는 --wiki 로 시작 시 고정한다. 도구는 경로를 인자로 받지
않는다. 도구 인자로 경로를 받으면 에이전트가 임의의 디렉토리를 읽을
수 있기 때문이다.

도구 결과는 각 커맨드의 --json 출력과 같은 구조다. 조회 도구는 재료를
낼 뿐 요약하지 않는다. 기준 시각은 실제 시계를 쓰고 --now 를 받지
않는다. 재현이 필요하면 CLI 를 쓰세요.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 시작 시 위키가 유효한지 확인한다. 서버가 뜬 뒤에 매번
			// 실패하는 것보다 낫다. 안내는 stderr 로만 낸다.
			root, _, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			s := mcpserver.New("engram", version)
			registerMCPTools(s, root)
			fmt.Fprintf(cmd.ErrOrStderr(), "engram MCP 서버를 띄웁니다: %s\n", root)
			return mcpserver.RunStdio(cmd.Context(), s)
		},
	}
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// mcpCaptureArgs는 capture 도구의 입력이다.
type mcpCaptureArgs struct {
	Body  string `json:"body" jsonschema:"문서 본문"`
	Title string `json:"title,omitempty" jsonschema:"문서 제목. 생략하면 본문 첫 줄에서 만든다"`
	Slug  string `json:"slug,omitempty" jsonschema:"파일명 슬러그. 생략하면 제목에서 만든다"`
}

// mcpQueryArgs는 질의를 받는 조회 도구의 입력이다.
type mcpQueryArgs struct {
	Query string `json:"query" jsonschema:"검색 질의"`
	Limit int    `json:"limit,omitempty" jsonschema:"결과 상한. 생략하면 기본값"`
}

// mcpSlugArgs는 슬러그를 받는 조회 도구의 입력이다.
type mcpSlugArgs struct {
	Slug string `json:"slug" jsonschema:"조회할 슬러그"`
}

// mcpLimitArgs는 상한만 받는 조회 도구의 입력이다.
type mcpLimitArgs struct {
	Limit int `json:"limit,omitempty" jsonschema:"결과 상한. 생략하면 기본값"`
}

// mcpBridgeArgs는 bridge 도구의 입력이다.
type mcpBridgeArgs struct {
	Min   float64 `json:"min,omitempty" jsonschema:"코사인 유사도 하한. 생략하면 기본값"`
	Limit int     `json:"limit,omitempty" jsonschema:"낼 쌍 수 상한. 생략하면 기본값"`
}

// mcpDigestArgs는 digest 도구의 입력이다.
type mcpDigestArgs struct {
	Days int `json:"days,omitempty" jsonschema:"집계 기간(일). 생략하면 기본값"`
}

// registerMCPTools는 도구 열을 서버에 등록한다. 도구 설명은 에이전트에게
// 주는 지시문이므로 스킬 문서(internal/skills/SKILL.md)의 문체를 따른다.
// 위키 경로는 여기서 고정되고 어떤 도구도 경로를 인자로 받지 않는다.
func registerMCPTools(s *mcp.Server, root string) {
	mcp.AddTool[mcpCaptureArgs, any](s, &mcp.Tool{
		Name: "capture",
		Description: "새 메모를 inbox 에 넣는다. 이 도구는 inbox 에만 쓴다. " +
			"context 로 올리는 승급은 게이트를 지나야 하고 확정은 사람이 한다. " +
			"승급을 제안하려면 사용자에게 engram promote 실행을 맡겨라.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpCaptureArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		titleFromFlag := in.Title != ""
		title := deriveTitle(in.Title, in.Body)
		content := withHeading(title, in.Body, titleFromFlag)
		slug, err := resolveSlug(in.Slug, title)
		if err != nil {
			return nil, nil, err
		}
		fm := wiki.Frontmatter(wiki.StageInbox, cfg)
		date := time.Now().Format("2006-01-02")
		fm["created"] = date
		path, err := wiki.Create(root, cfg, wiki.StageInbox, date, slug, fm, content)
		if err != nil {
			return nil, nil, err
		}
		return nil, ingestResult{Path: path, Slug: slug, Stage: string(wiki.StageInbox)}, nil
	})

	mcp.AddTool[mcpQueryArgs, any](s, &mcp.Tool{
		Name:        "search",
		Description: "위키를 검색해 문서 목록을 재료로 낸다. 요약하지 않는다. 문장은 네가 만든다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpQueryArgs) (*mcp.CallToolResult, any, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		res, err := searchJSON(root, in.Query, limit)
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[mcpQueryArgs, any](s, &mcp.Tool{
		Name:        "recall",
		Description: "질의에 맞는 원문 조각을 출처와 함께 낸다. 요약하지 않는다. 조각을 인용해 네가 문장을 만든다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpQueryArgs) (*mcp.CallToolResult, any, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 5
		}
		res, err := recallJSON(root, in.Query, limit)
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[mcpSlugArgs, any](s, &mcp.Tool{
		Name:        "backlinks",
		Description: "슬러그를 가리키는 링크를 본문 링크와 관계 필드로 구분해 낸다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpSlugArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		walked, err := walk.Files(root, cfg)
		if err != nil {
			return nil, nil, err
		}
		g := graph.Build(walked)
		links := g.Backlinks(in.Slug)
		res := backlinksResponse{
			Slug:      in.Slug,
			Exists:    g.Has(in.Slug),
			Backlinks: make([]backlinkHit, 0, len(links)),
		}
		for _, l := range links {
			res.Backlinks = append(res.Backlinks, backlinkHit{
				Path: l.From, Line: l.Line, Field: l.Field, Slug: l.Slug, Raw: l.Raw,
			})
		}
		return nil, res, nil
	})

	mcp.AddTool[struct{}, any](s, &mcp.Tool{
		Name:        "status",
		Description: "위키 현황을 수치로 낸다. 단계별 문서 수, 링크 수, 고아 수, lint 요약.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		res, err := status.Run(root, time.Now())
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[struct{}, any](s, &mcp.Tool{
		Name:        "lint",
		Description: "규칙 위반 목록을 낸다. 규칙 ID, 등급, 고치는 법이 함께 나온다. 재료일 뿐 판단하지 않는다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		res, err := lint.Run(root, cfg)
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[struct{}, any](s, &mcp.Tool{
		Name: "rules",
		Description: "이 위키에 지금 적용되는 규칙 전부를 낸다. 임계값과 허용값을 지어내지 말고 " +
			"이 도구로 얻어라. 위키마다 값이 다르다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		return nil, buildRulesReport(cfg), nil
	})

	mcp.AddTool[mcpLimitArgs, any](s, &mcp.Tool{
		Name:        "resurface",
		Description: "오래 안 본 context 문서 후보를 재료로 낸다. 제시 이력을 쓰지 않는다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpLimitArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 5
		}
		// dry-run 고정. 에이전트가 위키를 훑는 것만으로 사람의 재발견
		// 이력이 오염되면 안 된다(ADR 0043).
		res, err := resurface.Run(root, cfg, time.Now(), limit, true)
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[mcpBridgeArgs, any](s, &mcp.Tool{
		Name:        "bridge",
		Description: "연결되지 않은 문서 쌍을 유사도와 함께 낸다. 기각을 기록하지 않는다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpBridgeArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		min := in.Min
		if min <= 0 {
			min = 0.30
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		ix := index.Load(root)
		if ix == nil {
			return nil, nil, fmt.Errorf("검색 색인이 없습니다. engram reindex 로 색인을 만든 뒤 다시 실행하세요")
		}
		walked, err := walk.Files(root, cfg)
		if err != nil {
			return nil, nil, err
		}
		stale := !ix.Fresh(walked, root)
		st, err := state.Load(root)
		if err != nil {
			return nil, nil, err
		}
		res := bridge.Run(ix, graph.Build(walked), st, min, limit)
		out := bridgeResponse{Min: min, IndexStale: stale, Pairs: make([]bridgePairJSON, 0, len(res.Pairs))}
		for _, p := range res.Pairs {
			out.Pairs = append(out.Pairs, bridgePairJSON{A: p.A, B: p.B, Score: round2(p.Score)})
		}
		return nil, out, nil
	})

	mcp.AddTool[mcpDigestArgs, any](s, &mcp.Tool{
		Name:        "digest",
		Description: "기간 집계를 재료로 낸다. 신규 문서, 노후 문서, 고아 목록. 다이제스트 산문은 네가 쓴다.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpDigestArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		days := in.Days
		if days <= 0 {
			days = 30
		}
		res, err := digest.Run(root, cfg, time.Now(), days)
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})
}

// searchJSON은 search 커맨드의 --json 출력과 같은 구조를 만든다.
// 판정을 다시 만들지 않고 CLI 와 같은 경로를 돈다.
func searchJSON(root, query string, limit int) (searchResponse, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return searchResponse{}, err
	}
	ix := index.Load(root)
	status := indexMissing
	if ix != nil {
		walked, err := walk.Files(root, cfg)
		if err != nil {
			return searchResponse{}, err
		}
		if ix.Fresh(walked, root) {
			status = indexFresh
		} else {
			status = indexStale
		}
	}
	if ix == nil {
		walked, err := walk.Files(root, cfg)
		if err != nil {
			return searchResponse{}, err
		}
		ix, err = index.Build(root, walked, index.DefaultWeights())
		if err != nil {
			return searchResponse{}, err
		}
	}
	results := ix.Search(query, limit)
	res := searchResponse{Query: query, IndexStatus: status, Results: make([]searchHit, 0, len(results))}
	for i, r := range results {
		res.Results = append(res.Results, searchHit{
			Rank: i + 1, Slug: r.Slug, Score: round2(r.Score), Path: r.Path,
		})
	}
	return res, nil
}

// recallJSON은 recall 커맨드의 --json 출력과 같은 구조를 만든다.
func recallJSON(root, query string, limit int) (recallResponse, error) {
	ix := index.Load(root)
	if ix == nil {
		return recallResponse{}, fmt.Errorf("검색 색인이 없습니다. engram reindex 를 먼저 실행하세요")
	}
	cfg, err := config.Load(root)
	if err != nil {
		return recallResponse{}, err
	}
	walked, err := walk.Files(root, cfg)
	if err != nil {
		return recallResponse{}, err
	}
	status := indexFresh
	if !ix.Fresh(walked, root) {
		status = indexStale
	}
	scored, err := scoreChunks(root, ix.Search(query, limit), query)
	if err != nil {
		return recallResponse{}, err
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
	res := recallResponse{Query: query, IndexStatus: status, Chunks: make([]recallChunk, 0, len(scored))}
	for i, s := range scored {
		res.Chunks = append(res.Chunks, recallChunk{
			Rank: i + 1, Slug: s.slug, Path: s.path,
			Heading: s.heading, HeadingPath: s.headingPath,
			StartLine: s.fileStart, EndLine: s.fileEnd,
			Score: round2(float64(s.score)), Body: s.body,
		})
	}
	return res, nil
}

// mustLoadConfig는 위키 설정을 읽는다. MCP 서버는 시작 시 위키를
// 검증했으므로 여기서 실패하는 경우는 없다.
func mustLoadConfig(root string) config.Config {
	cfg, err := config.Load(root)
	if err != nil {
		return config.Config{}
	}
	return cfg
}
