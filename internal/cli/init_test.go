package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// runRoot는 루트 커맨드를 인자와 함께 실행하고 출력을 반환한다.
// 플래그 파싱과 전역 플래그 처리까지 실제 경로를 지난다.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// readWikiFile은 위키 루트의 파일 하나를 읽는다.
func readWikiFile(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("%s 읽기 실패: %v", name, err)
	}
	return string(data)
}

func TestInit(t *testing.T) {
	t.Run("빈 디렉토리에 위키를 만듭니다", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "wiki")
		out, err := runRoot(t, "init", dir)
		if err != nil {
			t.Fatalf("init 실패: %v", err)
		}
		if !strings.Contains(out, "위키를 초기화했습니다") {
			t.Errorf("온보딩 문구가 없음: %q", out)
		}
		for _, name := range []string{"inbox", "sources", "context", "archive"} {
			if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.IsDir() {
				t.Errorf("디렉토리 %s가 없음", name)
			}
		}
		cfg := readWikiFile(t, dir, "engram.yaml")
		if !strings.Contains(cfg, "preset: education") {
			t.Errorf("engram.yaml에 기본 프리셋이 없음:\n%s", cfg)
		}
		gitignore := readWikiFile(t, dir, ".gitignore")
		if !strings.Contains(gitignore, ".engram/") {
			t.Errorf(".gitignore에 .engram/이 없음:\n%s", gitignore)
		}
	})

	t.Run("프리셋별로 index.md 프론트매터 축이 달라집니다", func(t *testing.T) {
		base := t.TempDir()
		for _, preset := range []string{"personal", "education", "team"} {
			dir := filepath.Join(base, preset)
			if _, err := runRoot(t, "init", dir, "--preset", preset); err != nil {
				t.Fatalf("프리셋 %s init 실패: %v", preset, err)
			}
			fm := readWikiFile(t, dir, "index.md")
			if !strings.Contains(fm, "artifact_stage: context") || !strings.Contains(fm, "status: promoted") {
				t.Errorf("프리셋 %s: artifact_stage나 status가 잘못됨:\n%s", preset, fm)
			}
			if !strings.Contains(fm, "type: system") {
				t.Errorf("프리셋 %s: type 축이 없음", preset)
			}
			switch preset {
			case "personal":
				if strings.Contains(fm, "source_channel:") || strings.Contains(fm, "scope:") {
					t.Errorf("personal은 source_channel과 scope가 꺼져 있어야 함:\n%s", fm)
				}
			case "education":
				if !strings.Contains(fm, "source_channel:") {
					t.Errorf("education은 source_channel이 켜져 있어야 함:\n%s", fm)
				}
				if strings.Contains(fm, "scope:") {
					t.Errorf("education은 scope가 꺼져 있어야 함:\n%s", fm)
				}
			case "team":
				if !strings.Contains(fm, "scope: mixed") || !strings.Contains(fm, "sensitivity: internal") {
					t.Errorf("team은 scope와 sensitivity가 켜져 있어야 함:\n%s", fm)
				}
			}
		}
	})

	t.Run("이미 engram.yaml이 있으면 거절합니다", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := runRoot(t, "init", dir); err != nil {
			t.Fatalf("첫 init 실패: %v", err)
		}
		_, err := runRoot(t, "init", dir)
		if err == nil {
			t.Fatal("재실행은 거절되어야 함")
		}
		for _, want := range []string{"이미 engram 위키", "다른 경로"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("거절 메시지에 %q 없음: %v", want, err)
			}
		}
	})

	t.Run("기존 파일을 덮어쓰지 않고 없는 것만 채웁니다", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "inbox"), 0o755); err != nil {
			t.Fatal(err)
		}
		stray := filepath.Join(dir, "stray.md")
		if err := os.WriteFile(stray, []byte("사용자 파일\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		oldIndex := filepath.Join(dir, "index.md")
		if err := os.WriteFile(oldIndex, []byte("사용자 인덱스\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gi := filepath.Join(dir, ".gitignore")
		if err := os.WriteFile(gi, []byte("node_modules\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runRoot(t, "init", dir); err != nil {
			t.Fatalf("기존 파일이 있어도 init은 성공해야 함: %v", err)
		}
		if got := readWikiFile(t, dir, "stray.md"); got != "사용자 파일\n" {
			t.Errorf("무관 파일이 변경됨: %q", got)
		}
		if got := readWikiFile(t, dir, "index.md"); got != "사용자 인덱스\n" {
			t.Errorf("기존 index.md를 덮어썼음: %q", got)
		}
		gitignore := readWikiFile(t, dir, ".gitignore")
		if !strings.Contains(gitignore, "node_modules") || !strings.Contains(gitignore, ".engram/") {
			t.Errorf(".gitignore 병합이 잘못됨:\n%s", gitignore)
		}
	})

	t.Run("고정된 --now로 두 번 실행하면 결과가 바이트까지 같습니다", func(t *testing.T) {
		base := t.TempDir()
		a := filepath.Join(base, "a")
		b := filepath.Join(base, "b")
		for _, dir := range []string{a, b} {
			if _, err := runRoot(t, "init", dir, "--now", "2026-01-01T00:00:00Z"); err != nil {
				t.Fatalf("init 실패: %v", err)
			}
		}
		for _, name := range []string{"index.md", "engram.yaml", ".gitignore"} {
			x, err1 := os.ReadFile(filepath.Join(a, name))
			y, err2 := os.ReadFile(filepath.Join(b, name))
			if err1 != nil || err2 != nil {
				t.Fatalf("%s 읽기 실패: %v, %v", name, err1, err2)
			}
			if !bytes.Equal(x, y) {
				t.Errorf("%s가 바이트 동일하지 않음:\n%s\n---\n%s", name, x, y)
			}
		}
		if fm := readWikiFile(t, a, "index.md"); !strings.Contains(fm, "created: 2026-01-01") {
			t.Errorf("--now가 index.md 날짜에 반영되지 않음:\n%s", fm)
		}
	})

	t.Run("--json은 경로 목록과 프리셋을 냅니다", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "wiki")
		out, err := runRoot(t, "init", "--json", dir, "--preset", "team")
		if err != nil {
			t.Fatalf("init 실패: %v", err)
		}
		var res initResult
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %q", err, out)
		}
		if res.Preset != "team" {
			t.Errorf("프리셋 = %q, want team", res.Preset)
		}
		if want := []string{"inbox", "sources", "context", "archive"}; !reflect.DeepEqual(res.Dirs, want) {
			t.Errorf("dirs = %v, want %v", res.Dirs, want)
		}
		if want := []string{"engram.yaml", "index.md", ".gitignore"}; !reflect.DeepEqual(res.Files, want) {
			t.Errorf("files = %v, want %v", res.Files, want)
		}
	})

	t.Run("잘못된 프리셋은 허용값과 함께 거절합니다", func(t *testing.T) {
		_, err := runRoot(t, "init", t.TempDir(), "--preset", "hobby")
		if err == nil {
			t.Fatal("거절되어야 함")
		}
		if !strings.Contains(err.Error(), "personal, education, team") {
			t.Errorf("거절 메시지에 허용값이 없음: %v", err)
		}
	})
}
