// Package glossary는 위키가 소유한 용어 사전을 읽어 전사에 적용한다.
//
// **사전은 위키의 것이고 이 패키지는 읽기만 한다**(ADR 0079). 사람이
// 오탈자를 고치면 그 교정이 사전에 쌓이고 git 이 추적하며 다음 전사가
// 개선된다. 그 되먹임이 여정 2의 핵심이며, 사전이 도구 안에 있으면
// 끊긴다. 위키마다 용어가 달라 도구가 들고 다닐 수도 없다.
//
// # 인식 유도는 하지 못한다
//
// upstream 은 사전을 두 자리에 쓴다. 전사 전에 whisper 의 initial_prompt
// 로 넣어 인식을 유도하고, 전사 후에 자동 교정 항목을 치환한다.
//
// **앞의 것을 여기서는 못 한다.** sherpa-onnx 의 whisper 설정에 프롬프트
// 자리가 없다. C API 의 SherpaOnnxOfflineWhisperModelConfig 에 encoder,
// decoder, language, task, tail_paddings, 타임스탬프 플래그뿐이다.
// hotwords 는 있으나 transducer 계열의 빔 탐색을 건드리는 것이라
// whisper 의 greedy 디코딩에 닿지 않는다.
//
// 그래서 이 패키지는 후처리 치환만 한다. 그 절반은 오히려 결정론적이라
// 되먹임 고리에는 더 낫다. 같은 사전에 같은 전사를 넣으면 늘 같은
// 결과가 나온다.
package glossary

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 사전 파일 이름 후보다. 앞의 것을 먼저 본다.
//
// upstream 이름을 함께 받는 이유는 그 파일이 이미 존재하고 항목이
// 여든아홉이기 때문이다. 옮겨 오는 사람이 이름을 고치지 않아도 된다.
var fileNames = []string{
	"terminology.md",
	"terminology-normalization.md",
}

// metaDir는 사전이 사는 디렉토리다. 위키 루트 기준이다.
const metaDir = "meta"

// ErrNotFound는 위키에 사전이 없을 때의 오류다. 없는 것은 정상이므로
// 호출자가 이것을 보고 조용히 넘어갈 수 있다.
var ErrNotFound = errors.New("용어 사전 없음")

// Rule은 치환 하나다. Variant 를 Canonical 로 바꾼다.
type Rule struct {
	Variant   string
	Canonical string
}

// Glossary는 읽어 들인 사전이다.
type Glossary struct {
	// Path는 읽은 파일 경로다. 사용자에게 어느 사전을 썼는지 알린다.
	Path string
	// Rules는 치환 규칙이다. 긴 변형이 앞에 온다.
	Rules []Rule
	// Reviewed는 자동 교정 대상이 아닌 항목 수다. 사전에 있으나
	// 쓰지 않은 것이 얼마나 되는지 알린다.
	Reviewed int
}

// Find는 위키에서 사전 경로를 찾는다.
func Find(wikiRoot string) (string, error) {
	for _, name := range fileNames {
		p := filepath.Join(wikiRoot, metaDir, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	return "", ErrNotFound
}

// Load는 위키의 사전을 읽는다.
func Load(wikiRoot string) (*Glossary, error) {
	p, err := Find(wikiRoot)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	g := Parse(string(b))
	g.Path = p
	return g, nil
}

// Parse는 사전 본문에서 치환 규칙을 뽑는다.
//
// 형식은 마크다운 표이고 칸 넷이다. 첫째가 정규형, 둘째가 변형 목록,
// 셋째가 자동 교정 여부, 넷째가 설명이다. 칸 이름이 아니라 위치로
// 읽는다. upstream 파일이 절 여럿에 표를 나눠 두고 머리 행이 여러 번
// 나오기 때문이다.
//
// **자동 교정은 셋째 칸이 yes 로 시작하는 것만 한다.** review 와
// conditional 은 사람이 보라는 표시이므로 건드리지 않는다. "yes when
// 무엇무엇" 처럼 조건이 붙은 것은 yes 로 친다. 조건 판단은 도구가
// 할 수 없고, 그 항목을 넣은 사람이 대체로 맞다고 본 것이다.
func Parse(text string) *Glossary {
	g := &Glossary{}
	seen := map[string]bool{}
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitRow(line)
		if len(cells) < 3 {
			continue
		}
		canonical := strings.Trim(cells[0], " `")
		auto := strings.ToLower(strings.TrimSpace(cells[2]))
		// 구분 행과 머리 행을 거른다. 칸 이름이 아니라 모양으로 가린다.
		// 머리 행은 바로 다음 줄이 구분 행이라는 사실로 가리며, 그래야
		// 칸 이름이 어느 언어든 통한다. 이름으로 가리면 한국어로 쓴
		// 사전의 머리 행이 항목으로 세어진다.
		if canonical == "" || isSeparatorRow(line) {
			continue
		}
		if i+1 < len(lines) && isSeparatorRow(strings.TrimSpace(lines[i+1])) {
			continue
		}
		if !strings.HasPrefix(auto, "yes") {
			if auto != "" {
				g.Reviewed++
			}
			continue
		}
		for _, v := range splitVariants(cells[1]) {
			if v == "" || v == canonical || len([]rune(v)) < 2 {
				continue
			}
			key := v + "\x00" + canonical
			if seen[key] {
				continue
			}
			seen[key] = true
			g.Rules = append(g.Rules, Rule{Variant: v, Canonical: canonical})
		}
	}
	// 긴 변형을 먼저 친다. 짧은 것이 긴 것의 일부이면 순서가 뒤집혔을
	// 때 긴 변형이 영영 안 잡힌다. 길이가 같으면 사전 순으로 고정해
	// 같은 사전이 늘 같은 결과를 내게 한다.
	sort.SliceStable(g.Rules, func(i, j int) bool {
		a, b := g.Rules[i].Variant, g.Rules[j].Variant
		if len([]rune(a)) != len([]rune(b)) {
			return len([]rune(a)) > len([]rune(b))
		}
		return a < b
	})
	return g
}

// isSeparatorRow는 마크다운 표의 구분 행인지 본다. 칸이 대시와 콜론과
// 공백으로만 이루어진 행이다.
func isSeparatorRow(line string) bool {
	if !strings.HasPrefix(line, "|") {
		return false
	}
	body := strings.Trim(line, "|")
	if strings.TrimSpace(body) == "" {
		return false
	}
	return strings.IndexFunc(body, func(r rune) bool {
		return r != '-' && r != ':' && r != ' ' && r != '|'
	}) < 0
}

// splitRow는 표 한 줄을 칸으로 나눈다.
func splitRow(line string) []string {
	trimmed := strings.Trim(line, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// splitVariants는 변형 칸을 낱개로 나눈다. 쉼표와 한중일 쉼표 둘 다
// 구분자로 받는다. "무엇 when 조건" 형태는 조건 서술이므로 앞만 쓴다.
func splitVariants(cell string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(cell, func(r rune) bool {
		return r == ',' || r == '、'
	}) {
		v := strings.Trim(part, " `")
		if i := strings.Index(strings.ToLower(v), " when "); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		out = append(out, strings.Trim(v, " `"))
	}
	return out
}

// Applied는 치환 한 건의 결과다.
type Applied struct {
	Rule  Rule
	Count int
}

// Apply는 텍스트에 치환을 적용하고 무엇을 몇 번 바꿨는지 함께 낸다.
//
// 무엇을 바꿨는지 반드시 알린다. 조용히 바꾸면 사람이 전사를 검수할 때
// 도구가 손댄 자리를 모른다. 사전이 틀렸을 때 그것을 발견할 길이 그
// 목록뿐이다.
func (g *Glossary) Apply(text string) (string, []Applied) {
	if g == nil || len(g.Rules) == 0 {
		return text, nil
	}
	var applied []Applied
	for _, r := range g.Rules {
		n := strings.Count(text, r.Variant)
		if n == 0 {
			continue
		}
		text = strings.ReplaceAll(text, r.Variant, r.Canonical)
		applied = append(applied, Applied{Rule: r, Count: n})
	}
	return text, applied
}

// TotalReplacements는 치환 횟수의 합이다.
func TotalReplacements(applied []Applied) int {
	n := 0
	for _, a := range applied {
		n += a.Count
	}
	return n
}
