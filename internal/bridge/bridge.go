// Package bridge는 유사도가 높은데 링크가 없는 문서 쌍을 찾아
// 관계 축 재발견을 돕는다. 후보와 근거만 내고 문장은 만들지 않는다. ADR 0028.
package bridge

import (
	"math"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/state"
)

// Pair는 후보 쌍 하나다. A와 B는 슬러그 오름차순으로 담는다.
type Pair struct {
	A     string  `json:"a"`
	B     string  `json:"b"`
	Score float64 `json:"score"`
}

// Stats는 탈락 사유별 쌍 수다. 후보가 없을 때 이유를 대는 재료로 쓴다.
type Stats struct {
	ContextDocs int `json:"contextDocs"`
	Linked      int `json:"linked"`
	Rejected    int `json:"rejected"`
	BelowMin    int `json:"belowMin"`
}

// Result는 bridge 계산 결과 전부다.
type Result struct {
	Pairs []Pair `json:"pairs"`
	Stats Stats  `json:"stats"`
}

// candDoc는 후보 비교에 쓰는 문서 하나분의 재료다.
type candDoc struct {
	slug string        // 색인 슬러그. 화면에 내는 값이다
	key  string        // 링크 비교용 정규화 슬러그
	tf   map[string]float64
	out  map[string]bool // 이 문서가 거는 링크의 정규화 슬러그
}

// Run은 context 문서 쌍 가운데 코사인 유사도가 min 이상이고 링크도
// 기각도 없는 쌍을 유사도 내림차순으로 limit 개 낸다. 같은 입력에 같은
// 출력이다. 기각 목록은 입력이지 출력이 아니다.
func Run(ix *index.Index, g *graph.Graph, st state.State, min float64, limit int) Result {
	// ponytail: O(n^2) 쌍 비교. 문서 수 2000 규모까지는 감당한다.
	// 그보다 커지면 토큰 역색인으로 교집합 쌍만 비교하게 바꾼다.
	var docs []candDoc
	for _, e := range ix.Docs {
		if seg, _, _ := strings.Cut(e.Path, "/"); seg != "context" {
			continue
		}
		d := candDoc{slug: e.Slug, key: graph.Normalize(e.Slug), tf: e.TF, out: map[string]bool{}}
		for _, l := range g.Outgoing(e.Path) {
			d.out[l.Slug] = true
		}
		docs = append(docs, d)
	}
	res := Result{Pairs: []Pair{}, Stats: Stats{ContextDocs: len(docs)}}
	for i := 0; i < len(docs); i++ {
		for j := i + 1; j < len(docs); j++ {
			a, b := docs[i], docs[j]
			// 방향 무관이다. 한쪽만 가리켜도 이어진 쌍이다.
			if a.out[b.key] || b.out[a.key] {
				res.Stats.Linked++
				continue
			}
			if st.IsRejected(a.key, b.key) {
				res.Stats.Rejected++
				continue
			}
			score := cosine(a.tf, b.tf)
			if score < min {
				res.Stats.BelowMin++
				continue
			}
			p := Pair{Score: score}
			if a.slug <= b.slug {
				p.A, p.B = a.slug, b.slug
			} else {
				p.A, p.B = b.slug, a.slug
			}
			res.Pairs = append(res.Pairs, p)
		}
	}
	sort.Slice(res.Pairs, func(i, j int) bool {
		pi, pj := res.Pairs[i], res.Pairs[j]
		if pi.Score != pj.Score {
			return pi.Score > pj.Score
		}
		if pi.A != pj.A {
			return pi.A < pj.A
		}
		return pi.B < pj.B
	})
	if limit >= 0 && len(res.Pairs) > limit {
		res.Pairs = res.Pairs[:limit]
	}
	return res
}

// cosine은 TF 벡터 둘의 코사인 유사도를 낸다. BM25는 질의와 문서를 잇는
// 함수이므로 문서와 문서를 비교하는 여기에는 맞지 않다. 토크나이저가
// 한 벌이므로 search가 같다고 보는 문서를 bridge도 같다고 본다.
func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64
	for k, v := range a {
		na += v * v
		if w, ok := b[k]; ok {
			dot += v * w
		}
	}
	for _, v := range b {
		nb += v * v
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}
