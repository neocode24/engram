package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/doc"
)

func TestUpdateCmd(t *testing.T) {
	t.Run("--set 과 --unset 을 적용한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--wiki", root,
			"--set", "status=archived", "--unset", "topics", "context/note.md")
		if err != nil {
			t.Fatalf("update 실패: %v\n%s", err, out)
		}
		content := readWiki(t, root, "context/note.md")
		if !strings.Contains(content, "status: archived") {
			t.Errorf("status 갱신이 없음:\n%s", content)
		}
		if strings.Contains(content, "topics:") {
			t.Errorf("topics 제거가 안 됨:\n%s", content)
		}
		if !strings.Contains(content, "related:") {
			t.Errorf("같은 줄의 다른 키가 사라졌다:\n%s", content)
		}
	})

	t.Run("--body-from 파일로 본문을 교체한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		bodyFile := filepath.Join(t.TempDir(), "body.md")
		if err := os.WriteFile(bodyFile, []byte("새 본문이다\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runDemoteUpdate(t, "update", "--wiki", root, "--body-from", bodyFile, "context/note.md")
		if err != nil {
			t.Fatalf("update 실패: %v\n%s", err, out)
		}
		content := readWiki(t, root, "context/note.md")
		if !strings.Contains(content, "새 본문이다") || strings.Contains(content, "본문 [[hub]]") {
			t.Errorf("본문 교체가 안 됨:\n%s", content)
		}
	})

	t.Run("배열 값은 쉼표로 여러 값을 받는다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--wiki", root,
			"--set", "topics=go,cli", "--set", "related=hub,index", "context/note.md")
		if err != nil {
			t.Fatalf("update 실패: %v\n%s", err, out)
		}
		d, err := doc.Parse("note.md", []byte(readWiki(t, root, "context/note.md")))
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string][]string{
			"topics":  {"go", "cli"},
			"related": {"[[hub]]", "[[index]]"},
		} {
			got := ""
			for _, f := range d.Fields {
				if f.Key == key && f.Kind == doc.KindStringList {
					got = strings.Join(f.List, ",")
				}
			}
			if got != strings.Join(want, ",") {
				t.Errorf("%s = %q, want %q", key, got, strings.Join(want, ","))
			}
		}
	})

	t.Run("꺼진 축은 거절한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		// education 프리셋은 scope 축이 꺼져 있다.
		out, err := runDemoteUpdate(t, "update", "--wiki", root, "--set", "scope=work", "context/note.md")
		if err == nil {
			t.Fatal("꺼진 축은 거절이어야 한다")
		}
		if !strings.Contains(out, "꺼진 축") {
			t.Errorf("거절 안내가 없음: %s", out)
		}
	})

	t.Run("허용값 밖은 목록과 함께 거절한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--wiki", root, "--set", "status=weird", "context/note.md")
		if err == nil {
			t.Fatal("허용값 밖은 거절이어야 한다")
		}
		if !strings.Contains(out, "허용값") || !strings.Contains(out, "promoted") {
			t.Errorf("허용값 목록이 없음: %s", out)
		}
	})

	t.Run("artifact_stage 변경은 거절한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--wiki", root, "--set", "artifact_stage=inbox", "context/note.md")
		if err == nil {
			t.Fatal("거절이어야 한다")
		}
		if !strings.Contains(out, "engram promote") || !strings.Contains(out, "engram demote") {
			t.Errorf("커맨드 안내가 없음: %s", out)
		}
	})

	t.Run("키 순서를 보존한다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		if _, err := runDemoteUpdate(t, "update", "--wiki", root, "--set", "status=archived", "context/note.md"); err != nil {
			t.Fatal(err)
		}
		d, err := doc.Parse("note.md", []byte(readWiki(t, root, "context/note.md")))
		if err != nil {
			t.Fatal(err)
		}
		var keys []string
		for _, f := range d.Fields {
			keys = append(keys, f.Key)
		}
		want := []string{"type", "artifact_stage", "status", "indexable", "tags", "source_refs",
			"derived_from", "related", "source_channel", "derived_context", "topics", "created", "updated"}
		if strings.Join(keys, ",") != strings.Join(want, ",") {
			t.Errorf("키 순서가 바뀌었다:\n got %v\nwant %v", keys, want)
		}
	})

	t.Run("플래그가 없으면 사용법을 낸다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--wiki", root, "context/note.md")
		if err == nil {
			t.Fatal("에러여야 한다")
		}
		if !strings.Contains(out, "--set") || !strings.Contains(out, "--body-from") {
			t.Errorf("사용법이 없음: %s", out)
		}
	})

	t.Run("--json 은 적용 내용을 낸다", func(t *testing.T) {
		root := makeDemoteWiki(t)
		out, err := runDemoteUpdate(t, "update", "--json", "--wiki", root, "--set", "status=archived", "context/note.md")
		if err != nil {
			t.Fatalf("update 실패: %v\n%s", err, out)
		}
		var res updateOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if len(res.Set) != 1 || res.Set[0] != "status=archived" {
			t.Errorf("결과가 틀리다: %+v", res)
		}
	})
}
