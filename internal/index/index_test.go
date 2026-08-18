package index

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/walk"
)

// writeWiki는 임시 디렉토리에 문서를 만들고 순회 결과를 반환한다.
func writeWiki(t *testing.T, files map[string]string) (string, []walk.Doc, config.Config) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	walked, err := walk.Files(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return dir, walked, cfg
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "한글은 bigram으로 자른다", in: "게이트웨이",
			want: []string{"게이", "이트", "트웨", "웨이"}},
		{name: "1글자 한글은 그 글자 하나다", in: "각", want: []string{"각"}},
		{name: "라틴은 통짜로 유지하고 소문자로 내린다", in: "MinWikilinks",
			want: []string{"minwikilinks"}},
		{name: "언더스코어 조각을 추가로 낸다", in: "min_wikilinks",
			want: []string{"min_wikilinks", "min", "wikilinks"}},
		{name: "하이픈 조각을 추가로 낸다", in: "go-table",
			want: []string{"go-table", "go", "table"}},
		{name: "혼합 문자열은 구간을 나눈다", in: "LLM게이트웨이",
			want: []string{"llm", "게이", "이트", "트웨", "웨이"}},
		{name: "숫자는 라틴 구간에 붙는다", in: "BM25 v2", want: []string{"bm25", "v2"}},
		{name: "문장부호는 토큰이 되지 않는다", in: "search, 이것이다!", want: []string{"search", "이것", "것이", "이다"}},
		{name: "공백은 구분자다", in: "a  b", want: []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tokenize(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// simpleDoc는 최소 프론트매터를 가진 문서을 만든다.
func simpleDoc(stage, body string) string {
	return "---\ntype: concept\nartifact_stage: " + stage + "\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
		"source_channel: manual\nderived_context: []\n---\n\n" + body
}

func TestBuildAndSearch(t *testing.T) {
	t.Run("min_wikilinks를 담은 문서가 1위다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/gate.md":  simpleDoc("context", "# 승급 게이트\n\nmin_wikilinks 값이 2라는 규칙을 설명한다."),
			"context/other.md": simpleDoc("context", "# 다른 주제\n\n최소 링크 수 이야기가 아니다."),
			"context/third.md": simpleDoc("context", "# 세번째\n\nwikilinks 라는 단어만 곁들인 문서다."),
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		res := ix.Search("min_wikilinks", 3)
		if len(res) == 0 || res[0].Slug != "gate" {
			t.Fatalf("min_wikilinks 문서가 1위여야 함: %+v", res)
		}
	})

	t.Run("한국어 질의로 관련 문서가 상위에 온다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/gateway.md":  simpleDoc("context", "# LLM 게이트웨이 조사\n\n게이트웨이가 프롬프트를 중계한다."),
			"context/database.md": simpleDoc("context", "# 데이터베이스\n\n트랜잭션 격리 수준을 정리한다."),
			"context/misc.md":     simpleDoc("context", "# 잡담\n\n점심 메뉴를 기록한다."),
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		res := ix.Search("게이트웨이", 3)
		if len(res) == 0 || res[0].Slug != "gateway" {
			t.Fatalf("게이트웨이 문서가 1위여야 함: %+v", res)
		}
	})

	t.Run("BM25 길이 정규화가 동작한다", func(t *testing.T) {
		// 두 문서가 질의 토큰을 같은 횟수로 담으면 짧은 문서가 위다.
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/short.md": simpleDoc("context", "# 짧은 문서\n\n트랜잭션\n"),
			"context/long.md":  simpleDoc("context", "# 긴 문서\n\n"+repeat("나머지 이야기가 길게 이어진다. ", 200)+"트랜잭션"),
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		res := ix.Search("트랜잭션", 2)
		if len(res) < 2 {
			t.Fatalf("결과가 2건이어야 함: %+v", res)
		}
		if res[0].Slug != "short" {
			t.Fatalf("짧은 문서가 위여야 함: %+v", res)
		}
	})

	t.Run("동점은 슬러그 순으로 고정한다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/b.md": simpleDoc("context", "동일한 내용이다"),
			"context/a.md": simpleDoc("context", "동일한 내용이다"),
			"context/c.md": simpleDoc("context", "동일한 내용이다"),
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		res := ix.Search("내용", 3)
		if len(res) != 3 {
			t.Fatalf("결과가 3건이어야 함: %+v", res)
		}
		if res[0].Slug != "a" || res[1].Slug != "b" || res[2].Slug != "c" {
			t.Fatalf("슬러그 순이어야 함: %+v", res)
		}
	})

	t.Run("topics와 tags를 색인에 넣는다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\ntopics:\n  - kubernetes\n" +
				"---\n\n본문에는 없는 주제다",
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		res := ix.Search("kubernetes", 1)
		if len(res) != 1 || res[0].Slug != "a" {
			t.Fatalf("topics가 색인되어야 함: %+v", res)
		}
	})
}

// TestFieldWeights는 필드 가중치가 순위를 실제로 움직이는지 본다.
// 본문 쪽 문서가 질의어를 더 많이 담고 있어도 제목과 분류 축이 이긴다.
// 가중치를 전부 1.0으로 되돌리면 이 단언이 깨진다(ADR 0061).
func TestFieldWeights(t *testing.T) {
	// 슬러그는 색인 대상이 아니므로 파일명에 질의어를 넣지 않는다.
	// 동점일 때 슬러그 오름차순이 되도록 본문 쪽을 앞 글자로 둔다.
	const filler = "위키 문서를 정리하는 일반적인 설명 문장이다."

	t.Run("제목에 있는 낱말이 본문에만 있는 낱말을 이깁니다", func(t *testing.T) {
		// 본문 쪽 문서가 질의어를 세 번 담는다. 가중치가 전부 1.0이면
		// 그쪽이 이기고, 제목 가중치가 붙어야 뒤집힌다.
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/a-doc.md": simpleDoc("context",
				"# 문서 하나\n\n캐시 무효화 캐시 무효화 캐시 무효화\n"+filler),
			"context/b-doc.md": simpleDoc("context",
				"# 캐시 무효화\n\n문서 하나\n"+filler),
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		got := ix.Search("캐시 무효화", 2)
		if len(got) != 2 {
			t.Fatalf("두 문서가 다 걸려야 함: %+v", got)
		}
		if got[0].Slug != "b-doc" {
			t.Fatalf("제목이 이겨야 함. 1위=%s 점수=%v", got[0].Slug, got)
		}
	})

	t.Run("topics에 있는 낱말이 본문에만 있는 낱말을 이깁니다", func(t *testing.T) {
		withTopics := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\ntopics:\n  - 캐시 무효화\n---\n\n" +
			"# 다른 제목\n\n" + filler + "\n" + filler
		dir, walked, _ := writeWiki(t, map[string]string{
			"context/a-doc.md": simpleDoc("context",
				"# 또 다른 이야기\n\n캐시 무효화를 다룬다.\n"+filler),
			"context/b-doc.md": withTopics,
		})
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		got := ix.Search("캐시 무효화", 2)
		if len(got) != 2 {
			t.Fatalf("두 문서가 다 걸려야 함: %+v", got)
		}
		if got[0].Slug != "b-doc" {
			t.Fatalf("topics가 이겨야 함. 1위=%s 점수=%v", got[0].Slug, got)
		}
	})
}

// repeat은 문자열을 n번 이어 붙인다.
func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestSaveLoad(t *testing.T) {
	files := map[string]string{
		"context/a.md": simpleDoc("context", "# 첫 문서\n\n게이트웨이 이야기"),
		"context/b.md": simpleDoc("context", "# 둘째 문서\n\n검색 대상"),
	}

	t.Run("저장과 로드가 왕복한다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, files)
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		if err := ix.Save(dir); err != nil {
			t.Fatal(err)
		}
		loaded := Load(dir)
		if loaded == nil {
			t.Fatal("색인을 읽지 못했다")
		}
		if !reflect.DeepEqual(ix, loaded) {
			t.Fatalf("왕복 결과가 다름:\n%+v\n%+v", ix, loaded)
		}
		if got := loaded.Search("게이트웨이", 1); len(got) != 1 || got[0].Slug != "a" {
			t.Fatalf("로드한 색인으로 검색이 되어야 함: %+v", got)
		}
	})

	t.Run("두 번 저장하면 바이트까지 같다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, files)
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		if err := ix.Save(dir); err != nil {
			t.Fatal(err)
		}
		first := readIndexFile(t, dir)
		ix2, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		if err := ix2.Save(dir); err != nil {
			t.Fatal(err)
		}
		second := readIndexFile(t, dir)
		if first != second {
			t.Fatalf("두 저장이 다름:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("스키마 버전이 다르면 낡은 것으로 취급한다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, files)
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		ix.SchemaVersion = SchemaVersion + 1
		if err := ix.Save(dir); err != nil {
			t.Fatal(err)
		}
		if Load(dir) != nil {
			t.Fatal("버전이 다른 색인은 nil이어야 함")
		}
	})

	t.Run("깨진 JSON은 낡은 것으로 취급한다", func(t *testing.T) {
		dir, _, _ := writeWiki(t, files)
		p := filepath.Join(dir, IndexDirName)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, IndexFileName), []byte("{깨진"), 0o644); err != nil {
			t.Fatal(err)
		}
		if Load(dir) != nil {
			t.Fatal("깨진 색인은 nil이어야 함")
		}
	})

	t.Run("파일이 바뀌면 낡은 것으로 판정한다", func(t *testing.T) {
		dir, walked, _ := writeWiki(t, files)
		ix, err := Build(dir, walked, DefaultWeights())
		if err != nil {
			t.Fatal(err)
		}
		if !ix.Fresh(walked, dir) {
			t.Fatal("방금 만든 색인은 신선해야 함")
		}
		if err := os.WriteFile(filepath.Join(dir, "context", "a.md"),
			[]byte(simpleDoc("context", "# 바뀐 문서\n\n내용이 달라졌다")), 0o644); err != nil {
			t.Fatal(err)
		}
		// 바뀐 파일을 다시 순회해 판정한다.
		w2, err := walk.Files(dir, mustConfig(t, dir))
		if err != nil {
			t.Fatal(err)
		}
		if ix.Fresh(w2, dir) {
			t.Fatal("파일이 바뀌었으면 낡은 것이어야 함")
		}
	})
}

// mustConfig는 위키 설정을 읽는 테스트 헬퍼다.
func mustConfig(t *testing.T, dir string) config.Config {
	t.Helper()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// readIndexFile는 색인 파일 원문을 읽는다.
func readIndexFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, IndexDirName, IndexFileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestIgnoreFilesExcludedFromIndex는 순회에서 제외된 README가 색인에도
// 없는지를 본다. walk 한 곳에서만 제외하면 색인과 검색이 함께 따른다는
// 증거다(ADR 0036).
func TestIgnoreFilesExcludedFromIndex(t *testing.T) {
	dir, walked, _ := writeWiki(t, map[string]string{
		"context/README.md":  "디렉토리 설명. 게이트웨이라는 단어를 넣는다.",
		"context/gateway.md": simpleDoc("context", "# 게이트웨이 노트\n\n게이트웨이가 프롬프트를 중계한다."),
	})
	for _, wd := range walked {
		if filepath.Base(wd.Rel) == "README.md" {
			t.Fatalf("README가 순회에 남아 있음: %s", wd.Rel)
		}
	}
	ix, err := Build(dir, walked, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	// README에만 있는 단어는 검색되지 않는다.
	if res := ix.Search("설명", 3); len(res) != 0 {
		t.Errorf("제외된 README가 검색에 나옴: %+v", res)
	}
	// 두 문서에 다 있는 단어는 README를 제외한 문서만 낸다.
	res := ix.Search("게이트웨이", 3)
	if len(res) != 1 || res[0].Slug != "gateway" {
		t.Errorf("README를 뺀 문서만 검색되어야 함: %+v", res)
	}
}
