// Package gitdate는 git 이력에서 문서별 커밋 날짜를 얻는다. 커맨드가
// 아니라 순수 조회 계층이며 sync 가 부른다. 문서 수만큼 프로세스를
// 띄우지 않도록 전체 이력을 한 번의 git log 로 얻는다.
package gitdate

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/neocode24/engram/internal/i18n"
)

// BulkCommitThreshold는 한 커밋이 건드린 위키 문서 수의 상한이다. 이
// 수 이상을 건드린 커밋은 지식이 갱신된 것이 아니라 형식만 바꾼 것으로
// 보고 날짜 신호에서 뺀다.
//
// upstream 이 실제로 겪은 사고에서 온 값이다. 전 문서의 프론트매터를
// 한 커밋에 고치자 context 79개의 날짜가 전부 그날이 되어 재발견 결과가
// 0건이 되었고, 이 필터를 켜서 58건이 회복되었다. engram 에도 같은
// 경로가 있다. migrate 는 정의상 전 문서의 프론트매터를 한 커밋에
// 고치는 커맨드이고 그 뒤에 sync --apply 를 돌리면 같은 일이 난다.
//
// 설정으로 노출하지 않는다. 색인의 k1 과 b 를 노출하지 않기로 한 것과
// 같은 이유다(ADR 0010). upstream 과 같은 값을 쓴다.
const BulkCommitThreshold = 15

// Dates는 문서 하나의 커밋 날짜다. 이력이 없으면 빈 값이다.
type Dates struct {
	First    string // 최초 커밋 날짜. YYYY-MM-DD
	Last     string // 마지막 커밋 날짜. YYYY-MM-DD
	BulkOnly bool   // 대량 커밋에만 등장해 날짜를 비워 둔 문서다
}

// History는 위키 루트 아래 문서들의 커밋 날짜를 한 번의 git log 로 얻는다.
// 반환 맵의 키는 위키 루트 기준 슬래시 경로다. 위키 루트가 저장소의 하위
// 디렉토리일 수 있으므로 저장소 기준 경로 맞춤을 여기서 끝낸다.
// git 이 없거나 위키가 저장소가 아니면 안내를 담은 에러를 낸다.
func History(wikiRoot string) (map[string]Dates, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, errors.New(i18n.T("core.gitdate.git_missing"))
	}
	prefixRaw, err := runGit(wikiRoot, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, errors.New(i18n.T("core.gitdate.not_repo", wikiRoot))
	}
	prefix := strings.TrimSpace(prefixRaw)

	out, err := runGit(wikiRoot, "log", "--name-only", "--no-renames", "--format=%x1e%cI")
	if err != nil {
		// 커밋이 하나도 없는 저장소는 이력이 비었을 뿐 실패가 아니다.
		// 모든 문서가 커밋되지 않음으로 건너뛰어진다.
		if strings.Contains(err.Error(), "does not have any commits") {
			return map[string]Dates{}, nil
		}
		return nil, fmt.Errorf("%s: %w", i18n.T("core.gitdate.read_fail"), err)
	}

	hist := map[string]Dates{}
	bulkOnly := map[string]bool{}
	// git log 는 새 커밋부터 내보내므로 커밋 순서는 최신순이다.
	for _, c := range parseCommits(out, prefix) {
		if c.docs >= BulkCommitThreshold {
			for _, rel := range c.paths {
				bulkOnly[rel] = true
			}
			continue
		}
		for _, rel := range c.paths {
			d, seen := hist[rel]
			if !seen {
				// 처음 만난 날짜가 마지막 커밋이다.
				d.Last = c.date
			}
			// 끝까지 덮어 쓰면 마지막에 만난 값, 곧 최초 커밋이 남는다.
			d.First = c.date
			hist[rel] = d
		}
	}
	// 커밋이 전부 대량 커밋인 문서는 날짜를 없는 것으로 둔다. 대량
	// 커밋의 날짜로 대신 채우지 않는다. 없는 날짜를 만드는 것보다
	// 없다고 두는 것이 낫다. 부른 쪽이 개수를 알릴 수 있게 표시만 남긴다.
	for rel := range bulkOnly {
		if _, ok := hist[rel]; !ok {
			hist[rel] = Dates{BulkOnly: true}
		}
	}
	return hist, nil
}

// commit은 커밋 하나가 건드린 위키 안 경로와 그 날짜다. docs 는 그중
// 마크다운 문서 수다.
type commit struct {
	date  string
	paths []string
	docs  int
}

// parseCommits는 git log --name-only 출력을 커밋 단위로 끊는다. 위키
// 밖의 경로는 버린다.
//
// 대량 여부를 세는 대상은 커밋이 건드린 파일 전부가 아니라 위키의
// 마크다운 문서다. 설정 파일이나 스크립트를 함께 고친 커밋이 억울하게
// 걸리면 안 된다.
func parseCommits(out, prefix string) []commit {
	var commits []commit
	cur := commit{}
	started := false
	flush := func() {
		if started {
			commits = append(commits, cur)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "\x1e") {
			flush()
			cur = commit{date: day(line[1:])}
			started = true
			continue
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rel := strings.TrimPrefix(line, prefix)
		cur.paths = append(cur.paths, rel)
		if strings.HasSuffix(rel, ".md") {
			cur.docs++
		}
	}
	flush()
	return commits
}

// day는 ISO 8601 시각에서 날짜 부분만 남긴다. 형식이 짧으면 빈 값을 둔다.
func day(iso string) string {
	if len(iso) < 10 {
		return ""
	}
	return iso[:10]
}

// runGit는 git 을 위키 루트에서 돌리고 표준 출력을 반환한다. 프로세스의
// 작업 디렉토리는 바꾸지 않는다.
//
// core.quotepath 를 끈다. git 의 기본값은 켜짐이고, 켜져 있으면 비ASCII
// 경로를 따옴표로 감싸고 8진 이스케이프로 바꿔서 낸다. 한글 파일명이
// "sources/\354\233\220\353\263\270.md" 로 나오면 위키 문서 경로와
// 대조가 되지 않아 sync 가 그 문서를 조용히 건너뛴다. 한국어 위키가 주
// 대상이므로 여기서 못 박는다. 사용자 설정에 기대지 않는다.
func runGit(root string, args ...string) (string, error) {
	full := append([]string{"-C", root, "-c", "core.quotepath=false"}, args...)
	cmd := exec.Command("git", full...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", i18n.T("core.gitdate.git_fail", strings.Join(args, " ")), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
