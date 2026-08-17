// Package cli는 engram CLI의 커맨드 계층을 담는다.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// 플래그 이름 상수. 조회 커맨드가 직접 플래그를 읽으므로 오타 방지를 위해 한 곳에서 정의한다.
const (
	flagJSON = "json"
	flagNow  = "now"
	flagLang = "lang"
)

// nowKey는 커맨드 context에 기준 시각을 담기 위한 키 타이다.
type nowKey struct{}

// parseNow는 --now 값을 파싱해 기준 시각을 반환한다. 빈 값이면 실제 시계를 쓴다.
func parseNow(raw string) (time.Time, error) {
	if raw == "" {
		return time.Now(), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("--now 값이 RFC3339 형식이 아님: %q", raw)
	}
	return t, nil
}

// Now는 커맨드의 기준 시각을 반환한다. 루트 커맨드를 거치지 않은 경우 실제 시계로 대체한다.
func Now(cmd *cobra.Command) time.Time {
	if t, ok := cmd.Context().Value(nowKey{}).(time.Time); ok {
		return t
	}
	return time.Now()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "engram",
		Short: "지식관리 위키의 승급 파이프라인을 다루는 CLI",
		Long: `engram은 지식관리 위키의 문서 상태와 승급 파이프라인을 다루는 CLI입니다.

모든 조회 커맨드는 --json으로 JSON 출력을 지원하고, --now로 기준 시각을
고정해 결정론적인 결과를 얻을 수 있습니다.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 언어를 가장 먼저 정한다. 이 뒤의 에러 메시지부터 그 언어로 나온다.
			langRaw, err := cmd.Flags().GetString(flagLang)
			if err != nil {
				return err
			}
			lang, err := i18n.Resolve(langRaw)
			if err != nil {
				return err
			}
			i18n.SetLang(lang)

			raw, err := cmd.Flags().GetString(flagNow)
			if err != nil {
				return err
			}
			t, err := parseNow(raw)
			if err != nil {
				return err
			}
			cmd.SetContext(context.WithValue(cmd.Context(), nowKey{}, t))
			return nil
		},
	}
	root.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	root.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339). 빈 값이면 현재 시각")
	root.PersistentFlags().String(flagLang, "", "출력 언어(ko, en). 빈 값이면 "+i18n.EnvVar+" 환경변수, 그다음 ko")

	root.AddCommand(newArchiveCmd())
	root.AddCommand(newBridgeCmd())
	root.AddCommand(newDigestCmd())
	root.AddCommand(newResurfaceCmd())
	root.AddCommand(newRecallCmd())
	root.AddCommand(newNewCmd())
	root.AddCommand(newPromoteCmd())
	root.AddCommand(newReindexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newBacklinksCmd())
	root.AddCommand(newDemoteCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newMvCmd())
	root.AddCommand(newCaptureCmd())
	root.AddCommand(newSourceCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newLintCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newRulesCmd())
	root.AddCommand(newEjectCmd())
	root.AddCommand(newSkillsCmd())
	root.AddCommand(newMCPCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newVersionCmd())
	return root
}

// jsonOutput는 커맨드가 --json 플래그를 받았는지를 반환한다.
// --json은 루트의 persistent flag이므로 하위 커맨드에서도 이렇게 읽는다.
func jsonOutput(cmd *cobra.Command) bool {
	v, err := cmd.Flags().GetBool(flagJSON)
	return err == nil && v
}

// Execute는 루트 커맨드를 실행하고 프로세스 종료 코드를 반환한다.
func Execute() int {
	// Windows 콘솔이면 출력 코드페이지를 UTF-8 로 바꾸고 끝나면 되돌린다.
	// 다른 플랫폼은 no-op 이다. ADR 0026.
	defer setupConsole()()

	// 커맨드 트리를 만들기 전에 언어를 정한다. Short 와 Long 과 플래그
	// 설명은 트리를 만드는 시점에 굳으므로, PersistentPreRunE 까지
	// 기다리면 help 출력만 언어가 안 바뀐다. 여기서는 값이 틀려도
	// 조용히 넘긴다. 에러 보고는 PersistentPreRunE 가 맡는다.
	if lang, err := i18n.Resolve(langFromArgs(os.Args[1:])); err == nil {
		i18n.SetLang(lang)
	}

	if err := newRootCmd().Execute(); err != nil {
		return 1
	}
	return 0
}

// langFromArgs는 인자에서 --lang 값을 꺼낸다. cobra 가 파싱하기 전이라
// 직접 훑는다. 값이 없거나 형식이 틀리면 빈 문자열을 낸다.
func langFromArgs(args []string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--lang="); ok {
			return v
		}
		if a == "--lang" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
