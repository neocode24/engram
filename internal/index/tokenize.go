// token.go는 한국어 검색을 위한 토크나이저다. ADR 0010의 확정 결정을
// 그대로 구현한다. 문자 bigram 과 라틴 통짜 토큰을 구간 종류별로 갈라
// 낸다. 형태소 사전을 쓰지 않는다.
package index

import (
	"strings"
	"unicode"
)

// Tokenize는 문자열을 검색 토큰으로 나눈다. 결과에 중복이 있을 수 있으며
// 빈도 계산은 호출자 몫이다.
//
// 구간은 문자 종류가 바뀌는 지점에서 갈린다.
//   - 한글 구간은 문자 bigram으로 자른다. "게이트웨이"는 게이, 이트, 트웨, 웨이다.
//     1글자 구간은 그 글자 하나를 토큰으로 둔다
//   - 라틴 문자와 숫자 구간은 소문자로 내린 통짜 토큰을 유지한다.
//     min_wikilinks 같은 식별자가 bigram으로 쪼개지면 검색이 무너지기 때문이다.
//     언더스코어와 하이픈으로 나눈 조각을 추가로 낸다
//   - 공백과 문장부호 등 나머지는 구분자다. 토큰이 되지 않는다
func Tokenize(s string) []string {
	runes := []rune(s)
	var tokens []string
	i := 0
	for i < len(runes) {
		switch {
		case isHangul(runes[i]):
			j := i
			for j < len(runes) && isHangul(runes[j]) {
				j++
			}
			tokens = append(tokens, hangulGrams(runes[i:j])...)
			i = j
		case isWordRune(runes[i]):
			j := i
			for j < len(runes) && isWordRune(runes[j]) {
				j++
			}
			tokens = append(tokens, wordTokens(runes[i:j])...)
			i = j
		default:
			i++
		}
	}
	return tokens
}

// hangulGrams는 한글 구간을 bigram으로 자른다.
func hangulGrams(run []rune) []string {
	if len(run) == 1 {
		return []string{string(run)}
	}
	out := make([]string, 0, len(run)-1)
	for i := 0; i+1 < len(run); i++ {
		out = append(out, string(run[i:i+2]))
	}
	return out
}

// wordTokens는 라틴과 숫자 구간의 토큰을 낸다. 통짜 토큰 하나와
// 언더스코어와 하이픈으로 나눈 조각들이다. 구간이 구분자 문자만으로
// 이루어지면 토큰이 없다.
func wordTokens(run []rune) []string {
	word := strings.ToLower(string(run))
	hasAlnum := false
	for _, r := range word {
		if r != '_' && r != '-' {
			hasAlnum = true
			break
		}
	}
	if !hasAlnum {
		return nil
	}
	out := []string{word}
	for _, part := range strings.FieldsFunc(word, func(r rune) bool { return r == '_' || r == '-' }) {
		if part != word {
			out = append(out, part)
		}
	}
	return out
}

// isHangul은 한글 문자인지 검사한다.
func isHangul(r rune) bool {
	return unicode.Is(unicode.Hangul, r)
}

// isWordRune는 라틴 문자, 숫자, 단어 결합 문자인지 검사한다.
// 결합 문자는 라틴 구간 안에서만 살아남고 한글 사이에서는 구분자로 취급된다.
func isWordRune(r rune) bool {
	switch {
	case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z':
		return true
	case unicode.IsDigit(r):
		return true
	case r == '_' || r == '-':
		return true
	}
	return false
}
