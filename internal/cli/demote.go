package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// flagTo는 demote의 도착 단계 플래그 이름이다.
const flagTo = "to"

// newDemoteCmd는 잘못 올린 문서를 되돌리는 demote 커맨드를 반환한다.
// promote의 역동작이다. 되돌리기를 막으면 게이트를 강제하는 제품의
// 신뢰가 깨지므로 링크가 깨져도 경고로만 알리고 진행한다.
func newDemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demote <경로>",
		Short: "context 문서를 inbox나 sources로 되돌립니다",
		Long: `context 단계 문서를 inbox 또는 sources로 내립니다.

도착 단계의 기본값은 inbox다. inbox가 임시 계층이라 되돌리기의
도착지로 안전하입니다.

문서를 내리면 파일명에 날짜 접두사가 붙어 슬러그가 바뀝니다. 그 문서를
가리키던 위키링크는 전부 깨지므로 실행 전에 목록을 경고로 냅니다.
되돌리기를 막지는 않습니다. 깨진 링크는 engram mv로 고치세요.

파생 문서라면 원본 sources 문서의 derived_context도 어긋납니다.
원본 되돌리기는 이 커맨드의 범위가 아니므로 경고로만 알립니다.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			srcPath, err := resolveDocPath(root, args[0])
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("문서를 읽을 수 없음: %s: %w", srcPath, err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("문서를 파싱할 수 없음: %s: %w", srcPath, err)
			}
			if stage := fieldString(d, "artifact_stage"); stage != "context" {
				return fmt.Errorf("context 단계 문서만 되돌립니다: %s의 artifact_stage 값이 %q다", srcPath, stage)
			}

			toFlag, err := stringFlag(cmd, flagTo)
			if err != nil {
				return err
			}
			var stage wiki.Stage
			switch toFlag {
			case "":
				stage = wiki.StageInbox
			case "inbox":
				stage = wiki.StageInbox
			case "sources":
				stage = wiki.StageSource
			default:
				return fmt.Errorf("--to 값이 허용값 밖입니다: %q (허용값: inbox, sources)", toFlag)
			}

			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("위키 루트 기준 경로를 계산할 수 없음: %w", err)
			}
			rel = filepath.ToSlash(rel)
			slug := stripDatePrefix(filepath.Base(srcPath))

			// 날짜 접두사는 프론트매터 created를 쓰고 없으면 기준 시각을 쓴다.
			// created 는 파서가 날짜 종류로 분류하므로 문자열 전용 조회로는
			// 읽히지 않는다.
			date := scalarField(d, "created")
			if date == "" {
				date = Now(cmd).Format("2006-01-02")
			}

			// 깨질 링크를 먼저 본다. 순회해서 최소로만 센다.
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			broken := brokenLinks(walked, slug, rel)
			for _, b := range broken {
				fmt.Fprintf(cmd.ErrOrStderr(), "경고: 깨질 위키링크: %s %d줄이 [[%s]]를 가리킵니다. engram mv로 링크를 고치세요\n",
					b.Path, b.Line, slug)
			}
			derived := listFieldValues(d, "derived_from")
			for _, src := range derived {
				fmt.Fprintf(cmd.ErrOrStderr(), "경고: 이 문서는 원본 %s에서 파생되었습니다. 원본의 derived_context 갱신은 이 커맨드가 하지 않습니다. engram update로 직접 고치세요\n", src)
			}

			destRel, err := wiki.FilePath(cfg, stage, date, slug)
			if err != nil {
				return err
			}
			destPath := filepath.Join(root, filepath.FromSlash(destRel))
			if _, err := os.Stat(destPath); err == nil {
				return fmt.Errorf("도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 슬러그를 다르게 지정하세요", destPath)
			}

			fields := demoteFields(d.Fields, stage, cfg)
			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("context 원본을 지울 수 없음: %s: %w", srcPath, err)
			}

			res := demoteOutcome{
				Path:        destPath,
				Slug:        slug,
				Stage:       string(stage),
				Date:        date,
				BrokenLinks: broken,
				DerivedFrom: derived,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printDemoted(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagTo, "inbox", "도착 단계. 허용값: inbox, sources")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// demoteOutcome은 demote의 결과다. 깨질 링크와 파생 원본도 함께 낸다.
type demoteOutcome struct {
	Path        string       `json:"path"`
	Slug        string       `json:"slug"`
	Stage       string       `json:"stage"`
	Date        string       `json:"date"`
	BrokenLinks []brokenLink `json:"brokenLinks"`
	DerivedFrom []string     `json:"derivedFrom"`
}

// brokenLink는 깨질 위키링크 하나의 출처다.
type brokenLink struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// demoteFields는 도착 단계에 맞게 프론트매터를 되돌린다.
// 바꾸는 것은 artifact_stage, status, indexable 셋뿐이다. 값의 진실원은
// wiki의 단계별 초기값이라 여기에 목록을 두지 않는다. 기존 키 순서는
// 그대로 보존한다.
func demoteFields(src []doc.Field, stage wiki.Stage, cfg config.Config) []doc.Field {
	fm := wiki.Frontmatter(stage, cfg)
	fields := append([]doc.Field(nil), src...)
	fields = upsertField(fields, emptyField("artifact_stage", fm["artifact_stage"]))
	fields = upsertField(fields, emptyField("status", fm["status"]))
	fields = upsertField(fields, emptyField("indexable", fm["indexable"]))
	return fields
}

// brokenLinks는 슬러그를 가리키는 위키링크의 출처를 모은다.
// 자기 자신의 링크는 세지 않는다. 링크 조회는 internal/graph 가 생기면
// 그쪽으로 통일한다. 지금은 필요한 만큼만 최소로 센다.
func brokenLinks(walked []walk.Doc, slug, selfRel string) []brokenLink {
	var out []brokenLink
	for _, w := range walked {
		if w.Err != nil || !w.Parsed.HasFrontmatter || w.Rel == selfRel {
			continue
		}
		for _, l := range append(append([]doc.Link(nil), w.Parsed.FrontmatterLinks()...), w.Parsed.BodyLinks()...) {
			if l.Slug == slug {
				out = append(out, brokenLink{Path: w.Rel, Line: l.Line})
			}
		}
	}
	return out
}

// scalarField는 문자열이나 날짜 종류의 스칼라 값을 반환한다.
func scalarField(d doc.Doc, key string) string {
	for _, f := range d.Fields {
		if f.Key == key && (f.Kind == doc.KindString || f.Kind == doc.KindDate) {
			return f.Str
		}
	}
	return ""
}

// listFieldValues는 목록 필드의 값을 꺼낸다.
func listFieldValues(d doc.Doc, key string) []string {
	for _, f := range d.Fields {
		if f.Key == key && f.Kind == doc.KindStringList {
			return f.List
		}
	}
	return nil
}

// printDemoted은 내린 경로와 다음에 할 수 있는 일을 낸다.
func printDemoted(w io.Writer, res demoteOutcome) {
	fmt.Fprintf(w, "%s로 내렸습니다: %s\n", res.Stage, res.Path)
	fmt.Fprintf(w, "날짜 접두사: %s, 슬러그: %s\n", res.Date, res.Slug)
	if len(res.BrokenLinks) > 0 {
		fmt.Fprintf(w, "깨질 링크 %d건. engram mv로 고치세요\n", len(res.BrokenLinks))
	}
	if len(res.DerivedFrom) > 0 {
		fmt.Fprintf(w, "파생 원본 %d건의 derived_context가 어긋났습니다\n", len(res.DerivedFrom))
	}
	fmt.Fprintf(w, "다음: 정리가 끝나면 engram promote로 다시 올리세요\n")
}
