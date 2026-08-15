package doc

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMap(t *testing.T) {
	t.Run("키 순서는 표준 순서를 따르고 나머지는 이름순이다", func(t *testing.T) {
		got := string(RenderMap(map[string]any{
			"zebra":          "z",
			"related":        []string{"b"},
			"type":           "concept",
			"alpha":          "a",
			"artifact_stage": "context",
		}, "본문\n"))
		want := "---\n" +
			"type: concept\n" +
			"artifact_stage: context\n" +
			"related:\n  - b\n" +
			"alpha: a\n" +
			"zebra: z\n" +
			"---\n본문\n"
		if got != want {
			t.Errorf("직렬화 결과가 다름:\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("빈 값 키와 빈 배열을 계약대로 낸다", func(t *testing.T) {
		got := string(RenderMap(map[string]any{
			"source_channel": nil,
			"tags":           []string{},
			"indexable":      false,
		}, ""))
		want := "---\n" +
			"source_channel:\n" +
			"tags: []\n" +
			"indexable: false\n" +
			"---\n"
		if got != want {
			t.Errorf("직렬화 결과가 다름:\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("목록으로 읽히는 문자열은 인용해 왕복한다", func(t *testing.T) {
		got := string(RenderMap(map[string]any{
			"related": []string{"[[다른 문서]]"},
		}, ""))
		if !strings.Contains(got, `- "[[다른 문서]]"`) {
			t.Errorf("위키링크 항목이 인용되지 않음:\n%s", got)
		}
		re, err := Parse("roundtrip.md", []byte(got))
		if err != nil {
			t.Fatalf("재파싱 실패: %v", err)
		}
		if re.Fields[0].List[0] != "[[다른 문서]]" {
			t.Errorf("왕복 후 값이 다름: %q", re.Fields[0].List[0])
		}
	})

	t.Run("본문 끝에 개행을 보장한다", func(t *testing.T) {
		got := RenderMap(map[string]any{"type": "concept"}, "개행 없는 본문")
		if !bytes.HasSuffix(got, []byte("본문\n")) {
			t.Errorf("본문 끝에 개행이 없음: %q", got)
		}
	})
}

// TestRenderRoundTrip는 골든 위키 문서를 파싱한 뒤 다시 직렬화하면
// 원본과 바이트가 같은지 본다. ADR 0005의 프론트매터 정규화 parity의
// 전제다. fixture는 읽기만 한다.
func TestRenderRoundTrip(t *testing.T) {
	const fixtureDir = "../../harness/fixtures/golden-wiki"
	paths, err := filepath.Glob(filepath.Join(fixtureDir, "*", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Glob(filepath.Join(fixtureDir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths, root...)
	if len(paths) == 0 {
		t.Fatal("골든 위키 문서를 찾지 못했다")
	}

	var skipped, diffs []string
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(p, raw)
		if err != nil {
			// 파싱부터 실패하는 고의적 위반 문서는 왕복 대상이 아니다.
			skipped = append(skipped, filepath.Base(p))
			continue
		}
		if got := Render(d.Fields, d.Body); !bytes.Equal(raw, got) {
			diffs = append(diffs, filepath.Base(p))
		}
	}
	// CRLF로 저장된 문서는 파싱 단계에서 LF로 정규화되므로 바이트가 달라진다.
	// 직렬화 계약이 LF이므로 이 차이는 정상이다.
	crlf := map[string]bool{"crlf-meeting-note.md": true}
	for _, name := range diffs {
		if crlf[name] {
			continue
		}
		t.Errorf("왕복 결과가 원본과 다름: %s", name)
	}
	t.Logf("왕복 %d건, 파싱 실패로 건너뜀 %d건(%s), CRLF 정규화 차이 %d건",
		len(paths)-len(skipped)-len(diffs), len(skipped), strings.Join(skipped, ", "), countIn(diffs, crlf))
}

// countIn은 목록에서 조건에 들어가는 이름 수를 센다.
func countIn(names []string, set map[string]bool) int {
	n := 0
	for _, name := range names {
		if set[name] {
			n++
		}
	}
	return n
}
