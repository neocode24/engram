package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/lint"
)

// makeWiki는 임시 디렉토리에 검사 대상 위키를 만들고 그 루트를 반환한다.
func makeWiki(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"engram.yaml": "topics: [go]\nforms: [note]\n",
		"context/a.md": "---\n" +
			"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[b]]\"\nsource_channel: manual\nderived_context: []\n" +
			"form: memo\ntopics:\n  - kubernetes\n" +
			"---\n\n본문 [[b]] 링크\n",
		"inbox/b.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\n---\n\n링크 없는 메모\n",
		"inbox/c.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\n---\n\n어디서도 가리키지 않는 메모\n",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLintCmd(t *testing.T) {
	t.Run("텍스트 출력은 파일별 묶음과 요약 줄을 낸다", func(t *testing.T) {
		dir := makeWiki(t)
		out, err := runRoot(t, "lint", dir)
		if err == nil {
			t.Fatal("error 위반이 있으므로 종료 코드 1이어야 함")
		}
		for _, want := range []string{
			"context/a.md", "taxonomy.forms", "taxonomy.topics", "gate.min-wikilinks",
			"graph.orphan", "위키 진단", "wiki.broad-topic", "검사한 파일", "고치는 법",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("--json은 위반 배열과 요약 카운트를 낸다", func(t *testing.T) {
		dir := makeWiki(t)
		out, err := runRoot(t, "lint", "--json", dir)
		if err == nil {
			t.Fatal("error 위반이 있으므로 종료 코드 1이어야 함")
		}
		var res lint.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if len(res.Violations) == 0 {
			t.Fatal("위반이 파싱되지 않음")
		}
		if res.Summary.Files != 3 || res.Summary.Error == 0 || res.Summary.Reject == 0 || res.Summary.Warn == 0 {
			t.Errorf("요약 카운트가 잘못됨: %+v", res.Summary)
		}
		// 위키 단위 진단은 파일 위반 배열이 아니라 별도 배열로 나온다.
		if len(res.WikiFindings) != 1 || res.WikiFindings[0].Topic != "kubernetes" {
			t.Fatalf("wikiFindings가 별도 배열로 나와야 함: %+v", res.WikiFindings)
		}
		for _, v := range res.Violations {
			if v.Rule == "wiki.broad-topic" {
				t.Fatalf("위키 단위 진단이 violations에 섞였음: %+v", v)
			}
		}
		for _, v := range res.Violations {
			if v.Rule == "" || v.Severity == "" || v.Path == "" || v.Message == "" || v.Fix == "" {
				t.Errorf("위반 필드가 비어 있음: %+v", v)
			}
		}
	})

	t.Run("같은 위키를 두 번 검사하면 --json 출력이 바이트까지 같다", func(t *testing.T) {
		dir := makeWiki(t)
		first, err1 := runRoot(t, "lint", "--json", dir)
		second, err2 := runRoot(t, "lint", "--json", dir)
		if err1 == nil || err2 == nil {
			t.Fatal("error 위반이 있으므로 종료 코드 1이어야 함")
		}
		if first != second {
			t.Fatalf("두 실행의 --json 출력이 다름:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("깨끗한 위키는 종료 코드 0이다", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "context"), 0o755); err != nil {
			t.Fatal(err)
		}
		doc := "---\n" +
			"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[peer]]\"\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n본문 [[peer2]] 링크\n"
		inboxDoc := "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\nsource_channel: manual\n---\n\n메모\n"
		for name, content := range map[string]string{
			"context/a.md":   doc,
			"inbox/peer.md":  inboxDoc,
			"inbox/peer2.md": inboxDoc,
		} {
			p := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		out, err := runRoot(t, "lint", dir)
		if err != nil {
			t.Fatalf("깨끗한 위키는 종료 코드 0이어야 함: %v\n%s", err, out)
		}
		if !strings.Contains(out, "위반 없음") {
			t.Errorf("위반 없음 안내가 없음:\n%s", out)
		}
	})
}
