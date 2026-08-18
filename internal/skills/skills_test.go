package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDoc(t *testing.T) {
	t.Run("문서는 프론트매터에 name 과 description 이 있다", func(t *testing.T) {
		doc := Doc()
		if doc == "" {
			t.Fatal("임베드된 문서가 비어 있음")
		}
		if !strings.HasPrefix(doc, "---\n") {
			t.Fatalf("프론트매터가 없음:\n%s", doc[:40])
		}
		for _, key := range []string{"name:", "description:"} {
			if !strings.Contains(doc, key) {
				t.Errorf("프론트매터에 %s 없음", key)
			}
		}
	})

	// 문서가 반드시 담는 경계 문구다. 이 테스트가 그것이 지워지는 것을
	// 막는 유일한 장치다(ADR 0041, 0055, 0057).
	t.Run("문서는 경계 문구를 담는다", func(t *testing.T) {
		doc := Doc()
		for _, want := range []string{
			"engram은 LLM을 부르지 않는다", // 호출 방향
			"inbox까지",              // 혼자 판단해 쓰는 범위
			"확정은 사람이 한다",           // 승급 확정 주체
			"승인 없이 실행하지 마라",        // 승인 뒤에야 promote 한다
			"원문 보존을 반드시 묻는다",       // 되돌릴 수 없는 판단은 사람 몫
			"파일을 직접 만들거나 고치지 마라",   // 커맨드로만 바꾼다
			"--dry-run",     // 스스로 검증
			"--json",        // 조회의 주 경로
			"--now",         // 기준 시각
			"rules show",    // 규칙의 진실원
			"파일을 직접 읽지 않는다", // 위키 내용은 recall로 꺼낸다
			"recall",        // 위키 내용을 꺼내는 커맨드
		} {
			if !strings.Contains(doc, want) {
				t.Errorf("문서에 %q 없음. 경계 문구가 지워졌다", want)
			}
		}
	})

	t.Run("문서에는 임계값이나 허용값이 없다", func(t *testing.T) {
		// 정적 문서라는 계약을 지키는 장치다. 키 이름에 숫자가 붙으면
		// 위키 설정을 문서에 박은 것이다.
		doc := Doc()
		pattern := regexp.MustCompile(`(min_wikilinks|stale_days|max_lines|broad_topic_pct)[^\n]*\d`)
		if m := pattern.FindString(doc); m != "" {
			t.Errorf("문서에 임계값이 있다: %q", m)
		}
	})

	t.Run("커맨드 목록을 통째로 베끼지 않는다", func(t *testing.T) {
		// --help 가 목록의 진실원이다. 문서는 갈래와 대표만 담는다.
		doc := Doc()
		for _, cmd := range []string{"reindex", "doctor", "backlinks"} {
			if strings.Contains(doc, cmd) {
				t.Errorf("대표가 아닌 커맨드 %q 가 문서에 있음", cmd)
			}
		}
		if !strings.Contains(doc, "engram --help") {
			t.Error("--help 안내가 없음")
		}
	})
}

func TestDetect(t *testing.T) {
	t.Run("실제로 존재하는 디렉토리만 대상이다", func(t *testing.T) {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
		got := Detect(home)
		if len(got) != 1 || !strings.HasSuffix(filepath.ToSlash(got[0]), ".claude/skills") {
			t.Errorf("존재하는 디렉토리만 잡아야 함: %v", got)
		}
	})

	t.Run("하나도 없으면 빈 목록이다", func(t *testing.T) {
		if got := Detect(t.TempDir()); len(got) != 0 {
			t.Errorf("대상이 없어야 함: %v", got)
		}
	})

	t.Run("심볼릭 링크로 같은 곳을 가리키면 하나로 합친다", func(t *testing.T) {
		home := t.TempDir()
		real := filepath.Join(home, ".agents", "skills")
		if err := os.MkdirAll(real, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(home, ".claude", "skills")); err != nil {
			t.Skipf("심볼릭 링크를 만들 수 없는 환경: %v", err)
		}
		got := Detect(home)
		if len(got) != 1 {
			t.Errorf("같은 물리적 디렉토리는 하나로 합쳐야 함: %v", got)
		}
	})
}
