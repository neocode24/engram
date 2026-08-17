// Package gitdate는 git 이력에서 문서별 커밋 날짜를 얻는다. 커맨드가
// 아니라 순수 조회 계층이며 sync 가 부른다. 문서 수만큼 프로세스를
// 띄우지 않도록 전체 이력을 한 번의 git log 로 얻는다.
package gitdate

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Dates는 문서 하나의 커밋 날짜다. 이력이 없으면 빈 값이다.
type Dates struct {
	First string // 최초 커밋 날짜. YYYY-MM-DD
	Last  string // 마지막 커밋 날짜. YYYY-MM-DD
}

// History는 위키 루트 아래 문서들의 커밋 날짜를 한 번의 git log 로 얻는다.
// 반환 맵의 키는 위키 루트 기준 슬래시 경로다. 위키 루트가 저장소의 하위
// 디렉토리일 수 있으므로 저장소 기준 경로 맞춤을 여기서 끝낸다.
// git 이 없거나 위키가 저장소가 아니면 안내를 담은 에러를 낸다.
func History(wikiRoot string) (map[string]Dates, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git 을 찾을 수 없습니다. git 을 설치한 뒤 다시 시도하세요")
	}
	prefixRaw, err := runGit(wikiRoot, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, fmt.Errorf("위키가 git 저장소가 아닙니다: %s\n위키 루트에서 git init 으로 저장소를 만들고 커밋을 쌓으세요", wikiRoot)
	}
	prefix := strings.TrimSpace(prefixRaw)

	out, err := runGit(wikiRoot, "log", "--name-only", "--no-renames", "--format=%x1e%cI")
	if err != nil {
		// 커밋이 하나도 없는 저장소는 이력이 비었을 뿐 실패가 아니다.
		// 모든 문서가 커밋되지 않음으로 건너뛰어진다.
		if strings.Contains(err.Error(), "does not have any commits") {
			return map[string]Dates{}, nil
		}
		return nil, fmt.Errorf("git 이력을 읽을 수 없음: %w", err)
	}

	hist := map[string]Dates{}
	date := ""
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "\x1e") {
			date = day(line[1:])
			continue
		}
		// 파일 경로 줄이다. 위키 밖의 경로는 버린다.
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rel := strings.TrimPrefix(line, prefix)
		d, seen := hist[rel]
		if !seen {
			// git log 는 새 커밋부터 내보내므로 처음 만난 날짜가 마지막 커밋이다.
			d.Last = date
		}
		// 끝까지 덮어 쓰면 마지막에 만난 값, 곧 최초 커밋이 남는다.
		d.First = date
		hist[rel] = d
	}
	return hist, nil
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
		return "", fmt.Errorf("git %s 실패: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
