// Package doc는 마크다운 문서 하나를 프론트매터와 본문으로 가르고
// 프론트매터를 YAML로 파싱하고 본문에서 위키링크를 추출한다.
// 필드 검증은 lint 쪽 책임이고 여기는 파싱까지다.
package doc

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/token"

	"github.com/neocode24/engram/internal/i18n"
)

// ValueKind는 프론트매터 값의 종류를 구분한다.
type ValueKind int

const (
	KindEmpty ValueKind = iota // 값이 없는 키 (source_channel: 처럼)
	KindString
	KindStringList
	KindBool
	KindDate // 날짜 형식의 문자열
)

// Field는 프론트매터 키 하나의 파싱 결과다. 원본에 나온 순서를 유지한다.
type Field struct {
	Key  string
	Kind ValueKind
	Str  string   // KindString, KindDate
	List []string // KindStringList
	Bool bool     // KindBool
}

// Doc는 문서 하나의 파싱 결과다.
type Doc struct {
	Path           string
	HasFrontmatter bool
	Fields         []Field
	Body           string
	BodyLine       int // 본문 첫 줄의 원본 파일 줄 번호 (1 기반)

	// fmLines는 프론트매터 원문 줄들. related 링크의 줄 번호를 잡을 때 쓴다.
	fmLines []string
}

// tokenizedError는 goccy가 반환하는 문법/타입 에러에서 토큰 위치를 꺼내기 위한 인터페이스다.
type tokenizedError interface {
	GetToken() *token.Token
}

// datePattern은 날짜 문자열을 값 타입 구분용으로만 알아보는 패턴이다. 실제 유효성은 검사하지 않는다.
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}([Tt ].*)?$`)

// Parse는 문서 원문을 프론트매터와 본문으로 가른 뒤 프론트매터를 파싱한다.
// 프론트매터가 없는 문서도 정상 입력이며 이때 Body는 원문 전체고 BodyLine은 1이다.
func Parse(path string, content []byte) (Doc, error) {
	d := Doc{Path: path, BodyLine: 1}

	// BOM을 허용한다. CRLF도 줄 수를 바꾸지 않는 선에서 LF로 정규화한다.
	text := strings.TrimPrefix(string(content), "\xEF\xBB\xBF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == "---" {
		close := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimRight(lines[i], " \t") == "---" {
				close = i
				break
			}
		}
		if close < 0 {
			return Doc{}, errors.New(i18n.T("core.doc.fm_unclosed", path, 1))
		}
		d.HasFrontmatter = true
		d.fmLines = lines[1:close]
		d.BodyLine = close + 2
		d.Body = strings.Join(lines[close+1:], "\n")
		if err := parseFields(&d, strings.Join(d.fmLines, "\n")); err != nil {
			return Doc{}, err
		}
	} else {
		d.Body = text
	}
	return d, nil
}

// parseFields는 프론트매터 텍스트를 YAML로 파싱해 순서 보존 필드로 만든다.
// MapSlice를 쓰는 이유는 나중에 프론트매터 정규화 출력의 parity 비교 축이 되므로
// 키 순서가 결정론적이어야 하기 때문이다.
func parseFields(d *Doc, fmText string) error {
	var ms yaml.MapSlice
	if err := yaml.Unmarshal([]byte(fmText), &ms); err != nil {
		return fmt.Errorf("%s: %w", i18n.T("core.doc.fm_parse_fail", d.Path, fmErrorLine(err)), err)
	}
	for _, item := range ms {
		key, ok := item.Key.(string)
		if !ok {
			return errors.New(i18n.T("core.doc.key_not_string", d.Path, item.Key))
		}
		d.Fields = append(d.Fields, toField(key, item.Value))
	}
	return nil
}

// fmErrorLine은 goccy 에러의 토큰 위치에서 원본 파일 기준 줄 번호를 얻는다.
// 프론트매터 본문은 파일 2줄부터 시작하므로 1을 더한다. 토큰이 없으면 2를 기본으로 쓴다.
func fmErrorLine(err error) int {
	var te tokenizedError
	if errors.As(err, &te) && te.GetToken() != nil && te.GetToken().Position.Line > 0 {
		return te.GetToken().Position.Line + 1
	}
	return 2
}

// toField는 YAML 값 하나를 Field로 바꾼다. 빈 값은 정상 입력이다.
func toField(key string, v interface{}) Field {
	f := Field{Key: key}
	switch val := v.(type) {
	case nil:
		f.Kind = KindEmpty
	case bool:
		f.Kind, f.Bool = KindBool, val
	case string:
		f.Str = val
		f.Kind = KindString
		if datePattern.MatchString(val) {
			f.Kind = KindDate
		}
	case []interface{}:
		f.Kind = KindStringList
		for _, item := range val {
			f.List = append(f.List, fmt.Sprint(item))
		}
	default:
		// 숫자처럼 문자열이 아닌 스칼라도 문자열로 다룬다. 스키마 검증은 lint 몫이다.
		f.Kind, f.Str = KindString, fmt.Sprint(val)
	}
	return f
}
