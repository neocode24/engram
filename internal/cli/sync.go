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
	"github.com/neocode24/engram/internal/i18n"
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
		Short: i18n.T("cli.sync.short"),
		Long:  i18n.T("cli.sync.long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			apply, err := cmd.Flags().GetBool(flagApply)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.sync.flag_read_fail", flagApply), err)
			}
			hist, err := gitdate.History(root)
			if err != nil {
				return err
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.sync.walk_fail"), err)
			}

			var changes []syncChange
			uncommitted, bulkOnly := 0, 0
			for _, w := range walked {
				if w.Err != nil || !w.Parsed.HasFrontmatter {
					continue // 프론트매터를 읽지 못한 문서는 정정 대상이 아니다
				}
				d, ok := hist[w.Rel]
				if !ok || d.First == "" {
					// 대량 커밋에만 등장한 문서는 커밋이 없는 것과 다른
					// 사유로 건너뛴다. 다음 실제 편집 때 채워진다.
					if ok && d.BulkOnly {
						bulkOnly++
					} else {
						uncommitted++
					}
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

			res := syncOutcome{Applied: apply, Changed: changes, Uncommitted: uncommitted, BulkOnly: bulkOnly}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printSync(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Bool(flagApply, false, i18n.T("cli.sync.flag_apply"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.sync.flag_wiki"))
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
	BulkOnly    int          `json:"bulkOnly"`
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
		return fmt.Errorf("%s: %w", i18n.T("cli.sync.doc_write_fail", path), err)
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
		fmt.Fprint(w, i18n.T("cli.sync.none")+"\n")
	case res.Applied:
		fmt.Fprint(w, i18n.T("cli.sync.applied", len(docs), len(res.Changed))+"\n")
	default:
		fmt.Fprint(w, i18n.T("cli.sync.dry_run", len(docs), len(res.Changed))+"\n")
	}
	for _, c := range res.Changed {
		from := c.From
		if from == "" {
			from = i18n.T("cli.sync.field_absent")
		}
		fmt.Fprintf(w, "  %s %s=%s (%s)\n", c.Path, c.Field, c.To, i18n.T("cli.sync.current", from))
	}
	if res.Uncommitted > 0 {
		fmt.Fprint(w, i18n.T("cli.sync.uncommitted", res.Uncommitted)+"\n")
	}
	if res.BulkOnly > 0 {
		fmt.Fprint(w, i18n.T("cli.sync.bulk_only", res.BulkOnly, gitdate.BulkCommitThreshold)+"\n")
	}
}
