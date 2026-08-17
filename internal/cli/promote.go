package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
	"github.com/spf13/cobra"
)

// newPromoteCmd는 기존 문서를 context 단계로 올리는 promote 커맨드를 반환한다.
// inbox 문서는 이동하고 sources 문서는 파생을 만든다. 원본 보존 계층의
// 본문은 절대 고치지 않는다. ADR 0009.
func newPromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote " + i18n.T("usage.args.path"),
		Short: i18n.T("cli.promote.short"),
		Long:  i18n.T("cli.promote.long"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := ingestTarget(cmd)
			if err != nil {
				return err
			}
			srcPath, err := resolveDocPath(root, args[0])
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_read_fail", srcPath), err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_parse_fail", srcPath), err)
			}
			stage := fieldString(d, "artifact_stage")
			switch stage {
			case "inbox", "source":
			case "context":
				return errors.New(i18n.T("cli.promote.already_context", srcPath))
			default:
				return errors.New(i18n.T("cli.promote.unknown_stage", srcPath, stage))
			}

			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			slug := slugFlag
			if slug == "" {
				slug = stripDatePrefix(filepath.Base(srcPath))
			}
			relatedRaw, err := cmd.Flags().GetStringArray(flagRelated)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagRelated), err)
			}
			related := splitRelated(relatedRaw)
			// 문서 종류는 내용을 읽어야 아는 판단이라 도구가 정하지 못한다.
			// 사용자가 준 값만 반영하고 추론하지 않는다. ADR 0014.
			typeFlag, err := stringFlag(cmd, flagType)
			if err != nil {
				return err
			}
			if typeFlag != "" && !containsString(cfg.Schema.Types, typeFlag) {
				return errors.New(i18n.T("cli.ingest.type_invalid",
					typeFlag, strings.Join(cfg.Schema.Types, ", ")))
			}
			if typeFlag == "" {
				warnStageDefaultType(cmd.ErrOrStderr(), stage, fieldString(d, "type"), cfg)
			}

			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.rel_fail"), err)
			}
			rel = filepath.ToSlash(rel)

			// 승급 갱신. 파생의 경우 derived_from에 원본 경로를 남긴다.
			derivedFrom := ""
			if stage == "source" {
				derivedFrom = rel
			}
			fields := promoteFields(d.Fields, related, derivedFrom)
			fields = fillContextFields(fields, cfg)
			fields = fillCreated(fields, filepath.Base(srcPath), Now(cmd))
			if typeFlag != "" {
				fields = upsertField(fields, doc.Field{Key: "type", Kind: doc.KindString, Str: typeFlag})
			}
			updated := d
			updated.Fields = fields

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.walk_fail"), err)
			}
			warnUnknownRelated(cmd.ErrOrStderr(), related, knownSlugs(walked))

			destRel, err := wiki.FilePath(cfg, wiki.StageContext, "", slug)
			if err != nil {
				return err
			}
			destPath := filepath.Join(root, filepath.FromSlash(destRel))
			if _, err := os.Stat(destPath); err == nil {
				return errors.New(i18n.T("cli.ingest.dest_exists", destPath))
			}

			resolved, targets := gateInputs(linkSlugs(updated), walked, rel, slug)
			g := lint.EvaluateGate(resolved, targets, cfg.Thresholds.MinWikilinks)
			if !g.Passed {
				return gateRejectError(g)
			}
			if g.Deferred {
				warnDeferred(cmd.ErrOrStderr(), g)
			}

			dryRun, err := boolFlag(cmd, flagDryRun)
			if err != nil {
				return err
			}
			if dryRun {
				res := promoteOutcome{
					writeOutcome: writeOutcome{Path: destPath, Slug: slug, Stage: "context", Gate: gateOf(g)},
					Type:         finalType(fields),
					DryRun:       true,
				}
				if jsonOutput(cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(res)
				}
				printPromotePlan(cmd.OutOrStdout(), srcPath, res)
				return nil
			}

			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if stage == "inbox" {
				if err := os.Remove(srcPath); err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.promote.inbox_remove_fail", srcPath), err)
				}
			}
			if stage == "source" && cfg.Axes[config.AxisDerivedContext] {
				// 역방향 기록. 프론트매터 필드만 갱신하고 본문은 그대로 둔다.
				srcFields := mergeListField(d.Fields, "derived_context", []string{slug})
				if err := os.WriteFile(srcPath, doc.Render(srcFields, d.Body), 0o644); err != nil {
					return fmt.Errorf("%s: %w", i18n.T("cli.promote.derived_context_fail", srcPath), err)
				}
			}

			res := promoteOutcome{
				writeOutcome: writeOutcome{Path: destPath, Slug: slug, Stage: "context", Gate: gateOf(g)},
				Type:         finalType(fields),
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printPromoted(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().String(flagSlug, "", i18n.T("cli.promote.flag_slug"))
	cmd.Flags().StringArray(flagRelated, nil, i18n.T("cli.promote.flag_related"))
	cmd.Flags().String(flagType, "", i18n.T("cli.promote.flag_type"))
	cmd.Flags().Bool(flagDryRun, false, i18n.T("cli.promote.flag_dry_run"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// promoteOutcome은 promote의 결과다. 공통 결과에 문서 종류를 더 낸다.
type promoteOutcome struct {
	writeOutcome
	Type   string `json:"type"`
	DryRun bool   `json:"dryRun,omitempty"`
}

// finalType은 완성된 필드에서 문서 종류를 꺼낸다.
func finalType(fields []doc.Field) string {
	for _, f := range fields {
		if f.Key == "type" && f.Kind == doc.KindString {
			return f.Str
		}
	}
	return ""
}

// warnStageDefaultType은 문서 종류를 지정받지 못해 단계 기본값이 그대로
// 남았음을 알린다. 승급을 막을 사유가 아니므로 경고로만 낸다.
// 기본값의 진실원은 wiki의 단계별 초기값이다.
func warnStageDefaultType(w io.Writer, stage, cur string, cfg config.Config) {
	var def string
	switch stage {
	case "inbox":
		def, _ = wiki.Frontmatter(wiki.StageInbox, cfg)["type"].(string)
	case "source":
		def, _ = wiki.Frontmatter(wiki.StageSource, cfg)["type"].(string)
	}
	if cur == "" || cur != def {
		return
	}
	fmt.Fprintln(w, i18n.T("cli.promote.warn_default_type",
		stage, cur, strings.Join(cfg.Schema.Types, ", ")))
}

// printPromoted은 만들어진 경로와 다음에 할 수 있는 일을 낸다.
func printPromoted(w io.Writer, res promoteOutcome) {
	fmt.Fprintln(w, i18n.T("cli.promote.done", res.Path))
	fmt.Fprintln(w, i18n.T("cli.promote.type_line", res.Type))
	fmt.Fprintln(w, i18n.T("cli.ingest.gate_line", res.Gate.Links, res.Gate.Targets, res.Gate.Min,
		deferredNote(res.Gate.Deferred)))
	fmt.Fprintln(w, i18n.T("cli.promote.next"))
}

// printPromotePlan은 --dry-run 결과를 낸다. 승급이 지금 성공할지를
// 아무것도 쓰지 않고 알린다. 게이트가 막으면 여기까지 오지 않고 거절
// 메시지가 나가므로, 이 출력이 나왔다는 것은 통과한다는 뜻이다.
func printPromotePlan(w io.Writer, srcPath string, res promoteOutcome) {
	fmt.Fprintln(w, i18n.T("cli.promote.plan", srcPath, res.Path))
	fmt.Fprintln(w, i18n.T("cli.promote.type_line", res.Type))
	fmt.Fprintln(w, i18n.T("cli.ingest.gate_line", res.Gate.Links, res.Gate.Targets, res.Gate.Min,
		deferredNote(res.Gate.Deferred)))
	fmt.Fprintln(w, i18n.T("cli.ingest.dry_run_note"))
}

// deferredNote는 유예 안내 문구를 낸다.
func deferredNote(deferred bool) string {
	if deferred {
		return i18n.T("cli.ingest.deferred_note")
	}
	return ""
}

// resolveDocPath는 인자로 받은 경로를 파일 경로로 확정한다.
// 위키 루트 상대 경로를 먼저 본다. 커맨드 안내와 문서가 전부 그 형태를
// 쓰기 때문이다. 셸의 작업 디렉토리가 마침 위키 루트일 때 인자를 그대로
// 돌려주면 뒤의 filepath.Rel 이 절대 경로인 루트와 상대 경로를 섞어 받아
// 깨졌다. 어느 쪽을 골랐든 루트와 절대/상대를 맞춰서 낸다.
func resolveDocPath(wikiRoot, arg string) (string, error) {
	for _, c := range []string{filepath.Join(wikiRoot, arg), arg} {
		st, err := os.Stat(c)
		if err != nil || st.IsDir() {
			continue
		}
		if filepath.IsAbs(wikiRoot) && !filepath.IsAbs(c) {
			abs, err := filepath.Abs(c)
			if err != nil {
				return "", fmt.Errorf("%s: %w", i18n.T("cli.ingest.rel_fail"), err)
			}
			return abs, nil
		}
		return c, nil
	}
	return "", errors.New(i18n.T("cli.ingest.doc_not_found", arg))
}

// fieldString은 문자열 필드 값을 반환한다.
func fieldString(d doc.Doc, key string) string {
	for _, f := range d.Fields {
		if f.Key == key && f.Kind == doc.KindString {
			return f.Str
		}
	}
	return ""
}

// stripDatePrefix는 파일명에서 날짜 접두사를 뗀다. context 문서는 날짜
// 접두사를 쓰지 않는다. ADR 0020.
func stripDatePrefix(name string) string {
	base := strings.TrimSuffix(name, ".md")
	if m := regexp.MustCompile(`^\d{4}-\d{2}(-\d{2})?-`).FindString(base); m != "" {
		return strings.TrimPrefix(base, m)
	}
	return base
}

// datePrefix는 파일명 앞의 날짜 접두사를 반환한다. 없으면 빈 문자열이다.
func datePrefix(name string) string {
	m := regexp.MustCompile(`^(\d{4}-\d{2}(-\d{2})?)-`).FindStringSubmatch(strings.TrimSuffix(name, ".md"))
	if m == nil {
		return ""
	}
	return m[1]
}

// fillCreated는 created가 비어 있을 때 채운다. 값의 우선순위는 기존
// 프론트매터, 파일명의 날짜 접두사, 기준 시각 순이다. 승급 문서에
// 날짜가 없으면 resurface와 digest가 그 문서를 대상에서 빼므로
// 여기서 반드시 확정한다.
func fillCreated(fields []doc.Field, srcName string, now time.Time) []doc.Field {
	for _, f := range fields {
		if f.Key == "created" && f.Str != "" {
			return fields
		}
	}
	date := datePrefix(srcName)
	if date == "" {
		date = now.Format("2006-01-02")
	}
	return upsertField(fields, doc.Field{Key: "created", Kind: doc.KindDate, Str: date})
}

// promoteFields는 승급에 맞게 프론트매터 필드를 갱신한다.
// artifact_stage, status, indexable을 바꾸고 지정한 related와 derived_from을
// 합친다. 기존 키 순서를 보존한다. 입력 슬라이스를 바꾸지 않으므로
// 파생 원본의 역방향 기록은 원본 파싱 결과를 그대로 쓸 수 있다.
func promoteFields(src []doc.Field, related []string, derivedFrom string) []doc.Field {
	fields := append([]doc.Field(nil), src...)
	fields = upsertField(fields, doc.Field{Key: "artifact_stage", Kind: doc.KindString, Str: "context"})
	fields = upsertField(fields, doc.Field{Key: "status", Kind: doc.KindString, Str: "promoted"})
	fields = upsertField(fields, doc.Field{Key: "indexable", Kind: doc.KindBool, Bool: true})
	if len(related) > 0 {
		fields = mergeListField(fields, "related", related)
	}
	if derivedFrom != "" {
		fields = mergeListField(fields, "derived_from", []string{derivedFrom})
	}
	return fields
}

// fillContextFields는 context 단계가 요구하는 필드 중 문서에 없는 것을
// 빈 값으로 채운다. 필드 목록의 진실원은 wiki의 context 단계 초기값이라
// 여기에 목록을 따로 두지 않는다. 이미 값이 있는 필드은 그대로 두고
// 꺼진 속성은 초기값에 애초에 없으므로 추가되지 않는다.
// 결과는 internal/doc의 표준 키 순서를 따른다.
func fillContextFields(src []doc.Field, cfg config.Config) []doc.Field {
	byKey := map[string]doc.Field{}
	var order []string
	add := func(f doc.Field) {
		if _, ok := byKey[f.Key]; !ok {
			byKey[f.Key] = f
			order = append(order, f.Key)
		}
	}
	for _, f := range src {
		add(f)
	}
	for key, v := range wiki.Frontmatter(wiki.StageContext, cfg) {
		if _, ok := byKey[key]; !ok {
			add(emptyField(key, v))
		}
	}
	// 표준 순서를 먼저 놓고 표준에 없는 기존 키는 기존 순서로 뒤에 둔다.
	rank := map[string]int{}
	for i, k := range doc.StandardKeys() {
		rank[k] = i
	}
	keys := append([]string(nil), order...)
	sort.SliceStable(keys, func(i, j int) bool {
		ri, oki := rank[keys[i]]
		rj, okj := rank[keys[j]]
		switch {
		case oki && okj:
			return ri < rj
		case oki:
			return true
		case okj:
			return false
		}
		return false
	})
	out := make([]doc.Field, 0, len(keys))
	for _, k := range keys {
		out = append(out, byKey[k])
	}
	return out
}

// emptyField는 채울 값으로 필드 하나를 만든다. 관계 필드는 빈 배열이고
// source_channel처럼 값이 없는 키는 빈 값으로 둔다.
func emptyField(key string, v any) doc.Field {
	switch val := v.(type) {
	case nil:
		return doc.Field{Key: key, Kind: doc.KindEmpty}
	case bool:
		return doc.Field{Key: key, Kind: doc.KindBool, Bool: val}
	case []string:
		return doc.Field{Key: key, Kind: doc.KindStringList, List: val}
	case string:
		return doc.Field{Key: key, Kind: doc.KindString, Str: val}
	}
	return doc.Field{Key: key, Kind: doc.KindEmpty}
}

// upsertField는 필드를 바꾼다. 키가 없으면 끝에 추가한다.
func upsertField(fields []doc.Field, f doc.Field) []doc.Field {
	for i := range fields {
		if fields[i].Key == f.Key {
			fields[i] = f
			return fields
		}
	}
	return append(fields, f)
}

// mergeListField는 목록 필드에 값을 중복 없이 합친다. 필드가 없으면 만든다.
// related만 위키링크 껍데기를 입히고 나머지(derived_from 같은 경로 필드)는
// 값을 그대로 둔다. 중복은 껍데기를 벗긴 값으로 잡는다.
func mergeListField(fields []doc.Field, key string, values []string) []doc.Field {
	var list []string
	for i := range fields {
		if fields[i].Key == key {
			list = fields[i].List
			break
		}
	}
	seen := map[string]bool{}
	for _, v := range list {
		seen[bareLinkValue(v)] = true
	}
	for _, v := range values {
		if seen[bareLinkValue(v)] {
			continue
		}
		if key == "related" {
			v = linkValue(v)
		}
		list = append(list, v)
		seen[bareLinkValue(v)] = true
	}
	return upsertField(fields, doc.Field{Key: key, Kind: doc.KindStringList, List: list})
}

// linkValue는 슬러그를 위키링크 값 형식으로 낸다. upstream 계약이
// related 항목을 이 형태로 쓴다.
func linkValue(slug string) string {
	return "[[" + slug + "]]"
}

// bareLinkValue는 위키링크 값에서 슬러그를 꺼낸다.
func bareLinkValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[[")
	return strings.TrimSuffix(v, "]]")
}
