package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/bridge"
	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/state"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

const (
	flagMin      = "min"
	flagReject   = "reject"
	flagUnreject = "unreject"
)

// bridgePairJSON은 --json이 내는 쌍 한 건이다.
type bridgePairJSON struct {
	A     string  `json:"a"`
	B     string  `json:"b"`
	Score float64 `json:"score"`
}

// bridgeResponse는 후보 탐색의 --json 출력이다.
type bridgeResponse struct {
	Min        float64         `json:"min"`
	IndexStale bool            `json:"indexStale"`
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
		Short: "유사한데 링크가 없는 문서 쌍을 찾습니다",
		Long: `검색 색인의 TF 벡터로 context 문서끼리 코사인 유사도를 재고,
유사도가 높은데 링크가 없는 쌍을 보여줍니다.

후보에서 기각한 쌍은 engram-state.yaml 에 영구 기록되어 다시 나오지 않습니다.
--reject 로 기각하고 --unreject 로 되돌립니다. 기각 기록은 git 이 추적합니다.

색인이 없으면 진행하지 않습니다. engram reindex 로 색인을 만드세요.
낡은 색인은 경고를 내고 그대로 진행합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			reject, err := cmd.Flags().GetStringSlice(flagReject)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagReject, err)
			}
			unreject, err := cmd.Flags().GetStringSlice(flagUnreject)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagUnreject, err)
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
					return fmt.Errorf("bridge 는 위치 인자를 받지 않습니다. 기각은 engram bridge --reject <A> <B> 로 합니다")
				}
			}
			if len(reject) > 0 && len(unreject) > 0 {
				return fmt.Errorf("--reject 와 --unreject 를 함께 쓸 수 없습니다. 한 번에 한 동작만 고르세요")
			}
			if len(reject) > 0 {
				if len(reject) != 2 {
					return fmt.Errorf("--reject 는 슬러그 두 개를 받습니다. 지금 %d개입니다: %s",
						len(reject), strings.Join(reject, " "))
				}
				return rejectPair(cmd, root, cfg, reject[0], reject[1])
			}
			if len(unreject) > 0 {
				if len(unreject) != 2 {
					return fmt.Errorf("--unreject 는 슬러그 두 개를 받습니다. 지금 %d개입니다: %s",
						len(unreject), strings.Join(unreject, " "))
				}
				return unrejectPair(cmd, root, unreject[0], unreject[1])
			}

			min, err := cmd.Flags().GetFloat64(flagMin)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagMin, err)
			}
			limit, err := cmd.Flags().GetInt(flagLimit)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagLimit, err)
			}
			// 조회는 색인을 읽기만 한다. 색인이 없으면 안내하고 종료한다. ADR 0025.
			ix := index.Load(root)
			if ix == nil {
				return fmt.Errorf("검색 색인이 없습니다. engram reindex 로 색인을 만든 뒤 다시 실행하세요")
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			stale := false
			if !ix.Fresh(walked, root) {
				stale = true
				fmt.Fprintf(cmd.ErrOrStderr(),
					"경고: 색인이 낡았습니다. 낡은 색인으로 진행합니다. engram reindex로 갱신하세요\n")
			}
			st, err := state.Load(root)
			if err != nil {
				return err
			}
			res := bridge.Run(ix, graph.Build(walked), st, min, limit)
			if jsonOutput(cmd) {
				out := bridgeResponse{Min: min, IndexStale: stale, Pairs: make([]bridgePairJSON, 0, len(res.Pairs))}
				for _, p := range res.Pairs {
					out.Pairs = append(out.Pairs, bridgePairJSON{A: p.A, B: p.B, Score: round2(p.Score)})
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			printBridge(cmd.OutOrStdout(), res, min)
			return nil
		},
	}
	cmd.Flags().Float64(flagMin, 0.30, "코사인 유사도 하한")
	cmd.Flags().Int(flagLimit, 10, "낼 쌍 수 상한")
	cmd.Flags().StringSlice(flagReject, nil, "기각할 슬러그 둘 (예: --reject a b)")
	cmd.Flags().StringSlice(flagUnreject, nil, "기각을 되돌릴 슬러그 둘")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// rejectPair는 쌍을 기각해 상태 파일에 영구 기록한다. 없는 슬러그는
// 거절한다. 오타로 생긴 기각은 조용히 무효가 되어 영영 발견되지 않는다.
func rejectPair(cmd *cobra.Command, root string, cfg config.Config, a, b string) error {
	walked, err := walk.Files(root, cfg)
	if err != nil {
		return fmt.Errorf("위키를 순회할 수 없음: %w", err)
	}
	g := graph.Build(walked)
	var missing []string
	for _, s := range []string{a, b} {
		if !g.Has(s) {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("위키에 없는 슬러그라 기각하지 못했습니다: %s\n슬러그를 확인하세요. 문서를 찾으려면 engram search 를 쓰세요", strings.Join(missing, ", "))
	}
	na, nb := graph.Normalize(a), graph.Normalize(b)
	st, err := state.Load(root)
	if err != nil {
		return err
	}
	if !st.Reject(na, nb) {
		printBridgeAction(cmd, bridgeActionResponse{Action: "reject", A: na, B: nb},
			fmt.Sprintf("이미 기각된 쌍입니다: %s %s", na, nb))
		return nil
	}
	if err := st.Save(root); err != nil {
		return fmt.Errorf("기각을 저장할 수 없음: %w", err)
	}
	printBridgeAction(cmd, bridgeActionResponse{Action: "reject", A: na, B: nb},
		fmt.Sprintf("기각했습니다: %s %s\n%s 에 기록했습니다",
			na, nb, filepath.Join(root, state.StateFileName)))
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
			fmt.Sprintf("기각 기록에 없는 쌍입니다: %s %s", na, nb))
		return nil
	}
	if err := st.Save(root); err != nil {
		return fmt.Errorf("기각 되돌리기를 저장할 수 없음: %w", err)
	}
	printBridgeAction(cmd, bridgeActionResponse{Action: "unreject", A: na, B: nb},
		fmt.Sprintf("기각을 되돌렸습니다: %s %s", na, nb))
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

// printBridge는 후보 탐색의 사람용 출력을 인쇄한다. 재료만 반환하고
// 문장을 만들지 않는다.
func printBridge(w io.Writer, res bridge.Result, min float64) {
	if len(res.Pairs) == 0 {
		s := res.Stats
		fmt.Fprintf(w, "후보가 없습니다\n")
		fmt.Fprintf(w, "  context 문서 %d개, 링크로 이어진 쌍 %d, 기각된 쌍 %d, min %.2f 미달 %d\n",
			s.ContextDocs, s.Linked, s.Rejected, min, s.BelowMin)
		return
	}
	fmt.Fprintf(w, "유사도가 높은데 링크가 없는 문서 쌍 (min %.2f)\n", min)
	for i, p := range res.Pairs {
		fmt.Fprintf(w, "%3d  %.2f  %s  %s\n", i+1, round2(p.Score), p.A, p.B)
		fmt.Fprintf(w, "     기각하려면: engram bridge --reject %s %s\n", p.A, p.B)
	}
}
