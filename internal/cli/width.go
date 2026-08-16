package cli

import "strings"

// displayWidth는 문자열의 터미널 표시 폭을 센다. 룬 하나를 기본 한 칸으로
// 세고 동아시아 넓은 문자를 두 칸으로 센다. Go 의 fmt 폭은 룬 수를 세므로
// 한글이 섞인 열을 %-*s 로 맞추면 실제 화면에서 어긋난다. 표 정렬의 폭은
// 이 함수로 계산한다.
//
// 넓은 문자 범위는 이 저장소의 출력에 실제로 나오는 범위만 다룬다.
// 한글 음절과 호환 자모, 한중일 통합 한자, 전각 기호, 히라가나,
// 가타카나다. 완벽한 East Asian Width 구현이 아니다. 결합 문자와
// 이모지는 다루지 않는다. 이 저장소는 이모지를 쓰지 않는다.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if isWide(r) {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// isWide는 룬이 터미널에서 두 칸을 차지하는 동아시아 넓은 문자인지
// 검사한다.
func isWide(r rune) bool {
	switch {
	case r >= 0xAC00 && r <= 0xD7A3: // 한글 음절
		return true
	case r >= 0x3131 && r <= 0x3163: // 한글 호환 자모
		return true
	case r >= 0x4E00 && r <= 0x9FFF: // 한중일 통합 한자
		return true
	case r >= 0x3000 && r <= 0x30FF: // 전각 기호, 히라가나, 가타카나
		return true
	case r >= 0xFF00 && r <= 0xFF60: // 전각 형식
		return true
	}
	return false
}

// padRight는 문자열을 표시 폭 width 에 맞춰 오른쪽에 공백을 채운다.
// 문자열이 width 보다 넓으면 그대로 반환한다.
func padRight(s string, width int) string {
	if gap := width - displayWidth(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}
