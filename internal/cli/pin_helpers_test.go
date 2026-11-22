package cli

import (
	"testing"

	"github.com/neocode24/engram/internal/pin"
)

// withPinNone은 시험이 Pin 판단을 거치지 않고 문서를 넣을 수 있게 --pin none 을 붙이고, 손으로 만든
// 시험 위키에는 pin/ 을 채운다. 세 명령(capture, source, promote --to sources)은 모든 위키에서
// --pin 이 필요하므로(ADR 0108) 문서 넣기가 주제가 아닌 시험은 이것을 거친다. --pin 이 필요하다는
// 동작 자체는 pin_test.go 가 따로 본다.
func withPinNone(t *testing.T, args []string) []string {
	t.Helper()
	if len(args) == 0 {
		return args
	}
	value := func(name string) string {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	has := func(s string) bool {
		for _, a := range args {
			if a == s {
				return true
			}
		}
		return false
	}
	ingests := false
	switch args[0] {
	case "capture", "source":
		ingests = true
	case "promote":
		ingests = value("--to") == "sources"
	}
	if !ingests {
		return args
	}
	if wiki := value("--wiki"); wiki != "" {
		if _, err := pin.Init(wiki); err != nil {
			t.Fatalf("시험 위키에 pin/ 을 채우지 못했습니다: %v", err)
		}
	}
	if has("--pin") {
		return args
	}
	// "--" 뒤는 위치 인자라 플래그는 그 앞에 둔다.
	out := append([]string{}, args...)
	at := len(out)
	for i, a := range out {
		if a == "--" {
			at = i
			break
		}
	}
	return append(out[:at], append([]string{"--pin", "none"}, out[at:]...)...)
}
