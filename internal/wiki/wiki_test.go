package wiki

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/doc"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		want    string
		wantErr bool
	}{
		{name: "영문 제목은 소문자 하이픈으로", title: "Table Driven Tests", want: "table-driven-tests"},
		{name: "한글은 그대로 보존한다", title: "테이블 주도 테스트", want: "테이블-주도-테스트"},
		{name: "연속 구분자를 하나로 줄인다", title: "go  --  table  tests__v2", want: "go-table-tests-v2"},
		{name: "앞뒤 하이픈을 없앤다", title: " -- go tests -- ", want: "go-tests"},
		{name: "윈도우 예약 문자를 지운다", title: `a<b>c:d"e/f\g|h?i*j`, want: "abcdefghij"},
		{name: "빈 결과는 에러다", title: "??? ***", wantErr: true},
		{name: "윈도우 예약 파일명은 에러다", title: "con", wantErr: true},
		{name: "대소문자 구분 없는 예약 파일명도 에러다", title: "Aux", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Slug(tt.title)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("에러여야 함, got %q", got)
				}
				if !strings.Contains(err.Error(), "슬러그") {
					t.Errorf("에러 메시지에 안내가 없음: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("슬러그 = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilePath(t *testing.T) {
	t.Run("page_dirs에 단계 디렉토리가 없으면 설정 안내와 함께 에러다", func(t *testing.T) {
		cfg := defaultCfg(t)
		cfg.PageDirs = []string{"sources", "context", "archive"}
		_, err := FilePath(cfg, StageInbox, "2026-01-01", "slug")
		if err == nil {
			t.Fatal("page_dirs에 inbox가 없으면 에러여야 함")
		}
		for _, want := range []string{"page_dirs", "inbox", "engram.yaml"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("에러 메시지에 %q 없음: %v", want, err)
			}
		}
	})

	t.Run("page_dirs를 바꾼 위키에서 도착지가 설정을 따른다", func(t *testing.T) {
		// inbox와 context를 남기고 나머지를 바꾼 구성이다.
		cfg := defaultCfg(t)
		cfg.PageDirs = []string{"inbox", "journal", "context"}
		got, err := FilePath(cfg, StageContext, "2026-01-01", "slug")
		if err != nil || got != "context/slug.md" {
			t.Errorf("경로 = %q, %v, want context/slug.md", got, err)
		}
		// source 단계 디렉토리가 page_dirs에서 빠졌으므로 에러다.
		if _, err := FilePath(cfg, StageSource, "2026-01-01", "slug"); err == nil {
			t.Error("sources가 page_dirs에 없으면 에러여야 함")
		}
	})

	t.Run("context 문서는 날짜 접두사가 없다", func(t *testing.T) {
		got, err := FilePath(defaultCfg(t), StageContext, "2026-01-01", "slug")
		if err != nil || got != "context/slug.md" {
			t.Errorf("경로 = %q, %v, want context/slug.md", got, err)
		}
	})
	t.Run("inbox 문서는 날짜 접두사가 붙는다", func(t *testing.T) {
		got, err := FilePath(defaultCfg(t), StageInbox, "2026-01-01", "slug")
		if err != nil || got != "inbox/2026-01-01-slug.md" {
			t.Errorf("경로 = %q, %v, want inbox/2026-01-01-slug.md", got, err)
		}
	})
	t.Run("sources 문서는 연월 정밀도도 허용한다", func(t *testing.T) {
		got, err := FilePath(defaultCfg(t), StageSource, "2026-01", "slug")
		if err != nil || got != "sources/2026-01-slug.md" {
			t.Errorf("경로 = %q, %v, want sources/2026-01-slug.md", got, err)
		}
	})
	t.Run("잘못된 날짜 형식은 거절한다", func(t *testing.T) {
		if _, err := FilePath(defaultCfg(t), StageInbox, "2026/01", "slug"); err == nil {
			t.Error("YYYY-MM-DD나 YYYY-MM이 아니면 에러여야 함")
		}
	})

	t.Run("archive 단계는 파일 경로를 만들지 않는다", func(t *testing.T) {
		// archive는 파일명과 슬러그를 유지하는 커맨드 계약이다(ADR 0028).
		// FilePath가 archive 경로를 만들면 그 계약과 어긋난다.
		if _, err := FilePath(defaultCfg(t), StageArchive, "2026-01-01", "slug"); err == nil {
			t.Error("archive 단계는 FilePath 대상이 아니어야 함")
		}
	})
}

func TestStageForDir(t *testing.T) {
	t.Run("디렉토리의 단계는 stageDirs 표를 따른다", func(t *testing.T) {
		for dir, want := range map[string]Stage{
			"inbox":   StageInbox,
			"sources": StageSource,
			"context": StageContext,
			"archive": StageArchive,
		} {
			got, ok := StageForDir(dir)
			if !ok || got != want {
				t.Errorf("StageForDir(%q) = %q, %v, want %q, true", dir, got, ok, want)
			}
		}
	})

	t.Run("단계에 대응하지 않는 디렉토리는 false다", func(t *testing.T) {
		if _, ok := StageForDir("journal"); ok {
			t.Error("대응하지 않는 디렉토리는 false여야 함")
		}
	})
}

func TestCreate(t *testing.T) {
	t.Run("문서를 만들고 기존 파일을 덮어쓰지 않는다", func(t *testing.T) {
		dir := t.TempDir()
		path, err := Create(dir, defaultCfg(t), StageInbox, "2026-01-01", "memo",
			map[string]any{"type": "inbox-note", "tags": []string{}}, "메모 본문")
		if err != nil {
			t.Fatal(err)
		}
		wantPath := filepath.Join(dir, "inbox", "2026-01-01-memo.md")
		if path != wantPath {
			t.Errorf("경로 = %s, want %s", path, wantPath)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(raw)
		for _, want := range []string{"---\n", "type: inbox-note\n", "tags: []\n", "---\n메모 본문\n"} {
			if !strings.Contains(got, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, got)
			}
		}
		_, err = Create(dir, defaultCfg(t), StageInbox, "2026-01-01", "memo",
			map[string]any{"type": "inbox-note"}, "다시 만든 본문")
		if err == nil {
			t.Fatal("같은 경로의 재생성은 거절되어야 함")
		}
		for _, want := range []string{"이미 문서가 있습니다", "덮어쓰지 않습니다"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("거절 메시지에 %q 없음: %v", want, err)
			}
		}
		if got := string(mustRead(t, path)); strings.Contains(got, "다시 만든 본문") {
			t.Error("기존 파일이 덮어써졌음")
		}
	})

	t.Run("문서는 설정이 정한 단계 디렉토리에 만들어진다", func(t *testing.T) {
		dir := t.TempDir()
		cfg := defaultCfg(t)
		cfg.PageDirs = []string{"journal", "sources", "context"}
		path, err := Create(dir, cfg, StageSource, "2026-01-01", "memo",
			map[string]any{"type": "source-summary"}, "원본")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, "sources", "2026-01-01-memo.md"); path != want {
			t.Errorf("경로 = %s, want %s", path, want)
		}
		if _, err := Create(dir, cfg, StageInbox, "2026-01-01", "memo",
			map[string]any{"type": "inbox-note"}, "메모"); err == nil {
			t.Error("page_dirs에 inbox가 없으면 만들 수 없어야 함")
		} else if !strings.Contains(err.Error(), "page_dirs") {
			t.Errorf("에러 메시지에 설정 안내가 없음: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "inbox")); !os.IsNotExist(err) {
			t.Error("설정에 없는 디렉토리가 만들어졌음")
		}
	})
}

// defaultCfg는 설정 파일이 없는 기본 구성을 반환한다.
func defaultCfg(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// mustRead는 테스트 파일 읽기 실패를 치명적으로 처리한다.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestFrontmatter(t *testing.T) {
	t.Run("꺼진 축의 키를 아예 넣지 않는다", func(t *testing.T) {
		// personal 프리셋은 scope, sensitivity, source_channel, trigger_mode,
		// workflow를 끈다.
		cfg, err := config.Load(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		cfg.Preset = config.PresetPersonal
		cfg.Axes = map[config.Axis]bool{}
		for _, a := range []config.Axis{
			config.AxisType, config.AxisArtifactStage, config.AxisStatus, config.AxisIndexable,
			config.AxisTags, config.AxisSourceRefs, config.AxisDerivedFrom, config.AxisRelated,
		} {
			cfg.Axes[a] = true
		}
		fm := Frontmatter(StageSource, cfg)
		for _, key := range []string{"scope", "sensitivity", "source_channel", "trigger_mode", "workflow", "derived_context"} {
			if _, ok := fm[key]; ok {
				t.Errorf("꺼진 축의 키가 들어 있음: %s", key)
			}
		}
		if fm["status"] != "sourced" || fm["indexable"] != false {
			t.Errorf("source 단계 기본값이 잘못됨: %+v", fm)
		}
	})

	t.Run("켠 축은 단계별 기본값으로 채운다", func(t *testing.T) {
		cfg, err := config.Load(t.TempDir()) // 기본 education
		if err != nil {
			t.Fatal(err)
		}
		fm := Frontmatter(StageInbox, cfg)
		if fm["type"] != "inbox-note" || fm["artifact_stage"] != "inbox" ||
			fm["status"] != "inbox" || fm["indexable"] != false {
			t.Errorf("inbox 기본값이 잘못됨: %+v", fm)
		}
		if v, ok := fm["source_channel"]; !ok || v != nil {
			t.Errorf("source_channel은 빈 값 키여야 함: %v %v", v, ok)
		}
		if _, ok := fm["tags"]; ok {
			t.Error("inbox 단계에는 관계 필드가 없어야 함")
		}

		fm = Frontmatter(StageContext, cfg)
		if fm["status"] != "promoted" || fm["indexable"] != true {
			t.Errorf("context 기본값이 잘못됨: %+v", fm)
		}
		if v, ok := fm["related"]; !ok || !reflect.DeepEqual(v, []string{}) {
			t.Errorf("related는 빈 배열이어야 함: %v", v)
		}
		if _, ok := fm["updated"]; ok {
			t.Error("updated는 초기값에 없어야 함")
		}
	})

	t.Run("직렬화하면 계약 형식이 나온다", func(t *testing.T) {
		cfg, err := config.Load(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		got := string(doc.RenderMap(Frontmatter(StageSource, cfg), "본문"))
		for _, want := range []string{"source_channel:\n", "derived_context: []\n", "related: []\n"} {
			if !strings.Contains(got, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, got)
			}
		}
		if _, err := doc.Parse("x.md", []byte(got)); err != nil {
			t.Fatalf("초기값 프론트매터가 파싱되지 않음: %v", err)
		}
	})
}

func TestLinkable(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"context/b.md":  "x",
		"context/a.md":  "x",
		"inbox/c.md":    "x",
		"index.md":      "x",
		".engram/n.md":  "x",
		".hidden/h.md":  "x",
		"context/d.txt": "x",
	}
	for name := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Linkable(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b", "c", "index"}; !reflect.DeepEqual(got, want) {
		t.Errorf("링크 가능 슬러그 = %v, want %v", got, want)
	}
}
