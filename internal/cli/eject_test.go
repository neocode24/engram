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

// runEject는 eject 커맨드를 루트 등록 없이 시험한다. 전역 플래그는 실제
// 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runEject(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newEjectCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeEjectWiki는 eject 대상이 되는 임시 위키를 만든다.
func makeEjectWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "engram.yaml")
	if err := os.WriteFile(p, []byte("preset: education\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestEjectCmd(t *testing.T) {
	t.Run("산출물을 전부 만들고 안내를 낸다", func(t *testing.T) {
		root := makeEjectWiki(t)
		out, err := runEject(t, "eject", "--wiki", root)
		if err != nil {
			t.Fatalf("eject 실패: %v\n%s", err, out)
		}
		for _, name := range []string{
			"meta/frontmatter-schema.md", "scripts/lint-frontmatter.py",
			".githooks/pre-commit", "AGENTS.md", ".gitattributes",
		} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
				t.Errorf("산출물이 없음: %s (%v)", name, err)
			}
		}
		for _, want := range []string{
			"git config core.hooksPath .githooks",
			"search, recall, resurface, bridge, digest, backlinks",
			"python3",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("안내에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("충돌하면 실패하고 아무것도 새로 쓰지 않는다", func(t *testing.T) {
		root := makeEjectWiki(t)
		// 산출물 하나만 미리 둔다. 충돌 목록에 걸려야 전체가 멈춘다.
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("사용자가 고친 계약\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runEject(t, "eject", "--wiki", root)
		if err == nil {
			t.Fatal("충돌이 있으면 에러여야 함")
		}
		if !strings.Contains(out, "AGENTS.md") {
			t.Errorf("충돌 목록에 대상이 없음:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(root, "scripts", "lint-frontmatter.py")); !os.IsNotExist(err) {
			t.Errorf("충돌 시 새 산출물을 쓰면 안 됨 (scripts/lint-frontmatter.py 존재)")
		}
		if got := readWiki(t, root, "AGENTS.md"); got != "사용자가 고친 계약\n" {
			t.Errorf("기존 파일이 덮어 써짐: %q", got)
		}
	})

	t.Run("--force 는 덮고 덮은 목록을 낸다", func(t *testing.T) {
		root := makeEjectWiki(t)
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("사용자가 고친 계약\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runEject(t, "eject", "--wiki", root, "--force")
		if err != nil {
			t.Fatalf("--force 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "덮어 쓴 파일") || !strings.Contains(out, "AGENTS.md") {
			t.Errorf("덮은 목록이 없음:\n%s", out)
		}
		if got := readWiki(t, root, "AGENTS.md"); strings.Contains(got, "사용자가 고친 계약") {
			t.Error("--force 가 기존 파일을 안 덮었음")
		}
	})

	t.Run("--dry-run 은 파일을 쓰지 않는다", func(t *testing.T) {
		root := makeEjectWiki(t)
		out, err := runEject(t, "eject", "--wiki", root, "--dry-run")
		if err != nil {
			t.Fatalf("dry-run 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "dry-run") {
			t.Errorf("dry-run 표시가 없음:\n%s", out)
		}
		for _, name := range []string{"AGENTS.md", ".gitattributes", "meta/frontmatter-schema.md"} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !os.IsNotExist(err) {
				t.Errorf("dry-run 이 %s 를 썼음", name)
			}
		}
	})

	t.Run("두 번 돌리면 충돌로 실패한다", func(t *testing.T) {
		root := makeEjectWiki(t)
		if _, err := runEject(t, "eject", "--wiki", root); err != nil {
			t.Fatal(err)
		}
		out, err := runEject(t, "eject", "--wiki", root)
		if err == nil {
			t.Fatal("두 번째는 충돌이어야 함")
		}
		if !strings.Contains(out, "--force") {
			t.Errorf("안내에 --force 가 없음:\n%s", out)
		}
	})

	t.Run("--json 은 결과 구조체를 낸다", func(t *testing.T) {
		root := makeEjectWiki(t)
		out, err := runEject(t, "eject", "--json", "--wiki", root)
		if err != nil {
			t.Fatalf("eject 실패: %v\n%s", err, out)
		}
		var res ejectOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if res.DryRun || len(res.Files) == 0 || len(res.Written) != len(res.Files) {
			t.Errorf("결과가 틀립니다: %+v", res)
		}
	})

	t.Run("훅과 린터에 실행 권한이 있다", func(t *testing.T) {
		root := makeEjectWiki(t)
		if _, err := runEject(t, "eject", "--wiki", root); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"scripts/lint-frontmatter.py", ".githooks/pre-commit"} {
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o755 {
				t.Errorf("%s 권한 = %o, want 755", name, info.Mode().Perm())
			}
		}
	})
}
