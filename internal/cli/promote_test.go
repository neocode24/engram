package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/lint"
)

// runPromoteRoot는 promote와 new를 등록한 루트 커맨드를 실행한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 루트를 조립한다.
func runPromoteRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	root.AddCommand(newCaptureCmd())
	root.AddCommand(newSourceCmd())
	root.AddCommand(newPromoteCmd())
	root.AddCommand(newNewCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// captureMemo는 위키에 inbox 문서 하나를 넣고 그 상대 경로를 반환한다.
func captureMemo(t *testing.T, dir string) string {
	t.Helper()
	if _, err := runPromoteRoot(t, "capture", "--wiki", dir,
		"--now", "2026-01-01T00:00:00Z", "--slug", "memo", "메모 본문"); err != nil {
		t.Fatalf("capture 실패: %v", err)
	}
	return "inbox/2026-01-01-memo.md"
}

// addContextDocs는 게이트 대상이 충분하도록 context 문서를 직접 만든다.
func addContextDocs(t *testing.T, dir string) {
	t.Helper()
	body := "---\n" +
		"type: procedure\nartifact_stage: context\nstatus: promoted\n" +
		"indexable: true\nsource_refs: []\nderived_from: []\n" +
		"related:\n  - \"[[b-doc]]\"\nsource_channel: manual\nderived_context: []\n" +
		"---\n\n본문\n"
	files := map[string]string{"context/a-doc.md": body, "context/b-doc.md": body}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPromote(t *testing.T) {
	t.Run("inbox 문서는 이동합니다. 원본이 남지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err != nil {
			t.Fatalf("promote 실패: %v\n%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Error("inbox 원본이 남아 있음")
		}
		dest := filepath.Join(dir, "context", "memo.md")
		raw := readWikiFile(t, filepath.Dir(dest), filepath.Base(dest))
		for _, want := range []string{
			"artifact_stage: context\n", "status: promoted\n", "indexable: true\n", "메모 본문\n",
		} {
			if !strings.Contains(raw, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, raw)
			}
		}
		// 대상이 색인 하나뿐이므로 게이트는 유예로 통과하고 경고를 낸다.
		if !strings.Contains(out, "유예") {
			t.Errorf("유예 경고가 없음:\n%s", out)
		}
	})

	t.Run("sources 문서는 파생을 만들고 양방향으로 기록합니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runPromoteRoot(t, "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk", "--created", "2025-11-08", "원본 내용"); err != nil {
			t.Fatalf("source 실패: %v", err)
		}
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, "sources/2025-11-08-talk.md")
		if err != nil {
			t.Fatalf("promote 실패: %v\n%s", err, out)
		}
		derived := readWikiFile(t, filepath.Join(dir, "context"), "talk.md")
		if !strings.Contains(derived, "- sources/2025-11-08-talk.md\n") {
			t.Errorf("derived_from에 원본 경로가 없음:\n%s", derived)
		}
		if !strings.Contains(derived, "원본 내용\n") {
			t.Errorf("파생이 원본 본문을 가져야 함:\n%s", derived)
		}
		original := readWikiFile(t, filepath.Join(dir, "sources"), "2025-11-08-talk.md")
		if !strings.Contains(original, "derived_context:\n  - talk\n") {
			t.Errorf("원본 derived_context에 새 슬러그가 없음:\n%s", original)
		}
		if !strings.Contains(original, "derived_from: []\n") {
			t.Errorf("원본의 derived_from이 바뀜:\n%s", original)
		}
		if !strings.Contains(original, "원본 내용\n") {
			t.Errorf("원본 본문이 바뀜:\n%s", original)
		}
	})

	t.Run("derived_context 축이 꺼진 프리셋에서는 역방향을 생략합니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runPromoteRoot(t, "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk", "--created", "2025-11-08", "원본 내용"); err != nil {
			t.Fatal(err)
		}
		src := filepath.Join(dir, "sources", "2025-11-08-talk.md")
		before := readWikiFile(t, filepath.Dir(src), filepath.Base(src))
		if err := os.WriteFile(filepath.Join(dir, "engram.yaml"),
			[]byte("preset: minimal\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, "sources/2025-11-08-talk.md"); err != nil {
			t.Fatal(err)
		}
		after := readWikiFile(t, filepath.Dir(src), filepath.Base(src))
		if before != after {
			t.Errorf("역방향 기록을 생략해야 하는데 원본이 바뀜:\n%s\n===\n%s", before, after)
		}
	})

	t.Run("링크가 부족하면 부족분과 채우는 법을 냅니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		addContextDocs(t, dir) // 대상이 충분하므로 게이트가 엄격해진다
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err == nil {
			t.Fatal("게이트 거절이어야 함")
		}
		for _, want := range []string{"위키링크가 0개", "min_wikilinks 2개", "--related"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("거절 메시지에 %q 없음: %v", want, err)
			}
		}
	})

	t.Run("--related로 부족한 링크를 이 자리에서 채웁니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		addContextDocs(t, dir)
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, rel,
			"--related", "a-doc", "--related", "b-doc")
		if err != nil {
			t.Fatalf("채워서 통과해야 함: %v\n%s", err, out)
		}
		dest := readWikiFile(t, filepath.Join(dir, "context"), "memo.md")
		if !strings.Contains(dest, "- \"[[a-doc]]\"\n") || !strings.Contains(dest, "- \"[[b-doc]]\"\n") {
			t.Errorf("related에 준 슬러그가 없음:\n%s", dest)
		}
	})

	t.Run("존재하지 않는 --related 슬러그는 경고하되 막지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		addContextDocs(t, dir)
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, rel,
			"--related", "a-doc", "--related", "없는문서")
		if err != nil {
			t.Fatalf("경고는 거절이 아닙니다: %v", err)
		}
		if !strings.Contains(out, "경고: --related 슬러그") {
			t.Errorf("경고가 없음:\n%s", out)
		}
	})

	t.Run("이미 context 단계인 문서는 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		addContextDocs(t, dir)
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, "context/a-doc.md")
		if err == nil || !strings.Contains(err.Error(), "이미 context") {
			t.Fatalf("거절되어야 함: %v", err)
		}
	})

	t.Run("원본 파일명의 날짜 접두사를 뗍니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir) // inbox/2026-01-01-memo.md
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, rel); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "context", "memo.md")); err != nil {
			t.Fatalf("날짜 접두사가 없는 문서가 있어야 함: %v", err)
		}
	})

	t.Run("도착지에 같은 이름이 있으면 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		addContextDocs(t, dir)
		dest := filepath.Join(dir, "context", "memo.md")
		if err := os.WriteFile(dest, []byte("---\nartifact_stage: context\n---\n기존\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		rel := captureMemo(t, dir)
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel, "--related", "a-doc", "--related", "b-doc")
		if err == nil || !strings.Contains(err.Error(), "도착지에 이미 문서가 있습니다") {
			t.Fatalf("도착지 중복 거절이어야 함: %v", err)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init을 안내합니다", func(t *testing.T) {
		_, err := runPromoteRoot(t, "promote", "--wiki", t.TempDir(), "inbox/x.md")
		if err == nil || !strings.Contains(err.Error(), "engram init") {
			t.Fatalf("거절되어야 함: %v", err)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("허용값 밖의 type과 form은 목록과 함께 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		if err := os.WriteFile(filepath.Join(dir, "engram.yaml"),
			[]byte("forms: [note]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := runPromoteRoot(t, "new", "--wiki", dir, "--type", "diary", "제목")
		if err == nil || !strings.Contains(err.Error(), "허용값") {
			t.Fatalf("type 거절이어야 함: %v", err)
		}
		_, err = runPromoteRoot(t, "new", "--wiki", dir, "--form", "memo", "제목")
		if err == nil || !strings.Contains(err.Error(), "note") {
			t.Fatalf("form 거절이 허용값을 담아야 함: %v", err)
		}
	})

	t.Run("본문은 승급 문서의 절 골격을 담습니다", func(t *testing.T) {
		dir := initWiki(t)
		if err := os.WriteFile(filepath.Join(dir, "engram.yaml"),
			[]byte("forms: [note]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runPromoteRoot(t, "new", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "skeleton",
			"--type", "concept", "--form", "note",
			"--topics", "go", "--tags", "t1", "제목"); err != nil {
			t.Fatal(err)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "skeleton.md")
		for _, want := range []string{
			"# 제목\n", "## 결론\n", "## 맥락\n", "## 현재 이해\n", "## 근거\n", "## 관련 링크\n",
			"type: concept\n", "form: note\n", "created: 2026-01-01\n",
			"topics:\n  - go\n", "tags:\n  - t1\n",
		} {
			if !strings.Contains(raw, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, raw)
			}
		}
	})

	t.Run("--json은 경로와 슬러그와 게이트 판정을 냅니다", func(t *testing.T) {
		dir := initWiki(t)
		out, err := runPromoteRoot(t, "new", "--json", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "json-doc", "제목")
		if err != nil {
			t.Fatalf("new 실패: %v\n%s", err, out)
		}
		var res writeOutcome
		// 경고 문구가 JSON 앞에 같은 버퍼에 섞일 수 있으므로 본문부터 파싱한다.
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Slug != "json-doc" || res.Stage != "context" {
			t.Errorf("JSON 내용이 잘못됨: %+v", res)
		}
		if !res.Gate.Passed || !res.Gate.Deferred || res.Gate.Min != 2 {
			t.Errorf("게이트 판정이 잘못됨: %+v", res.Gate)
		}
	})

	t.Run("--now를 고정하면 두 번 실행 결과가 바이트까지 같습니다", func(t *testing.T) {
		a, b := initWiki(t), initWiki(t)
		args := []string{"new", "--wiki", a, "--now", "2026-01-01T00:00:00Z",
			"--slug", "same", "--type", "concept", "제목"}
		if _, err := runPromoteRoot(t, args...); err != nil {
			t.Fatal(err)
		}
		args[2] = b
		if _, err := runPromoteRoot(t, args...); err != nil {
			t.Fatal(err)
		}
		x := readWikiFile(t, filepath.Join(a, "context"), "same.md")
		y := readWikiFile(t, filepath.Join(b, "context"), "same.md")
		if x != y {
			t.Fatalf("두 결과가 다름:\n%s\n===\n%s", x, y)
		}
	})
}

// lintMissingField는 위키에서 frontmatter.missing-field 위반만 모은다.
// promote와 new의 산출물이 제품 자신의 검사를 통과하는지 본다.
func lintMissingField(t *testing.T, dir string) []lint.Violation {
	t.Helper()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := lint.Run(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out []lint.Violation
	for _, v := range res.Violations {
		if v.Rule == "frontmatter.missing-field" {
			out = append(out, v)
		}
	}
	return out
}

func TestPromoteFillsContextFields(t *testing.T) {
	t.Run("promote 후 문서는 context 필수 필드를 모두 갖습니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, rel); err != nil {
			t.Fatal(err)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "memo.md")
		for _, want := range []string{
			"source_refs: []\n", "derived_from: []\n", "related: []\n",
			"tags: []\n", "derived_context: []\n", "source_channel:\n",
		} {
			if !strings.Contains(raw, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, raw)
			}
		}
		if vs := lintMissingField(t, dir); len(vs) != 0 {
			t.Fatalf("promote 산출물이 lint를 통과해야 함: %+v", vs)
		}
	})

	t.Run("이미 값이 있는 필드는 보존합니다", func(t *testing.T) {
		dir := initWiki(t)
		p := filepath.Join(dir, "inbox", "2026-01-01-rich.md")
		content := "---\n" +
			"type: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n" +
			"tags:\n  - go\n" +
			"---\n\n본문\n"
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, "inbox/2026-01-01-rich.md"); err != nil {
			t.Fatal(err)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "rich.md")
		if !strings.Contains(raw, "tags:\n  - go\n") {
			t.Errorf("기존 tags 값이 보존되지 않음:\n%s", raw)
		}
	})

	t.Run("꺼진 축의 필드는 추가하지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		if err := os.WriteFile(filepath.Join(dir, "engram.yaml"),
			[]byte("preset: minimal\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "inbox", "2026-01-01-plain.md")
		content := "---\ntype: inbox-note\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n---\n\n본문\n"
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, "inbox/2026-01-01-plain.md"); err != nil {
			t.Fatal(err)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "plain.md")
		for _, key := range []string{"scope:", "sensitivity:", "source_channel:", "trigger_mode:", "workflow:", "derived_context:"} {
			if strings.Contains(raw, key) {
				t.Errorf("꺼진 축의 키 %q가 들어 있음:\n%s", key, raw)
			}
		}
	})

	t.Run("sources 파생 문서도 context 필수 필드를 갖습니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runPromoteRoot(t, "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk", "--created", "2025-11-08", "원본"); err != nil {
			t.Fatal(err)
		}
		if _, err := runPromoteRoot(t, "promote", "--wiki", dir, "sources/2025-11-08-talk.md"); err != nil {
			t.Fatal(err)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "talk.md")
		for _, want := range []string{"source_refs: []\n", "related: []\n", "tags: []\n", "derived_context: []\n"} {
			if !strings.Contains(raw, want) {
				t.Errorf("내용에 %q 없음:\n%s", want, raw)
			}
		}
		if vs := lintMissingField(t, dir); len(vs) != 0 {
			t.Fatalf("파생 산출물이 lint를 통과해야 함: %+v", vs)
		}
	})

	t.Run("new 산출물도 frontmatter.missing-field가 없습니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runPromoteRoot(t, "new", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "born", "제목"); err != nil {
			t.Fatal(err)
		}
		if vs := lintMissingField(t, dir); len(vs) != 0 {
			t.Fatalf("new 산출물이 lint를 통과해야 함: %+v", vs)
		}
	})
}

func TestPromoteType(t *testing.T) {
	t.Run("--type로 지정한 문서 종류를 반영합니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, rel, "--type", "concept")
		if err != nil {
			t.Fatalf("promote 실패: %v\n%s", err, out)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "memo.md")
		if !strings.Contains(raw, "type: concept\n") {
			t.Errorf("type이 반영되지 않음:\n%s", raw)
		}
		if strings.Contains(out, "경고: 문서 종류가") {
			t.Errorf("지정했으면 경고가 없어야 함:\n%s", out)
		}
	})

	t.Run("허용값 밖의 --type은 목록과 함께 거절합니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		_, err := runPromoteRoot(t, "promote", "--wiki", dir, rel, "--type", "diary")
		if err == nil || !strings.Contains(err.Error(), "허용값") {
			t.Fatalf("거절되어야 함: %v", err)
		}
		if !strings.Contains(err.Error(), "concept") {
			t.Errorf("거절 메시지에 허용값 목록이 없음: %v", err)
		}
	})

	t.Run("미지정이고 기존 값이 inbox-note면 경고합니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir) // capture는 type을 inbox-note로 만든다
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, rel)
		if err != nil {
			t.Fatalf("경고는 거절이 아닙니다: %v", err)
		}
		for _, want := range []string{"경고: 문서 종류가", "inbox-note", "--type"} {
			if !strings.Contains(out, want) {
				t.Errorf("경고에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("미지정이고 기존 값이 적절하면 경고하지 않습니다", func(t *testing.T) {
		dir := initWiki(t)
		p := filepath.Join(dir, "inbox", "2026-01-01-typed.md")
		content := "---\ntype: procedure\nartifact_stage: inbox\nstatus: inbox\nindexable: false\n---\n\n본문\n"
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, "inbox/2026-01-01-typed.md")
		if err != nil {
			t.Fatalf("promote 실패: %v", err)
		}
		if strings.Contains(out, "경고: 문서 종류가") {
			t.Errorf("적절한 값에는 경고가 없어야 함:\n%s", out)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "typed.md")
		if !strings.Contains(raw, "type: procedure\n") {
			t.Errorf("기존 종류가 보존되지 않음:\n%s", raw)
		}
	})

	t.Run("sources 파생 경로에서도 같은 규칙을 적용합니다", func(t *testing.T) {
		dir := initWiki(t)
		if _, err := runPromoteRoot(t, "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk", "--created", "2025-11-08", "원본"); err != nil {
			t.Fatal(err)
		}
		out, err := runPromoteRoot(t, "promote", "--wiki", dir, "sources/2025-11-08-talk.md")
		if err != nil {
			t.Fatalf("경고는 거절이 아닙니다: %v", err)
		}
		if !strings.Contains(out, "source-summary") {
			t.Errorf("source 단계 기본값 경고가 없음:\n%s", out)
		}
		out, err = runPromoteRoot(t, "source", "--wiki", dir,
			"--now", "2026-01-01T00:00:00Z", "--slug", "talk2", "--created", "2025-11-08", "원본")
		if err != nil {
			t.Fatal(err)
		}
		out, err = runPromoteRoot(t, "promote", "--wiki", dir, "sources/2025-11-08-talk2.md",
			"--type", "source-summary", "--related", "talk", "--related", "index")
		if err != nil {
			t.Fatalf("promote 실패: %v\n%s", err, out)
		}
		if strings.Contains(out, "경고: 문서 종류가") {
			t.Errorf("명시적으로 지정했으면 경고가 없어야 함:\n%s", out)
		}
		raw := readWikiFile(t, filepath.Join(dir, "context"), "talk2.md")
		if !strings.Contains(raw, "type: source-summary\n") {
			t.Errorf("--type 값이 반영되지 않음:\n%s", raw)
		}
	})

	t.Run("--json에 문서 종류가 반영됩니다", func(t *testing.T) {
		dir := initWiki(t)
		rel := captureMemo(t, dir)
		out, err := runPromoteRoot(t, "promote", "--json", "--wiki", dir, rel, "--type", "concept")
		if err != nil {
			t.Fatalf("promote 실패: %v\n%s", err, out)
		}
		var res promoteOutcome
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Type != "concept" || res.Slug != "memo" || res.Stage != "context" {
			t.Errorf("JSON 내용이 잘못됨: %+v", res)
		}
	})
}
