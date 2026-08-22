// measure는 whisper 모델 둘을 같은 오디오로 돌려 비교한다.
//
// ADR 0079가 기본 모델 크기를 미결로 두면서 "실제 한국어 녹음으로 두
// 모델을 돌려 비교하는 것을 구현의 첫 단계로 둔다"고 적었다. 이 커맨드가
// 그 측정이다. 결정이 서면 지운다.
//
// 오디오는 16kHz 모노 wav 만 받는다. 변환은 호출자가 한다.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/neocode24/engram/voice/internal/stt"
)

func main() {
	var (
		wav     = flag.String("wav", "", "16kHz 모노 wav 경로")
		dir     = flag.String("model-dir", "", "encoder/decoder/tokens 가 있는 디렉토리")
		prefix  = flag.String("prefix", "", "모델 파일 접두사(medium, large-v3)")
		vad     = flag.String("vad", "", "silero_vad.onnx 경로")
		threads = flag.Int("threads", 4, "인코딩 스레드 수")
		out     = flag.String("out", "", "결과 JSON 경로. 비우면 표준 출력")
		diar    = flag.Bool("diarize", false, "전사 대신 화자 분할을 잰다")
		nspk    = flag.Int("speakers", 0, "화자 수. 0 이면 자동 추정")
		thr     = flag.Float64("threshold", 0, "자동 추정의 군집 임계값. 0 이면 기본값")
		ratio   = flag.Float64("min-speech-ratio", 0, "파편 필터 하한. 0 이면 기본값")
	)
	flag.Parse()
	if *wav == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "--wav 와 --model-dir 가 필요합니다")
		os.Exit(2)
	}
	if !*diar && (*prefix == "" || *vad == "") {
		fmt.Fprintln(os.Stderr, "전사에는 --prefix 와 --vad 가 더 필요합니다")
		os.Exit(2)
	}

	samples, rate, err := stt.ReadWAV(*wav)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wav 읽기 실패:", err)
		os.Exit(1)
	}
	if rate != 16000 {
		fmt.Fprintf(os.Stderr, "16000Hz 가 아닙니다: %d\n", rate)
		os.Exit(1)
	}
	audioSeconds := float64(len(samples)) / float64(rate)

	if *diar {
		if err := runDiarize(samples, rate, *dir, *nspk, *thr, *ratio, *out); err != nil {
			fmt.Fprintln(os.Stderr, "화자 분할 실패:", err)
			os.Exit(1)
		}
		return
	}

	// 구간 나누기를 먼저 끝낸다. 두 모델이 완전히 같은 입력을 받아야
	// 차이가 모델의 것이 된다.
	t0 := time.Now()
	segs, err := stt.Segment(samples, rate, *vad)
	if err != nil {
		fmt.Fprintln(os.Stderr, "구간 나누기 실패:", err)
		os.Exit(1)
	}
	vadElapsed := time.Since(t0)

	cfg := sherpa.OfflineRecognizerConfig{}
	cfg.ModelConfig.Whisper.Encoder = *dir + "/" + *prefix + "-encoder.int8.onnx"
	cfg.ModelConfig.Whisper.Decoder = *dir + "/" + *prefix + "-decoder.int8.onnx"
	cfg.ModelConfig.Whisper.Language = "ko"
	cfg.ModelConfig.Whisper.Task = "transcribe"
	cfg.ModelConfig.Tokens = *dir + "/" + *prefix + "-tokens.txt"
	cfg.ModelConfig.NumThreads = *threads
	cfg.ModelConfig.Debug = 0
	cfg.DecodingMethod = "greedy_search"

	t1 := time.Now()
	rec := sherpa.NewOfflineRecognizer(&cfg)
	if rec == nil {
		fmt.Fprintln(os.Stderr, "인식기를 만들 수 없습니다. 모델 경로를 확인하세요")
		os.Exit(1)
	}
	defer sherpa.DeleteOfflineRecognizer(rec)
	loadElapsed := time.Since(t1)

	type segOut struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	}
	res := struct {
		Model        string   `json:"model"`
		Threads      int      `json:"threads"`
		AudioSeconds float64  `json:"audioSeconds"`
		Segments     int      `json:"segments"`
		VadSeconds   float64  `json:"vadSeconds"`
		LoadSeconds  float64  `json:"loadSeconds"`
		DecodeSecond float64  `json:"decodeSeconds"`
		RealtimeX    float64  `json:"realtimeFactor"`
		Text         string   `json:"text"`
		Out          []segOut `json:"segments_detail"`
	}{Model: *prefix, Threads: *threads, AudioSeconds: audioSeconds,
		Segments: len(segs), VadSeconds: vadElapsed.Seconds(), LoadSeconds: loadElapsed.Seconds()}

	t2 := time.Now()
	full := ""
	for _, s := range segs {
		stream := sherpa.NewOfflineStream(rec)
		stream.AcceptWaveform(rate, s.Samples)
		rec.Decode(stream)
		text := rec2text(stream)
		sherpa.DeleteOfflineStream(stream)
		if text == "" {
			continue
		}
		res.Out = append(res.Out, segOut{Start: s.Start, End: s.End, Text: text})
		full += text + " "
	}
	res.DecodeSecond = time.Since(t2).Seconds()
	res.RealtimeX = audioSeconds / res.DecodeSecond
	res.Text = full

	enc := json.NewEncoder(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "결과 파일을 만들 수 없음:", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		enc = json.NewEncoder(f)
	}
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintln(os.Stderr, "결과를 쓸 수 없음:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%s: 오디오 %.1f초, 구간 %d개, 디코딩 %.1f초, 실시간 배속 %.2fx\n",
		*prefix, audioSeconds, len(segs), res.DecodeSecond, res.RealtimeX)
}

// rec2text는 스트림 결과에서 텍스트만 꺼낸다.
func rec2text(s *sherpa.OfflineStream) string {
	r := s.GetResult()
	if r == nil {
		return ""
	}
	return r.Text
}
