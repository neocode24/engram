package main

// 화자 분할을 재는 자리다. measure 커맨드의 --diarize 로 켠다.
// 결정이 서면 measure 와 함께 지운다.

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/neocode24/engram/voice/internal/stt"
)

// runDiarize는 표본에서 화자를 갈라 결과를 낸다.
func runDiarize(samples []float32, rate int, dir string, speakers int, threshold float64, out string) error {
	t0 := time.Now()
	segs, err := stt.Diarize(samples, rate, dir, stt.DiarizeOptions{Speakers: speakers, Threshold: float32(threshold)})
	if err != nil {
		return err
	}
	elapsed := time.Since(t0)
	audio := float64(len(samples)) / float64(rate)

	var covered float64
	for _, s := range segs {
		covered += s.End - s.Start
	}
	res := struct {
		AudioSeconds  float64       `json:"audioSeconds"`
		Seconds       float64       `json:"diarizeSeconds"`
		RealtimeX     float64       `json:"realtimeFactor"`
		Segments      int           `json:"segments"`
		Speakers      int           `json:"speakers"`
		Unknown       int           `json:"unknownSegments"`
		CoveredSecond float64       `json:"coveredSeconds"`
		CoveredPct    float64       `json:"coveredPercent"`
		Detail        []stt.Speaker `json:"detail"`
	}{
		AudioSeconds: audio, Seconds: elapsed.Seconds(),
		RealtimeX: audio / elapsed.Seconds(), Segments: len(segs),
		Speakers: stt.CountSpeakers(segs), Unknown: stt.CountUnknown(segs), CoveredSecond: covered,
		CoveredPct: covered / audio * 100, Detail: segs,
	}
	fmt.Fprintf(os.Stderr, "화자 분할: %.1f초, 구간 %d개, 화자 %d명, 미상 %d구간, 덮은 비율 %.1f%%, 실시간 %.2fx\n",
		elapsed.Seconds(), len(segs), res.Speakers, res.Unknown, res.CoveredPct, res.RealtimeX)

	w := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		w = f
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}
