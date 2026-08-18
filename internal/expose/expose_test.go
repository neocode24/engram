package expose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neocode24/engram/internal/config"
)

// doc는 context 문서 하나의 원문을 만든다. status 와 indexable 과
// sensitivity 를 시험마다 다르게 준다. 빈 문자열이면 그 키를 넣지 않는다.
func contextDoc(status, indexable, sensitivity string) string {
	fm := "---\ntype: concept\nartifact_stage: context\n"
	if status != "" {
		fm += "status: " + status + "\n"
	}
	if indexable != "" {
		fm += "indexable: " + indexable + "\n"
	}
	if sensitivity != "" {
		fm += "sensitivity: " + sensitivity + "\n"
	}
	fm += "created: 2026-01-01\nupdated: 2026-08-01\n---\n\n# 문서\n\n본문입니다.\n"
	return fm
}

// makeWiki는 노출 판정을 시험할 위키를 만든다. team 프리셋은 sensitivity
// 속성을 켜고 personal 은 끈다.
func makeWiki(t *testing.T, preset string) (string, config.Config) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml":            "preset: " + preset + "\n",
		"context/열림.md":          contextDoc("promoted", "true", "public-reference"),
		"context/색인제외.md":        contextDoc("promoted", "false", "public-reference"),
		"context/대체됨.md":         contextDoc("superseded", "true", "public-reference"),
		"context/사내.md":          contextDoc("promoted", "true", "internal"),
		"context/로컬전용.md":        contextDoc("promoted", "true", "private-local-only"),
		"context/축없음.md":         contextDoc("promoted", "", ""),
		"archive/보관.md":          contextDoc("archived", "true", "public-reference"),
		"inbox/2026-08-01-러프.md": contextDoc("inbox", "false", "private-local-only"),
	}
	for name, content := range files {
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

// visible은 노출된 문서의 상대 경로 집합을 만든다.
func visible(t *testing.T, root string, cfg config.Config, opts Options) (map[string]bool, Exposure) {
	t.Helper()
	res, err := Select(root, cfg, opts)
	if err != nil {
		t.Fatalf("노출 판정 실패: %v", err)
	}
	out := map[string]bool{}
	for _, d := range res.Docs {
		out[d.Rel] = true
	}
	return out, res.Exposure
}

func TestSelectExcludesNotIndexable(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	got, e := visible(t, root, cfg, Options{})
	if got["context/색인제외.md"] {
		t.Error("indexable 이 false 인 문서가 노출되었습니다")
	}
	if e.ExcludedNotIndexable != 1 {
		t.Errorf("indexable 제외 = %d, 기대 1", e.ExcludedNotIndexable)
	}
	// 값을 안 적은 문서는 거르지 않는다.
	if !got["context/축없음.md"] {
		t.Error("indexable 을 안 적은 문서를 걸렀습니다")
	}
}

func TestSelectIndexableAxisOff(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	cfg.Axes[config.AxisIndexable] = false
	got, e := visible(t, root, cfg, Options{})
	if !got["context/색인제외.md"] {
		t.Error("indexable 축이 꺼진 위키인데 그 값으로 걸렀습니다")
	}
	if e.ExcludedNotIndexable != 0 {
		t.Errorf("축이 꺼졌는데 indexable 제외 = %d", e.ExcludedNotIndexable)
	}
}

func TestSelectExcludesSuperseded(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	got, e := visible(t, root, cfg, Options{})
	if got["context/대체됨.md"] {
		t.Error("status 가 superseded 인 문서가 노출되었습니다")
	}
	if e.ExcludedSuperseded != 1 {
		t.Errorf("superseded 제외 = %d, 기대 1", e.ExcludedSuperseded)
	}
}

func TestSelectSupersededIsNotAnAxis(t *testing.T) {
	// status 는 축이 아니라 필수 필드다. 축을 꺼도 판정이 걸린다.
	root, cfg := makeWiki(t, "team")
	cfg.Axes[config.AxisStatus] = false
	got, _ := visible(t, root, cfg, Options{})
	if got["context/대체됨.md"] {
		t.Error("status 축을 껐다고 superseded 문서를 노출했습니다")
	}
}

func TestSelectInternalIsClosedByDefaultAndOpensWithOption(t *testing.T) {
	root, cfg := makeWiki(t, "team")

	closed, ce := visible(t, root, cfg, Options{})
	if closed["context/사내.md"] {
		t.Error("internal 문서가 기본으로 노출되었습니다")
	}
	if ce.ExcludedInternal != 1 || ce.IncludedInternal != 0 {
		t.Errorf("internal 집계 = 제외 %d 포함 %d, 기대 제외 1 포함 0",
			ce.ExcludedInternal, ce.IncludedInternal)
	}

	open, oe := visible(t, root, cfg, Options{IncludeInternal: true})
	if !open["context/사내.md"] {
		t.Error("IncludeInternal 인데 internal 문서가 빠졌습니다")
	}
	if oe.ExcludedInternal != 0 || oe.IncludedInternal != 1 {
		t.Errorf("internal 집계 = 제외 %d 포함 %d, 기대 제외 0 포함 1",
			oe.ExcludedInternal, oe.IncludedInternal)
	}
	// 뒤집을 수 없는 제외는 그대로다.
	if open["context/로컬전용.md"] {
		t.Error("IncludeInternal 이 private-local-only 제외까지 뚫었습니다")
	}
	if oe.ExcludedSensitive != 1 {
		t.Errorf("민감도 제외 = %d, 기대 1", oe.ExcludedSensitive)
	}
}

func TestSelectInternalNotHidden(t *testing.T) {
	// internal 은 플래그로 열리므로 뒤집을 수 없는 제외 목록에 없다(ADR 0063).
	if Hidden(SensitivityInternal) {
		t.Error("HiddenSensitivities 에 internal 이 들어갔습니다")
	}
}

func TestSelectSensitivityAxisOffKeepsInternal(t *testing.T) {
	// personal 프리셋은 sensitivity 속성이 꺼져 있어 거를 값이 없다.
	root, cfg := makeWiki(t, "personal")
	got, e := visible(t, root, cfg, Options{})
	if !got["context/사내.md"] {
		t.Error("축이 꺼진 위키인데 internal 로 걸렀습니다")
	}
	if e.ExcludedInternal != 0 {
		t.Errorf("축이 꺼졌는데 internal 제외 = %d", e.ExcludedInternal)
	}
}

func TestSelectCountsByReason(t *testing.T) {
	root, cfg := makeWiki(t, "team")
	got, e := visible(t, root, cfg, Options{})
	if len(got) != 2 || !got["context/열림.md"] || !got["context/축없음.md"] {
		t.Fatalf("노출 문서 = %v, 기대 열림/축없음 둘", got)
	}
	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"inbox", e.ExcludedInbox, 1},
		{"archive", e.ExcludedArchive, 1},
		{"sensitive", e.ExcludedSensitive, 1},
		{"internal", e.ExcludedInternal, 1},
		{"not_indexable", e.ExcludedNotIndexable, 1},
		{"superseded", e.ExcludedSuperseded, 1},
		{"context", e.Context, 2},
	} {
		if c.got != c.want {
			t.Errorf("%s 집계 = %d, 기대 %d", c.name, c.got, c.want)
		}
	}
}

func TestReasonTextCoversNewReasons(t *testing.T) {
	for _, r := range []string{ReasonInternal, ReasonNotIndexable, ReasonSuperseded} {
		if got := ReasonText(r); got == "" || got == ReasonText(ReasonNone) {
			t.Errorf("%s 사유 문장이 없습니다: %q", r, got)
		}
	}
}
