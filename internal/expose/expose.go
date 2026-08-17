// Package expose는 위키를 밖으로 내보낼 때 무엇이 나가고 무엇이 빠지는지
// 판정한다.
//
// 규칙은 ADR 0044가 정한다. context 와 root_files 만 나가고 inbox 와
// sources 는 나가지 않는다. archive 는 기본으로 빠지고 옵션으로 연다.
// sensitivity 축이 켜진 위키에서는 private-local-only 와 restricted 를
// 뺀다. 그 제외를 뒤집는 수단은 없다.
//
// 이 판정을 serve 와 pack 이 함께 부른다. 두 벌 두면 serve 가 감추는
// 문서를 pack 이 내보내는 상태가 되고 민감도 선언이 무의미해진다(ADR 0046).
package expose

import (
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/walk"
	"github.com/neocode24/engram/internal/wiki"
)

// HiddenSensitivities는 밖으로 내보내지 않는 민감도 값이다.
// private-local-only 는 이름 그대로 로컬 전용이고 restricted 는 제한
// 공개다. 값을 붙인 사람의 판단을 도구가 덮지 않으므로 이 목록을 뒤집는
// 플래그는 없다(ADR 0044).
var HiddenSensitivities = []string{"private-local-only", "restricted"}

// 문서의 위치 구분이다. 노출 판정의 입력이며 표시 묶음의 이름이기도 하다.
const (
	LocIndex   = "index"   // root_files 문서
	LocContext = "context" // 검수를 지난 문서
	LocArchive = "archive" // 수명이 끝난 문서
	LocInbox   = "inbox"   // 미검수 러프 캡처
	LocSources = "sources" // 원본 보존 계층
	LocOutside = "outside" // 네 단계 어디에도 속하지 않는 위치
)

// 제외 사유다. 무엇이 왜 빠졌는지 집계해 사용자에게 알리는 데 쓴다.
const (
	ReasonNone        = ""
	ReasonInbox       = "inbox"
	ReasonSources     = "sources"
	ReasonArchive     = "archive"
	ReasonSensitivity = "sensitivity"
	ReasonUnparsed    = "unparsed"
	ReasonOutside     = "outside"
)

// Options는 노출 범위를 넓히는 선택지다. 넓히는 쪽만 있고 좁히는 쪽은
// 기본값이다. 기본이 전체 노출이면 사고가 나기 때문이다.
type Options struct {
	// IncludeArchive는 archive 문서를 노출할지 정한다. 기본은 거짓이다.
	IncludeArchive bool
}

// Exposure는 무엇이 노출되고 무엇이 제외되는지의 집계다. 사용자에게
// 알리는 것이 목적이므로 수치만 담고 경로는 담지 않는다.
type Exposure struct {
	Context int // 노출된 context 문서 수
	Index   int // 노출된 root_files 문서 수
	Archive int // 노출된 archive 문서 수

	ExcludedInbox     int
	ExcludedSources   int
	ExcludedArchive   int
	ExcludedSensitive int
	ExcludedUnparsed  int
	ExcludedOutside   int

	// SensitivityOn은 이 위키에서 sensitivity 축이 켜져 있는지다.
	// 꺼진 위키에는 거를 값이 없으므로 민감도 제외가 걸리지 않는다.
	SensitivityOn bool
	// IncludeArchive는 archive 를 열었는지다.
	IncludeArchive bool
}

// Visible은 노출 문서 수 합계다.
func (e Exposure) Visible() int {
	return e.Context + e.Index + e.Archive
}

// Doc은 노출 판정을 마친 문서 하나다. 순회 결과에 위치 구분을 얹는다.
type Doc struct {
	walk.Doc
	Loc string
}

// Result는 노출 판정의 결과다.
type Result struct {
	// Docs는 노출 대상이다. 순회 순서를 유지하므로 경로순이다.
	Docs []Doc
	// Walked는 순회 원본 전체다. 제외된 문서도 들어 있다. 링크 그래프와
	// 색인 신선도 판정은 위키 전체를 봐야 하므로 여기에 남긴다.
	Walked []walk.Doc
	// Exposure는 노출과 제외의 집계다.
	Exposure Exposure
}

// Select는 위키를 순회해 무엇이 나가는지 판정한다.
func Select(root string, cfg config.Config, opts Options) (Result, error) {
	walked, err := walk.Files(root, cfg)
	if err != nil {
		return Result{}, fmt.Errorf("위키를 순회할 수 없음: %w", err)
	}
	res := Result{Walked: walked}
	res.Exposure.SensitivityOn = cfg.Axes[config.AxisSensitivity]
	res.Exposure.IncludeArchive = opts.IncludeArchive

	for _, wd := range walked {
		loc := Locate(wd)
		switch reason := Exclude(wd, loc, cfg, opts); reason {
		case ReasonNone:
			res.Docs = append(res.Docs, Doc{Doc: wd, Loc: loc})
			switch loc {
			case LocIndex:
				res.Exposure.Index++
			case LocArchive:
				res.Exposure.Archive++
			default:
				res.Exposure.Context++
			}
		case ReasonInbox:
			res.Exposure.ExcludedInbox++
		case ReasonSources:
			res.Exposure.ExcludedSources++
		case ReasonArchive:
			res.Exposure.ExcludedArchive++
		case ReasonSensitivity:
			res.Exposure.ExcludedSensitive++
		case ReasonUnparsed:
			res.Exposure.ExcludedUnparsed++
		case ReasonOutside:
			res.Exposure.ExcludedOutside++
		}
	}
	return res, nil
}

// Locate는 문서가 어느 위치에 있는지 판정한다. 단계와 디렉토리의 대응은
// internal/wiki 의 표가 진실원이므로 여기서 이름을 다시 적지 않는다.
func Locate(wd walk.Doc) string {
	if wd.Root {
		return LocIndex
	}
	seg, _, _ := strings.Cut(wd.Rel, "/")
	st, ok := wiki.StageForDir(seg)
	if !ok {
		return LocOutside
	}
	switch st {
	case wiki.StageContext:
		return LocContext
	case wiki.StageArchive:
		return LocArchive
	case wiki.StageInbox:
		return LocInbox
	case wiki.StageSource:
		return LocSources
	}
	return LocOutside
}

// Exclude는 문서를 감출 이유를 판정한다. 빈 문자열이면 노출한다.
// 판정은 허용 목록이다. 아는 위치만 열고 모르는 위치는 닫는다. 설정이
// page_dirs 를 늘렸을 때 새 디렉토리가 조용히 노출되면 안 되기 때문이다.
func Exclude(wd walk.Doc, loc string, cfg config.Config, opts Options) string {
	switch loc {
	case LocInbox:
		return ReasonInbox
	case LocSources:
		return ReasonSources
	case LocOutside:
		return ReasonOutside
	case LocArchive:
		if !opts.IncludeArchive {
			return ReasonArchive
		}
	}
	// 프론트매터를 읽을 수 없는 문서는 민감도를 판정할 수 없다.
	// 판정하지 못하는 것은 노출하지 않는다.
	if wd.Err != nil {
		return ReasonUnparsed
	}
	if cfg.Axes[config.AxisSensitivity] && Hidden(SensitivityOf(wd.Parsed)) {
		return ReasonSensitivity
	}
	return ReasonNone
}

// Hidden은 민감도 값이 노출 금지 값인지 본다.
func Hidden(v string) bool {
	for _, h := range HiddenSensitivities {
		if v == h {
			return true
		}
	}
	return false
}

// SensitivityOf는 프론트매터의 sensitivity 값을 꺼낸다. 없으면 빈 문자열이다.
func SensitivityOf(d doc.Doc) string {
	for _, f := range d.Fields {
		if f.Key == "sensitivity" {
			return strings.TrimSpace(f.Str)
		}
	}
	return ""
}

// ReasonText는 제외 사유를 사용자에게 낼 문장으로 바꾼다. 커맨드가
// 제외된 문서를 지목당했을 때 왜 안 나가는지 알리는 데 쓴다.
func ReasonText(reason string) string {
	switch reason {
	case ReasonInbox:
		return "inbox 문서는 검수를 지나지 않아 반출하지 않습니다"
	case ReasonSources:
		return "sources 문서는 원본 보존 계층이라 반출하지 않습니다"
	case ReasonArchive:
		return "archive 문서입니다. --include-archive 로 엽니다"
	case ReasonSensitivity:
		return "민감도가 " + strings.Join(HiddenSensitivities, " 또는 ") + " 입니다. 반출하려면 문서의 값을 고치세요"
	case ReasonUnparsed:
		return "프론트매터를 읽을 수 없어 민감도를 판정하지 못했습니다 (engram lint 로 확인하세요)"
	case ReasonOutside:
		return "단계 디렉토리 밖에 있는 문서입니다"
	}
	return "노출 대상이 아닙니다"
}
