// Package bridge는 유사도가 높은데 링크가 없는 문서 쌍을 찾아
// 관계를 축으로 한 재발견을 돕는다. 후보와 근거만 내고 문장은 만들지 않는다. ADR 0028.
// 단어 축과 임베딩 축을 각각 자기 하한으로 걸러 합집합으로 낸다. ADR 0075.
package bridge

import (
	"math"
	"sort"
	"strings"

	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/state"
)

// 축 식별자다. Pair.Axes 의 값이고 --json 출력에 그대로 나간다.
const (
	AxisTerm  = "term"
	AxisEmbed = "embed"
)

// Pair는 후보 쌍 하나다. A와 B는 슬러그 오름차순으로 담는다.
type Pair struct {
	A     string   `json:"a"`
	B     string   `json:"b"`
	Score float64  `json:"score"`
	Axes  []string `json:"axes"`
	// termScore 와 embedScore 는 각 축의 원점수고 rank 는 최선 축
	// 순위다. 셋 다 정렬과 출력 재료로 쓰고 JSON 에는 Score 와 Axes 만
	// 낸다.
	termScore  float64
	embedScore float64
	rank       int
}

// Stats는 탈락 사유별 쌍 수다. 후보가 없을 때 이유를 대는 재료로 쓴다.
// BelowMin 은 두 축 모두 하한에 못 미친 쌍 수다. 한 축이라도 통과하면
// 후보가 되므로 축별 미달은 세지 않는다.
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

// Options는 Run 의 인자다.
type Options struct {
	// Min 은 단어 축의 코사인 하한이다.
	Min float64
	// EmbedMin 은 임베딩 축의 코사인 하한이다. 단어 축과 눈금이 다르므로
	// 값을 공유하지 않는다(ADR 0075).
	EmbedMin float64
	// Limit 은 낼 쌍 수 상한이다. 음수면 제한 없다.
	Limit int
	// Vectors 는 색인 경로 기준 임베딩 벡터 맵이다. nil 이면 임베딩
	// 축을 끄고 단어 축만 돈다. 맵에 문서가 빠져 있으면 그 문서는
	// 임베딩 축 후보에서만 빠진다.
	Vectors map[string][]float32
}

// candDoc는 후보 비교에 쓰는 문서 하나분의 재료다.
type candDoc struct {
	slug string // 색인 슬러그. 화면에 내는 값이다
	path string // 임베딩 벡터 맵의 키. 색인 경로다
	key  string // 링크 비교용 정규화 슬러그
	tf   map[string]float64
	vec  []float32       // 임베딩 벡터. 없으면 nil 이다
	out  map[string]bool // 이 문서가 거는 링크의 정규화 슬러그
}

// Run은 context 문서 쌍 가운데 단어 축이나 임베딩 축 하나라도 통과하고
// 링크도 기각도 없는 쌍을 낸다. 두 축을 통과한 쌍의 합집합이지 교집합이
// 아니다. 두 축은 서로 다른 것을 잡으므로 교집합을 보면 둘 다 잡는 좁은
// 영역만 남아 재발견이 죽는다(ADR 0075). 같은 입력에 같은 출력이다.
// 기각 목록은 입력이지 출력이 아니다.
func Run(ix *index.Index, g *graph.Graph, st state.State, opts Options) Result {
	// ponytail: O(n^2) 쌍 비교. 문서 수 2000 규모까지는 감당한다.
	// 그보다 커지면 토큰 역색인으로 교집합 쌍만 비교하게 바꾼다.
	var docs []candDoc
	for _, e := range ix.Docs {
		if seg, _, _ := strings.Cut(e.Path, "/"); seg != "context" {
			continue
		}
		d := candDoc{slug: e.Slug, path: e.Path, key: graph.Normalize(e.Slug), tf: e.TF, vec: opts.Vectors[e.Path], out: map[string]bool{}}
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
			// 링크와 기각은 두 축 모두에 같은 기준으로 걸린다.
			if a.out[b.key] || b.out[a.key] {
				res.Stats.Linked++
				continue
			}
			if st.IsRejected(a.key, b.key) {
				res.Stats.Rejected++
				continue
			}
			term := cosine(a.tf, b.tf)
			embed := dot(a.vec, b.vec)
			// 임베딩 축은 벡터가 둘 다 있을 때만 판정한다. 벡터가 없는
			// 문서의 내적은 0 인데 하한이 0 이면 통과로 세어지므로 nil
			// 검사로 막는다.
			termOK := term >= opts.Min
			embedOK := a.vec != nil && b.vec != nil && embed >= opts.EmbedMin
			if !termOK && !embedOK {
				res.Stats.BelowMin++
				continue
			}
			p := Pair{termScore: term, embedScore: embed}
			if termOK {
				p.Axes = append(p.Axes, AxisTerm)
			}
			if embedOK {
				p.Axes = append(p.Axes, AxisEmbed)
			}
			if a.slug <= b.slug {
				p.A, p.B = a.slug, b.slug
			} else {
				p.A, p.B = b.slug, a.slug
			}
			// 대표 점수는 두 축 점수의 최댓값으로 내되 출력용이다.
			// 정렬 키는 아래에서 축 순위로 따로 만든다.
			p.Score = math.Max(term, embed)
			res.Pairs = append(res.Pairs, p)
		}
	}
	// 정렬은 최선 축 순위 오름차순, 동점은 대표 점수 내림차순, 그다음
	// 첫 슬러그와 둘째 슬러그 오름차순이다.
	//
	// 두 축의 점수 눈금은 다르다(ADR 0075). 원점수의 최댓값을 키로 쓰면
	// 분포가 높은 축이 항상 이겨 한 축의 서열이 된다. 실측(llm-wiki
	// context 80문서, 2026-08)에서 두 축을 모두 통과한 쌍 396개 전부에서
	// 임베딩 점수가 단어 점수보다 높았다. 최댓값 키는 임베딩 서열이었다.
	// 축 내 상대 서열인 순위만이 같은 지위의 두 축을 섞는 공통 눈금이다.
	// 쌍의 키는 자기가 통과한 축들 가운데 최선 순위다. 단어 축 최상위
	// 쌍이 임베딩 축 중하위 쌍들에 묻히지 않는다.
	//
	// 순위가 같으면 대표 점수, 슬러그 순으로 매듭는다. 같은 위키와 같은
	// 설정이 항상 같은 순서를 내 골든 스냅샷이 성립한다. 순위 키 때문에
	// 대표 점수가 출력에서 단조롭지 않게 나열될 수 있는데, 그것이 두 축을
	// 같은 지위로 다루는 값이다.
	// 순위를 Pair 에 박은 뒤 정렬한다. 정렬은 슬라이스를 제자리에서
	// 재배치하므로 바깥 배열의 인덱스로 순위를 보면 재배치 중 어긋난다.
	assignRanks(res.Pairs)
	sort.SliceStable(res.Pairs, func(i, j int) bool {
		pi, pj := res.Pairs[i], res.Pairs[j]
		// 두 축이 함께 잡은 쌍을 맨 앞에 둔다. 서로 다른 것을 보는 두
		// 신호가 같은 쌍을 가리켰다는 것이 한 신호가 강한 것보다 나은
		// 증거다. upstream 이 정렬의 첫째 키로 삼는 것도 이것이다
		// (scripts/wiki_resurface.py 의 len(by)).
		if len(pi.Axes) != len(pj.Axes) {
			return len(pi.Axes) > len(pj.Axes)
		}
		if pi.rank != pj.rank {
			return pi.rank < pj.rank
		}
		if pi.Score != pj.Score {
			return pi.Score > pj.Score
		}
		if pi.A != pj.A {
			return pi.A < pj.A
		}
		return pi.B < pj.B
	})
	if opts.Limit >= 0 && len(res.Pairs) > opts.Limit {
		res.Pairs = res.Pairs[:opts.Limit]
	}
	return res
}

// assignRanks는 쌍마다 자기가 통과한 축들 가운데 최선 순위를 rank 에
// 넣는다. 순위는 축 안에서의 밀집 순위(동점 동순위)이고 통과 못 한 축은
// 세지 않는다.
func assignRanks(pairs []Pair) {
	term := axisRanks(pairs, func(p Pair) bool { return p.has(AxisTerm) }, func(p Pair) float64 { return p.termScore })
	embed := axisRanks(pairs, func(p Pair) bool { return p.has(AxisEmbed) }, func(p Pair) float64 { return p.embedScore })
	for i := range pairs {
		pairs[i].rank = term[i]
		if embed[i] > 0 && (pairs[i].rank == 0 || embed[i] < pairs[i].rank) {
			pairs[i].rank = embed[i]
		}
	}
}

// axisRanks는 축을 통과한 쌍의 점수로 밀집 순위를 맨다. 통과 못 한
// 쌍은 0 이다. 동점은 같은 순위를 받고 다음 순위는 건너뛴다(1, 2, 2, 4).
func axisRanks(pairs []Pair, passed func(Pair) bool, score func(Pair) float64) []int {
	var idx []int
	for i, p := range pairs {
		if passed(p) {
			idx = append(idx, i)
		}
	}
	sort.Slice(idx, func(a, b int) bool {
		sa, sb := score(pairs[idx[a]]), score(pairs[idx[b]])
		if sa != sb {
			return sa > sb
		}
		// 점수 동점의 순서를 슬러그로 고정해 순위 산정 자체가 결정적이게
		// 한다. 동점은 같은 순위를 받으므로 이 순서는 결과에 남지 않는다.
		return pairs[idx[a]].A+pairs[idx[a]].B < pairs[idx[b]].A+pairs[idx[b]].B
	})
	ranks := make([]int, len(pairs))
	for n, i := range idx {
		rank := n + 1
		if n > 0 && score(pairs[idx[n]]) == score(pairs[idx[n-1]]) {
			rank = ranks[idx[n-1]]
		}
		ranks[i] = rank
	}
	return ranks
}

// has는 쌍이 축을 통과했는지 본다.
func (p Pair) has(axis string) bool {
	for _, a := range p.Axes {
		if a == axis {
			return true
		}
	}
	return false
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

// dot은 L2 정규화된 임베딩 벡터 둘의 내적을 낸다. 정규화되어 있으므로
// 이 값이 곧 코사인 유사도다. 제곱근을 다시 계산하지 않는다(ADR 0075).
// 어느 쪽이 nil 이면 임베딩이 없는 것이므로 0 을 낸다.
func dot(a, b []float32) float64 {
	if a == nil || b == nil {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}
