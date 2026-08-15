// Package doctor는 환경과 위키 설정을 진단한다. 진단만 하고 고치지 않는다.
// 점검 실패를 나열만 하면 지원 요청이 그대로 들어오므로 ok 가 아닌 항목에는
// 조치를 함께 담는다. 이것이 이 패키지의 존재 이유다.
package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/neocode24/engram/internal/config"
)

// Status는 점검 항목 하나의 상태다.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Finding은 점검 항목 하나의 결과다. ID 는 점 표기 소문자로 지었고
// 출력과 JSON 의 안정적인 키다. Fix 는 ok 가 아닌 항목의 조치다.
type Finding struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
	Fix    string `json:"fix"`
}

// Summary는 상태별 항목 수다.
type Summary struct {
	Items int `json:"items"`
	OK    int `json:"ok"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
	Skip  int `json:"skip"`
}

// Result는 doctor 실행 결과다. 항목 순서는 Run 이 고정하므로
// 같은 환경에서는 항상 같은 순서로 나온다.
type Result struct {
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

// HasFail은 fail 항목이 하나라도 있는지를 반환한다. 종료 코드 판정에 쓴다.
func (r Result) HasFail() bool {
	return r.Summary.Fail > 0
}

// Run은 대상 경로의 환경과 위키를 진단한다. 위키 여부는 engram.yaml 존재로
// 판단하며 위키가 아니면 환경 항목만 검사하고 위키 항목은 skip 이다.
func Run(root string) Result {
	// 환경 항목 다섯. 순서가 곧 출력 순서다.
	gitFinding, hasGit := checkGit()
	findings := []Finding{gitFinding}
	findings = append(findings, checkAutoCRLF(root, hasGit))
	findings = append(findings, checkFSCase())
	findings = append(findings, checkConsoleEncoding())
	findings = append(findings, checkWritePerm(root))

	// 위키 항목 여섯. engram.yaml 이 없으면 모두 skip 이다.
	cfg, isWiki, cfgFinding := loadConfig(root)
	findings = append(findings, cfgFinding)
	findings = append(findings, checkUnknownKeys(cfg, isWiki))
	findings = append(findings, checkMinWikilinks(cfg, isWiki))
	findings = append(findings, checkPageDirs(root, cfg, isWiki))
	findings = append(findings, checkRootFiles(root, cfg, isWiki))
	findings = append(findings, checkEngramGitignore(root, hasGit, isWiki))

	res := Result{Findings: findings}
	res.Summary.Items = len(findings)
	for _, f := range findings {
		switch f.Status {
		case StatusOK:
			res.Summary.OK++
		case StatusWarn:
			res.Summary.Warn++
		case StatusFail:
			res.Summary.Fail++
		case StatusSkip:
			res.Summary.Skip++
		}
	}
	return res
}

// gitOutput는 git 명령을 실행해 표준 출력을 돌려준다.
func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// checkGit는 git 실행 가능 여부와 버전을 본다. git 이 없어도 fail 이
// 아니라 warn 이다. fail 은 종료 코드 1 을 만들어 스크립트와 CI 를
// 막는데 지금 구현된 커맨드는 git 없이 전부 동작하고 git 이 필요한 것은
// sync 와 updated 필드 자동 채움뿐이기 때문이다. 필수가 아닌 것으로 막지 않는다.
func checkGit() (Finding, bool) {
	ver, err := gitOutput("--version")
	if err != nil {
		return Finding{
			ID:     "env.git",
			Status: StatusWarn,
			Detail: "git 을 실행할 수 없다",
			Fix:    "지금 구현된 커맨드는 git 없이 전부 동작하므로 당장은 문제가 아니다. 이후 sync 와 updated 필드 자동 채움에 필요하니 그 전에 설치한다. macOS 는 xcode-select --install, Windows 는 Git for Windows 설치",
		}, false
	}
	return Finding{ID: "env.git", Status: StatusOK, Detail: ver}, true
}

// isGitRepo는 대상 경로가 git 작업 폴더 안에 있는지를 본다.
func isGitRepo(root string) bool {
	out, err := gitOutput("-C", root, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// checkAutoCRLF는 core.autocrlf 설정을 본다. true 면 프론트매터와 골든 비교가
// 줄바꿈 변환으로 틀어진다. 위키가 git 저장소가 아니면 skip 이다.
func checkAutoCRLF(root string, hasGit bool) Finding {
	f := Finding{ID: "env.git-autocrlf"}
	if !hasGit {
		f.Status, f.Detail = StatusSkip, "git 이 없어 확인할 수 없다"
		return f
	}
	if !isGitRepo(root) {
		f.Status, f.Detail = StatusSkip, "대상 경로가 git 저장소가 아니다"
		return f
	}
	val, err := gitOutput("-C", root, "config", "--get", "core.autocrlf")
	if err != nil && !valueAbsent(err) {
		// --get 은 값이 없으면 종료 코드 1 을 낸다. 그 외의 실패면 설정을 못 읽은 것이다.
		f.Status, f.Detail = StatusWarn, "core.autocrlf 값을 읽을 수 없다"
		f.Fix = "git config core.autocrlf input 으로 직접 확인한다"
		return f
	}
	if val == "" {
		val = "설정 없음"
	}
	if val == "true" {
		if runtime.GOOS == "windows" {
			f.Status = StatusFail
		} else {
			f.Status = StatusWarn
		}
		f.Detail = "core.autocrlf 가 true 다. 줄바꿈이 자동 변환되어 프론트매터와 골든 비교가 틀어진다"
		f.Fix = "git config core.autocrlf input"
		return f
	}
	f.Status, f.Detail = StatusOK, fmt.Sprintf("core.autocrlf %s", val)
	return f
}

// valueAbsent는 git config --get 이 값이 없어 종료 코드 1 을 낸 것인지 본다.
// 그것은 에러가 아니라 설정이 없다는 뜻이다.
func valueAbsent(err error) bool {
	var exitErr *exec.ExitError
	return err != nil && errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

// checkFSCase는 임시 파일을 실제로 만들어 대소문자 구분 여부를 본다.
// 추측하지 않는다. 대소문자를 무시하는 파일시스템이면 슬러그 충돌이
// 조용히 일어난다.
func checkFSCase() Finding {
	f := Finding{ID: "env.fs-case"}
	dir, err := os.MkdirTemp("", "engram-doctor-case-")
	if err != nil {
		f.Status, f.Detail = StatusFail, "임시 디렉토리를 만들 수 없어 확인에 실패했다"
		f.Fix = "TMPDIR 환경변수와 임시 디렉토리 권한을 확인한다"
		return f
	}
	defer os.RemoveAll(dir)
	lower := filepath.Join(dir, "probe")
	if err := os.WriteFile(lower, []byte("x"), 0o644); err != nil {
		f.Status, f.Detail = StatusFail, "임시 파일을 만들 수 없어 확인에 실패했다"
		f.Fix = "임시 디렉토리 쓰기 권한을 확인한다"
		return f
	}
	if _, err := os.Stat(filepath.Join(dir, "Probe")); err == nil {
		f.Status, f.Detail = StatusWarn, "파일시스템이 대소문자를 무시한다. 대소문자만 다른 슬러그가 서로 겹친다"
		f.Fix = "슬러그에 대소문자만 다른 이름을 쓰지 않는다"
		return f
	}
	f.Status, f.Detail = StatusOK, "파일시스템이 대소문자를 구분한다"
	return f
}

// checkConsoleEncoding은 콘솔 출력 인코딩을 본다.
// Windows 에서 UTF-8 이 아니면 한국어 메시지가 깨진다.
func checkConsoleEncoding() Finding {
	f := Finding{ID: "env.console-encoding"}
	if runtime.GOOS != "windows" {
		f.Status, f.Detail = StatusOK, "Windows 가 아니므로 콘솔은 UTF-8 을 쓴다"
		return f
	}
	out, err := exec.Command("cmd", "/c", "chcp").Output()
	if err != nil {
		f.Status, f.Detail = StatusWarn, "콘솔 코드페이지를 읽을 수 없다"
		f.Fix = "콘솔에서 chcp 를 직접 실행해 코드페이지를 확인한다"
		return f
	}
	text := strings.TrimSpace(string(out))
	// chcp 출력은 "Active code page: 949" 형태다. 마지막 필드가 코드페이지 숫자다.
	fields := strings.Fields(text)
	code := ""
	if len(fields) > 0 && isDigits(fields[len(fields)-1]) {
		code = fields[len(fields)-1]
	}
	if code == "" {
		f.Status, f.Detail = StatusWarn, fmt.Sprintf("콘솔 코드페이지를 해석할 수 없다: %q", text)
		f.Fix = "콘솔에서 chcp 를 직접 실행해 코드페이지를 확인한다"
		return f
	}
	if code != "65001" {
		f.Status = StatusWarn
		f.Detail = fmt.Sprintf("콘솔 코드페이지가 %s 다. 한국어 메시지가 깨질 수 있다", code)
		f.Fix = "콘솔에서 chcp 65001 을 실행한다"
		return f
	}
	f.Status, f.Detail = StatusOK, "콘솔 코드페이지가 65001 (UTF-8) 이다"
	return f
}

// probeWrite는 디렉토리에 실제로 파일을 써 본다.
func probeWrite(dir string) error {
	tmp, err := os.CreateTemp(dir, ".engram-doctor-write-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

// isDigits는 문자열이 숫자로만 되어 있는지 본다.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// checkWritePerm은 위키 루트에 실제로 쓸 수 있는지 본다.
// 아직 없는 page_dir 은 wiki.page-dirs 항목이 다루므로 여기서는 루트만 본다.
func checkWritePerm(root string) Finding {
	f := Finding{ID: "env.write-perm"}
	info, err := os.Stat(root)
	if err != nil {
		f.Status, f.Detail = StatusSkip, "대상 경로가 없다"
		return f
	}
	if !info.IsDir() {
		f.Status, f.Detail = StatusSkip, "대상 경로가 디렉토리가 아니다"
		return f
	}
	if err := probeWrite(root); err != nil {
		f.Status = StatusFail
		f.Detail = fmt.Sprintf("위키 루트에 쓸 수 없다: %v", err)
		f.Fix = fmt.Sprintf("chmod u+w %s 또는 디렉토리 소유자를 확인한다", root)
		return f
	}
	f.Status, f.Detail = StatusOK, "위키 루트에 쓸 수 있다"
	return f
}

// loadConfig는 설정 파일을 읽는다. 파일이 없으면 위키가 아니다.
func loadConfig(root string) (config.Config, bool, Finding) {
	f := Finding{ID: "wiki.config"}
	path := filepath.Join(root, config.ConfigFileName)
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		f.Status, f.Detail = StatusSkip, "engram.yaml 이 없어 위키가 아니다. 환경 점검만 진행한다"
		return config.Config{}, false, f
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		f.Status = StatusFail
		f.Detail = fmt.Sprintf("engram.yaml 을 파싱할 수 없다: %v", err)
		f.Fix = "engram.yaml 의 YAML 문법을 고친다"
		return config.Config{}, true, f
	}
	f.Status, f.Detail = StatusOK, "engram.yaml 을 읽었다"
	return cfg, true, f
}

// skipFinding은 위키가 아닐 때의 항목을 만든다.
func skipFinding(id string) Finding {
	return Finding{ID: id, Status: StatusSkip, Detail: "engram.yaml 이 없어 위키가 아니다"}
}

// checkUnknownKeys는 설정의 알 수 없는 키를 본다. 오타를 여기서 잡는다.
func checkUnknownKeys(cfg config.Config, isWiki bool) Finding {
	if !isWiki {
		return skipFinding("wiki.config-unknown-keys")
	}
	f := Finding{ID: "wiki.config-unknown-keys"}
	if len(cfg.UnknownKeys) > 0 {
		f.Status = StatusWarn
		f.Detail = fmt.Sprintf("알 수 없는 키: %s", strings.Join(cfg.UnknownKeys, ", "))
		f.Fix = "이 키들을 지우거나 맞는 이름으로 고친다. 지원 키는 engram config list --origin 에서 본다"
		return f
	}
	f.Status, f.Detail = StatusOK, "알 수 없는 키가 없다"
	return f
}

// checkMinWikilinks는 게이트가 꺼져 있는지를 본다. ADR 0009 가
// min_wikilinks 0 을 경고로 보고하라고 정했다.
func checkMinWikilinks(cfg config.Config, isWiki bool) Finding {
	if !isWiki {
		return skipFinding("wiki.min-wikilinks")
	}
	f := Finding{ID: "wiki.min-wikilinks"}
	if cfg.Thresholds.MinWikilinks == 0 {
		f.Status = StatusWarn
		f.Detail = "min_wikilinks 가 0 이라 승급 게이트가 꺼져 있다"
		f.Fix = "engram.yaml 에 min_wikilinks: 2 를 지정한다"
		return f
	}
	f.Status, f.Detail = StatusOK, fmt.Sprintf("min_wikilinks %d", cfg.Thresholds.MinWikilinks)
	return f
}

// checkPageDirs는 page_dirs 가 실제로 존재하는지 본다.
func checkPageDirs(root string, cfg config.Config, isWiki bool) Finding {
	if !isWiki {
		return skipFinding("wiki.page-dirs")
	}
	f := Finding{ID: "wiki.page-dirs"}
	var missing []string
	for _, d := range cfg.PageDirs {
		if _, err := os.Stat(filepath.Join(root, d)); err != nil {
			missing = append(missing, d)
		}
	}
	if len(missing) > 0 {
		f.Status = StatusFail
		f.Detail = fmt.Sprintf("없는 디렉토리: %s", strings.Join(missing, ", "))
		f.Fix = fmt.Sprintf("mkdir %s", strings.Join(missing, " "))
		return f
	}
	f.Status, f.Detail = StatusOK, fmt.Sprintf("page_dirs %d개가 모두 있다", len(cfg.PageDirs))
	return f
}

// checkRootFiles는 root_files 가 실제로 존재하는지 본다.
func checkRootFiles(root string, cfg config.Config, isWiki bool) Finding {
	if !isWiki {
		return skipFinding("wiki.root-files")
	}
	f := Finding{ID: "wiki.root-files"}
	var missing []string
	for _, name := range cfg.RootFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		f.Status = StatusFail
		f.Detail = fmt.Sprintf("없는 루트 파일: %s", strings.Join(missing, ", "))
		f.Fix = fmt.Sprintf("%s 파일을 위키 루트에 만든다", strings.Join(missing, ", "))
		return f
	}
	f.Status, f.Detail = StatusOK, fmt.Sprintf("root_files %d개가 모두 있다", len(cfg.RootFiles))
	return f
}

// checkEngramGitignore는 .engram 캐시 디렉토리가 gitignore 되어 있는지 본다.
// 위키가 git 저장소일 때만 검사한다.
func checkEngramGitignore(root string, hasGit, isWiki bool) Finding {
	if !isWiki {
		return skipFinding("wiki.engram-gitignore")
	}
	f := Finding{ID: "wiki.engram-gitignore"}
	if !hasGit {
		f.Status, f.Detail = StatusSkip, "git 이 없어 확인할 수 없다"
		return f
	}
	if !isGitRepo(root) {
		f.Status, f.Detail = StatusSkip, "위키가 git 저장소가 아니다"
		return f
	}
	// 캐시 안 파일 경로로 묻는다. ".engram/" 형태의 패턴은 디렉토리에만
	// 걸리는데 .engram 이 아직 없으면 디렉토리로서 매칭되지 않기 때문이다.
	err := exec.Command("git", "-C", root, "check-ignore", "-q", filepath.FromSlash(".engram/cache")).Run()
	if err == nil {
		f.Status, f.Detail = StatusOK, ".engram 이 gitignore 된다"
		return f
	}
	f.Status = StatusWarn
	f.Detail = ".engram 캐시 디렉토리가 gitignore 되지 않았다"
	f.Fix = "위키 루트 .gitignore 에 .engram/ 줄을 추가한다"
	return f
}
