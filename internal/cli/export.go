package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/export"
	"github.com/neocode24/engram/internal/i18n"
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
		Short: i18n.T("cli.export.short"),
		Long:  i18n.T("cli.export.long"),
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
				return fmt.Errorf("%s", i18n.T("cli.export.out_required", flagExportOut))
			}
			replPath, err := stringFlag(cmd, flagExportReplacements)
			if err != nil {
				return err
			}
			includeArchive, err := cmd.Flags().GetBool(flagIncludeArchive)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.export.flag_read_fail", flagIncludeArchive), err)
			}
			dryRun, err := cmd.Flags().GetBool(flagExportDryRun)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.export.flag_read_fail", flagExportDryRun), err)
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
				return fmt.Errorf("%s", i18n.T("cli.export.no_files"))
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
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.export.flag_wiki"))
	cmd.Flags().String(flagExportOut, "", i18n.T("cli.export.flag_out"))
	cmd.Flags().String(flagExportReplacements, "", i18n.T("cli.export.flag_replacements"))
	cmd.Flags().Bool(flagIncludeArchive, false, i18n.T("cli.export.flag_include_archive"))
	cmd.Flags().Bool(flagExportDryRun, false, i18n.T("cli.export.flag_dry_run"))
	return cmd
}

// loadReplacements는 치환 파일을 읽는다. 경로가 없으면 규칙이 없다.
func loadReplacements(path string) ([]export.Rule, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.export.repl_read_fail"), err)
	}
	rules, err := export.ParseReplacements(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.export.repl_parse_fail", path), err)
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("cli.export.repl_empty", path))
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
		return fmt.Errorf("%s: %w", i18n.T("cli.export.outdir_check_fail"), err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s\n%s",
			i18n.T("cli.export.outdir_not_empty", dir), i18n.T("cli.export.outdir_not_empty_hint"))
	}
	return nil
}

// writeExportFile은 번들 파일 하나를 쓴다.
func writeExportFile(out string, f export.File) error {
	path := filepath.Join(out, filepath.FromSlash(f.Rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.export.mkdir_fail"), err)
	}
	if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.export.write_fail", path), err)
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
		fmt.Fprint(w, i18n.T("cli.export.outcome_dryrun", len(o.Files))+"\n")
	} else {
		fmt.Fprint(w, i18n.T("cli.export.outcome_done", len(o.Files), o.Out)+"\n")
	}
	for _, p := range o.Files {
		fmt.Fprintf(w, "  %s\n", p)
	}

	fmt.Fprint(w, i18n.T("cli.export.excluded_summary",
		o.Excluded.Inbox, o.Excluded.Sources, o.Excluded.Archive)+"\n")
	if o.ExcludedByFilter > 0 {
		fmt.Fprint(w, i18n.T("cli.export.excluded_filter", o.ExcludedByFilter)+"\n")
	}
	if o.Excluded.Outside > 0 {
		fmt.Fprint(w, i18n.T("cli.export.excluded_outside", o.Excluded.Outside)+"\n")
	}
	if o.Excluded.Unparsed > 0 {
		fmt.Fprint(w, i18n.T("cli.export.excluded_unparsed", o.Excluded.Unparsed)+"\n")
	}
	if o.SensitivityOn {
		fmt.Fprint(w, i18n.T("cli.export.sensitivity_on", o.Excluded.Sensitive)+"\n")
	} else {
		fmt.Fprint(w, i18n.T("cli.export.sensitivity_off")+"\n")
	}

	if o.Anonymized {
		fmt.Fprint(w, i18n.T("cli.export.anonymized_count", o.Replaced)+"\n")
		if len(o.UnusedRules) > 0 {
			fmt.Fprint(w, i18n.T("cli.export.unused_rules_warning", len(o.UnusedRules))+"\n")
			for _, r := range o.UnusedRules {
				fmt.Fprintf(w, "  %s\n", r)
			}
		}
	} else {
		fmt.Fprint(w, i18n.T("cli.export.anonymized_none")+"\n")
	}

	if o.DanglingLinks > 0 {
		fmt.Fprint(w, i18n.T("cli.export.dangling_links", o.DanglingLinks, len(o.DanglingSlugs))+"\n")
		for _, s := range o.DanglingSlugs {
			fmt.Fprintf(w, "  %s\n", s)
		}
	}
}
