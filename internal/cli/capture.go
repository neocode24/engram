package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// capture와 source가 함께 쓰는 플래그 이름이다.
const (
	flagTitle = "title"
	flagSlug  = "slug"
	flagWiki  = "wiki"
)

// ingestResult는 capture와 source의 공통 결과다. --json 출력에 그대로 쓰인다.
type ingestResult struct {
	Path  string `json:"path"`
	Slug  string `json:"slug"`
	Stage string `json:"stage"`
}

// newCaptureCmd는 새 메모를 검증 없이 inbox에 넣는 capture 커맨드를 반환한다.
// 회의 중에 쓰는 명령이라 마찰이 있으면 안 된다. 거절하는 유일한 경우는
// 같은 이름의 파일이 이미 있을 때다.
func newCaptureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture [내용]",
		Short: "새 메모를 검증 없이 inbox에 넣는다",
		Long: `새 메모를 inbox에 넣는다. 스키마 검증을 하지 않는다.

내용을 인자로 받거나 파이프로 연결된 표준 입력으로 받는다.
제목은 --title으로 주고, 생략하면 본문 첫 줄에서 만든다.
파일명은 <날짜>-<슬러그>.md이고 날짜는 전역 --now 기준이다.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			content, err := readContent(cmd, args)
			if err != nil {
				return err
			}
			title, err := stringFlag(cmd, flagTitle)
			if err != nil {
				return err
			}
			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			title = deriveTitle(title, content)
			slug, err := resolveSlug(slugFlag, title)
			if err != nil {
				return err
			}
			fm := wiki.Frontmatter(wiki.StageInbox, cfg)
			date := Now(cmd).Format("2006-01-02")
			path, err := wiki.Create(root, cfg, wiki.StageInbox, date, slug, fm, content)
			if err != nil {
				return err
			}
			res := ingestResult{Path: path, Slug: slug, Stage: string(wiki.StageInbox)}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printIngested(cmd.OutOrStdout(), res, "문서를 정리한 뒤 승급한다. 지금은 engram lint로 위키 상태를 본다")
			return nil
		},
	}
	cmd.Flags().String(flagTitle, "", "문서 제목. 생략하면 본문 첫 줄에서 만든다")
	cmd.Flags().String(flagSlug, "", "파일명 슬러그. 생략하면 제목에서 만든다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// ingestTarget는 --wiki 플래그가 가리키는 위키의 설정을 반환한다.
// 대상이 초기화된 위키가 아니면 init을 안내한다.
func ingestTarget(cmd *cobra.Command) (string, config.Config, error) {
	root, err := stringFlag(cmd, flagWiki)
	if err != nil {
		return "", config.Config{}, err
	}
	if _, err := os.Stat(filepath.Join(root, config.ConfigFileName)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", config.Config{}, fmt.Errorf("위키가 아닌 디렉토리다: %s\n먼저 engram init을 실행한다", root)
		}
		return "", config.Config{}, fmt.Errorf("대상 경로를 확인할 수 없음: %w", err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return "", config.Config{}, fmt.Errorf("위키 설정을 읽을 수 없음: %w", err)
	}
	return root, cfg, nil
}

// stringFlag는 문자열 플래그 값을 읽는다.
func stringFlag(cmd *cobra.Command, name string) (string, error) {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", name, err)
	}
	return v, nil
}

// readContent는 인자로 받은 내용이나 파이프로 연결된 표준 입력을 읽는다.
// 인자가 우선이다. 둘 다 없으면 사용법을 함께 안내한다.
// 실제 파일인 입력은 캐릭터 장치(터미널)가 아닐 때만 읽는다. 테스트가
// 주입하는 메모리 리더는 파이프로 본다.
func readContent(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	read := false
	if f, ok := cmd.InOrStdin().(*os.File); ok {
		if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
			read = true
		}
	} else {
		read = true
	}
	if read {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("표준 입력을 읽을 수 없음: %w", err)
		}
		if s := strings.TrimSpace(string(data)); s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("내용을 받지 못했다\n인자로 내용을 주거나 파이프로 표준 입력을 넘긴다. 예: engram capture \"메모 내용\"")
}

// deriveTitle은 제목 플래그가 비었으면 본문 첫 줄에서 제목을 만든다.
// 첫 줄이 마크다운 헤딩이면 헤딩 기호를 뗀다.
func deriveTitle(flagValue, content string) string {
	if flagValue != "" {
		return flagValue
	}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if t != "" {
			return t
		}
	}
	return "무제"
}

// resolveSlug는 슬러그 플래그가 비었으면 제목에서 슬러그를 만든다. ADR 0020.
// 직접 지정한 슬러그는 경로 구분자를 포함하지 않는지만 검사한다.
func resolveSlug(flagValue, title string) (string, error) {
	if flagValue != "" {
		if strings.ContainsAny(flagValue, `/\`) {
			return "", fmt.Errorf("슬러그에 경로 구분자가 들어 있다: %q", flagValue)
		}
		return flagValue, nil
	}
	return wiki.Slug(title)
}

// printIngested는 만들어진 경로와 다음에 할 수 있는 일을 낸다.
func printIngested(w io.Writer, res ingestResult, next string) {
	fmt.Fprintf(w, "%s에 넣었다: %s\n", res.Stage, res.Path)
	fmt.Fprintf(w, "다음: %s\n", next)
}
