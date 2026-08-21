package stt

import (
	"errors"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// Seg는 전사에 넘길 구간 하나다. Start 와 End 는 초다.
type Seg struct {
	Start   float64
	End     float64
	Samples []float32
}

// 구간 나누기 상수.
const (
	vadThreshold     = 0.35
	vadMinSilence    = 0.35 // 초. 이보다 짧은 침묵은 자를 곳으로 보지 않는다
	vadMinSpeech     = 0.2  // 초
	vadWindow        = 512
	vadMaxSpeech     = 20.0 // 초
	vadBufferSeconds = 60.0
	vadThreads       = 1

	// maxChunk는 한 번에 넘길 구간의 상한이다. whisper 는 30초 창으로
	// 돌고 그보다 긴 입력은 뒤가 잘린다. 여유를 두고 25초로 끊는다.
	maxChunk = 25.0
)

// Segment는 표본 전체를 빠짐없이 덮는 구간들로 나눈다.
//
// **VAD 가 고른 구간만 넘기지 않는다.** 그렇게 하면 VAD 가 놓친 말이
// 통째로 사라진다. 실측에서 그 방식이 402초 오디오의 61%만 덮었고,
// 놓친 39%는 조용하고 알아듣기 어려운 대목에 몰려 있었다. 모델을
// 비교하는 자리에서 그 대목이 빠지면 어려운 입력이 사라져 작은 모델이
// 유리해진다.
//
// 그래서 VAD 는 **자를 곳을 고르는 데만** 쓴다. 침묵 한가운데를 경계로
// 삼아 오디오 전체를 이어붙는 구간으로 나눈다. 침묵이 없어 구간이
// maxChunk 를 넘으면 그 자리에서 균등 분할한다.
func Segment(samples []float32, rate int, vadModel string) ([]Seg, error) {
	cuts, err := silenceCuts(samples, rate, vadModel)
	if err != nil {
		return nil, err
	}
	total := float64(len(samples)) / float64(rate)

	bounds := []float64{0}
	bounds = append(bounds, cuts...)
	bounds = append(bounds, total)

	var out []Seg
	for i := 0; i+1 < len(bounds); i++ {
		start, end := bounds[i], bounds[i+1]
		if end-start <= 0 {
			continue
		}
		// 침묵이 없어 길어진 구간은 균등 분할한다. 낱말 가운데가
		// 잘리지만 이 경우는 애초에 자를 침묵이 없다.
		parts := 1
		for (end-start)/float64(parts) > maxChunk {
			parts++
		}
		step := (end - start) / float64(parts)
		for p := 0; p < parts; p++ {
			s := start + step*float64(p)
			e := s + step
			if p == parts-1 {
				e = end
			}
			si, ei := int(s*float64(rate)), int(e*float64(rate))
			if ei > len(samples) {
				ei = len(samples)
			}
			if ei-si <= 0 {
				continue
			}
			out = append(out, Seg{Start: s, End: e, Samples: samples[si:ei]})
		}
	}
	return out, nil
}

// silenceCuts는 말과 말 사이 침묵의 한가운데를 자를 지점으로 낸다.
// 초 단위 오름차순이다.
func silenceCuts(samples []float32, rate int, vadModel string) ([]float64, error) {
	cfg := sherpa.VadModelConfig{}
	cfg.SileroVad.Model = vadModel
	cfg.SileroVad.Threshold = vadThreshold
	cfg.SileroVad.MinSilenceDuration = vadMinSilence
	cfg.SileroVad.MinSpeechDuration = vadMinSpeech
	cfg.SileroVad.WindowSize = vadWindow
	cfg.SileroVad.MaxSpeechDuration = vadMaxSpeech
	cfg.SampleRate = rate
	cfg.NumThreads = vadThreads

	vad := sherpa.NewVoiceActivityDetector(&cfg, vadBufferSeconds)
	if vad == nil {
		return nil, errors.New("VAD 를 만들 수 없습니다. silero_vad.onnx 경로를 확인하세요")
	}
	defer sherpa.DeleteVoiceActivityDetector(vad)

	type span struct{ start, end float64 }
	var spans []span
	drain := func() {
		for !vad.IsEmpty() {
			s := vad.Front()
			st := float64(s.Start) / float64(rate)
			spans = append(spans, span{st, st + float64(len(s.Samples))/float64(rate)})
			vad.Pop()
		}
	}
	for i := 0; i < len(samples); i += vadWindow {
		end := i + vadWindow
		if end > len(samples) {
			end = len(samples)
		}
		vad.AcceptWaveform(samples[i:end])
		drain()
	}
	vad.Flush()
	drain()

	var cuts []float64
	for i := 0; i+1 < len(spans); i++ {
		gap := spans[i+1].start - spans[i].end
		if gap < vadMinSilence {
			continue
		}
		cuts = append(cuts, spans[i].end+gap/2)
	}
	return cuts, nil
}
