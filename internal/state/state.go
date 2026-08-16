// Package state는 사용자 판단이 담긴 영구 상태를 위키 루트의
// engram-state.yaml 에 저장한다. 잃었을 때 사용자 판단이 사라지는 것은
// 이 파일에 두고, 원문에서 재생성되는 것은 .engram/ 에 둔다. ADR 0028.
package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

// StateFileName은 영구 상태 파일 이름이다. 위키 루트에 두고 git이 추적한다.
const StateFileName = "engram-state.yaml"

// State는 위키의 영구 상태다. 현재 키는 bridge의 기각 쌍 하나다.
// 메모리 안의 상태는 언제나 정규형이다. 쌍은 슬러그 오름차순,
// 목록은 쌍 기준 오름차순이며 중복이 없다.
type State struct {
	BridgeRejections [][2]string
}

// wireState는 YAML 변환 전용 모양이다. [2]string으로 바로 읽으면 원소 수가
// 다른 쌍에서 파서가 죽으므로 슬라이스로 받아 여기서 검사한다.
type wireState struct {
	BridgeRejections [][]string `yaml:"bridge_rejections"`
}

// Load는 위키 루트의 상태 파일을 읽는다. 파일이 없으면 빈 상태와 nil 에러를
// 낸다. 기각은 아직 없다는 뜻이지 판단을 잃었다는 뜻이 아니다.
// 파싱 실패는 에러다. 이 파일은 사용자 판단의 진실원이므로 조용히 빈 것으로
// 취급하면 판단을 버리는 것이다.
func Load(wikiRoot string) (State, error) {
	path := filepath.Join(wikiRoot, StateFileName)
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("%s 파일을 읽을 수 없습니다: %w", path, err)
	}
	var w wireState
	// 모르는 키가 있으면 에러로 막는다. 키를 모르는 채로 덮어쓰면 그 키가
	// 담은 판단이 사라진다.
	if err := yaml.UnmarshalWithOptions(raw, &w, yaml.DisallowUnknownField()); err != nil {
		return State{}, fmt.Errorf("%s 파일을 해석할 수 없습니다: %w\n문법을 고쳐 주세요. 파일을 지우면 기각한 판단이 사라집니다", path, err)
	}
	var s State
	for i, p := range w.BridgeRejections {
		if len(p) != 2 {
			return State{}, fmt.Errorf("%s 파일의 bridge_rejections %d번째 쌍이 슬러그 %d개를 가집니다. 슬러그 두 개여야 합니다", path, i+1, len(p))
		}
		s.BridgeRejections = append(s.BridgeRejections, [2]string{p[0], p[1]})
	}
	return canonicalize(s), nil
}

// Save는 상태를 위키 루트에 쓴다. 정규형으로 쓰므로 같은 상태를 두 번
// 저장하면 바이트까지 같다. 파일이 없을 때 만드는 것은 호출자가 판단한다.
func (s State) Save(wikiRoot string) error {
	pairs := canonicalize(s).BridgeRejections
	var b strings.Builder
	if len(pairs) == 0 {
		b.WriteString("bridge_rejections: []\n")
	} else {
		b.WriteString("bridge_rejections:\n")
		for _, p := range pairs {
			b.WriteString("  - [" + scalar(p[0]) + ", " + scalar(p[1]) + "]\n")
		}
	}
	return os.WriteFile(filepath.Join(wikiRoot, StateFileName), []byte(b.String()), 0o644)
}

// IsRejected는 쌍이 기각됐는지를 반환한다. 순서는 무관하다.
func (s State) IsRejected(a, b string) bool {
	a, b = sortPair(a, b)
	for _, p := range s.BridgeRejections {
		if p[0] == a && p[1] == b {
			return true
		}
	}
	return false
}

// Reject는 쌍을 기각 목록에 추가한다. 새로 추가됐으면 true를 낸다.
// 추가 뒤 목록을 다시 정규화해 메모리 상태의 정렬 불변식을 유지한다.
func (s *State) Reject(a, b string) bool {
	if s.IsRejected(a, b) {
		return false
	}
	a, b = sortPair(a, b)
	s.BridgeRejections = append(s.BridgeRejections, [2]string{a, b})
	*s = canonicalize(*s)
	return true
}

// Unreject는 기각을 되돌린다. 실제로 지웠으면 true를 낸다.
func (s *State) Unreject(a, b string) bool {
	a, b = sortPair(a, b)
	for i, p := range s.BridgeRejections {
		if p[0] == a && p[1] == b {
			s.BridgeRejections = append(s.BridgeRejections[:i], s.BridgeRejections[i+1:]...)
			return true
		}
	}
	return false
}

// canonicalize는 상태를 정규형으로 낸다. 쌍 안 정렬, 목록 정렬, 중복 제거.
func canonicalize(s State) State {
	var out [][2]string
	for _, p := range s.BridgeRejections {
		a, b := sortPair(p[0], p[1])
		dup := false
		for _, q := range out {
			if q[0] == a && q[1] == b {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, [2]string{a, b})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	s.BridgeRejections = out
	return s
}

// sortPair는 쌍의 슬러그 둘을 오름차순으로 놓는다.
func sortPair(a, b string) (string, string) {
	if b < a {
		return b, a
	}
	return a, b
}

// scalar는 문자열 하나를 YAML 스칼라로 낸다. bool이나 숫자로 읽히는
// 슬러그(true, 123)는 파서가 인용해 준 결과를 그대로 쓴다.
func scalar(v string) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return v
	}
	return strings.TrimSpace(string(b))
}
