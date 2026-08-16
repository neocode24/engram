package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// source 커맨드의 플래그 이름이다.
const (
	flagCreated = "created"
	flagChannel = "channel"
	flagRef     = "ref"
	flagType    = "type"
)

// defaultSourceType는 source 커맨드의 type 축 기본값이다.
const defaultSourceType = "source-summary"

// newSourceCmd는 원본 자료를 sources에 넣는 source 커맨드를 반환한다.
// 이 계층은 원본 보존이 계약이므로 updated 필드를 쓰지 않는다. ADR 0009.
func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source [내용]",
		Short: "원본 자료를 sources에 넣고 출처를 확정합니다",
		Long: `원본 자료를 sources에 넣고 원본 필드를 확정합니다.

내용을 인자로 받거나 파이프로 연결된 표준 입력으로 받습니다.
--created로 원본이 작성된 날을 줍니다. 하루(YYYY-MM-DD) 또는 연월(YYYY-MM)
정밀도를 허용하고 생략하면 전역 --now 기준 날짜를 씁니다.
이 계층은 원본 보존이 계약이므로 문서를 고치지 않고 updated 필드도 쓰지 않습니다.`,
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
			titleFlag, err := stringFlag(cmd, flagTitle)
			if err != nil {
				return err
			}
			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			title := deriveTitle(titleFlag, content)
			slug, err := resolveSlug(slugFlag, title)
			if err != nil {
				return err
			}

			created, err := stringFlag(cmd, flagCreated)
			if err != nil {
				return err
			}
			if created == "" {
				created = Now(cmd).Format("2006-01-02")
			} else if !validDatePrecision(created) {
				return fmt.Errorf("--created 값이 YYYY-MM-DD 또는 YYYY-MM 형식이 아닙니다: %q", created)
			}
			docType, err := sourceType(cmd, cfg)
			if err != nil {
				return err
			}

			fm := wiki.Frontmatter(wiki.StageSource, cfg)
			fm["type"] = docType
			fm["created"] = created
			fm["sourced_at"] = Now(cmd).Format("2006-01-02")
			if err := applyChannel(cmd, cfg, fm); err != nil {
				return err
			}
			if err := applyRefs(cmd, cfg, fm); err != nil {
				return err
			}

			path, err := wiki.Create(root, cfg, wiki.StageSource, created, slug, fm, content)
			if err != nil {
				return err
			}
			res := ingestResult{Path: path, Slug: slug, Stage: string(wiki.StageSource)}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printIngested(cmd.OutOrStdout(), res,
				"정리가 끝나면 이 원본을 인용하는 맥락 문서를 만드세요")
			return nil
		},
	}
	cmd.Flags().String(flagTitle, "", "문서 제목. 생략하면 본문 첫 줄에서 만듭니다")
	cmd.Flags().String(flagSlug, "", "파일명 슬러그. 생략하면 제목에서 만듭니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	cmd.Flags().String(flagCreated, "", "원본이 작성된 날(YYYY-MM-DD 또는 YYYY-MM)")
	cmd.Flags().String(flagChannel, "", "입력 경로. source_channel 축 값")
	cmd.Flags().StringArray(flagRef, nil, "원본 출처(경로나 URL). 여러 번 쓸 수 있습니다")
	cmd.Flags().String(flagType, defaultSourceType, "문서 종류. 허용값은 위키 설정의 types입니다")
	return cmd
}

// sourceType는 --type 값을 검증해 반환한다. 허용값 밖이면 목록과 함께 거절한다.
func sourceType(cmd *cobra.Command, cfg config.Config) (string, error) {
	v, err := stringFlag(cmd, flagType)
	if err != nil {
		return "", err
	}
	if v == "" {
		v = defaultSourceType
	}
	for _, t := range cfg.Schema.Types {
		if t == v {
			return v, nil
		}
	}
	return "", fmt.Errorf("--type 값이 허용값 밖입니다: %q (허용값: %s)",
		v, strings.Join(cfg.Schema.Types, ", "))
}

// validDatePrecision은 날짜가 하루 또는 연월 정밀도인지 검사한다.
func validDatePrecision(v string) bool {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if _, err := time.Parse(layout, v); err == nil {
			return true
		}
	}
	return false
}

// applyChannel은 --channel 값을 프론트매터에 넣는다. 축이 꺼져 있으면
// 값을 무시하고 경고를 낸다.
func applyChannel(cmd *cobra.Command, cfg config.Config, fm map[string]any) error {
	v, err := stringFlag(cmd, flagChannel)
	if err != nil {
		return err
	}
	if v == "" {
		return nil
	}
	if cfg.Axes[config.AxisSourceChannel] {
		fm["source_channel"] = v
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"경고: source_channel 축이 꺼져 있어 --channel 값을 무시합니다. 켜려면 engram.yaml의 axes에서 source_channel을 true로 두세요\n")
	return nil
}

// applyRefs는 --ref 값을 프론트매터의 source_refs에 넣는다.
// 축이 꺼져 있으면 값을 무시하고 경고를 낸다.
func applyRefs(cmd *cobra.Command, cfg config.Config, fm map[string]any) error {
	refs, err := cmd.Flags().GetStringArray(flagRef)
	if err != nil {
		return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagRef, err)
	}
	if len(refs) == 0 {
		return nil
	}
	if cfg.Axes[config.AxisSourceRefs] {
		fm["source_refs"] = refs
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"경고: source_refs 축이 꺼져 있어 --ref 값을 무시합니다. 켜려면 engram.yaml의 axes에서 source_refs를 true로 두세요\n")
	return nil
}
