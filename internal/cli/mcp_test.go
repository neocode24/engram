package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/mcpserver"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// forbiddenTools는 MCP 에 노출하면 안 되는 커맨드들이다. 전부 위키의
// 파일을 바꾸거나 승급을 실행한다. promote 가 특히 중요하다. 도구가
// 있으면 스킬 문서의 "실행은 사람에게 맡겨라"가 부탁이 된다(ADR 0043).
var forbiddenTools = []string{
	"promote", "demote", "archive", "new", "source", "mv", "update",
	"migrate", "sync", "init", "eject", "skills", "reindex", "doctor", "version",
}

// runMCP는 mcp 커맨드를 루트 등록 없이 시험한다.
func runMCP(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.AddCommand(newMCPCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeMCPWiki는 조회 도구가 볼 문서를 갖춘 위키를 만들고 색인도 둔다.
func makeMCPWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: education\n",
		"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
		"context/first.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[index]]\"\n  - \"[[second]]\"\nsource_channel: manual\nderived_context: []\n" +
			"created: 2026-01-01\n---\n\n## 첫 문서\n\n위키 검색 시험이다. 게이트 링크를 채운다.\n",
		"context/second.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\ntags: []\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[index]]\"\n  - \"[[first]]\"\nsource_channel: manual\nderived_context: []\n" +
			"created: 2026-01-02\n---\n\n## 둘째 문서\n\n검색 시험을 위한 또 하나의 문서다.\n",
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
	// 색인을 만들어 둔다. recall 과 bridge 는 색인이 필수다.
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	walked, err := walk.Files(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	ix, err := index.Build(root, walked, index.DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(root); err != nil {
		t.Fatal(err)
	}
	return root
}

// connectMCP는 도구를 등록한 서버에 클라이언트로 붙는다.
func connectMCP(t *testing.T, root string) (*mcp.ClientSession, func()) {
	t.Helper()
	s := mcpserver.New("engram", "dev")
	registerMCPTools(s, root)
	ctx := context.Background()
	session, done, err := mcpserver.Connect(ctx, s)
	if err != nil {
		t.Fatalf("서버 접속 실패: %v", err)
	}
	return session, done
}

// callTool은 도구 하나를 실제로 호출한다.
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s 호출 실패: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s 가 도구 에러를 냄: %+v", name, res.Content)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("%s 결과가 구조화 출력이 아님: %+v", name, res.StructuredContent)
	}
	return out
}

// toolNames는 등록된 도구 이름을 정렬해 낸다.
func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("tools/list 실패: %v", err)
	}
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

// inputKeys는 도구 입력 스키마의 최상위 키를 낸다.
func inputKeys(t *testing.T, session *mcp.ClientSession, name string) []string {
	t.Helper()
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("tools/list 실패: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name != name {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		keys := make([]string, 0, len(schema.Properties))
		for k := range schema.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	t.Fatalf("도구가 없음: %s", name)
	return nil
}

func TestMCPCmd(t *testing.T) {
	t.Run("engram.yaml 이 없으면 시작에 실패한다", func(t *testing.T) {
		out, err := runMCP(t, "mcp", "--wiki", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 함")
		}
		if !strings.Contains(out, "init") {
			t.Errorf("init 안내가 없음: %s", out)
		}
	})
}

func TestMCPTools(t *testing.T) {
	root := makeMCPWiki(t)
	session, done := connectMCP(t, root)
	defer done()

	t.Run("도구가 정확히 열이다", func(t *testing.T) {
		want := []string{"backlinks", "bridge", "capture", "digest", "lint",
			"recall", "resurface", "rules", "search", "status"}
		got := toolNames(t, session)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("도구 목록이 다름:\n got %v\nwant %v", got, want)
		}
	})

	t.Run("쓰기 도구는 capture 하나다", func(t *testing.T) {
		names := toolNames(t, session)
		if !containsString(names, "capture") {
			t.Fatal("capture 가 없음")
		}
		// 나머지 아홉은 전부 조회다. 쓰기 가능성은 도구 목록의 이름으로
		// 가린다. 승급 계열 이름이 하나라도 섞이면 아래 금지 테스트가 잡는다.
		if len(names) != 10 {
			t.Fatalf("도구 수 = %d, want 10", len(names))
		}
	})

	t.Run("금지 목록의 이름이 도구에 없다", func(t *testing.T) {
		names := toolNames(t, session)
		for _, forbidden := range forbiddenTools {
			if containsString(names, forbidden) {
				t.Errorf("금지된 도구가 노출됨: %s", forbidden)
			}
		}
	})

	t.Run("모든 도구가 위키 경로를 인자로 받지 않는다", func(t *testing.T) {
		for _, name := range toolNames(t, session) {
			for _, key := range inputKeys(t, session, name) {
				for _, banned := range []string{"wiki", "path", "root", "dir"} {
					if strings.Contains(key, banned) {
						t.Errorf("도구 %s 의 입력에 경로성 키가 있음: %s", name, key)
					}
				}
			}
		}
	})

	t.Run("capture 가 inbox 에 파일을 만든다", func(t *testing.T) {
		out := callTool(t, session, "capture", map[string]any{
			"title": "회의 메모", "slug": "meeting", "body": "회의 중 적은 메모다",
		})
		path, _ := out["path"].(string)
		if path == "" || !strings.Contains(filepath.ToSlash(path), "/inbox/") {
			t.Fatalf("inbox 경로가 아님: %+v", out)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("파일이 없음: %v", err)
		}
	})

	t.Run("capture 는 슬러그로 inbox 밖에 쓰지 못한다", func(t *testing.T) {
		for _, hostile := range []string{"../escape-one", "..\\escape-two", "sub/dir-three", "../../없는디렉토리-네"} {
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name: "capture",
				Arguments: map[string]any{
					"title": "탈출 시험", "slug": hostile, "body": "경계 시험",
				},
			})
			if err != nil {
				t.Fatalf("슬러그 %q 호출 실패: %v", hostile, err)
			}
			if res.IsError {
				// 거절이 최선이다. 실제로 막히는지 본다(ADR 0043).
				continue
			}
			// 에러 없이 통과했다면 경로는 반드시 inbox 안이어야 한다.
			out, ok := res.StructuredContent.(map[string]any)
			if !ok {
				t.Fatalf("슬러그 %q 결과가 구조화 출력이 아님", hostile)
			}
			path, _ := out["path"].(string)
			rel, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatalf("상대 경로 계산 실패: %v", err)
			}
			if strings.HasPrefix(rel, "..") || filepath.ToSlash(filepath.Dir(rel)) != "inbox" {
				t.Fatalf("슬러그 %q 로 inbox 밖에 씀: %s", hostile, path)
			}
		}
		// 어떤 시도로도 위키 루트 밖이나 루트에 파일이 생기지 않았다.
		for _, leaked := range []string{"escape-one", "escape-two", "dir-three"} {
			if _, err := os.Stat(filepath.Join(root, leaked)); err == nil {
				t.Fatalf("슬러그 시도로 위키 루트에 파일이 생김: %s", leaked)
			}
		}
	})

	t.Run("조회 도구를 전부 불러도 위키가 하나도 안 바뀐다", func(t *testing.T) {
		root := makeMCPWiki(t)
		session, done := connectMCP(t, root)
		defer done()
		before := snapshotWiki(t, root)

		callTool(t, session, "search", map[string]any{"query": "검색"})
		callTool(t, session, "recall", map[string]any{"query": "검색"})
		callTool(t, session, "backlinks", map[string]any{"slug": "first"})
		callTool(t, session, "status", map[string]any{})
		callTool(t, session, "lint", map[string]any{})
		callTool(t, session, "rules", map[string]any{})
		callTool(t, session, "resurface", map[string]any{})
		callTool(t, session, "bridge", map[string]any{})
		callTool(t, session, "digest", map[string]any{})

		after := snapshotWiki(t, root)
		if before != after {
			t.Fatalf("조회 도구가 위키를 바꿈:\n%s", diffSnapshot(before, after))
		}
	})

	t.Run("search 결과가 CLI --json 과 같은 구조다", func(t *testing.T) {
		root := makeMCPWiki(t)
		cliOut, err := runRoot(t, "search", "--json", "--wiki", root, "검색")
		if err != nil {
			t.Fatalf("CLI search 실패: %v\n%s", err, cliOut)
		}
		session, done := connectMCP(t, root)
		defer done()
		out := callTool(t, session, "search", map[string]any{"query": "검색"})

		toolJSON, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		var cliAny, toolAny any
		if err := json.Unmarshal([]byte(strings.TrimSpace(cliOut)), &cliAny); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(toolJSON, &toolAny); err != nil {
			t.Fatal(err)
		}
		cliNorm, _ := json.Marshal(cliAny)
		toolNorm, _ := json.Marshal(toolAny)
		if string(cliNorm) != string(toolNorm) {
			t.Fatalf("search 구조가 다름:\n CLI:   %s\n 도구: %s", cliNorm, toolNorm)
		}
	})

	t.Run("lint 결과가 CLI --json 과 같은 구조다", func(t *testing.T) {
		root := makeMCPWiki(t)
		// 위반을 하나 넣어 결과에 내용이 있게 한다.
		writeWikiFileAt(t, root, "inbox/note.md",
			"---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel:\n---\n\n링크 없는 메모\n")
		cliOut, err := runRoot(t, "lint", "--json", root)
		if err != nil {
			t.Fatalf("CLI lint 실패: %v\n%s", err, cliOut)
		}
		session, done := connectMCP(t, root)
		defer done()
		out := callTool(t, session, "lint", map[string]any{})

		toolJSON, _ := json.Marshal(out)
		var cliAny, toolAny any
		if err := json.Unmarshal([]byte(strings.TrimSpace(cliOut)), &cliAny); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(toolJSON, &toolAny); err != nil {
			t.Fatal(err)
		}
		cliNorm, _ := json.Marshal(cliAny)
		toolNorm, _ := json.Marshal(toolAny)
		if string(cliNorm) != string(toolNorm) {
			t.Fatalf("lint 구조가 다름:\n CLI:   %s\n 도구: %s", cliNorm, toolNorm)
		}
	})
}

// snapshotWiki는 위키 전체 파일의 해시를 합친다. .engram 도 포함한다.
// 조회 도구가 상태를 쓰는지 보는 판정의 진실원이다.
func snapshotWiki(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&b, "%s %s\n", filepath.ToSlash(rel), hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatalf("위키 스냅샷 실패: %v", err)
	}
	return b.String()
}

// diffSnapshot은 두 스냅샷의 다른 줄만 낸다.
func diffSnapshot(before, after string) string {
	var out []string
	b := map[string]bool{}
	for _, line := range strings.Split(before, "\n") {
		b[line] = true
	}
	a := map[string]bool{}
	for _, line := range strings.Split(after, "\n") {
		a[line] = true
	}
	for _, line := range strings.Split(before, "\n") {
		if !a[line] {
			out = append(out, "- "+line)
		}
	}
	for _, line := range strings.Split(after, "\n") {
		if !b[line] {
			out = append(out, "+ "+line)
		}
	}
	return strings.Join(out, "\n")
}
