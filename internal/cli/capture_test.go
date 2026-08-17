package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runIngestRoot는 capture와 source를 등록한 루트 커맨드를 실행한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 루트를 조립한다.
func runIngestRoot(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	root.AddCommand(newCaptureCmd())
	root.AddCommand(newSourceCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if stdin != "" {
		root.SetIn(strings.NewReader(stdin))
	}
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// initWiki는 임시 디렉토리에 init으로 위키를 만들고 그 경로를 반환한다.
func initWiki(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wiki")
	if _, err := runIngestRoot(t, "", "init", dir); err != nil {
		t.Fatalf("init 실패: %v", err)
	}
	return dir
}

func TestCapture(t *testing.T) {
	t.Run("인자로 받은 내용을 inbox에 넣습니다", func(t *testing.T) {
		dir := initWiki(t)
		out, err := runIngestRoot(t, "", "capture", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "회의 메모 내용")
		if err != nil {
			t.Fatalf("capture 실패: %v", err)
		}
		path := filepath.Join(dir, "inbox", "2026-01-01-회의-메모-내용.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("inbox 문서가 없음: %v\n출력: %s", err, out)
		}
		raw := string(readWikiFile(t, filepath.Dir(path), filepath.Base(path)))
		for _, want := range []string{"type: inbox-note\n", "artifact_stage: inbox\n", "status: inbox\n"} {
			if !strings.Contains(raw, want) {
				t.Errorf("프론트매터에 %q 없음:\n%s", want, raw)
			}
		}
		if !strings.Contains(raw, "회의 메모 내용\n") {
			t.Errorf("본문이 잘못됨:\n%s", raw)
		}
	})

	t.Run("표준 입력으로 받은 내용을 넣습니다", func(t *testing.T) {
		dir := initWiki(t)
		out, err := runIngestRoot(t, "파이프로 넘긴 메모",
			"capture", "--wiki", dir, "--now", "2026-02-03T00:00:00Z")
		if err != nil {
			t.Fatalf("capture 실패: %v\n%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "inbox", "2026-02-03-파이프로-넘긴-메모.md")); err != nil {
			t.Fatalf("stdin 문서가 없음: %v", err)
		}
	})

	t.Run("내용이 없으면 사용법을 안내합니다", func(t *testing.T) {
		_, err := runIngestRoot(t, "", "capture", "--wiki", initWiki(t))
		if err == nil {
			t.Fatal("내용이 없으면 에러여야 함")
		}
		if !strings.Contains(err.Error(), "내용을 받지 못했습니다") {
			t.Errorf("에러에 사용법 안내가 없음: %v", err)
		}
	})

	t.Run("제목은 본문 첫 줄에서 파생하고 헤딩 기호를 뗍니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runIngestRoot(t, "", "capture", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--", "# 주간 회의록\n내용"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "inbox", "2026-01-01-주간-회의록.md")); err != nil {
			t.Fatalf("제목 파생 파일이 없음: %v", err)
		}
	})

	t.Run("--slug로 슬러그를 직접 지정합니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runIngestRoot(t, "", "capture", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "weekly-notes", "내용"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "inbox", "2026-01-01-weekly-notes.md")); err != nil {
			t.Fatalf("지정 슬러그 파일이 없음: %v", err)
		}
	})

	t.Run("프론트매터는 프리셋이 켠 축만 담습니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runIngestRoot(t, "", "capture", "--wiki", dir, "내용"); err != nil {
			t.Fatal(err)
		}
		// 기본 프리셋 personal은 source_channel을 켜고 scope를 끈다.
		entries, err := os.ReadDir(filepath.Join(dir, "inbox"))
		if err != nil || len(entries) != 1 {
			t.Fatalf("inbox 문서를 찾을 수 없음: %v", err)
		}
		raw := string(readWikiFile(t, filepath.Join(dir, "inbox"), entries[0].Name()))
		if !strings.Contains(raw, "source_channel:\n") {
			t.Errorf("켜진 축이 없음:\n%s", raw)
		}
		if strings.Contains(raw, "scope:") {
			t.Errorf("꺼진 축이 들어 있음:\n%s", raw)
		}
	})

	t.Run("같은 이름의 파일이 있으면 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		args := []string{"capture", "--wiki", dir, "--now", "2026-01-01T00:00:00Z", "--slug", "dup", "내용"}
		if _, err := runIngestRoot(t, "", args...); err != nil {
			t.Fatal(err)
		}
		_, err := runIngestRoot(t, "", args...)
		if err == nil {
			t.Fatal("중복 생성은 거절되어야 함")
		}
		if !strings.Contains(err.Error(), "이미 문서가 있습니다") {
			t.Errorf("거절 메시지가 잘못됨: %v", err)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init을 안내합니다", func(t *testing.T) {
		_, err := runIngestRoot(t, "", "capture", "--wiki", t.TempDir(), "내용")
		if err == nil {
			t.Fatal("위키가 아니면 거절되어야 함")
		}
		if !strings.Contains(err.Error(), "engram init") {
			t.Errorf("거절 메시지에 init 안내가 없음: %v", err)
		}
	})
}

func TestCaptureSlugSafety(t *testing.T) {
	// 명시한 슬러그도 파일시스템 안전 검사를 받는다(ADR 0045).
	// 거절 대상은 파일을 만들 수 없게 하는 것뿐이고 취향인 규칙은
	// 강제하지 않는다.
	t.Run("파일시스템이 감당하지 못하는 슬러그를 거절합니다", func(t *testing.T) {
		tests := []struct {
			slug string
			want string
		}{
			{slug: "a:b", want: "쓸 수 없는 문자"},
			{slug: "CON", want: "예약 파일명"},
			{slug: "con", want: "예약 파일명"},
			{slug: "PRN", want: "예약 파일명"},
			{slug: "aux", want: "예약 파일명"},
			{slug: "NUL", want: "예약 파일명"},
			{slug: "com1", want: "예약 파일명"},
			{slug: "LPT1", want: "예약 파일명"},
			{slug: "trailing.", want: "점이나 공백으로 끝납니다"},
			{slug: "trailing ", want: "점이나 공백으로 끝납니다"},
			{slug: "제어\t문자", want: "제어 문자"},
			{slug: "sub/dir", want: "경로 구분자"},
			{slug: `sub\dir`, want: "경로 구분자"},
			{slug: "../escape", want: "경로 구분자"},
		}
		for _, tt := range tests {
			dir := initWiki(t)
			out, err := runIngestRoot(t, "", "capture", "--wiki", dir,
				"--now", "2026-01-01T00:00:00Z", "--slug", tt.slug, "내용")
			if err == nil {
				t.Errorf("슬러그 %q는 거절되어야 함\n출력: %s", tt.slug, out)
				continue
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("슬러그 %q 거절 메시지에 %q 없음: %v", tt.slug, tt.want, err)
			}
			// 거절했으면 아무 파일도 남지 않는다.
			entries, readErr := os.ReadDir(filepath.Join(dir, "inbox"))
			if readErr == nil && len(entries) > 0 {
				t.Errorf("슬러그 %q가 거절되었는데 파일이 남음: %v", tt.slug, entries[0].Name())
			}
		}
	})

	t.Run("소문자와 하이픈 정규화를 강제하지 않습니다", func(t *testing.T) {
		// ADR 0020의 "사용자는 언제든 슬러그를 명시해 파생을 덮어쓸 수
		// 있다"가 사는 자리다. 명시한 값을 조용히 고치지 않는다.
		for _, slug := range []string{"대문자-슬러그", "UPPER-Case", "공백 있는 슬러그", "한글-슬러그"} {
			dir := initWiki(t)
			out, err := runIngestRoot(t, "", "capture", "--wiki", dir,
				"--now", "2026-01-01T00:00:00Z", "--slug", slug, "내용")
			if err != nil {
				t.Errorf("슬러그 %q는 통과해야 함: %v\n출력: %s", slug, err, out)
				continue
			}
			want := filepath.Join(dir, "inbox", "2026-01-01-"+slug+".md")
			if _, err := os.Stat(want); err != nil {
				t.Errorf("슬러그 %q가 그대로 파일명이 되지 않음: %v", slug, err)
			}
		}
	})

	t.Run("파생 경로의 파일명이 그대로입니다", func(t *testing.T) {
		// 안전 검사는 명시 경로에만 붙는다. 제목에서 만드는 슬러그는
		// 전과 같은 파일명을 낸다.
		tests := []struct {
			title string
			want  string
		}{
			{title: "회의 메모 내용", want: "2026-01-01-회의-메모-내용.md"},
			{title: "Table Driven Tests", want: "2026-01-01-table-driven-tests.md"},
			{title: "go  --  table  tests__v2", want: "2026-01-01-go-table-tests-v2.md"},
		}
		for _, tt := range tests {
			dir := initWiki(t)
			if _, err := runIngestRoot(t, "", "capture", "--wiki", dir,
				"--now", "2026-01-01T00:00:00Z", "--title", tt.title, "내용"); err != nil {
				t.Errorf("제목 %q capture 실패: %v", tt.title, err)
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, "inbox", tt.want)); err != nil {
				t.Errorf("제목 %q의 파생 파일명이 바뀜: %v", tt.title, err)
			}
		}
	})
}

func TestSource(t *testing.T) {
	t.Run("원본을 sources에 넣고 updated를 쓰지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		_, err := runIngestRoot(t, "", "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk",
			"--created", "2025-11-08", "--channel", "web",
			"--ref", "https://example.com/talk", "--", "발표 요약 내용")
		if err != nil {
			t.Fatalf("source 실패: %v", err)
		}
		path := filepath.Join(dir, "sources", "2025-11-08-talk.md")
		raw := string(readWikiFile(t, filepath.Dir(path), filepath.Base(path)))
		for _, want := range []string{
			"type: source-summary\n", "status: sourced\n", "indexable: false\n",
			"created: 2025-11-08\n", "sourced_at: 2026-01-01\n",
			"source_channel: web\n", "- https://example.com/talk\n",
		} {
			if !strings.Contains(raw, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, raw)
			}
		}
		if strings.Contains(raw, "updated") {
			t.Errorf("sources 문서에 updated가 있음:\n%s", raw)
		}
	})

	t.Run("--created는 연월 정밀도도 허용합니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runIngestRoot(t, "", "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "old", "--created", "2023-05", "내용"); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "sources", "2023-05-old.md")
		raw := string(readWikiFile(t, filepath.Dir(path), filepath.Base(path)))
		if !strings.Contains(raw, "created: 2023-05\n") {
			t.Errorf("연월 created가 없음:\n%s", raw)
		}
	})

	t.Run("잘못된 --created 형식은 거절합니다", func(t *testing.T) {
		_, err := runIngestRoot(t, "", "source", "--wiki", initWiki(t),
			"--now", "2026-01-01T00:00:00Z", "--created", "2026/01", "내용")
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM") {
			t.Fatalf("형식 거절이 잘못됨: %v", err)
		}
	})

	t.Run("허용값 밖의 --type은 목록과 함께 거절합니다", func(t *testing.T) {
		_, err := runIngestRoot(t, "", "source", "--wiki", initWiki(t),
			"--type", "diary", "내용")
		if err == nil {
			t.Fatal("거절되어야 함")
		}
		if !strings.Contains(err.Error(), "concept") || !strings.Contains(err.Error(), "source-summary") {
			t.Errorf("거절 메시지에 허용값 목록이 없음: %v", err)
		}
	})

	t.Run("꺼진 축에 값을 주면 경고하고 무시합니다", func(t *testing.T) {
		dir := initWiki(t)
		// personal 프리셋은 source_channel을 끈다. source_refs는 켜져 있어
		// 함께 끄는 것을 설정에 명시한다.
		cfgPath := filepath.Join(dir, "engram.yaml")
		if err := os.WriteFile(cfgPath, []byte("preset: minimal\naxes:\n  source_refs: false\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runIngestRoot(t, "", "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "warn",
			"--channel", "web", "--ref", "https://example.com/a", "내용")
		if err != nil {
			t.Fatalf("경고는 거절이 아닙니다: %v", err)
		}
		if !strings.Contains(out, "경고: source_channel 속성이 꺼져 있어") ||
			!strings.Contains(out, "경고: source_refs 속성이 꺼져 있어") {
			t.Errorf("경고가 출력되지 않음:\n%s", out)
		}
		raw := string(readWikiFile(t, filepath.Join(dir, "sources"), "2026-01-01-warn.md"))
		if strings.Contains(raw, "source_channel: web") || strings.Contains(raw, "example.com") {
			t.Errorf("꺼진 축의 값이 들어 있음:\n%s", raw)
		}
	})
}

func TestIngestCommon(t *testing.T) {
	t.Run("--json은 경로와 슬러그와 단계를 냅니다", func(t *testing.T) {
		dir := initWiki(t)
		out, err := runIngestRoot(t, "", "capture", "--json", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "json-test", "내용")
		if err != nil {
			t.Fatal(err)
		}
		var res ingestResult
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Stage != "inbox" || res.Slug != "json-test" {
			t.Errorf("JSON 내용이 잘못됨: %+v", res)
		}
		if filepath.Base(res.Path) != "2026-01-01-json-test.md" {
			t.Errorf("JSON 경로가 잘못됨: %s", res.Path)
		}
	})

	t.Run("같은 --now로 두 번 만들면 바이트까지 같습니다", func(t *testing.T) {
		a, b := initWiki(t), initWiki(t)
		args := []string{"source", "--wiki", a, "--now", "2026-01-01T00:00:00Z",
			"--slug", "same", "--created", "2025-12-01", "--channel", "web",
			"--ref", "https://example.com/x", "--", "# 발표\n요약 본문"}
		if _, err := runIngestRoot(t, "", args...); err != nil {
			t.Fatal(err)
		}
		args[2] = b
		if _, err := runIngestRoot(t, "", args...); err != nil {
			t.Fatal(err)
		}
		x := readWikiFile(t, filepath.Join(a, "sources"), "2025-12-01-same.md")
		y := readWikiFile(t, filepath.Join(b, "sources"), "2025-12-01-same.md")
		if x != y {
			t.Fatalf("두 결과가 다름:\n%s\n===\n%s", x, y)
		}
	})
}
