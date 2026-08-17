package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/export"
	"github.com/spf13/cobra"
)

// export 커맨드의 플래그 이름이다.
const (
	flagExportOut          = "out"
	flagExportReplacements = "replacements"
	flagExportDryRun       = "dry-run"
)

// newExportCmd는 위키의 일부를 밖으로 내보낼 번들을 만드는 export 커맨드를 반환한다.
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [슬러그...]",
		Short: "검수된 문서를 익명화해 반출 번들로 내보냅니다",
		Long: `검수를 지난 문서를 --out 디렉토리에 마크다운 그대로 내보냅니다.

병합도 압축도 포맷 변환도 하지 않습니다. 보고서나 발표자료로 만들려면
pandoc 같은 도구를 뒤에 붙이세요.

나가는 것은 serve 와 같은 규칙입니다. context 문서와 색인 문서만 나가고
inbox 와 sources 는 나가지 않습니다. archive 는 --include-archive 로
엽니다. sensitivity 축이 켜진 위키에서는 private-local-only 와 restricted
문서를 뺍니다. 슬러그로 지목해도 이 제외는 뚫리지 않습니다. 반출해야
하면 문서의 값을 고치세요.

슬러그를 주면 그 문서만 나갑니다. 링크를 따라가지 않으므로 함께
내보낼 문서는 함께 적으세요.

익명화는 --replacements 파일로 합니다. 한 줄에 하나씩 원문==>대체어
형식이며 # 으로 시작하는 줄은 건너뜁니다. 본문과 프론트매터와 파일명
전부에 적용합니다. 파일을 주지 않으면 치환하지 않습니다.

--dry-run 은 무엇이 나갈지 봅니다. 파일을 쓰지 않습니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			out, err := stringFlag(cmd, flagExportOut)
			if err != nil {
				return err
			}
			if out == "" {
				return fmt.Errorf("--%s 로 내보낼 디렉토리를 지정하세요", flagExportOut)
			}
			replPath, err := stringFlag(cmd, flagExportReplacements)
			if err != nil {
				return err
			}
			includeArchive, err := cmd.Flags().GetBool(flagIncludeArchive)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagIncludeArchive, err)
			}
			dryRun, err := cmd.Flags().GetBool(flagExportDryRun)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagExportDryRun, err)
			}

			rules, err := loadReplacements(replPath)
			if err != nil {
				return err
			}

			res, err := export.Plan(root, cfg, export.Options{
				IncludeArchive: includeArchive,
				Slugs:          args,
				Rules:          rules,
			})
			if err != nil {
				return err
			}
			if len(res.Files) == 0 {
				return fmt.Errorf("반출할 문서가 없습니다")
			}

			if !dryRun {
				// 비어 있지 않은 디렉토리에 쓰면 이전 반출의 잔재가 섞여
				// 이번에 무엇이 나갔는지 알 수 없게 된다(ADR 0046).
				if err := requireEmptyDir(out); err != nil {
					return err
				}
				for _, f := range res.Files {
					if err := writeExportFile(out, f); err != nil {
						return err
					}
				}
			}

			o := exportOutcome{
				DryRun:           dryRun,
				Out:              out,
				Files:            filePaths(res.Files),
				Replaced:         res.Replaced(),
				Anonymized:       len(rules) > 0,
				UnusedRules:      ruleTexts(res.UnusedRules()),
				DanglingLinks:    res.DanglingLinks,
				DanglingSlugs:    res.DanglingSlugs,
				ExcludedByFilter: res.ExcludedByFilter,
				Excluded: exportExcluded{
					Inbox:     res.Exposure.ExcludedInbox,
					Sources:   res.Exposure.ExcludedSources,
					Archive:   res.Exposure.ExcludedArchive,
					Sensitive: res.Exposure.ExcludedSensitive,
					Unparsed:  res.Exposure.ExcludedUnparsed,
					Outside:   res.Exposure.ExcludedOutside,
				},
				SensitivityOn: res.Exposure.SensitivityOn,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(o)
			}
			printExport(cmd.OutOrStdout(), o)
			return nil
		},
	}
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	cmd.Flags().String(flagExportOut, "", "번들을 내보낼 디렉토리")
	cmd.Flags().String(flagExportReplacements, "", "익명화 치환 파일. 한 줄에 원문==>대체어")
	cmd.Flags().Bool(flagIncludeArchive, false, "archive 문서도 반출합니다")
	cmd.Flags().Bool(flagExportDryRun, false, "무엇이 나갈지 봅니다. 파일을 쓰지 않습니다")
	return cmd
}

// loadReplacements는 치환 파일을 읽는다. 경로가 없으면 규칙이 없다.
func loadReplacements(path string) ([]export.Rule, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("치환 파일을 읽을 수 없음: %w", err)
	}
	rules, err := export.ParseReplacements(string(b))
	if err != nil {
		return nil, fmt.Errorf("치환 파일 %s: %w", path, err)
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("치환 파일에 규칙이 없습니다: %s", path)
	}
	return rules, nil
}

// requireEmptyDir는 출력 디렉토리가 없거나 비어 있어야 통과시킨다.
// 덮어쓰는 플래그를 두지 않는다. 지우는 것은 사용자가 자기 눈으로 본다.
func requireEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("출력 디렉토리를 확인할 수 없음: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("출력 디렉토리가 비어 있지 않습니다: %s\n이전 반출물이 섞이지 않도록 비우고 다시 실행하세요", dir)
	}
	return nil
}

// writeExportFile은 번들 파일 하나를 쓴다.
func writeExportFile(out string, f export.File) error {
	path := filepath.Join(out, filepath.FromSlash(f.Rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("디렉토리를 만들 수 없음: %w", err)
	}
	if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
		return fmt.Errorf("번들 파일을 쓸 수 없음: %s: %w", path, err)
	}
	return nil
}

// exportExcluded는 노출 규칙이 무엇을 걸렀는지다.
type exportExcluded struct {
	Inbox     int `json:"inbox"`
	Sources   int `json:"sources"`
	Archive   int `json:"archive"`
	Sensitive int `json:"sensitive"`
	Unparsed  int `json:"unparsed"`
	Outside   int `json:"outside"`
}

// exportOutcome은 export 의 결과다. DryRun 이면 Files 는 내보낼 예정이다.
type exportOutcome struct {
	DryRun           bool           `json:"dryRun"`
	Out              string         `json:"out"`
	Files            []string       `json:"files"`
	Replaced         int            `json:"replaced"`
	Anonymized       bool           `json:"anonymized"`
	UnusedRules      []string       `json:"unusedRules,omitempty"`
	DanglingLinks    int            `json:"danglingLinks"`
	DanglingSlugs    []string       `json:"danglingSlugs,omitempty"`
	ExcludedByFilter int            `json:"excludedByFilter"`
	Excluded         exportExcluded `json:"excluded"`
	SensitivityOn    bool           `json:"sensitivityOn"`
}

// filePaths는 번들 경로만 낸다.
func filePaths(files []export.File) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}

// ruleTexts는 규칙을 사람이 읽을 한 줄로 만든다.
func ruleTexts(rules []export.Rule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		out = append(out, r.From)
	}
	return out
}

// printExport은 무엇이 나갔고 무엇이 빠졌는지, 치환이 몇 건 걸렸는지 낸다.
// 반출은 되돌릴 수 없으므로 매번 전부 알린다.
func printExport(w io.Writer, o exportOutcome) {
	if o.DryRun {
		fmt.Fprintf(w, "내보낼 문서 %d개 (dry-run. 아직 쓰지 않았습니다)\n", len(o.Files))
	} else {
		fmt.Fprintf(w, "내보냈습니다. 문서 %d개 -> %s\n", len(o.Files), o.Out)
	}
	for _, p := range o.Files {
		fmt.Fprintf(w, "  %s\n", p)
	}

	fmt.Fprintf(w, "제외: inbox %d개, sources %d개, archive %d개\n",
		o.Excluded.Inbox, o.Excluded.Sources, o.Excluded.Archive)
	if o.ExcludedByFilter > 0 {
		fmt.Fprintf(w, "제외: 지목한 슬러그에 들지 않은 문서 %d개\n", o.ExcludedByFilter)
	}
	if o.Excluded.Outside > 0 {
		fmt.Fprintf(w, "제외: 단계 디렉토리 밖에 있는 문서 %d개\n", o.Excluded.Outside)
	}
	if o.Excluded.Unparsed > 0 {
		fmt.Fprintf(w, "제외: 프론트매터를 읽을 수 없어 판정하지 못한 문서 %d개 (engram lint 로 확인하세요)\n",
			o.Excluded.Unparsed)
	}
	if o.SensitivityOn {
		fmt.Fprintf(w, "민감도: private-local-only 와 restricted 문서 %d개를 제외했습니다. 뒤집는 플래그는 없습니다\n",
			o.Excluded.Sensitive)
	} else {
		fmt.Fprintf(w, "민감도: 이 위키는 sensitivity 축이 꺼져 있어 거를 값이 없습니다\n")
	}

	if o.Anonymized {
		fmt.Fprintf(w, "익명화: %d건을 치환했습니다\n", o.Replaced)
		if len(o.UnusedRules) > 0 {
			fmt.Fprintf(w, "경고: 한 번도 걸리지 않은 치환 규칙이 %d건 있습니다. 사전의 오타를 확인하세요\n",
				len(o.UnusedRules))
			for _, r := range o.UnusedRules {
				fmt.Fprintf(w, "  %s\n", r)
			}
		}
	} else {
		fmt.Fprintf(w, "익명화: 치환 파일을 주지 않아 원문 그대로 나갑니다\n")
	}

	if o.DanglingLinks > 0 {
		fmt.Fprintf(w, "번들 밖을 가리키는 위키링크 %d개 (문서 %d개). 본문은 고치지 않았습니다\n",
			o.DanglingLinks, len(o.DanglingSlugs))
		for _, s := range o.DanglingSlugs {
			fmt.Fprintf(w, "  %s\n", s)
		}
	}
}
