package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neocode24/engram/internal/eject"
	"github.com/spf13/cobra"
)

// eject 커맨드의 플래그 이름.
const (
	flagEjectForce  = "force"
	flagEjectDryRun = "dry-run"
)

// newEjectCmd는 내장 규칙을 사용자가 고칠 수 있는 파일로 내보내는 eject
// 커맨드를 반환한다. 규칙만 사용자 것이 되고 연산은 engram 에 남는다.
func newEjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eject",
		Short: "규칙 명세와 Python 린터를 위키로 내보냅니다",
		Long: `내장 규칙을 사용자가 고칠 수 있는 파일로 내보냅니다.

규칙 명세 문서(meta/), 문서 단위 규칙을 판정하는 Python 린터
(scripts/lint-frontmatter.py), 커밋 훅(.githooks/pre-commit),
에이전트 계약(AGENTS.md), 줄바꿈 설정(.gitattributes)을 만듭니다.
전부 이 위키의 engram.yaml 을 반영해 생성합니다. 린터는 값을 박지
않고 engram.yaml 을 실행 시점에 읽으므로 설정을 바꾸면 따라갑니다.

제품에서 나가는 문이 아닙니다. 규칙만 사용자 것이 되고 연산은
engram 에 남습니다. 이후에도 search, recall, resurface, bridge,
digest, backlinks 가 그대로 동작합니다.

단방향입니다. 되돌리는 커맨드가 없으므로 이미 있는 파일을 덮지
않습니다. 충돌하면 무엇이 충돌하는지 전부 알리고 멈춥니다.
--force 를 주면 덮되 무엇을 덮는지 먼저 알립니다.
--dry-run 은 무엇이 만들어질지 봅니다. 쓰지는 않습니다.

린터와 훅에는 python3 가 필요합니다. Windows 는 기본 제공되지
않으므로 따로 설치해야 합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool(flagEjectForce)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagEjectForce, err)
			}
			dryRun, err := cmd.Flags().GetBool(flagEjectDryRun)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagEjectDryRun, err)
			}

			plan := eject.Plan(cfg)

			// 충돌을 전부 찾은 뒤에야 쓰기 시작한다. 일부만 쓰고 멈추면
			// 위키가 어중간해진다.
			var conflicts []string
			for _, a := range plan {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(a.Path))); err == nil {
					conflicts = append(conflicts, a.Path)
				}
			}
			if len(conflicts) > 0 && !force {
				res := ejectOutcome{Conflicts: conflicts}
				if jsonOutput(cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				printEjectConflicts(cmd.ErrOrStderr(), res)
				return fmt.Errorf("이미 있는 파일을 덮지 않습니다. 덮으려면 --force 를 주세요")
			}

			var written, overwritten []string
			if !dryRun {
				for _, a := range plan {
					if err := writeArtifact(root, a); err != nil {
						return err
					}
					written = append(written, a.Path)
					if containsString(conflicts, a.Path) {
						overwritten = append(overwritten, a.Path)
					}
				}
			}

			res := ejectOutcome{
				DryRun:      dryRun,
				Force:       force,
				Files:       planPaths(plan),
				Written:     written,
				Overwritten: overwritten,
				Conflicts:   conflicts,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printEject(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Bool(flagEjectForce, false, "이미 있는 파일을 덮어 씁니다")
	cmd.Flags().Bool(flagEjectDryRun, false, "무엇이 만들어질지 봅니다. 파일을 쓰지 않습니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// ejectOutcome은 eject 의 결과다. DryRun 이면 Files 는 만들 예정,
// 아니면 만든 목록이다.
type ejectOutcome struct {
	DryRun      bool     `json:"dryRun"`
	Force       bool     `json:"force"`
	Files       []string `json:"files"`
	Written     []string `json:"written,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

// planPaths는 산출물 경로만 낸다.
func planPaths(plan []eject.Artifact) []string {
	out := make([]string, 0, len(plan))
	for _, a := range plan {
		out = append(out, a.Path)
	}
	return out
}

// writeArtifact는 산출물 하나를 위키에 쓴다. 훅은 실행 권한이 필요하다.
func writeArtifact(root string, a eject.Artifact) error {
	path := filepath.Join(root, filepath.FromSlash(a.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("디렉토리를 만들 수 없음: %w", err)
	}
	if err := os.WriteFile(path, []byte(a.Content), a.Mode); err != nil {
		return fmt.Errorf("산출물을 쓸 수 없음: %s: %w", path, err)
	}
	return nil
}

// printEject는 만든 목록과 세 가지 안내를 낸다.
func printEject(w io.Writer, res ejectOutcome) {
	if res.DryRun {
		fmt.Fprintf(w, "만들 예정인 파일 %d개 (dry-run. 아직 쓰지 않았습니다)\n", len(res.Files))
	} else {
		fmt.Fprintf(w, "만들었습니다. 파일 %d개\n", len(res.Files))
	}
	for _, p := range res.Files {
		fmt.Fprintf(w, "  %s\n", p)
	}
	if len(res.Overwritten) > 0 {
		fmt.Fprintf(w, "덮어 쓴 파일 %d개\n", len(res.Overwritten))
		for _, p := range res.Overwritten {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	fmt.Fprintf(w, "\n안내:\n")
	fmt.Fprintf(w, "  훅을 켜려면: git config core.hooksPath .githooks\n")
	fmt.Fprintf(w, "  eject 이후에도 search, recall, resurface, bridge, digest, backlinks 가 그대로 동작합니다\n")
	fmt.Fprintf(w, "  린터와 훅에는 python3 이 필요합니다. Windows 는 기본 제공되지 않으므로 설치해야 합니다\n")
}

// printEjectConflicts는 충돌 목록을 알린다.
func printEjectConflicts(w io.Writer, res ejectOutcome) {
	fmt.Fprintf(w, "이미 있는 파일이 %d개 있습니다\n", len(res.Conflicts))
	for _, p := range res.Conflicts {
		fmt.Fprintf(w, "  %s\n", p)
	}
}
