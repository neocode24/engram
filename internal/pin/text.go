package pin

import (
	"strings"
	"unicode"
)

// 질문 문장에 쓰는 한국어 조사와 제목 줄임. upstream 의 pinlib 과 같은 규칙이다.
// 상태 파일의 asks[].question 이 이 글을 그대로 담으므로, 규칙이 다르면 하네스가
// 상태 값 차이로 잡는다.

var (
	digitBatchim  = setOf("0", "1", "3", "6", "7", "8") // 영, 일, 삼, 육, 칠, 팔은 받침이 있다
	letterBatchim = setOf("L", "M", "N", "R")           // 엘, 엠, 엔, 알
)

var josaPairs = map[string][2]string{
	"과/와": {"과", "와"}, "이/가": {"이", "가"}, "을/를": {"을", "를"}, "은/는": {"은", "는"},
}

// hasBatchim은 글의 마지막 읽는 소리에 받침이 있는지 낸다. 판정이 어려우면 ok가 거짓이다.
func hasBatchim(text string) (batchim, ok bool) {
	core := []rune(strings.TrimRight(text, " \t\r\n\v\f"))
	for len(core) > 0 && !isAlnum(core[len(core)-1]) {
		core = core[:len(core)-1]
	}
	if len(core) == 0 {
		return false, false
	}
	last := core[len(core)-1]
	switch {
	case last >= 0xAC00 && last <= 0xD7A3:
		return (int(last)-0xAC00)%28 != 0, true
	case last >= '0' && last <= '9':
		return digitBatchim[string(last)], true
	case last < 128 && unicode.IsLetter(last):
		var run []rune
		for i := len(core) - 1; i >= 0; i-- {
			if core[i] < 128 && unicode.IsLetter(core[i]) {
				run = append([]rune{core[i]}, run...)
			} else {
				break
			}
		}
		if isUpper(run) {
			return letterBatchim[string(last)], true
		}
	}
	return false, false
}

func isAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsNumber(r) }

// isUpper는 파이썬 str.isupper()와 같다. 글자가 하나라도 있고 대문자 아닌 글자가 없다.
func isUpper(run []rune) bool {
	cased := false
	for _, r := range run {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsUpper(r) {
			cased = true
		}
	}
	return cased
}

// josa는 받침에 맞는 조사를 낸다. 판정이 어려우면 ok가 거짓이다.
func josa(text, pair string) (string, bool) {
	b, ok := hasBatchim(text)
	if !ok {
		return "", false
	}
	p := josaPairs[pair]
	if b {
		return p[0], true
	}
	return p[1], true
}

func relationQuestion(kind, titleA, titleB string) string {
	a, b := "「"+shortTitle(titleA)+"」", "「"+shortTitle(titleB)+"」"
	ta, tb := shortTitle(titleA), shortTitle(titleB)
	if kind == "before" {
		j1, ok1 := josa(ta, "이/가")
		j2, ok2 := josa(tb, "을/를")
		if ok1 && ok2 {
			return a + j1 + " 끝나야 " + b + j2 + " 할 수 있나요?"
		}
		return a + ", " + b + ": 앞의 일이 끝나야 뒤의 일을 할 수 있나요?"
	}
	tail := map[string]string{"related": "관련된 일", "duplicate": "같은 일"}[kind]
	j1, ok1 := josa(ta, "과/와")
	j2, ok2 := josa(tb, "은/는")
	if ok1 && ok2 {
		return a + j1 + " " + b + j2 + " " + tail + "인가요?"
	}
	return a + ", " + b + ": " + tail + "인가요?"
}

func relationLabel(kind, side string) string {
	if kind == "before" {
		if side == "from" {
			return "후행"
		}
		return "선행"
	}
	return map[string]string{"related": "관련", "duplicate": "중복"}[kind]
}
