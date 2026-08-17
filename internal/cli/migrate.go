package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/migrate"
	"github.com/spf13/cobra"
)

// flagForce는 migrate가 값이 있는 꺼진 속성 필드까지 지우는 플래그 이름이다.
const flagForce = "force"

// maxDetailDocs는 사람용 출력에서 문서별 상세를 보이는 상한이다. 수백 문서의
// 위키에서 전체를 인쇄하면 요약이 묻힌다. 전체는 --json 에 있다.
const maxDetailDocs = 20

// newMigrateCmd는 기존 문서를 지금의 설정과 규칙에 맞추는 migrate 커맨드를
// 반환한다. 루트 등록은 root.go가 한다.
func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: i18n.T("cli.migrate.short"),
		Long:  i18n.T("cli.migrate.long"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			apply, err := cmd.Flags().GetBool(flagApply)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.migrate.flag_read_fail", flagApply), err)
			}
			force, err := cmd.Flags().GetBool(flagForce)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.migrate.flag_read_fail", flagForce), err)
			}
			rep, err := migrate.Run(root, cfg, migrate.Options{Apply: apply, Force: force})
			if err != nil {
				return err
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rep)
			}
			printMigrate(cmd.OutOrStdout(), rep)
			return nil
		},
	}
	cmd.Flags().Bool(flagApply, false, i18n.T("cli.migrate.flag_apply"))
	cmd.Flags().Bool(flagForce, false, i18n.T("cli.migrate.flag_force"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.migrate.flag_wiki"))
	return cmd
}

// printMigrate는 사람용 보고를 인쇄한다. 요약, 문서별 상세, 고치지 않는
// 항목 순이다. 상세는 상한을 넘으면 줄이고 전체는 --json으로 안내한다.
// 채우지 못해 남은 필드가 있으면 그 사실을 요약과 상세에 모두 낸다.
// 남은 것이 있는데 모두 맞았다고 말하지 않는다.
func printMigrate(w io.Writer, rep migrate.Report) {
	switch {
	case rep.NonConforming == 0:
		fmt.Fprint(w, i18n.T("cli.migrate.all_ok", rep.Docs)+"\n")
	case rep.Applied:
		fmt.Fprint(w, i18n.T("cli.migrate.applied_summary", rep.Docs, rep.NonConforming, rep.Written))
		if rep.Partial > 0 {
			fmt.Fprint(w, " "+i18n.T("cli.migrate.applied_partial", rep.Partial))
		}
		fmt.Fprintln(w)
	default:
		fmt.Fprint(w, i18n.T("cli.migrate.dry_summary", rep.Docs, rep.NonConforming))
		if rep.Partial > 0 {
			fmt.Fprint(w, " "+i18n.T("cli.migrate.dry_partial", rep.Partial))
		}
		fmt.Fprint(w, " "+i18n.T("cli.migrate.dry_notice")+"\n")
	}
	if len(rep.Unparsed) > 0 {
		fmt.Fprint(w, i18n.T("cli.migrate.unparsed", len(rep.Unparsed))+"\n")
	}

	detail := 0
	for _, d := range rep.Documents {
		if detail >= maxDetailDocs {
			fmt.Fprint(w, "\n"+i18n.T("cli.migrate.more_detail", len(rep.Documents)-detail)+"\n")
			break
		}
		detail++
		fmt.Fprintf(w, "\n%s\n", d.Path)
		for _, c := range d.Changes {
			switch c.Kind {
			case migrate.KindStage:
				fmt.Fprint(w, "  "+i18n.T("cli.migrate.change_stage", quoteValue(c.Old), quoteValue(c.New))+"\n")
			case migrate.KindFill:
				fmt.Fprint(w, "  "+i18n.T("cli.migrate.change_fill", c.Field, quoteValue(c.New))+"\n")
			case migrate.KindRemove:
				fmt.Fprint(w, "  "+i18n.T("cli.migrate.change_remove", c.Field, quoteValue(c.Old))+"\n")
			}
		}
		for _, c := range d.Blocked {
			fmt.Fprint(w, "  "+i18n.T("cli.migrate.blocked", c.Field, quoteValue(c.Old))+"\n")
		}
		for _, r := range d.Remainders {
			fmt.Fprintf(w, "  %s: %s\n", r.Field, r.Reason)
		}
	}

	gate, broken := splitAdvisories(rep.Advisories)
	if len(gate) > 0 {
		fmt.Fprintf(w, "\n%s\n  %s\n",
			i18n.T("cli.migrate.gate_advice", len(gate)), strings.Join(gate, ", "))
	}
	if len(broken) > 0 {
		fmt.Fprintf(w, "\n%s\n  %s\n",
			i18n.T("cli.migrate.broken_advice", len(broken)), strings.Join(broken, ", "))
	}
}

// splitAdvisories는 보고 항목을 게이트 위반 경로와 깨진 링크 경로로 나눈다.
func splitAdvisories(as []migrate.Advisory) (gate, broken []string) {
	seenG := map[string]bool{}
	seenB := map[string]bool{}
	for _, a := range as {
		switch a.Rule {
		case "gate.min-wikilinks":
			if !seenG[a.Path] {
				seenG[a.Path] = true
				gate = append(gate, a.Path)
			}
		case "link.broken":
			if !seenB[a.Path] {
				seenB[a.Path] = true
				broken = append(broken, a.Path)
			}
		}
	}
	return gate, broken
}

// quoteValue는 표시값을 읽기 좋게 감싼다. 빈 값은 빈 표시를 쓴다.
func quoteValue(v string) string {
	if v == "" {
		return i18n.T("cli.migrate.empty_value")
	}
	return "\"" + v + "\""
}
