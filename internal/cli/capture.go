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
	"github.com/neocode24/engram/internal/i18n"
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
		Use:   "capture " + i18n.T("usage.args.content"),
		Short: i18n.T("cli.capture.short"),
		Long:  i18n.T("cli.capture.long"),
		Args:  cobra.ArbitraryArgs,
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
			titleFromFlag := title != ""
			title = deriveTitle(title, content)
			content = withHeading(title, content, titleFromFlag)
			slug, err := resolveSlug(slugFlag, title)
			if err != nil {
				return err
			}
			fm := wiki.Frontmatter(wiki.StageInbox, cfg)
			date := Now(cmd).Format("2006-01-02")
			// created를 프론트매터에도 남긴다. 파일명 접두사만으로는
			// 승급 시 날짜가 사라져 resurface와 digest가 이 문서를
			// 대상에서 빼게 된다.
			fm["created"] = date
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
			printIngested(cmd.OutOrStdout(), res, i18n.T("cli.capture.next_hint"))
			return nil
		},
	}
	cmd.Flags().String(flagTitle, "", i18n.T("cli.ingest.flag_title"))
	cmd.Flags().String(flagSlug, "", i18n.T("cli.ingest.flag_slug"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
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
			return "", config.Config{}, errors.New(i18n.T("cli.ingest.not_wiki", root))
		}
		return "", config.Config{}, fmt.Errorf("%s: %w", i18n.T("cli.ingest.stat_fail"), err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return "", config.Config{}, fmt.Errorf("%s: %w", i18n.T("cli.ingest.config_load_fail"), err)
	}
	return root, cfg, nil
}

// stringFlag는 문자열 플래그 값을 읽는다.
func stringFlag(cmd *cobra.Command, name string) (string, error) {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", name), err)
	}
	return v, nil
}

// boolFlag는 참/거짓 플래그 값을 읽는다.
func boolFlag(cmd *cobra.Command, name string) (bool, error) {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", name), err)
	}
	return v, nil
}

// pathOrWikiFlag는 위치 인자와 --wiki 플래그 중 실제로 주어진 것을 위키
// 경로로 고른다. 커맨드 스물이 --wiki를 받는데 lint, status, reindex,
// doctor 넷만 위치 인자였다. 에이전트가 플래그를 전 커맨드에 통하는
// 것으로 보고 부르다 막혔다([ADR 0053]). 둘 다 주면 어느 쪽을 뜻하는지
// 알 수 없으므로 거절한다.
func pathOrWikiFlag(cmd *cobra.Command, args []string) (string, error) {
	root, err := pickWikiPath(cmd, args)
	if err != nil {
		return "", err
	}
	// 파일을 주면 그대로 통과시켰다가 <파일>/engram.yaml 을 여는 데서
	// 깨졌다. 에이전트가 문서 하나만 검사하려고 파일 경로를 준다.
	if fi, err := os.Stat(root); err == nil && !fi.IsDir() {
		return "", errors.New(i18n.T("cli.wiki_path.not_dir", root))
	}
	return root, nil
}

func pickWikiPath(cmd *cobra.Command, args []string) (string, error) {
	if cmd.Flags().Changed(flagWiki) {
		if len(args) == 1 {
			return "", errors.New(i18n.T("cli.wiki_path.both_given"))
		}
		return stringFlag(cmd, flagWiki)
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return ".", nil
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
			return "", fmt.Errorf("%s: %w", i18n.T("cli.ingest.stdin_read_fail"), err)
		}
		if s := strings.TrimSpace(string(data)); s != "" {
			return s, nil
		}
	}
	return "", errors.New(i18n.T("cli.ingest.no_content"))
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

// withHeading은 본문 맨 앞에 제목 헤딩을 붙인다.
//
// 제목이 파일명 슬러그로만 남으면 사라진다. 슬러그는 공백을 하이픈으로
// 바꾸고 대소문자와 구두점을 잃으므로 되돌려도 원문이 아니다. 본문에
// 헤딩을 두면 문서가 스스로 제목을 들고 다니고, index의 docTitle이
// 슬러그 역변환 대신 이 헤딩을 읽는다. new와 init이 이미 같은 방식이다.
//
// 본문이 이미 헤딩으로 시작하면 붙이지 않는다. --title을 생략해 본문
// 첫 줄에서 제목을 뽑은 경우도 붙이지 않는다. 첫 줄이 곧 제목이라
// 같은 문장이 두 번 나오기 때문이다.
func withHeading(title, content string, titleFromFlag bool) string {
	if !titleFromFlag || title == "" {
		return content
	}
	if strings.HasPrefix(strings.TrimSpace(content), "#") {
		return content
	}
	return "# " + title + "\n\n" + content
}

// resolveSlug는 슬러그 플래그가 비었으면 제목에서 슬러그를 만든다. ADR 0020.
//
// 직접 지정한 슬러그에는 파생 규칙(소문자, 하이픈 정규화)을 적용하지 않는다.
// 대문자와 공백은 그대로 두고 파일시스템 안전 검사만 받는다. 어기면 조용히
// 고치지 않고 거절한다(ADR 0045). 검사는 파생 경로와 같은 wiki.ValidateSlug다.
// CLI의 --slug와 MCP capture 도구의 slug 인자가 이 함수를 함께 지난다.
func resolveSlug(flagValue, title string) (string, error) {
	if flagValue != "" {
		if err := wiki.ValidateSlug(flagValue); err != nil {
			return "", err
		}
		return flagValue, nil
	}
	return wiki.Slug(title)
}

// printIngested는 만들어진 경로와 다음에 할 수 있는 일을 낸다.
func printIngested(w io.Writer, res ingestResult, next string) {
	fmt.Fprintln(w, i18n.T("cli.ingest.stage_put", res.Stage, res.Path))
	fmt.Fprintln(w, i18n.T("cli.ingest.next", next))
}
