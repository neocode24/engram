// Package pin은 engram 의 Pin 게이트와 상태 생성을 upstream 의 pin 코드와 견주는 하네스다.
//
// 같은 견본 위키와 같은 요청 묶음을 upstream 파이썬(scripts/pin/apply.py)과 engram Go
// (internal/pin)에 돌려 원장 파일을 바이트 단위로, 거절 코드와 메시지를 글자 단위로,
// 상태를 값으로 견준다. ENGRAM_UPSTREAM 이 없으면 건너뛴다. CI 는 항상 건너뛴다(ADR 0029).
//
// 견본은 harness/pin/fixtures 에 있다. upstream 과 일부러 달리한 점은 아래 두 곳에만 적는다.
//   - 견본에서 뺀다: 프로젝트가 0개인 원장, 화면에 쓰지 않는 표현 아홉이 든 제목,
//     옮긴 문서에서 인용문이 사라진 경로 이동(opMovePath 는 인용문을 다시 대조하지 않는다),
//     inbox 와 sources 문서의 링크(Go 는 context 문서에만 단다),
//     상위 항목을 비우는 set_parent(upstream 로그 문장이 파이썬 str(None)인 "상위 None"이다.
//     Go 는 "상위 없음"으로 쓴다).
//   - 정규화한다: normalizeState 가 적은 칸(주소). 그 밖의 차이는 실패다.
package pin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	enginepin "github.com/neocode24/engram/internal/pin"
)

const fixtureDir = "fixtures"

func expandTilde(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// upstreamPin은 upstream 의 scripts/pin 경로다. 없으면 건너뛴다.
func upstreamPin(t *testing.T) string {
	t.Helper()
	root := expandTilde(os.Getenv("ENGRAM_UPSTREAM"))
	if root == "" {
		t.Skip("ENGRAM_UPSTREAM 이 없어 비교를 건너뛴다. ENGRAM_UPSTREAM=<llm-wiki 경로> 로 설정하면 돈다. CI 는 이 변수가 없으므로 항상 건너뛴다(ADR 0029).")
	}
	dir := filepath.Join(root, "scripts", "pin")
	if _, err := os.Stat(filepath.Join(dir, "apply.py")); err != nil {
		t.Skipf("upstream scripts/pin 이 없어 건너뛴다: %v", err)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skipf("python3 가 없어 건너뛴다: %v", err)
	}
	return dir
}

func python(t *testing.T, args ...string) []byte {
	t.Helper()
	driver, err := filepath.Abs("driver.py")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", append([]string{driver}, args...)...)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("upstream 드라이버 실패: %v\nstderr:\n%s", err, stderr.String())
	}
	return stdout.Bytes()
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// newWiki는 견본 위키를 새 임시 디렉터리에 복사하고 견본 설정을 pin/ 에 둔다.
func newWiki(t *testing.T) string {
	t.Helper()
	wiki := t.TempDir()
	copyTree(t, filepath.Join(fixtureDir, "wiki"), wiki)
	if err := os.MkdirAll(filepath.Join(wiki, "pin"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(fixtureDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "pin", "config.json"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	return wiki
}

type batch struct {
	Now      string `json:"now"`
	Requests []any  `json:"requests"`
}

func loadBatches(t *testing.T) (names []string, batches map[string]batch) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(fixtureDir, "batches", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("요청 묶음이 없습니다: %v", err)
	}
	sort.Strings(files)
	batches = map[string]batch{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b batch
		if err := json.Unmarshal(data, &b); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		names = append(names, name)
		batches[name] = b
	}
	return
}

// readTree는 디렉터리의 모든 파일을 상대 경로 순서로 읽는다. 잠금과 임시 파일은 뺀다.
func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if name == ".lock" || strings.HasSuffix(name, ".tmp") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func diffTrees(a, b map[string]string) []string {
	var out []string
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		av, aok := a[k]
		bv, bok := b[k]
		switch {
		case !aok:
			out = append(out, k+": upstream 에 없고 Go 에만 있다")
		case !bok:
			out = append(out, k+": Go 에 없고 upstream 에만 있다")
		case av != bv:
			out = append(out, k+": 내용이 다르다\n--- upstream\n"+av+"\n--- go\n"+bv)
		}
	}
	return out
}

// normalizeState는 견줄 수 없는 칸만 지운다. 상태의 주소가 그것이다.
//   - page_url, items[].url: upstream 은 운영 화면 주소, engram 은 /pin
//   - evidence[].url, mentions[].url: upstream 은 Obsidian 링크, engram 은 context 문서에만 /doc/ 링크
//
// 제목(title)은 지우지 않는다. 문서 제목을 읽는 규칙은 같아야 한다.
func normalizeState(v any) any {
	st, _ := v.(map[string]any)
	if st == nil {
		return v
	}
	delete(st, "page_url")
	items, _ := st["items"].([]any)
	for _, it := range items {
		io := it.(map[string]any)
		delete(io, "url")
		for _, key := range []string{"evidence", "mentions"} {
			list, _ := io[key].([]any)
			for _, e := range list {
				delete(e.(map[string]any), "url")
			}
		}
	}
	return st
}

func decode(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("JSON 을 읽지 못했다: %v\n%s", err, data)
	}
	return v
}

type pyOut struct {
	Results []struct {
		ID      string  `json:"id"`
		Result  string  `json:"result"`
		Code    *string `json:"code"`
		Message *string `json:"message"`
	} `json:"results"`
	Archived []string `json:"archived"`
	Written  []string `json:"written"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TestPinParity는 요청 묶음 다섯 회차를 같은 순서로 두 구현에 돌려 매 회차마다 견준다.
// 견주는 것은 pin/ 의 모든 파일(바이트), 요청마다의 결과 코드와 메시지, 보관된 항목, 상태다.
func TestPinParity(t *testing.T) {
	up := upstreamPin(t)
	names, batches := loadBatches(t)

	pyWiki, goWiki := newWiki(t), newWiki(t)
	pyCreated := python(t, "init", up, pyWiki)
	goCreated, err := enginepin.Init(goWiki)
	if err != nil {
		t.Fatal(err)
	}
	var pyList []string
	if err := json.Unmarshal(pyCreated, &pyList); err != nil {
		t.Fatal(err)
	}
	sort.Strings(goCreated)
	if !reflect.DeepEqual(pyList, goCreated) {
		t.Errorf("init 이 만든 파일이 다르다\n  upstream: %v\n  go:       %v", pyList, goCreated)
	}
	if d := diffTrees(readTree(t, filepath.Join(pyWiki, "pin")), readTree(t, filepath.Join(goWiki, "pin"))); len(d) > 0 {
		t.Fatalf("init 직후 pin/ 이 다르다:\n%s", strings.Join(d, "\n"))
	}

	reqDir := t.TempDir()
	for _, name := range names {
		b := batches[name]
		t.Run(name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, b.Now)
			if err != nil {
				t.Fatal(err)
			}
			reqFile := filepath.Join(reqDir, name+".json")
			data, _ := json.Marshal(map[string]any{"requests": b.Requests})
			if err := os.WriteFile(reqFile, data, 0o644); err != nil {
				t.Fatal(err)
			}
			stateFile := filepath.Join(reqDir, name+"-state.json")
			var py pyOut
			if err := json.Unmarshal(python(t, "apply", up, pyWiki, reqFile, b.Now, stateFile), &py); err != nil {
				t.Fatal(err)
			}
			sum, err := enginepin.Apply(goWiki, b.Requests, now)
			if err != nil {
				t.Fatalf("Go 게이트가 묶음 전체를 거절했다: %v", err)
			}

			codes := map[string]int{}
			for _, r := range sum.Results {
				codes[r.Result+"/"+r.Code]++
			}
			t.Logf("요청 %d건, 결과 %v, 보관 %v, 쓴 파일 %d개", len(sum.Results), codes, sum.Archived, len(sum.Written))
			if os.Getenv("PIN_VERBOSE") != "" {
				for _, r := range sum.Results {
					t.Logf("  %s %s %s %s", r.ID, r.Result, r.Code, r.Message)
				}
			}
			if len(py.Results) != len(sum.Results) {
				t.Fatalf("결과 수가 다르다: upstream %d, go %d", len(py.Results), len(sum.Results))
			}
			for i, pr := range py.Results {
				gr := sum.Results[i]
				got := fmt.Sprintf("%s|%s|%s|%s", gr.ID, gr.Result, gr.Code, gr.Message)
				want := fmt.Sprintf("%s|%s|%s|%s", pr.ID, pr.Result, deref(pr.Code), deref(pr.Message))
				if got != want {
					t.Errorf("요청 %d(%s) 결과가 다르다\n  upstream: %s\n  go:       %s", i+1, pr.ID, want, got)
				}
			}
			var pyArch []string
			pyArch = append(pyArch, py.Archived...)
			if !reflect.DeepEqual(append([]string{}, pyArch...), append([]string{}, sum.Archived...)) && !(len(pyArch) == 0 && len(sum.Archived) == 0) {
				t.Errorf("보관된 항목이 다르다\n  upstream: %v\n  go:       %v", pyArch, sum.Archived)
			}
			if d := diffTrees(readTree(t, filepath.Join(pyWiki, "pin")), readTree(t, filepath.Join(goWiki, "pin"))); len(d) > 0 {
				t.Errorf("pin/ 파일이 다르다:\n%s", strings.Join(d, "\n"))
			}

			pyState, err := os.ReadFile(stateFile)
			if err != nil {
				t.Fatal(err)
			}
			goState, err := enginepin.State(goWiki, now)
			if err != nil {
				t.Fatalf("Go 상태 생성 실패: %v", err)
			}
			ps, gs := normalizeState(decode(t, pyState)), normalizeState(decode(t, goState))
			if !reflect.DeepEqual(ps, gs) {
				t.Errorf("상태가 다르다:\n%s", firstDiff("", ps, gs))
			}
		})
	}
}

// firstDiff는 두 JSON 값이 처음 갈리는 자리를 찾는다.
func firstDiff(path string, a, b any) string {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: 형이 다르다", path)
		}
		var keys []string
		seen := map[string]bool{}
		for k := range av {
			keys = append(keys, k)
			seen[k] = true
		}
		for k := range bv {
			if !seen[k] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			x, aok := av[k]
			y, bok := bv[k]
			switch {
			case !aok:
				out = append(out, fmt.Sprintf("%s.%s: go 에만 있다 %v", path, k, y))
			case !bok:
				out = append(out, fmt.Sprintf("%s.%s: upstream 에만 있다 %v", path, k, x))
			default:
				if d := firstDiff(path+"."+k, x, y); d != "" {
					out = append(out, d)
				}
			}
		}
		return strings.Join(out, "\n")
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return fmt.Sprintf("%s: 형이 다르다", path)
		}
		if len(av) != len(bv) {
			return fmt.Sprintf("%s: 길이가 다르다 upstream %d, go %d", path, len(av), len(bv))
		}
		var out []string
		for i := range av {
			if d := firstDiff(fmt.Sprintf("%s[%d]", path, i), av[i], bv[i]); d != "" {
				out = append(out, d)
			}
		}
		return strings.Join(out, "\n")
	}
	if !reflect.DeepEqual(a, b) {
		return fmt.Sprintf("%s: upstream %v, go %v", path, a, b)
	}
	return ""
}

// TestPinSurface는 하네스가 upstream 과 견주는 상수 표면을 본다. 요청 경로마다 허용된 연산 표,
// 항목 키, 상태값, 근거 키가 upstream 과 글자 그대로 같아야 한다. 표를 줄이면 견줄 때마다 예외를 적게 된다.
func TestPinSurface(t *testing.T) {
	up := upstreamPin(t)
	var py struct {
		OpsByVia   map[string][]string `json:"ops_by_via"`
		Statuses   []string            `json:"STATUSES"`
		Waits      []string            `json:"WAIT_REASONS"`
		Sources    []string            `json:"SOURCES"`
		Stages     []string            `json:"STAGES"`
		Bases      []string            `json:"BASES"`
		ItemKeys   []string            `json:"ITEM_KEYS"`
		EvidenceKs []string            `json:"EVIDENCE_KEYS"`
	}
	if err := json.Unmarshal(python(t, "consts", up), &py); err != nil {
		t.Fatal(err)
	}
	if got := enginepin.OpsByVia(); !reflect.DeepEqual(got, py.OpsByVia) {
		t.Errorf("요청 경로별 허용 연산이 다르다\n  upstream: %v\n  go:       %v", py.OpsByVia, got)
	}
	got := enginepin.Constants()
	for name, want := range map[string][]string{
		"STATUSES": py.Statuses, "WAIT_REASONS": py.Waits, "SOURCES": py.Sources, "STAGES": py.Stages,
		"BASES": py.Bases, "ITEM_KEYS": py.ItemKeys, "EVIDENCE_KEYS": py.EvidenceKs,
	} {
		var have []string
		for _, e := range got[name].([]any) {
			have = append(have, e.(string))
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("%s 가 다르다\n  upstream: %v\n  go:       %v", name, want, have)
		}
	}
}
