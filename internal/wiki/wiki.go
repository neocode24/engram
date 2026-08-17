// Package wiki는 문서를 위키의 올바른 자리에 만드는 쓰기 계층이다.
// 커맨드가 아니라 capture, source, promote, new 같은 상위 계층이 부르는
// 순수 함수 모음이다. 이 패키지는 시계를 읽지 않는다. 날짜는 전부
// 호출자가 인자로 준다. 결정론이 ADR 0005의 골든 비교 전제다.
package wiki

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
)

// Stage는 문서가 놓이는 단계다. 단계에 대응하는 디렉토리는 stageDirs
// 표가 정한다.
type Stage string

const (
	StageInbox   Stage = "inbox"
	StageSource  Stage = "source"
	StageContext Stage = "context"
	StageArchive Stage = "archive"
)

// stageDirs는 단계와 디렉토리 이름의 대응표다. 대부분 단계 이름과
// 디렉토리 이름이 같고 source 단계의 원본 보존 계층만 복수형 sources가
// 관례다. 이 표가 대응의 단일 진실원이므로 같은 지식을 다른 곳에
// 두지 않는다(ADR 0031).
var stageDirs = map[Stage]string{
	StageInbox:   "inbox",
	StageSource:  "sources",
	StageContext: "context",
	StageArchive: "archive",
}

// unsafeFilenameChars는 파일명에 쓸 수 없는 문자다. Windows가 tier 1
// 플랫폼이므로(ADR 0007) Windows 예약 문자를 포함한다.
const unsafeFilenameChars = `<>:"/\|?*`

// unsafeFilenameRune은 파일명에 쓸 수 없는 문자인지 판정한다. 예약 문자와
// 제어 문자가 대상이다. 파생 경로는 이 문자를 지우고 명시 경로는 거절하지만
// 무엇이 위험한지의 판정은 이 함수 하나가 한다(ADR 0045). 판정을 두 벌
// 두면 한쪽만 고치는 사고가 난다.
func unsafeFilenameRune(r rune) bool {
	return r < 0x20 || r == 0x7f || strings.ContainsRune(unsafeFilenameChars, r)
}

// Slug는 제목에서 파일명으로 쓸 슬러그를 만든다.
// 공백과 구분자를 하이픈 하나로 바꾸고 연속 하이픈을 줄이며 앞뒤 하이픈을
// 없앤다. 파일시스템에서 문제가 되는 문자는 지운다. Windows가 tier 1
// 플랫폼이므로(ADR 0007) 예약 문자와 예약 파일명을 포함한다.
// 한글은 그대로 보존한다. 이 바이너리에는 LLM이 없으므로(ADR 0014)
// 기계적 음차나 번역을 하지 않는다. upstream 실제 파일명에 한글이
// 섞여 있으므로 허용이 계약이다.
//
// 만든 결과는 ValidateSlug를 통과한다. 명시 경로가 거절하는 이름을 파생
// 경로가 만들어내면 같은 규칙이 두 얼굴을 갖게 되므로 마지막에 같은
// 검사를 받는다.
func Slug(title string) (string, error) {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if unsafeFilenameRune(r) {
			return -1
		}
		return r
	}, s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '_' {
			return '-'
		}
		return r
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	// 앞뒤의 하이픈과 점을 없앤다. Windows는 이름 끝의 점을 지우고 앞의
	// 점은 숨김 파일을 만들어 walk가 문서를 지나친다.
	s = strings.Trim(s, "-.")
	if s == "" {
		return "", fmt.Errorf("제목 %q에서 슬러그를 만들 수 없습니다. 제목 대신 슬러그를 직접 지정하세요", title)
	}
	if err := ValidateSlug(s); err != nil {
		return "", fmt.Errorf("제목 %q에서 만든 슬러그를 파일명으로 쓸 수 없습니다: %w", title, err)
	}
	return s, nil
}

// ValidateSlug는 슬러그를 파일명으로 쓸 수 있는지 검사한다. 사용자가 직접
// 지정한 슬러그가 주 대상이다(ADR 0045).
//
// 파생 경로는 위험한 문자를 지우지만 명시한 값은 조용히 고치지 않고
// 거절한다. 사용자가 이름을 명시한 것은 그 이름을 원한다는 뜻이므로
// 도구가 말없이 다른 이름으로 바꾸면 사용자가 기대한 위키링크가 어긋난다.
//
// 소문자와 하이픈 정규화는 강제하지 않는다. 대문자, 공백, 한글은 파일을
// 못 만들게 하지 않으므로 통과시킨다. ADR 0020의 "사용자는 언제든 슬러그를
// 명시해 파생을 덮어쓸 수 있다"가 사는 자리이며, 덮어쓸 수 있는 것은
// 취향인 규칙 하나부터 셋까지이고 제약인 규칙 넷은 아니다.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("슬러그가 비었습니다\n파일명으로 쓸 이름을 지정하세요")
	}
	if strings.ContainsAny(slug, `/\`) {
		return fmt.Errorf("슬러그에 경로 구분자가 들어 있습니다: %q\n슬러그는 파일 이름 한 조각입니다. / 와 \\ 를 빼고 다시 지정하세요", slug)
	}
	for _, r := range slug {
		if !unsafeFilenameRune(r) {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("슬러그에 제어 문자가 들어 있습니다: %q (U+%04X)\n보이지 않는 문자입니다. 빼고 다시 지정하세요", slug, r)
		}
		return fmt.Errorf("슬러그에 파일명으로 쓸 수 없는 문자가 들어 있습니다: %q (문제 문자: %q)\nWindows에서 만들 수 없는 이름입니다. %s 를 빼고 다시 지정하세요",
			slug, string(r), unsafeFilenameChars)
	}
	runes := []rune(slug)
	if last := runes[len(runes)-1]; last == '.' || unicode.IsSpace(last) {
		return fmt.Errorf("슬러그가 점이나 공백으로 끝납니다: %q\nWindows가 이름 끝의 점과 공백을 지웁니다. 끝을 다듬어 다시 지정하세요", slug)
	}
	if runes[0] == '.' {
		return fmt.Errorf("슬러그가 점으로 시작합니다: %q\n숨김 파일이 되어 engram이 문서를 찾지 못합니다. 앞의 점을 빼고 다시 지정하세요", slug)
	}
	if isReservedName(slug) {
		return fmt.Errorf("슬러그 %q가 Windows 예약 파일명입니다\nCON, PRN, AUX, NUL, COM1부터 COM9, LPT1부터 LPT9는 파일명으로 쓸 수 없습니다. 다른 이름을 지정하세요", slug)
	}
	return nil
}

// isReservedName은 Windows가 장치 이름으로 예약한 파일명인지 검사한다.
// 예약 여부는 확장자와 무관하게 이름 부분으로 판정한다.
func isReservedName(s string) bool {
	name, _, _ := strings.Cut(s, ".")
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}

// FilePath는 단계에 맞는 문서의 위키 루트 상대 경로를 만든다.
// context 문서는 날짜 접두사 없이 <슬러그>.md 이고 inbox와 sources 문서는
// <날짜>-<슬러그>.md 다. 날짜는 YYYY-MM-DD 또는 연월까지만 아는 경우
// YYYY-MM 이다. upstream 실제 파일명에서 확인한 규칙이다.
// 도착지 디렉토리는 config의 page_dirs가 진실원이다.
func FilePath(cfg config.Config, stage Stage, date, slug string) (string, error) {
	dir, err := DirFor(cfg, stage)
	if err != nil {
		return "", err
	}
	switch stage {
	case StageContext:
		return dir + "/" + slug + ".md", nil
	case StageInbox, StageSource:
		if err := validDate(date); err != nil {
			return "", err
		}
		return dir + "/" + date + "-" + slug + ".md", nil
	}
	return "", fmt.Errorf("알 수 없는 단계: %q (허용값: inbox, source, context)", string(stage))
}

// DirFor는 단계에 대응하는 디렉토리를 page_dirs에서 찾는다. 대응은
// stageDirs 표가 정한다. page_dirs에 없으면 기본값으로 조용히 떨어지지
// 않고 설정에 무엇을 추가해야 하는지를 알린다.
func DirFor(cfg config.Config, stage Stage) (string, error) {
	want, ok := stageDirs[stage]
	if !ok {
		return "", fmt.Errorf("알 수 없는 단계: %q (허용값: inbox, source, context, archive)", string(stage))
	}
	for _, d := range cfg.PageDirs {
		if d == want {
			return d, nil
		}
	}
	return "", fmt.Errorf("설정의 page_dirs에 %s 단계 디렉토리가 없습니다: %q\nengram.yaml의 page_dirs에 %q를 추가하세요",
		stage, want, want)
}

// StageForDir는 최상위 디렉토리 이름에 대응하는 단계를 반환한다.
// lint의 위치 검사가 이 조회를 써서 단계와 디렉토리의 대응이 두 벌이
// 되지 않게 한다(ADR 0031). 대응하지 않는 디렉토리면 false를 반환한다.
func StageForDir(dir string) (Stage, bool) {
	for st, d := range stageDirs {
		if d == dir {
			return st, true
		}
	}
	return "", false
}

// validDate는 날짜 접두사의 정밀도를 검사한다. 하루 단위와 연월 단위만
// 허용한다. 슬러그에 붙는 값이므로 형식이 곧 파일명이다.
func validDate(date string) error {
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if _, err := time.Parse(layout, date); err == nil {
			return nil
		}
	}
	return fmt.Errorf("날짜 %q가 YYYY-MM-DD 또는 YYYY-MM 형식이 아닙니다", date)
}

// Create는 위키 루트 아래 단계에 맞는 자리에 문서를 만들고 만든 파일
// 경로를 반환한다. 도착지 디렉토리는 config의 page_dirs를 따른다.
// 기존 파일은 덮어쓰지 않는다. 디렉토리가 없으면 만든다.
func Create(wikiRoot string, cfg config.Config, stage Stage, date, slug string, fm map[string]any, body string) (string, error) {
	rel, err := FilePath(cfg, stage, date, slug)
	if err != nil {
		return "", err
	}
	path := filepath.Join(wikiRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("디렉토리를 만들 수 없음: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 슬러그를 다르게 지정하거나 기존 문서를 손으로 고치세요", path)
	}
	if err != nil {
		return "", fmt.Errorf("문서를 만들 수 없음: %s: %w", path, err)
	}
	if _, err := f.Write(doc.RenderMap(fm, body)); err != nil {
		f.Close()
		return "", fmt.Errorf("문서를 쓸 수 없음: %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("문서를 닫을 수 없음: %s: %w", path, err)
	}
	return path, nil
}

// Frontmatter는 단계별 프론트매터 초기값을 만든다. config가 켠 속성만 채우고
// 꺼진 속성의 키는 아예 넣지 않는다. 단계별 기본값은 upstream 계약
// meta/frontmatter-schema.md를 따른다. 날짜 필드는 호출자가 넣는다.
func Frontmatter(stage Stage, cfg config.Config) map[string]any {
	fm := map[string]any{
		"type":           stageType(stage),
		"artifact_stage": string(stage),
		"status":         stageStatus(stage),
		"indexable":      stage == StageContext,
	}
	setIfOn(fm, cfg, config.AxisScope, "unknown")
	setIfOn(fm, cfg, config.AxisSensitivity, stageSensitivity(stage))
	setIfOn(fm, cfg, config.AxisSourceChannel, nil)
	setIfOn(fm, cfg, config.AxisTriggerMode, nil)
	setIfOn(fm, cfg, config.AxisWorkflow, nil)
	if stage != StageInbox {
		// 관계 필드는 source와 context 단계에서 빈 배열로 시작한다.
		// updated는 도구가 git 이력에서 채우는 필드라 초기값에 넣지 않는다.
		for _, field := range []string{"source_refs", "derived_from", "derived_context", "related", "tags"} {
			setIfOn(fm, cfg, config.Axis(field), []string{})
		}
	}
	return fm
}

// setIfOn은 속성이 켜져 있을 때만 키를 채운다.
func setIfOn(fm map[string]any, cfg config.Config, axis config.Axis, v any) {
	if cfg.Axes[axis] {
		fm[string(axis)] = v
	}
}

// stageType은 단계별 문서 종류 기본값이다.
func stageType(stage Stage) string {
	switch stage {
	case StageInbox:
		return "inbox-note"
	case StageSource:
		return "source-summary"
	}
	return "concept"
}

// stageStatus은 단계별 상태 기본값이다.
func stageStatus(stage Stage) string {
	switch stage {
	case StageInbox:
		return "inbox"
	case StageSource:
		return "sourced"
	case StageArchive:
		return "archived"
	}
	return "promoted"
}

// stageSensitivity은 단계별 민감도 기본값이다. 캡처 시점에는 가장 좁은
// 값에서 시작하고 승급 심사에서 넓힌다.
func stageSensitivity(stage Stage) string {
	if stage == StageInbox {
		return "private-local-only"
	}
	return "internal"
}

// Linkable은 위키에서 링크 대상이 될 수 있는 문서의 슬러그를 정렬해
// 반환한다. 대상은 page_dirs 아래 .md 파일과 root_files이다.
// promote와 new가 게이트 판정에 쓴다.
func Linkable(wikiRoot string, cfg config.Config) ([]string, error) {
	seen := map[string]bool{}
	var slugs []string
	add := func(rel string) {
		slug := strings.TrimSuffix(filepath.Base(filepath.ToSlash(rel)), ".md")
		if !seen[slug] {
			seen[slug] = true
			slugs = append(slugs, slug)
		}
	}
	dirs := append([]string{}, cfg.PageDirs...)
	sort.Strings(dirs)
	for _, d := range dirs {
		if strings.HasPrefix(d, ".") {
			continue
		}
		err := filepath.WalkDir(filepath.Join(wikiRoot, d), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".md") {
				rel, err := filepath.Rel(wikiRoot, path)
				if err != nil {
					return err
				}
				add(rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, f := range cfg.RootFiles {
		if strings.HasSuffix(f, ".md") {
			if _, err := os.Stat(filepath.Join(wikiRoot, f)); err == nil {
				add(f)
			}
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}
