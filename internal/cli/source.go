package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
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

// defaultSourceType는 source 커맨드의 type 속성 기본값이다.
const defaultSourceType = "source-summary"

// newSourceCmd는 원본 자료를 sources에 넣는 source 커맨드를 반환한다.
// 이 계층은 원본 보존이 계약이므로 updated 필드를 쓰지 않는다. ADR 0009.
func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source [내용]",
		Short: i18n.T("cli.source.short"),
		Long:  i18n.T("cli.source.long"),
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
			titleFlag, err := stringFlag(cmd, flagTitle)
			if err != nil {
				return err
			}
			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			title := deriveTitle(titleFlag, content)
			content = withHeading(title, content, titleFlag != "")
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
				return errors.New(i18n.T("cli.source.created_invalid", created))
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
				i18n.T("cli.source.next_hint"))
			return nil
		},
	}
	cmd.Flags().String(flagTitle, "", i18n.T("cli.ingest.flag_title"))
	cmd.Flags().String(flagSlug, "", i18n.T("cli.ingest.flag_slug"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	cmd.Flags().String(flagCreated, "", i18n.T("cli.source.flag_created"))
	cmd.Flags().String(flagChannel, "", i18n.T("cli.source.flag_channel"))
	cmd.Flags().StringArray(flagRef, nil, i18n.T("cli.source.flag_ref"))
	cmd.Flags().String(flagType, defaultSourceType, i18n.T("cli.source.flag_type"))
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
	return "", errors.New(i18n.T("cli.ingest.type_invalid",
		v, strings.Join(cfg.Schema.Types, ", ")))
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

// applyChannel은 --channel 값을 프론트매터에 넣는다. 속성이 꺼져 있으면
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
	fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.source.warn_channel_off"))
	return nil
}

// applyRefs는 --ref 값을 프론트매터의 source_refs에 넣는다.
// 속성이 꺼져 있으면 값을 무시하고 경고를 낸다.
func applyRefs(cmd *cobra.Command, cfg config.Config, fm map[string]any) error {
	refs, err := cmd.Flags().GetStringArray(flagRef)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagRef), err)
	}
	if len(refs) == 0 {
		return nil
	}
	if cfg.Axes[config.AxisSourceRefs] {
		fm["source_refs"] = refs
		return nil
	}
	fmt.Fprintln(cmd.ErrOrStderr(), i18n.T("cli.source.warn_refs_off"))
	return nil
}
