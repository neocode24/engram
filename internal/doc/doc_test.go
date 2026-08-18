package doc

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("프론트매터가 있으면 값 타입을 구분해 파싱한다", func(t *testing.T) {
		src := "---\n" +
			"title: 승급 게이트\n" +
			"promoted: true\n" +
			"created: 2026-01-01\n" +
			"source_channel:\n" +
			"related:\n" +
			"  - go-scaffold\n" +
			"  - adr-0017\n" +
			"---\n" +
			"본문 첫 줄\n"
		d, err := Parse("wiki/a.md", []byte(src))
		if err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		if !d.HasFrontmatter {
			t.Error("프론트매터가 있다고 나와야 함")
		}
		if len(d.Fields) != 5 {
			t.Fatalf("필드 5개여야 함, got %d", len(d.Fields))
		}
		want := []struct {
			key  string
			kind ValueKind
		}{
			{"title", KindString},
			{"promoted", KindBool},
			{"created", KindDate},
			{"source_channel", KindEmpty},
			{"related", KindStringList},
		}
		for i, w := range want {
			if d.Fields[i].Key != w.key || d.Fields[i].Kind != w.kind {
				t.Errorf("필드 %d: got (%s, %d), want (%s, %d)", i, d.Fields[i].Key, d.Fields[i].Kind, w.key, w.kind)
			}
		}
		if d.Fields[2].Str != "2026-01-01" {
			t.Errorf("created 값: got %q", d.Fields[2].Str)
		}
		if !d.Fields[1].Bool {
			t.Error("promoted는 true여야 함")
		}
		if got := d.Fields[4].List; len(got) != 2 || got[0] != "go-scaffold" || got[1] != "adr-0017" {
			t.Errorf("related 목록: got %v", got)
		}
		if d.Body != "본문 첫 줄\n" {
			t.Errorf("본문: got %q", d.Body)
		}
		if d.BodyLine != 10 {
			t.Errorf("본문 시작 줄: got %d, want 10", d.BodyLine)
		}
	})

	t.Run("프론트매터가 없으면 본문 전체가 된다", func(t *testing.T) {
		d, err := Parse("wiki/a.md", []byte("그냥 본문\n둘째 줄\n"))
		if err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		if d.HasFrontmatter || len(d.Fields) != 0 {
			t.Error("프론트매터 없음이어야 함")
		}
		if d.Body != "그냥 본문\n둘째 줄\n" || d.BodyLine != 1 {
			t.Errorf("본문/줄 번호: got %q, %d", d.Body, d.BodyLine)
		}
	})

	t.Run("키 순서를 보존한다", func(t *testing.T) {
		src := "---\nz: 1\na: 2\nm: 3\n---\n"
		d, err := Parse("wiki/order.md", []byte(src))
		if err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		got := ""
		for _, f := range d.Fields {
			got += f.Key
		}
		if got != "zam" {
			t.Errorf("키 순서: got %q, want %q", got, "zam")
		}
	})

	t.Run("닫는 구분자가 없으면 시작 줄을 담아 에러다", func(t *testing.T) {
		_, err := Parse("wiki/broken.md", []byte("---\ntitle: x\n본문뿐\n"))
		if err == nil {
			t.Fatal("에러여야 함")
		}
		if !strings.Contains(err.Error(), "wiki/broken.md") || !strings.Contains(err.Error(), "1") {
			t.Errorf("에러에 경로와 시작 줄이 없음: %v", err)
		}
	})

	t.Run("YAML 실패 메시지에 경로와 줄 번호를 담는다", func(t *testing.T) {
		_, err := Parse("wiki/bad-yaml.md", []byte("---\ntitle: [unclosed\n---\n"))
		if err == nil {
			t.Fatal("에러여야 함")
		}
		if !strings.Contains(err.Error(), "wiki/bad-yaml.md") {
			t.Errorf("에러에 경로가 없음: %v", err)
		}
		if !strings.Contains(err.Error(), "줄") {
			t.Errorf("에러에 줄 표기가 없음: %v", err)
		}
	})

	t.Run("BOM을 허용한다", func(t *testing.T) {
		src := "\xEF\xBB\xBF---\ntitle: x\n---\n본문\n"
		d, err := Parse("wiki/bom.md", []byte(src))
		if err != nil {
			t.Fatalf("BOM 파일은 파싱되어야 함: %v", err)
		}
		if d.Fields[0].Key != "title" || d.Fields[0].Str != "x" {
			t.Errorf("BOM 제거 후 파싱: got %+v", d.Fields)
		}
		if d.Body != "본문\n" {
			t.Errorf("본문: got %q", d.Body)
		}
	})

	t.Run("CRLF 파일에서 줄 번호가 틀어지지 않는다", func(t *testing.T) {
		src := strings.ReplaceAll("---\ntitle: x\nrelated:\n  - z\n---\n본문\n[[link]]\n", "\n", "\r\n")
		d, err := Parse("wiki/crlf.md", []byte(src))
		if err != nil {
			t.Fatalf("CRLF 파일은 파싱되어야 함: %v", err)
		}
		if d.BodyLine != 6 {
			t.Errorf("본문 시작 줄: got %d, want 6", d.BodyLine)
		}
		links := d.BodyLinks()
		if len(links) != 1 || links[0].Line != 7 {
			t.Errorf("링크 줄: got %+v, want 줄 7", links)
		}
		if fm := d.FrontmatterLinks(); len(fm) != 1 || fm[0].Line != 4 {
			t.Errorf("related 링크 줄: got %+v, want 줄 4", fm)
		}
	})
}

func TestBodyLinks(t *testing.T) {
	t.Run("기본, 별칭, 헤딩 형태를 지원한다", func(t *testing.T) {
		d, _ := Parse("a.md", []byte("[[go]]\n[[go-scaffold|스캐폴드]]\n[[adr-0017#결정]]\n"))
		got := d.BodyLinks()
		want := []Link{{"go", 1}, {"go-scaffold", 2}, {"adr-0017", 3}}
		if len(got) != len(want) {
			t.Fatalf("링크 수: got %d", len(got))
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("링크 %d: got %+v, want %+v", i, got[i], w)
			}
		}
	})

	t.Run("코드 펜스 안은 링크가 아니다", func(t *testing.T) {
		d, _ := Parse("a.md", []byte("[[real]]\n```\n[[in-fence]]\n```\n[[real2]]\n"))
		got := d.BodyLinks()
		if len(got) != 2 || got[0].Slug != "real" || got[1].Slug != "real2" {
			t.Errorf("펜스 밖 링크만 나와야 함: %+v", got)
		}
	})

	t.Run("인라인 코드 안은 링크가 아니다", func(t *testing.T) {
		d, _ := Parse("a.md", []byte("`[[in-code]]` 그리고 [[out]]\n"))
		got := d.BodyLinks()
		if len(got) != 1 || got[0].Slug != "out" {
			t.Errorf("인라인 코드 밖 링크만 나와야 함: %+v", got)
		}
	})

	t.Run("중복 링크를 제거하지 않는다", func(t *testing.T) {
		d, _ := Parse("a.md", []byte("[[dup]]\n[[dup]]\n"))
		if got := d.BodyLinks(); len(got) != 2 {
			t.Errorf("중복 유지: got %d개", len(got))
		}
	})

	t.Run("프론트매터가 있을 때 줄 번호는 파일 기준이다", func(t *testing.T) {
		src := "---\ntitle: x\nrelated:\n  - [[other]]\n---\n첫 줄\n[[second]]\n"
		d, _ := Parse("a.md", []byte(src))
		if got := d.BodyLinks(); len(got) != 1 || got[0].Line != 7 {
			t.Errorf("본문 링크 줄: got %+v, want 7", got)
		}
		fm := d.FrontmatterLinks()
		if len(fm) != 1 || fm[0].Slug != "other" || fm[0].Line != 4 {
			t.Errorf("related 링크: got %+v, want {other 4}", fm)
		}
	})
}

// TestMarkdownBodyLinks는 ADR 0065의 마크다운 링크 추출을 확인한다.
func TestMarkdownBodyLinks(t *testing.T) {
	t.Run("마크다운 링크를 관계로 센다", func(t *testing.T) {
		src := "[고 스캐폴드](context/go-scaffold.md)\n[옆 문서](sibling.md)\n[증거](../sources/2026-01-01-bar.md)\n"
		got := bodyLinksOf(t, src)
		want := []Link{{"go-scaffold", 1}, {"sibling", 2}, {"2026-01-01-bar", 3}}
		if len(got) != len(want) {
			t.Fatalf("링크 수: got %d (%+v)", len(got), got)
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("링크 %d: got %+v, want %+v", i, got[i], w)
			}
		}
	})

	t.Run("스킴과 .md 아닌 경로와 앵커는 관계가 아니다", func(t *testing.T) {
		src := "[웹](https://example.com/a.md)\n[메일](mailto:a@b.md)\n" +
			"![그림](img/diagram.png)\n[문서](notes.txt)\n[안](#결정)\n[진짜](real.md)\n"
		got := bodyLinksOf(t, src)
		if len(got) != 1 || got[0].Slug != "real" {
			t.Errorf("위키 안 문서만 나와야 함: %+v", got)
		}
	})

	t.Run("코드 펜스와 인라인 코드 안은 관계가 아니다", func(t *testing.T) {
		src := "```\n[펜스](in-fence.md)\n```\n`[코드](in-code.md)` 그리고 [밖](out.md)\n"
		got := bodyLinksOf(t, src)
		if len(got) != 1 || got[0].Slug != "out" {
			t.Errorf("코드 밖 링크만 나와야 함: %+v", got)
		}
	})

	t.Run("두 문법이 섞여도 등장 순서를 지킨다", func(t *testing.T) {
		got := bodyLinksOf(t, "[[first]] 그리고 [둘째](second.md) 그리고 [[third]]\n")
		if len(got) != 3 || got[0].Slug != "first" || got[1].Slug != "second" || got[2].Slug != "third" {
			t.Errorf("등장 순서: %+v", got)
		}
	})
}

// bodyLinksOf는 본문만 있는 문서를 파싱해 본문 링크를 낸다.
func bodyLinksOf(t *testing.T, body string) []Link {
	t.Helper()
	d, err := Parse("a.md", []byte(body))
	if err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	return d.BodyLinks()
}
