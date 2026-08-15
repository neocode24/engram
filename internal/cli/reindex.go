package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/config"
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
		Use:   "reindex [경로]",
		Short: "검색 색인을 만듭니다",
		Long: `위키를 순회해 검색 색인을 만들고 .engram/index.json에 씁니다.

reindex가 인덱스를 만드는 유일한 커맨드입니다. 조회 커맨드는 색인 파일을
갱신하지 않습니다. 경로를 생략하면 현재 디렉토리입니다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			cfg, err := loadWikiAt(root)
			if err != nil {
				return err
			}
			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			ix, err := index.Build(root, walked, index.DefaultWeights())
			if err != nil {
				return fmt.Errorf("색인을 만들 수 없음: %w", err)
			}
			if err := ix.Save(root); err != nil {
				return fmt.Errorf("색인을 쓸 수 없음: %w", err)
			}
			path := filepath.Join(root, index.IndexDirName, index.IndexFileName)
			fi, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("색인 파일을 확인할 수 없음: %w", err)
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
			fmt.Fprintf(cmd.OutOrStdout(), "색인을 만들었습니다: %s\n", res.Path)
			fmt.Fprintf(cmd.OutOrStdout(), "문서 %d개, 토큰 %d개, 크기 %d 바이트\n",
				res.Docs, res.Tokens, res.IndexBytes)
			return nil
		},
	}
	return cmd
}

// loadWikiAt는 경로가 초기화된 위키인지 검사하고 설정을 반환한다.
// 위키가 아니면 init을 안내한다.
func loadWikiAt(root string) (config.Config, error) {
	if _, err := os.Stat(filepath.Join(root, config.ConfigFileName)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return config.Config{}, fmt.Errorf("위키가 아닌 디렉토리입니다: %s\n먼저 engram init을 실행하세요", root)
		}
		return config.Config{}, fmt.Errorf("대상 경로를 확인할 수 없음: %w", err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return config.Config{}, fmt.Errorf("위키 설정을 읽을 수 없음: %w", err)
	}
	return cfg, nil
}
