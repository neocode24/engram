package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runRecall은 recall과 reindex 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 부모 커맨드를
// 조립한다. 전역 플래그는 실제 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runRecall(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		raw, err := cmd.Flags().GetString(flagNow)
		if err != nil {
			return err
		}
		parsed, err := parseNow(raw)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), nowKey{}, parsed))
		return nil
	}
	parent.AddCommand(newRecallCmd())
	parent.AddCommand(newReindexCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// recallGatewayDoc는 헤딩 세 개를 가진 검색 대상 문서다.
// 프론트매터가 11줄이므로 본문 첫 줄은 파일 기준 12줄이다.
const recallGatewayDoc = "---\n" +
	"type: concept\nartifact_stage: context\nstatus: promoted\n" +
	"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
	"source_channel: manual\nderived_context: []\n" +
	"---\n\n" +
	"# LLM 게이트웨이 조사\n\n서두입니다.\n\n" +
	"## 판단 근거\n\n게이트웨이가 프롬프트를 중계합니다.\n\n" +
	"### 저장 형식\n\n인덱스는 JSON으로 저장합니다.\n"

// makeRecallWiki는 임시 디렉토리에 헤딩 구조를 갖춘 위키를 만든다.
func makeRecallWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml":        "preset: education\n",
		"context/gateway.md": recallGatewayDoc,
	}
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

func TestRecallCmd(t *testing.T) {
	t.Run("색인이 없으면 reindex 를 안내하고 거절합니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		out, err := runRecall(t, "recall", "--wiki", wiki, "게이트웨이")
		if err == nil || !strings.Contains(out, "engram reindex") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
		// 조회 커맨드는 색인 파일을 만들지 않는다. ADR 0025.
		if _, err := os.Stat(filepath.Join(wiki, ".engram", "index.json")); !os.IsNotExist(err) {
			t.Error("recall 이 색인 파일을 만들었음")
		}
	})

	t.Run("원문 조각을 출처와 구분선과 함께 냅니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		out, err := runRecall(t, "recall", "--wiki", wiki, "게이트웨이")
		if err != nil {
			t.Fatalf("recall 실패: %v\n%s", err, out)
		}
		for _, want := range []string{
			"[[gateway]]", "판단 근거", "게이트웨이가 프롬프트를 중계합니다.",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
		// 조각이 둘이므로 구분선이 하나 있어야 한다.
		if got := strings.Count(out, "\n---\n"); got != 1 {
			t.Errorf("구분선 수 = %d, want 1:\n%s", got, out)
		}
	})

	t.Run("--json 은 헤딩 경로와 파일 기준 줄 범위를 냅니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		out, err := runRecall(t, "recall", "--json", "--wiki", wiki, "저장")
		if err != nil {
			t.Fatalf("recall 실패: %v\n%s", err, out)
		}
		var res recallResponse
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.IndexStatus != indexFresh || len(res.Chunks) != 1 {
			t.Fatalf("응답이 잘못됨: %+v", res)
		}
		c := res.Chunks[0]
		if c.Rank != 1 || c.Slug != "gateway" || c.Heading != "저장 형식" {
			t.Errorf("조각 필드가 잘못됨: %+v", c)
		}
		if strings.Join(c.HeadingPath, "/") != "판단 근거" {
			t.Errorf("헤딩 경로 = %v, want [판단 근거]", c.HeadingPath)
		}
		// 본문 첫 줄이 파일 기준 12줄이고 ### 저장 형식 은 본문 10줄이므로
		// 파일 기준 21-23 줄이다.
		if c.StartLine != 21 || c.EndLine != 23 {
			t.Errorf("줄 범위 = %d-%d, want 21-23", c.StartLine, c.EndLine)
		}
		if !strings.Contains(c.Body, "인덱스는 JSON으로 저장합니다.") {
			t.Errorf("조각 원문이 잘못됨: %q", c.Body)
		}
	})

	t.Run("낡은 색인은 경고하고 진행합니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		// 문서를 하나 더 넣어 색인을 낡게 만든다.
		extra := filepath.Join(wiki, "context", "extra.md")
		if err := os.WriteFile(extra, []byte(recallGatewayDoc), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runRecall(t, "recall", "--json", "--wiki", wiki, "게이트웨이")
		if err != nil {
			t.Fatalf("낡음은 거절이 아닙니다: %v\n%s", err, out)
		}
		var res recallResponse
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.IndexStatus != indexStale {
			t.Errorf("indexStatus = %q, want %q", res.IndexStatus, indexStale)
		}
		if !strings.Contains(out, "경고: 색인이 낡았습니다") {
			t.Errorf("낡음 경고가 없음:\n%s", out)
		}
		// 낡은 색인은 extra 문서를 모른다.
		if strings.Contains(jsonPart, "extra") {
			t.Errorf("낡은 색인 그대로여야 하는데 새 문서가 나옴:\n%s", out)
		}
	})

	t.Run("--limit 로 상한을 정합니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		out, err := runRecall(t, "recall", "--json", "--limit", "1", "--wiki", wiki, "게이트웨이")
		if err != nil {
			t.Fatalf("recall 실패: %v\n%s", err, out)
		}
		var res recallResponse
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if len(res.Chunks) != 1 {
			t.Errorf("조각 수 = %d, want 1: %+v", len(res.Chunks), res.Chunks)
		}
	})

	t.Run("결과가 없으면 토큰화를 보여줍니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		out, err := runRecall(t, "recall", "--wiki", wiki, "없는단어")
		if err != nil {
			t.Fatalf("결과 없음은 종료 코드 0이어야 함: %v\n%s", err, out)
		}
		if !strings.Contains(out, "결과가 없습니다") || !strings.Contains(out, "토큰으로 검색했습니다") {
			t.Errorf("안내가 없음:\n%s", out)
		}
	})

	t.Run("같은 질의에 두 번 실행해 출력이 같습니다", func(t *testing.T) {
		wiki := makeRecallWiki(t)
		if _, err := runRecall(t, "reindex", wiki); err != nil {
			t.Fatal(err)
		}
		first, err1 := runRecall(t, "recall", "--json", "--wiki", wiki, "게이트웨이")
		second, err2 := runRecall(t, "recall", "--json", "--wiki", wiki, "게이트웨이")
		if err1 != nil || err2 != nil {
			t.Fatalf("실행 에러: %v %v", err1, err2)
		}
		if first != second {
			t.Fatalf("두 실행의 출력이 다릅니다:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init 을 안내합니다", func(t *testing.T) {
		out, err := runRecall(t, "recall", "--wiki", t.TempDir(), "질의")
		if err == nil || !strings.Contains(out, "engram init") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
	})
}
