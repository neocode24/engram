package glossary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `# 용어 정규화

## 환경 용어

| Canonical | STT / spoken variants | Auto-correct? | Notes |
| --- | --- | --- | --- |
| ` + "`샌드박스`" + ` | 샌드 박스, 샌드빡스 | yes | 임시 검증 환경 |
| ` + "`운영`" + ` | 프로덕션, 프로덕숀 | yes when 배포 맥락이 분명하면 | |
| ` + "`검토대상`" + ` | 검토 대상 | review | 사람이 본다 |

## 다른 절

| Canonical | STT / spoken variants | Auto-correct? | Notes |
| --- | --- | --- | --- |
| ` + "`게이트웨이`" + ` | 게이트 웨이, 게웨 | yes | |
`

func TestParsePicksOnlyAutoCorrect(t *testing.T) {
	g := Parse(sample)
	got := map[string]string{}
	for _, r := range g.Rules {
		got[r.Variant] = r.Canonical
	}
	want := map[string]string{
		"샌드 박스": "샌드박스", "샌드빡스": "샌드박스",
		"프로덕션": "운영", "프로덕숀": "운영",
		"게이트 웨이": "게이트웨이", "게웨": "게이트웨이",
	}
	for v, c := range want {
		if got[v] != c {
			t.Errorf("%q 는 %q 로 가야 함: %q", v, c, got[v])
		}
	}
	// review 는 사람이 보라는 표시다. 건드리지 않는다.
	if _, ok := got["검토 대상"]; ok {
		t.Error("review 항목을 치환하면 안 됨")
	}
	if g.Reviewed != 1 {
		t.Errorf("검토 대상이 하나여야 함: %d", g.Reviewed)
	}
	// 절이 여럿이라 머리 행이 여러 번 나온다. 그것이 규칙이 되면 안 된다.
	for _, r := range g.Rules {
		if strings.Contains(r.Canonical, "Canonical") || strings.HasPrefix(r.Canonical, "---") {
			t.Errorf("머리 행이나 구분 행이 규칙이 됨: %+v", r)
		}
	}
}

func TestParseSortsLongestFirst(t *testing.T) {
	// 짧은 변형이 긴 변형의 일부이면 순서가 뒤집혔을 때 긴 것이
	// 영영 안 잡힌다.
	g := Parse("| `가나다` | 가나, 가나다라 | yes | |")
	if len(g.Rules) != 2 {
		t.Fatalf("규칙이 둘이어야 함: %d", len(g.Rules))
	}
	if g.Rules[0].Variant != "가나다라" {
		t.Errorf("긴 변형이 앞이어야 함: %+v", g.Rules)
	}
}

func TestApplyReportsWhatChanged(t *testing.T) {
	g := Parse(sample)
	text := "샌드 박스에서 게웨를 붙였고 샌드 박스가 또 나온다"
	got, applied := g.Apply(text)
	if strings.Contains(got, "샌드 박스") || strings.Contains(got, "게웨") {
		t.Errorf("치환이 안 됨: %s", got)
	}
	if !strings.Contains(got, "샌드박스") || !strings.Contains(got, "게이트웨이") {
		t.Errorf("정규형이 없음: %s", got)
	}
	// 무엇을 몇 번 바꿨는지 알려야 한다. 조용히 바꾸면 검수할 수 없다.
	if n := TotalReplacements(applied); n != 3 {
		t.Errorf("치환이 셋이어야 함: %d (%+v)", n, applied)
	}
}

func TestApplyOnEmptyGlossaryIsNoop(t *testing.T) {
	g := Parse("사전이 아닌 글")
	text := "그대로 남아야 한다"
	got, applied := g.Apply(text)
	if got != text || len(applied) != 0 {
		t.Errorf("건드리면 안 됨: %q %+v", got, applied)
	}
	var nilG *Glossary
	if got, _ := nilG.Apply(text); got != text {
		t.Errorf("nil 사전도 건드리면 안 됨: %q", got)
	}
}

func TestFindPrefersEngramName(t *testing.T) {
	root := t.TempDir()
	meta := filepath.Join(root, "meta")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		t.Fatal(err)
	}
	// upstream 이름만 있으면 그것을 쓴다. 옮겨 오는 사람이 파일
	// 이름을 고치지 않아도 된다.
	up := filepath.Join(meta, "terminology-normalization.md")
	if err := os.WriteFile(up, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Find(root)
	if err != nil || p != up {
		t.Fatalf("upstream 이름을 찾아야 함: %q %v", p, err)
	}
	// engram 이름이 있으면 그쪽이 먼저다.
	own := filepath.Join(meta, "terminology.md")
	if err := os.WriteFile(own, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, _ := Find(root); p != own {
		t.Errorf("engram 이름이 우선이어야 함: %q", p)
	}
}

func TestLoadMissingIsNotAnError(t *testing.T) {
	// 사전이 없는 것은 정상이다. 호출자가 조용히 넘어갈 수 있어야 한다.
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("없으면 ErrNotFound 여야 함")
	} else if !strings.Contains(err.Error(), "용어 사전 없음") {
		t.Errorf("ErrNotFound 여야 함: %v", err)
	}
}

// TestApplyDoesNotCascade는 앞 규칙이 낸 정규형을 뒤 규칙이 다시 잡지
// 않는지 본다. 실측에서 `임계을`이 `임계값`이 된 뒤 `임계` 규칙에 다시
// 걸려 `임계값값`이 됐다.
func TestApplyDoesNotCascade(t *testing.T) {
	g := Parse(`| 정규형 | 변형 | 자동 교정 | 설명 |
|---|---|---|---|
| 임계값 | 임계을, 임계 | yes | |
`)
	got, applied := g.Apply("임계을 처음 잡을 때")
	if want := "임계값 처음 잡을 때"; got != want {
		t.Errorf("%q 를 기대했으나 %q", want, got)
	}
	if n := TotalReplacements(applied); n != 1 {
		t.Errorf("교정 1건을 기대했으나 %d건", n)
	}
}

// TestApplyLongestWins는 짧은 변형이 긴 변형의 앞부분이어도 긴 쪽이
// 먼저 잡히는지 본다.
func TestApplyLongestWins(t *testing.T) {
	g := Parse(`| 정규형 | 변형 | 자동 교정 | 설명 |
|---|---|---|---|
| 헬스체크 | 스체크 | yes | |
| 검사 | 체크 | yes | |
`)
	got, _ := g.Apply("스체크가 실패했다")
	if want := "헬스체크가 실패했다"; got != want {
		t.Errorf("%q 를 기대했으나 %q", want, got)
	}
}
