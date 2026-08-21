// Package model은 음성 모델의 기대값과 캐시 위치를 정한다.
//
// 받고 검증하는 일은 루트 모듈의 internal/modelfetch 가 한다. 이 패키지는
// 그 기계에 whisper 와 화자 분할 모델이라는 값을 먹인다. 크기와 체크섬을
// 고정하는 규율, Range 이어받기, 오프라인 반입은 engram 본체의 모델과
// 같은 것을 쓴다(ADR 0080).
package model

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/neocode24/engram/internal/modelfetch"
)

// ModelFile과 FileStatus와 ProgressFn은 modelfetch 의 것을 그대로 쓴다.
type (
	ModelFile  = modelfetch.ModelFile
	FileStatus = modelfetch.FileStatus
	ProgressFn = modelfetch.ProgressFn
)

// EnvModelDir는 모델 디렉토리를 바꾸는 환경변수다. engram 본체와 같은
// 이름을 쓴다. 사용자가 모델을 한 곳에 모으고 싶어 할 때 변수를 둘
// 기억하게 하지 않는다.
const EnvModelDir = "ENGRAM_MODEL_DIR"

// Size는 whisper 모델의 크기 갈래다.
type Size string

// 고를 수 있는 크기 셋이다. 기본은 Large 다. 실제 한국어 회의 녹음
// 둘에서 기준 일치율이 medium 보다 12%p 높았다(ADR 0081).
const (
	Large  Size = "large-v3"
	Medium Size = "medium"
	Small  Size = "small"
)

// Default는 기본 크기다.
const Default = Large

// Sizes는 고를 수 있는 크기를 고정 순서로 낸다.
func Sizes() []Size { return []Size{Large, Medium, Small} }

// ParseSize는 문자열을 크기로 바꾼다. 허용값이 아니면 에러다.
// 조용히 기본값으로 떨어지지 않는다. 오타를 낸 사용자가 왜 다른
// 모델이 도는지 모르게 된다.
func ParseSize(s string) (Size, error) {
	for _, v := range Sizes() {
		if string(v) == s {
			return v, nil
		}
	}
	return "", fmt.Errorf("크기 값이 허용값이 아님: %q (허용값: large-v3, medium, small)", s)
}

// 내려받기 호스트다. 모델이 한 곳에서 오지 않는다. whisper 는
// HuggingFace 의 변환본이고 화자 임베딩은 GitHub 릴리스다(ADR 0081).
const (
	hfBase = "https://huggingface.co/csukuangfj"
	ghBase = "https://github.com/k2-fsa/sherpa-onnx/releases/download"
)

// 모델 디렉토리 안에 놓이는 파일 이름이다. 크기마다 디렉토리가 다르므로
// 이름에 크기를 넣지 않는다. 쓰는 쪽 코드가 크기를 몰라도 된다.
const (
	EncoderName   = "encoder.int8.onnx"
	DecoderName   = "decoder.int8.onnx"
	TokensName    = "tokens.txt"
	SegmentName   = "segmentation.onnx"
	SpeakerName   = "speaker-embedding.onnx"
	VadName       = "silero_vad.onnx"
	whisperRemote = "/resolve/main/"
)

// whisperFiles는 크기별 whisper 파일 셋이다. 크기와 sha256 은 실제로
// 받아 계산한 값이다.
//
// int8 양자화판을 쓴다. fp32 는 large-v3 에서 6GB 에 가깝고 순수 CPU
// 에서 감당할 수 없다.
//
// tokens.txt 는 셋이 같은 파일이다. sha256 이 일치한다. whisper 다국어
// 모델이 토크나이저를 공유하기 때문이며, 그래도 크기별로 각자 받는다.
// 같은 값을 공유한다는 사실에 기대면 한쪽이 바뀔 때 조용히 깨진다.
var whisperFiles = map[Size][]ModelFile{
	Large: {
		{Base: hfBase, Remote: "sherpa-onnx-whisper-large-v3" + whisperRemote + "large-v3-encoder.int8.onnx", Name: EncoderName, Size: 766671985, SHA256: "d531cf17248acc43e8c09b472a0877055e770877857a5332fc1304b36534ec85"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-large-v3" + whisperRemote + "large-v3-decoder.int8.onnx", Name: DecoderName, Size: 1008265203, SHA256: "ebc6bfd88e162a46cb3edee8a7e727e1dcbc65cabecb19e2573695e4d495e1af"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-large-v3" + whisperRemote + "large-v3-tokens.txt", Name: TokensName, Size: 816730, SHA256: "b34b360dbb493e781e479794586d661700670d65564001f23024971d1f2fa126"},
	},
	Medium: {
		{Base: hfBase, Remote: "sherpa-onnx-whisper-medium" + whisperRemote + "medium-encoder.int8.onnx", Name: EncoderName, Size: 374196283, SHA256: "1c54582b4d829de0089f6cb63bbbdb3bf7555398bacaf855fbecf1a84dfd193e"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-medium" + whisperRemote + "medium-decoder.int8.onnx", Name: DecoderName, Size: 571059257, SHA256: "595d00a338a365a7bfa0ca7f296cabc639583bef770ab6130df90f49a6412747"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-medium" + whisperRemote + "medium-tokens.txt", Name: TokensName, Size: 816730, SHA256: "b34b360dbb493e781e479794586d661700670d65564001f23024971d1f2fa126"},
	},
	Small: {
		{Base: hfBase, Remote: "sherpa-onnx-whisper-small" + whisperRemote + "small-encoder.int8.onnx", Name: EncoderName, Size: 112442483, SHA256: "4cbe7b22fa9026b843b60a68640c747de05bafb1a11b57edc0e66c232d9f33a9"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-small" + whisperRemote + "small-decoder.int8.onnx", Name: DecoderName, Size: 262226114, SHA256: "acad50b5c782696e91b55914cc5ab4f756f1532f76e22aa6fc615f39fb69a8ee"},
		{Base: hfBase, Remote: "sherpa-onnx-whisper-small" + whisperRemote + "small-tokens.txt", Name: TokensName, Size: 816730, SHA256: "b34b360dbb493e781e479794586d661700670d65564001f23024971d1f2fa126"},
	},
}

// diarizationFiles는 화자 분할에 쓰는 둘이다. 크기와 무관하게 같다.
//
// segmentation 을 GitHub 릴리스의 tar.bz2 가 아니라 HuggingFace 의 단일
// 파일로 받는다. 아카이브를 풀 필요가 없어진다(ADR 0079).
//
// 원본 pyannote/segmentation-3.0 은 MIT 이나 HuggingFace 에서 게이트되어
// 토큰을 요구한다. MIT 가 재배포를 허용하므로 게이트 없는 변환본에서
// 받는다.
//
// 화자 임베딩 릴리스 태그의 recongition 은 오타가 아니라 upstream 이
// 실제로 쓰는 이름이다. 고치면 404 다.
var diarizationFiles = []ModelFile{
	{Base: hfBase, Remote: "sherpa-onnx-pyannote-segmentation-3-0/resolve/main/model.onnx", Name: SegmentName, Size: 5992913, SHA256: "220ad67ca923bef2fa91f2390c786097bf305bceb5e261d4af67b38e938e1079"},
	{Base: ghBase, Remote: "speaker-recongition-models/3dspeaker_speech_campplus_sv_zh_en_16k-common_advanced.onnx", Name: SpeakerName, Size: 28281164, SHA256: "aa3cfc16963a10586a9393f5035d6d6b57e98d358b347f80c2a30bf4f00ceba2"},
}

// vadFile은 구간을 나누는 데 쓰는 VAD 모델이다. 이것이 없으면 긴 녹음을
// whisper 의 30초 창에 맞게 자를 수 없다.
var vadFile = ModelFile{
	Base: ghBase, Remote: "asr-models/silero_vad.onnx", Name: VadName,
	Size: 643854, SHA256: "9e2449e1087496d8d4caba907f23e0bd3f78d91fa552479bb9c23ac09cbb1fd6",
}

// ErrUnknownSize는 모르는 크기를 받았을 때의 오류다.
var ErrUnknownSize = errors.New("모르는 모델 크기")

// Files는 크기 하나를 쓰는 데 필요한 파일 전부를 낸다. whisper 셋과
// 화자 분할 둘과 VAD 하나로 여섯이다.
func Files(size Size) ([]ModelFile, error) {
	w, ok := whisperFiles[size]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSize, size)
	}
	out := slices.Clone(w)
	out = append(out, diarizationFiles...)
	out = append(out, vadFile)
	return out, nil
}

// TotalSize는 크기 하나를 받는 데 필요한 바이트 합계다. 사용자에게
// 얼마나 받는지 미리 알린다.
func TotalSize(size Size) (int64, error) {
	files, err := Files(size)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total, nil
}

// Dir는 크기별 모델 디렉토리를 낸다. ENGRAM_MODEL_DIR 이 있으면 그
// 아래에, 없으면 사용자 캐시 아래에 둔다. whisper 크기마다 디렉토리를
// 나누는 이유는 large-v3 와 medium 의 파일 이름이 접두사만 다르고 화자
// 분할 모델은 같아서, 한 디렉토리에 섞으면 어느 크기가 온전한지
// 판정할 수 없기 때문이다.
func Dir(size Size) (string, error) {
	if v := os.Getenv(EnvModelDir); v != "" {
		return filepath.Join(v, "voice", string(size)), nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "engram", "models", "voice", string(size)), nil
}
