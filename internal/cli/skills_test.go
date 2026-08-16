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

// runSkills는 skills 커맨드를 루트 등록 없이 시험한다.
func runSkills(t *testing.T, args ...string) (string, error) {
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
	parent.AddCommand(newSkillsCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// skillsPath는 스킬 루트 안에 심길 문서 경로를 낸다.
func skillsPath(root string) string {
	return filepath.Join(root, "engram", "SKILL.md")
}

func TestSkillsCmd(t *testing.T) {
	t.Run("사용법을 낸다", func(t *testing.T) {
		out, err := runSkills(t, "skills")
		// 하위 커맨드 없이 치면 cobra 가 사용법을 인쇄한다. rules 와 같다.
		if err != nil {
			t.Fatalf("사용법 인쇄는 에러가 아니어야 함: %v", err)
		}
		if !strings.Contains(out, "install") {
			t.Errorf("사용법에 install 이 없음: %s", out)
		}
	})

	t.Run("--dir 로 지정한 곳에 심는다", func(t *testing.T) {
		root := t.TempDir()
		out, err := runSkills(t, "skills", "install", "--dir", root)
		if err != nil {
			t.Fatalf("install 실패: %v\n%s", err, out)
		}
		raw, err := os.ReadFile(skillsPath(root))
		if err != nil {
			t.Fatalf("심긴 문서가 없음: %v", err)
		}
		if !strings.Contains(string(raw), "engram은 LLM을 부르지 않는다") {
			t.Errorf("심긴 문서에 경계 문구가 없음")
		}
		if !strings.Contains(out, "다시 시작") {
			t.Errorf("재시작 안내가 없음: %s", out)
		}
	})

	t.Run("충돌하면 실패하고 아무것도 쓰지 않는다", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "engram"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(skillsPath(root), []byte("사용자가 고친 스킬\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runSkills(t, "skills", "install", "--dir", root)
		if err == nil {
			t.Fatal("충돌이 있으면 에러여야 함")
		}
		if !strings.Contains(out, "--force") {
			t.Errorf("--force 안내가 없음: %s", out)
		}
		raw, _ := os.ReadFile(skillsPath(root))
		if string(raw) != "사용자가 고친 스킬\n" {
			t.Errorf("기존 파일이 덮어 써짐: %q", raw)
		}
	})

	t.Run("--force 는 덮고 덮은 목록을 낸다", func(t *testing.T) {
		root := t.TempDir()
		if _, err := runSkills(t, "skills", "install", "--dir", root); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(skillsPath(root), []byte("사용자가 고친 스킬\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runSkills(t, "skills", "install", "--dir", root, "--force")
		if err != nil {
			t.Fatalf("--force 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "덮어 쓴 파일") {
			t.Errorf("덮은 목록이 없음: %s", out)
		}
		raw, _ := os.ReadFile(skillsPath(root))
		if strings.Contains(string(raw), "사용자가 고친 스킬") {
			t.Error("--force 가 기존 파일을 안 덮었음")
		}
	})

	t.Run("--dry-run 은 쓰지 않는다", func(t *testing.T) {
		root := t.TempDir()
		out, err := runSkills(t, "skills", "install", "--dir", root, "--dry-run")
		if err != nil {
			t.Fatalf("dry-run 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "dry-run") {
			t.Errorf("dry-run 표시가 없음: %s", out)
		}
		if _, err := os.Stat(skillsPath(root)); !os.IsNotExist(err) {
			t.Error("dry-run 이 파일을 썼음")
		}
	})

	t.Run("감지 대상이 없고 --dir 도 없으면 실패한다", func(t *testing.T) {
		// 홈 디렉토리를 비어 있는 임시 디렉토리로 돌린다. 테스트가
		// 실제 홈을 건드리지 않게 t.Setenv 로만 바꾼다.
		t.Setenv("HOME", t.TempDir())
		out, err := runSkills(t, "skills", "install")
		if err == nil {
			t.Fatal("대상이 없으면 에러여야 함")
		}
		if !strings.Contains(out, "--dir") {
			t.Errorf("--dir 안내가 없음: %s", out)
		}
	})

	t.Run("--dir 이 없는 디렉토리면 실패한다", func(t *testing.T) {
		root := t.TempDir()
		out, err := runSkills(t, "skills", "install", "--dir", filepath.Join(root, "없음"))
		if err == nil {
			t.Fatal("없는 디렉토리는 에러여야 함")
		}
		if !strings.Contains(out, "디렉토리") {
			t.Errorf("안내가 없음: %s", out)
		}
	})

	t.Run("두 번 돌리면 같은 내용이 심긴다", func(t *testing.T) {
		first := t.TempDir()
		second := t.TempDir()
		if _, err := runSkills(t, "skills", "install", "--dir", first); err != nil {
			t.Fatal(err)
		}
		if _, err := runSkills(t, "skills", "install", "--dir", second); err != nil {
			t.Fatal(err)
		}
		a, err := os.ReadFile(skillsPath(first))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(skillsPath(second))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Error("두 설치의 내용이 다름")
		}
	})

	t.Run("--json 은 결과 구조체를 낸다", func(t *testing.T) {
		root := t.TempDir()
		out, err := runSkills(t, "skills", "install", "--json", "--dir", root)
		if err != nil {
			t.Fatalf("install 실패: %v\n%s", err, out)
		}
		var res skillsOutcome
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n%s", err, out)
		}
		if res.DryRun || len(res.Files) != 1 || len(res.Written) != 1 {
			t.Errorf("결과가 틀립니다: %+v", res)
		}
	})

	t.Run("위키 밖에서 실행해도 동작한다", func(t *testing.T) {
		// 이 테스트의 작업 디렉토리에는 engram.yaml 이 없다. 위키가
		// 아니라 에이전트를 다루는 커맨드이므로 그대로 동작해야 한다.
		if _, err := os.Stat("engram.yaml"); err == nil {
			t.Skip("작업 디렉토리에 engram.yaml 이 있어 이 하위 테스트의 전제가 성립하지 않음")
		}
		root := t.TempDir()
		if out, err := runSkills(t, "skills", "install", "--dir", root); err != nil {
			t.Fatalf("위키 밖에서 실패: %v\n%s", err, out)
		}
	})
}
