package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// new 커맨드의 플래그 이름이다.
const (
	flagForm   = "form"
	flagTopics = "topics"
	flagTags   = "tags"
)

// newNewCmd는 처음부터 검수된 지식으로 context에 쓰는 new 커맨드를 반환한다.
func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new " + i18n.T("usage.args.title"),
		Short: i18n.T("cli.new.short"),
		Long:  i18n.T("cli.new.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			// capture, source와 같은 경로를 지난다. 명시한 슬러그도
			// 파일시스템 안전 검사를 받는다(ADR 0045).
			slug, err := resolveSlug(slugFlag, title)
			if err != nil {
				return err
			}
			fm, err := newFrontmatter(cmd, cfg)
			if err != nil {
				return err
			}
			relatedRaw, err := cmd.Flags().GetStringArray(flagRelated)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagRelated), err)
			}
			related := splitRelated(relatedRaw)
			if len(related) > 0 {
				values := make([]string, 0, len(related))
				for _, s := range related {
					values = append(values, linkValue(s))
				}
				fm["related"] = values
			}
			fm["created"] = Now(cmd).Format("2006-01-02")

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.walk_fail"), err)
			}
			warnUnknownRelated(cmd.ErrOrStderr(), related, knownSlugs(walked))

			links := map[string]bool{}
			for _, s := range related {
				links[s] = true
			}
			resolved, targets := gateInputs(links, walked, "", slug)
			g := lint.EvaluateGate(resolved, targets, cfg.Thresholds.MinWikilinks)
			if !g.Passed {
				return gateRejectError(g)
			}
			if g.Deferred {
				warnDeferred(cmd.ErrOrStderr(), g)
			}

			path, err := wiki.Create(root, cfg, wiki.StageContext, "", slug, fm, skeletonBody(title))
			if err != nil {
				return err
			}
			res := writeOutcome{Path: path, Slug: slug, Stage: "context", Gate: gateOf(g)}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			fmt.Fprintln(cmd.OutOrStdout(), i18n.T("cli.new.done", res.Path))
			fmt.Fprintln(cmd.OutOrStdout(), i18n.T("cli.ingest.gate_line",
				res.Gate.Links, res.Gate.Targets, res.Gate.Min, deferredNote(res.Gate.Deferred)))
			fmt.Fprintln(cmd.OutOrStdout(), i18n.T("cli.new.next"))
			return nil
		},
	}
	cmd.Flags().String(flagSlug, "", i18n.T("cli.new.flag_slug"))
	cmd.Flags().String(flagType, "", i18n.T("cli.new.flag_type"))
	cmd.Flags().String(flagForm, "", i18n.T("cli.new.flag_form"))
	cmd.Flags().StringArray(flagTopics, nil, i18n.T("cli.new.flag_topics"))
	cmd.Flags().StringArray(flagTags, nil, i18n.T("cli.new.flag_tags"))
	cmd.Flags().StringArray(flagRelated, nil, i18n.T("cli.new.flag_related"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// newFrontmatter는 new 커맨드의 프론트매터를 만든다. context 단계 초기값에
// 플래그로 준 값을 얹고 허용값 밖의 값은 목록과 함께 거절한다.
func newFrontmatter(cmd *cobra.Command, cfg config.Config) (map[string]any, error) {
	fm := wiki.Frontmatter(wiki.StageContext, cfg)
	if v, err := stringFlag(cmd, flagType); err != nil {
		return nil, err
	} else if v != "" {
		if !containsString(cfg.Schema.Types, v) {
			return nil, errors.New(i18n.T("cli.ingest.type_invalid",
				v, strings.Join(cfg.Schema.Types, ", ")))
		}
		fm["type"] = v
	}
	if v, err := stringFlag(cmd, flagForm); err != nil {
		return nil, err
	} else if v != "" {
		forms := cfg.Schema.Taxonomy.Forms.Values
		if !containsString(forms, v) {
			if len(forms) == 0 {
				return nil, errors.New(i18n.T("cli.new.form_empty", v))
			}
			return nil, errors.New(i18n.T("cli.new.form_invalid",
				v, strings.Join(forms, ", ")))
		}
		fm["form"] = v
	}
	if v, err := cmd.Flags().GetStringArray(flagTopics); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagTopics), err)
	} else if len(v) > 0 {
		fm["topics"] = v
	}
	if v, err := cmd.Flags().GetStringArray(flagTags); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagTags), err)
	} else if len(v) > 0 {
		fm["tags"] = v
	}
	return fm, nil
}

// skeletonBody는 승급 문서의 최소 골격을 만든다. upstream
// promotion-rules.md가 요구하는 절 제목을 빈 상태로 둔다. 내용을 지어내지 않는다.
func skeletonBody(title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", title)
	for _, sec := range []string{"결론", "맥락", "현재 이해", "근거", "관련 링크"} {
		fmt.Fprintf(&b, "\n## %s\n", sec)
	}
	return b.String()
}

// containsString은 값이 목록에 있는지 검사한다.
func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
