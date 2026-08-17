// Package skills는 에이전트가 engram 을 다루는 법을 가르치는 정적 스킬
// 문서를 임베드하고 설치 대상을 고른다. 문서는 위키와 무관하므로 한 번
// 심으면 모든 위키에 통한다(ADR 0041). 위키별 규칙은 에이전트가
// engram rules show 로 얻는다.
package skills

import (
	_ "embed"
	"os"
	"path/filepath"
	"sort"

	"github.com/neocode24/engram/internal/i18n"
)

// 임베드된 스킬 문서. Go 소스에 문자열로 박지 않고 마크다운 파일 하나를
// 그대로 싣는다. 사람이 읽고 고칠 수 있어야 하기 때문이다.
//
//go:embed SKILL.md
var skillDoc string

// FileName는 설치되는 파일 이름이다. 스킬 문서의 통용 형식을 따른다.
const FileName = "SKILL.md"

// SkillName은 스킬 하나가 차지하는 디렉토리 이름이다.
const SkillName = "engram"

// Doc은 임베드된 스킬 문서를 반환한다.
func Doc() string {
	return skillDoc
}

// candidate는 설치 대상 후보와 그 규약의 출처다.
type candidate struct {
	rel  string // 홈 디렉토리 기준 상대 경로
	note string // 어느 도구의 규약인지를 알리는 메시지 ID
}

// candidates는 설치 대상 후보 목록이다. 실제로 존재하는 디렉토리만
// 대상이 되고 없으면 만들지 않는다. 쓰지 않는 도구를 위해 홈에 빈
// 디렉토리를 남기지 않기 위해서다. 규약이 바뀌면 이 목록만 고친다.
//
// 근거가 없는 경로는 넣지 않았다. 후보에 올렸다가 뺀 것은 보고를
// 참고한다.
var candidates = []candidate{
	{".claude/skills", "core.skills.note_claude"},
	{".agents/skills", "core.skills.note_agents"},
}

// Detect는 홈 디렉토리에서 실제로 존재하는 설치 대상을 절대 경로로
// 반환한다. 심볼릭 링크로 같은 디렉토리를 가리키는 대상이 여럿이면
// 하나로 합친다. 두 도구가 같은 물리적 디렉토리를 공유하는 배치가
// 있어서다. 순서는 candidates 순서를 따르므로 결정론적이다.
func Detect(home string) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		dir := filepath.Join(home, filepath.FromSlash(c.rel))
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			continue
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		out = append(out, dir)
	}
	return out
}

// Sources는 후보 경로와 출처를 정렬해 낸다. 감지 결과가 비었을 때
// 사용자에게 어떤 경로를 보고 있는지 알리는 데 쓴다.
func Sources() []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.rel+" ("+i18n.T(c.note)+")")
	}
	sort.Strings(out)
	return out
}

// InstallPath는 스킬 루트 안에 스킬 문서가 놓일 경로를 반환한다.
func InstallPath(skillsRoot string) string {
	return filepath.Join(skillsRoot, SkillName, FileName)
}
