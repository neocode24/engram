// Package embed는 문서를 벡터로 바꾼다. 재발견의 의미 축이 이것을 쓴다.
//
// 백엔드는 hugot 의 순수 Go 경로 하나이며 코어의 CGO_ENABLED=0 을 지킨다.
// onnxruntime 이 22배 빠르나 네이티브 라이브러리 둘이 붙어 단일 정적
// 바이너리 계약을 깬다. 그 대신 계산 시점을 reindex 가 아니라 bridge 로
// 옮겨 편집마다 비용을 물지 않게 했다. 근거와 재검토 조건은 ADR 0074 에 있다.
//
// 벡터는 문서 하나에 하나이고 대상은 제목과 본문을 이은 앞 Chars 자다.
// ADR 0075.
package embed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// Chars는 인코딩에 쓰는 텍스트 길이다. upstream 의 EMBED_CHARS 와 같은
// 값으로 두어 재발견 순위를 비교할 수 있게 한다(ADR 0075).
const Chars = 2000

// Dims는 bge-m3 의 출력 차원이다.
const Dims = 1024

// ModelName은 모델 디렉토리 이름이다. 사이드카 목록이 하나이므로
// 고정한다(ADR 0068, 0074).
const ModelName = "bge-m3"

// EnvModelDir는 모델 디렉토리를 덮어쓰는 환경변수다. 오프라인 반입,
// 공유 마운트, 테스트가 이 경로를 쓴다.
const EnvModelDir = "ENGRAM_MODEL_DIR"

// onnxFilename은 모델 디렉토리 안의 그래프 파일 이름이다. 가중치는
// 같은 디렉토리의 model.onnx_data 에 따로 있다.
const onnxFilename = "model.onnx"

// ModelDir는 모델이 놓이는 디렉토리를 반환한다.
//
// 사용자 전역 캐시에 둔다. 위키 로컬 .engram/ 이 아닌 이유는 위키마다
// 2.3GB 사본이 생기기 때문이고, 설정 디렉토리가 아닌 이유는 모델이
// 설정이 아니라 다시 받을 수 있는 파생물이기 때문이다(ADR 0074).
func ModelDir() (string, error) {
	if v := os.Getenv(EnvModelDir); v != "" {
		return v, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "engram", "models", ModelName), nil
}

// Key는 캐시 키를 만든다. 잘라낸 텍스트의 sha256 이다.
//
// 파일 수정 시각이 아니라 내용을 보는 이유가 있다. 어휘 색인은 다시
// 만드는 데 0.1초라 시각 기준으로 충분하지만 임베딩은 문서당 12.6초다.
// 형식만 바뀐 커밋이나 파일 이동으로 다시 계산하면 안 된다(ADR 0075).
func Key(text string) string {
	sum := sha256.Sum256([]byte(Truncate(text)))
	return hex.EncodeToString(sum[:])
}

// Truncate는 인코딩 대상 길이로 자른다. 바이트가 아니라 룬 기준이다.
// 한국어 한 글자가 3바이트라 바이트로 자르면 upstream 과 다른 양을
// 보게 된다.
func Truncate(text string) string {
	r := []rune(text)
	if len(r) > Chars {
		r = r[:Chars]
	}
	return string(r)
}

// Encoder는 텍스트를 벡터로 바꾼다. 백엔드가 바뀌면 이 인터페이스를
// 구현하는 쪽만 바뀐다(ADR 0074).
type Encoder interface {
	// Encode는 텍스트 여럿을 한 번에 인코딩한다. 반환 순서는 입력
	// 순서와 같다. 벡터는 L2 정규화되어 있으므로 코사인 유사도가
	// 내적과 같다.
	Encode(ctx context.Context, texts []string) ([][]float32, error)
	// Close는 백엔드 자원을 놓는다.
	Close() error
}
