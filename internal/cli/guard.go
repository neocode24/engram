package cli

import (
	"errors"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/secret"
)

// sensitivityBlocksContext는 민감도 값이 context 승급을 막는지 본다.
// upstream taxonomy.md가 "오직 public-reference와 안전한 internal 자료만
// 승급 후보"로 정한 것을 값으로 읽는다. 무엇이 민감한가의 판단은 여전히
// 사람의 몫이고 코드는 확정된 값만 읽는다(ADR 0069). 축이 꺼진 위키에서는
// 판정하지 않는다. 값을 읽을 축 자체가 없기 때문이다.
func sensitivityBlocksContext(v string, cfg config.Config) bool {
	if !cfg.Axes[config.AxisSensitivity] {
		return false
	}
	return v == "restricted" || v == "private-local-only"
}

// checkContextGuards는 context 로 들어가는 문서의 민감도와 시크릿을
// 검사한다. promote 와 new 가 같은 판정을 쓴다(ADR 0069). content 는
// 줄 번호를 사용자가 편집기에서 보는 그대로 내기 위해 파일 전문을 받는다.
func checkContextGuards(sensitivity string, cfg config.Config, content string) error {
	if sensitivityBlocksContext(sensitivity, cfg) {
		return errors.New(i18n.T("cli.promote.sensitivity_blocked", sensitivity))
	}
	return checkSecrets(content)
}

// checkSecrets는 시크릿만 검사한다. 단계를 가리지 않는다. --to sources 는
// 노출 판정에서 이미 닫혀 있어 민감도로 막지 않으나 시크릿은 증거 계층에도
// 두면 안 된다(ADR 0069). 우회로를 두지 않는다. 문서에서 시크릿을 지우는
// 것이 유일한 길이며 그 행위가 곧 기록으로 남는다.
func checkSecrets(content string) error {
	findings := secret.Scan(content)
	if len(findings) == 0 {
		return nil
	}
	lines := []string{i18n.T("cli.promote.secret_blocked")}
	for _, f := range findings {
		lines = append(lines, i18n.T("cli.promote.secret_finding", f.Rule, f.Line))
	}
	return errors.New(strings.Join(lines, "\n"))
}
