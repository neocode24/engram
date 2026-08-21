package embed

import (
	"context"
	"net/http"
	"slices"

	"github.com/neocode24/engram/internal/modelfetch"
)

// Revision은 내려받기를 고정하는 HuggingFace 커밋 SHA 다. main 을 쓰지
// 않는 이유는 같은 model pull 이 언제 돌아도 같은 바이트를 받아야 하기
// 때문이다(ADR 0074). Xenova/bge-m3 의 2026-02-10 시점 HEAD 다.
const Revision = "4de13258303883538bd53b696b452bf8099f0858"

// DownloadBase는 내려받기 URL 의 앞부분이다. 파일의 저장소 안 경로를
// 붙여 쓴다. 테스트는 이 값을 httptest 서버 주소로 바꿔 끼운다.
const DownloadBase = "https://huggingface.co/Xenova/bge-m3/resolve/" + Revision

// 받고 검증하는 일 자체는 internal/modelfetch 에 있다. 이 파일은 그
// 기계에 bge-m3 라는 값을 먹인다. 나누는 이유는 modelfetch 의 머리말에
// 있다. 요약하면 이 패키지가 hugot 을 끌고 오기 때문이다.
type (
	// ModelFile은 모델을 이루는 파일 하나의 기대값이다.
	ModelFile = modelfetch.ModelFile
	// FileStatus는 model status 가 낼 파일 하나의 상태다.
	FileStatus = modelfetch.FileStatus
	// ProgressFn은 내려받기 진행률을 받는 콜백이다.
	ProgressFn = modelfetch.ProgressFn
)

// 내려받기 실패의 종류다. 호출자가 문구를 고르게 보내는 값이다.
var (
	ErrChecksum    = modelfetch.ErrChecksum
	ErrSize        = modelfetch.ErrSize
	ErrMissingFile = modelfetch.ErrMissingFile
)

// modelFiles는 기대값 여섯이다. 크기와 sha256 은 리비전 4de1325 에서
// 실제로 받아 계산했고 LFS 파일 둘은 HuggingFace API 의 lfs.oid 와
// 같은 값임을 겹쳐 확인했다.
//
// Base 를 비워 둔다. 여섯이 한 저장소에서 오므로 호출자가 주는 기본
// base 하나로 충분하다.
var modelFiles = []ModelFile{
	{Remote: "onnx/sentence_transformers.onnx", Name: "sentence_transformers.onnx", Size: 724923, SHA256: "c53a8fe59f64ae6babb972b59b6679d8173e88b378637eba495ed0f7227f3dca"},
	{Remote: "onnx/model.onnx_data", Name: "model.onnx_data", Size: 2266820608, SHA256: "1eebfb28493f67bba03ce0ef64bfdc7fc5a3bd9d7493f818bb1d78cd798416b4"},
	{Remote: "tokenizer.json", Name: "tokenizer.json", Size: 17082821, SHA256: "6710678b12670bc442b99edc952c4d996ae309a7020c1fa0096dd245c2faf790"},
	{Remote: "config.json", Name: "config.json", Size: 770, SHA256: "734a79bf12d388c1467a4e3ab625f45de7f6906cffcfb93a1eca1787504bed95"},
	{Remote: "tokenizer_config.json", Name: "tokenizer_config.json", Size: 1173, SHA256: "7e4c1cc848840aeccdd763458c18dd525eb0f795c992e00ebe9c28554e7db2d4"},
	{Remote: "special_tokens_map.json", Name: "special_tokens_map.json", Size: 964, SHA256: "8c785abebea9ae3257b61681b4e6fd8365ceafde980c21970d001e834cf10835"},
}

// ModelFiles는 기대값 여섯의 복사본을 반환한다.
func ModelFiles() []ModelFile {
	return slices.Clone(modelFiles)
}

// Download는 모델 파일 여섯을 base URL 에서 dir 로 받는다.
func Download(ctx context.Context, client *http.Client, base, dir string, prog ProgressFn) ([]string, error) {
	return modelfetch.Download(ctx, client, base, dir, modelFiles, prog)
}

// Inspect는 dir 안 모델 파일의 상태를 낸다.
func Inspect(dir string, verify bool) ([]FileStatus, error) {
	return modelfetch.Inspect(dir, modelFiles, verify)
}

// Import는 오프라인 자료에서 파일 여섯을 가져와 dir 에 놓는다.
func Import(src, dir string) ([]string, error) {
	return modelfetch.Import(src, dir, modelFiles)
}
