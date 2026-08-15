package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
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
		Use:   "new <제목>",
		Short: "처음부터 검수된 지식으로 context에 씁니다",
		Long: `검수된 지식을 곧바로 context 단계로 씁니다.

승급 게이트는 promote와 같은 함수로 판정합니다. 링크가 부족하면
--related <슬러그>를 반복해 이 자리에서 채울 수 있습니다.
본문은 upstream promotion-rules.md가 요구하는 절 제목의 골격만 넣습니다.`,
		Args: cobra.ExactArgs(1),
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
			slug := slugFlag
			if slug == "" {
				slug, err = wiki.Slug(title)
				if err != nil {
					return err
				}
			}
			fm, err := newFrontmatter(cmd, cfg)
			if err != nil {
				return err
			}
			related, err := cmd.Flags().GetStringArray(flagRelated)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagRelated, err)
			}
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
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			warnUnknownRelated(cmd.ErrOrStderr(), related, knownSlugs(walked))

			links := map[string]bool{}
			for _, s := range related {
				links[s] = true
			}
			g := lint.EvaluateGate(len(links), countTargets(walked, "", slug), cfg.Thresholds.MinWikilinks)
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
			fmt.Fprintf(cmd.OutOrStdout(), "context에 썼습니다: %s\n", res.Path)
			fmt.Fprintf(cmd.OutOrStdout(), "게이트: 링크 %d개, 대상 %d개, 기준 %d개%s\n",
				res.Gate.Links, res.Gate.Targets, res.Gate.Min, deferredNote(res.Gate.Deferred))
			fmt.Fprintf(cmd.OutOrStdout(), "다음: 골격의 절을 채우고 engram lint로 확인하세요\n")
			return nil
		},
	}
	cmd.Flags().String(flagSlug, "", "문서 슬러그. 생략하면 제목에서 만듭니다")
	cmd.Flags().String(flagType, "", "문서 종류. 기본값은 context 단계 초기값입니다")
	cmd.Flags().String(flagForm, "", "문서 형태. forms 폐쇄 집합의 값이어야 합니다")
	cmd.Flags().StringArray(flagTopics, nil, "주제. 여러 번 쓸 수 있습니다")
	cmd.Flags().StringArray(flagTags, nil, "광범위 묶음 태그. 여러 번 쓸 수 있습니다")
	cmd.Flags().StringArray(flagRelated, nil, "관련 문서 슬러그. 여러 번 쓸 수 있습니다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
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
			return nil, fmt.Errorf("--type 값이 허용값 밖입니다: %q (허용값: %s)",
				v, strings.Join(cfg.Schema.Types, ", "))
		}
		fm["type"] = v
	}
	if v, err := stringFlag(cmd, flagForm); err != nil {
		return nil, err
	} else if v != "" {
		forms := cfg.Schema.Taxonomy.Forms.Values
		if !containsString(forms, v) {
			if len(forms) == 0 {
				return nil, fmt.Errorf("--form 값을 받을 수 없습니다: %q\n위키 설정의 forms가 비어 있습니다. engram.yaml의 forms에 값을 정의하세요", v)
			}
			return nil, fmt.Errorf("--form 값이 forms 폐쇄 집합에 없습니다: %q (허용값: %s)",
				v, strings.Join(forms, ", "))
		}
		fm["form"] = v
	}
	if v, err := cmd.Flags().GetStringArray(flagTopics); err != nil {
		return nil, fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagTopics, err)
	} else if len(v) > 0 {
		fm["topics"] = v
	}
	if v, err := cmd.Flags().GetStringArray(flagTags); err != nil {
		return nil, fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagTags, err)
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
