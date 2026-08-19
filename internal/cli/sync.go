package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
			uncommitted, bulkOnly, fromFilename := 0, 0, 0
			for _, w := range walked {
				if w.Err != nil || !w.Parsed.HasFrontmatter {
					continue // 프론트매터를 읽지 못한 문서는 정정 대상이 아니다
				}
				d, inHist := hist[w.Rel]
				// 파일명 접두사는 git 이력이 없을 때의 대체 재료다. 이력이
				// 있으면 파일명은 보지 않는다(ADR 0072). 두 값이 다를 때
				// 어느 쪽이 옳은지 정할 근거가 없기 때문이다.
				fn := ""
				if d.First == "" {
					fn = filenameDate(w.Rel)
				}
				cs := dateChanges(w.Rel, w.Parsed, d.First, d.Last, fn)
				if len(cs) > 0 {
					if d.First == "" && fn != "" {
						fromFilename += len(cs)
					}
					if apply {
						if err := writeDates(root, w.Rel, w.Parsed, cs); err != nil {
							return err
						}
					}
					changes = append(changes, cs...)
					continue
				}
				// 채울 것이 없는 문서는 사유를 나눠 알린다. 대량 커밋에만
				// 등장한 문서는 커밋이 없는 것과 다른 사유로 건너뛴다.
				// 다음 실제 편집 때 채워진다.
				if inHist && d.BulkOnly {
					bulkOnly++
				} else if !inHist {
					uncommitted++
				}
			}

			res := syncOutcome{Applied: apply, Changed: changes, Uncommitted: uncommitted,
				BulkOnly: bulkOnly, FilenameDates: fromFilename}
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
	// FilenameDates는 파일명 접두사에서 얻은 필드 수다. git 이력에서 온
	// 것과 나눠 알려 사용자가 근거를 알게 한다(ADR 0072).
	FilenameDates int `json:"filenameDates"`
}

// dateChanges는 문서의 현재 값과 날짜 재료를 비교해 정정할 필드를 뽑는다.
// 값이 이미 같으면 아무것도 내지 않는다. 필드마다 쓰는 단계가 정해져
// 있다(ADR 0072). updated 는 재발견이 읽고 재발견 대상은 context 뿐이므로
// context 단계 문서에만 쓰고, sourced_at 은 원본 입수 시각이라 sources 단계
// 문서에만 쓴다. 아무도 읽지 않는 단계에 값을 쓰면 그 값에 뜻이 있다고
// 믿게 만든다. created 는 단계를 가리지 않되 비어 있을 때만 채운다.
// 이미 있는 값은 사람이 정한 것이므로 건드리지 않는다. 계층 판정은 wiki
// 의 단계 대응 표가 진실원이다.
func dateChanges(rel string, d doc.Doc, gitFirst, gitLast, fn string) []syncChange {
	var cs []syncChange
	stage, stageOK := wiki.StageForDir(topDir(rel))
	if stageOK && stage == wiki.StageContext && gitLast != "" {
		if cur := scalarField(d, "updated"); cur != gitLast {
			cs = append(cs, syncChange{Path: rel, Field: "updated", From: cur, To: gitLast})
		}
	}
	if stageOK && stage == wiki.StageSource {
		switch {
		case gitFirst != "":
			// git 이력이 있으면 정정한다. 판정 근거가 git 이기 때문이다.
			if cur := scalarField(d, "sourced_at"); cur != gitFirst {
				cs = append(cs, syncChange{Path: rel, Field: "sourced_at", From: cur, To: gitFirst})
			}
		case fn != "" && scalarField(d, "sourced_at") == "":
			// 파일명은 이력이 없을 때의 대체 재료다. 이미 있는 값은
			// 사람이 정한 것이므로 덮어 쓰지 않는다.
			cs = append(cs, syncChange{Path: rel, Field: "sourced_at", From: "", To: fn})
		}
	}
	// created 는 채울 재료가 있고 값이 비어 있을 때만 채운다.
	first := gitFirst
	if first == "" {
		first = fn
	}
	if first != "" && scalarField(d, "created") == "" {
		cs = append(cs, syncChange{Path: rel, Field: "created", From: "", To: first})
	}
	return cs
}

// filenameDate는 파일명 앞의 날짜 접두사를 낸다. YYYY-MM-DD- 와 YYYY-MM-
// 형태를 받고 뒤에 슬러그가 이어져야 한다. 날짜만 있는 파일명은 받지
// 않는다. 연월 정밀도 자료를 파일명으로 남기는 운용이 upstream 에서
// 왔다(ADR 0072).
func filenameDate(rel string) string {
	base := strings.TrimSuffix(filepath.Base(rel), ".md")
	m := regexp.MustCompile(`^(\d{4}-\d{2}(?:-\d{2})?)-(.+)$`).FindStringSubmatch(base)
	if m == nil || !validDatePrecision(m[1]) {
		return ""
	}
	return m[1]
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
	if res.FilenameDates > 0 {
		fmt.Fprint(w, i18n.T("cli.sync.filename_dates", res.FilenameDates)+"\n")
	}
}
