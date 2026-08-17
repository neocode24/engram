package i18n

import (
	"sort"
	"strings"
	"testing"
)

// TestCatalogsAreComplete는 언어마다 같은 ID 집합을 담고 있는지 본다.
// 골든 스냅샷은 한국어 하나로만 찍으므로, 영어 카탈로그가 빠진 것은
// 이 시험이 아니면 아무도 잡지 못한다.
func TestCatalogsAreComplete(t *testing.T) {
	base := IDs(Fallback)
	sort.Strings(base)
	for _, l := range Langs() {
		if l == Fallback {
			continue
		}
		var missing []string
		for _, id := range base {
			if !Has(l, id) {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			t.Errorf("%s 카탈로그에 %d개가 빠졌습니다:\n  %s", l, len(missing), strings.Join(missing, "\n  "))
		}
		var extra []string
		for _, id := range IDs(l) {
			if !Has(Fallback, id) {
				extra = append(extra, id)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("%s 카탈로그에 %s에 없는 ID가 있습니다:\n  %s", l, Fallback, strings.Join(extra, "\n  "))
		}
	}
}

// TestVerbsMatch는 같은 ID의 서식 지정자가 언어마다 같은지 본다.
// 하나가 %d인데 다른 하나가 %s면 그 언어에서만 %!s(int=3) 이 나온다.
func TestVerbsMatch(t *testing.T) {
	for _, id := range IDs(Fallback) {
		want := verbs(catalogs[Fallback][id])
		for _, l := range Langs() {
			if l == Fallback || !Has(l, id) {
				continue
			}
			got := verbs(catalogs[l][id])
			if !sameMultiset(want, got) {
				t.Errorf("%s: 서식 지정자가 다릅니다. %s=%v, %s=%v", id, Fallback, want, l, got)
			}
		}
	}
}

// verbs는 서식 문자열의 지정자를 순서대로 뽑는다. 인자 색인(%[1]s)은
// 언어마다 어순이 달라 바뀌는 것이 정상이므로 색인을 뗀 동사만 본다.
func verbs(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		j := i + 1
		if j < len(s) && s[j] == '%' {
			i = j
			continue
		}
		if j < len(s) && s[j] == '[' {
			for j < len(s) && s[j] != ']' {
				j++
			}
			j++
		}
		for j < len(s) && strings.ContainsRune("+-# 0123456789.", rune(s[j])) {
			j++
		}
		if j < len(s) {
			out = append(out, string(s[j]))
		}
		i = j
	}
	return out
}

func sameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func TestResolve(t *testing.T) {
	t.Setenv(EnvVar, "")
	if got, err := Resolve(""); err != nil || got != Fallback {
		t.Errorf("빈 입력 = %q, %v, want %q, nil", got, err, Fallback)
	}
	if got, err := Resolve("en"); err != nil || got != LangEN {
		t.Errorf("--lang en = %q, %v", got, err)
	}
	if got, err := Resolve(" KO "); err != nil || got != LangKO {
		t.Errorf("공백과 대문자를 다듬어야 합니다: %q, %v", got, err)
	}
	if _, err := Resolve("fr"); err == nil {
		t.Error("모르는 언어는 에러여야 합니다")
	}

	t.Setenv(EnvVar, "en")
	if got, _ := Resolve(""); got != LangEN {
		t.Errorf("환경변수 = %q, want en", got)
	}
	if got, _ := Resolve("ko"); got != LangKO {
		t.Errorf("플래그가 환경변수를 이겨야 합니다: %q", got)
	}
}

// TestTFallsBackToID는 없는 ID가 조용히 빈 줄이 되지 않는지 본다.
func TestTFallsBackToID(t *testing.T) {
	if got := T("없는.메시지.아이디"); got != "없는.메시지.아이디" {
		t.Errorf("T = %q, want ID 그대로", got)
	}
}
