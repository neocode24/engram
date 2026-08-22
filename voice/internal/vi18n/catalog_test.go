package vi18n

import (
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/i18n"
)

// TestCatalogsMatch은 voice 항목이 언어마다 빠짐없이 있는지 본다.
// 한쪽에만 있으면 그 언어에서 ID 가 그대로 노출된다.
func TestCatalogsMatch(t *testing.T) {
	sets := map[i18n.Lang]map[string]bool{}
	for _, l := range i18n.Langs() {
		m := map[string]bool{}
		for _, id := range i18n.IDs(l) {
			if strings.HasPrefix(id, "voice.") {
				m[id] = true
			}
		}
		sets[l] = m
	}
	if len(sets[i18n.LangKO]) == 0 {
		t.Fatal("voice 항목이 하나도 등록되지 않았습니다")
	}
	for id := range sets[i18n.LangKO] {
		if !sets[i18n.LangEN][id] {
			t.Errorf("영어 카탈로그에 없습니다: %s", id)
		}
	}
	for id := range sets[i18n.LangEN] {
		if !sets[i18n.LangKO][id] {
			t.Errorf("한국어 카탈로그에 없습니다: %s", id)
		}
	}
}

// TestVerbsMatch는 같은 ID 의 서식 지시자 수가 언어마다 같은지 본다.
// 다르면 Sprintf 가 %!(EXTRA ...) 나 %!s(MISSING) 를 낸다.
func TestVerbsMatch(t *testing.T) {
	count := func(s string) int {
		n := 0
		for i := 0; i < len(s)-1; i++ {
			if s[i] == '%' {
				if s[i+1] == '%' {
					i++
					continue
				}
				n++
			}
		}
		return n
	}
	for _, id := range i18n.IDs(i18n.LangKO) {
		if !strings.HasPrefix(id, "voice.") {
			continue
		}
		i18n.SetLang(i18n.LangKO)
		ko := i18n.T(id)
		i18n.SetLang(i18n.LangEN)
		en := i18n.T(id)
		i18n.SetLang(i18n.Fallback)
		if count(ko) != count(en) {
			t.Errorf("%s: 서식 지시자 수가 다릅니다. ko=%d en=%d", id, count(ko), count(en))
		}
	}
}
