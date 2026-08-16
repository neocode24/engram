package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/skills"
	"github.com/spf13/cobra"
)

// skills 커맨드의 플래그 이름. 다른 커맨드의 force, dry-run 과 값이
// 같지만 소속을 밝히는 이름을 쓴다.
const (
	flagSkillsDir    = "dir"
	flagSkillsForce  = "force"
	flagSkillsDryRun = "dry-run"
)

// newSkillsCmd는 에이전트 스킬을 다루는 skills 커맨드를 반환한다.
// 하위 커맨드 없이 치면 사용법을 낸다.
func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "에이전트 스킬을 다룹니다",
		Long: `에이전트가 engram을 다루는 법을 가르치는 스킬 문서를 다룹니다.

skills install이 바이너리에 임베드된 스킬 문서를 감지된 에이전트의
스킬 디렉토리에 심습니다. 이것이 LLM 통합의 전부입니다(ADR 0014).

이 커맨드는 위키가 아니라 에이전트를 다룹니다. 위키 밖에서 실행해도
동작합니다.`,
	}
	cmd.AddCommand(newSkillsInstallCmd())
	return cmd
}

// newSkillsInstallCmd는 임베드된 스킬 문서를 에이전트 스킬 디렉토리에
// 심는 install 커맨드를 반환한다.
func newSkillsInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "스킬 문서를 에이전트 스킬 디렉토리에 심습니다",
		Long: `임베드된 스킬 문서를 에이전트의 스킬 디렉토리에 심습니다.

문서는 정적입니다. 위키의 임계값이나 허용값을 담지 않습니다. 그 위키에
적용되는 규칙은 에이전트가 engram rules show로 얻습니다. 그래서 한 번
심으면 모든 위키에 통합니다.

--dir이 없으면 홈 디렉토리에서 실제로 존재하는 에이전트 스킬 디렉토리를
찾아 전부에 심습니다. 없는 도구를 위해 디렉토리를 만들지 않습니다.
하나도 찾지 못하면 실패하니 --dir로 직접 지정하세요. --dir은 스킬
루트(스킬 디렉토리들이 있는 곳)를 받고 그 아래 engram/SKILL.md를
만듭니다. 이미 있는 디렉토리여야 합니다.

이미 있는 파일은 덮지 않습니다. 충돌하면 무엇이 충돌하는지 알리고
멈춥니다. 덮으려면 --force를 주세요.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dirFlag, err := stringFlag(cmd, flagSkillsDir)
			if err != nil {
				return err
			}
			force, err := cmd.Flags().GetBool(flagSkillsForce)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagSkillsForce, err)
			}
			dryRun, err := cmd.Flags().GetBool(flagSkillsDryRun)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagSkillsDryRun, err)
			}

			var roots []string
			if dirFlag != "" {
				info, err := os.Stat(dirFlag)
				if err != nil || !info.IsDir() {
					return fmt.Errorf("--dir 경로가 디렉토리가 아닙니다: %s\n실제로 있는 스킬 루트 디렉토리를 주세요", dirFlag)
				}
				roots = []string{dirFlag}
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("홈 디렉토리를 알 수 없음: %w\n--dir로 설치 위치를 직접 지정하세요", err)
				}
				roots = skills.Detect(home)
				if len(roots) == 0 {
					return fmt.Errorf("설치 대상이 될 에이전트 스킬 디렉토리를 찾지 못했습니다\n찾아본 곳(홈 디렉토리 기준):\n  %s\n없는 도구를 위해 디렉토리를 만들지 않습니다. 설치 위치를 --dir로 직접 지정하세요",
						strings.Join(skills.Sources(), "\n  "))
				}
			}

			paths := make([]string, 0, len(roots))
			for _, root := range roots {
				paths = append(paths, skills.InstallPath(root))
			}

			// 충돌을 전부 찾은 뒤에야 쓰기 시작한다. 일부만 심고 멈추면
			// 어느 에이전트에만 스킬이 들어간 어중간한 상태가 된다.
			var conflicts []string
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					conflicts = append(conflicts, p)
				}
			}
			if len(conflicts) > 0 && !force {
				res := skillsOutcome{Conflicts: conflicts}
				if jsonOutput(cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				printSkillsConflicts(cmd.ErrOrStderr(), res)
				return fmt.Errorf("이미 있는 파일을 덮지 않습니다. 덮으려면 --force를 주세요")
			}

			var written, overwritten []string
			if !dryRun {
				for _, p := range paths {
					if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
						return fmt.Errorf("스킬 디렉토리를 만들 수 없음: %w", err)
					}
					if err := os.WriteFile(p, []byte(skills.Doc()), 0o644); err != nil {
						return fmt.Errorf("스킬 문서를 쓸 수 없음: %s: %w", p, err)
					}
					written = append(written, p)
					if containsString(conflicts, p) {
						overwritten = append(overwritten, p)
					}
				}
			}

			res := skillsOutcome{
				DryRun:      dryRun,
				Force:       force,
				Files:       paths,
				Written:     written,
				Overwritten: overwritten,
				Conflicts:   conflicts,
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printSkillsInstall(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagSkillsDir, "", "설치 위치를 직접 지정합니다. 감지를 건너뜁니다")
	cmd.Flags().Bool(flagSkillsForce, false, "이미 있는 파일을 덮어 씁니다")
	cmd.Flags().Bool(flagSkillsDryRun, false, "어디에 무엇을 심을지만 냅니다. 쓰지 않습니다")
	return cmd
}

// skillsOutcome은 skills install의 결과다. DryRun 이면 Files 는 심을
// 예정, 아니면 심은 목록이다.
type skillsOutcome struct {
	DryRun      bool     `json:"dryRun"`
	Force       bool     `json:"force"`
	Files       []string `json:"files"`
	Written     []string `json:"written,omitempty"`
	Overwritten []string `json:"overwritten,omitempty"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

// printSkillsInstall는 심은 목록과 재시작 안내를 낸다.
func printSkillsInstall(w io.Writer, res skillsOutcome) {
	if res.DryRun {
		fmt.Fprintf(w, "심을 예정인 파일 %d개 (dry-run. 아직 쓰지 않았습니다)\n", len(res.Files))
	} else {
		fmt.Fprintf(w, "심었습니다. 파일 %d개\n", len(res.Files))
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
	fmt.Fprintf(w, "에이전트를 다시 시작해야 스킬이 잡힐 수 있습니다\n")
}

// printSkillsConflicts는 충돌 목록을 알린다.
func printSkillsConflicts(w io.Writer, res skillsOutcome) {
	fmt.Fprintf(w, "이미 있는 파일이 %d개 있습니다\n", len(res.Conflicts))
	for _, p := range res.Conflicts {
		fmt.Fprintf(w, "  %s\n", p)
	}
}
