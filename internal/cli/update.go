package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/spf13/cobra"
)

// update 커맨드의 플래그 이름.
const (
	flagSetKey   = "set"
	flagUnsetKey = "unset"
	flagBodyFrom = "body-from"
)

// newUpdateCmd는 문서의 프론트매터와 본문을 갱신하는 update 커맨드를 반환한다.
// 슬러그와 파일명은 바꾸지 않는다. 단계 이동은 promote와 demote의 일이다.
func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <경로>",
		Short: "문서의 프론트매터와 본문을 갱신합니다",
		Long: `문서의 프론트매터 키와 본문을 갱신합니다.

--set key=value 로 키를 설정합니다. 반복할 수 있습니다. 배열 속성은 쉼표로
여러 값을 줍니다. 예: --set topics=go,cli
--unset key 로 키를 지웁니다. 반복할 수 있습니다.
--body-from <파일> 로 본문을 통째로 바꿉니다. - 를 주면 표준 입력을 읽습니다.

꺼진 속성의 키와 허용값 밖의 값은 거절합니다. artifact_stage는 여기서
바꾸지 못합니다. 단계 이동은 engram promote와 engram demote의 일입니다.
키 순서는 파싱이 보존한 그대로 유지됩니다.`,
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
			sets, err := cmd.Flags().GetStringArray(flagSetKey)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagSetKey, err)
			}
			unsets, err := cmd.Flags().GetStringArray(flagUnsetKey)
			if err != nil {
				return fmt.Errorf("--%s 플래그를 읽을 수 없음: %w", flagUnsetKey, err)
			}
			bodyFrom, err := stringFlag(cmd, flagBodyFrom)
			if err != nil {
				return err
			}
			if len(sets) == 0 && len(unsets) == 0 && bodyFrom == "" {
				return fmt.Errorf("갱신할 내용이 없습니다\n--set key=value, --unset key, --body-from <파일|-> 중 하나를 쓰세요")
			}

			raw, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("문서를 읽을 수 없음: %s: %w", srcPath, err)
			}
			d, err := doc.Parse(args[0], raw)
			if err != nil {
				return fmt.Errorf("문서를 파싱할 수 없음: %s: %w", srcPath, err)
			}

			fields := append([]doc.Field(nil), d.Fields...)
			for _, kv := range sets {
				key, value, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("--set 값이 key=value 형식이 아닙니다: %q", kv)
				}
				f, err := settableField(key, value, cfg)
				if err != nil {
					return err
				}
				fields = upsertField(fields, f)
			}
			for _, key := range unsets {
				if key == "artifact_stage" {
					return fmt.Errorf("artifact_stage는 update로 지울 수 없습니다. 단계 이동은 engram promote와 engram demote가 합니다")
				}
				if axisOff(key, cfg) {
					return fmt.Errorf("꺼진 속성의 키는 지울 것도 없습니다: %s", key)
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
				if rel := stageOfPath(root, srcPath); rel == "sources" {
					fmt.Fprintf(cmd.ErrOrStderr(), "경고: sources는 원본 보존 계층입니다. 본문을 바꾼 사실을 기억해 두세요\n")
				}
			}

			// updated는 도구가 채우는 필드다(ADR 0009). 갱신 사실을 날짜로
			// 남겨야 재발견 루프가 노후를 올바르게 판정한다. sources 계층과
			// 사용자가 updated를 직접 정하거나 지운 경우는 채우지 않는다.
			autoUpdated := ""
			if stageOfPath(root, srcPath) != "sources" &&
				!hasSetKey(sets, "updated") && !containsString(unsets, "updated") {
				autoUpdated = Now(cmd).Format("2006-01-02")
				fields = upsertField(fields, doc.Field{Key: "updated", Kind: doc.KindDate, Str: autoUpdated})
			}

			if err := os.WriteFile(srcPath, doc.Render(fields, body), 0o644); err != nil {
				return fmt.Errorf("문서를 쓸 수 없음: %s: %w", srcPath, err)
			}

			res := updateOutcome{Path: srcPath, Set: sets, Unset: unsets, Updated: autoUpdated}
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
	cmd.Flags().StringArray(flagSetKey, nil, "프론트매터 키 설정. key=value. 반복 가능")
	cmd.Flags().StringArray(flagUnsetKey, nil, "프론트매터 키 제거. 반복 가능")
	cmd.Flags().String(flagBodyFrom, "", "본문을 통째로 바꿉니다. 파일 경로 또는 - (표준 입력)")
	cmd.Flags().String(flagWiki, ".", "대상 위키 경로")
	return cmd
}

// updateOutcome은 update의 결과다. Updated는 도구가 updated 필드를
// 자동으로 채웠을 때만 값이 있다.
type updateOutcome struct {
	Path     string   `json:"path"`
	Set      []string `json:"set"`
	Unset    []string `json:"unset"`
	BodyFrom string   `json:"bodyFrom,omitempty"`
	Updated  string   `json:"updated,omitempty"`
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
		return doc.Field{}, fmt.Errorf("artifact_stage는 update로 바꿀 수 없습니다. 단계 이동은 engram promote와 engram demote가 합니다")
	}
	if axisOff(key, cfg) {
		return doc.Field{}, fmt.Errorf("꺼진 속성의 키는 설정할 수 없습니다: %s (프리셋 %s). engram.yaml의 axes에서 켜거나 문서에서 쓰지 않습니다", key, cfg.Preset)
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
					return doc.Field{}, fmt.Errorf("--set %s=%s 값에 빈 항목이 있습니다", key, value)
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
		return doc.Field{}, fmt.Errorf("--set %s=%s 값이 참/거짓이 아닙니다 (true 또는 false)", key, value)
	}
	if allowed, ok := allowedFor(key, cfg); ok && value != "" {
		if !containsString(allowed, value) {
			return doc.Field{}, fmt.Errorf("--set %s=%s 값이 허용값 밖입니다 (허용값: %s)", key, value, strings.Join(allowed, ", "))
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
			return "", fmt.Errorf("표준 입력을 읽을 수 없음: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return "", fmt.Errorf("본문 파일을 읽을 수 없음: %s: %w", from, err)
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
	fmt.Fprintf(w, "갱신했습니다: %s\n", res.Path)
	for _, kv := range res.Set {
		fmt.Fprintf(w, "설정: %s\n", kv)
	}
	for _, key := range res.Unset {
		fmt.Fprintf(w, "제거: %s\n", key)
	}
	if res.BodyFrom != "" {
		fmt.Fprintf(w, "본문 교체: %s\n", res.BodyFrom)
	}
	if res.Updated != "" {
		fmt.Fprintf(w, "updated에 갱신 날짜 %s를 기록했습니다\n", res.Updated)
	}
	fmt.Fprintf(w, "다음: engram lint로 갱신된 문서의 스키마를 확인하세요\n")
}
