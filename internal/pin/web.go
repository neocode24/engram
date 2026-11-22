package pin

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"regexp"
	"strings"
)

// template.html은 upstream 의 Pin 화면에서 설정 구역 값을 engram 값(API='/pin/api/',
// SOURCES=['wiki'], AUTH='none')으로 바꾸고 사이트 구역(`<!-- pin:site begin -->`부터 `end`까지)을
// 지운 것이다. 원본은 upstream main 의 scripts/pin/template.html 이다. 변환 방법과 원본 커밋은
// ADR 0106 에 있다. 이 파일을 손으로 고치지 않는다.
//
//go:embed web/template.html
var page []byte

// Page는 화면 파일 전체다.
func Page() []byte { return page }

var inlineRe = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>(.*?)</(?:script|style)\s*>`)

// PageCSP는 화면에 거는 보안 정책을 낸다. 화면 파일의 인라인 script 와 style 을 sha256 해시로 허용한다.
// 해시는 임베드한 파일에서 계산하므로 화면이 바뀌어도 정책이 따라간다. 바깥으로 나가는 요청은 모두 막는다.
func PageCSP() string {
	var scripts, styles []string
	for _, m := range inlineRe.FindAllStringSubmatch(string(page), -1) {
		sum := sha256.Sum256([]byte(m[2]))
		h := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if strings.EqualFold(m[1], "script") {
			scripts = append(scripts, h)
		} else {
			styles = append(styles, h)
		}
	}
	return "default-src 'none'; script-src " + strings.Join(scripts, " ") + "; style-src " + strings.Join(styles, " ") +
		"; connect-src 'self'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
}
