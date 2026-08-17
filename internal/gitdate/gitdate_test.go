package gitdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// needGit는 git 이 없으면 테스트를 건너뛴다.
func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 이 PATH 에 없다")
	}
}

// gitRun은 임시 저장소에서 git 을 돌린다.
func gitRun(t *testing.T, root, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if date != "" {
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE="+date+"T12:00:00+00:00",
			"GIT_COMMITTER_DATE="+date+"T12:00:00+00:00",
		)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v 실패: %v\n%s", args, err, out)
	}
}

// commitAll은 날짜를 고정해 전체를 커밋한다.
func commitAll(t *testing.T, root, date, msg string) {
	t.Helper()
	gitRun(t, root, date, "add", "-A")
	gitRun(t, root, date,
		"-c", "user.name=시험", "-c", "user.email=test@example.com",
		"commit", "-m", msg)
}

// writeFile은 임시 디렉토리에 파일 하나를 만든다.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHistory(t *testing.T) {
	t.Run("최초와 마지막 커밋 날짜를 나눠 얻는다", func(t *testing.T) {
		needGit(t)
		root := t.TempDir()
		gitRun(t, root, "", "init")
		writeFile(t, filepath.Join(root, "context", "a.md"), "첫 내용\n")
		commitAll(t, root, "2026-01-01", "첫 커밋")
		writeFile(t, filepath.Join(root, "context", "a.md"), "둘째 내용\n")
		commitAll(t, root, "2026-02-02", "둘째 커밋")

		hist, err := History(root)
		if err != nil {
			t.Fatal(err)
		}
		d, ok := hist["context/a.md"]
		if !ok {
			t.Fatalf("context/a.md 가 이력에 없음: %+v", hist)
		}
		if d.First != "2026-01-01" || d.Last != "2026-02-02" {
			t.Errorf("날짜 = First %q, Last %q, want 2026-01-01 / 2026-02-02", d.First, d.Last)
		}
	})

	t.Run("저장소가 위키를 하위 디렉토리로 둬도 경로를 맞춘다", func(t *testing.T) {
		needGit(t)
		repo := t.TempDir()
		gitRun(t, repo, "", "init")
		writeFile(t, filepath.Join(repo, "wiki", "context", "a.md"), "내용\n")
		writeFile(t, filepath.Join(repo, "바깥.md"), "위키 밖 파일\n")
		commitAll(t, repo, "2026-03-03", "커밋")

		hist, err := History(filepath.Join(repo, "wiki"))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := hist["context/a.md"]; !ok {
			t.Fatalf("위키 기준 경로가 키여야 함: %+v", hist)
		}
		if _, ok := hist["바깥.md"]; ok {
			t.Errorf("위키 밖 파일이 이력에 들어옴: %+v", hist)
		}
	})

	t.Run("커밋되지 않은 파일은 이력에 없다", func(t *testing.T) {
		needGit(t)
		root := t.TempDir()
		gitRun(t, root, "", "init")
		writeFile(t, filepath.Join(root, "context", "a.md"), "내용\n")
		commitAll(t, root, "2026-01-01", "첫 커밋")
		writeFile(t, filepath.Join(root, "context", "b.md"), "아직 커밋 안 함\n")

		hist, err := History(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := hist["context/b.md"]; ok {
			t.Errorf("커밋되지 않은 파일이 이력에 있음: %+v", hist)
		}
	})

	t.Run("저장소가 아니면 안내와 함께 실패한다", func(t *testing.T) {
		needGit(t)
		root := t.TempDir()
		_, err := History(root)
		if err == nil {
			t.Fatal("저장소가 아니면 에러여야 함")
		}
		if want := "git init"; !strings.Contains(err.Error(), want) {
			t.Errorf("에러에 %q 안내가 없음: %v", want, err)
		}
	})

	t.Run("커밋이 없는 저장소는 빈 이력을 낸다", func(t *testing.T) {
		needGit(t)
		root := t.TempDir()
		gitRun(t, root, "", "init")
		writeFile(t, filepath.Join(root, "context", "a.md"), "내용\n")
		hist, err := History(root)
		if err != nil {
			t.Fatalf("커밋이 없어도 실패가 아니어야 함: %v", err)
		}
		if len(hist) != 0 {
			t.Errorf("빈 이력이어야 함: %+v", hist)
		}
	})
}

// TestHistoryWithQuotedPaths는 core.quotepath 가 켜진 저장소에서 한글
// 파일명의 이력을 읽는지 본다.
//
// git 의 기본값이 켜짐이고, 켜져 있으면 비ASCII 경로가
// "sources/\354\233\220\353\263\270.md" 처럼 이스케이프되어 나온다. 그
// 상태에서 경로 대조가 깨지면 sync 가 한글 문서를 조용히 건너뛴다.
//
// 이 결함은 작성자 기계의 시스템 git 설정에 core.quotepath=false 가 있어
// 로컬에서만 통과했고 CI 에서만 실패했다. 그래서 저장소에 값을 명시로
// 박아 환경과 무관하게 재현한다. 환경변수로 흉내내면 그 환경변수를
// 지원하지 않는 git 버전에서 검사가 조용히 사라진다.
func TestHistoryWithQuotedPaths(t *testing.T) {
	needGit(t)
	root := t.TempDir()
	gitRun(t, root, "", "init")
	// CI 기본값을 저장소에 못 박는다. 작성자 기계에서도 실패가 재현된다.
	gitRun(t, root, "", "config", "core.quotepath", "true")

	korean := filepath.Join("sources", "2026-01-01-원본.md")
	writeFile(t, filepath.Join(root, korean), "원본\n")
	writeFile(t, filepath.Join(root, "context", "ascii.md"), "ascii\n")
	commitAll(t, root, "2026-01-01", "첫 커밋")

	hist, err := History(root)
	if err != nil {
		t.Fatalf("이력을 읽을 수 없음: %v", err)
	}

	rel := filepath.ToSlash(korean)
	d, ok := hist[rel]
	if !ok {
		var keys []string
		for k := range hist {
			keys = append(keys, k)
		}
		t.Fatalf("한글 파일명 문서가 이력에 없다: %s\n실제 키: %v", rel, keys)
	}
	if d.First != "2026-01-01" || d.Last != "2026-01-01" {
		t.Errorf("한글 문서 날짜 = %+v, 기대 2026-01-01", d)
	}
	if _, ok := hist["context/ascii.md"]; !ok {
		t.Error("ascii 문서가 이력에 없다")
	}
}
