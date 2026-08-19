package bridge

import (
	"math"
	"path"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/graph"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/state"
	"github.com/neocode24/engram/internal/walk"
)

// parseDoc는 본문 문자열로 순회 문서 하나를 만든다.
func parseDoc(t *testing.T, rel, content string) walk.Doc {
	t.Helper()
	parsed, err := doc.Parse(rel, []byte(content))
	if err != nil {
		t.Fatalf("%s 파싱 실패: %v", rel, err)
	}
	return walk.Doc{Rel: rel, Content: content, Parsed: parsed}
}

// mkIndex는 경로와 TF로 색인을 만든다. 슬러그를 비우면 실제 색인과
// 같이 파일명에서 확장자를 뗀 값으로 채운다.
func mkIndex(docs ...index.DocEntry) *index.Index {
	out := append([]index.DocEntry{}, docs...)
	for i := range out {
		if out[i].Slug == "" {
			base := strings.TrimSuffix(path.Base(out[i].Path), ".md")
			out[i].Slug = base
		}
	}
	return &index.Index{Docs: out}
}

func TestCosine(t *testing.T) {
	same := map[string]float64{"go": 1, "언어": 2}
	if got := cosine(same, same); math.Abs(got-1) > 1e-9 {
		t.Errorf("같은 벡터의 코사인은 1: %v", got)
	}
	if got := cosine(same, map[string]float64{"rust": 1}); got != 0 {
		t.Errorf("겹치지 않으면 0: %v", got)
	}
	if got := cosine(same, nil); got != 0 {
		t.Errorf("빈 벡터는 0: %v", got)
	}
}

func TestRunFindsSimilarUnlinkedPair(t *testing.T) {
	// go-rust 유사도는 (1*1+2*2)/(sqrt(5)*sqrt(5)) = 0.8.
	ix := mkIndex(
		index.DocEntry{Path: "context/go.md", TF: map[string]float64{"go": 1, "언어": 2}},
		index.DocEntry{Path: "context/rust.md", TF: map[string]float64{"rust": 1, "언어": 2}},
		index.DocEntry{Path: "context/tea.md", TF: map[string]float64{"차": 5}},
	)
	g := graph.Build([]walk.Doc{
		parseDoc(t, "context/go.md", "go 는 언어"),
		parseDoc(t, "context/rust.md", "rust 는 언어"),
		parseDoc(t, "context/tea.md", "차"),
	})
	res := Run(ix, g, state.State{}, Options{Min: 0.3, Limit: 10})
	if len(res.Pairs) != 1 {
		t.Fatalf("후보는 go-rust 하나여야 합니다: %+v", res.Pairs)
	}
	p := res.Pairs[0]
	if p.A != "go" || p.B != "rust" {
		t.Errorf("쌍: %+v", p)
	}
	if p.Score < 0.79 || p.Score > 0.81 {
		t.Errorf("go 와 rust 의 유사도는 0.8 이어야 합니다: %v", p.Score)
	}
	if res.Stats.ContextDocs != 3 || res.Stats.BelowMin != 2 {
		t.Errorf("통계: %+v", res.Stats)
	}
}

func TestRunExcludesLinkedPairsBothDirections(t *testing.T) {
	tf := map[string]float64{"go": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: tf},
		index.DocEntry{Path: "context/b.md", TF: tf},
		index.DocEntry{Path: "context/c.md", TF: tf},
		index.DocEntry{Path: "context/d.md", TF: tf},
	)
	// a 가 b 를 본문에서 가리키고, d 가 c 를 related 에서 가리킨다.
	// 방향만 다를 뿐 둘 다 이어진 쌍이다.
	g := graph.Build([]walk.Doc{
		parseDoc(t, "context/a.md", "[[b]] 를 봅니다"),
		parseDoc(t, "context/b.md", "b"),
		parseDoc(t, "context/c.md", "c"),
		parseDoc(t, "context/d.md", "---\nrelated:\n  - \"[[c]]\"\n---\n\nd"),
	})
	res := Run(ix, g, state.State{}, Options{Min: 0.3, Limit: 10})
	if len(res.Pairs) != 4 {
		t.Fatalf("남는 쌍은 a-c, a-d, b-c, b-d 넷: %+v", res.Pairs)
	}
	if res.Stats.Linked != 2 {
		t.Errorf("이어진 쌍 2개를 세야 합니다: %+v", res.Stats)
	}
}

func TestRunExcludesRejectedPair(t *testing.T) {
	tf := map[string]float64{"go": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: tf},
		index.DocEntry{Path: "context/b.md", TF: tf},
	)
	g := graph.Build(nil)
	var st state.State
	st.Reject("a", "b")
	res := Run(ix, g, st, Options{Min: 0.3, Limit: 10})
	if len(res.Pairs) != 0 {
		t.Errorf("기각한 쌍은 다시 나오지 않아야 합니다: %+v", res.Pairs)
	}
	if res.Stats.Rejected != 1 {
		t.Errorf("기각 통계: %+v", res.Stats)
	}
}

func TestRunComparesContextOnly(t *testing.T) {
	tf := map[string]float64{"go": 1}
	ix := mkIndex(
		index.DocEntry{Path: "inbox/a.md", TF: tf},
		index.DocEntry{Path: "sources/b.md", TF: tf},
		index.DocEntry{Path: "archive/c.md", TF: tf},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, Limit: 10})
	if res.Stats.ContextDocs != 0 || len(res.Pairs) != 0 {
		t.Errorf("inbox/sources/archive 는 비교 대상이 아닙니다: %+v", res)
	}
}

func TestRunSortsByScoreThenSlug(t *testing.T) {
	// a-d 는 같은 벡터로 유사도 1. 나머지는 1/sqrt(2) 로 정확히 동점이고,
	// 동점은 첫 슬러그, 그다음 둘째 슬러그 오름차순이다.
	xy := map[string]float64{"x": 1, "y": 1}
	x := map[string]float64{"x": 1}
	y := map[string]float64{"y": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: xy},
		index.DocEntry{Path: "context/b.md", TF: x},
		index.DocEntry{Path: "context/c.md", TF: y},
		index.DocEntry{Path: "context/d.md", TF: xy},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.1, Limit: 10})
	if len(res.Pairs) != 5 {
		t.Fatalf("후보 5개: %+v", res.Pairs)
	}
	want := []string{"a d", "a b", "a c", "b d", "c d"}
	for i, p := range res.Pairs {
		if got := p.A + " " + p.B; got != want[i] {
			t.Errorf("%d번째 쌍 %q, want %q (%+v)", i, got, want[i], res.Pairs)
		}
	}
}

func TestRunLimit(t *testing.T) {
	tf := map[string]float64{"go": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: tf},
		index.DocEntry{Path: "context/b.md", TF: tf},
		index.DocEntry{Path: "context/c.md", TF: tf},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, Limit: 2})
	if len(res.Pairs) != 2 {
		t.Errorf("상한 2: %+v", res.Pairs)
	}
}

// sameAxes는 축 목록이 순서까지 같은지 본다.
func sameAxes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRunIsDeterministic(t *testing.T) {
	tf := map[string]float64{"go": 1, "언어": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: tf},
		index.DocEntry{Path: "context/b.md", TF: tf},
		index.DocEntry{Path: "context/c.md", TF: tf},
	)
	first := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, Limit: 10})
	second := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, Limit: 10})
	if len(first.Pairs) != len(second.Pairs) {
		t.Fatal("길이가 다릅니다")
	}
	for i := range first.Pairs {
		a, b := first.Pairs[i], second.Pairs[i]
		if a.A != b.A || a.B != b.B || a.Score != b.Score || !sameAxes(a.Axes, b.Axes) {
			t.Errorf("%d번째 결과가 다릅니다: %+v vs %+v", i, a, b)
		}
	}
}

// mkVecs는 경로별 단위 벡터 맵을 만든다. 벡터는 이미 L2 정규화된
// 것으로 본다. 내적이 곧 코사인이다.
func mkVecs(kv ...any) map[string][]float32 {
	out := map[string][]float32{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1].([]float32)
	}
	return out
}

func TestRunUnionOfAxes(t *testing.T) {
	// a-b 는 단어만 겹친다(임베딩은 직교). a-c 는 임베딩만 같은 방향이다
	// (단어는 겹치지 않는다). 합집합이므로 둘 다 나와야 한다. 교집합이면
	// 아무것도 나오지 않는다.
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: map[string]float64{"x": 1}},
		index.DocEntry{Path: "context/b.md", TF: map[string]float64{"x": 1, "y": 1}},
		index.DocEntry{Path: "context/c.md", TF: map[string]float64{"z": 1}},
	)
	vecs := mkVecs(
		"context/a.md", []float32{1, 0},
		"context/b.md", []float32{0, 1},
		"context/c.md", []float32{1, 0},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.72, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 2 {
		t.Fatalf("단어 축과 임베딩 축의 합집합은 a-b, a-c 둘이어야 합니다: %+v", res.Pairs)
	}
	byPair := map[string]Pair{}
	for _, p := range res.Pairs {
		byPair[p.A+"-"+p.B] = p
	}
	if p := byPair["a-b"]; !sameAxes(p.Axes, []string{AxisTerm}) {
		t.Errorf("a-b 는 단어 축만 잡아야 합니다: %+v", p.Axes)
	}
	if p := byPair["a-c"]; !sameAxes(p.Axes, []string{AxisEmbed}) {
		t.Errorf("a-c 는 임베딩 축만 잡아야 합니다: %+v", p.Axes)
	}
	// b-c 는 두 축 모두 0 이므로 어느 합집합에도 없다.
	if _, ok := byPair["b-c"]; ok {
		t.Error("두 축 모두 하한 미달인 b-c 가 나왔습니다")
	}
}

func TestRunAxesHaveSeparateFloors(t *testing.T) {
	// 같은 쌍의 단어 점수는 0.707, 임베딩 점수는 0.65 로 고정한다.
	// 단어 하한과 임베딩 하한이 별개 값이므로 어느 쪽이 쌍을 살리는지
	// 하한 조합이 결정한다. 하나의 하한을 두 축에 걸면 이 판정이 무너진다.
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: map[string]float64{"x": 1}},
		index.DocEntry{Path: "context/b.md", TF: map[string]float64{"x": 1, "y": 1}},
	)
	vecs := mkVecs(
		"context/a.md", []float32{1, 0},
		"context/b.md", []float32{0.65, 0.76},
	)
	// 단어 0.3 통과, 임베딩 0.65 는 0.72 에 미달: 단어 축만.
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.72, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 1 || !sameAxes(res.Pairs[0].Axes, []string{AxisTerm}) {
		t.Fatalf("단어 축만 잡아야 합니다: %+v", res.Pairs)
	}
	// 단어 0.8 미달, 임베딩 0.65 는 0.6 통과: 임베딩 축만.
	res = Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.8, EmbedMin: 0.6, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 1 || !sameAxes(res.Pairs[0].Axes, []string{AxisEmbed}) {
		t.Fatalf("임베딩 축만 잡아야 합니다: %+v", res.Pairs)
	}
	// 둘 다 통과하면 축 둘을 밝힌다.
	res = Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.6, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 1 || !sameAxes(res.Pairs[0].Axes, []string{AxisTerm, AxisEmbed}) {
		t.Fatalf("두 축을 모두 밝혀야 합니다: %+v", res.Pairs)
	}
}

func TestRunEmbedAxisOffWithoutVectors(t *testing.T) {
	// 벡터가 없으면 임베딩 하한이 0 이어도 임베딩 축은 켜지지 않는다.
	// 단어는 통과하지 못하므로 쌍이 없다.
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: map[string]float64{"x": 1}},
		index.DocEntry{Path: "context/b.md", TF: map[string]float64{"z": 1}},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0, Limit: 10})
	if len(res.Pairs) != 0 {
		t.Errorf("벡터 없이는 임베딩 축이 켜지지 않아야 합니다: %+v", res.Pairs)
	}
	// 벡터 맵이 있어도 문서 하나가 빠지면 그 문서는 임베딩 축 후보가 아니다.
	vecs := mkVecs("context/a.md", []float32{1, 0})
	res = Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.5, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 0 {
		t.Errorf("한쪽 벡터가 빠진 쌍은 임베딩 축에서 나오지 않아야 합니다: %+v", res.Pairs)
	}
}

func TestRunSortsAcrossAxes(t *testing.T) {
	// 임베딩 축이 잡은 강한 쌍(1.0)이 단어 축이 잡은 약한 쌍(0.707)
	// 보다 위에 오는지 본다. 대표 점수는 두 축의 최댓값이다.
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: map[string]float64{"x": 1}},
		index.DocEntry{Path: "context/b.md", TF: map[string]float64{"x": 1, "y": 1}},
		index.DocEntry{Path: "context/c.md", TF: map[string]float64{"z": 1}},
	)
	vecs := mkVecs(
		"context/a.md", []float32{1, 0},
		"context/b.md", []float32{0, 1},
		"context/c.md", []float32{1, 0},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.72, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 2 {
		t.Fatalf("쌍 둘: %+v", res.Pairs)
	}
	if res.Pairs[0].A != "a" || res.Pairs[0].B != "c" {
		t.Errorf("임베딩 점수 1.0 인 a-c 가 먼저 나와야 합니다: %+v", res.Pairs)
	}
}

func TestRunRankKeyDoesNotBuryTermAxis(t *testing.T) {
	// 임베딩 점수 분포가 단어 점수보다 체계적으로 높은 위키를 만든다.
	// a-b 는 단어 축 1위(0.707)인데 임베딩은 없다. c-d 는 임베딩 1위(1.0),
	// c-e 와 d-e 는 임베딩 2위(0.898)다. 최댓값을 키로 쓰면 a-b 는
	// 0.707 로 꼴찌가 된다. 축 순위 키라면 a-b 는 단어 축 1위로 최상단
	// 그룹에 오른다. 이 시험이 순위 키를 지킨다.
	x := map[string]float64{"x": 1}
	xy := map[string]float64{"x": 1, "y": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: x},
		index.DocEntry{Path: "context/b.md", TF: xy},
		index.DocEntry{Path: "context/c.md", TF: map[string]float64{"q": 1}},
		index.DocEntry{Path: "context/d.md", TF: map[string]float64{"r": 1}},
		index.DocEntry{Path: "context/e.md", TF: map[string]float64{"s": 1}},
	)
	vecs := mkVecs(
		"context/c.md", []float32{1, 0},
		"context/d.md", []float32{1, 0},
		"context/e.md", []float32{0.9, 0.44},
	)
	res := Run(ix, graph.Build(nil), state.State{}, Options{Min: 0.3, EmbedMin: 0.8, Limit: 10, Vectors: vecs})
	if len(res.Pairs) != 4 {
		t.Fatalf("쌍 넷: %+v", res.Pairs)
	}
	want := []struct{ a, b string }{
		{"c", "d"}, {"a", "b"}, {"c", "e"}, {"d", "e"},
	}
	for i, w := range want {
		if res.Pairs[i].A != w.a || res.Pairs[i].B != w.b {
			t.Fatalf("%d번째 쌍 = %s-%s, want %s-%s (전체 %+v)",
				i, res.Pairs[i].A, res.Pairs[i].B, w.a, w.b, res.Pairs)
		}
	}
	if !sameAxes(res.Pairs[1].Axes, []string{AxisTerm}) {
		t.Errorf("a-b 는 단어 축만 통과해야 합니다: %+v", res.Pairs[1].Axes)
	}
}
