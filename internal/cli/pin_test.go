package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runEngram은 등록이 끝난 루트 커맨드를 실행한다.
func runEngram(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
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

func newPinWiki(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wiki")
	if _, err := runEngram(t, "", "init", dir); err != nil {
		t.Fatal(err)
	}
	if out, err := runEngram(t, "", "pin", "init", "--wiki", dir); err != nil {
		t.Fatalf("pin init: %v\n%s", err, out)
	}
	return dir
}

func writeDecl(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "decl.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const memoBody = "김대리가 금요일까지 견적서를 보낸다."

func addDecl(quote string) string {
	return `{"requests":[{"op":"add","args":{"title":"견적서 보내기","evidence":[{"section":"","quote":"` + quote + `"}]}}]}`
}

const nowFlag = "2026-11-22T10:00:00+09:00"

func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() != ".lock" {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

func TestPinDisabledWikiBehavesLikeBefore(t *testing.T) {
	dir := initWiki(t)
	// Pin 을 켜지 않은 위키는 --pin 을 보지 않는다. 주어도 받지 않는다.
	for _, args := range [][]string{
		{"capture", "--wiki", dir, "--now", nowFlag, "메모"},
		{"capture", "--wiki", dir, "--now", nowFlag, "--pin", "/없는/파일.json", "다른 메모"},
		{"source", "--wiki", dir, "--now", nowFlag, "--title", "자료", "내용"},
	} {
		out, err := runEngram(t, "", args...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		if strings.Contains(out, "Pin:") {
			t.Errorf("%v: 켜지 않은 위키의 출력에 Pin 이 나옵니다:\n%s", args, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "pin")); err == nil {
		t.Fatal("Pin 을 켜지 않았는데 pin/ 이 생겼습니다")
	}
	out, err := runEngram(t, "", "capture", "--wiki", dir, "--now", nowFlag, "--json", "세번째")
	if err != nil || strings.Contains(out, `"pin"`) {
		t.Errorf("--json 출력에 pin 칸이 생겼습니다: %v\n%s", err, out)
	}
}

func TestPinEnabledWikiRequiresDeclaration(t *testing.T) {
	dir := newPinWiki(t)
	before := listFiles(t, dir)
	for _, args := range [][]string{
		{"capture", "--wiki", dir, "--now", nowFlag, "메모"},
		{"source", "--wiki", dir, "--now", nowFlag, "--title", "자료", "내용"},
	} {
		out, err := runEngram(t, "", args...)
		if err == nil || !strings.Contains(err.Error(), "--pin") {
			t.Errorf("%v: --pin 이 필요하다는 에러가 아닙니다: %v\n%s", args, err, out)
		}
	}
	if after := listFiles(t, dir); strings.Join(after, "|") != strings.Join(before, "|") {
		t.Fatalf("선언이 없는데 파일이 생겼습니다: %v", after)
	}
}

func TestCaptureWithDeclarationWritesDocumentAndLedger(t *testing.T) {
	dir := newPinWiki(t)
	decl := writeDecl(t, addDecl(memoBody))
	out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "P0001") || !strings.Contains(out, "Q0001") {
		t.Errorf("새 일과 제안이 출력되지 않았습니다:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "inbox", "2026-11-22-메모.md")); err != nil {
		t.Fatal("문서가 없습니다")
	}
	if _, err := os.Stat(filepath.Join(dir, "pin", "items", "P0001.json")); err != nil {
		t.Fatal("원장이 없습니다")
	}
	list, err := runEngram(t, "", "pin", "list", "--wiki", dir, "--json")
	if err != nil || !strings.Contains(list, `"견적서 보내기"`) {
		t.Errorf("pin list: %v\n%s", err, list)
	}
}

func TestCaptureRejectedDeclarationWritesNothing(t *testing.T) {
	dir := newPinWiki(t)
	before := listFiles(t, dir)
	decl := writeDecl(t, addDecl("본문에 없는 문장입니다"))
	out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl)
	if err == nil || !strings.Contains(err.Error(), "quote_missing") {
		t.Fatalf("거절 사유가 에러에 없습니다: %v\n%s", err, out)
	}
	if after := listFiles(t, dir); strings.Join(after, "|") != strings.Join(before, "|") {
		t.Fatalf("거절됐는데 파일이 바뀌었습니다: %v", after)
	}
}

func TestPinNoneLeavesALogLine(t *testing.T) {
	dir := newPinWiki(t)
	out, err := runEngram(t, "", "capture", "--wiki", dir, "--now", nowFlag, "--pin", "none", "할 일 없는 메모")
	if err != nil || !strings.Contains(out, "할 일이 없다고") {
		t.Fatalf("%v\n%s", err, out)
	}
	logs, _ := filepath.Glob(filepath.Join(dir, "pin", "log", "*.jsonl"))
	if len(logs) != 1 {
		t.Fatal("none 이 pin/log 에 남지 않았습니다")
	}
}

func TestPinDeclarationFromStdinConflictsWithBody(t *testing.T) {
	// --pin - 와 본문 표준 입력은 같은 입력을 두 번 읽을 수 없다.
	dir := newPinWiki(t)
	before := listFiles(t, dir)
	_, err := runEngram(t, addDecl(memoBody), "capture", "--wiki", dir, "--now", nowFlag, "--pin", "-")
	if err == nil {
		t.Fatal("본문이 없는데 성공했습니다")
	}
	if after := listFiles(t, dir); strings.Join(after, "|") != strings.Join(before, "|") {
		t.Fatalf("파일이 생겼습니다: %v", after)
	}
}

func TestPromoteToSourcesFollowsEvidencePath(t *testing.T) {
	dir := newPinWiki(t)
	decl := writeDecl(t, addDecl(memoBody))
	if out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	// 선언 없이는 거절되고 아무것도 옮기지 않는다.
	if _, err := runEngram(t, "", "promote", "--wiki", dir, "--now", nowFlag, "--to", "sources", "inbox/2026-11-22-메모.md"); err == nil {
		t.Fatal("선언 없이 sources 로 승급됐습니다")
	}
	if _, err := os.Stat(filepath.Join(dir, "inbox", "2026-11-22-메모.md")); err != nil {
		t.Fatal("거절됐는데 원본이 사라졌습니다")
	}
	out, err := runEngram(t, "", "promote", "--wiki", dir, "--now", nowFlag, "--to", "sources", "--pin", "none", "inbox/2026-11-22-메모.md")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "P0001") {
		t.Errorf("근거 경로를 따라간 일이 알려지지 않았습니다:\n%s", out)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "pin", "items", "P0001.json"))
	if strings.Contains(string(data), `"path": "inbox/`) || !strings.Contains(string(data), `"path": "sources/`) {
		t.Errorf("근거 경로가 따라가지 않았습니다:\n%s", data)
	}
}

func TestPromoteToContextRejectsPinFlag(t *testing.T) {
	dir := newPinWiki(t)
	decl := writeDecl(t, addDecl(memoBody))
	if out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	_, err := runEngram(t, "", "promote", "--wiki", dir, "--now", nowFlag, "--pin", "none", "inbox/2026-11-22-메모.md")
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("context 승급이 --pin 을 받았습니다: %v", err)
	}
}

// capture 로 선언과 함께 넣은 문서를 근거로 둔 일이 있는 위키를 만든다.
func pinWikiWithInboxDoc(t *testing.T) string {
	t.Helper()
	dir := newPinWiki(t)
	decl := writeDecl(t, addDecl(memoBody))
	if out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return dir
}

func evidencePath(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "pin", "items", "P0001.json"))
	if err != nil {
		t.Fatal(err)
	}
	var item struct {
		Evidence []struct {
			Path string `json:"path"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(data, &item); err != nil || len(item.Evidence) == 0 {
		t.Fatalf("원장을 읽지 못했습니다: %v\n%s", err, data)
	}
	return item.Evidence[0].Path
}

func TestMvFollowsEvidencePath(t *testing.T) {
	dir := pinWikiWithInboxDoc(t)
	out, err := runEngram(t, "", "mv", "--wiki", dir, "--now", nowFlag, "메모", "개명")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := evidencePath(t, dir); got != "inbox/2026-11-22-개명.md" {
		t.Errorf("mv 가 근거 경로를 따라가지 않았습니다: %s\n%s", got, out)
	}
	if !strings.Contains(out, "P0001") {
		t.Errorf("영향받은 일이 출력되지 않았습니다:\n%s", out)
	}
}

func TestMvOfUnreferencedDocumentLeavesLedgerAlone(t *testing.T) {
	dir := pinWikiWithInboxDoc(t)
	if out, err := runEngram(t, "", "capture", "--wiki", dir, "--now", nowFlag, "--pin", "none", "다른 문서"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "pin", "items", "P0001.json"))
	logsBefore, _ := filepath.Glob(filepath.Join(dir, "pin", "log", "*.jsonl"))
	var sizeBefore int64
	for _, l := range logsBefore {
		st, _ := os.Stat(l)
		sizeBefore += st.Size()
	}
	if out, err := runEngram(t, "", "mv", "--wiki", dir, "--now", nowFlag, "다른-문서", "개명한-문서"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "pin", "items", "P0001.json"))
	if string(before) != string(after) {
		t.Error("근거와 상관없는 문서의 mv 가 일을 바꿨습니다")
	}
	var sizeAfter int64
	for _, l := range logsBefore {
		st, _ := os.Stat(l)
		sizeAfter += st.Size()
	}
	if sizeAfter != sizeBefore {
		t.Error("근거와 상관없는 문서의 mv 가 pin/log 에 줄을 더했습니다")
	}
}

func TestDemoteAndArchiveFollowEvidencePaths(t *testing.T) {
	for _, cmdName := range []string{"demote", "archive"} {
		t.Run(cmdName, func(t *testing.T) {
			dir := pinWikiWithInboxDoc(t)
			// context 로 승급하면 inbox 원본이 사라지고 근거 경로가 따라간다.
			if out, err := runEngram(t, "", "promote", "--wiki", dir, "--now", nowFlag, "inbox/2026-11-22-메모.md"); err != nil {
				t.Fatalf("promote: %v\n%s", err, out)
			}
			ctxPath := evidencePath(t, dir)
			if !strings.HasPrefix(ctxPath, "context/") {
				t.Fatalf("promote 가 근거 경로를 따라가지 않았습니다: %s", ctxPath)
			}
			if out, err := runEngram(t, "", cmdName, "--wiki", dir, "--now", nowFlag, ctxPath); err != nil {
				t.Fatalf("%s: %v\n%s", cmdName, err, out)
			}
			want := "inbox/"
			if cmdName == "archive" {
				want = "archive/"
			}
			if got := evidencePath(t, dir); !strings.HasPrefix(got, want) {
				t.Errorf("%s 가 근거 경로를 따라가지 않았습니다: %s, 기대 접두 %s", cmdName, got, want)
			}
		})
	}
}

func TestPinApplyCommand(t *testing.T) {
	dir := newPinWiki(t)
	decl := writeDecl(t, addDecl(memoBody))
	if out, err := runEngram(t, "# 메모\n\n"+memoBody+"\n", "capture", "--wiki", dir, "--now", nowFlag, "--title", "메모", "--pin", decl); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	req := writeDecl(t, `{"requests":[{"op":"accept","args":{"proposal":"Q0001"}},{"op":"set_status","item":"P0001","args":{"status":"doing"}}]}`)
	out, err := runEngram(t, "", "pin", "apply", "--wiki", dir, "--requests", req, "--now", nowFlag)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "반영 2") {
		t.Errorf("요약이 이상합니다:\n%s", out)
	}
	bad := writeDecl(t, `{"requests":[{"op":"set_status","item":"P7777","args":{"status":"done"}}]}`)
	_, err = runEngram(t, "", "pin", "apply", "--wiki", dir, "--requests", bad)
	if err == nil {
		t.Fatal("거절된 요청이 있는데 종료 코드가 0입니다")
	}
}

func TestPinListJSONShape(t *testing.T) {
	dir := newPinWiki(t)
	out, err := runEngram(t, "", "pin", "list", "--wiki", dir, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Items    []any `json:"items"`
		Works    []any `json:"works"`
		Rejected []any `json:"rejected"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("JSON 이 아닙니다: %v\n%s", err, out)
	}
	if v.Items == nil || v.Works == nil {
		t.Errorf("빈 목록이 null 입니다(에이전트가 읽기 어렵습니다): %s", out)
	}
}
