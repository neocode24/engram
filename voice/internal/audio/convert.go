// Package audio는 오디오 파일을 전사에 넣을 형태로 바꾼다.
//
// 변환을 직접 하지 않고 바깥 도구를 부른다. m4a 와 mp3 를 순수 Go 로
// 디코딩하는 길이 없고, 그것을 위해 코덱을 하나 더 들이면 이 바이너리가
// 지고 있는 네이티브 의존이 늘어난다(ADR 0081). macOS 는 afconvert 가
// 늘 있고 나머지는 ffmpeg 를 요구한다. upstream 도 같은 둘을 쓴다.
package audio

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SampleRate는 전사와 화자 분할이 요구하는 표본율이다. 둘 다 이 값
// 하나만 받는다.
const SampleRate = 16000

// ErrNoConverter는 쓸 변환기가 없을 때의 오류다.
var ErrNoConverter = errors.New("오디오 변환기 없음")

// IsWAV는 경로가 wav 인지 확장자로 본다. 확장자가 wav 여도 형식이
// 다를 수 있으므로 판정은 결국 읽는 쪽이 한다. 여기서는 변환을
// 건너뛸지만 정한다.
func IsWAV(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".wav")
}

// Converter는 쓸 수 있는 변환기의 이름과 경로다.
type Converter struct {
	Name string
	Path string
}

// FindConverter는 쓸 변환기를 고른다. afconvert 를 먼저 보는 이유는
// macOS 에 늘 있고 Voice Memos 가 내는 m4a 를 그 플랫폼의 코덱으로
// 그대로 풀기 때문이다.
func FindConverter() (Converter, error) {
	for _, name := range []string{"afconvert", "ffmpeg"} {
		if p, err := exec.LookPath(name); err == nil {
			return Converter{Name: name, Path: p}, nil
		}
	}
	return Converter{}, fmt.Errorf("%w: afconvert 나 ffmpeg 가 PATH 에 있어야 합니다", ErrNoConverter)
}

// ToWAV는 src 를 16kHz 모노 16비트 wav 로 바꿔 dst 에 쓴다. src 가
// 이미 그 형태여도 변환기를 거친다. 확장자만 보고 내용을 믿지 않는다.
func ToWAV(c Converter, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch c.Name {
	case "afconvert":
		// LEI16 은 리틀엔디언 16비트 정수다. -c 1 이 모노,
		// -r 이 표본율이다.
		cmd = exec.Command(c.Path, src, dst,
			"-f", "WAVE", "-d", fmt.Sprintf("LEI16@%d", SampleRate), "-c", "1")
	case "ffmpeg":
		cmd = exec.Command(c.Path, "-y", "-i", src,
			"-ac", "1", "-ar", fmt.Sprintf("%d", SampleRate),
			"-c:a", "pcm_s16le", "-loglevel", "error", dst)
	default:
		return fmt.Errorf("%w: %s", ErrNoConverter, c.Name)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		// 변환기의 말을 그대로 넘긴다. 우리가 다시 쓰면 원인이 흐려진다.
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("%s 변환 실패: %w", c.Name, err)
		}
		return fmt.Errorf("%s 변환 실패: %w\n%s", c.Name, err, msg)
	}
	if fi, err := os.Stat(dst); err != nil || fi.Size() == 0 {
		return fmt.Errorf("%s 가 빈 파일을 냈습니다: %s", c.Name, dst)
	}
	return nil
}
