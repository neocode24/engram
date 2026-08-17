package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// backlinkHit는 --json이 내는 백링크 한 건이다.
type backlinkHit struct {
	Path  string     `json:"path"`
	Line  int        `json:"line"`
	Field graph.Kind `json:"field"`
	Slug  string     `json:"slug"`
	Raw   string     `json:"raw"`
}

// backlinksResponse는 backlinks의 --json 출력이다.
type backlinksResponse struct {
	Slug      string        `json:"slug"`
	Exists    bool          `json:"exists"`
	Backlinks []backlinkHit `json:"backlinks"`
}

// newBacklinksCmd는 슬러그를 가리키는 링크를 조회하는 backlinks 커맨드를
// 반환한다. 조회는 판정이 아니므로 결과가 없어도 종료 코드는 0이다.
func newBacklinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backlinks " + i18n.T("usage.args.slug"),
		Short: i18n.T("cli.backlinks.short"),
		Long:  i18n.T("cli.backlinks.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.backlinks.walk_fail"), err)
			}
			g := graph.Build(walked)
			links := g.Backlinks(slug)
			res := backlinksResponse{
				Slug:      slug,
				Exists:    g.Has(slug),
				Backlinks: make([]backlinkHit, 0, len(links)),
			}
			for _, l := range links {
				res.Backlinks = append(res.Backlinks, backlinkHit{
					Path: l.From, Line: l.Line, Field: l.Field, Slug: l.Slug, Raw: l.Raw,
				})
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printBacklinks(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.backlinks.flag_wiki"))
	return cmd
}

// printBacklinks는 사람용 백링크 목록을 인쇄한다.
func printBacklinks(w io.Writer, res backlinksResponse) {
	if !res.Exists {
		fmt.Fprintln(w, i18n.T("cli.backlinks.missing_notice", res.Slug))
	}
	if len(res.Backlinks) == 0 {
		fmt.Fprintln(w, i18n.T("cli.backlinks.no_backlinks"))
		return
	}
	for _, b := range res.Backlinks {
		fmt.Fprintf(w, "%s:%d  %s  %s\n", b.Path, b.Line, b.Field.Label(), b.Raw)
	}
}
