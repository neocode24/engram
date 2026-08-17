package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/index"
	"github.com/neocode24/engram/internal/walk"
	"github.com/spf13/cobra"
)

// runBridge는 bridge 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 가 root.go 에서 하므로 테스트용 부모에
// 전역 플래그만 붙인다. stderr 는 분리해 --json 파싱을 보호한다.
func runBridge(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.AddCommand(newBridgeCmd())
	var out, errOut bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&errOut)
	parent.SetArgs(args)
	err = parent.Execute()
	return out.String(), errOut.String(), err
}

// makeBridgeWiki는 유사한 context 문서 둘과 관련 없는 문서 하나를 갖춘
// 위키를 만들고 색인까지 만든다. 유사한 둘은 링크로 이어지지 않았다.
func makeBridgeWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	fm := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
		"source_channel: manual\nderived_context: []\n---\n\n"
	files := map[string]string{
		"engram.yaml":     "preset: personal\n",
		"context/go.md":   fm + "# Go 언어\n\nGo 는 컴파일 언어입니다. 고루틴으로 병행성을 다룹니다.",
		"context/rust.md": fm + "# Rust 언어\n\nRust 는 컴파일 언어입니다. 소유권으로 메모리를 다룹니다.",
		"context/tea.md":  fm + "# 차\n\n차는 끓여 마십니다. 음료입니다.",
		"inbox/draft.md":  "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n---\n\nGo 언어 컴파일 메모",
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
	indexBridgeWiki(t, root)
	return root
}

// indexBridgeWiki는 위키를 순회해 색인을 만들고 저장한다.
// reindex 커맨드에 의존하지 않아 커맨드 등록 상태와 무관하다.
func indexBridgeWiki(t *testing.T, root string) {
	t.Helper()
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
}

func TestBridgeCmd(t *testing.T) {
	t.Run("유사한 쌍을 유사도 순으로 내고 기각 명령을 함께 냅니다", func(t *testing.T) {
		out, _, err := runBridge(t, "bridge", "--wiki", makeBridgeWiki(t))
		if err != nil {
			t.Fatalf("후보 탐색은 종료 코드 0 이어야 합니다: %v\n%s", err, out)
		}
		if !strings.Contains(out, "go  rust") {
			t.Errorf("유사한 쌍 go-rust 가 나와야 합니다:\n%s", out)
		}
		if !strings.Contains(out, "engram bridge --reject") {
			t.Errorf("기각 명령 안내가 없습니다:\n%s", out)
		}
		if strings.Contains(out, "\x1b[") {
			t.Error("색상 이스케이프 코드가 있습니다")
		}
	})

	t.Run("--json 은 min 과 indexStale 을 함께 냅니다", func(t *testing.T) {
		out, _, err := runBridge(t, "bridge", "--json", "--wiki", makeBridgeWiki(t))
		if err != nil {
			t.Fatalf("에러: %v\n%s", err, out)
		}
		var res bridgeResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Min != 0.30 {
			t.Errorf("min 기본값: %v", res.Min)
		}
		if res.IndexStale {
			t.Error("갓 만든 색인은 낡지 않았습니다")
		}
		if len(res.Pairs) == 0 {
			t.Fatalf("후보가 비어 있습니다: %s", out)
		}
		if res.Pairs[0].A != "go" || res.Pairs[0].B != "rust" {
			t.Errorf("첫 후보는 go-rust 여야 합니다: %+v", res.Pairs[0])
		}
	})

	t.Run("색인이 없으면 진행하지 않고 reindex 를 안내합니다", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "context"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte("preset: personal\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, errOut, err := runBridge(t, "bridge", "--wiki", root)
		if err == nil {
			t.Fatal("색인이 없으면 에러여야 합니다")
		}
		if !strings.Contains(out+errOut, "engram reindex") {
			t.Errorf("안내에 engram reindex 가 없습니다: %s", out)
		}
	})

	t.Run("낡은 색인은 경고를 내고 진행합니다", func(t *testing.T) {
		root := makeBridgeWiki(t)
		// 문서를 하나 더 넣어 색인을 낡게 만든다.
		p := filepath.Join(root, "context", "extra.md")
		if err := os.WriteFile(p, []byte("---\ntype: concept\nartifact_stage: context\n---\n\n추가 문서"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, errOut, err := runBridge(t, "bridge", "--json", "--wiki", root)
		if err != nil {
			t.Fatalf("낡은 색인은 진행해야 합니다: %v\n%s%s", err, out, errOut)
		}
		if !strings.Contains(errOut, "색인이 낡았습니다") {
			t.Errorf("낡음 경고가 없습니다: %s", errOut)
		}
		var res bridgeResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if !res.IndexStale {
			t.Error("indexStale 이 참이어야 합니다")
		}
	})

	t.Run("--reject 는 쌍을 상태 파일에 기록하고 후보에서 뺍니다", func(t *testing.T) {
		root := makeBridgeWiki(t)
		out, _, err := runBridge(t, "bridge", "--wiki", root, "--reject", "go", "rust")
		if err != nil {
			t.Fatalf("기각 실패: %v\n%s", err, out)
		}
		raw, err := os.ReadFile(filepath.Join(root, "engram-state.yaml"))
		if err != nil {
			t.Fatalf("상태 파일이 생겨야 합니다: %v", err)
		}
		if want := "bridge_rejections:\n  - [go, rust]\n"; string(raw) != want {
			t.Errorf("상태 파일:\n got %q\nwant %q", raw, want)
		}
		out, _, err = runBridge(t, "bridge", "--wiki", root)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "go  rust") {
			t.Errorf("기각한 쌍이 다시 나왔습니다:\n%s", out)
		}
	})

	t.Run("없는 슬러그의 기각은 거절합니다", func(t *testing.T) {
		root := makeBridgeWiki(t)
		out, errOut, err := runBridge(t, "bridge", "--wiki", root, "--reject", "go", "없는문서")
		if err == nil {
			t.Fatal("없는 슬러그는 에러여야 합니다")
		}
		if !strings.Contains(out+errOut, "없는문서") {
			t.Errorf("무엇이 없는지 알려야 합니다: %s", out)
		}
		if _, err := os.Stat(filepath.Join(root, "engram-state.yaml")); !os.IsNotExist(err) {
			t.Error("거절했으면 상태 파일을 만들지 않아야 합니다")
		}
	})

	t.Run("--reject 는 슬러그 두 개를 요구합니다", func(t *testing.T) {
		out, errOut, err := runBridge(t, "bridge", "--wiki", makeBridgeWiki(t), "--reject", "a", "b", "c")
		if err == nil {
			t.Fatal("슬러그 세 개는 에러여야 합니다")
		}
		if !strings.Contains(out+errOut, "슬러그 두 개") {
			t.Errorf("안내: %s", out)
		}
	})

	t.Run("--reject 와 --unreject 를 함께 쓰면 거절합니다", func(t *testing.T) {
		out, errOut, err := runBridge(t, "bridge", "--wiki", makeBridgeWiki(t),
			"--reject", "go", "rust", "--unreject", "go", "rust")
		if err == nil {
			t.Fatal("동시 사용은 에러여야 합니다")
		}
		if !strings.Contains(out+errOut, "함께 쓸 수 없습니다") {
			t.Errorf("안내: %s", out)
		}
	})

	t.Run("--unreject 는 기각을 되돌립니다", func(t *testing.T) {
		root := makeBridgeWiki(t)
		if out, _, err := runBridge(t, "bridge", "--wiki", root, "--reject", "go", "rust"); err != nil {
			t.Fatalf("기각 실패: %v\n%s", err, out)
		}
		out, _, err := runBridge(t, "bridge", "--wiki", root, "--unreject", "rust", "go")
		if err != nil {
			t.Fatalf("되돌리기 실패: %v\n%s", err, out)
		}
		raw, err := os.ReadFile(filepath.Join(root, "engram-state.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "[]") {
			t.Errorf("기각 목록이 비어야 합니다: %q", raw)
		}
		out, _, err = runBridge(t, "bridge", "--wiki", root)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "go  rust") {
			t.Errorf("되돌린 쌍이 다시 나와야 합니다:\n%s", out)
		}
	})

	t.Run("위키가 아니면 거절하고 init 을 안내합니다", func(t *testing.T) {
		out, errOut, err := runBridge(t, "bridge", "--wiki", t.TempDir())
		if err == nil {
			t.Fatal("위키가 아니면 에러여야 합니다")
		}
		if !strings.Contains(out+errOut, "engram init") {
			t.Errorf("안내에 engram init 이 없습니다: %s", out)
		}
	})
}
