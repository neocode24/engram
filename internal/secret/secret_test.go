package secret

import (
	"fmt"
	"testing"
)

// ruleLines는 찾은 것을 "규칙:줄" 목록으로 낸다. 단언에 쓴다.
func ruleLines(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s:%d", f.Rule, f.Line))
	}
	return out
}

func TestScan(t *testing.T) {
	t.Run("여섯 패턴을 잡습니다", func(t *testing.T) {
		cases := []struct {
			rule  string
			input string
		}{
			{"private-key", "-----BEGIN RSA PRIVATE KEY-----"},
			{"private-key", "-----BEGIN OPENSSH PRIVATE KEY-----"},
			{"aws-access-key", "aws_key = AKIAIOSFODNN7EXAMPLE"},
			{"github-token", "token: ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD"},
			{"slack-token", "SLACK=xoxb-123456789012-abcdefghij"},
			{"bearer-token", "Authorization: Bearer abcdefghijklmnopqrstuvwxyz123456"},
			{"bearer-token", "authorization: bearer abcdefghijklmnopqrstuvwxyz123456"},
			{"home-path", "설정은 /Users/jay/Documents/wiki에 있습니다"},
			{"home-path", "설정은 /home/jay/wiki에 있습니다"},
			{"home-path", `설정은 C:\Users\jay\wiki에 있습니다`},
		}
		for _, c := range cases {
			got := ruleLines(Scan(c.input + "\n"))
			if len(got) != 1 || got[0] != c.rule+":1" {
				t.Errorf("%q = %v, 기대 [%s:1]", c.input, got, c.rule)
			}
		}
	})

	t.Run("비슷하지만 아닌 문자열은 잡지 않습니다", func(t *testing.T) {
		cases := []string{
			"-----BEGIN CERTIFICATE-----",
			"AKIAIOSFODNN7EXAMPL",     // 15글자
			"AKIAiosfodnn7EXAMPLEE",   // 소문자 포함
			"ghp_0123456789abcdefgh",  // 짧다
			"gkx_0123456789abcdefghijklmnopqrstuvwxyzABCD", // 접두가 다르다
			"xoxy-123456789012345",
			"Authorization: Bearer short_token",
			"Bearer 라는 단어만 있습니다",
			"/usr/local/bin/engram",
			"/Users/README.md", // 홈 바로 아래 디렉토리가 없다
			"/Users//tmp",
			"C:\\Users\\", // 사용자 이름이 비었다
			"문서에 /home/ 경로 예시를 적습니다",
		}
		for _, c := range cases {
			if got := Scan(c); got != nil {
				t.Errorf("%q = %v, 기대 nil", c, ruleLines(got))
			}
		}
	})

	t.Run("코드 펜스 안도 검사합니다", func(t *testing.T) {
		body := "설명 줄입니다.\n```bash\n" +
			"export GITHUB_TOKEN=ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD\n" +
			"```\n마무리 줄입니다.\n"
		got := ruleLines(Scan(body))
		want := []string{"github-token:3"}
		if len(got) != len(want) {
			t.Fatalf("결과 = %v, 기대 %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("결과[%d] = %s, 기대 %s", i, got[i], want[i])
			}
		}
	})

	t.Run("한 줄에 여러 규칙이 걸리면 모두 낸다", func(t *testing.T) {
		got := ruleLines(Scan("AKIAIOSFODNN7EXAMPLE 그리고 xoxb-123456789012-abcdefghij\n"))
		if len(got) != 2 || got[0] != "aws-access-key:1" || got[1] != "slack-token:1" {
			t.Errorf("결과 = %v, 기대 [aws-access-key:1 slack-token:1]", got)
		}
	})
}
