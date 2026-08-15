package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// wantIDs는 항목 순서 계약이다. 순서가 바뀌면 출력이 바뀌므로 테스트가 잡는다.
var wantIDs = []string{
	"env.git",
	"env.git-autocrlf",
	"env.fs-case",
	"env.console-encoding",
	"env.write-perm",
	"wiki.config",
	"wiki.config-unknown-keys",
	"wiki.min-wikilinks",
	"wiki.page-dirs",
	"wiki.root-files",
	"wiki.engram-gitignore",
}

func ids(res Result) []string {
	out := make([]string, 0, len(res.Findings))
	for _, f := range res.Findings {
		out = append(out, f.ID)
	}
	return out
}

func finding(res Result, id string) Finding {
	for _, f := range res.Findings {
		if f.ID == id {
			return f
		}
	}
	return Finding{}
}

// makeWiki는 정상 위키를 임시 디렉토리에 만든다. git 저장소가 아니므로
// autocrlf 와 gitignore 항목은 skip 이 된다.
func makeWiki(t *testing.T, engramYAML string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"inbox", "sources", "context", "archive"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"engram.yaml": engramYAML,
		"index.md":    "# 색인\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const healthyYAML = "preset: education\ntopics: [go]\nforms: [note]\n"

func TestRun(t *testing.T) {
	t.Run("위키가 아닌 디렉토리에서는 위키 항목이 skip 이다", func(t *testing.T) {
		res := Run(t.TempDir())
		for _, id := range wantIDs {
			f := finding(res, id)
			if !strings.HasPrefix(id, "wiki.") {
				continue
			}
			if f.Status != StatusSkip {
				t.Errorf("%s 상태가 %s 다, skip 이어야 한다", id, f.Status)
			}
			if f.Detail == "" {
				t.Errorf("%s skip 사유가 비어 있다", id)
			}
		}
		// env.git-autocrlf 도 git 저장소가 아니면 skip 이므로 위키 항목만 센다.
		wikiSkips := 0
		for _, f := range res.Findings {
			if strings.HasPrefix(f.ID, "wiki.") && f.Status == StatusSkip {
				wikiSkips++
			}
		}
		if wikiSkips != 6 {
			t.Errorf("위키 항목 6개가 skip 이어야 한다, got %d", wikiSkips)
		}
	})

	t.Run("정상 위키에서는 fail 이 없다", func(t *testing.T) {
		res := Run(makeWiki(t, healthyYAML))
		if res.HasFail() {
			for _, f := range res.Findings {
				if f.Status == StatusFail {
					t.Errorf("fail 항목: %+v", f)
				}
			}
		}
		for _, id := range []string{"wiki.config", "wiki.config-unknown-keys", "wiki.min-wikilinks", "wiki.page-dirs", "wiki.root-files"} {
			if f := finding(res, id); f.Status != StatusOK {
				t.Errorf("%s 상태가 %s 다: %s", id, f.Status, f.Detail)
			}
		}
	})

	t.Run("min_wikilinks 0 은 warn 이다", func(t *testing.T) {
		res := Run(makeWiki(t, "min_wikilinks: 0\n"))
		f := finding(res, "wiki.min-wikilinks")
		if f.Status != StatusWarn {
			t.Fatalf("상태가 %s 다, warn 이어야 한다", f.Status)
		}
		if f.Fix == "" {
			t.Error("warn 항목에는 조치가 있어야 한다")
		}
	})

	t.Run("설정 파싱 실패는 fail 이다", func(t *testing.T) {
		root := makeWiki(t, healthyYAML)
		if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte("preset: [깨진\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res := Run(root)
		f := finding(res, "wiki.config")
		if f.Status != StatusFail {
			t.Fatalf("상태가 %s 다, fail 이어야 한다", f.Status)
		}
		if f.Fix == "" {
			t.Error("fail 항목에는 조치가 있어야 한다")
		}
	})

	t.Run("알 수 없는 키는 warn 이다", func(t *testing.T) {
		res := Run(makeWiki(t, healthyYAML+"no-such-key: 1\n"))
		if f := finding(res, "wiki.config-unknown-keys"); f.Status != StatusWarn {
			t.Errorf("상태가 %s 다, warn 이어야 한다: %s", f.Status, f.Detail)
		}
	})

	t.Run("없는 page_dir 과 root_file 은 fail 이다", func(t *testing.T) {
		root := makeWiki(t, healthyYAML)
		if err := os.Remove(filepath.Join(root, "context")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, "index.md")); err != nil {
			t.Fatal(err)
		}
		res := Run(root)
		if f := finding(res, "wiki.page-dirs"); f.Status != StatusFail || !strings.Contains(f.Detail, "context") {
			t.Errorf("page-dirs 판정이 틀리다: %+v", f)
		}
		if f := finding(res, "wiki.root-files"); f.Status != StatusFail || !strings.Contains(f.Detail, "index.md") {
			t.Errorf("root-files 판정이 틀리다: %+v", f)
		}
	})

	t.Run("항목 순서는 두 번 돌려도 같다", func(t *testing.T) {
		root := makeWiki(t, healthyYAML)
		first, second := Run(root), Run(root)
		if got := ids(first); !equal(got, wantIDs) {
			t.Errorf("항목 순서가 계약과 다르다:\n got %v\nwant %v", got, wantIDs)
		}
		if !equal(ids(first), ids(second)) {
			t.Errorf("두 실행의 순서가 다르다:\n%v\n%v", ids(first), ids(second))
		}
	})

	t.Run("git 저장소 위키에서 gitignore 여부를 본다", func(t *testing.T) {
		// git 이 없는 환경을 흉내내기 어려워 git 이 있는 전제로 검사한다.
		// 이 항목의 git 부재 분기(env.git fail 파급)는 환경을 강제로 바꿀 수 없어 여기서는 못 돌린다.
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git 이 없는 환경이라 gitignore 점검 검증을 건너뛴다")
		}
		root := makeWiki(t, healthyYAML)
		if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
			t.Fatalf("git init 실패: %v\n%s", err, out)
		}
		// gitignore 규칙이 없으면 warn 이다.
		res := Run(root)
		if f := finding(res, "wiki.engram-gitignore"); f.Status != StatusWarn || f.Fix == "" {
			t.Errorf("gitignore 없을 때 warn 이어야 한다: %+v", f)
		}
		// 규칙을 추가하면 ok 로 돌아온다.
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".engram/\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		res = Run(root)
		if f := finding(res, "wiki.engram-gitignore"); f.Status != StatusOK {
			t.Errorf("gitignore 있을 때 ok 여야 한다: %+v", f)
		}
	})

	t.Run("ok 가 아닌 항목에는 조치가 있다", func(t *testing.T) {
		res := Run(makeWiki(t, "min_wikilinks: 0\n"))
		for _, f := range res.Findings {
			if f.Status == StatusOK || f.Status == StatusSkip {
				continue
			}
			if f.Fix == "" {
				t.Errorf("%s (%s) 항목에 조치가 없다", f.ID, f.Status)
			}
		}
	})

	t.Run("요약 카운트는 항목 수와 일치한다", func(t *testing.T) {
		res := Run(makeWiki(t, healthyYAML))
		s := res.Summary
		if s.Items != len(res.Findings) {
			t.Errorf("items %d, findings %d", s.Items, len(res.Findings))
		}
		if s.OK+s.Warn+s.Fail+s.Skip != s.Items {
			t.Errorf("상태 합이 items 와 다르다: %+v", s)
		}
	})
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGitMissingIsWarn(t *testing.T) {
	// git 이 없는 환경을 흉내낸다. PATH 를 비워 git 실행을 실패하게 한다.
	// 이 테스트는 PATH 를 바꾸므로 병렬로 돌면 안 된다.
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	f, ok := checkGit()
	if ok {
		t.Skip("PATH 를 비웠는데 git 이 실행되었다. 흉내가 실패했다")
	}
	if f.Status != StatusWarn {
		t.Fatalf("git 부재는 warn 이어야 한다, got %s", f.Status)
	}
	// 조치 문구에 왜 지금은 문제가 아닌지가 담겨야 한다.
	if !strings.Contains(f.Fix, "git 없이") || !strings.Contains(f.Fix, "sync") {
		t.Errorf("조치 문구에 현재는 문제없다는 사실이 없음: %s", f.Fix)
	}
}

// TestConsoleFinding은 콘솔 인코딩 진단의 세 상황을 플랫폼 무관하게 덮는다.
// probeConsole 의 Windows 구현 자체는 macOS 에서 돌릴 수 없으므로
// 순수 함수인 consoleFinding 만 검증한다. 실제 콘솔에서의 판정은
// docs/windows-verification.md 절차로 사람이 확인한다.
func TestConsoleFinding(t *testing.T) {
	t.Run("콘솔 직결이고 출력 코드페이지가 65001 이면 ok", func(t *testing.T) {
		f := consoleFinding(consoleState{IsConsole: true, OutputCP: 65001, InputCP: 949})
		if f.Status != StatusOK {
			t.Fatalf("상태가 %s 입니다, ok 여야 합니다", f.Status)
		}
		if !strings.Contains(f.Detail, "전환") {
			t.Errorf("전환 사실을 알려야 합니다: %s", f.Detail)
		}
		// 입력 코드페이지가 949 여도 경고가 아니라는 것이 이번 수정의 핵심이다.
	})

	t.Run("콘솔 직결인데 65001 이 아니면 전환 실패 경고", func(t *testing.T) {
		f := consoleFinding(consoleState{IsConsole: true, OutputCP: 949})
		if f.Status != StatusWarn {
			t.Fatalf("상태가 %s 입니다, warn 이어야 합니다", f.Status)
		}
		if !strings.Contains(f.Detail, "출력 코드페이지") || !strings.Contains(f.Fix, "chcp 65001") {
			t.Errorf("경고 문구가 잘못되었습니다: %+v", f)
		}
	})

	t.Run("stdout 이 콘솔이 아니면 UTF-8 안내를 낸다", func(t *testing.T) {
		f := consoleFinding(consoleState{IsConsole: false})
		if f.Status != StatusOK {
			t.Fatalf("상태가 %s 입니다, 파이프는 이쪽 결함이 아니므로 ok 여야 합니다", f.Status)
		}
		if !strings.Contains(f.Detail, "UTF-8 바이트") {
			t.Errorf("안내가 잘못되었습니다: %s", f.Detail)
		}
		if !strings.Contains(f.Fix, "OutputEncoding") {
			t.Errorf("PowerShell 조치가 없습니다: %s", f.Fix)
		}
	})
}
