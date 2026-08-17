package pack

import (
	"fmt"
	"strings"
)

// separator는 치환 파일의 구분자다. 이 저장소가 자기 vendoring 익명화에
// 이미 쓰고 있는 형식과 같다. 새 형식을 만들 이유가 없다(ADR 0046).
const separator = "==>"

// ParseReplacements는 치환 파일 내용을 규칙 목록으로 만든다.
//
// 한 줄에 규칙 하나이며 원문과 대체어를 ==> 로 나눈다. # 으로 시작하는
// 줄과 빈 줄은 건너뛴다. 대체어는 비어도 된다. 지우는 규칙이 된다.
//
// 사전은 engram 이 내장하지 않는다. 조직 어휘 목록을 담을 수 없기
// 때문이며 메커니즘만 주고 사전은 쓰는 사람이 채운다(ADR 0024, 0046).
func ParseReplacements(src string) ([]Rule, error) {
	var rules []Rule
	seen := map[string]int{}
	for i, line := range strings.Split(src, "\n") {
		no := i + 1
		t := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		from, to, ok := strings.Cut(t, separator)
		if !ok {
			return nil, fmt.Errorf("%d번째 줄에 %s 가 없습니다: %s", no, separator, t)
		}
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" {
			return nil, fmt.Errorf("%d번째 줄의 원문이 비었습니다", no)
		}
		if prev, dup := seen[from]; dup {
			return nil, fmt.Errorf("%d번째 줄의 원문이 %d번째 줄과 겹칩니다: %s", no, prev, from)
		}
		seen[from] = no
		rules = append(rules, Rule{From: from, To: to})
	}
	return rules, nil
}
