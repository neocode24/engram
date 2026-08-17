package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
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
		Use:   "demote " + i18n.T("usage.args.path"),
		Short: i18n.T("cli.demote.short"),
		Long:  i18n.T("cli.demote.long"),
		Args:  cobra.ExactArgs(1),
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
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_read_fail", srcPath), err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_parse_fail", srcPath), err)
			}
			if stage := fieldString(d, "artifact_stage"); stage != "context" {
				return errors.New(i18n.T("cli.demote.not_context", srcPath, stage))
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
				return errors.New(i18n.T("cli.demote.to_invalid", toFlag))
			}

			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.rel_fail"), err)
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
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.walk_fail"), err)
			}
			broken := brokenLinks(walked, slug, rel)
			for _, b := range broken {
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.demote.warn_broken_link",
					b.Path, b.Line, slug))
			}
			derived := listFieldValues(d, "derived_from")
			for _, src := range derived {
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.demote.warn_derived", src))
			}

			destRel, err := wiki.FilePath(cfg, stage, date, slug)
			if err != nil {
				return err
			}
			destPath := filepath.Join(root, filepath.FromSlash(destRel))
			if _, err := os.Stat(destPath); err == nil {
				return errors.New(i18n.T("cli.ingest.dest_exists", destPath))
			}

			fields := demoteFields(d.Fields, stage, cfg)
			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.demote.remove_fail", srcPath), err)
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
	cmd.Flags().String(flagTo, "inbox", i18n.T("cli.demote.flag_to"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
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
	fmt.Fprintln(w, i18n.T("cli.demote.done", res.Stage, res.Path))
	fmt.Fprintln(w, i18n.T("cli.demote.meta", res.Date, res.Slug))
	if len(res.BrokenLinks) > 0 {
		fmt.Fprintln(w, i18n.T("cli.demote.broken_count", len(res.BrokenLinks)))
	}
	if len(res.DerivedFrom) > 0 {
		fmt.Fprintln(w, i18n.T("cli.demote.derived_count", len(res.DerivedFrom)))
	}
	fmt.Fprintln(w, i18n.T("cli.demote.next"))
}
