package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// archiveOutcome은 archive의 결과다.
type archiveOutcome struct {
	Path          string `json:"path"`
	Slug          string `json:"slug"`
	IncomingLinks int    `json:"incomingLinks"`
}

// newArchiveCmd는 수명이 끝난 문서를 보관하는 archive 커맨드를 반환한다.
// demote가 승급이 잘못됐을 때 쓰는 것과 달리 archive는 수명이 끝났을 때
// 쓴다. 슬러그를 바꾸지 않으므로 들어오는 링크는 깨지지 않는다. ADR 0028.
func newArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <경로>",
		Short: i18n.T("cli.archive.short"),
		Long:  i18n.T("cli.archive.long"),
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
			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.rel_fail"), err)
			}
			rel = filepath.ToSlash(rel)

			dir, err := wiki.DirFor(cfg, wiki.StageArchive)
			if err != nil {
				return err
			}
			if strings.HasPrefix(rel, dir+"/") {
				return errors.New(i18n.T("cli.archive.already_archived", srcPath))
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
				return errors.New(i18n.T("cli.archive.not_context", srcPath, stage))
			}

			// 슬러그와 파일명은 그대로 둔다. 들어오는 링크가 유지되는 것이
			// 이 커맨드의 계약이다.
			name := filepath.Base(srcPath)
			destPath := filepath.Join(root, dir, name)
			if _, err := os.Stat(destPath); err == nil {
				return errors.New(i18n.T("cli.archive.dest_exists", destPath))
			}

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.walk_fail"), err)
			}
			slug := stripDatePrefix(name)
			incoming := 0
			for _, l := range graph.Build(walked).Backlinks(slug) {
				if l.From != rel {
					incoming++
				}
			}
			if incoming > 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.archive.notice_incoming", incoming))
			}

			fields := archiveFields(d.Fields, Now(cmd).Format("2006-01-02"))
			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.archive.remove_fail", srcPath), err)
			}

			res := archiveOutcome{Path: destPath, Slug: slug, IncomingLinks: incoming}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printArchived(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// archiveFields는 보관에 맞게 프론트매터를 갱신한다. 바꾸는 것은
// artifact_stage, status, updated 셋뿐이다. 기존 키 순서는 그대로
// 보존한다.
func archiveFields(src []doc.Field, now string) []doc.Field {
	fields := append([]doc.Field(nil), src...)
	fields = upsertField(fields, doc.Field{Key: "artifact_stage", Kind: doc.KindString, Str: "archive"})
	fields = upsertField(fields, doc.Field{Key: "status", Kind: doc.KindString, Str: "archived"})
	fields = upsertField(fields, doc.Field{Key: "updated", Kind: doc.KindDate, Str: now})
	return fields
}

// printArchived는 보관 결과를 낸다.
func printArchived(w io.Writer, res archiveOutcome) {
	fmt.Fprintln(w, i18n.T("cli.archive.done", res.Path))
	fmt.Fprintln(w, i18n.T("cli.archive.slug_kept", res.Slug))
	if res.IncomingLinks > 0 {
		fmt.Fprintln(w, i18n.T("cli.archive.incoming_now_dead", res.IncomingLinks))
	}
}
