package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runBacklinksRoot는 backlinks를 등록한 루트 커맨드를 실행한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 루트를 조립한다.
func runBacklinksRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	root.AddCommand(newBacklinksCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// backlinksWiki는 백링크 대상 위키를 만들고 그 경로를 반환한다.
func backlinksWiki(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wiki")
	if _, err := runBacklinksRoot(t, "init", dir); err != nil {
		t.Fatalf("init 실패: %v", err)
	}
	files := map[string]string{
		"context/hub.md": "---\n" +
			"type: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\nrelated: []\n" +
			"source_channel: manual\nderived_context: []\n" +
			"---\n\n# 중심 문서\n\n본문에서 [[게이트웨이]] 를 가리킨다.",
		"context/peer.md": "---\n" +
			"type: concept\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\nderived_from: []\n" +
			"related:\n  - \"[[게이트웨이]]\"\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n# 동료 문서",
		"context/derived.md": "---\n" +
			"type: source-summary\nartifact_stage: context\nstatus: promoted\n" +
			"indexable: true\nsource_refs: []\n" +
			"derived_from:\n  - sources/2026-01-08-게이트웨이.md\n" +
			"related: []\nsource_channel: manual\nderived_context: []\n" +
			"---\n\n# 파생 문서",
		"sources/2026-01-08-게이트웨이.md": "---\n" +
			"type: source-summary\nartifact_stage: source\nstatus: sourced\n" +
			"indexable: false\nsource_refs: []\nderived_from: []\nderived_context: []\n" +
			"related: []\ntags: []\nsource_channel: web\n" +
			"---\n\n# 원본",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBacklinksCmd(t *testing.T) {
	t.Run("링크 종류를 구분해 낸다", func(t *testing.T) {
		dir := backlinksWiki(t)
		out, err := runBacklinksRoot(t, "backlinks", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("backlinks 실패: %v\n%s", err, out)
		}
		for _, want := range []string{
			"context/hub.md", "본문 링크", "context/peer.md", "related 필드",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
	})

	t.Run("경로 형태 질의도 정규화해 잡는다", func(t *testing.T) {
		dir := backlinksWiki(t)
		out, err := runBacklinksRoot(t, "backlinks", "--wiki", dir, "sources/2026-01-08-게이트웨이.md")
		if err != nil {
			t.Fatalf("backlinks 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "derived_from 필드") || !strings.Contains(out, "context/derived.md") {
			t.Errorf("경로 질의가 관계 필드를 잡아야 함:\n%s", out)
		}
	})

	t.Run("문서가 없으면 깨진 링크임을 알린다", func(t *testing.T) {
		dir := backlinksWiki(t)
		// 위키에는 없는 슬러그를 hub 본문에서 가리키게 만든다.
		hub := filepath.Join(dir, "context", "hub.md")
		raw, err := os.ReadFile(hub)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(hub, append(raw, []byte("\n[[없는문서]] 링크")...), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runBacklinksRoot(t, "backlinks", "--wiki", dir, "없는문서")
		if err != nil {
			t.Fatalf("조회는 판정이 아니다: %v", err)
		}
		if !strings.Contains(out, "문서가 위키에 없다") || !strings.Contains(out, "context/hub.md") {
			t.Errorf("깨진 링크 안내가 없음:\n%s", out)
		}
	})

	t.Run("백링크가 없으면 고아 가능성을 알린다", func(t *testing.T) {
		dir := backlinksWiki(t)
		out, err := runBacklinksRoot(t, "backlinks", "--wiki", dir, "nobody-links-this")
		if err != nil {
			t.Fatalf("결과 없음은 종료 코드 0이어야 함: %v", err)
		}
		if !strings.Contains(out, "백링크가 없다") || !strings.Contains(out, "고아 문서") {
			t.Errorf("고아 안내가 없음:\n%s", out)
		}
	})

	t.Run("--json은 종류와 줄과 원본 값을 낸다", func(t *testing.T) {
		dir := backlinksWiki(t)
		out, err := runBacklinksRoot(t, "backlinks", "--json", "--wiki", dir, "게이트웨이")
		if err != nil {
			t.Fatalf("backlinks 실패: %v\n%s", err, out)
		}
		var res backlinksResponse
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Slug != "게이트웨이" || !res.Exists || len(res.Backlinks) != 3 {
			t.Fatalf("JSON 내용이 잘못됨: %+v", res)
		}
		for _, b := range res.Backlinks {
			if b.Path == "" || b.Line == 0 || b.Field == "" || b.Raw == "" {
				t.Errorf("백링크 필드가 비어 있음: %+v", b)
			}
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init을 안내한다", func(t *testing.T) {
		_, err := runBacklinksRoot(t, "backlinks", "--wiki", t.TempDir(), "x")
		if err == nil || !strings.Contains(err.Error(), "engram init") {
			t.Fatalf("거절되어야 함: %v", err)
		}
	})
}
