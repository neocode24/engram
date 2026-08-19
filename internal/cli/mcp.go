package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neocode24/engram/internal/bridge"
	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/digest"
	"github.com/neocode24/engram/internal/embed"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
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
		Short: i18n.T("cli.mcp.short"),
		Long:  i18n.T("cli.mcp.long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 시작 시 위키가 유효한지 확인한다. 서버가 뜬 뒤에 매번
			// 실패하는 것보다 낫다. 안내는 stderr 로만 낸다.
			root, _, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			s := mcpserver.New("engram", version)
			registerMCPTools(s, root)
			fmt.Fprint(cmd.ErrOrStderr(), i18n.T("cli.mcp.starting", root)+"\n")
			return mcpserver.RunStdio(cmd.Context(), s)
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.mcp.flag_wiki"))
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
	Min      float64 `json:"min,omitempty" jsonschema:"단어 축 코사인 하한. 생략하면 기본값"`
	MinEmbed float64 `json:"minEmbed,omitempty" jsonschema:"임베딩 축 코사인 하한. 두 축은 눈금이 다르므로 값을 공유하지 않는다. 생략하면 기본값"`
	Limit    int     `json:"limit,omitempty" jsonschema:"낼 쌍 수 상한. 생략하면 기본값"`
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
		Name:        "capture",
		Description: i18n.T("cli.mcp.tool_capture"),
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
		Description: i18n.T("cli.mcp.tool_search"),
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
		Description: i18n.T("cli.mcp.tool_recall"),
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
		Description: i18n.T("cli.mcp.tool_backlinks"),
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
		Description: i18n.T("cli.mcp.tool_status"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		res, err := status.Run(root, time.Now())
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})

	mcp.AddTool[struct{}, any](s, &mcp.Tool{
		Name:        "lint",
		Description: i18n.T("cli.mcp.tool_lint"),
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
		Name:        "rules",
		Description: i18n.T("cli.mcp.tool_rules"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		return nil, buildRulesReport(cfg), nil
	})

	mcp.AddTool[mcpLimitArgs, any](s, &mcp.Tool{
		Name:        "resurface",
		Description: i18n.T("cli.mcp.tool_resurface"),
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
		Description: i18n.T("cli.mcp.tool_bridge"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpBridgeArgs) (*mcp.CallToolResult, any, error) {
		cfg, err := config.Load(root)
		if err != nil {
			return nil, nil, err
		}
		min := in.Min
		if min <= 0 {
			min = cfg.Thresholds.BridgeWordMin
		}
		minEmbed := in.MinEmbed
		if minEmbed <= 0 {
			minEmbed = cfg.Thresholds.BridgeEmbedMin
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		ix := index.Load(root)
		if ix == nil {
			return nil, nil, errors.New(i18n.T("cli.mcp.index_missing_build"))
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
		// 벡터는 캐시에 있는 것만 쓴다. 계산은 문서당 12.6초라 도구 호출
		// 안에서 할 수 없다. 캐시를 채우는 것은 bridge 커맨드의 몫이다.
		// 축이 돌았는지는 embedAxis 로 밝힌다. 조용히 단어 축만 도는
		// 것이 CLI 와 다른 판정을 내는 자리가 되면 안 된다.
		vectors := embed.Cached(root, contextComputeDocs(ix, walked))
		res := bridge.Run(ix, graph.Build(walked), st,
			bridge.Options{Min: min, EmbedMin: minEmbed, Limit: limit, Vectors: vectors})
		out := bridgeResponse{Min: min, MinEmbed: minEmbed, EmbedAxis: len(vectors) > 0,
			IndexStale: stale, Pairs: make([]bridgePairJSON, 0, len(res.Pairs))}
		for _, p := range res.Pairs {
			out.Pairs = append(out.Pairs, bridgePairJSON{A: p.A, B: p.B, Score: round2(p.Score)})
		}
		return nil, out, nil
	})

	mcp.AddTool[mcpDigestArgs, any](s, &mcp.Tool{
		Name:        "digest",
		Description: i18n.T("cli.mcp.tool_digest"),
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
			Rank: i + 1, Slug: r.Slug, Title: r.Title,
			Score: round2(r.Score), Path: r.Path,
		})
	}
	return res, nil
}

// recallJSON은 recall 커맨드의 --json 출력과 같은 구조를 만든다.
func recallJSON(root, query string, limit int) (recallResponse, error) {
	ix := index.Load(root)
	if ix == nil {
		return recallResponse{}, errors.New(i18n.T("cli.mcp.index_missing_first"))
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
	scored, err := scoreChunks(root, ix.Search(query, recallCandidateDocs), query)
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
