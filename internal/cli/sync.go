package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/gitdate"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// flagApply는 sync 의 실제 적용 플래그 이름이다. 기본이 dry-run 이므로
// 이 플래그를 줘야 파일을 쓴다.
const flagApply = "apply"

// newSyncCmd는 git 이력에서 날짜 필드를 정정하는 sync 커맨드를 반환한다.
// 날짜의 진실원이 git 이므로 전역 --now 는 이 커맨드의 판정에 쓰지 않는다.
func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "git 이력에서 updated와 sourced_at을 정정합니다",
		Long: `git 이력에서 날짜 필드를 정정합니다.

updated 는 그 파일의 마지막 커밋 날짜로, sourced_at 은 최초 커밋
날짜로 채웁니다. created 는 git 이 모르는 필드라 건드리지 않습니다.
sources 계층 문서에는 updated 를 넣지 않습니다. 원본 보존 계층이라
updated 가 최신이 되면 신선도를 오해하게 만들기 때문입니다.

기본은 dry-run 입니다. 실제로 쓰려면 --apply 를 주세요.
값이 이미 같으면 쓰지 않으므로 두 번 돌려도 같은 결과입니다.
커밋되지 않은 파일은 이력이 없으므로 건너뛰고 개수를 알립니다.
워킹트리가 더러워도 판정 근거가 커밋 이력이라 그대로 돕니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			apply, err := cmd.Flags().GetBool(flagApply)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagApply, err)
			}
			hist, err := gitdate.History(root)
			if err != nil {
				return err
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}

			var changes []syncChange
			uncommitted := 0
			for _, w := range walked {
				if w.Err != nil || !w.Parsed.HasFrontmatter {
					continue // 프론트매터를 읽지 못한 문서는 정정 대상이 아니다
				}
				d, ok := hist[w.Rel]
				if !ok || d.First == "" {
					uncommitted++
					continue
				}
				cs := dateChanges(w.Rel, w.Parsed, d)
				if len(cs) == 0 {
					continue
				}
				if apply {
					if err := writeDates(root, w.Rel, w.Parsed, cs); err != nil {
						return err
					}
				}
				changes = append(changes, cs...)
			}

			res := syncOutcome{Applied: apply, Changed: changes, Uncommitted: uncommitted}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printSync(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Bool(flagApply, false, "변경을 파일에 씁니다. 기본은 dry-run 입니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// syncChange는 문서 하나의 필드 하나가 바뀌는 내용이다.
type syncChange struct {
	Path  string `json:"path"`
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// syncOutcome은 sync 의 결과다. Changed 는 dry-run 이면 바뀔 예정,
// 적용이면 바꾼 내용이다.
type syncOutcome struct {
	Applied     bool         `json:"applied"`
	Changed     []syncChange `json:"changed"`
	Uncommitted int          `json:"uncommitted"`
}

// dateChanges는 문서의 현재 값과 git 이력을 비교해 정정할 필드를 뽑는다.
// 값이 이미 같으면 아무것도 내지 않는다. sources 계층은 updated 를 채우지
// 않는다(ADR 0009). created 는 git 이 모르는 필드라 다루지 않는다(ADR 0037).
// 계층 판정은 wiki 의 단계 대응 표가 진실원이다.
func dateChanges(rel string, d doc.Doc, dt gitdate.Dates) []syncChange {
	var cs []syncChange
	if stage, ok := wiki.StageForDir(topDir(rel)); !ok || stage != wiki.StageSource {
		if cur := scalarField(d, "updated"); cur != dt.Last {
			cs = append(cs, syncChange{Path: rel, Field: "updated", From: cur, To: dt.Last})
		}
	}
	if cur := scalarField(d, "sourced_at"); cur != dt.First {
		cs = append(cs, syncChange{Path: rel, Field: "sourced_at", From: cur, To: dt.First})
	}
	return cs
}

// writeDates는 정정할 필드를 문서에 반영해 쓴다. upsertField 가 키
// 위치를 보존하므로 프론트매터 순서는 그대로다.
func writeDates(root, rel string, d doc.Doc, cs []syncChange) error {
	fields := append([]doc.Field(nil), d.Fields...)
	for _, c := range cs {
		fields = upsertField(fields, doc.Field{Key: c.Field, Kind: doc.KindDate, Str: c.To})
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.WriteFile(path, doc.Render(fields, d.Body), 0o644); err != nil {
		return fmt.Errorf("문서를 쓸 수 없음: %s: %w", path, err)
	}
	return nil
}

// topDir은 경로의 첫 디렉토리를 반환한다. 위키 루트 파일이면 파일명 그대로다.
func topDir(rel string) string {
	seg, _, _ := strings.Cut(rel, "/")
	return seg
}

// printSync는 정정 내용을 낸다. dry-run 이면 예정임이 드러나게 쓴다.
func printSync(w io.Writer, res syncOutcome) {
	docs := map[string]bool{}
	for _, c := range res.Changed {
		docs[c.Path] = true
	}
	switch {
	case len(res.Changed) == 0:
		fmt.Fprintf(w, "정정할 문서가 없습니다\n")
	case res.Applied:
		fmt.Fprintf(w, "정정했습니다. 문서 %d개, 필드 %d개\n", len(docs), len(res.Changed))
	default:
		fmt.Fprintf(w, "정정 대상 문서 %d개, 필드 %d개. 아직 dry-run 입니다. 적용하려면 --apply 를 주세요\n", len(docs), len(res.Changed))
	}
	for _, c := range res.Changed {
		from := c.From
		if from == "" {
			from = "(없음)"
		}
		fmt.Fprintf(w, "  %s %s=%s (현재 %s)\n", c.Path, c.Field, c.To, from)
	}
	if res.Uncommitted > 0 {
		fmt.Fprintf(w, "커밋되지 않은 문서 %d개는 건너뛰었습니다\n", res.Uncommitted)
	}
}
