package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runSearchRoot는 search와 reindex를 등록한 루트 커맨드를 실행한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 루트를 조립한다.
func runSearchRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	root.AddCommand(newSearchCmd())
	root.AddCommand(newReindexCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// searchWiki는 임시 디렉토리에 검색 대상 위키를 만들고 그 경로를 반환한다.
func searchWiki(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wiki")
	if _, err := runSearchRoot(t, "init", dir); err != nil {
		t.Fatalf("init 실패: %v", err)
	}
	doc := "---\n" +
		"type: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
		"source_channel: manual\nderived_context: []\n" +
		"---\n\n# LLM 게이트웨이 조사\n\n게이트웨이가 프롬프트를 중계합니다. min_wikilinks 규칙도 다룹니다."
	p := filepath.Join(dir, "context", "gateway.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReindexCmd(t *testing.T) {
	t.Run("색인 파일을 만들고 두 번 돌리면 바이트까지 같습니다", func(t *testing.T) {
		dir := searchWiki(t)
		out, err := runSearchRoot(t, "reindex", dir)
		if err != nil {
			t.Fatalf("reindex 실패: %v\n%s", err, out)
		}
		path := filepath.Join(dir, ".engram", "index.json")
		first := readWikiFile(t, filepath.Dir(path), filepath.Base(path))
		for _, want := range []string{"문서 2", "토큰"} {
			if !strings.Contains(out, want) {
				t.Errorf("요약에 %q 없음:\n%s", want, out)
			}
		}
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		second := readWikiFile(t, filepath.Dir(path), filepath.Base(path))
		if first != second {
			t.Fatalf("두 번의 색인이 다름:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("--json은 문서 수와 토큰 수와 크기를 냅니다", func(t *testing.T) {
		dir := searchWiki(t)
		out, err := runSearchRoot(t, "reindex", "--json", dir)
		if err != nil {
			t.Fatalf("reindex 실패: %v\n%s", err, out)
		}
		var res reindexResult
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Docs != 2 || res.Tokens == 0 || res.IndexBytes == 0 {
			t.Errorf("요약이 잘못됨: %+v", res)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init을 안내합니다", func(t *testing.T) {
		_, err := runSearchRoot(t, "reindex", t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "engram init") {
			t.Fatalf("거절되어야 함: %v", err)
		}
	})
}

func TestSearchCmd(t *testing.T) {
	t.Run("신선한 색인으로 조용히 검색합니다", func(t *testing.T) {
		dir := searchWiki(t)
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		out, err := runSearchRoot(t, "search", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("search 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "gateway") {
			t.Errorf("결과에 gateway가 없음:\n%s", out)
		}
		for _, noise := range []string{"경고", "안내"} {
			if strings.Contains(out, noise) {
				t.Errorf("신선한 색인에는 %q이 없어야 함:\n%s", noise, out)
			}
		}
	})

	t.Run("낡은 색인은 경고하고 낡은 결과를 그대로 씁니다", func(t *testing.T) {
		dir := searchWiki(t)
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		// 문서를 하나 더 넣어 색인을 낡게 만든다.
		extra := filepath.Join(dir, "context", "extra.md")
		if err := os.WriteFile(extra, []byte(
			"---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"+
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n"+
				"source_channel: manual\nderived_context: []\n---\n\n# 새 문서\n\n게이트웨이 이야기를 덧붙입니다"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runSearchRoot(t, "search", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("경고는 거절이 아닙니다: %v", err)
		}
		if !strings.Contains(out, "경고: 색인이 낡았습니다") || !strings.Contains(out, "engram reindex") {
			t.Errorf("낡음 경고가 없음:\n%s", out)
		}
		// 낡은 색인은 extra 문서를 모른다. 재색인 없이 낡은 결과를 낸다.
		if strings.Contains(out, "extra") {
			t.Errorf("낡은 색인 그대로여야 하는데 새 문서가 나옴:\n%s", out)
		}
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		out, err = runSearchRoot(t, "search", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "extra") {
			t.Errorf("재색인 후에는 새 문서가 나와야 함:\n%s", out)
		}
	})

	t.Run("색인이 없으면 즉석 색인으로 결과를 내고 파일을 만들지 않습니다", func(t *testing.T) {
		dir := searchWiki(t)
		out, err := runSearchRoot(t, "search", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("search 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "gateway") {
			t.Errorf("즉석 색인 결과가 나와야 함:\n%s", out)
		}
		if !strings.Contains(out, "안내: 색인이 없어") {
			t.Errorf("즉석 색인 안내가 없음:\n%s", out)
		}
		// 조회가 색인 파일을 쓰면 안 된다. ADR 0025.
		if _, err := os.Stat(filepath.Join(dir, ".engram", "index.json")); !os.IsNotExist(err) {
			t.Fatal("조회 커맨드가 색인 파일을 만들었음")
		}
	})

	t.Run("--limit로 상한을 정합니다", func(t *testing.T) {
		dir := searchWiki(t)
		// 문서를 여러 개 만들어 질의에 여러 건이 걸리게 한다.
		for _, name := range []string{"b", "a", "c"} {
			p := filepath.Join(dir, "context", name+".md")
			if err := os.WriteFile(p, []byte(
				"---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"+
					"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n"+
					"source_channel: manual\nderived_context: []\n---\n\n"+name+"\n\n게이트웨이 얘기"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		out, err := runSearchRoot(t, "search", "--wiki", dir, "--limit", "2", "게이트웨이")
		if err != nil {
			t.Fatalf("search 실패: %v\n%s", err, out)
		}
		if got := strings.Count(out, "\n"); got != 2 {
			t.Errorf("결과가 2건이어야 함. 줄 수 %d:\n%s", got, out)
		}
	})

	t.Run("결과가 없으면 토큰화를 보여줍니다", func(t *testing.T) {
		dir := searchWiki(t)
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		out, err := runSearchRoot(t, "search", "--wiki", dir, "없는단어")
		if err != nil {
			t.Fatalf("결과 없음은 종료 코드 0이어야 함: %v", err)
		}
		for _, want := range []string{"결과가 없습니다", "토큰으로 검색했습니다", "없는", "는단", "는단어"} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("--json은 순위와 점수와 인덱스 상태를 냅니다", func(t *testing.T) {
		dir := searchWiki(t)
		if _, err := runSearchRoot(t, "reindex", dir); err != nil {
			t.Fatal(err)
		}
		out, err := runSearchRoot(t, "search", "--json", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("search 실패: %v\n%s", err, out)
		}
		var res searchResponse
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.IndexStatus != "fresh" || len(res.Results) == 0 {
			t.Fatalf("JSON 내용이 잘못됨: %+v", res)
		}
		first := res.Results[0]
		if first.Rank != 1 || first.Slug == "" || first.Score <= 0 || first.Path == "" {
			t.Errorf("결과 필드가 잘못됨: %+v", first)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init을 안내합니다", func(t *testing.T) {
		_, err := runSearchRoot(t, "search", "--wiki", t.TempDir(), "질의")
		if err == nil || !strings.Contains(err.Error(), "engram init") {
			t.Fatalf("거절되어야 함: %v", err)
		}
	})
}
