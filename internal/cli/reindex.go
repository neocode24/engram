package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// reindexResult는 reindex의 요약이다. --json 출력에 그대로 쓰인다.
type reindexResult struct {
	Docs       int    `json:"docs"`
	Tokens     int    `json:"tokens"`
	IndexBytes int64  `json:"indexBytes"`
	Path       string `json:"path"`
}

// newReindexCmd는 위키를 순회해 검색 색인을 만드는 reindex 커맨드를 반환한다.
// reindex가 인덱스를 만드는 유일한 커맨드다. ADR 0025.
func newReindexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reindex " + i18n.T("usage.args.path_opt"),
		Short: i18n.T("cli.reindex.short"),
		Long:  i18n.T("cli.reindex.long"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := pathOrWikiFlag(cmd, args)
			if err != nil {
				return err
			}
			cfg, err := loadWikiAt(root)
			if err != nil {
				return err
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.reindex.walk_fail"), err)
			}
			ix, err := index.Build(root, walked, index.DefaultWeights())
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.reindex.build_fail"), err)
			}
			if err := ix.Save(root); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.reindex.save_fail"), err)
			}
			path := filepath.Join(root, index.IndexDirName, index.IndexFileName)
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.reindex.stat_fail"), err)
			}
			res := reindexResult{
				Docs:       len(ix.Docs),
				Tokens:     len(ix.DF),
				IndexBytes: fi.Size(),
				Path:       path,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			fmt.Fprint(cmd.OutOrStdout(), i18n.T("cli.reindex.done", res.Path)+"\n")
			fmt.Fprint(cmd.OutOrStdout(), i18n.T("cli.reindex.summary", res.Docs, res.Tokens, res.IndexBytes)+"\n")
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// loadWikiAt는 경로가 초기화된 위키인지 검사하고 설정을 반환한다.
// 위키가 아니면 init을 안내한다.
func loadWikiAt(root string) (config.Config, error) {
	if _, err := os.Stat(filepath.Join(root, config.ConfigFileName)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return config.Config{}, fmt.Errorf("%s", i18n.T("cli.reindex.not_wiki", root))
		}
		return config.Config{}, fmt.Errorf("%s: %w", i18n.T("cli.reindex.path_check_fail"), err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return config.Config{}, fmt.Errorf("%s: %w", i18n.T("cli.reindex.config_load_fail"), err)
	}
	return cfg, nil
}
