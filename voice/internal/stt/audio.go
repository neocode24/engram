// Package stt는 오디오를 전사 가능한 형태로 다루는 조각들이다.
package stt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// ReadWAV는 16비트 PCM wav 를 [-1, 1] 부동소수 표본으로 읽는다.
// 표본율을 함께 낸다.
//
// 외부 라이브러리를 쓰지 않는다. 받는 형식이 하나뿐이고 헤더가 단순해서
// 의존을 하나 더 들일 이유가 없다. 형식이 다르면 거절하고 변환은
// 호출자에게 맡긴다.
func ReadWAV(path string) ([]float32, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()

	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, 0, fmt.Errorf("헤더를 읽을 수 없음: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, 0, errors.New("wav 파일이 아닙니다")
	}

	var (
		rate     int
		channels int
		bits     int
		data     []byte
	)
	for {
		var hdr [8]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, 0, err
		}
		id := string(hdr[0:4])
		size := int(binary.LittleEndian.Uint32(hdr[4:8]))
		switch id {
		case "fmt ":
			buf := make([]byte, size)
			if _, err := io.ReadFull(f, buf); err != nil {
				return nil, 0, err
			}
			if len(buf) < 16 {
				return nil, 0, errors.New("fmt 청크가 짧습니다")
			}
			channels = int(binary.LittleEndian.Uint16(buf[2:4]))
			rate = int(binary.LittleEndian.Uint32(buf[4:8]))
			bits = int(binary.LittleEndian.Uint16(buf[14:16]))
		case "data":
			buf := make([]byte, size)
			if _, err := io.ReadFull(f, buf); err != nil {
				return nil, 0, err
			}
			data = buf
		default:
			// 청크 크기가 홀수면 패딩 1바이트가 붙는다.
			skip := int64(size)
			if size%2 == 1 {
				skip++
			}
			if _, err := f.Seek(skip, io.SeekCurrent); err != nil {
				return nil, 0, err
			}
		}
		if data != nil && rate != 0 {
			break
		}
	}
	if data == nil || rate == 0 {
		return nil, 0, errors.New("fmt 또는 data 청크가 없습니다")
	}
	if bits != 16 {
		return nil, 0, fmt.Errorf("16비트 PCM 만 받습니다: %d비트", bits)
	}
	if channels != 1 {
		return nil, 0, fmt.Errorf("모노만 받습니다: %d채널", channels)
	}

	n := len(data) / 2
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		v := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		out[i] = float32(v) / 32768.0
	}
	return out, rate, nil
}
