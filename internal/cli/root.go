// Package cli는 engram CLI의 커맨드 계층을 담는다.
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// 플래그 이름 상수. 조회 커맨드가 직접 플래그를 읽으므로 오타 방지를 위해 한 곳에서 정의한다.
const (
	flagJSON = "json"
	flagNow  = "now"
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
		Long: `engram은 지식관리 위키의 문서 상태와 승급 파이프라인을 다루는 CLI다.

모든 조회 커맨드는 --json으로 JSON 출력을 지원하고, --now로 기준 시각을
고정해 결정론적인 결과를 얻을 수 있다.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
	root.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력한다")
	root.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339). 빈 값이면 현재 시각")

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
	if err := newRootCmd().Execute(); err != nil {
		return 1
	}
	return 0
}
