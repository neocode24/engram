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
	res := Run(ix, g, state.State{}, 0.3, 10)
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
	res := Run(ix, g, state.State{}, 0.3, 10)
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
	res := Run(ix, g, st, 0.3, 10)
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
	res := Run(ix, graph.Build(nil), state.State{}, 0.3, 10)
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
	res := Run(ix, graph.Build(nil), state.State{}, 0.1, 10)
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
	res := Run(ix, graph.Build(nil), state.State{}, 0.3, 2)
	if len(res.Pairs) != 2 {
		t.Errorf("상한 2: %+v", res.Pairs)
	}
}

func TestRunIsDeterministic(t *testing.T) {
	tf := map[string]float64{"go": 1, "언어": 1}
	ix := mkIndex(
		index.DocEntry{Path: "context/a.md", TF: tf},
		index.DocEntry{Path: "context/b.md", TF: tf},
		index.DocEntry{Path: "context/c.md", TF: tf},
	)
	first := Run(ix, graph.Build(nil), state.State{}, 0.3, 10)
	second := Run(ix, graph.Build(nil), state.State{}, 0.3, 10)
	if len(first.Pairs) != len(second.Pairs) {
		t.Fatal("길이가 다릅니다")
	}
	for i := range first.Pairs {
		if first.Pairs[i] != second.Pairs[i] {
			t.Errorf("%d번째 결과가 다릅니다: %+v vs %+v", i, first.Pairs[i], second.Pairs[i])
		}
	}
}
