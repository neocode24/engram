package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runExport은 export 커맨드를 루트 등록 없이 시험한다. 전역 플래그는 테스트용
// 부모 커맨드에 붙인다.
func runExport(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.AddCommand(newExportCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeExportWiki는 반출 대상이 있는 임시 위키를 만든다.
func makeExportWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"engram.yaml": "preset: team\n",
		"context/아크미-현황.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"sensitivity: internal\nindexable: true\ncreated: 2026-01-01\n---\n\n" +
			"# 사내 현황\n\n아크미 이야기입니다.\n",
		"context/secret.md": "---\ntype: concept\nartifact_stage: context\nstatus: promoted\n" +
			"sensitivity: private-local-only\nindexable: true\ncreated: 2026-01-01\n---\n\n" +
			"# 로컬 전용\n\n나가면 안 됩니다.\n",
		"inbox/2026-08-01-rough.md": "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\n" +
			"indexable: false\ncreated: 2026-08-01\n---\n\n# 러프\n\n미검수입니다.\n",
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
	return root
}

// bundleNames는 번들 디렉토리의 파일 상대 경로를 모은다.
func bundleNames(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestExportWritesBundle(t *testing.T) {
	root := makeExportWiki(t)
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal")
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	names := bundleNames(t, out)
	if len(names) != 1 || names[0] != "context/아크미-현황.md" {
		t.Fatalf("번들 = %v, 기대 context/아크미-현황.md 하나", names)
	}
	if !strings.Contains(stdout, "민감도") {
		t.Errorf("민감도 안내가 없습니다:\n%s", stdout)
	}
	if !strings.Contains(stdout, "치환 파일을 주지 않아") {
		t.Errorf("치환하지 않았다는 안내가 없습니다:\n%s", stdout)
	}
}

func TestExportDryRunWritesNothing(t *testing.T) {
	root := makeExportWiki(t)
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run 실패: %v\n%s", err, stdout)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("dry-run 인데 출력 디렉토리를 만들었습니다")
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Errorf("dry-run 안내가 없습니다:\n%s", stdout)
	}
}

func TestExportRejectsNonEmptyOutDir(t *testing.T) {
	root := makeExportWiki(t)
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "잔재.md"), []byte("이전 반출물\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal")
	if err == nil || !strings.Contains(err.Error(), "비어 있지 않습니다") {
		t.Fatalf("비어 있지 않은 디렉토리 오류 = %v", err)
	}
	if names := bundleNames(t, out); len(names) != 1 {
		t.Errorf("거절했는데 파일을 썼습니다: %v", names)
	}
}

func TestExportAppliesReplacements(t *testing.T) {
	root := makeExportWiki(t)
	dict := filepath.Join(t.TempDir(), "repl.txt")
	if err := os.WriteFile(dict, []byte("# 사전\n아크미==>사내조직\n안걸리는말==>X\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal", "--replacements", dict)
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	names := bundleNames(t, out)
	if len(names) != 1 || names[0] != "context/사내조직-현황.md" {
		t.Fatalf("파일명 치환 실패: %v", names)
	}
	body, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(names[0])))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "아크미") {
		t.Error("본문에 원문이 남았습니다")
	}
	if !strings.Contains(stdout, "한 번도 걸리지 않은 치환 규칙") {
		t.Errorf("미사용 규칙 경고가 없습니다:\n%s", stdout)
	}
}

func TestExportRejectsBadReplacementFile(t *testing.T) {
	root := makeExportWiki(t)
	dict := filepath.Join(t.TempDir(), "repl.txt")
	if err := os.WriteFile(dict, []byte("구분자가 없는 줄\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runExport(t, "export", "--wiki", root, "--out", filepath.Join(t.TempDir(), "b"), "--replacements", dict)
	if err == nil || !strings.Contains(err.Error(), "==>") {
		t.Fatalf("형식 오류를 알리지 않았습니다: %v", err)
	}
}

func TestExportRequiresOut(t *testing.T) {
	root := makeExportWiki(t)
	_, err := runExport(t, "export", "--wiki", root)
	if err == nil || !strings.Contains(err.Error(), "--out") {
		t.Fatalf("--out 없이 통과했습니다: %v", err)
	}
}

func TestExportJSON(t *testing.T) {
	root := makeExportWiki(t)
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	var o exportOutcome
	if err := json.Unmarshal([]byte(stdout), &o); err != nil {
		t.Fatalf("JSON 파싱 실패: %v\n%s", err, stdout)
	}
	if !o.DryRun || len(o.Files) != 1 {
		t.Errorf("결과가 다릅니다: %+v", o)
	}
	if o.Excluded.Sensitive != 1 || o.Excluded.Inbox != 1 {
		t.Errorf("제외 집계 = %+v, 기대 민감도 1 inbox 1", o.Excluded)
	}
	if o.Anonymized {
		t.Error("치환 파일이 없는데 익명화했다고 보고했습니다")
	}
}

// makeBoundaryWiki는 ADR 0063이 더한 판정 셋을 한 위키에 모은다.
// 공개 문서 하나, internal 하나, indexable 이 false 인 것 하나, status 가
// superseded 인 것 하나다.
func makeBoundaryWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	head := "---\ntype: concept\nartifact_stage: context\n"
	files := map[string]string{
		"engram.yaml": "preset: team\n",
		"context/공개.md": head + "status: promoted\nindexable: true\nsensitivity: public-reference\n" +
			"created: 2026-01-01\n---\n\n# 공개 문서\n\n나갑니다.\n",
		"context/사내.md": head + "status: promoted\nindexable: true\nsensitivity: internal\n" +
			"created: 2026-01-01\n---\n\n# 사내 문서\n\n기본으로 빠집니다.\n",
		"context/색인제외.md": head + "status: promoted\nindexable: false\nsensitivity: public-reference\n" +
			"created: 2026-01-01\n---\n\n# 색인 제외\n\n빠집니다.\n",
		"context/대체됨.md": head + "status: superseded\nindexable: true\nsensitivity: public-reference\n" +
			"created: 2026-01-01\n---\n\n# 대체된 문서\n\n빠집니다.\n",
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
	return root
}

func TestExportExcludesInternalByDefault(t *testing.T) {
	root := makeBoundaryWiki(t)
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out)
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	names := bundleNames(t, out)
	if len(names) != 1 || names[0] != "context/공개.md" {
		t.Fatalf("번들 = %v, 기대 context/공개.md 하나", names)
	}
	// 빠진 것을 반드시 말한다.
	for _, want := range []string{
		"민감도가 internal 인 문서 1개",
		"--include-internal",
		"indexable 이 false 인 문서 1개",
		"status 가 superseded 인 문서 1개",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("출력에 %q 가 없습니다:\n%s", want, stdout)
		}
	}
}

func TestExportIncludeInternalOpensAndSaysSo(t *testing.T) {
	root := makeBoundaryWiki(t)
	out := filepath.Join(t.TempDir(), "bundle")
	stdout, err := runExport(t, "export", "--wiki", root, "--out", out, "--include-internal")
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	names := bundleNames(t, out)
	if len(names) != 2 {
		t.Fatalf("번들 = %v, 기대 2개", names)
	}
	if !strings.Contains(stdout, "민감도가 internal 인 문서 1개를 포함했습니다") {
		t.Errorf("넓힌 사실이 출력에 없습니다:\n%s", stdout)
	}
}

func TestExportBoundaryCountsInJSON(t *testing.T) {
	root := makeBoundaryWiki(t)
	stdout, err := runExport(t, "export", "--wiki", root,
		"--out", filepath.Join(t.TempDir(), "bundle"), "--dry-run", "--json")
	if err != nil {
		t.Fatalf("반출 실패: %v\n%s", err, stdout)
	}
	var o exportOutcome
	if err := json.Unmarshal([]byte(stdout), &o); err != nil {
		t.Fatalf("JSON 파싱 실패: %v\n%s", err, stdout)
	}
	if o.Excluded.Internal != 1 || o.Excluded.NotIndexable != 1 || o.Excluded.Superseded != 1 {
		t.Errorf("제외 집계 = %+v, 기대 internal/notIndexable/superseded 각 1", o.Excluded)
	}
	if o.IncludeInternal || o.IncludedInternal != 0 {
		t.Errorf("포함 집계 = %v/%d, 기대 false/0", o.IncludeInternal, o.IncludedInternal)
	}
}
