// Package i18n는 사용자 대면 문자열의 언어를 고른다. 외부 의존성 없이
// 컴파일 시점에 카탈로그를 묶는다. ADR 0049.
//
// 카탈로그는 패키지별 파일로 나뉜다. catalog_<영역>.go 가 init 에서
// Register 를 부르고, 이 파일은 조회와 언어 결정만 맡는다. 파일을
// 나누는 이유는 여러 사람이 동시에 손댈 때 한 파일에서 충돌하지
// 않게 하기 위해서다.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Lang은 지원 언어다.
type Lang string

const (
	LangKO Lang = "ko"
	LangEN Lang = "en"
)

// Fallback은 어디에서도 언어를 정하지 못했을 때의 값이다.
//
// 한국어다. 이 저장소의 문서와 강의 자료, 골든 스냅샷이 전부 한국어
// 출력을 전제로 하므로 기본이 바뀌면 그것들이 한꺼번에 어긋난다.
// LANG 환경변수는 보지 않는다. 한국어 사용자의 macOS LANG 이 흔히
// en_US 라서, 그것을 보면 강의 수강생 다수가 문서와 다른 출력을 받는다.
// 영어를 보려면 ENGRAM_LANG=en 또는 --lang en 으로 명시한다.
const Fallback = LangKO

// EnvVar는 언어를 정하는 환경변수 이름이다.
const EnvVar = "ENGRAM_LANG"

var (
	mu       sync.RWMutex
	current  = Fallback
	catalogs = map[Lang]map[string]string{}
)

// Register는 카탈로그 조각을 등록한다. catalog_*.go 의 init 이 부른다.
// 같은 ID 를 두 번 등록하면 프로그램을 멈춘다. 조용히 덮으면 어느 쪽이
// 이겼는지 알 수 없고 출력이 파일 순서에 따라 달라진다.
func Register(lang Lang, entries map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	c, ok := catalogs[lang]
	if !ok {
		c = map[string]string{}
		catalogs[lang] = c
	}
	for id, text := range entries {
		if _, dup := c[id]; dup {
			panic(fmt.Sprintf("i18n: 메시지 ID가 %s 카탈로그에 중복 등록되었습니다: %s", lang, id))
		}
		c[id] = text
	}
}

// Known은 지원하는 언어인지 본다.
func Known(l Lang) bool {
	return l == LangKO || l == LangEN
}

// Langs는 지원 언어를 고정 순서로 반환한다.
func Langs() []Lang {
	return []Lang{LangKO, LangEN}
}

// Resolve는 쓸 언어를 정한다. 우선순위는 인자로 받은 플래그 값,
// ENGRAM_LANG 환경변수, Fallback 순이다. 알 수 없는 값은 에러다.
// 조용히 무시하면 오타를 낸 사용자가 왜 언어가 안 바뀌는지 모른다.
func Resolve(flag string) (Lang, error) {
	for _, raw := range []string{flag, os.Getenv(EnvVar)} {
		v := strings.TrimSpace(strings.ToLower(raw))
		if v == "" {
			continue
		}
		l := Lang(v)
		if !Known(l) {
			return "", fmt.Errorf("언어 값이 허용값이 아님: %q (허용값: ko, en)", raw)
		}
		return l, nil
	}
	return Fallback, nil
}

// SetLang은 현재 언어를 바꾼다. 커맨드 시작 시 한 번 부른다.
func SetLang(l Lang) {
	mu.Lock()
	defer mu.Unlock()
	current = l
}

// Current는 지금 언어다.
func Current() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// T는 메시지 ID로 문자열을 만든다. 인자가 있으면 fmt.Sprintf 로 채운다.
//
// 현재 언어에 ID 가 없으면 Fallback 카탈로그를 본다. 거기에도 없으면
// ID 자체를 낸다. 빈 문자열을 내면 출력에서 문장이 사라져 원인을 찾기
// 어렵지만, ID 가 그대로 보이면 어느 메시지가 빠졌는지 즉시 드러난다.
func T(id string, args ...any) string {
	mu.RLock()
	text, ok := catalogs[current][id]
	if !ok {
		text, ok = catalogs[Fallback][id]
	}
	mu.RUnlock()
	if !ok {
		return id
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// Has는 그 언어 카탈로그에 ID 가 있는지 본다. 카탈로그 완결성 시험이 쓴다.
func Has(l Lang, id string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := catalogs[l][id]
	return ok
}

// IDs는 그 언어 카탈로그의 ID 전부를 반환한다. 완결성 시험이 쓴다.
func IDs(l Lang) []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(catalogs[l]))
	for id := range catalogs[l] {
		out = append(out, id)
	}
	return out
}
