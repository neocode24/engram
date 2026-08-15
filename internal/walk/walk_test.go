package walk

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/neocode24/engram/internal/config"
)

// writeWiki는 임시 디렉토리에 위키 파일들을 만든다.
func writeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFiles(t *testing.T) {
	doc := "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n---\n\n메모\n"

	t.Run("page_dirs 와 root_files 를 경로순으로 반환한다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":  "preset: education\n",
			"inbox/b.md":   doc,
			"inbox/a.md":   doc,
			"context/c.md": doc,
			"index.md":     doc,
			"루트아님/d.md":    doc,
		})
		docs, err := Files(root, mustConfig(t, root))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"context/c.md", "inbox/a.md", "inbox/b.md", "index.md"}
		got := make([]string, 0, len(docs))
		for _, d := range docs {
			got = append(got, d.Rel)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("순회 결과 = %v, want %v", got, want)
		}
		for _, d := range docs {
			if d.Root != (d.Rel == "index.md") {
				t.Errorf("색인 여부가 틀리다: %s", d.Rel)
			}
			if d.Err != nil || !d.Parsed.HasFrontmatter {
				t.Errorf("정상 문서가 파싱되지 않았다: %s: %v", d.Rel, d.Err)
			}
		}
	})

	t.Run("숨김 디렉토리와 .engram 은 제외한다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":        "preset: education\n",
			"inbox/a.md":         doc,
			"inbox/.hidden/b.md": doc,
			"inbox/.engram/c.md": doc,
		})
		docs, err := Files(root, mustConfig(t, root))
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 || docs[0].Rel != "inbox/a.md" {
			t.Errorf("숨김 디렉토리가 걸러지지 않았다: %+v", docs)
		}
	})

	t.Run("닫는 구분자가 없는 문서는 ErrUnclosed 를 싣는다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml": "preset: education\n",
			"inbox/a.md":  "---\ntype: inbox-note\n",
		})
		docs, err := Files(root, mustConfig(t, root))
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 || !errors.Is(docs[0].Err, ErrUnclosed) {
			t.Errorf("ErrUnclosed 여야 함: %+v", docs)
		}
	})

	t.Run("CRLF 와 BOM 을 정규화해 파싱한다", func(t *testing.T) {
		crlf := "\xEF\xBB\xBF---\r\ntype: inbox-note\r\nartifact_stage: inbox\r\nstatus: inbox\r\nindexable: false\r\n---\r\n\r\n본문\r\n"
		root := writeWiki(t, map[string]string{
			"engram.yaml": "preset: education\n",
			"inbox/a.md":  crlf,
		})
		docs, err := Files(root, mustConfig(t, root))
		if err != nil {
			t.Fatal(err)
		}
		if docs[0].Err != nil || !docs[0].Parsed.HasFrontmatter {
			t.Errorf("BOM 과 CRLF 문서가 파싱되지 않았다: %v", docs[0].Err)
		}
		if docs[0].Parsed.BodyLine != 7 {
			t.Errorf("줄 번호가 틀어짐: %d, want 7", docs[0].Parsed.BodyLine)
		}
	})
}

// mustConfig는 테스트 위키의 설정을 읽는다.
func mustConfig(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
