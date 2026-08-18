package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/spf13/cobra"
)

// update 커맨드의 플래그 이름.
const (
	flagSetKey   = "set"
	flagUnsetKey = "unset"
	flagBodyFrom = "body-from"
	// flagUpdateForce는 sources 원본 보존 거절을 넘기는 플래그 이름이다(ADR 0064).
	flagUpdateForce = "force"
)

// newUpdateCmd는 문서의 프론트매터와 본문을 갱신하는 update 커맨드를 반환한다.
// 슬러그와 파일명은 바꾸지 않는다. 단계 이동은 promote와 demote의 일이다.
func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update " + i18n.T("usage.args.path"),
		Short: i18n.T("cli.update.short"),
		Long:  i18n.T("cli.update.long"),
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
			sets, err := cmd.Flags().GetStringArray(flagSetKey)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagSetKey), err)
			}
			unsets, err := cmd.Flags().GetStringArray(flagUnsetKey)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagUnsetKey), err)
			}
			bodyFrom, err := stringFlag(cmd, flagBodyFrom)
			if err != nil {
				return err
			}
			if len(sets) == 0 && len(unsets) == 0 && bodyFrom == "" {
				return errors.New(i18n.T("cli.update.no_changes"))
			}
			force, err := cmd.Flags().GetBool(flagUpdateForce)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.flag_read_fail", flagUpdateForce), err)
			}
			// sources는 원본 보존 계층이다(ADR 0064). 모르고 고치는 것을
			// 막는 것이 목적이므로 --force를 준 경우는 통과시킨다.
			inSources := stageOfPath(root, srcPath) == "sources"
			if inSources && !force {
				return errors.New(i18n.T("cli.update.sources_refused"))
			}

			raw, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_read_fail", srcPath), err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.ingest.doc_parse_fail", srcPath), err)
			}

			fields := append([]doc.Field(nil), d.Fields...)
			for _, kv := range sets {
				key, value, ok := strings.Cut(kv, "=")
				if !ok {
					return errors.New(i18n.T("cli.update.set_invalid", kv))
				}
				f, err := settableField(key, value, cfg)
				if err != nil {
					return err
				}
				fields = upsertField(fields, f)
			}
			for _, key := range unsets {
				if key == "artifact_stage" {
					return errors.New(i18n.T("cli.update.unset_stage_forbidden"))
				}
				if axisOff(key, cfg) {
					return errors.New(i18n.T("cli.update.unset_axis_off", key))
				}
				fields = removeField(fields, key)
			}

			body := d.Body
			bodyChanged := false
			if bodyFrom != "" {
				content, err := readBody(cmd.InOrStdin(), bodyFrom)
				if err != nil {
					return err
				}
				body = content
				bodyChanged = true
			}

			// updated는 도구가 채우는 필드다(ADR 0009). 갱신 사실을 날짜로
			// 남겨야 재발견 루프가 노후를 올바르게 판정한다. sources 계층과
			// 사용자가 updated를 직접 정하거나 지운 경우는 채우지 않는다.
			autoUpdated := ""
			if !inSources &&
				!hasSetKey(sets, "updated") && !containsString(unsets, "updated") {
				autoUpdated = Now(cmd).Format("2006-01-02")
				fields = upsertField(fields, doc.Field{Key: "updated", Kind: doc.KindDate, Str: autoUpdated})
			}

			if err := os.WriteFile(srcPath, doc.Render(fields, body), 0o644); err != nil {
				return fmt.Errorf("%s: %w", i18n.T("cli.update.write_fail", srcPath), err)
			}

			res := updateOutcome{Path: srcPath, Set: sets, Unset: unsets, Updated: autoUpdated, ForcedSources: inSources}
			if bodyChanged {
				res.BodyFrom = bodyFrom
			}
			if jsonOutput(cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}
			printUpdated(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringArray(flagSetKey, nil, i18n.T("cli.update.flag_set"))
	cmd.Flags().StringArray(flagUnsetKey, nil, i18n.T("cli.update.flag_unset"))
	cmd.Flags().String(flagBodyFrom, "", i18n.T("cli.update.flag_body_from"))
	cmd.Flags().Bool(flagUpdateForce, false, i18n.T("cli.update.flag_force"))
	cmd.Flags().String(flagWiki, ".", i18n.T("cli.ingest.flag_wiki"))
	return cmd
}

// updateOutcome은 update의 결과다. Updated는 도구가 updated 필드를
// 자동으로 채웠을 때만 값이 있다. ForcedSources는 원본 보존 계층을
// --force로 고친 경우다(ADR 0064).
type updateOutcome struct {
	Path          string   `json:"path"`
	Set           []string `json:"set"`
	Unset         []string `json:"unset"`
	BodyFrom      string   `json:"bodyFrom,omitempty"`
	Updated       string   `json:"updated,omitempty"`
	ForcedSources bool     `json:"forcedSources,omitempty"`
}

// hasSetKey는 --set 목록에 키가 이미 있는지 본다. 사용자가 직접 정한
// 값은 자동 갱신이 덮지 않는다.
func hasSetKey(sets []string, key string) bool {
	for _, kv := range sets {
		if k, _, _ := strings.Cut(kv, "="); k == key {
			return true
		}
	}
	return false
}

// arrayFields는 쉼표로 여러 값을 받는 필드다.
func arrayFields() []string {
	return []string{"tags", "topics", "source_refs", "derived_from", "derived_context", "related"}
}

// knownAxes는 스키마가 다루는 속성 이름 14종이다. 속성 on/off 판정에 쓴다.
// internal/config 이 목록을 내보내지 않아 여기에 둔다.
func knownAxes() []config.Axis {
	return []config.Axis{
		config.AxisType, config.AxisArtifactStage, config.AxisStatus,
		config.AxisIndexable, config.AxisTags, config.AxisSourceRefs,
		config.AxisDerivedFrom, config.AxisRelated, config.AxisSourceChannel,
		config.AxisDerivedContext, config.AxisScope, config.AxisSensitivity,
		config.AxisTriggerMode, config.AxisWorkflow,
	}
}

// axisOff는 키가 속성 이름이면서 설정에서 꺼져 있는지를 반환한다.
// 속성 이름이 아닌 키는 여기서 가리지 않는다.
func axisOff(key string, cfg config.Config) bool {
	for _, ax := range knownAxes() {
		if string(ax) == key {
			return !cfg.Axes[ax]
		}
	}
	return false
}

// allowedFor는 문자열 필드의 허용값 집합을 반환한다. 두 번째 반환값이
// 거짓이면 허용값 검사 대상이 아니다(개방 집합).
func allowedFor(key string, cfg config.Config) ([]string, bool) {
	switch key {
	case "type":
		return cfg.Schema.Types, true
	case "status":
		return cfg.Schema.Statuses.Values, true
	case "scope":
		return cfg.Schema.Scopes.Values, true
	case "sensitivity":
		return cfg.Schema.Sensitivities.Values, true
	case "trigger_mode":
		return cfg.Schema.TriggerModes.Values, true
	case "form":
		if len(cfg.Schema.Taxonomy.Forms.Values) > 0 {
			return cfg.Schema.Taxonomy.Forms.Values, true
		}
	}
	return nil, false
}

// settableField는 --set 하나를 검증해 필드로 만든다.
func settableField(key, value string, cfg config.Config) (doc.Field, error) {
	if key == "artifact_stage" {
		return doc.Field{}, errors.New(i18n.T("cli.update.set_stage_forbidden"))
	}
	if axisOff(key, cfg) {
		return doc.Field{}, errors.New(i18n.T("cli.update.set_axis_off", key, cfg.Preset))
	}
	for _, f := range arrayFields() {
		if f != key {
			continue
		}
		var list []string
		if value != "" {
			for _, item := range strings.Split(value, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					return doc.Field{}, errors.New(i18n.T("cli.update.set_empty_item", key, value))
				}
				if key == "related" {
					item = linkValue(item)
				}
				list = append(list, item)
			}
		}
		// 빈 값은 빈 목록으로 둔다. 값을 비우는 표현이다.
		return doc.Field{Key: key, Kind: doc.KindStringList, List: list}, nil
	}
	if key == "indexable" {
		switch value {
		case "true", "false":
			return doc.Field{Key: key, Kind: doc.KindBool, Bool: value == "true"}, nil
		}
		return doc.Field{}, errors.New(i18n.T("cli.update.set_not_bool", key, value))
	}
	if allowed, ok := allowedFor(key, cfg); ok && value != "" {
		if !containsString(allowed, value) {
			return doc.Field{}, errors.New(i18n.T("cli.update.set_not_allowed",
				key, value, strings.Join(allowed, ", ")))
		}
	}
	if value == "" {
		// 값이 없는 키는 정상 표현이다. upstream 계약이 이렇게 쓴다.
		return doc.Field{Key: key, Kind: doc.KindEmpty}, nil
	}
	return doc.Field{Key: key, Kind: doc.KindString, Str: value}, nil
}

// removeField는 키를 지운다. 없으면 그대로 둔다.
func removeField(fields []doc.Field, key string) []doc.Field {
	out := make([]doc.Field, 0, len(fields))
	for _, f := range fields {
		if f.Key != key {
			out = append(out, f)
		}
	}
	return out
}

// readBody는 --body-from 값을 읽는다. - 이면 표준 입력을 읽는다.
func readBody(cmd io.Reader, from string) (string, error) {
	if from == "-" {
		data, err := io.ReadAll(cmd)
		if err != nil {
			return "", fmt.Errorf("%s: %w", i18n.T("cli.ingest.stdin_read_fail"), err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T("cli.update.body_read_fail", from), err)
	}
	return string(data), nil
}

// stageOfPath는 위키 루트 안 경로의 첫 디렉토리로 계층을 잡는다.
func stageOfPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	seg, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
	return seg
}

// printUpdated은 갱신 내용과 다음에 할 수 있는 일을 낸다.
func printUpdated(w io.Writer, res updateOutcome) {
	fmt.Fprintln(w, i18n.T("cli.update.done", res.Path))
	if res.ForcedSources {
		fmt.Fprintln(w, i18n.T("cli.update.forced_sources"))
	}
	for _, kv := range res.Set {
		fmt.Fprintln(w, i18n.T("cli.update.set_line", kv))
	}
	for _, key := range res.Unset {
		fmt.Fprintln(w, i18n.T("cli.update.unset_line", key))
	}
	if res.BodyFrom != "" {
		fmt.Fprintln(w, i18n.T("cli.update.body_line", res.BodyFrom))
	}
	if res.Updated != "" {
		fmt.Fprintln(w, i18n.T("cli.update.updated_line", res.Updated))
	}
	fmt.Fprintln(w, i18n.T("cli.update.next"))
}
