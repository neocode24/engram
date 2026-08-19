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
		// b 와 c 를 context 단계로 둬야 게이트의 링크 대상이 2개가 된다.
		// inbox 문서는 대상 집계에서 빠진다(ADR 0022).
		"context/b.md": "---\ntype: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n링크 없는 메모\n",
		"context/c.md": "---\ntype: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n어디서도 가리키지 않는 메모\n",
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
	t.Run("텍스트 출력은 파일별 묶음과 요약 줄을 냅니다", func(t *testing.T) {
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

	t.Run("--json은 위반 배열과 요약 카운트를 냅니다", func(t *testing.T) {
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

	t.Run("같은 위키를 두 번 검사하면 --json 출력이 바이트까지 같습니다", func(t *testing.T) {
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

	t.Run("깨끗한 위키는 종료 코드 0입니다", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "context"), 0o755); err != nil {
			t.Fatal(err)
		}
		// 셋이 서로를 가리켜 각자 링크 2개를 갖는다. inbox 문서를
		// 가리키면 게이트가 세지 않는다(ADR 0054).
		ctxDoc := func(related, body string) string {
			return "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs:\n  - sources/원본.md\nderived_from: []\n" +
				"related:\n  - \"[[" + related + "]]\"\nsource_channel: manual\nderived_context: []\n" +
				"---\n\n본문 [[" + body + "]] 링크\n"
		}
		for name, content := range map[string]string{
			"context/a.md":     ctxDoc("peer", "peer2"),
			"context/peer.md":  ctxDoc("a", "peer2"),
			"context/peer2.md": ctxDoc("a", "peer"),
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

	t.Run("마크다운 링크만 쓴 위키도 게이트를 넘습니다", func(t *testing.T) {
		dir := t.TempDir()
		// 셋이 마크다운 링크로만 서로를 가리킨다(ADR 0065). 위키링크는 한 개도 없다.
		mdDoc := func(first, second string) string {
			return "---\n" +
				"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\n---\n\n" +
				"본문 [첫째](" + first + ".md) 와 [둘째](context/" + second + ".md) 링크\n"
		}
		for name, content := range map[string]string{
			"context/a.md":     mdDoc("peer", "peer2"),
			"context/peer.md":  mdDoc("a", "peer2"),
			"context/peer2.md": mdDoc("a", "peer"),
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
			t.Fatalf("마크다운 링크만 있어도 종료 코드 0이어야 함: %v\n%s", err, out)
		}
		for _, unwanted := range []string{"gate.min-wikilinks", "graph.orphan"} {
			if strings.Contains(out, unwanted) {
				t.Errorf("%s 가 나오면 안 됨:\n%s", unwanted, out)
			}
		}
	})
}

func TestLintSecuritySecret(t *testing.T) {
	t.Run("시크릿을 error로 고발합니다", func(t *testing.T) {
		dir := makeWiki(t)
		p := filepath.Join(dir, "context", "열쇠.md")
		body := "---\n" +
			"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n" +
			"---\n\n토큰은 ghp_0123456789abcdefghijklmnopqrstuvwxyzABCD 입니다.\n"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runRoot(t, "lint", "--json", dir)
		if err == nil {
			t.Fatal("error 위반이 있으므로 종료 코드 1이어야 함")
		}
		var res lint.Result
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		var found bool
		for _, v := range res.Violations {
			if v.Rule != "security.secret" {
				continue
			}
			found = true
			if v.Severity != lint.SevError {
				t.Errorf("등급 = %s, 기대 error", v.Severity)
			}
			if v.Path != "context/열쇠.md" || v.Line != 13 {
				t.Errorf("위치 = %s:%d, 기대 context/열쇠.md:13", v.Path, v.Line)
			}
			if strings.Contains(v.Message, "ghp_") {
				t.Errorf("위반 메시지에 값이 실림: %s", v.Message)
			}
		}
		if !found {
			t.Fatalf("security.secret 위반이 없음:\n%s", out)
		}
	})
}

func TestLintIncludeInboxFlag(t *testing.T) {
	// capture 를 거치지 않고 inbox 에 떨어진 파일이다. 기본 범위에서는
	// 잡히지 않고 종료 코드 0 이어야 한다(ADR 0070).
	files := map[string]string{
		"engram.yaml": "preset: personal\n",
		"context/peer.md": "---\n" +
			"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n링크 없는 메모\n",
		"inbox/dropped.md": "프론트매터 없는 메모\n",
	}
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

	t.Run("기본 범위는 inbox 를 건너뛰고 안내 줄을 낸다", func(t *testing.T) {
		out, err := runRoot(t, "lint", dir)
		if err != nil {
			t.Fatalf("기본 범위에서는 종료 코드 0이어야 함: %v\n%s", err, out)
		}
		if !strings.Contains(out, "inbox 문서 1개를 건너뛰었습니다") {
			t.Errorf("건너뛰기 안내가 없음:\n%s", out)
		}
		if strings.Contains(out, "frontmatter.missing") {
			t.Errorf("inbox 위반이 기본 범위에서 나오면 안 됨:\n%s", out)
		}
	})

	t.Run("--include-inbox 는 inbox 도 검사한다", func(t *testing.T) {
		out, err := runRoot(t, "lint", "--include-inbox", dir)
		if err == nil {
			t.Fatal("error 위반이 있으므로 종료 코드 1이어야 함")
		}
		if !strings.Contains(out, "frontmatter.missing") {
			t.Errorf("inbox 위반이 잡히지 않음:\n%s", out)
		}
		if strings.Contains(out, "건너뛰었습니다") {
			t.Errorf("범위를 열었으면 건너뛰기 안내가 없어야 함:\n%s", out)
		}
	})

	t.Run("건너뛴 inbox 가 없으면 안내 줄을 내지 않는다", func(t *testing.T) {
		if err := os.Remove(filepath.Join(dir, "inbox", "dropped.md")); err != nil {
			t.Fatal(err)
		}
		out, err := runRoot(t, "lint", dir)
		if err != nil {
			t.Fatalf("warn 은 종료 코드를 바꾸지 않는다: %v\n%s", err, out)
		}
		if strings.Contains(out, "건너뛰었습니다") {
			t.Errorf("건너뛴 문서가 없으면 안내가 없어야 함:\n%s", out)
		}
	})
}
