package resurface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/walk"
)

// makeWiki는 임시 디렉토리에 engram.yaml 과 문서 파일들을 만든다.
// resurface 는 lint 를 돌리지 않으므로 프론트매터는 날짜에 필요한 최소만
// 쓴다.
func makeWiki(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{"engram.yaml": "preset: education\n"}
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

// fixedNow는 테스트 전체가 쓰는 기준 시각이다.
func fixedNow() time.Time {
	now, err := time.Parse(time.RFC3339, "2026-08-16T12:00:00Z")
	if err != nil {
		panic(err)
	}
	return now
}

// loadCfg는 임시 위키의 설정을 읽는다.
func loadCfg(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// firstParsed는 임시 위키의 첫 문서 파싱 결과를 반환한다.
func firstParsed(t *testing.T, root string) walk.Doc {
	t.Helper()
	walked, err := walk.Files(root, loadCfg(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if len(walked) == 0 {
		t.Fatal("문서가 없습니다")
	}
	return walked[0]
}

func TestBaseDate(t *testing.T) {
	cases := []struct {
		name    string
		created string
		updated string
		want    string
		ok      bool
	}{
		{"updated 를 우선한다", "2026-01-01", "2026-03-01", "2026-03-01", true},
		{"updated 가 없으면 created", "2026-01-01", "", "2026-01-01", true},
		{"둘 다 없으면 없음", "", "", "", false},
		{"연월 정밀도도 받는다", "2026-03", "", "2026-03", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := makeWiki(t, map[string]string{
				"context/a.md": docFM(c.created, c.updated, "본문"),
			})
			got, ok := BaseDate(firstParsed(t, root).Parsed)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if ok {
				if want := date(t, c.want); !got.Equal(want) {
					t.Fatalf("기준 날짜 = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestIsStale(t *testing.T) {
	now := fixedNow()
	// 경계를 정확히 잣는다. 90일째는 노후가 아니고 91일째는 노후다.
	base := now.AddDate(0, 0, -90)
	if days(now, base) != 90 {
		t.Fatalf("90일 전 경과일 = %d", days(now, base))
	}
	root := makeWiki(t, map[string]string{
		"context/fresh.md":  docFM("", now.Format("2006-01-02"), "본문"),
		"context/edge90.md": docFM("", base.Format("2006-01-02"), "본문"),
		"context/edge91.md": docFM("", base.AddDate(0, 0, -1).Format("2006-01-02"), "본문"),
		// 기준 날짜가 없으면 노후가 아니라 대상 밖이다.
		"context/nodate.md": "---\ntype: concept\n---\n\n본문",
	})
	walked, err := walk.Files(root, loadCfg(t, root))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range walked {
		want := w.Rel == "context/edge91.md"
		if got := IsStale(w.Parsed, now, 90); got != want {
			t.Fatalf("%s: IsStale = %v, want %v", w.Rel, got, want)
		}
	}
}

func TestRunSelectsAndSorts(t *testing.T) {
	now := fixedNow()
	root := makeWiki(t, map[string]string{
		// 후보. 기준 날짜가 오래된 순으로 old-b 가 old-a 보다 먼저다.
		"context/old-a.md": docFM("2026-01-01", "", "# 첫 결정"),
		"context/old-b.md": docFM("2025-12-01", "", "# 더 오래된 결정"),
		// 최근 문서와 inbox 문서는 후보가 아니다.
		"context/fresh.md":                docFM("", "2026-08-01", "본문"),
		"inbox/2026-01-01-stale-inbox.md": docFM("2026-01-01", "", "본문"),
		// 기준 날짜가 없는 문서는 대상에서 빼고 알린다.
		"context/nodate.md": "---\ntype: concept\n---\n\n본문",
	})
	res, err := Run(root, loadCfg(t, root), now, 5, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("후보 수 = %d, want 2: %+v", len(res.Candidates), res.Candidates)
	}
	if res.Candidates[0].Slug != "old-b" || res.Candidates[1].Slug != "old-a" {
		t.Fatalf("정렬 순서가 다릅니다: %s, %s", res.Candidates[0].Slug, res.Candidates[1].Slug)
	}
	if res.Candidates[0].LastShown != nil {
		t.Error("제시한 적 없는 문서에 LastShown 이 있습니다")
	}
	if res.Candidates[1].Title != "첫 결정" {
		t.Errorf("제목 = %q, want %q", res.Candidates[1].Title, "첫 결정")
	}
	if res.SkippedNoDate != 1 {
		t.Errorf("날짜 없음 = %d, want 1", res.SkippedNoDate)
	}
	// 실행 후 낸 문서의 제시 시각이 now 로 기록된다.
	shown := LoadHistory(root)
	if len(shown) != 2 {
		t.Fatalf("이력 크기 = %d, want 2", len(shown))
	}
	for slug, ts := range shown {
		if !ts.Equal(now) {
			t.Errorf("%s의 제시 시각 = %v, want %v", slug, ts, now)
		}
	}
}

func TestRunNeverShownFirstThenOldestShown(t *testing.T) {
	now := fixedNow()
	older, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	recent, _ := time.Parse(time.RFC3339, "2026-08-10T00:00:00Z")

	t.Run("둘 다 제시했으면 오래된 쪽이 먼저", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/old-a.md": docFM("2026-01-01", "", "본문"),
			"context/old-b.md": docFM("2025-12-01", "", "본문"),
		})
		if err := SaveHistory(root, map[string]time.Time{"old-a": recent, "old-b": older}); err != nil {
			t.Fatal(err)
		}
		res, err := Run(root, loadCfg(t, root), now, 5, true)
		if err != nil {
			t.Fatal(err)
		}
		if res.Candidates[0].Slug != "old-b" || res.Candidates[1].Slug != "old-a" {
			t.Fatalf("제시 이력 정렬이 다릅니다: %s, %s", res.Candidates[0].Slug, res.Candidates[1].Slug)
		}
		if res.Candidates[0].LastShown == nil || !res.Candidates[0].LastShown.Equal(older) {
			t.Errorf("old-b 의 마지막 제시 = %v, want %v", res.Candidates[0].LastShown, older)
		}
	})

	t.Run("제시한 적 없는 문서가 먼저", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/old-a.md": docFM("2026-01-01", "", "본문"),
			"context/old-b.md": docFM("2025-12-01", "", "본문"),
		})
		if err := SaveHistory(root, map[string]time.Time{"old-b": older}); err != nil {
			t.Fatal(err)
		}
		res, err := Run(root, loadCfg(t, root), now, 5, true)
		if err != nil {
			t.Fatal(err)
		}
		if res.Candidates[0].Slug != "old-a" {
			t.Fatalf("제시한 적 없는 문서가 먼저여야 합니다: %+v", res.Candidates)
		}
	})

	t.Run("경과일이 같으면 슬러그 오름차순", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/b-same.md": docFM("2026-01-01", "", "본문"),
			"context/a-same.md": docFM("2026-01-01", "", "본문"),
		})
		res, err := Run(root, loadCfg(t, root), now, 5, true)
		if err != nil {
			t.Fatal(err)
		}
		if res.Candidates[0].Slug != "a-same" || res.Candidates[1].Slug != "b-same" {
			t.Fatalf("동점 정렬이 다릅니다: %+v", res.Candidates)
		}
	})
}

func TestRunLimitAndDryRun(t *testing.T) {
	now := fixedNow()
	root := makeWiki(t, map[string]string{
		"context/old-a.md": docFM("2026-01-01", "", "본문"),
		"context/old-b.md": docFM("2025-12-01", "", "본문"),
	})
	// dry-run 은 후보를 내되 이력을 남기지 않는다.
	if _, err := Run(root, loadCfg(t, root), now, 5, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, stateDir, historyName)); !os.IsNotExist(err) {
		t.Fatalf("dry-run 이 이력을 남겼습니다: %v", err)
	}
	// limit 은 낸 문서 수와 기록 수를 같게 제한한다.
	res, err := Run(root, loadCfg(t, root), now, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].Slug != "old-b" {
		t.Fatalf("limit 1 결과: %+v", res.Candidates)
	}
	shown := LoadHistory(root)
	if len(shown) != 1 || shown["old-b"].IsZero() {
		t.Fatalf("기록된 이력: %+v", shown)
	}
}

func TestRunNoCandidatesReason(t *testing.T) {
	now := fixedNow()
	t.Run("context 문서가 없으면 이유를 낸다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"inbox/2026-01-01-a.md": docFM("2026-01-01", "", "본문"),
		})
		res, err := Run(root, loadCfg(t, root), now, 5, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Candidates) != 0 || res.Reason == "" {
			t.Fatalf("후보와 이유: %+v", res)
		}
	})
	t.Run("전부 최신이면 이유를 낸다", func(t *testing.T) {
		root := makeWiki(t, map[string]string{
			"context/fresh.md": docFM("", "2026-08-01", "본문"),
		})
		res, err := Run(root, loadCfg(t, root), now, 5, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Candidates) != 0 || res.Reason == "" {
			t.Fatalf("후보와 이유: %+v", res)
		}
	})
}

func TestLoadHistoryLenient(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, stateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, historyName)
	t.Run("파일이 없으면 빈 이력", func(t *testing.T) {
		if got := LoadHistory(root); len(got) != 0 {
			t.Fatalf("빈 이력이 아닙니다: %+v", got)
		}
	})
	t.Run("깨진 파일은 빈 이력", func(t *testing.T) {
		if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := LoadHistory(root); len(got) != 0 {
			t.Fatalf("깨진 파일을 읽었습니다: %+v", got)
		}
	})
	t.Run("스키마 버전이 다르면 빈 이력", func(t *testing.T) {
		if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"shown":{"a":"2026-01-01T00:00:00Z"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := LoadHistory(root); len(got) != 0 {
			t.Fatalf("다른 버전을 읽었습니다: %+v", got)
		}
	})
	t.Run("일부 항목이 깨져도 나머지를 쓴다", func(t *testing.T) {
		if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"shown":{"a":"2026-01-01T00:00:00Z","b":"not-a-time"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		got := LoadHistory(root)
		if len(got) != 1 {
			t.Fatalf("이력 = %+v, want a 하나", got)
		}
	})
}

func TestSaveHistoryDeterministic(t *testing.T) {
	root := t.TempDir()
	ts, _ := time.Parse(time.RFC3339, "2026-08-16T12:00:00+09:00")
	shown := map[string]time.Time{"b": ts, "a": ts, "c": ts}
	if err := SaveHistory(root, shown); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(root, stateDir, historyName))
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveHistory(root, shown); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(root, stateDir, historyName))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("같은 상태의 두 저장이 다릅니다:\n%s\n===\n%s", first, second)
	}
	// 키가 정렬되어 있고 시각은 UTC 로 정규화된다.
	var f struct {
		SchemaVersion int               `json:"schemaVersion"`
		Shown         map[string]string `json:"shown"`
	}
	if err := json.Unmarshal(first, &f); err != nil {
		t.Fatal(err)
	}
	if f.SchemaVersion != 1 || len(f.Shown) != 3 {
		t.Fatalf("이력 파일 구조: %+v", f)
	}
	if f.Shown["a"] != "2026-08-16T03:00:00Z" {
		t.Errorf("시각 정규화: %q", f.Shown["a"])
	}
	// 파일 안의 키 순서가 정렬인지 본문에서 직접 확인한다.
	body := string(first)
	posA, posB, posC := strings.Index(body, `"a":`), strings.Index(body, `"b":`), strings.Index(body, `"c":`)
	if !(posA < posB && posB < posC) {
		t.Errorf("키가 정렬되지 않았습니다: a=%d b=%d c=%d", posA, posB, posC)
	}
	if !strings.HasSuffix(body, "\n") {
		t.Error("파일이 줄바꿈으로 끝나지 않습니다")
	}
}

// date는 테스트 날짜 리터럴을 time.Time 으로 만든다. 하루 단위와 연월
// 단위를 모두 받는다.
func date(t *testing.T, s string) time.Time {
	t.Helper()
	for _, layout := range []string{"2006-01-02", "2006-01"} {
		if tm, err := time.Parse(layout, s); err == nil {
			return tm
		}
	}
	t.Fatalf("날짜 리터럴을 파싱할 수 없습니다: %q", s)
	return time.Time{}
}
