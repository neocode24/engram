package chunk

import (
	"strings"
	"testing"
)

func TestSplit(t *testing.T) {
	t.Run("헤딩이 없으면 문서 전체가 조각 하나다", func(t *testing.T) {
		body := "첫 줄입니다.\n두 번째 줄입니다.\n"
		got := Split(body)
		if len(got) != 1 {
			t.Fatalf("조각 수 = %d, want 1: %+v", len(got), got)
		}
		c := got[0]
		if c.Heading != "" || len(c.Path) != 0 {
			t.Errorf("헤딩 없는 조각의 Heading/Path = %q/%v", c.Heading, c.Path)
		}
		if c.StartLine != 1 || c.EndLine != 2 {
			t.Errorf("줄 범위 = %d-%d, want 1-2", c.StartLine, c.EndLine)
		}
		if c.Body != body {
			t.Errorf("본문이 원문과 다름:\n%q\nwant:\n%q", c.Body, body)
		}
	})

	t.Run("빈 본문은 조각이 없다", func(t *testing.T) {
		if got := Split(""); got != nil {
			t.Errorf("빈 본문의 조각 = %+v, want nil", got)
		}
	})

	t.Run("문서 제목과 서두는 첫 조각이 되고 헤딩은 상위 경로를 쌓는다", func(t *testing.T) {
		body := "# 제목\n\n서두 설명\n\n" +
			"## 판단 근거\n\n본문 A\n\n" +
			"### 저장 형식\n\n본문 B\n\n" +
			"## 결과\n\n본문 C\n"
		got := Split(body)
		if len(got) != 4 {
			t.Fatalf("조각 수 = %d, want 4: %+v", len(got), got)
		}
		want := []struct {
			heading   string
			path      []string
			start     int
			end       int
			firstLine string
		}{
			{"", []string{}, 1, 4, "# 제목"},
			{"판단 근거", []string{}, 5, 8, "## 판단 근거"},
			{"저장 형식", []string{"판단 근거"}, 9, 12, "### 저장 형식"},
			{"결과", []string{}, 13, 15, "## 결과"},
		}
		for i, w := range want {
			c := got[i]
			if c.Heading != w.heading {
				t.Errorf("조각 %d Heading = %q, want %q", i, c.Heading, w.heading)
			}
			if strings.Join(c.Path, "/") != strings.Join(w.path, "/") {
				t.Errorf("조각 %d Path = %v, want %v", i, c.Path, w.path)
			}
			if c.StartLine != w.start || c.EndLine != w.end {
				t.Errorf("조각 %d 줄 범위 = %d-%d, want %d-%d", i, c.StartLine, c.EndLine, w.start, w.end)
			}
			if first := strings.SplitN(c.Body, "\n", 2)[0]; first != w.firstLine {
				t.Errorf("조각 %d 첫 줄 = %q, want %q", i, first, w.firstLine)
			}
		}
		// 조각을 이어 붙이면 원문이 그대로 돌아온다.
		var joined strings.Builder
		for _, c := range got {
			joined.WriteString(c.Body)
		}
		if joined.String() != body {
			t.Fatalf("조각을 이어 붙여도 원문이 안 됨:\n%q\nwant:\n%q", joined.String(), body)
		}
	})

	t.Run("코드 펜스 안의 # 는 헤딩으로 잡히지 않는다", func(t *testing.T) {
		body := "## 배포 절차\n\n" +
			"```bash\n" +
			"# 이 줄은 셸 주석이다\n" +
			"### 이것도 셸 주석이다\n" +
			"```\n\n" +
			"## 다음 절\n"
		got := Split(body)
		if len(got) != 2 {
			t.Fatalf("펜스 안 헤딩을 경계로 세면 안 됩니다. 조각 수 = %d: %+v", len(got), got)
		}
		if !strings.Contains(got[0].Body, "# 이 줄은 셸 주석이다") {
			t.Errorf("펜스 내용이 첫 조각 본문에 없음: %q", got[0].Body)
		}
		if got[1].Heading != "다음 절" {
			t.Errorf("펜스 닫힌 뒤 헤딩 = %q, want %q", got[1].Heading, "다음 절")
		}
	})

	t.Run("펜스가 닫히지 않아도 이후 헤딩을 노리지 않는다", func(t *testing.T) {
		body := "## 절\n\n```\n# 펜스가 닫히지 않은 채 끝난다\n"
		got := Split(body)
		if len(got) != 1 || got[0].Heading != "절" {
			t.Fatalf("미종결 펜스 뒤의 # 를 헤딩으로 잡으면 안 됩니다: %+v", got)
		}
	})

	t.Run("# 하나짜리 문서 제목은 경계가 아니다", func(t *testing.T) {
		body := "# 큰 제목\n# 두 번째 큰 제목\n\n## 절\n\n본문\n"
		got := Split(body)
		if len(got) != 2 {
			t.Fatalf("조각 수 = %d, want 2: %+v", len(got), got)
		}
		if got[0].Heading != "" || !strings.Contains(got[0].Body, "# 두 번째 큰 제목") {
			t.Errorf("문서 제목 줄이 첫 조각 본문에 있어야 함: %+v", got[0])
		}
		if got[1].Heading != "절" {
			t.Errorf("둘째 조각 Heading = %q, want %q", got[1].Heading, "절")
		}
	})

	t.Run("끝 줄에 개행이 없어도 원문 그대로다", func(t *testing.T) {
		body := "## 절\n\n본문 마지막 줄"
		got := Split(body)
		if len(got) != 1 {
			t.Fatalf("조각 수 = %d, want 1: %+v", len(got), got)
		}
		if got[0].Body != body {
			t.Errorf("본문 = %q, want %q", got[0].Body, body)
		}
		if got[0].EndLine != 3 {
			t.Errorf("끝 줄 = %d, want 3", got[0].EndLine)
		}
	})

	t.Run("헤딩 표시 뒤에 공백이 없으면 헤딩이 아니다", func(t *testing.T) {
		body := "##절 이 아니다\n\n## 절 이다\n"
		got := Split(body)
		if len(got) != 2 {
			t.Fatalf("조각 수 = %d, want 2: %+v", len(got), got)
		}
		// ##절 줄은 경계가 아니므로 그대로 첫 조각 본문에 남는다.
		if got[0].Heading != "" || !strings.Contains(got[0].Body, "##절 이 아니다") {
			t.Errorf("##절 줄은 첫 조각 본문에 있어야 함: %+v", got[0])
		}
		if got[1].Heading != "절 이다" {
			t.Errorf("둘째 조각 Heading = %q, want %q", got[1].Heading, "절 이다")
		}
	})
}
