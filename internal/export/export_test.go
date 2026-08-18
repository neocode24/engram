package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
)

// contextDoc는 context 문서 하나의 원문을 만든다. sensitivity 를 빈
// 문자열로 주면 그 키를 넣지 않는다.
func contextDoc(title, sensitivity, body string) string {
	fm := "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n"
	if sensitivity != "" {
		fm += "sensitivity: " + sensitivity + "\n"
	}
	fm += "indexable: true\ncreated: 2026-01-01\nupdated: 2026-08-01\n---\n\n"
	return fm + "# " + title + "\n\n" + body + "\n"
}

// wikiFiles는 시험용 위키다. team 프리셋은 sensitivity 속성을 켜고
// personal 은 끈다.
func wikiFiles(preset string) map[string]string {
	return map[string]string{
		"engram.yaml": "preset: " + preset + "\n",
		"index.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n---\n\n" +
			"# 위키 색인\n\n[[hub]] 부터 봅니다\n",
		"context/hub.md": contextDoc("허브 문서", "internal",
			"[[peer]] 로 이어집니다. 제외 대상은 [[secret]] 입니다.\n사내명은 아크미인프라입니다.\n"),
		"context/peer.md":      contextDoc("이웃 문서", "public-reference", "[[hub]] 를 가리킵니다.\n"),
		"context/secret.md":    contextDoc("로컬 전용", "private-local-only", "[[hub]] 를 가리킵니다.\n"),
		"context/limited.md":   contextDoc("제한 공개", "restricted", "[[hub]] 를 가리킵니다.\n"),
		"context/아크미인프라-현황.md": contextDoc("사내 시스템", "internal", "[[hub]] 를 가리킵니다.\n"),
		"archive/old.md":       contextDoc("보관 문서", "internal", "[[hub]] 를 가리킵니다.\n"),
		"inbox/2026-08-01-rough.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\ncreated: 2026-08-01\n---\n\n# 러프 메모\n\n[[hub]] 입니다.\n",
		"sources/2026-08-01-src.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\ncreated: 2026-08-01\n---\n\n# 원본 요약\n\n원본입니다.\n",
	}
}

// makeWiki는 임시 디렉토리에 시험용 위키를 만든다.
func makeWiki(t *testing.T, preset string) (string, config.Config) {
	t.Helper()
	root := t.TempDir()
	for name, content := range wikiFiles(preset) {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("설정을 읽을 수 없습니다: %v", err)
	}
	return root, cfg
}

// relSet은 번들 경로 집합을 만든다.
func relSet(res Result) map[string]string {
	out := map[string]string{}
	for _, f := range res.Files {
		out[f.Rel] = f.Content
	}
	return out
}

func TestPlanFollowsExposureRules(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	got := relSet(res)
	for _, want := range []string{"index.md", "context/peer.md"} {
		if _, ok := got[want]; !ok {
			t.Errorf("%s 가 번들에 없습니다", want)
		}
	}
	for _, deny := range []string{
		"context/secret.md",         // private-local-only
		"context/limited.md",        // restricted
		"context/hub.md",            // internal. 기본 제외 (ADR 0063)
		"archive/old.md",            // 기본 제외
		"inbox/2026-08-01-rough.md", // 미검수
		"sources/2026-08-01-src.md", // 원본 보존 계층
	} {
		if _, ok := got[deny]; ok {
			t.Errorf("%s 가 번들에 들었습니다", deny)
		}
	}
	if res.Exposure.ExcludedSensitive != 2 {
		t.Errorf("민감도 제외 = %d, 기대 2", res.Exposure.ExcludedSensitive)
	}
	// hub 와 아크미인프라-현황 둘이다. archive/old.md 는 위치로 먼저 걸린다.
	if res.Exposure.ExcludedInternal != 2 {
		t.Errorf("internal 제외 = %d, 기대 2", res.Exposure.ExcludedInternal)
	}
	if res.Exposure.IncludedInternal != 0 {
		t.Errorf("internal 포함 = %d, 기대 0", res.Exposure.IncludedInternal)
	}
	if res.Exposure.ExcludedInbox != 1 || res.Exposure.ExcludedSources != 1 {
		t.Errorf("inbox/sources 제외 = %d/%d, 기대 1/1",
			res.Exposure.ExcludedInbox, res.Exposure.ExcludedSources)
	}
}

func TestPlanIncludeInternal(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{IncludeInternal: true})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	got := relSet(res)
	if _, ok := got["context/hub.md"]; !ok {
		t.Error("--include-internal 인데 internal 문서가 없습니다")
	}
	if _, ok := got["context/secret.md"]; ok {
		t.Error("--include-internal 이 private-local-only 제외까지 뚫었습니다")
	}
	if res.Exposure.ExcludedInternal != 0 {
		t.Errorf("internal 제외 = %d, 기대 0", res.Exposure.ExcludedInternal)
	}
	if res.Exposure.IncludedInternal != 2 {
		t.Errorf("internal 포함 = %d, 기대 2", res.Exposure.IncludedInternal)
	}
}

func TestPlanIncludeArchive(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{IncludeArchive: true, IncludeInternal: true})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	if _, ok := relSet(res)["archive/old.md"]; !ok {
		t.Error("--include-archive 인데 archive 문서가 없습니다")
	}
}

func TestPlanSensitivityAxisOff(t *testing.T) {
	// personal 프리셋은 sensitivity 속성이 꺼져 있으므로 거를 값이 없다.
	root, cfg := makeWiki(t, "personal")
	res, err := Plan(root, cfg, Options{})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	if _, ok := relSet(res)["context/secret.md"]; !ok {
		t.Error("축이 꺼진 위키인데 민감도로 걸렀습니다")
	}
	if res.Exposure.SensitivityOn {
		t.Error("personal 프리셋에서 sensitivity 속성이 켜졌다고 보고했습니다")
	}
}

func TestPlanNarrowsBySlug(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{Slugs: []string{"peer"}})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	if len(res.Files) != 1 || res.Files[0].Rel != "context/peer.md" {
		t.Fatalf("번들 = %v, 기대 context/peer.md 하나", relSet(res))
	}
	if res.ExcludedByFilter == 0 {
		t.Error("슬러그로 좁혔는데 걸러진 수가 0입니다")
	}
	// peer 는 hub 를 가리키는데 hub 가 번들에 없다. 링크를 따라가지 않는다.
	if res.DanglingLinks != 1 {
		t.Errorf("번들 밖 링크 = %d, 기대 1", res.DanglingLinks)
	}
}

func TestPlanSlugCannotPierceSensitivity(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	_, err := Plan(root, cfg, Options{Slugs: []string{"secret"}})
	if err == nil {
		t.Fatal("민감도로 제외된 문서를 슬러그로 뚫었습니다")
	}
	if !strings.Contains(err.Error(), "민감도") {
		t.Errorf("사유를 알리지 않았습니다: %v", err)
	}
}

func TestPlanSlugRejectsInbox(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	_, err := Plan(root, cfg, Options{Slugs: []string{"2026-08-01-rough"}})
	if err == nil {
		t.Fatal("inbox 문서를 슬러그로 뚫었습니다")
	}
}

func TestPlanSlugUnknown(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	_, err := Plan(root, cfg, Options{Slugs: []string{"없는문서"}})
	if err == nil || !strings.Contains(err.Error(), "그런 문서가 없습니다") {
		t.Fatalf("없는 슬러그 오류 = %v", err)
	}
}

func TestPlanReplacesBodyAndFilename(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{
		IncludeInternal: true,
		Rules:           []Rule{{From: "아크미인프라", To: "사내 인프라"}},
	})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	got := relSet(res)
	if _, ok := got["context/사내 인프라-현황.md"]; !ok {
		t.Errorf("파일명이 치환되지 않았습니다: %v", filePaths(res))
	}
	for rel, content := range got {
		if strings.Contains(content, "아크미인프라") {
			t.Errorf("%s 본문에 원문이 남았습니다", rel)
		}
	}
	if res.Replaced() != 2 {
		t.Errorf("치환 건수 = %d, 기대 2 (본문 1, 파일명 1)", res.Replaced())
	}
	if len(res.UnusedRules()) != 0 {
		t.Errorf("걸린 규칙을 미사용으로 셌습니다: %v", res.UnusedRules())
	}
}

func TestPlanReportsUnusedRules(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	res, err := Plan(root, cfg, Options{
		Rules: []Rule{{From: "없는말", To: "X"}},
	})
	if err != nil {
		t.Fatalf("계획을 만들 수 없습니다: %v", err)
	}
	unused := res.UnusedRules()
	if len(unused) != 1 || unused[0].From != "없는말" {
		t.Errorf("미사용 규칙 = %v, 기대 없는말 하나", unused)
	}
}

func TestPlanCollisionAfterReplacement(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	// hub 와 peer 를 같은 이름으로 만드는 규칙이다.
	_, err := Plan(root, cfg, Options{
		IncludeInternal: true,
		Rules:           []Rule{{From: "peer", To: "hub"}},
	})
	if err == nil || !strings.Contains(err.Error(), "같은 경로") {
		t.Fatalf("파일명 충돌 오류 = %v", err)
	}
}

func TestPlanReplacementEmptiesFilename(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	_, err := Plan(root, cfg, Options{
		Rules: []Rule{{From: "peer.md", To: ""}},
	})
	if err == nil || !strings.Contains(err.Error(), "파일명이 비었습니다") {
		t.Fatalf("빈 파일명 오류 = %v", err)
	}
}

func TestPlanIsDeterministicRegardlessOfSlugOrder(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	a, err := Plan(root, cfg, Options{IncludeInternal: true, Slugs: []string{"peer", "hub"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Plan(root, cfg, Options{IncludeInternal: true, Slugs: []string{"hub", "peer"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(filePaths(a), ",") != strings.Join(filePaths(b), ",") {
		t.Errorf("인자 순서가 번들 순서를 바꿨습니다: %v vs %v", filePaths(a), filePaths(b))
	}
}

// filePaths는 시험용 경로 목록이다.
func filePaths(res Result) []string {
	out := make([]string, 0, len(res.Files))
	for _, f := range res.Files {
		out = append(out, f.Rel)
	}
	return out
}

func TestApplyDoesNotChain(t *testing.T) {
	// A 를 B 로, B 를 C 로 바꾸는 규칙이 함께 있어도 A 가 C 가 되면 안 된다.
	rules := sortRules([]Rule{{From: "가", To: "나"}, {From: "나", To: "다"}})
	got, counts := apply("가나", rules)
	if got != "나다" {
		t.Errorf("치환 결과 = %q, 기대 %q", got, "나다")
	}
	if counts[0] != 1 || counts[1] != 1 {
		t.Errorf("규칙별 건수 = %v, 기대 각 1", counts)
	}
}

func TestApplyPrefersLongestRule(t *testing.T) {
	rules := sortRules([]Rule{{From: "사내", To: "X"}, {From: "사내 제품", To: "Y"}})
	got, _ := apply("사내 제품", rules)
	if got != "Y" {
		t.Errorf("치환 결과 = %q, 기대 Y (긴 규칙 우선)", got)
	}
}

func TestParseReplacements(t *testing.T) {
	rules, err := ParseReplacements("# 주석\n\n원문==>대체어\n지울말==>\n")
	if err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("규칙 수 = %d, 기대 2", len(rules))
	}
	if rules[0] != (Rule{From: "원문", To: "대체어"}) {
		t.Errorf("첫 규칙 = %+v", rules[0])
	}
	if rules[1] != (Rule{From: "지울말", To: ""}) {
		t.Errorf("둘째 규칙 = %+v", rules[1])
	}
}

func TestParseReplacementsRejectsBadLines(t *testing.T) {
	cases := map[string]string{
		"구분자 없음": "원문 대체어\n",
		"원문 비었음": "==>대체어\n",
		"원문 중복":  "원문==>하나\n원문==>둘\n",
	}
	for name, src := range cases {
		if _, err := ParseReplacements(src); err == nil {
			t.Errorf("%s: 오류를 내지 않았습니다", name)
		}
	}
}
