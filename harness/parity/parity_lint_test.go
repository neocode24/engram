package parity

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
)

// fixtureWiki는 공유 골든 위키의 패키지 기준 상대 경로다. lint_golden_test.go
// 와 같은 픽스처를 쓴다. 이 픽스처를 고치면 골든 스냅샷(harness/golden)도
// 함께 갱신해야 한다.
var fixtureWiki = filepath.Join("..", "fixtures", "golden-wiki")

// upstreamScript는 ENGRAM_UPSTREAM 저장소 안의 비교 대상 스크립트 경로다.
const upstreamScript = "scripts/lint-frontmatter.sh"

// TestLintParity는 같은 픽스처 위키에 대해 upstream lint-frontmatter.sh 와
// engram lint 의 위반 목록을 비교한다. 차이는 실패가 아니라 측정 결과다.
// 두 구현이 완전히 같지 않다는 전제로 측정하는 작업이므로, 차이를 로그로만
// 남긴다. 실패 조건은 양쪽 모두 위반 0건, 곧 비교 자체가 성립하지 않은
// 경우뿐이다.
func TestLintParity(t *testing.T) {
	root := expandTilde(os.Getenv("ENGRAM_UPSTREAM"))
	if root == "" {
		t.Skip("ENGRAM_UPSTREAM 이 없어 비교를 건너뛴다. ENGRAM_UPSTREAM=<llm-wiki 경로> 로 설정하면 돈다. CI 는 이 변수가 없으므로 항상 건너뛴다(ADR 0029).")
	}
	scriptPath := filepath.Join(root, filepath.FromSlash(upstreamScript))
	if _, err := os.Stat(scriptPath); err != nil {
		t.Skipf("upstream 스크립트가 없어 건너뛴다: %s: %v", scriptPath, err)
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash 가 없어 건너뛴다: %v", err)
	}

	// 두 구현을 같은 바이트 위에 돌린다. upstream 스크립트는 인자로 위키
	// 루트를 받지 않고 자기 위치의 상위 디렉토리를 루트로 보므로, 픽스처
	// 사본 루트의 scripts/ 아래에 스크립트 사본을 둔다. --include-inbox 를
	// 주는 이유는 engram lint 가 inbox 를 검사 범위에 두기 때문이다.
	wiki := t.TempDir()
	copyTree(t, fixtureWiki, wiki)
	usePresetTeam(t, wiki)
	if err := os.MkdirAll(filepath.Join(wiki, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Skipf("upstream 스크립트를 읽을 수 없어 건너뛴다: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "scripts", filepath.Base(upstreamScript)), script, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bash, filepath.Join(wiki, "scripts", filepath.Base(upstreamScript)), "--include-inbox")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("upstream 스크립트 실행 실패: %v\nstderr:\n%s", err, stderr.String())
		}
		// 종료 코드 0 은 통과, 1 은 위반 있음이다. 둘 다 정상 출력이다.
		// 그 밖의 종료 코드는 스크립트나 실행 환경이 바뀌었다는 뜻이다.
		// stderr 의 find 오류는 픽스처에 없는 스캔 루트에 대한 것이므로
		// 정상이다(README 의 스캔 범위 항 참고).
		if code := ee.ExitCode(); code != 1 {
			t.Fatalf("upstream 스크립트가 종료 코드 %d 로 끝났다. stderr:\n%s", code, stderr.String())
		}
	}
	up := ParseUpstreamOutput(stdout.String())

	cfg, err := config.Load(wiki)
	if err != nil {
		t.Fatalf("픽스처 사본의 설정을 읽을 수 없음: %v", err)
	}
	res, err := lint.Run(wiki, cfg)
	if err != nil {
		t.Fatalf("engram lint 실패: %v", err)
	}
	mine := make([]Violation, 0, len(res.Violations))
	for _, v := range res.Violations {
		mine = append(mine, Violation{Path: v.Path, Rule: NormalizeEngram(v.Rule, v.Message)})
	}

	if len(up) == 0 && len(mine) == 0 {
		t.Fatal("양쪽 모두 위반 0건이면 일치가 아니라 비교가 성립하지 않은 것이다. 스크립트가 안 돌았거나 픽스처를 못 찾은 경우다.")
	}
	report(t, up, mine, res)
}

// expandTilde는 경로가 ~ 로 시작하면 홈 디렉토리로 펼친다. 셸에 따라
// 변수 대입문의 ~ 확장이 안 되는 경우를 받아 준다.
func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// copyTree는 디렉토리 트리 전체를 바이트 그대로 복사한다. CRLF 픽스처의
// 줄바꿈이 바뀌지 않아야 upstream 스크립트의 판정이 원본과 같다.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("픽스처 복사 실패: %v", err)
	}
}

// pairCount는 정규화한 위반 쌍 하나의 양쪽 개수다.
type pairCount struct {
	v          Violation
	up, engram int
}

// splitPairs는 위반 목록 두 벌을 (경로, 규칙) 쌍 단위로 갈라 세 갈래로
// 낸다. 같은 쌍이 여러 번 나오면 개수로 센다. 깨진 위키링크처럼 한 문서에
// 같은 규칙 위반이 겹치는 경우 개수 차이가 곧 차이이기 때문이다.
func splitPairs(up, mine []Violation) (both, upOnly, mineOnly []pairCount) {
	count := func(vs []Violation) map[Violation]int {
		m := make(map[Violation]int, len(vs))
		for _, v := range vs {
			m[v]++
		}
		return m
	}
	upCount, mineCount := count(up), count(mine)

	ordered := append(append([]Violation{}, up...), mine...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Path != ordered[j].Path {
			return ordered[i].Path < ordered[j].Path
		}
		return ordered[i].Rule < ordered[j].Rule
	})
	seen := map[Violation]bool{}
	for _, v := range ordered {
		if seen[v] {
			continue
		}
		seen[v] = true
		pc := pairCount{v: v, up: upCount[v], engram: mineCount[v]}
		switch {
		case pc.up > 0 && pc.engram > 0:
			both = append(both, pc)
		case pc.up > 0:
			upOnly = append(upOnly, pc)
		default:
			mineOnly = append(mineOnly, pc)
		}
	}
	return both, upOnly, mineOnly
}

// report는 비교 결과를 세 갈래로 로그에 남긴다. 사람이 docs/parity.md 로
// 옮길 수 있는 형태로 낸다.
func report(t *testing.T, up, mine []Violation, res lint.Result) {
	t.Helper()
	both, upOnly, mineOnly := splitPairs(up, mine)
	t.Logf("upstream 위반 %d건, engram 위반 %d건", len(up), len(mine))

	t.Logf("양쪽 다 잡음(일치) %d 쌍:", len(both))
	logPairs(t, both)
	t.Logf("upstream 만 잡음(engram 이 놓친 규칙) %d 쌍:", len(upOnly))
	logPairs(t, upOnly)
	t.Logf("engram 만 잡음(engram 이 더 엄격하거나 오탐) %d 쌍:", len(mineOnly))
	logPairs(t, mineOnly)

	unmapped := 0
	for _, pcs := range [][]pairCount{both, upOnly, mineOnly} {
		for _, pc := range pcs {
			if strings.HasPrefix(pc.v.Rule, "unmapped:") {
				unmapped++
			}
		}
	}
	if unmapped > 0 {
		t.Logf("매핑 없음 %d 쌍. normalize.go 의 매핑 표를 늘려야 한다.", unmapped)
	}

	// 위키 단위 진단은 upstream 에 대응 개념이 없어 쌍 비교에서 뺀다.
	// 그대로 묻으면 위반 수로 보이므로 따로 밝혀 둔다.
	for _, f := range res.WikiFindings {
		t.Logf("위키 단위 진단(비교 축 밖): %s 주제 %q, 문서 %d개 중 %d개", f.Rule, f.Topic, f.Total, len(f.Paths))
	}
}

// logPairs는 쌍 목록을 로그로 남긴다. 양쪽이 잡았는데 개수가 다른 쌍은 그
// 사실을 함께 적는다.
func logPairs(t *testing.T, pcs []pairCount) {
	t.Helper()
	for _, pc := range pcs {
		note := ""
		if pc.up > 0 && pc.engram > 0 && pc.up != pc.engram {
			note = fmt.Sprintf(" (upstream %d건, engram %d건)", pc.up, pc.engram)
		}
		t.Logf("  %s %s%s", pc.v.Path, pc.v.Rule, note)
	}
}

// usePresetTeam은 사본 위키의 프리셋을 team으로 올린다.
//
// upstream 스크립트는 engram.yaml을 읽지 않고 자기 스키마를 하드코딩한다.
// 그 스키마는 scope, sensitivity, source_channel, trigger_mode, workflow를
// 전부 요구하는데 골든 픽스처의 personal 프리셋은 그 속성들을 끈다. 프리셋을
// 맞추지 않으면 비교 결과가 축 on/off 차이로 뒤덮여 실제 규칙 차이가 묻힌다.
// 측정으로는 personal에서 error 4건이던 것이 team에서 48건이 되고, 늘어난
// 44건이 전부 upstream만 잡던 필드 누락이다.
//
// 원본 픽스처는 건드리지 않는다. lint_golden_test.go가 personal 프리셋
// 기준으로 스냅샷을 들고 있다.
func usePresetTeam(t *testing.T, wiki string) {
	t.Helper()
	path := filepath.Join(wiki, "engram.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	swapped := strings.Replace(string(raw), "preset: minimal", "preset: team", 1)
	if swapped == string(raw) {
		t.Fatalf("픽스처 engram.yaml에서 preset: minimal 을 찾지 못했다: %s", path)
	}
	if err := os.WriteFile(path, []byte(swapped), 0o644); err != nil {
		t.Fatal(err)
	}
}
