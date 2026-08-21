package stt

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writeWAV는 시험용 16비트 PCM wav 를 만든다. extra 가 참이면 data
// 앞에 모르는 청크를 하나 끼워 넣는다. 실제 파일에는 LIST 나 fact
// 같은 청크가 흔히 있고, 그것을 건너뛰지 못하면 파일을 못 읽는다.
func writeWAV(t *testing.T, rate, channels, bits int, samples []int16, extra bool) string {
	t.Helper()
	data := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(s))
	}
	var body []byte
	fmtChunk := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtChunk[0:], 1)
	binary.LittleEndian.PutUint16(fmtChunk[2:], uint16(channels))
	binary.LittleEndian.PutUint32(fmtChunk[4:], uint32(rate))
	binary.LittleEndian.PutUint16(fmtChunk[14:], uint16(bits))
	body = append(body, chunk("fmt ", fmtChunk)...)
	if extra {
		// 홀수 크기다. 패딩 1바이트를 건너뛰지 못하면 뒤가 깨진다.
		body = append(body, chunk("LIST", []byte("abc"))...)
		body = append(body, 0)
	}
	body = append(body, chunk("data", data)...)

	out := []byte("RIFF")
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(4+len(body)))
	out = append(out, size...)
	out = append(out, []byte("WAVE")...)
	out = append(out, body...)

	p := filepath.Join(t.TempDir(), "a.wav")
	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func chunk(id string, payload []byte) []byte {
	out := []byte(id)
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(len(payload)))
	out = append(out, size...)
	return append(out, payload...)
}

func TestReadWAV(t *testing.T) {
	samples := []int16{0, 32767, -32768, 16384}
	path := writeWAV(t, 16000, 1, 16, samples, false)
	got, rate, err := ReadWAV(path)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 16000 {
		t.Errorf("표본율이 16000 이어야 함: %d", rate)
	}
	if len(got) != len(samples) {
		t.Fatalf("표본 수가 %d 여야 함: %d", len(samples), len(got))
	}
	want := []float32{0, 32767.0 / 32768.0, -1, 0.5}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("표본 %d 가 %v 여야 함: %v", i, want[i], got[i])
		}
	}
}

func TestReadWAVSkipsUnknownChunks(t *testing.T) {
	path := writeWAV(t, 16000, 1, 16, []int16{1, 2, 3}, true)
	got, _, err := ReadWAV(path)
	if err != nil {
		t.Fatalf("모르는 청크는 건너뛰어야 함: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("표본 수가 3 이어야 함: %d", len(got))
	}
}

func TestReadWAVRejectsUnsupported(t *testing.T) {
	// 받는 형식이 하나뿐이므로 나머지는 거절하고 변환은 호출자에게
	// 맡긴다. 조용히 잘못 읽으면 전사가 통째로 망가진다.
	t.Run("스테레오", func(t *testing.T) {
		if _, _, err := ReadWAV(writeWAV(t, 16000, 2, 16, []int16{1, 2}, false)); err == nil {
			t.Error("모노가 아니면 거절해야 함")
		}
	})
	t.Run("8비트", func(t *testing.T) {
		if _, _, err := ReadWAV(writeWAV(t, 16000, 1, 8, []int16{1, 2}, false)); err == nil {
			t.Error("16비트가 아니면 거절해야 함")
		}
	})
	t.Run("wav 아님", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "b.wav")
		if err := os.WriteFile(p, []byte("이건 wav 가 아니다"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ReadWAV(p); err == nil {
			t.Error("RIFF/WAVE 가 아니면 거절해야 함")
		}
	})
}
