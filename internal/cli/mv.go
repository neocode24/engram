package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/state"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// flagDryRun은 mv의 시험 실행 플래그 이름이다.
const flagDryRun = "dry-run"

// newMvCmd는 문서의 슬러그를 바꾸고 모든 링크를 따라가 고치는 mv 커맨드를
// 반환한다. 링크 갱신이 이 커맨드의 본체다. mv가 백링크를 따라가지
// 않으면 링크 무결성이 즉시 깨지고 게이트의 의미가 사라진다.
func newMvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mv " + i18n.T("usage.args.slug_rename"),
		Short: i18n.T("cli.mv.short"),
		Long:  i18n.T("cli.mv.long"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			oldSlug := graph.Normalize(args[0])
			newSlug, err := wiki.Slug(args[1])
			if err != nil {
				return err
			}
			if oldSlug == "" {
				return errors.New(i18n.T("cli.mv.old_empty", args[0]))
			}
			if oldSlug == newSlug {
				return errors.New(i18n.T("cli.mv.same_slug", newSlug))
			}

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.walk_fail"), err)
			}
			g := graph.Build(walked)

			// 옛 슬러그의 문서를 찾는다.
			var srcRel string
			for _, w := range walked {
				if graph.Normalize(w.Rel) == oldSlug {
					srcRel = w.Rel
					break
				}
			}
			if srcRel == "" {
				return errors.New(i18n.T("cli.mv.old_not_found", args[0]))
			}
			if g.Has(newSlug) {
				return errors.New(i18n.T("cli.mv.slug_taken", newSlug))
			}

			backlinks := g.Backlinks(oldSlug)

			// 새 경로. 날짜 접두사는 원본에서 유지한다.
			base := strings.TrimSuffix(filepath.Base(srcRel), ".md")
			prefix := base[:len(base)-len(graph.Normalize(base))]
			dir := filepath.ToSlash(filepath.Dir(srcRel))
			dstRel := dir + "/" + prefix + newSlug + ".md"
			srcPath := filepath.Join(root, filepath.FromSlash(srcRel))
			dstPath := filepath.Join(root, filepath.FromSlash(dstRel))
			if _, err := os.Stat(dstPath); err == nil {
				return errors.New(i18n.T("cli.mv.dest_exists", dstPath))
			}

			// 파일별 고친 링크 수를 센다. 출력과 JSON 에 쓴다.
			perFile := map[string]int{}
			for _, l := range backlinks {
				perFile[l.From]++
			}

			dryRun, err := cmd.Flags().GetBool(flagDryRun)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagDryRun), err)
			}
			if !dryRun {
				// 링크를 먼저 다 고친다. 파일 이동은 마지막이다.
				for _, l := range backlinks {
					if err := rewriteLinks(root, l.From, oldSlug, newSlug); err != nil {
						return err
					}
				}
				if err := os.Rename(srcPath, dstPath); err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.mv.rename_fail", dstPath), err)
				}
			}

			// bridge 기각 쌍도 옛 슬러그를 가리킨다. 함께 고치지 않으면
			// 기각이 조용히 무효가 되어 같은 쌍이 다시 제시된다. ADR 0028.
			rejections, err := renameRejections(root, oldSlug, newSlug, dryRun)
			if err != nil {
				return err
			}

			res := mvOutcome{From: srcPath, To: dstPath, Slug: newSlug, DryRun: dryRun, Rejections: rejections}
			for from, n := range perFile {
				res.Updated = append(res.Updated, mvUpdated{Path: from, Links: n})
			}
			sortMvUpdated(res.Updated)
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printMoved(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().Bool(flagDryRun, false, i18n.T("cli.mv.flag_dry_run"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// mvOutcome은 mv의 결과다.
type mvOutcome struct {
	From    string      `json:"from"`
	To      string      `json:"to"`
	Slug    string      `json:"slug"`
	DryRun  bool        `json:"dryRun"`
	Updated []mvUpdated `json:"updated"`
	// Rejections는 함께 고친 bridge 기각 쌍의 수다.
	Rejections int `json:"rejections"`
}

// renameRejections는 engram-state.yaml의 기각 쌍에서 옛 슬러그를 새
// 슬러그로 바꾸고 고친 쌍의 수를 반환한다. 상태 파일이 없으면 0이다.
// dryRun이면 세기만 하고 쓰지 않는다.
func renameRejections(root, oldSlug, newSlug string, dryRun bool) (int, error) {
	st, err := state.Load(root)
	if err != nil {
		return 0, err
	}
	from, to := graph.Normalize(oldSlug), graph.Normalize(newSlug)
	var pairs [][2]string
	changed := 0
	for _, p := range st.BridgeRejections {
		a, b := p[0], p[1]
		if a == from || b == from {
			changed++
			if a == from {
				a = to
			}
			if b == from {
				b = to
			}
		}
		pairs = append(pairs, [2]string{a, b})
	}
	if changed == 0 || dryRun {
		return changed, nil
	}
	// 정규형(정렬과 중복 제거)은 Reject가 보장하므로 빈 상태에 다시 넣는다.
	next := state.State{}
	for _, p := range pairs {
		next.Reject(p[0], p[1])
	}
	if err := next.Save(root); err != nil {
		return 0, err
	}
	return changed, nil
}

// mvUpdated는 파일 하나에서 고친 링크 수다.
type mvUpdated struct {
	Path  string `json:"path"`
	Links int    `json:"links"`
}

// sortMvUpdated는 갱신 목록을 경로순으로 정렬해 출력이 결정론적이게 한다.
func sortMvUpdated(list []mvUpdated) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j-1].Path > list[j].Path; j-- {
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
}

// rewriteLinks는 출처 파일 하나의 링크를 옛 슬러그에서 새 슬러그로 고친다.
// 파일의 다른 내용은 그대로 두고 키 순서도 보존한다.
func rewriteLinks(root, rel, oldSlug, newSlug string) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.mv.link_read_fail", path), err)
	}
	d, err := doc.Parse(rel, raw)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.mv.link_parse_fail", path), err)
	}

	body := rewriteBodyLinks(d.Body, oldSlug, newSlug)
	fields := append([]doc.Field(nil), d.Fields...)
	for i := range fields {
		if !isLinkField(string(fields[i].Key)) {
			continue
		}
		fields[i] = rewriteFieldValues(fields[i], oldSlug, newSlug)
	}
	if err := os.WriteFile(path, doc.Render(fields, body), 0o644); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("cli.mv.link_write_fail", path), err)
	}
	return nil
}

// isLinkField는 링크를 담는 프론트매터 필드인지 본다.
func isLinkField(key string) bool {
	switch key {
	case "related", "derived_from", "derived_context", "source_refs":
		return true
	}
	return false
}

// rewriteFieldValues은 링크 필드의 값에서 옛 슬러그를 새 슬러그로 바꾼다.
// 값이 위키링크든 슬러그든 경로든 정규화 슬러그가 같으면 고친다.
func rewriteFieldValues(f doc.Field, oldSlug, newSlug string) doc.Field {
	switch f.Kind {
	case doc.KindStringList:
		list := append([]string(nil), f.List...)
		for i, v := range list {
			if graph.Normalize(v) == oldSlug {
				list[i] = replacedValue(v, newSlug)
			}
		}
		f.List = list
	case doc.KindString:
		if f.Str != "" && graph.Normalize(f.Str) == oldSlug {
			f.Str = replacedValue(f.Str, newSlug)
		}
	}
	return f
}

// replacedValue는 원본 값의 형태를 유지해 새 슬러그 값을 만든다.
// 위키링크 껍데기와 경로의 날짜 접두사를 보존한다.
func replacedValue(raw, newSlug string) string {
	if strings.HasPrefix(raw, "[[") {
		return "[[" + newSlug + "]]"
	}
	base := filepath.Base(raw)
	rest := filepath.Dir(raw)
	name := strings.TrimSuffix(base, ".md")
	prefix := name[:len(name)-len(graph.Normalize(name))]
	if rest == "." {
		return prefix + newSlug + ".md"
	}
	return rest + "/" + prefix + newSlug + ".md"
}

// rewriteBodyLinks는 본문의 위키링크에서 슬러그만 바꾼다.
// 표시 문자열과 헤딩은 보존한다. 코드 펜스와 인라인 코드 안은 고치지
// 않는다. 문서에 링크 문법을 설명한 자리가 바뀌면 안 되기 때문이다.
func rewriteBodyLinks(body, oldSlug, newSlug string) string {
	lines := strings.Split(body, "\n")
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// 백틱을 기준으로 나눠 짝수 구간만 고친다. 인라인 코드 밖이다.
		parts := strings.Split(line, "`")
		for j := range parts {
			if j%2 == 0 {
				parts[j] = replaceSlugOccurrence(parts[j], oldSlug, newSlug)
			}
		}
		lines[i] = strings.Join(parts, "`")
	}
	return strings.Join(lines, "\n")
}

// replaceSlugOccurrence는 세 링크 형태의 슬러그만 바꾼다.
// 접두만 겹치는 긴 슬러그를 같은 것으로 보지 않게 정확히 맞춘다.
func replaceSlugOccurrence(s, oldSlug, newSlug string) string {
	s = strings.ReplaceAll(s, "[["+oldSlug+"]]", "[["+newSlug+"]]")
	s = strings.ReplaceAll(s, "[["+oldSlug+"|", "[["+newSlug+"|")
	s = strings.ReplaceAll(s, "[["+oldSlug+"#", "[["+newSlug+"#")
	return s
}

// printMoved은 옮긴 경로와 고친 링크를 파일별로 낸다.
func printMoved(w io.Writer, res mvOutcome) {
	verb := i18n.T("cli.mv.verb_moved")
	if res.DryRun {
		verb = i18n.T("cli.mv.verb_will_change")
	}
	fmt.Fprintln(w, i18n.T("cli.mv.summary", verb, res.From, res.To, res.Slug))
	if len(res.Updated) == 0 {
		fmt.Fprintln(w, i18n.T("cli.mv.no_links"))
	} else {
		total := 0
		for _, u := range res.Updated {
			total += u.Links
		}
		fmt.Fprintln(w, i18n.T("cli.mv.links_total", total))
		for _, u := range res.Updated {
			fmt.Fprintln(w, i18n.T("cli.mv.links_per_file", u.Path, u.Links))
		}
	}
	if res.Rejections > 0 {
		fmt.Fprintln(w, i18n.T("cli.mv.rejections_fixed", res.Rejections))
	}
	if res.DryRun {
		fmt.Fprintln(w, i18n.T("cli.mv.dry_run_note"))
	} else {
		fmt.Fprintln(w, i18n.T("cli.mv.next"))
	}
}
