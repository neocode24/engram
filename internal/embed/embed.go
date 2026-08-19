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
// 같은 디렉토리의 model.onnx_data 에 따로 있고 이 그래프가 그 이름을
// 내부에서 참조한다.
const onnxFilename = "sentence_transformers.onnx"

// outputName은 문장 벡터를 꺼낼 ONNX 출력 이름이다.
//
// bge-m3 는 CLS 풀링을 쓴다. BAAI/bge-m3 의 1_Pooling/config.json 이
// pooling_mode_cls_token 을 참으로 두었다. 그런데 hugot 의 특징추출
// 파이프라인은 출력이 토큰 임베딩이면 평균 풀링을 강제하며 선택지를
// 주지 않는다. 평균 풀링을 bge-m3 에 쓰면 유사도 분포가 뭉개져 하한이
// 필터 구실을 못 한다. upstream 과 같은 80문서에서 코사인 0.72 이상이
// upstream 45쌍인데 평균 풀링은 2897쌍이었다.
//
// 그래서 풀링이 그래프 안에 들어 있는 sentence_transformers.onnx 를
// 받고 그 출력을 골라 쓴다. 이렇게 하면 hugot 이 이미 풀링된 벡터를
// 받아 정규화만 한다. 실측에서 0.72 이상이 45쌍으로 upstream 과 같다.
const outputName = "sentence_embedding"

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
