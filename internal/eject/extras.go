package eject

import (
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
)

// preCommitHook는 생성된 린터를 부르는 git 훅이다. engram 저장소의 훅과
// 같은 구조를 쓴다. 문구는 카탈로그에서 얻어 시점 언어로 굳는다(ADR 0049).
func preCommitHook() string {
	return "#!/bin/sh\n" +
		i18n.T("eject.hook.comment") +
		"set -e\n" +
		"\n" +
		"ROOT=$(git rev-parse --show-toplevel)\n" +
		"\n" +
		"python3 \"$ROOT/scripts/lint-frontmatter.py\" \"$ROOT\" || {\n" +
		"    " + fmt.Sprintf("echo %q", i18n.T("eject.hook.blocked")) + "\n" +
		"    " + fmt.Sprintf("echo %q", i18n.T("eject.hook.no_python")) + "\n" +
		"    exit 1\n" +
		"}\n"
}

// agentsDoc는 그 위키에서 일하는 에이전트의 작업 계약이다. 규칙은 이제
// 이 위키의 파일이고 연산은 engram 이 계속 수행한다는 사실이 중심이다.
func agentsDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString(i18n.T("eject.agents_doc.heading"))
	b.WriteString(i18n.T("eject.agents_doc.rules_heading"))
	b.WriteString(i18n.T("eject.agents_doc.rules_body"))
	b.WriteString(i18n.T("eject.agents_doc.hook_body"))
	b.WriteString(i18n.T("eject.agents_doc.compute_heading"))
	b.WriteString(i18n.T("eject.agents_doc.compute_body"))
	b.WriteString("- engram search, engram recall\n- engram resurface, engram bridge, engram digest\n- engram backlinks\n\n")
	b.WriteString(i18n.T("eject.agents_doc.lint_body"))
	b.WriteString(i18n.T("eject.agents_doc.dirs_heading"))
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, i18n.T("eject.agents_doc.dirs_row"), pair[1], pair[0])
	}
	fmt.Fprintf(&b, i18n.T("eject.agents_doc.root_files"), strings.Join(cfg.RootFiles, ", "))
	fmt.Fprintf(&b, i18n.T("eject.agents_doc.ignore_files"), strings.Join(cfg.IgnoreFiles, ", "))
	b.WriteString(i18n.T("eject.agents_doc.preset_heading"))
	fmt.Fprintf(&b, i18n.T("eject.agents_doc.preset_body"), cfg.Preset)
	b.WriteString(i18n.T("eject.agents_doc.oneway_heading"))
	b.WriteString(i18n.T("eject.agents_doc.oneway_body"))
	return b.String()
}

// gitattributes는 줄바꿈 처리를 정의한다. 린터는 CRLF 도 인식하지만
// 저장소는 단일 형태를 유지하는 것이 비교와 병합에 좋다. 주석도
// 시점 언어로 굳는다.
func gitattributes() string {
	return i18n.T("eject.gitattributes.comment") +
		"* text=auto\n" +
		"*.md text eol=lf\n" +
		"*.png binary\n" +
		"*.jpg binary\n"
}
