package embed

import (
	"context"
	"errors"
)

// BatchSize는 한 번에 인코딩할 문서 수다. 8은 upstream 의 값이다(ADR 0075).
const BatchSize = 8

// ComputeDoc는 임베딩 계산 대상 문서 하나다. Title 과 Body 는 호출자가
// 색인에서 뽑은 값이다.
type ComputeDoc struct {
	Path  string
	Title string
	Body  string
}

// Compute는 문서 목록의 임베딩 벡터를 경로 기준 맵으로 낸다. 캐시에 있는
// 것은 다시 계산하지 않고, 캐시에 없는 것이 하나도 없으면 모델을 열지
// 않는다. 적재에 0.5초가 걸리므로 그 비용을 물지 않기 위해서다.
//
// 계산이 끝나면 현재 문서 키만 남겨 캐시를 저장한다. 계산 중 에러가
// 나도 이미 계산한 것은 버리지 않고 저장한 뒤 에러를 올린다. 문서당
// 12.6초짜리 작업을 부분 실패로 통째로 날리는 것은 낭비다.
//
// embed.ErrNoModel 은 그대로 올린다. 에러가 아니라 의미 축 강등 신호다.
// 호출자가 판별해 단어 축만으로 계속한다.
//
// progress 는 진행률 콜백이다. 문서 수가 많으면 수십 분이 걸리므로
// 사용자를 침묵 속에 기다리게 하지 않는다. total 은 이번에 실제로
// 인코딩할 문서 수이고 done 은 마친 수다. 콜백이 nil 이면 부르지 않는다.
func Compute(wikiRoot string, docs []ComputeDoc, progress func(done, total int)) (map[string][]float32, error) {
	cache := LoadCache(wikiRoot)
	out, err := compute(cache, docs, nil, progress)
	if err != nil {
		// ErrNoModel 은 계산 전에 난 신호라 저장할 것이 없다. 그 외
		// 에러에서는 앞 배치의 결과가 cache 에 남으므로 저장하고 올린다.
		if !errors.Is(err, ErrNoModel) {
			_ = SaveCache(wikiRoot, cache, keepKeys(docs))
		}
		return nil, err
	}
	return out, SaveCache(wikiRoot, cache, keepKeys(docs))
}

// Cached는 캐시에 이미 있는 벡터만 낸다. 모델을 열지 않고 없는 것을
// 계산하지도 않으며 캐시를 다시 쓰지도 않는다.
//
// 응답이 빨라야 하는 자리에서 쓴다. 임베딩은 문서당 12.6초라 도구
// 호출 안에서 계산할 수 없다. 캐시를 채우는 것은 bridge 커맨드의 몫이고
// 여기서는 그 결과를 읽기만 한다. 캐시가 비어 있으면 빈 맵을 낸다.
// 호출자는 반환된 맵이 문서 전부를 덮지 않을 수 있다는 것을 전제한다.
func Cached(wikiRoot string, docs []ComputeDoc) map[string][]float32 {
	cache := LoadCache(wikiRoot)
	out := make(map[string][]float32, len(docs))
	for _, d := range docs {
		if v, ok := cache[Key(d.Title+"\n"+d.Body)]; ok {
			out[d.Path] = v
		}
	}
	return out
}

// compute는 캐시를 채워 문서별 벡터를 낸다. enc 가 nil 이면 필요해진
// 시점에 Open 한다. 캐시에 없는 것이 없으면 enc 를 한 번도 열지 않는다.
// 계산된 벡터는 cache 에도 넣으므로 호출자는 계산 후 cache 를 저장하면 된다.
func compute(cache map[string][]float32, docs []ComputeDoc, enc Encoder, progress func(done, total int)) (map[string][]float32, error) {
	out := make(map[string][]float32, len(docs))
	var missing []ComputeDoc
	seen := map[string]bool{}
	// 내용이 같은 문서가 여럿이면 인코딩은 한 번만 한다. 다만 경로는
	// 저마다 결과를 받아야 한다. 뒤 문서를 건너뛰기만 하면 그 문서가
	// 임베딩 축에서 통째로 빠진다.
	dupes := map[string][]string{}
	for _, d := range docs {
		key := Key(d.Title + "\n" + d.Body)
		if v, ok := cache[key]; ok {
			out[d.Path] = v
			continue
		}
		if seen[key] {
			dupes[key] = append(dupes[key], d.Path)
			continue
		}
		seen[key] = true
		missing = append(missing, d)
	}
	defer func() {
		for key, paths := range dupes {
			v, ok := cache[key]
			if !ok {
				continue
			}
			for _, p := range paths {
				out[p] = v
			}
		}
	}()
	if len(missing) == 0 {
		return out, nil
	}
	own := false
	if enc == nil {
		var err error
		enc, err = Open()
		if err != nil {
			return nil, err
		}
		own = true
	}
	if own {
		defer func() { _ = enc.Close() }()
	}
	total := len(missing)
	done := 0
	for start := 0; start < total; start += BatchSize {
		end := start + BatchSize
		if end > total {
			end = total
		}
		texts := make([]string, 0, end-start)
		for _, d := range missing[start:end] {
			texts = append(texts, Truncate(d.Title+"\n"+d.Body))
		}
		vecs, err := enc.Encode(context.Background(), texts)
		if err != nil {
			return nil, err
		}
		if len(vecs) != end-start {
			return nil, errors.New("인코딩 결과 수가 입력과 다릅니다")
		}
		for i, d := range missing[start:end] {
			cache[Key(d.Title+"\n"+d.Body)] = vecs[i]
			out[d.Path] = vecs[i]
		}
		done = end
		if progress != nil {
			progress(done, total)
		}
	}
	return out, nil
}

// keepKeys는 저장할 키의 집합을 낸다. 같은 내용의 문서가 여럿이면 키가
// 하나로 합쳐진다. 키가 내용 해시이므로 그대로 옳다.
func keepKeys(docs []ComputeDoc) map[string]bool {
	keep := make(map[string]bool, len(docs))
	for _, d := range docs {
		keep[Key(d.Title+"\n"+d.Body)] = true
	}
	return keep
}
