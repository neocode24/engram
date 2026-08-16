package parity

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
	"github.com/neocode24/engram/internal/resurface"
)

// upstreamResurfaceScript는 비교 대상 upstream 스크립트의 저장소 안 경로다.
const upstreamResurfaceScript = "scripts/wiki_resurface.py"

// resurfaceLimit는 양쪽이 내는 후보 수 상한이다. 픽스처의 context 문서
// 전부를 후보로 삼을 수 있는 값이다. 전체 순서를 비교하려면 이만큼 넉넉해야
// 한다.
const resurfaceLimit = 6

// TestResurfaceParity는 같은 픽스처 위키에 대해 upstream wiki_resurface.py
// 와 engram resurface 의 선정 순위를 비교한다. 비교 단위는 후보 슬러그의
// 목록과 그 순서다. 점수 계산이 달라도 순서가 같으면 같은 판정으로 본다.
// 차이는 실패가 아니라 측정 결과다. 실패 조건은 양쪽 모두 후보 0건, 곧
// 비교가 성립하지 않은 경우뿐이다.
//
// resurface 는 상태를 쓰는 유일한 조회 커맨드라 그대로 두 번 돌리면
// 결과가 달라진다(ADR 0005의 함정). 그래서 양쪽 모두 상태를 쓰지 않는
// 방식으로 부른다. upstream 은 --no-state, engram 은 --dry-run 이다.
// 시작 상태도 사본에 상태 파일이 없는 빈 상태다.
//
// 두 구현은 날짜의 진실원이 다르다. upstream 은 git 커밋 시각이고 engram 은
// 프론트매터(updated 우선)다. 그래서 사본을 git 저장소로 만들고 문서의 기준
// 날짜로 커밋 시각을 맞춘다. 기준 날짜 판정의 진실원은 resurface.BaseDate 다.
func TestResurfaceParity(t *testing.T) {
	root := expandTilde(os.Getenv("ENGRAM_UPSTREAM"))
	if root == "" {
		t.Skip("ENGRAM_UPSTREAM 이 없어 비교를 건너뛴다. ENGRAM_UPSTREAM=<llm-wiki 경로> 로 설정하면 돈다. CI 는 이 변수가 없으므로 항상 건너뛴다(ADR 0029).")
	}
	scriptPath := filepath.Join(root, filepath.FromSlash(upstreamResurfaceScript))
	if _, err := os.Stat(scriptPath); err != nil {
		t.Skipf("upstream 스크립트가 없어 건너뛴다: %s: %v", scriptPath, err)
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skipf("python3 가 없어 건너뛴다: %v", err)
	}
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git 이 없어 건너뛴다. upstream 스크립트가 문서 나이를 git 커밋 시각에서 읽는다: %v", err)
	}

	wiki := t.TempDir()
	copyTree(t, fixtureWiki, wiki)
	if err := os.MkdirAll(filepath.Join(wiki, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Skipf("upstream 스크립트를 읽을 수 없어 건너뛴다: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "scripts", filepath.Base(upstreamResurfaceScript)), script, 0o644); err != nil {
		t.Fatal(err)
	}

	// upstream 스크립트는 자기 위치의 상위를 위키 루트로 보므로 사본 루트의
	// scripts/ 아래에 둔다. lint 축과 같은 방식이다.
	gitInit(t, gitBin, wiki)
	for _, c := range contextCommits(t, wiki) {
		gitCommitAt(t, gitBin, wiki, c.date, c.rel)
	}
	// context 밖의 파일은 순위에 들어오지 않는다. 커밋하지 않으면 빈
	// 커밋 목록이 되므로 한 커밋으로 마무리한다.
	gitCommitRest(t, gitBin, wiki)

	// upstream 은 실행 시각을 고정할 수 없다. 대신 커밋 시각을 고정했으므로
	// 나잇값 사이 격차가 며칠 이상이면 순위가 초 단위 흔들림에 뒤집히지
	// 않는다. engram 쪽 기준 시각은 같은 실행의 실제 시각으로 맞춘다.
	cmd := exec.Command(py,
		filepath.Join(wiki, "scripts", filepath.Base(upstreamResurfaceScript)),
		"--json", "--no-state", "--no-embed", "-n", "6")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream 스크립트 실행 실패: %v\nstderr:\n%s", err, errBuf.String())
	}
	var up struct {
		Resurface []struct {
			Slug         string `json:"slug"`
			StaleDays    int    `json:"stale_days"`
			InboundLinks int    `json:"inbound_links"`
		} `json:"resurface"`
	}
	if err := json.Unmarshal(out.Bytes(), &up); err != nil {
		t.Fatalf("upstream --json 출력을 파싱할 수 없다: %v\n%s", err, out.String())
	}

	cfg, err := config.Load(wiki)
	if err != nil {
		t.Fatalf("사본의 설정을 읽을 수 없음: %v", err)
	}
	now := time.Now().UTC()
	res, err := resurface.Run(wiki, cfg, now, resurfaceLimit, true)
	if err != nil {
		t.Fatalf("engram resurface 실패: %v", err)
	}

	if len(up.Resurface) == 0 && len(res.Candidates) == 0 {
		t.Fatal("양쪽 모두 후보 0건이면 일치가 아니라 비교가 성립하지 않은 것이다. 스크립트가 안 돌았거나 픽스처를 못 찾은 경우다.")
	}
	reportResurface(t, up.Resurface, res)
}

// contextCommits은 context 문서별 커밋 시각을 만든다. 날짜는 문서의 기준
// 날짜를 그대로 쓴다. 기준 날짜 판정의 진실원은 resurface.BaseDate 이므로
// 여기서 다시 정의하지 않는다. 커밋 시각은 그날 정오로 고정한다.
func contextCommits(t *testing.T, wiki string) []commitSpec {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(wiki, "context"))
	if err != nil {
		t.Fatalf("픽스처 사본의 context 디렉토리를 읽을 수 없음: %v", err)
	}
	var out []commitSpec
	for _, e := range entries {
		// upstream 스크립트가 context/README.md 를 건너뛰므로 여기도 뺀다.
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" {
			continue
		}
		rel := "context/" + e.Name()
		raw, err := os.ReadFile(filepath.Join(wiki, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		d, err := doc.Parse(rel, raw)
		if err != nil {
			t.Fatalf("픽스처 문서 파싱 실패: %s: %v", rel, err)
		}
		base, ok := resurface.BaseDate(d)
		if !ok {
			t.Fatalf("기준 날짜가 없는 context 문서는 커밋 시각을 정할 수 없다: %s", rel)
		}
		out = append(out, commitSpec{date: base.Format("2006-01-02T15:04:05"), rel: rel})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

// commitSpec은 파일 하나를 남길 커밋의 시각과 경로다.
type commitSpec struct {
	date string
	rel  string
}

// gitInit은 사본을 git 저장소로 만든다. upstream 스크립트가 문서 나이를
// git 로그에서 읽기 때문이다.
func gitInit(t *testing.T, gitBin, repo string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "parity"},
		{"config", "user.email", "parity@example.com"},
	} {
		runGit(t, gitBin, repo, nil, args...)
	}
}

// gitCommitAt은 문서 하나를 지정 시각의 커밋으로 남긴다. upstream 이 형식
// 변경 커밋을 걸러내는 기준이 커밋당 파일 15개이므로 커밋마다 문서 하나만
// 둔다. 그래야 문서별 시각을 독립적으로 정할 수 있다.
func gitCommitAt(t *testing.T, gitBin, repo, date, rel string) {
	t.Helper()
	env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	runGit(t, gitBin, repo, nil, "add", rel)
	runGit(t, gitBin, repo, env, "commit", "-q", "-m", rel)
}

// gitCommitRest은 context 밖의 나머지 파일을 한 커밋으로 남긴다. 이 커밋은
// upstream 의 git 로그 조회가 context 경로만 보므로 순위에 영향을 주지
// 않는다.
func gitCommitRest(t *testing.T, gitBin, repo string) {
	t.Helper()
	env := []string{"GIT_AUTHOR_DATE=2026-01-01T12:00:00", "GIT_COMMITTER_DATE=2026-01-01T12:00:00"}
	runGit(t, gitBin, repo, nil, "add", "-A")
	runGit(t, gitBin, repo, env, "commit", "-q", "-m", "rest")
}

// runGit는 git 명령을 사본 안에서 돈다.
func runGit(t *testing.T, gitBin, repo string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command(gitBin, args...)
	cmd.Dir = repo
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s 실패: %v\n%s", strings.Join(args, " "), err, errBuf.String())
	}
}

// reportResurface는 선정 순위 비교 결과를 로그로 남긴다. 사람이
// docs/parity.md 로 옮겨 적는 형태다.
func reportResurface(t *testing.T, up []struct {
	Slug         string `json:"slug"`
	StaleDays    int    `json:"stale_days"`
	InboundLinks int    `json:"inbound_links"`
}, res resurface.Result) {
	t.Helper()
	upPos := map[string]int{}
	var upSlugs []string
	for i, r := range up {
		upPos[r.Slug] = i + 1
		upSlugs = append(upSlugs, r.Slug)
	}
	var mineSlugs []string
	for _, c := range res.Candidates {
		mineSlugs = append(mineSlugs, c.Slug)
	}
	t.Logf("upstream 후보 %d건, engram 후보 %d건", len(up), len(res.Candidates))
	t.Logf("upstream 순위: %s", strings.Join(upSlugs, " > "))
	t.Logf("engram 순위: %s", strings.Join(mineSlugs, " > "))

	for i, c := range res.Candidates {
		if j, ok := upPos[c.Slug]; ok {
			t.Logf("  %d위 %s: engram 경과 %d일, upstream 경과 %d일 인바운드 %d개",
				i+1, c.Slug, c.AgeDays, up[j-1].StaleDays, up[j-1].InboundLinks)
		} else {
			t.Logf("  %d위 %s: engram 만 후보로 삼았다", i+1, c.Slug)
		}
	}
	minePos := map[string]bool{}
	for _, s := range mineSlugs {
		minePos[s] = true
	}
	for _, s := range upSlugs {
		if !minePos[s] {
			t.Logf("  %s: upstream 만 후보로 삼았다", s)
		}
	}
	if strings.Join(upSlugs, " > ") == strings.Join(mineSlugs, " > ") {
		t.Log("순위가 완전히 같다")
		return
	}
	for i := 0; i < len(upSlugs) && i < len(mineSlugs); i++ {
		if upSlugs[i] != mineSlugs[i] {
			t.Logf("첫 갈림은 %d위다. upstream %s, engram %s", i+1, upSlugs[i], mineSlugs[i])
			return
		}
	}
}
