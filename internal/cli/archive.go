package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/graph"
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
		Short: "수명이 끝난 context 문서를 보관합니다",
		Long: `context 단계 문서를 archive 디렉토리로 옮깁니다.

수명이 끝난 문서를 걷어 내되 링크는 깨지 않습니다. 슬러그를 바꾸지
않고 날짜 접두사도 붙이지 않으므로 이 문서를 가리키던 위키링크는
그대로 남습니다. 폐기된 결정을 가리키는 링크가 깨지면 안 되기
때문입니다. 승급이 잘못됐다면 engram demote를 쓰세요.

프론트매터의 artifact_stage를 archive로, status를 archived로 바꾸고
updated를 갱신합니다. 이 문서를 가리키는 링크가 있으면 개수를
알립니다. 막지는 않습니다.`,
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
			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("위키 루트 기준 경로를 계산할 수 없음: %w", err)
			}
			rel = filepath.ToSlash(rel)

			dir, err := wiki.DirFor(cfg, wiki.StageArchive)
			if err != nil {
				return err
			}
			if strings.HasPrefix(rel, dir+"/") {
				return fmt.Errorf("이미 보관 상태입니다: %s", srcPath)
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
				return fmt.Errorf("context 단계 문서만 보관합니다: %s의 artifact_stage 값이 %q입니다\n수명이 끝난 context 문서가 대상입니다. 승급이 잘못됐다면 engram demote를 쓰세요", srcPath, stage)
			}

			// 슬러그와 파일명은 그대로 둔다. 들어오는 링크가 유지되는 것이
			// 이 커맨드의 계약이다.
			name := filepath.Base(srcPath)
			destPath := filepath.Join(root, dir, name)
			if _, err := os.Stat(destPath); err == nil {
				return fmt.Errorf("도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 보관 디렉토리에서 정리한 뒤 다시 시도하세요", destPath)
			}

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			slug := stripDatePrefix(name)
			incoming := 0
			for _, l := range graph.Build(walked).Backlinks(slug) {
				if l.From != rel {
					incoming++
				}
			}
			if incoming > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"알림: 이 문서를 가리키는 링크 %d개가 있습니다. 링크는 깨지지 않고 가리키는 대상이 폐기 상태가 됩니다\n", incoming)
			}

			fields := archiveFields(d.Fields, Now(cmd).Format("2006-01-02"))
			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("보관 전 원본을 지울 수 없음: %s: %w", srcPath, err)
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
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
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
	fmt.Fprintf(w, "보관했습니다: %s\n", res.Path)
	fmt.Fprintf(w, "슬러그 %s는 유지됩니다. 들어오는 링크는 깨지지 않습니다\n", res.Slug)
	if res.IncomingLinks > 0 {
		fmt.Fprintf(w, "이 문서를 가리키던 링크 %d개가 폐기 상태를 가리키게 됩니다\n", res.IncomingLinks)
	}
}
