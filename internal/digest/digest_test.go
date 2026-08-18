package digest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
)

// makeWiki는 임시 디렉토리에 engram.yaml 과 문서 파일들을 만든다.
func makeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{"engram.yaml": "preset: personal\n"}
	for k, v := range files {
		all[k] = v
	}
	for name, content := range all {
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

// docFM은 created 와 updated 를 채운 최소 프론트매터 문서를 만든다.
func docFM(created, updated, body string) string {
	fm := "---\n"
	if created != "" {
		fm += "created: " + created + "\n"
	}
	if updated != "" {
		fm += "updated: " + updated + "\n"
	}
	return fm + "---\n\n" + body
}

// fixedNow는 테스트 전체가 쓰는 기준 시각이다. --days 30 의 창 시작은
// 2026-07-17T12:00:00Z 다.
func fixedNow() time.Time {
	now, err := time.Parse(time.RFC3339, "2026-08-16T12:00:00Z")
	if err != nil {
		panic(err)
	}
	return now
}

func loadCfg(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestRunAggregates(t *testing.T) {
	now := fixedNow()
	root := makeWiki(t, map[string]string{
		// 창 안에 만들어진 context 문서. old-context 와 서로 링크해 고아가 아니다.
		"context/new-context.md": docFM("2026-08-01", "2026-08-01", "[[old-context]]"),
		// 창 밖에서 만들어지고 stale_days 를 넘긴 문서. 신규가 아니고 노후다.
		"context/old-context.md": docFM("2025-01-01", "2025-01-01", "[[new-context]]"),
		// 창 안이고 링크가 없어 신규이면서 고아다.
		"context/orphan.md": docFM("2026-07-20", "2026-07-20", "본문"),
		// 창 밖이라 신규가 아니다. 링크가 없어 고아다. 고아는 단계를 가리지 않는다.
		"inbox/2026-07-15-inbox-new.md": docFM("2026-07-15", "", "본문"),
	})
	res, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"new-context", "orphan"}; !reflect.DeepEqual(res.Created, want) {
		t.Errorf("신규 = %v, want %v", res.Created, want)
	}
	if want := []string{"old-context"}; !reflect.DeepEqual(res.Stale, want) {
		t.Errorf("노후 = %v, want %v", res.Stale, want)
	}
	if want := []string{"2026-07-15-inbox-new", "orphan"}; !reflect.DeepEqual(res.Orphans, want) {
		t.Errorf("고아 = %v, want %v", res.Orphans, want)
	}
	if res.Days != 30 || res.StaleDays != 30 {
		t.Errorf("기간/노후 기준: days=%d staleDays=%d", res.Days, res.StaleDays)
	}
	if res.Since != "2026-07-17T12:00:00Z" || res.Until != "2026-08-16T12:00:00Z" {
		t.Errorf("창: since=%s until=%s", res.Since, res.Until)
	}
}

func TestRunWindowBoundary(t *testing.T) {
	now := fixedNow()
	// 창 시작이 2026-07-17T12:00:00Z 이므로 같은 날 00:00 으로 파싱되는
	// created 2026-07-17 은 창 밖이고 07-18 은 창 안이다.
	root := makeWiki(t, map[string]string{
		"context/edge-in.md":  docFM("2026-07-18", "", "본문"),
		"context/edge-out.md": docFM("2026-07-17", "", "본문"),
	})
	res, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"edge-in"}; !reflect.DeepEqual(res.Created, want) {
		t.Errorf("신규 = %v, want %v", res.Created, want)
	}
}

func TestRunStaleMatchesResurfaceRule(t *testing.T) {
	now := fixedNow()
	// 노후는 resurface.IsStale 과 같은 규칙이어야 한다. updated 가 없으면
	// created 로 잰다. 2026-04-10 은 128일로 90일을 넘는다.
	root := makeWiki(t, map[string]string{
		"context/a.md": docFM("2026-04-10", "", "본문"),
		"context/b.md": docFM("", "2026-08-01", "본문"),
		// inbox 문서가 아무리 낡아도 노후가 아니다.
		"inbox/2026-01-01-old-inbox.md": docFM("2026-01-01", "", "본문"),
	})
	res, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a"}; !reflect.DeepEqual(res.Stale, want) {
		t.Errorf("노후 = %v, want %v", res.Stale, want)
	}
}

func TestRunOrphansMatchLintCount(t *testing.T) {
	now := fixedNow()
	root := makeWiki(t, map[string]string{
		"context/lonely1.md":          docFM("2025-01-01", "", "본문"),
		"context/lonely2.md":          docFM("2025-01-01", "", "본문"),
		"inbox/2026-01-01-lonely3.md": docFM("2026-01-01", "", "본문"),
	})
	res, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	cfg := loadCfg(t, root)
	lintRes, err := lint.Run(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Orphans) != lint.OrphanCount(lintRes) {
		t.Errorf("고아 %d개가 lint 판정 %d개와 다릅니다", len(res.Orphans), lint.OrphanCount(lintRes))
	}
	if want := []string{"2026-01-01-lonely3", "lonely1", "lonely2"}; !reflect.DeepEqual(res.Orphans, want) {
		t.Errorf("고아 = %v, want %v (정렬 포함)", res.Orphans, want)
	}
}

func TestRunDeterministic(t *testing.T) {
	now := fixedNow()
	root := makeWiki(t, map[string]string{
		"context/new.md": docFM("2026-08-01", "2026-08-01", "본문"),
		"context/old.md": docFM("2025-01-01", "2025-01-01", "본문"),
	})
	first, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(root, loadCfg(t, root), now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("두 실행의 결과가 다릅니다:\n%+v\n===\n%+v", first, second)
	}
}
