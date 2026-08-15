package graph

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
	"github.com/neocode24/engram/internal/walk"
)

// writeWiki는 임시 디렉토리에 위키를 만들고 순회 결과와 설정을 반환한다.
func writeWiki(t *testing.T, files map[string]string) (string, []walk.Doc, config.Config) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	walked, err := walk.Files(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return dir, walked, cfg
}

func TestBuild(t *testing.T) {
	t.Run("본문 링크만 잡는다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\n---\n\n[[target]] 을 본문에서 가리킨다",
		})
		bl := Build(walked).Backlinks("target")
		if len(bl) != 1 || bl[0].Field != KindBody || bl[0].From != "context/a.md" {
			t.Fatalf("본문 링크가 잡혀야 함: %+v", bl)
		}
		if bl[0].Raw != "[[target]]" {
			t.Errorf("원본 값이 잘못됨: %+v", bl[0])
		}
	})

	t.Run("related 필드만 잡는다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\nrelated:\n  - \"[[target]]\"\n---\n\n본문",
		})
		bl := Build(walked).Backlinks("target")
		if len(bl) != 1 || bl[0].Field != KindRelated {
			t.Fatalf("related 링크가 잡혀야 함: %+v", bl)
		}
		if bl[0].Line != 4 {
			t.Errorf("related 항목 줄 번호가 잘못됨: %+v", bl[0])
		}
	})

	t.Run("관계 필드를 잡는다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\nderived_from:\n  - sources/2026-01-08-talk.md\n---\n\n본문",
		})
		bl := Build(walked).Backlinks("talk")
		if len(bl) != 1 || bl[0].Field != KindDerivedFrom {
			t.Fatalf("derived_from 링크가 잡혀야 함: %+v", bl)
		}
		if bl[0].Raw != "sources/2026-01-08-talk.md" {
			t.Errorf("원본 값이 잘못됨: %+v", bl[0])
		}
	})

	t.Run("세 종류가 섞여도 각각 잡는다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\nrelated:\n  - \"[[target]]\"\n---\n\n본문 [[target]] 링크",
			"context/b.md": "---\ntype: system\nderived_context:\n  - target\n---\n\n본문",
			"context/c.md": "---\ntype: system\nsource_refs:\n  - sources/note.md\n---\n\n본문",
		})
		bl := Build(walked).Backlinks("target")
		if len(bl) != 3 {
			t.Fatalf("백링크 3건이어야 함: %+v", bl)
		}
		kinds := map[Kind]bool{}
		for _, l := range bl {
			kinds[l.Field] = true
		}
		if !kinds[KindBody] || !kinds[KindRelated] || !kinds[KindDerivedContext] {
			t.Fatalf("세 종류가 모두 잡혀야 함: %+v", bl)
		}
	})

	t.Run("경로 형태 값과 슬러그 형태 값이 같은 문서를 가리킨다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"sources/2026-01-08-talk.md": "---\ntype: system\n---\n\n원본",
			"context/a.md":               "---\ntype: system\nderived_from:\n  - sources/2026-01-08-talk.md\n---\n\n본문 [[2026-01-08-talk]] 링크",
		})
		g := Build(walked)
		if !g.Has("talk") || !g.Has("sources/2026-01-08-talk.md") || !g.Has("2026-01-08-talk") {
			t.Fatal("세 형태가 같은 문서를 가리켜야 함")
		}
		if bl := g.Backlinks("talk"); len(bl) != 2 {
			t.Fatalf("날짜 접두사가 붙은 위키링크도 잡혀야 함: %+v", bl)
		}
	})

	t.Run("코드 펜스 안의 링크는 세지 않는다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\n---\n\n```\n[[target]]\n```\n인라인 `[[target]]` 도 아니다",
		})
		if bl := Build(walked).Backlinks("target"); len(bl) != 0 {
			t.Fatalf("펜스 안 링크가 세짐: %+v", bl)
		}
	})

	t.Run("결과는 출처 경로와 줄 번호 순으로 정렬된다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/b.md": "---\ntype: system\nrelated:\n  - \"[[t]]\"\n---\n\n본문 [[t]]",
			"context/a.md": "---\ntype: system\n---\n\n본문 [[t]] 첫줄\n둘째 줄 [[t]]",
		})
		g := Build(walked)
		if out := g.Outgoing("context/a.md"); out[0].Line > out[1].Line {
			t.Fatalf("같은 파일 안은 줄 번호순이어야 함: %+v", out)
		}
		bl := g.Backlinks("t")
		if bl[0].From != "context/a.md" || bl[len(bl)-1].From != "context/b.md" {
			t.Fatalf("출처 경로순이어야 함: %+v", bl)
		}
	})

	t.Run("존재하지 않는 슬러그도 조회된다", func(t *testing.T) {
		_, walked, _ := writeWiki(t, map[string]string{
			"context/a.md": "---\ntype: system\n---\n\n본문 [[없는문서]] 링크",
		})
		g := Build(walked)
		if g.Has("없는문서") {
			t.Fatal("문서가 없으면 Has는 거짓이어야 함")
		}
		if bl := g.Backlinks("없는문서"); len(bl) != 1 {
			t.Fatalf("깨진 링크도 백링크로 나와야 함: %+v", bl)
		}
	})
}

// consistencyWiki는 lint 의 고아 판정과 graph 이 같은 집합을 보는지
// 확인하는 데 쓰는 위키다. 고아, 링크된 문서, 관계 필드로 이어진 문서,
// 색인이 섞여 있다.
func consistencyWiki() map[string]string {
	return map[string]string{
		"index.md":                   "---\ntype: system\n---\n\n색인",
		"context/linked.md":          "---\ntype: system\nrelated:\n  - \"[[hub]]\"\n---\n\n본문",
		"context/hub.md":             "---\ntype: system\n---\n\n본문 [[linked]] 링크",
		"context/derived.md":         "---\ntype: system\nderived_from:\n  - sources/2026-01-08-talk.md\n---\n\n본문",
		"sources/2026-01-08-talk.md": "---\ntype: system\n---\n\n원본",
		"context/orphan.md":          "---\ntype: system\n---\n\n아무 연결이 없다",
		"inbox/2026-01-01-m.md":      "---\ntype: inbox-note\nartifact_stage: inbox\n---\n\n[[hub]] 을 가리키는 메모",
	}
}

// TestLintOrphanConsistency는 lint 의 고아 판정과 같은 문서 집합을 보는지
// 확인한다. 슬러그 정규화가 어긋나면 두 집합이 갈라진다.
func TestLintOrphanConsistency(t *testing.T) {
	root, walked, cfg := writeWiki(t, consistencyWiki())

	// lint 가 고아로 보는 문서 집합.
	lres, err := lint.Run(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	lintOrphans := map[string]bool{}
	for _, v := range lres.Violations {
		if v.Rule == "graph.orphan" {
			lintOrphans[v.Path] = true
		}
	}
	if len(lintOrphans) == 0 {
		t.Fatal("lint 가 고아를 찾지 못했다. 테스트 위키가 잘못됨")
	}

	// graph 로 계산한 고아 집합. 고아는 나가는 링크가 없고 들어오는
	// 링크도 없는 문서다. 색인(root_files)은 lint 가 판정에서 뺀다.
	g := Build(walked)
	rootFiles := map[string]bool{}
	for _, f := range cfg.RootFiles {
		rootFiles[f] = true
	}
	graphOrphans := map[string]bool{}
	for _, wd := range walked {
		if wd.Err != nil || rootFiles[wd.Rel] {
			continue
		}
		hasIncoming := false
		for _, l := range g.Backlinks(Normalize(wd.Rel)) {
			if l.From != wd.Rel {
				hasIncoming = true
			}
		}
		if len(g.Outgoing(wd.Rel)) == 0 && !hasIncoming {
			graphOrphans[wd.Rel] = true
		}
	}

	if !mapEqual(lintOrphans, graphOrphans) {
		t.Fatalf("lint 와 graph 의 고아 집합이 다름:\nlint:   %v\ngraph: %v",
			sortedKeys(lintOrphans), sortedKeys(graphOrphans))
	}
}

// mapEqual은 두 문자열 집합이 같은지 본다.
func mapEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// sortedKeys는 집합의 키를 정렬해 낸다.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
