package embed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

// ErrNoModel은 모델이 없을 때의 오류다. 호출자는 이것을 받으면 의미
// 축을 빼고 계속 간다. 시맨틱의 부재는 결손이 아니라 성능 저하다(ADR 0007).
var ErrNoModel = errors.New("모델 없음")

// goEncoder는 hugot 의 순수 Go 백엔드다.
type goEncoder struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
}

// Present는 모델 파일이 자리에 있는지 본다. 무결성은 보지 않는다.
// 체크섬 검증은 model 커맨드와 doctor 의 몫이다.
func Present() bool {
	dir, err := ModelDir()
	if err != nil {
		return false
	}
	for _, name := range []string{onnxFilename, "tokenizer.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

// Open은 인코더를 연다. 모델이 없으면 ErrNoModel 을 반환한다.
//
// 적재에 0.5초가 걸리므로 문서 하나마다 여는 것이 아니라 한 번 열어
// 배치로 돌린다. 호출자가 Close 를 부른다.
func Open() (Encoder, error) {
	if !Present() {
		return nil, ErrNoModel
	}
	dir, err := ModelDir()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	session, err := hugot.NewGoSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("세션을 열 수 없음: %w", err)
	}
	pipeline, err := hugot.NewPipeline(session, hugot.FeatureExtractionConfig{
		ModelPath:    dir,
		Name:         ModelName,
		OnnxFilename: onnxFilename,
		Options: []hugot.FeatureExtractionOption{
			// 정규화해 두면 코사인 유사도가 내적과 같아진다.
			pipelines.WithNormalization(),
			// sentence_embedding 출력을 골라야 CLS 풀링이 된다. 이유는
			// outputName 상수의 주석에 있다.
			pipelines.WithOutputName(outputName),
		},
	})
	if err != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("모델을 적재할 수 없음: %w", err)
	}
	return &goEncoder{session: session, pipeline: pipeline}, nil
}

func (e *goEncoder) Encode(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out, err := e.pipeline.RunPipeline(ctx, texts)
	if err != nil {
		return nil, err
	}
	return out.Embeddings, nil
}

func (e *goEncoder) Close() error {
	return e.session.Destroy()
}
