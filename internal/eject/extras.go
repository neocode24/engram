package eject

import (
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
)

// preCommitHook는 생성된 린터를 부르는 git 훅이다. engram 저장소의 훅과
// 같은 구조를 쓴다.
func preCommitHook() string {
	return `#!/bin/sh
# 커밋 전 문서 규칙 검사. engram eject 가 만들었다.
# 활성화: git config core.hooksPath .githooks
set -e

ROOT=$(git rev-parse --show-toplevel)

python3 "$ROOT/scripts/lint-frontmatter.py" "$ROOT" || {
    echo "문서 규칙 위반으로 커밋을 막았습니다. 위반 목록을 고친 뒤 다시 커밋하세요"
    echo "python3 가 없다면 scripts/lint-frontmatter.py 상단의 안내를 확인하세요"
    exit 1
}
`
}

// agentsDoc는 그 위키에서 일하는 에이전트의 작업 계약이다. 규칙은 이제
// 이 위키의 파일이고 연산은 engram 이 계속 수행한다는 사실이 중심이다.
func agentsDoc(cfg config.Config, dirs map[string]string) string {
	var b strings.Builder
	b.WriteString("# AGENTS.md\n\n이 위키에서 작업하는 에이전트는 아래를 따른다.\n\n")
	b.WriteString("## 규칙은 이 위키의 것이다\n\n")
	b.WriteString("문서 규칙의 소유권은 이 위키에 있다. 규칙 명세는 meta/ 디렉토리의 문서에 있고 판정은 scripts/lint-frontmatter.py 가 한다. 규칙을 바꾸려면 그 둘을 직접 고친다. 속성, 허용값, 임계값, 디렉토리는 engram.yaml 이 진실원이므로 스크립트는 그 파일을 실행 시점에 읽는다.\n\n")
	b.WriteString("커밋 전 검사는 .githooks/pre-commit 이 돌린다. 한 번 활성화한다.\n\n    git config core.hooksPath .githooks\n\n")
	b.WriteString("## 연산은 engram 이 계속 수행한다\n\n")
	b.WriteString("eject 는 규칙만 내보냈다. 검색 색인, 재발견, 링크 그래프 계산, 다이제스트는 파일로 표현되지 않는 연산이므로 engram 이 계속 맡는다. 아래 커맨드는 그대로 동작한다.\n\n")
	b.WriteString("- engram search, engram recall\n- engram resurface, engram bridge, engram digest\n- engram backlinks\n\n")
	b.WriteString("engram lint 를 돌리면 내장 규칙으로 검사한다. 위키 단위 진단 wiki.broad-topic 은 스크립트가 내보내지 않으므로 engram lint 만 판정한다.\n\n")
	b.WriteString("## 디렉토리\n\n")
	for _, pair := range stageDirsSorted(dirs) {
		fmt.Fprintf(&b, "- %s/. %s 단계 문서가 사는 곳\n", pair[1], pair[0])
	}
	fmt.Fprintf(&b, "- 루트 파일(root_files): %s\n", strings.Join(cfg.RootFiles, ", "))
	fmt.Fprintf(&b, "- 문서가 아닌 파일(ignore_files): %s\n", strings.Join(cfg.IgnoreFiles, ", "))
	fmt.Fprintf(&b, "\n## 프리셋\n\n이 위키는 %s 프리셋을 쓴다.\n", cfg.Preset)
	b.WriteString("\n## 단방향\n\neject 는 되돌리는 커맨드가 없다. 규칙 파일을 고쳤다면 engram 이 다시 만들면 덮어 쓴다. 되돌리고 싶은 부분은 git 이력으로 본다.\n")
	return b.String()
}

// gitattributes는 줄바꿈 처리를 정의한다. 린터는 CRLF 도 인식하지만
// 저장소는 단일 형태를 유지하는 것이 비교와 병합에 좋다.
func gitattributes() string {
	return `# 줄바꿈을 LF 로 통일한다. 린터는 CRLF 도 인식하지만 저장소는
# 단일 형태를 유지하는 것이 비교와 병합에 좋다.
* text=auto
*.md text eol=lf
*.png binary
*.jpg binary
`
}
