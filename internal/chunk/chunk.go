// Package chunk는 마크다운 본문을 헤딩 단위 조각으로 자른다.
// 헤딩이 저자가 이미 그은 경계이므로 도구가 새 경계를 발명하지 않는다.
// ADR 0028.
package chunk

import "strings"

// Chunk는 본문을 자른 조각 하나다.
type Chunk struct {
	Heading   string   // 이 조각을 여는 헤딩 텍스트. 헤딩이 없으면 빈 문자열
	Path      []string // 상위 헤딩 누적 경로. 이 조각 자신의 헤딩은 빠진다
	StartLine int      // 입력 본문 기준 1 기반 시작 줄. 파일 전체 기준이 아니다
	EndLine   int      // 입력 본문 기준 1 기반 끝 줄. 끝 줄을 포함한다
	Body      string   // 헤딩 줄을 포함한 원문 그대로
}

// Split은 프론트매터를 뗀 본문을 헤딩 단위 조각으로 자른다.
//
// 경계는 ## 이상(##, ###, #### ...) 헤딩이다. # 하나짜리 문서 제목은
// 절 구분이 아니므로 경계로 쓰지 않는다. 코드 펜스(```) 안의 # 는
// 헤딩이 아니다. internal/doc 의 링크 추출과 같은 펜스 규칙을 쓴다.
// 첫 헤딩 앞 본문도 조각 하나이며 이때 Heading 은 빈 문자열이고
// Path 는 빈 슬라이스다. 헤딩이 하나도 없으면 문서 전체가 조각 하나다.
//
// 조각을 이어 붙이면 입력 본문이 그대로 돌아온다. 줄 번호는 입력 본문
// 기준이므로 파일 기준 줄 번호가 필요하면 호출자가 프론트매터 줄
// 수만큼 더한다.
func Split(body string) []Chunk {
	if body == "" {
		return nil
	}
	lines := strings.Split(body, "\n")
	// 입력이 개행으로 끝나면 Split 이 마지막에 빈 줄 하나를 더 낸다.
	// 그것은 실제 줄이 아니므로 실줄 수에서 뺀다.
	realCount := len(lines)
	trailing := strings.HasSuffix(body, "\n")
	if trailing {
		realCount--
	}

	var chunks []Chunk
	// 열린 상위 헤딩 스택. path 는 헤딩 텍스트, levels 는 그 레벨이다.
	var path []string
	var levels []int
	start := 1 // 열린 조각의 첫 줄. 1 기반
	chunkHeading := ""
	chunkPath := []string{}

	flush := func(end int) {
		if end < start {
			return
		}
		text := strings.Join(lines[start-1:end], "\n")
		if end < realCount || trailing {
			text += "\n"
		}
		chunks = append(chunks, Chunk{
			Heading: chunkHeading, Path: chunkPath,
			StartLine: start, EndLine: end, Body: text,
		})
	}

	inFence := false
	for i := 0; i < realCount; i++ {
		line := lines[i]
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			// 펜스 줄 자체는 조각 본문에 속한다. 헤딩 판정에서만 건너뛴다.
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		level, text, ok := heading(line)
		if !ok {
			continue
		}
		flush(i)
		// 이 헤딩보 깊거나 같은 상위 헤딩은 닫는다.
		for len(levels) > 0 && levels[len(levels)-1] >= level {
			levels = levels[:len(levels)-1]
			path = path[:len(path)-1]
		}
		chunkPath = append([]string{}, path...)
		chunkHeading = text
		levels = append(levels, level)
		path = append(path, text)
		start = i + 1
	}
	flush(realCount)
	return chunks
}

// heading은 줄이 ## 이상 헤딩이면 레벨과 텍스트를 반환한다.
// # 하나짜리 문서 제목은 경계가 아니므로 여기서 걸러진다. 헤딩
// 표시 뒤에는 공백이 와야 하며 ## 처럼 텍스트 없이 끝난 줄은
// 헤딩으로 치지 않는다.
func heading(line string) (level int, text string, ok bool) {
	t := strings.TrimLeft(line, " \t")
	for level = 0; level < len(t); level++ {
		if t[level] != '#' {
			break
		}
	}
	if level < 2 || level >= len(t) {
		return 0, "", false
	}
	if c := t[level]; c != ' ' && c != '\t' {
		return 0, "", false
	}
	return level, strings.TrimSpace(t[level:]), true
}
