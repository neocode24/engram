package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/bridge"
	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/embed"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/state"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

const (
	flagMin      = "min"
	flagMinEmbed = "min-embed"
	flagReject   = "reject"
	flagUnreject = "unreject"
)

// bridgePairJSON은 --json이 내는 쌍 한 건이다.
type bridgePairJSON struct {
	A     string   `json:"a"`
	B     string   `json:"b"`
	Score float64  `json:"score"`
	Axes  []string `json:"axes"`
}

// bridgeResponse는 후보 탐색의 --json 출력이다. Min 은 단어 축 하한,
// MinEmbed 는 임베딩 축 하한이다.
type bridgeResponse struct {
	Min      float64 `json:"min"`
	MinEmbed float64 `json:"minEmbed"`
	// EmbedAxis 는 임베딩 축이 실제로 돌았는지다. 거짓이면 이 결과는
	// 단어 축만으로 나온 것이다. 모델이 없거나 벡터 캐시가 비어 있을
	// 때 그렇게 된다. 소비자가 결과의 성격을 알아야 하므로 밝힌다.
	EmbedAxis  bool             `json:"embedAxis"`
	IndexStale bool             `json:"indexStale"`
	Pairs      []bridgePairJSON `json:"pairs"`
}

// bridgeActionResponse는 --reject/--unreject 의 --json 출력이다.
type bridgeActionResponse struct {
	Action string `json:"action"`
	A      string `json:"a"`
	B      string `json:"b"`
}

// newBridgeCmd는 유사도가 높은데 링크가 없는 문서 쌍을 찾는 bridge 커맨드를
// 반환한다. 후보 탐색은 조회다. 파일을 쓰는 것은 기각과 되돌리기뿐이다.
func newBridgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: i18n.T("cli.bridge.short"),
		Long:  i18n.T("cli.bridge.long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			reject, err := cmd.Flags().GetStringSlice(flagReject)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.bridge.flag_read_fail", flagReject), err)
			}
			unreject, err := cmd.Flags().GetStringSlice(flagUnreject)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.bridge.flag_read_fail", flagUnreject), err)
			}
			// cobra 의 StringSlice 는 "--reject a b" 를 값 하나와 위치 인자로
			// 쪼갠다. 안내 문구가 "--reject A B" 형태이므로 위치 인자를
			// 슬러그로 이어 붙여 그대로 동작하게 한다.
			if len(args) > 0 {
				switch {
				case len(reject) > 0:
					reject = append(reject, args...)
				case len(unreject) > 0:
					unreject = append(unreject, args...)
				default:
					return errors.New(i18n.T("cli.bridge.no_positional_args"))
				}
			}
			if len(reject) > 0 && len(unreject) > 0 {
				return errors.New(i18n.T("cli.bridge.both_flags"))
			}
			if len(reject) > 0 {
				if len(reject) != 2 {
					return errors.New(i18n.T("cli.bridge.reject_needs_two",
						len(reject), strings.Join(reject, " ")))
				}
				return rejectPair(cmd, root, cfg, reject[0], reject[1])
			}
			if len(unreject) > 0 {
				if len(unreject) != 2 {
					return errors.New(i18n.T("cli.bridge.unreject_needs_two",
						len(unreject), strings.Join(unreject, " ")))
				}
				return unrejectPair(cmd, root, unreject[0], unreject[1])
			}

			// 플래그를 주지 않았으면 설정 파일 값, 그것도 없으면 코드
			// 기본값을 쓴다. 플래그 기본값과 설정 기본값이 같아 결과는
			// 같지만 설정을 바꾼 위키에서 플래그 없이도 따라오게 한다.
			wordMin := cfg.Thresholds.BridgeWordMin
			if cmd.Flags().Changed(flagMin) {
				f, err := cmd.Flags().GetFloat64(flagMin)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.bridge.flag_read_fail", flagMin), err)
				}
				wordMin = f
			}
			embedMin := cfg.Thresholds.BridgeEmbedMin
			if cmd.Flags().Changed(flagMinEmbed) {
				f, err := cmd.Flags().GetFloat64(flagMinEmbed)
				if err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.bridge.flag_read_fail", flagMinEmbed), err)
				}
				embedMin = f
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.bridge.flag_read_fail", flagLimit), err)
			}
			// 조회는 색인을 읽기만 한다. 색인이 없으면 안내하고 종료한다. ADR 0025.
			ix := index.Load(root)
			if ix == nil {
				return errors.New(i18n.T("cli.bridge.no_index"))
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.bridge.walk_fail"), err)
			}
			stale := false
			if !ix.Fresh(walked, root) {
				stale = true
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.bridge.warn_stale"))
			}
			st, err := state.Load(root)
			if err != nil {
				return err
			}
			// 임베딩 계산은 bridge 가 필요할 때만 한다(ADR 0074). 모델이
			// 없으면 경고를 내고 단어 축으로 계속 간다.
			vectors, err := embedVectors(cmd, root, ix, walked)
			if err != nil {
				return err
			}
			opts := bridge.Options{Min: wordMin, EmbedMin: embedMin, Limit: limit, Vectors: vectors}
			res := bridge.Run(ix, graph.Build(walked), st, opts)
			if jsonOutput(cmd) {
				out := bridgeResponse{Min: wordMin, MinEmbed: embedMin, EmbedAxis: len(vectors) > 0, IndexStale: stale, Pairs: make([]bridgePairJSON, 0, len(res.Pairs))}
				for _, p := range res.Pairs {
					out.Pairs = append(out.Pairs, bridgePairJSON{A: p.A, B: p.B, Score: round2(p.Score), Axes: p.Axes})
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			printBridge(cmd.OutOrStdout(), res, wordMin, embedMin, vectors != nil)
			return nil
		},
	}
	cmd.Flags().Float64(flagMin, config.DefaultBridgeWordMin, i18n.T("cli.bridge.flag_min"))
	cmd.Flags().Float64(flagMinEmbed, config.DefaultBridgeEmbedMin, i18n.T("cli.bridge.flag_min_embed"))
	cmd.Flags().Int(flagLimit, 10, i18n.T("cli.bridge.flag_limit"))
	cmd.Flags().StringSlice(flagReject, nil, i18n.T("cli.bridge.flag_reject"))
	cmd.Flags().StringSlice(flagUnreject, nil, i18n.T("cli.bridge.flag_unreject"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.bridge.flag_wiki"))
	return cmd
}

// embedVectors는 context 문서의 임베딩 벡터를 경로 기준으로 낸다.
// 모델이 없으면 경고 한 줄을 내고 nil 을 반환해 단어 축만으로 계속하게
// 한다. 시맨틱의 부재는 결손이 아니라 성능 저하다(ADR 0007, 0074).
func embedVectors(cmd *cobra.Command, root string, ix *index.Index, walked []walk.Doc) (map[string][]float32, error) {
	docs := contextComputeDocs(ix, walked)
	if len(docs) == 0 {
		return nil, nil
	}
	// 캐시가 다 차 있으면 진행률이 한 번도 오지 않는다. 그때는 개행도
	// 내지 않는다.
	reported := false
	progress := func(done, total int) {
		reported = true
		fmt.Fprintf(cmd.ErrOrStderr(), "\r%s", i18n.T("cli.bridge.embed_progress", done, total))
	}
	vecs, err := embed.Compute(root, docs, progress)
	if errors.Is(err, embed.ErrNoModel) {
		fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.bridge.warn_no_model"))
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.bridge.embed_fail"), err)
	}
	if reported {
		fmt.Fprintln(cmd.ErrOrStderr())
	}
	return vecs, nil
}

// rejectPair는 쌍을 기각해 상태 파일에 영구 기록한다. 없는 슬러그는
// 거절한다. 오타로 생긴 기각은 조용히 무효가 되어 영영 발견되지 않는다.
func rejectPair(cmd *cobra.Command, root string, cfg config.Config, a, b string) error {
	walked, err := walk.Files(root, cfg)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.bridge.walk_fail"), err)
	}
	g := graph.Build(walked)
	var missing []string
	for _, s := range []string{a, b} {
		if !g.Has(s) {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return errors.New(i18n.T("cli.bridge.reject_missing", strings.Join(missing, ", ")))
	}
	na, nb := graph.Normalize(a), graph.Normalize(b)
	st, err := state.Load(root)
	if err != nil {
		return err
	}
	if !st.Reject(na, nb) {
		printBridgeAction(cmd, bridgeActionResponse{Action: "reject", A: na, B: nb},
			i18n.T("cli.bridge.already_rejected", na, nb))
		return nil
	}
	if err := st.Save(root); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.bridge.reject_save_fail"), err)
	}
	printBridgeAction(cmd, bridgeActionResponse{Action: "reject", A: na, B: nb},
		i18n.T("cli.bridge.rejected", na, nb, filepath.Join(root, state.StateFileName)))
	return nil
}

// unrejectPair는 기각을 되돌린다. 지운 문서를 가리키는 기각도 지울 수
// 있어야 하므로 슬러그 존재는 검사하지 않는다.
func unrejectPair(cmd *cobra.Command, root, a, b string) error {
	na, nb := graph.Normalize(a), graph.Normalize(b)
	st, err := state.Load(root)
	if err != nil {
		return err
	}
	if !st.Unreject(na, nb) {
		printBridgeAction(cmd, bridgeActionResponse{Action: "unreject", A: na, B: nb},
			i18n.T("cli.bridge.not_rejected", na, nb))
		return nil
	}
	if err := st.Save(root); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.bridge.unreject_save_fail"), err)
	}
	printBridgeAction(cmd, bridgeActionResponse{Action: "unreject", A: na, B: nb},
		i18n.T("cli.bridge.unrejected", na, nb))
	return nil
}

// printBridgeAction은 기각/되돌리기 결과를 --json 여부에 따라 낸다.
func printBridgeAction(cmd *cobra.Command, res bridgeActionResponse, text string) {
	if jsonOutput(cmd) {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}
	fmt.Fprint(cmd.OutOrStdout(), text+"\n")
}

// axisLabel은 쌍을 잡은 축의 눈금 이름을 낸다. 사용자가 근거를 알아야
// 기각 여부를 판단할 수 있다(ADR 0075).
func axisLabel(axes []string) string {
	hasTerm, hasEmbed := false, false
	for _, a := range axes {
		switch a {
		case bridge.AxisTerm:
			hasTerm = true
		case bridge.AxisEmbed:
			hasEmbed = true
		}
	}
	switch {
	case hasTerm && hasEmbed:
		return i18n.T("cli.bridge.axis_both")
	case hasEmbed:
		return i18n.T("cli.bridge.axis_embed")
	case hasTerm:
		return i18n.T("cli.bridge.axis_term")
	}
	return ""
}

// printBridge는 후보 탐색의 사람용 출력을 인쇄한다. 재료만 반환하고
// 문장을 만들지 않는다. embedOn 은 임베딩 축이 돌았는지다.
func printBridge(w io.Writer, res bridge.Result, wordMin, embedMin float64, embedOn bool) {
	if len(res.Pairs) == 0 {
		s := res.Stats
		fmt.Fprintln(w, i18n.T("cli.bridge.no_pairs"))
		if embedOn {
			fmt.Fprintln(w, i18n.T("cli.bridge.stats_axes",
				s.ContextDocs, s.Linked, s.Rejected, s.BelowMin))
		} else {
			fmt.Fprintln(w, i18n.T("cli.bridge.stats",
				s.ContextDocs, s.Linked, s.Rejected, wordMin, s.BelowMin))
		}
		return
	}
	if embedOn {
		fmt.Fprintln(w, i18n.T("cli.bridge.header_axes", wordMin, embedMin))
	} else {
		fmt.Fprintln(w, i18n.T("cli.bridge.header", wordMin))
	}
	for i, p := range res.Pairs {
		fmt.Fprintf(w, "%3d  %.2f  %s  %s  %s\n", i+1, round2(p.Score), axisLabel(p.Axes), p.A, p.B)
		fmt.Fprintln(w, i18n.T("cli.bridge.reject_hint", p.A, p.B))
	}
}

// contextComputeDocs는 색인에서 context 문서만 뽑아 임베딩 계산 대상
// 목록으로 만든다. bridge 커맨드와 MCP 도구가 같은 목록을 써야 두
// 경로의 판정이 갈라지지 않는다.
func contextComputeDocs(ix *index.Index, walked []walk.Doc) []embed.ComputeDoc {
	bodies := map[string]string{}
	for _, wd := range walked {
		if wd.Err == nil {
			bodies[wd.Rel] = wd.Parsed.Body
		}
	}
	docs := make([]embed.ComputeDoc, 0, len(ix.Docs))
	for _, e := range ix.Docs {
		if seg, _, _ := strings.Cut(e.Path, "/"); seg != "context" {
			continue
		}
		docs = append(docs, embed.ComputeDoc{Path: e.Path, Title: e.Title, Body: bodies[e.Path]})
	}
	return docs
}
