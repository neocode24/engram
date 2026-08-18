// Package secret는 시크릿으로 보이는 문자열을 찾는 판정을 담는다.
// promote, new, export, lint 가 같은 판정을 쓴다. 같은 판정을 두 벌
// 두면 어느 커맨드는 통과하고 어느 커맨드는 거절하는 문서가 생긴다(ADR 0069).
package secret

import (
	"regexp"
	"strings"
)

// Finding은 찾은 것 하나다.
type Finding struct {
	Rule string // 패턴 이름
	Line int    // 1 기반 줄 번호
}

// patterns는 여섯 패턴이다. home-path 만 유닉스와 윈도우 표기 둘을 갖는다.
// 엔트로피 기반 판정은 넣지 않는다. 오탐 한 건이 규칙 전체의 신뢰를
// 깎으며, 패턴을 늘리는 것은 요구가 실제로 나올 때 하고 늘릴 때마다
// 오탐 비용을 함께 재야 한다(ADR 0069).
var patterns = []struct {
	rule string
	re   *regexp.Regexp
}{
	{"private-key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"aws-access-key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"bearer-token", regexp.MustCompile(`(?i)authorization:\s*bearer\s+[A-Za-z0-9._~+/-]{20,}`)},
	{"home-path", regexp.MustCompile(`(/Users/|/home/)[A-Za-z0-9._-]+/`)},
	{"home-path", regexp.MustCompile(`(?i)C:\\Users\\[A-Za-z0-9._-]+\\`)},
}

// Scan은 본문에서 시크릿으로 보이는 것을 찾는다.
// 코드 펜스 안도 검사한다. 시크릿은 코드 블록에 붙여 넣는 일이 더
// 흔하다. 위키링크 추출(internal/doc)이 코드 펜스를 건너뛰는 것과 반대
// 방향이다. 링크 문법은 문서에 설명으로 등장하지만 실제 시크릿은 코드
// 블록에 붙는다(ADR 0069).
// 줄마다 규칙당 최대 한 건을 낸다. 값은 세지 않는다. 호출자는 규칙 이름과
// 줄 번호만 알리고 값을 출력하지 않는다. 거절 메시지에 값이 실리면 그
// 메시지가 로그와 터미널 스크롤백으로 옮겨지므로 그것 자체가 유출이다.
func Scan(body string) []Finding {
	var out []Finding
	for i, line := range strings.Split(body, "\n") {
		for _, p := range patterns {
			if p.re.MatchString(line) {
				out = append(out, Finding{Rule: p.rule, Line: i + 1})
			}
		}
	}
	return out
}
