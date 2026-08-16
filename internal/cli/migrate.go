package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/neocode24/engram/internal/migrate"
	"github.com/spf13/cobra"
)

// flagForce는 migrate가 값이 있는 꺼진 축 필드까지 지우는 플래그 이름이다.
const flagForce = "force"

// maxDetailDocs는 사람용 출력에서 문서별 상세를 보이는 상한이다. 수백 문서의
// 위키에서 전체를 인쇄하면 요약이 묻힌다. 전체는 --json 에 있다.
const maxDetailDocs = 20

// newMigrateCmd는 기존 문서를 지금의 설정과 규칙에 맞추는 migrate 커맨드를
// 반환한다. 루트 등록은 root.go가 한다.
func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "기존 문서를 지금의 설정과 규칙에 맞게 정리합니다",
		Long: `기존 문서를 지금의 engram.yaml과 지금의 규칙에 맞춥니다.

켜진 축의 필수 필드를 단계별 초기값으로 채우고, 꺼진 축의 필드를 지우고,
문서가 놓인 디렉토리에 맞게 artifact_stage를 고칩니다.
파일을 옮기지 않고 슬러그를 바꾸지 않습니다. 문서를 승급시키지도 않습니다.
inbox에 있으면서 context라고 선언한 문서는 선언이 inbox로 내려갈 뿐입니다.
올리려면 engram promote를 쓰세요.

기본은 시험 실행입니다. 실제로 쓰려면 --apply를 주세요.
꺼진 축의 필드에 값이 있으면 --force 없이는 지우지 않습니다. 값을 비운
필드는 --force 없이도 지웁니다.
승급 게이트 위반과 깨진 위키링크는 고치지 않고 보고합니다. 어떤 문서에
이어야 하는지는 판단이므로 engram promote나 demote로 직접 정리하세요.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			apply, err := cmd.Flags().GetBool(flagApply)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagApply, err)
			}
			force, err := cmd.Flags().GetBool(flagForce)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagForce, err)
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
	cmd.Flags().Bool(flagApply, false, "변경을 파일에 씁니다. 기본은 시험 실행입니다")
	cmd.Flags().Bool(flagForce, false, "값이 있는 꺼진 축 필드도 지웁니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// printMigrate는 사람용 보고를 인쇄한다. 요약, 문서별 상세, 고치지 않는
// 항목 순이다. 상세는 상한을 넘으면 줄이고 전체는 --json으로 안내한다.
// 채우지 못해 남은 필드가 있으면 그 사실을 요약과 상세에 모두 낸다.
// 남은 것이 있는데 모두 맞았다고 말하지 않는다.
func printMigrate(w io.Writer, rep migrate.Report) {
	switch {
	case rep.NonConforming == 0:
		fmt.Fprintf(w, "검사한 문서 %d개가 규칙에 맞습니다.\n", rep.Docs)
	case rep.Applied:
		fmt.Fprintf(w, "검사한 문서 %d개 중 %d개가 규칙에 맞지 않아 %d개를 고쳤습니다.",
			rep.Docs, rep.NonConforming, rep.Written)
		if rep.Partial > 0 {
			fmt.Fprintf(w, " %d개는 migrate로도 완전히 맞추지 못했고 남은 필드를 아래에 적습니다.", rep.Partial)
		}
		fmt.Fprintln(w)
	default:
		fmt.Fprintf(w, "검사한 문서 %d개 중 %d개가 규칙에 맞지 않습니다.", rep.Docs, rep.NonConforming)
		if rep.Partial > 0 {
			fmt.Fprintf(w, " 이 중 %d개는 migrate로도 완전히 맞추지 못합니다. 남은 필드를 아래에 적습니다.", rep.Partial)
		}
		fmt.Fprintf(w, " 시험 실행이므로 파일을 쓰지 않았습니다. 적용하려면 --apply를 주세요.\n")
	}
	if len(rep.Unparsed) > 0 {
		fmt.Fprintf(w, "프론트매터를 읽을 수 없어 건너뛴 문서 %d개가 있습니다. 먼저 프론트매터를 고치세요.\n", len(rep.Unparsed))
	}

	detail := 0
	for _, d := range rep.Documents {
		if detail >= maxDetailDocs {
			fmt.Fprintf(w, "\n... 외 %d개 문서의 상세는 --json으로 볼 수 있습니다.\n", len(rep.Documents)-detail)
			break
		}
		detail++
		fmt.Fprintf(w, "\n%s\n", d.Path)
		for _, c := range d.Changes {
			switch c.Kind {
			case migrate.KindStage:
				fmt.Fprintf(w, "  artifact_stage: %s -> %s (문서가 놓인 디렉토리에 맞춥니다)\n", quoteValue(c.Old), quoteValue(c.New))
			case migrate.KindFill:
				fmt.Fprintf(w, "  %s: (없음) -> %s\n", c.Field, quoteValue(c.New))
			case migrate.KindRemove:
				fmt.Fprintf(w, "  %s: %s -> (삭제)\n", c.Field, quoteValue(c.Old))
			}
		}
		for _, c := range d.Blocked {
			fmt.Fprintf(w, "  %s: %s (지우면 이 값을 잃습니다. 지우려면 --force를 주세요)\n", c.Field, quoteValue(c.Old))
		}
		for _, r := range d.Remainders {
			fmt.Fprintf(w, "  %s: %s\n", r.Field, r.Reason)
		}
	}

	gate, broken := splitAdvisories(rep.Advisories)
	if len(gate) > 0 {
		fmt.Fprintf(w, "\n승급 게이트 위반 문서 %d개는 migrate가 고치지 않습니다. 어떤 문서에 이어야 하는지는 판단입니다. engram promote나 demote로 정리하세요.\n  %s\n",
			len(gate), strings.Join(gate, ", "))
	}
	if len(broken) > 0 {
		fmt.Fprintf(w, "\n깨진 위키링크가 있는 문서 %d개는 migrate가 고치지 않습니다. 슬러그를 고치거나 대상 문서를 만드세요.\n  %s\n",
			len(broken), strings.Join(broken, ", "))
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
		return "(빈 값)"
	}
	return "\"" + v + "\""
}
