// Package migrate는 기존 문서를 지금의 설정과 규칙에 맞춘다. ADR 0038.
//
// 고치는 것은 셋이다. 켜진 속성의 필수 필드를 단계별 초기값으로 채우고,
// 꺼진 속성의 필드를 지우고, 위치와 단계의 불일치를 프론트매터를 위치에
// 맞춰 고친다. 단계별 초기값과 단계-디렉토리 대응의 진실원은 internal/wiki
// 다. 값을 따로 정의하지 않고 그것을 부른다.
//
// 파일을 옮기지 않고 슬러그를 바꾸지 않는다. 이동은 promote, demote,
// archive, mv 의 일이다. 문서를 승급시키지도 않는다. inbox 디렉토리에
// 있으면서 context 라고 선언한 문서는 선언이 inbox 로 내려갈 뿐이고
// 파일은 제자리에 남는다. 대량 작업이 게이트를 우회하는 경로가 되면
// 게이트가 유일한 관문이라는 전제가 무너진다.
//
// 게이트 위반과 깨진 링크는 고치지 않는다. 어떤 문서에 이어야 하는지는
// 판단이므로 보고만 하고 promote 나 demote 를 안내한다.
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
)

// Options는 migrate 실행 옵션이다.
type Options struct {
	Apply bool // 변경을 파일에 쓴다. 기본은 시험 실행이다
	Force bool // 값이 있는 꺼진 속성 필드도 지운다
}

// 변경 종류다. JSON 에 그대로 나간다.
const (
	KindStage  = "stage"  // artifact_stage 를 위치에 맞춘다
	KindFill   = "fill"   // 켜진 속성의 필수 필드를 단계별 초기값로 채운다
	KindRemove = "remove" // 꺼진 속성의 필드를 지운다
)

// Change는 문서 하나에 가할 변경 하나다. 필드 이름과 옛값, 새값을 함께
// 낸다. 값은 displayValue 의 표시형을 따른다.
type Change struct {
	Kind  string `json:"kind"`
	Field string `json:"field"`
	Old   string `json:"old,omitempty"` // 옛값. 없었으면 빈 문자열
	New   string `json:"new,omitempty"` // 새값. 지우면 빈 문자열
}

// Remainder는 문서를 완전히 맞추지 못하고 남긴 필수 필드다. 채울 진실원이
// migrate 밖에 있는 필드다. 남긴 것이 있으면 그 문서는 맞춘 것으로 세지
// 않는다.
type Remainder struct {
	Field  string `json:"field"`
	Reason string `json:"reason"` // 채우지 못한 이유. 사용자가 읽는 문장이다
}

// DocResult는 문서 하나의 마이그레이션 결과다.
type DocResult struct {
	Path       string      `json:"path"`
	Changes    []Change    `json:"changes"`              // 적용했거나 적용할 변경
	Blocked    []Change    `json:"blocked,omitempty"`    // --force 없이는 못 지운 값 있는 필드
	Remainders []Remainder `json:"remainders,omitempty"` // 채우지 못해 남은 필수 필드
	Written    bool        `json:"written"`
}

// Advisory는 migrate가 고치지 않고 보고만 하는 항목이다. 게이트 위반과
// 깨진 링크가 여기 들어간다.
type Advisory struct {
	Path string `json:"path"`
	Rule string `json:"rule"`
}

// Report는 migrate 실행 결과다. Documents 는 변경, 보류, 남은 필드 중
// 하나라도 있는 문서만 실는다. 변화가 없는 문서까지 실으면 수백 줄이 된다.
type Report struct {
	Applied       bool        `json:"applied"`
	Docs          int         `json:"docs"`
	NonConforming int         `json:"nonConforming"`      // 규칙에 맞지 않은 문서 수. 변경, 보류, 남은 필드를 다 센다
	Partial       int         `json:"partial"`            // 채우지 못한 필드가 남은 문서 수
	Changed       int         `json:"changed"`            // 변경이 있는 문서 수
	Written       int         `json:"written"`            // 실제로 쓴 문서 수
	Blocked       int         `json:"blocked"`            // --force 없이 못 지운 변경 수
	Unparsed      []string    `json:"unparsed,omitempty"` // 프론트매터를 읽지 못한 문서
	Advisories    []Advisory  `json:"advisories"`
	Documents     []DocResult `json:"documents"`
}

// Run은 위키를 순회하며 문서를 지금의 설정과 규칙에 맞춘다. 시험 실행이
// 기본이므로 Options.Apply 없이는 파일을 쓰지 않는다. 값이 이미 맞으면
// 쓰지 않으므로 두 번 돌려도 두 번째는 바뀌는 문서가 없다.
func Run(wikiRoot string, cfg config.Config, opts Options) (Report, error) {
	walked, err := walk.Files(wikiRoot, cfg)
	if err != nil {
		return Report{}, fmt.Errorf("%s: %w", i18n.T("core.migrate.walk_fail"), err)
	}
	rep := Report{Applied: opts.Apply, Docs: len(walked), Advisories: []Advisory{}, Documents: []DocResult{}}
	for _, w := range walked {
		// 프론트매터가 없거나 깨진 문서는 필드를 다룰 수 없다. 조용히
		// 넘기지 않고 목록에 남긴다.
		if w.Err != nil || !w.Parsed.HasFrontmatter {
			rep.Unparsed = append(rep.Unparsed, w.Rel)
			continue
		}
		res, fields := planDoc(w, cfg, opts)
		if len(res.Changes) == 0 && len(res.Blocked) == 0 && len(res.Remainders) == 0 {
			continue
		}
		rep.NonConforming++
		if len(res.Remainders) > 0 {
			rep.Partial++
		}
		if opts.Apply && len(res.Changes) > 0 {
			path := filepath.Join(wikiRoot, filepath.FromSlash(w.Rel))
			if err := os.WriteFile(path, doc.Render(fields, w.Parsed.Body), 0o644); err != nil {
				return rep, fmt.Errorf("%s: %w", i18n.T("core.migrate.write_fail", path), err)
			}
			res.Written = true
			rep.Written++
		}
		if len(res.Changes) > 0 {
			rep.Changed++
		}
		rep.Blocked += len(res.Blocked)
		rep.Documents = append(rep.Documents, res)
	}

	// 고치지 않는 것은 상태를 읽어 보고한다. 판정의 진실원은 lint 다.
	// 적용 뒤의 파일 상태에서 읽으므로 시험 실행이면 현재 상태, 적용
	// 이후면 남은 상태가 나온다.
	lintRes, err := lint.Run(wikiRoot, cfg)
	if err != nil {
		return rep, fmt.Errorf("%s: %w", i18n.T("core.migrate.lint_fail"), err)
	}
	seen := map[Advisory]bool{}
	for _, v := range lintRes.Violations {
		if v.Rule != "gate.min-wikilinks" && v.Rule != "link.broken" {
			continue
		}
		a := Advisory{Path: v.Path, Rule: v.Rule}
		if !seen[a] {
			seen[a] = true
			rep.Advisories = append(rep.Advisories, a)
		}
	}
	return rep, nil
}

// planDoc는 문서 하나의 변경을 계산해 새 필드 목록과 함께 낸다. 계산만
// 하고 쓰지 않는다. Run 이 쓰기 여부를 정한다. 변경 순서는 단계, 채우기,
// 지우기 순으로 결정론적이다. 있는 키는 그 자리를 유지하고 없는 키는
// 끝에 추가하므로 기존 키 순서는 보존된다.
func planDoc(w walk.Doc, cfg config.Config, opts Options) (DocResult, []doc.Field) {
	d := w.Parsed
	fields := append([]doc.Field(nil), d.Fields...)
	res := DocResult{Path: w.Rel, Changes: []Change{}}
	declared := fieldStr(d, "artifact_stage")

	// 위치와 단계의 불일치. 프론트매터를 위치에 맞춘다. 파일을 옮기지
	// 않는다. 색인(root_files)은 위키 루트에 있어 비교할 디렉토리가 없다.
	// 단계에 대응하지 않는 디렉토리도 비교 기준이 없다. 선언이 비었거나
	// 허용값 밖이어도 위치가 진실원이므로 위치 단계로 쓴다.
	effective := declared
	if !w.Root {
		top, _, _ := strings.Cut(w.Rel, "/")
		if expected, ok := wiki.StageForDir(top); ok && declared != string(expected) {
			stageField := doc.Field{Key: "artifact_stage", Kind: doc.KindString, Str: string(expected)}
			if i, ok := indexOfField(fields, "artifact_stage"); ok {
				fields[i] = stageField
			} else {
				fields = append(fields, stageField)
			}
			res.Changes = append(res.Changes, Change{
				Kind: KindStage, Field: "artifact_stage",
				Old: declared, New: string(expected),
			})
			effective = string(expected)
		}
	}

	// 켜진 속성의 필수 필드를 단계별 초기값으로 채운다. 단계를 고쳤으면
	// 고친 단계의 초기값을 쓴다. 그래야 적용 뒤 lint 의 필수 필드 판정과
	// 어긋나지 않는다.
	if stage, ok := wikiStage(effective); ok {
		defaults := wiki.Frontmatter(stage, cfg)
		for _, key := range sortedKeys(defaults) {
			if _, ok := indexOfField(fields, key); ok {
				continue
			}
			f := fieldOf(key, defaults[key])
			fields = append(fields, f)
			res.Changes = append(res.Changes, Change{Kind: KindFill, Field: key, New: displayValue(f)})
		}

		// source 단계의 날짜 필수 필드는 속성이 아니므로 초기값에 없다.
		// created 는 파일명의 날짜 접두사에서 뽑는다. promote 의
		// fillCreated 와 같은 방식이다. sourced_at 은 채우지 않는다.
		// git 최초 커밋일이 진실원이고 그것은 sync 의 일이다(ADR 0037).
		// 못 채운 것은 남긴 것으로 보고한다. 조용히 넘기면 lint 는
		// 위반이라 하고 migrate 는 맞았다고 하는 어긋남이 생긴다.
		if stage == wiki.StageSource {
			if _, ok := indexOfField(fields, "created"); !ok {
				if prefix := datePrefix(filepath.Base(w.Rel)); prefix != "" {
					fields = append(fields, doc.Field{Key: "created", Kind: doc.KindDate, Str: prefix})
					res.Changes = append(res.Changes, Change{Kind: KindFill, Field: "created", New: prefix})
				} else {
					res.Remainders = append(res.Remainders, Remainder{Field: "created", Reason: i18n.T("core.migrate.reason_created_no_prefix")})
				}
			}
			if _, ok := indexOfField(fields, "sourced_at"); !ok {
				res.Remainders = append(res.Remainders, Remainder{Field: "sourced_at", Reason: i18n.T("core.migrate.reason_sourced_at")})
			}
		}
	}

	// 꺼진 속성의 필드를 지운다. 값이 비어 있으면 잃을 것이 없으므로 그대로
	// 지운다. 값이 있으면 --force 가 필요하다. 보류해도 나머지 변경은
	// 그대로 적용한다. 전체를 멈추지 않는다.
	for _, ax := range axisNames() {
		if cfg.Axes[config.Axis(ax)] {
			continue
		}
		i, ok := indexOfField(fields, ax)
		if !ok {
			continue
		}
		ch := Change{Kind: KindRemove, Field: ax, Old: displayValue(fields[i])}
		if valueEmpty(fields[i]) || opts.Force {
			fields = append(fields[:i], fields[i+1:]...)
			res.Changes = append(res.Changes, ch)
		} else {
			res.Blocked = append(res.Blocked, ch)
		}
	}
	return res, fields
}

// axisNames는 스키마가 다루는 속성 14종이다. 설정의 Axes 맵은 켜진 속성만
// 담으므로 꺼진 속성을 찾으려면 전체 목록이 필요하다. config 이 목록을
// 내보내지 않아 lint 과 cli 와 같은 방식으로 여기에 둔다.
func axisNames() []string {
	return []string{
		string(config.AxisType), string(config.AxisArtifactStage), string(config.AxisStatus),
		string(config.AxisIndexable), string(config.AxisTags), string(config.AxisSourceRefs),
		string(config.AxisDerivedFrom), string(config.AxisRelated), string(config.AxisSourceChannel),
		string(config.AxisDerivedContext), string(config.AxisScope), string(config.AxisSensitivity),
		string(config.AxisTriggerMode), string(config.AxisWorkflow),
	}
}

// wikiStage는 문자열 단계 값을 wiki.Stage 로 변환한다. 모르는 값이면
// false 다.
func wikiStage(s string) (wiki.Stage, bool) {
	switch wiki.Stage(s) {
	case wiki.StageInbox, wiki.StageSource, wiki.StageContext, wiki.StageArchive:
		return wiki.Stage(s), true
	}
	return "", false
}

// datePrefix는 파일명 앞의 날짜 접두사를 반환한다. 없으면 빈 문자열이다.
// internal/cli/promote.go 의 같은 이름 함수와 같은 규칙이므로 한쪽을 고치면
// 다른 쪽도 함께 고친다.
func datePrefix(name string) string {
	m := regexp.MustCompile(`^(\d{4}-\d{2}(-\d{2})?)-`).FindStringSubmatch(strings.TrimSuffix(name, ".md"))
	if m == nil {
		return ""
	}
	return m[1]
}

// sortedKeys는 맵의 키를 정렬해 낸다. 맵 순회는 무작위이므로 변경 순서의
// 결정론을 위해 정렬한다.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// fieldOf는 초기값 맵의 값을 필드로 바꾼다. 값의 종류는 wiki.Frontmatter
// 가 내는 문자열, 불리언, nil, []string 넷뿐이다.
func fieldOf(key string, v any) doc.Field {
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

// fieldStr은 문서의 문자열 필드 값을 반환한다.
func fieldStr(d doc.Doc, key string) string {
	for _, f := range d.Fields {
		if f.Key == key && f.Kind == doc.KindString {
			return f.Str
		}
	}
	return ""
}

// indexOfField는 필드 목록에서 키의 위치를 찾는다.
func indexOfField(fields []doc.Field, key string) (int, bool) {
	for i, f := range fields {
		if f.Key == key {
			return i, true
		}
	}
	return 0, false
}

// valueEmpty는 필드에 값이 없는지 검사한다. 빈 키, 빈 문자열, 빈 목록은
// 값이 없다. 불리언은 값이 있다. 거짓도 정보이기 때문이다.
func valueEmpty(f doc.Field) bool {
	switch f.Kind {
	case doc.KindEmpty:
		return true
	case doc.KindString, doc.KindDate:
		return f.Str == ""
	case doc.KindStringList:
		return len(f.List) == 0
	}
	return false
}

// displayValue는 필드 값을 사람이 읽는 표시형으로 낸다. 문자열 목록은
// 대괄호 묶음 쉼표 나열, 불리언은 true/false, 빈 키는 빈 문자열이다.
func displayValue(f doc.Field) string {
	switch f.Kind {
	case doc.KindBool:
		if f.Bool {
			return "true"
		}
		return "false"
	case doc.KindStringList:
		return "[" + strings.Join(f.List, ", ") + "]"
	default:
		return f.Str
	}
}
