package embed

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// fakeEncoder는 텍스트를 결정적 벡터로 바꾸는 시험용 인코더다.
// 벡터의 첫 원소는 배치 안 순번, 둘째 원소는 문서별 고유 번호다.
type fakeEncoder struct {
	calls int
	fail  error
}

func (f *fakeEncoder) Encode(ctx context.Context, texts []string) ([][]float32, error) {
	f.calls++
	if f.fail != nil {
		return nil, f.fail
	}
	out := make([][]float32, len(texts))
	for i, txt := range texts {
		out[i] = []float32{float32(i), float32(len([]rune(txt)) % 97)}
	}
	return out, nil
}

func (f *fakeEncoder) Close() error { return nil }

func mkDocs(n int) []ComputeDoc {
	docs := make([]ComputeDoc, n)
	for i := range docs {
		docs[i] = ComputeDoc{Path: fmt.Sprintf("context/d%d.md", i), Title: fmt.Sprintf("제목 %d", i), Body: fmt.Sprintf("본문 %d", i)}
	}
	return docs
}

func TestComputeFillsMissingOnly(t *testing.T) {
	// 캐시에 있는 문서는 다시 계산하지 않는다.
	docs := mkDocs(3)
	cache := map[string][]float32{Key(docs[0].Title + "\n" + docs[0].Body): {9, 9}}
	enc := &fakeEncoder{}
	out, err := compute(cache, docs, enc, nil)
	if err != nil {
		t.Fatalf("계산 실패: %v", err)
	}
	if _, ok := out[docs[0].Path]; !ok || out[docs[0].Path][0] != 9 {
		t.Errorf("캐시 히트한 벡터를 그대로 내야 합니다: %+v", out[docs[0].Path])
	}
	if enc.calls != 1 {
		t.Errorf("캐시에 없는 문서만 인코딩해야 합니다. Encode 호출 %d회", enc.calls)
	}
	if len(out) != 3 {
		t.Errorf("문서 셋 모두 벡터를 가져야 합니다: %d", len(out))
	}
}

func TestComputeSkipsOpenWhenCacheFull(t *testing.T) {
	// 캐시에 없는 것이 하나도 없으면 모델을 열지 않는다. enc 에 nil 을
	// 주면 compute 내부에서 Open 을 부르는데, 모델이 없는 환경이므로
	// 열었다면 ErrNoModel 로 실패한다. 성공은 곧 열지 않았다는 증명이다.
	t.Setenv(EnvModelDir, t.TempDir())
	docs := mkDocs(2)
	cache := map[string][]float32{}
	for _, d := range docs {
		cache[Key(d.Title+"\n"+d.Body)] = []float32{1, 2}
	}
	out, err := compute(cache, docs, nil, nil)
	if err != nil {
		t.Fatalf("캐시가 다 차면 모델을 열지 않아야 합니다: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("문서 둘의 벡터: %d", len(out))
	}
}

func TestComputeBatchesAndReportsProgress(t *testing.T) {
	// 문서 10개는 배치 8과 2로 나뉜다. 진행률은 배치가 끝날 때마다 온다.
	enc := &fakeEncoder{}
	var seen [][2]int
	out, err := compute(map[string][]float32{}, mkDocs(10), enc, func(done, total int) {
		seen = append(seen, [2]int{done, total})
	})
	if err != nil {
		t.Fatalf("계산 실패: %v", err)
	}
	if enc.calls != 2 {
		t.Errorf("배치 8이면 Encode 호출은 2회여야 합니다: %d", enc.calls)
	}
	if len(out) != 10 {
		t.Errorf("문서 10개의 벡터: %d", len(out))
	}
	want := [][2]int{{8, 10}, {10, 10}}
	if len(seen) != 2 || seen[0] != want[0] || seen[1] != want[1] {
		t.Errorf("진행률 콜백: %+v, want %+v", seen, want)
	}
}

func TestComputeNilProgressIsFine(t *testing.T) {
	// 콜백이 nil 이면 부르지 않는다. 이 시험이 통과하면 그뿐이다.
	if _, err := compute(map[string][]float32{}, mkDocs(1), &fakeEncoder{}, nil); err != nil {
		t.Fatalf("nil 진행률 콜백: %v", err)
	}
}

func TestComputePropagatesEncodeError(t *testing.T) {
	enc := &fakeEncoder{fail: errors.New("인코딩 불가")}
	if _, err := compute(map[string][]float32{}, mkDocs(1), enc, nil); err == nil {
		t.Error("인코딩 에러를 올려야 합니다")
	}
}

func TestComputeReturnsNoModelErrorWithoutModel(t *testing.T) {
	// 계산할 것이 있는데 모델이 없으면 ErrNoModel 을 그대로 올린다.
	// 에러가 아니라 의미 축 강등 신호다.
	t.Setenv(EnvModelDir, t.TempDir())
	_, err := compute(map[string][]float32{}, mkDocs(1), nil, nil)
	if !errors.Is(err, ErrNoModel) {
		t.Fatalf("ErrNoModel 이어야 합니다: %v", err)
	}
}

func TestComputePersistsToCacheFile(t *testing.T) {
	// Compute 전체는 계산 결과를 .engram/vectors.json 에 남긴다.
	// 두 번째 호출은 캐시에서 곧바로 낸다. 모델이 없는 환경에서
	// 캐시가 다 차 있으면 성공이 곧 캐시 적중의 증명이다.
	t.Setenv(EnvModelDir, t.TempDir())
	root := t.TempDir()
	docs := mkDocs(2)
	cache := map[string][]float32{}
	for _, d := range docs {
		cache[Key(d.Title+"\n"+d.Body)] = []float32{0.5, 0.5}
	}
	if err := SaveCache(root, cache, map[string]bool{Key(docs[0].Title + "\n" + docs[0].Body): true, Key(docs[1].Title + "\n" + docs[1].Body): true}); err != nil {
		t.Fatal(err)
	}
	out, err := Compute(root, docs, nil)
	if err != nil {
		t.Fatalf("캐시가 다 찼으면 모델 없이도 성공해야 합니다: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("문서 둘의 벡터: %d", len(out))
	}
	// 저장 시 keep 밖의 항목은 사라진다.
	if got := LoadCache(root); len(got) != 2 {
		t.Errorf("캐시 파일: %d건", len(got))
	}
}
