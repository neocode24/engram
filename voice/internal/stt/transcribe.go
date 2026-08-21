package stt

import (
	"errors"
	"path/filepath"
	"strings"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/neocode24/engram/voice/internal/model"
)

// Line은 전사 한 줄이다. Speaker 가 Unknown 이면 화자를 붙이지 못한
// 것이고 그것은 오류가 아니라 모른다는 뜻이다(ADR 0082).
type Line struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker int     `json:"speaker"`
	Text    string  `json:"text"`
}

// 전사 상수.
const (
	// decodeThreads는 인코더와 디코더에 쓰는 스레드 수다.
	decodeThreads = 6
	// language는 whisper 에 넘기는 언어다. 자동 감지에 맡기지 않는다.
	// 짧은 구간에서 언어를 잘못 잡으면 그 구간이 통째로 엉뚱한 글자가
	// 되고, 위키가 한국어라 감지가 줄 이득이 없다.
	language = "ko"
)

// Transcriber는 열린 모델을 쥔다. 구간마다 다시 여는 것을 막기 위해
// 있다. 모델 적재가 구간 하나 디코딩보다 비싸다.
type Transcriber struct {
	rec *sherpa.OfflineRecognizer
	dir string
}

// Open은 모델 디렉토리에서 전사기를 연다. 닫는 것은 호출자 몫이다.
func Open(dir string) (*Transcriber, error) {
	cfg := sherpa.OfflineRecognizerConfig{}
	cfg.ModelConfig.Whisper.Encoder = filepath.Join(dir, model.EncoderName)
	cfg.ModelConfig.Whisper.Decoder = filepath.Join(dir, model.DecoderName)
	cfg.ModelConfig.Whisper.Language = language
	cfg.ModelConfig.Whisper.Task = "transcribe"
	cfg.ModelConfig.Tokens = filepath.Join(dir, model.TokensName)
	cfg.ModelConfig.NumThreads = decodeThreads
	cfg.DecodingMethod = "greedy_search"

	rec := sherpa.NewOfflineRecognizer(&cfg)
	if rec == nil {
		return nil, errors.New("전사기를 만들 수 없습니다. engram-voice model status 로 모델을 확인하세요")
	}
	return &Transcriber{rec: rec, dir: dir}, nil
}

// Close는 모델을 놓는다.
func (t *Transcriber) Close() {
	if t.rec != nil {
		sherpa.DeleteOfflineRecognizer(t.rec)
		t.rec = nil
	}
}

// Progress는 전사 진행률 콜백이다. done 이 total 에 이르면 끝이다.
type Progress func(done, total int)

// Transcribe는 구간 목록을 전사한다. speakers 가 비어 있지 않으면
// 구간마다 화자를 붙인다.
//
// 빈 결과가 나온 구간은 버린다. whisper 가 침묵이나 잡음에 빈 문자열을
// 내는데 그것을 줄로 남기면 전사가 빈 줄로 뒤덮인다.
func (t *Transcriber) Transcribe(segs []Seg, speakers []Speaker, prog Progress) []Line {
	out := make([]Line, 0, len(segs))
	for i, s := range segs {
		stream := sherpa.NewOfflineStream(t.rec)
		stream.AcceptWaveform(SampleRate, s.Samples)
		t.rec.Decode(stream)
		text := ""
		if r := stream.GetResult(); r != nil {
			text = strings.TrimSpace(r.Text)
		}
		sherpa.DeleteOfflineStream(stream)
		if prog != nil {
			prog(i+1, len(segs))
		}
		if text == "" {
			continue
		}
		line := Line{Start: s.Start, End: s.End, Speaker: Unknown, Text: text}
		if len(speakers) > 0 {
			line.Speaker = AssignSpeaker(s.Start, s.End, speakers)
		}
		out = append(out, line)
	}
	return out
}

// SampleRate는 전사가 요구하는 표본율이다.
const SampleRate = 16000

// MergeAdjacent는 화자가 같고 이어지는 줄을 하나로 합치되 합친 줄이
// maxLine 초를 넘지 않게 한다.
//
// 구간 나누기가 침묵을 기준으로 하므로 한 사람이 쉬어 가며 말한 것이
// 여러 줄이 된다. 읽는 사람에게는 한 발화다.
//
// **길이 상한이 필요한 이유는 구간 사이 간격이 항상 0이기 때문이다.**
// Segment 가 오디오 전체를 덮는 구간으로 나누므로 앞 구간의 끝이 다음
// 구간의 시작이다. 간격만 보고 합치면 "같은 화자면 무조건 합침" 이
// 되고, 실측에서 6.7분 대화가 다섯 줄이 되었다. 읽을 수 없다.
//
// 상한은 읽기 편한 문단 길이이고 정확성 주장이 아니다.
func MergeAdjacent(lines []Line, maxLine float64) []Line {
	if len(lines) == 0 {
		return lines
	}
	out := []Line{lines[0]}
	for _, l := range lines[1:] {
		last := &out[len(out)-1]
		if l.Speaker == last.Speaker && l.End-last.Start <= maxLine {
			last.End = l.End
			last.Text += " " + l.Text
			continue
		}
		out = append(out, l)
	}
	return out
}
