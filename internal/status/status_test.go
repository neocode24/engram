package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
)

// fixedNow는 나이 계산의 고정 기준 시각이다. 시계를 읽지 않는다.
var fixedNow = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// writeWiki는 임시 디렉토리에 위키 파일들을 만든다.
func writeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// inboxDoc는 필수 필드를 갖춘 inbox 문서를 만든다.
func inboxDoc(created string, links string) string {
	fm := "type: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
		"indexable: false\nsource_channel: manual\n"
	if created != "" {
		fm += "created: " + created + "\n"
	}
	return "---\n" + fm + "---\n\n" + links + "\n"
}

// contextDoc는 필수 필드를 갖춘 context 문서를 만든다.
func contextDoc(created, updated string) string {
	fm := "type: concept\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\n" +
		"related:\n  - \"[[hub]]\"\nsource_channel: manual\nderived_context: []\n"
	if created != "" {
		fm += "created: " + created + "\n"
	}
	if updated != "" {
		fm += "updated: " + updated + "\n"
	}
	return "---\n" + fm + "---\n\n[[hub]] 링크\n"
}

func TestRun(t *testing.T) {
	t.Run("단계별 집계와 위키링크 수를 낸다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":           "preset: personal\n",
			"inbox/2026-07-15-a.md": inboxDoc("2026-07-15", "[[hub]] 와 [[peer]]"),
			"inbox/2026-08-01-b.md": inboxDoc("2026-08-01", ""),
			"context/hub.md":        contextDoc("2026-01-01", "2026-08-01"),
			"archive/old.md":        contextDoc("2025-01-01", "2026-01-01"),
			"sources/2026-06-01.md": "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\nindexable: false\nsource_channel: web\ncreated: 2026-06-01\nsourced_at: 2026-06-02\n---\n\n원본\n",
		})
		res, err := Run(root, fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		if res.Stages.Inbox != 2 || res.Stages.Context != 1 || res.Stages.Archive != 1 || res.Stages.Source != 1 {
			t.Errorf("단계별 집계가 틀리다: %+v", res.Stages)
		}
		// inbox 문서 2개(2개와 0개 링크), context 와 archive 문서가 각각 hub 1슬러그.
		if res.Links != 4 {
			t.Errorf("위키링크 수: got %d, want 4", res.Links)
		}
	})

	t.Run("들어오고 나가는 링크가 모두 없는 문서는 고아다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":           "preset: personal\n",
			"inbox/2026-07-15-a.md": inboxDoc("2026-07-15", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Orphans != 1 {
			t.Errorf("고아 수: got %d, want 1", res.Orphans)
		}
	})

	t.Run("고아 수는 lint 의 graph.orphan 건수와 일치한다", func(t *testing.T) {
		// 관계 필드만 있는 문서, 위키링크만 있는 문서, 아무 관계도 없는
		// 문서, 프론트매터가 없는 문서를 섞는다. lint 판정이 단일 진실원이다.
		root := writeWiki(t, map[string]string{
			"engram.yaml": "min_wikilinks: 0\n",
			// 관계 필드만 있다. 고아가 아니다.
			"sources/only-context.md": "---\n" +
				"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
				"indexable: false\nsource_refs: []\nderived_from: []\n" +
				"derived_context:\n  - 파생문서\n" +
				"source_channel: web\ncreated: 2026-01-01\nsourced_at: 2026-01-02\n" +
				"---\n\n원본\n",
			// 위키링크만 있다. 고아가 아니다.
			"inbox/2026-07-15-linked.md": inboxDoc("2026-07-15", "[[hub]] 본문 링크"),
			// 아무 관계도 없다. 고아다.
			"inbox/2026-08-01-alone.md": inboxDoc("2026-08-01", ""),
			// 프론트매터가 없다. lint 는 판정 대상에서 빼므로 고아 수에
			// 들어가지 않는다.
			"inbox/plain.md": "프론트매터 없는 메모\n",
		})
		res, err := Run(root, fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		lintRes, err := lint.Run(root, cfg)
		if err != nil {
			t.Fatal(err)
		}
		lintOrphans := 0
		for _, v := range lintRes.Violations {
			if v.Rule == "graph.orphan" {
				lintOrphans++
			}
		}
		if lintOrphans != 1 {
			t.Fatalf("lint 고아 건수 전제가 틀어졌다: %d건, want 1: %+v", lintOrphans, lintRes.Violations)
		}
		if res.Orphans != lintOrphans {
			t.Errorf("status 고아 수 %d 와 lint 고아 건수 %d 가 다르다", res.Orphans, lintOrphans)
		}
	})

	t.Run("lint 요약을 연동한다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml": "preset: personal\nforms: [note]\n",
			// form 이 폐쇄 집합에 없어 error 위반이 난다.
			"inbox/2026-07-15-a.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\nsource_channel: manual\nform: memo\ncreated: 2026-07-15\n---\n\n[[hub]] [[peer]]\n",
		})
		res, _ := Run(root, fixedNow)
		if res.Lint.Files != 1 {
			t.Errorf("lint 파일 수: got %d", res.Lint.Files)
		}
		if res.Lint.Error < 1 {
			t.Errorf("lint error 가 잡히지 않았다: %+v", res.Lint)
		}
	})

	t.Run("inbox 최고령 나이를 기준 시각으로 잰다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":           "preset: personal\n",
			"inbox/2026-07-15-a.md": inboxDoc("2026-07-15", ""),
			"inbox/2026-08-10-b.md": inboxDoc("2026-08-10", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.OldestDays == nil || *res.Backlog.OldestDays != 31 {
			t.Errorf("최고령 나이: got %v, want 31", res.Backlog.OldestDays)
		}
		// 기준 시각을 바꾸면 나이도 바뀐다. 시계가 아니라 인자로 잰 증거다.
		other, _ := Run(root, fixedNow.AddDate(0, 0, 10))
		if other.Backlog.OldestDays == nil || *other.Backlog.OldestDays != 41 {
			t.Errorf("기준 시각 반영: got %v, want 41", other.Backlog.OldestDays)
		}
	})

	t.Run("날짜를 알 수 없는 inbox 문서는 따로 센다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":      "preset: personal\n",
			"inbox/plain.md":   inboxDoc("", ""),
			"inbox/dated-c.md": inboxDoc("2026-08-01", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.UnknownAge != 1 {
			t.Errorf("알 수 없는 나이 문서: got %d, want 1", res.Backlog.UnknownAge)
		}
		if res.Backlog.OldestDays == nil || *res.Backlog.OldestDays != 14 {
			t.Errorf("최고령은 아는 문서 기준이어야 한다: %v", res.Backlog.OldestDays)
		}
	})

	t.Run("파일명 날짜 접두사로 나이를 잰다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":              "preset: personal\n",
			"inbox/2026-07-01-nofm.md": inboxDoc("", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.OldestDays == nil || *res.Backlog.OldestDays != 45 {
			t.Errorf("파일명 접두사 나이: got %v, want 45", res.Backlog.OldestDays)
		}
	})

	t.Run("stale_days 를 넘긴 context 문서를 센다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":      "stale_days: 30\n",
			"context/fresh.md": contextDoc("2026-01-01", "2026-08-01"),
			"context/old.md":   contextDoc("2025-01-01", "2026-01-01"),
			"context/nodate.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated:\n  - \"[[hub]]\"\nsource_channel: manual\nderived_context: []\n---\n\n[[hub]]\n",
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.Stale != 1 {
			t.Errorf("stale 문서: got %d, want 1 (날짜 없는 문서는 판정하지 않는다)", res.Backlog.Stale)
		}
	})

	t.Run("링크가 min_wikilinks 이상인 inbox 문서가 승급 대기다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml": "preset: personal\n",
			// related 와 본문에 서로 다른 슬러그 2개. min_wikilinks 기본 2 를 넘는다.
			// 링크 대상은 context 문서 2개(과 hub2)다. peer 는 inbox 라 대상에서 빠진다.
			"inbox/2026-07-15-ready.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel: manual\ncreated: 2026-07-15\nrelated:\n  - \"[[hub]]\"\n---\n\n[[peer]] 도 본다\n",
			"inbox/2026-08-01-notyet.md": inboxDoc("2026-08-01", "[[hub]]"),
			"context/hub.md":             contextDoc("2026-01-01", "2026-08-01"),
			"context/hub2.md":            contextDoc("2026-01-01", "2026-08-01"),
			"inbox/peer.md":              inboxDoc("2026-08-10", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.Promotable != 1 {
			t.Fatalf("승급 가능 문서: got %d, want 1", res.Backlog.Promotable)
		}
		if len(res.Backlog.PromotablePaths) != 1 || res.Backlog.PromotablePaths[0] != "inbox/2026-07-15-ready.md" {
			t.Errorf("승급 가능 경로: %v", res.Backlog.PromotablePaths)
		}
		// 첫 제안은 promote 명령을 안내한다.
		if len(res.Suggestions) == 0 || res.Suggestions[0].Action != "engram promote inbox/2026-07-15-ready.md" {
			t.Errorf("첫 제안: %+v", res.Suggestions)
		}
	})

	t.Run("올릴 수 없는 inbox 가 밀리면 링크를 채우라고 안내한다", func(t *testing.T) {
		// 링크 대상을 min_wikilinks 만큼 확보해 게이트가 동작하는 상태에서
		// 링크 없는 inbox 문서는 승급 대기가 아니다. 대상은 context 문서와
		// sources 문서다. inbox 문서는 대상에서 빠진다.
		//
		// 예전에는 이 상태에서 제안이 하나도 없었다. 밀린 것이 있는데
		// 도구가 할 말이 없다고 읽혀서 사용자가 다음에 뭘 할지 모른다.
		root := writeWiki(t, map[string]string{
			"engram.yaml":              "preset: personal\n",
			"inbox/2026-08-01-memo.md": inboxDoc("2026-08-01", ""),
			"inbox/2026-08-02-peer.md": inboxDoc("2026-08-02", ""),
			"context/hub.md":           contextDoc("2026-01-01", "2026-08-10"),
			"sources/2026-06-01.md":    "---\ntype: source-summary\nartifact_stage: source\nstatus: sourced\nindexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\nsource_channel: web\ncreated: 2026-06-01\nsourced_at: 2026-06-02\n---\n\n원본\n",
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.Inbox == 0 || res.Backlog.Promotable != 0 {
			t.Fatalf("시험 전제가 깨졌다. inbox %d, promotable %d", res.Backlog.Inbox, res.Backlog.Promotable)
		}
		if len(res.Suggestions) != 1 {
			t.Fatalf("제안이 하나여야 한다: %+v", res.Suggestions)
		}
		if !strings.Contains(res.Suggestions[0].Detail, "위키링크가 2개 필요합니다") {
			t.Errorf("게이트 기준을 알려야 한다: %+v", res.Suggestions[0])
		}
	})

	t.Run("inbox 문서는 링크 대상 집계에서 빠진다", func(t *testing.T) {
		// 문서 3개지만 링크 대상은 색인 하나뿐이다. inbox 문서를 세면
		// 대상이 2가 되어 이 문서는 유예 없이 거절된다. 승급 대기 판정으로
		// 대상 집계를 검증한다.
		root := writeWiki(t, map[string]string{
			"engram.yaml": "preset: personal\n",
			"index.md": "---\ntype: system\nartifact_stage: context\nstatus: promoted\n" +
				"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
				"source_channel: manual\nderived_context: []\n---\n\n# 색인\n",
			"inbox/2026-08-01-memo.md": inboxDoc("2026-08-01", "[[index]]"),
			"inbox/peer.md":            inboxDoc("2026-08-02", ""),
		})
		res, _ := Run(root, fixedNow)
		// 대상 1개로 유예. 링크가 1개뿐이어도 승급 대기로 본다.
		if res.Backlog.Promotable != 2 {
			t.Errorf("유예 중 위키의 승급 가능 수 = %d, want 2", res.Backlog.Promotable)
		}
	})

	t.Run("대상이 부족한 위키에서는 링크가 적어도 승급 대기로 본다", func(t *testing.T) {
		// 문서 하나뿐인 위키. 게이트가 유예되므로 promote 가능 목록에 오른다.
		root := writeWiki(t, map[string]string{
			"engram.yaml":              "preset: personal\n",
			"inbox/2026-08-01-memo.md": inboxDoc("2026-08-01", ""),
		})
		res, _ := Run(root, fixedNow)
		if res.Backlog.Promotable != 1 {
			t.Errorf("유예 중 위키에서 승급 가능 수 = %d, want 1", res.Backlog.Promotable)
		}
		if len(res.Suggestions) == 0 || !strings.HasPrefix(res.Suggestions[0].Action, "engram promote") {
			t.Errorf("유예 중 위키의 첫 제안은 promote 여야 한다: %+v", res.Suggestions)
		}
	})

	t.Run("제안은 최대 3개다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml": "stale_days: 1\nforms: [note]\n",
			// 승급 가능 + stale + lint error(form 위반) 세 가지와 빈 inbox 안내까지 4개 조건.
			"inbox/2026-07-15-ready.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
				"indexable: false\nsource_channel: manual\ncreated: 2026-07-15\nrelated:\n  - \"[[hub]]\"\nform: memo\n---\n\n[[peer]]\n",
			"inbox/peer.md":  inboxDoc("2026-08-01", ""),
			"context/hub.md": contextDoc("2026-01-01", "2026-01-01"),
		})
		res, _ := Run(root, fixedNow)
		if len(res.Suggestions) != 3 {
			t.Errorf("제안은 최대 3개다: %d개, %+v", len(res.Suggestions), res.Suggestions)
		}
	})

	t.Run("같은 위키와 같은 기준 시각에 두 번 실행하면 JSON 이 바이트까지 같다", func(t *testing.T) {
		root := writeWiki(t, map[string]string{
			"engram.yaml":           "preset: personal\n",
			"inbox/2026-07-15-a.md": inboxDoc("2026-07-15", "[[hub]] [[peer]]"),
			"context/hub.md":        contextDoc("2026-01-01", "2026-08-01"),
			"inbox/peer.md":         inboxDoc("2026-08-01", ""),
		})
		first, err := Run(root, fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := Run(root, fixedNow)
		a, _ := json.Marshal(first)
		b, _ := json.Marshal(second)
		if string(a) != string(b) {
			t.Errorf("두 실행 결과가 다르다:\n%s\n%s", a, b)
		}
	})

	t.Run("위키가 아니면 거절한다", func(t *testing.T) {
		_, err := Run(t.TempDir(), fixedNow)
		if err == nil {
			t.Fatal("위키가 아닌 디렉토리는 에러여야 한다")
		}
		if want := "engram init"; !strings.Contains(err.Error(), want) {
			t.Errorf("안내에 engram init 이 없다: %v", err)
		}
	})
}
