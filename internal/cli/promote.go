package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
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
		Use:   "promote <경로>",
		Short: "기존 문서를 context 단계로 올린다",
		Long: `기존 문서를 승급 게이트로 검사해 context 단계로 올린다.

inbox 문서는 이동한다. 원본이 남지 않는다. inbox는 임시 계층이다.
sources 문서는 파생을 만든다. 원본이 그대로 남는다. sources는 원본
보존 계층이고 이후 본문을 고치지 않는 것이 계약이다.

게이트는 lint.EvaluateGate가 단일 진실원이다. 링크가 부족하면
--related <슬러그>를 반복해 이 자리에서 채울 수 있다.`,
		Args: cobra.ExactArgs(1),
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
				return fmt.Errorf("문서를 읽을 수 없음: %s: %w", srcPath, err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("문서를 파싱할 수 없음: %s: %w", srcPath, err)
			}
			stage := fieldString(d, "artifact_stage")
			switch stage {
			case "inbox", "source":
			case "context":
				return fmt.Errorf("이미 context 단계다: %s\npromote는 inbox나 sources 문서를 올린다", srcPath)
			default:
				return fmt.Errorf("문서 단계를 알 수 없다: %s의 artifact_stage 값이 %q다 (허용값: inbox, source, context)", srcPath, stage)
			}

			slugFlag, err := stringFlag(cmd, flagSlug)
			if err != nil {
				return err
			}
			slug := slugFlag
			if slug == "" {
				slug = stripDatePrefix(filepath.Base(srcPath))
			}
			related, err := cmd.Flags().GetStringArray(flagRelated)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagRelated, err)
			}
			// 문서 종류는 내용을 읽어야 아는 판단이라 도구가 정하지 못한다.
			// 사용자가 준 값만 반영하고 추론하지 않는다. ADR 0014.
			typeFlag, err := stringFlag(cmd, flagType)
			if err != nil {
				return err
			}
			if typeFlag != "" && !containsString(cfg.Schema.Types, typeFlag) {
				return fmt.Errorf("--type 값이 허용값 밖이다: %q (허용값: %s)",
					typeFlag, strings.Join(cfg.Schema.Types, ", "))
			}
			if typeFlag == "" {
				warnStageDefaultType(cmd.ErrOrStderr(), stage, fieldString(d, "type"), cfg)
			}

			rel, err := filepath.Rel(root, srcPath)
			if err != nil {
				return fmt.Errorf("위키 루트 기준 경로를 계산할 수 없음: %w", err)
			}
			rel = filepath.ToSlash(rel)

			// 승급 갱신. 파생의 경우 derived_from에 원본 경로를 남긴다.
			derivedFrom := ""
			if stage == "source" {
				derivedFrom = rel
			}
			fields := promoteFields(d.Fields, related, derivedFrom)
			fields = fillContextFields(fields, cfg)
			if typeFlag != "" {
				fields = upsertField(fields, doc.Field{Key: "type", Kind: doc.KindString, Str: typeFlag})
			}
			updated := d
			updated.Fields = fields

			walked, err := walk.Files(root, cfg)
			if err != nil {
				return fmt.Errorf("위키를 순회할 수 없음: %w", err)
			}
			warnUnknownRelated(cmd.ErrOrStderr(), related, knownSlugs(walked))

			destRel, err := wiki.FilePath(cfg, wiki.StageContext, "", slug)
			if err != nil {
				return err
			}
			destPath := filepath.Join(root, filepath.FromSlash(destRel))
			if _, err := os.Stat(destPath); err == nil {
				return fmt.Errorf("도착지에 이미 문서가 있다: %s\n기존 문서를 덮어쓰지 않는다. 슬러그를 다르게 지정한다", destPath)
			}

			g := lint.EvaluateGate(len(linkSlugs(updated)), countTargets(walked, rel, slug), cfg.Thresholds.MinWikilinks)
			if !g.Passed {
				return gateRejectError(g)
			}
			if g.Deferred {
				warnDeferred(cmd.ErrOrStderr(), g)
			}

			if err := createDocFile(destPath, doc.Render(fields, d.Body)); err != nil {
				return err
			}
			if stage == "inbox" {
				if err := os.Remove(srcPath); err != nil {
					return fmt.Errorf("inbox 원본을 지울 수 없음: %s: %w", srcPath, err)
				}
			}
			if stage == "source" && cfg.Axes[config.AxisDerivedContext] {
				// 역방향 기록. 프론트매터 필드만 갱신하고 본문은 그대로 둔다.
				srcFields := mergeListField(d.Fields, "derived_context", []string{slug})
				if err := os.WriteFile(srcPath, doc.Render(srcFields, d.Body), 0o644); err != nil {
					return fmt.Errorf("원본의 derived_context를 갱신할 수 없음: %s: %w", srcPath, err)
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
	cmd.Flags().String(flagSlug, "", "도착지 문서 슬러그. 생략하면 원본 파일명에서 날짜 접두사를 뗀 값이다")
	cmd.Flags().StringArray(flagRelated, nil, "related 필드에 추가할 슬러그. 여러 번 쓸 수 있다")
	cmd.Flags().String(flagType, "", "승급 문서의 문서 종류. 허용값은 위키 설정의 types다")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// promoteOutcome은 promote의 결과다. 공통 결과에 문서 종류를 더 낸다.
type promoteOutcome struct {
	writeOutcome
	Type string `json:"type"`
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
	fmt.Fprintf(w, "경고: 문서 종류가 %s 단계 기본값 %q 그대로다. --type으로 지정한다 (허용값: %s)\n",
		stage, cur, strings.Join(cfg.Schema.Types, ", "))
}

// printPromoted은 만들어진 경로와 다음에 할 수 있는 일을 낸다.
func printPromoted(w io.Writer, res promoteOutcome) {
	fmt.Fprintf(w, "context로 올렸다: %s\n", res.Path)
	fmt.Fprintf(w, "문서 종류: %s\n", res.Type)
	fmt.Fprintf(w, "게이트: 링크 %d개, 대상 %d개, 기준 %d개%s\n", res.Gate.Links, res.Gate.Targets, res.Gate.Min,
		deferredNote(res.Gate.Deferred))
	fmt.Fprintf(w, "다음: engram lint로 승급 문서의 스키마를 확인한다\n")
}

// deferredNote는 유예 안내 문구를 낸다.
func deferredNote(deferred bool) string {
	if deferred {
		return " (유예)"
	}
	return ""
}

// resolveDocPath는 인자로 받은 경로를 파일 경로로 확정한다.
// 파일 경로가 아니면 위키 루트 상대 경로로 해석한다.
func resolveDocPath(wikiRoot, arg string) (string, error) {
	for _, c := range []string{arg, filepath.Join(wikiRoot, arg)} {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("문서를 찾을 수 없다: %s\n위키 루트 상대 경로로 준다. 예: engram promote inbox/2026-01-01-메모.md", arg)
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
// 꺼진 축은 초기값에 애초에 없으므로 추가되지 않는다.
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
