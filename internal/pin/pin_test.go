package pin

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 22, 10, 0, 0, 0, time.FixedZone("KST", 9*3600))

func newTestWiki(t *testing.T) string {
	t.Helper()
	wiki := t.TempDir()
	if _, err := Init(wiki); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(wiki, "inbox", "2026-11-22-메모.md")
	if err := os.MkdirAll(filepath.Dir(doc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doc, []byte("# 메모\n\n김대리가 금요일까지 견적서를 보낸다.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return wiki
}

func decl(t *testing.T, text string) *Declaration {
	t.Helper()
	d, err := ParseDeclaration(text)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

const declAdd = `{"requests":[{"op":"add","args":{"title":"견적서 보내기","evidence":[{"section":"","quote":"김대리가 금요일까지 견적서를 보낸다."}]}}]}`

func memoDoc() Doc {
	return Doc{Path: "inbox/2026-11-22-메모.md", Stage: "inbox", Date: "2026-11-22",
		Text: "# 메모\n\n김대리가 금요일까지 견적서를 보낸다.\n"}
}

func TestPresentFollowsConfigFile(t *testing.T) {
	wiki := t.TempDir()
	if Present(wiki) {
		t.Fatal("pin/config.json 이 없는데 있는 것으로 판정했습니다")
	}
	if _, err := Init(wiki); err != nil {
		t.Fatal(err)
	}
	if !Present(wiki) {
		t.Fatal("Init 뒤에도 없는 것으로 판정했습니다")
	}
}

func TestInitIsIdempotentAndKeepsExistingFiles(t *testing.T) {
	wiki := t.TempDir()
	first, err := Init(wiki)
	if err != nil || len(first) == 0 {
		t.Fatalf("첫 init: %v %v", first, err)
	}
	cfg := filepath.Join(wiki, "pin", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"works":[{"id":"a","name":"에이","words":[]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Init(wiki)
	if err != nil || len(second) != 0 {
		t.Fatalf("둘째 init 이 무언가 만들었습니다: %v %v", second, err)
	}
	data, _ := os.ReadFile(cfg)
	if !strings.Contains(string(data), "에이") {
		t.Fatal("기존 config.json 을 덮어썼습니다")
	}
}

func TestDeclarationCommitsDocAndLedgerTogether(t *testing.T) {
	wiki := newTestWiki(t)
	var wrote bool
	rep, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(),
		Commit: func() error { wrote = true; return nil }})
	if err != nil {
		t.Fatalf("선언이 거절됐습니다: %v", err)
	}
	if !wrote {
		t.Fatal("게이트를 통과했는데 문서 쓰기를 부르지 않았습니다")
	}
	if len(rep.Items) != 1 || rep.Items[0] != "P0001" || len(rep.Proposals) != 1 {
		t.Fatalf("새 일과 제안이 보고되지 않았습니다: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(wiki, "pin", "items", "P0001.json")); err != nil {
		t.Fatal("원장 파일이 없습니다")
	}
}

func TestRejectedDeclarationWritesNothing(t *testing.T) {
	wiki := newTestWiki(t)
	before := snapshotTree(t, filepath.Join(wiki, "pin"))
	var wrote bool
	// 인용문이 문서에 없다. 요청 둘 중 하나만 틀려도 문서도 원장도 쓰지 않는다.
	bad := `{"requests":[` +
		`{"op":"add","args":{"title":"맞는 일","evidence":[{"section":"","quote":"김대리가 금요일까지 견적서를 보낸다."}]}},` +
		`{"op":"add","args":{"title":"지어낸 일","evidence":[{"section":"","quote":"문서에 없는 문장입니다"}]}}]}`
	rep, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, bad), Doc: memoDoc(),
		Commit: func() error { wrote = true; return nil }})
	var rej *RejectedError
	if !errors.As(err, &rej) {
		t.Fatalf("거절 에러가 아닙니다: %v", err)
	}
	if rej.Rejected[0].Code != "quote_missing" {
		t.Errorf("거절 코드: %s", rej.Rejected[0].Code)
	}
	if wrote {
		t.Fatal("거절됐는데 문서를 썼습니다")
	}
	if after := snapshotTree(t, filepath.Join(wiki, "pin")); after != before {
		t.Fatalf("거절됐는데 pin/ 이 바뀌었습니다:\n%s", after)
	}
	if rep == nil || len(rep.Results) != 2 {
		t.Fatalf("거절 사유가 보고되지 않았습니다: %+v", rep)
	}
}

func TestDeclarationCannotCiteAnotherDocument(t *testing.T) {
	for _, field := range []string{"path", "stage", "date"} {
		text := `{"requests":[{"op":"add","args":{"title":"x","evidence":[{"` + field + `":"inbox/다른.md","quote":"q"}]}}]}`
		d := decl(t, text)
		if _, err := fillDeclared(d, memoDoc(), t0); err == nil {
			t.Errorf("근거에 %s 를 적은 선언을 받았습니다", field)
		}
	}
}

func TestDeclarationOpsAreLimited(t *testing.T) {
	for _, text := range []string{
		`{"requests":[{"op":"move_path","args":{"from":"a","to":"b"}}]}`,
		`{"requests":[{"op":"set_status","item":"P0001","args":{"status":"done"}}]}`,
		`{"requests":[{"op":"delete","item":"P0001"}]}`,
		`{"requests":[{"op":"accept","args":{"proposal":"Q0001"}}]}`,
		`{"requests":[{"op":"add","via":"web","args":{"title":"x"}}]}`,
		`{"requests":[{"op":"add","id":"x","args":{"title":"x"}}]}`,
		`{"requests":[]}`,
		`[]`,
	} {
		if _, err := ParseDeclaration(text); err == nil {
			t.Errorf("선언이 받으면 안 되는 모양을 받았습니다: %s", text)
		}
	}
	if d, err := ParseDeclaration("none"); err != nil || !d.None {
		t.Errorf("none 을 받지 못했습니다: %v", err)
	}
}

func TestNewItemFromDeclarationIsAProposal(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	l, err := List(wiki)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 1 || !l.Items[0].Proposed {
		t.Fatalf("선언으로 들어온 새 일에 제안 표시가 없습니다: %+v", l.Items)
	}
}

func TestSameDeclarationTwiceDoesNotDuplicate(t *testing.T) {
	wiki := newTestWiki(t)
	run := func() *Report {
		rep, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }})
		if err != nil {
			var rej *RejectedError
			if errors.As(err, &rej) {
				return &Report{Results: rej.Rejected}
			}
			t.Fatal(err)
		}
		return rep
	}
	run()
	second := run()
	l, _ := List(wiki)
	if len(l.Items) != 1 {
		t.Fatalf("같은 선언을 다시 보냈더니 항목이 %d개가 됐습니다", len(l.Items))
	}
	if len(second.Results) != 1 || second.Results[0].Result == "applied" && !second.Results[0].Noop {
		t.Errorf("둘째 선언이 새로 적용됐습니다: %+v", second.Results)
	}
}

func TestNoneIsLoggedAndWritesNoItem(t *testing.T) {
	wiki := newTestWiki(t)
	var wrote bool
	rep, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, "none"), Doc: memoDoc(), Commit: func() error { wrote = true; return nil }})
	if err != nil || !rep.None {
		t.Fatalf("none: %v %+v", err, rep)
	}
	if !wrote {
		t.Fatal("none 도 문서는 써야 합니다")
	}
	logs, _ := filepath.Glob(filepath.Join(wiki, "pin", "log", "*.jsonl"))
	if len(logs) != 1 {
		t.Fatalf("none 의 흔적이 pin/log 에 없습니다")
	}
	data, _ := os.ReadFile(logs[0])
	if !strings.Contains(string(data), "inbox/2026-11-22-메모.md") || !strings.Contains(string(data), "declared_none") {
		t.Errorf("로그에 문서 경로와 none 이 없습니다: %s", data)
	}
	if l, _ := List(wiki); len(l.Items) != 0 {
		t.Error("none 이 항목을 만들었습니다")
	}
}

func TestLedgerFailureRollsBackDocument(t *testing.T) {
	wiki := newTestWiki(t)
	var rolled bool
	// pin/items 를 파일로 막아 저장이 실패하게 한다.
	if err := os.RemoveAll(filepath.Join(wiki, "pin", "items")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "pin", "items"), []byte("막음"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(),
		Commit: func() error { return nil }, Rollback: func() { rolled = true }})
	if err == nil {
		t.Fatal("저장이 실패했는데 에러가 없습니다")
	}
	if !rolled {
		t.Fatal("원장 저장이 실패했는데 방금 쓴 문서를 되돌리지 않았습니다")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	wiki := newTestWiki(t)
	before := snapshotTree(t, filepath.Join(wiki, "pin"))
	var wrote bool
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), DryRun: true,
		Commit: func() error { wrote = true; return nil }}); err != nil {
		t.Fatal(err)
	}
	if wrote || snapshotTree(t, filepath.Join(wiki, "pin")) != before {
		t.Fatal("--dry-run 이 썼습니다")
	}
}

func TestFollowRewritesEvidencePathWithoutRevBump(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	// 문서를 sources 로 옮긴 것으로 친다. 옮긴 문서는 디스크에 있어야 한다.
	to := filepath.Join(wiki, "sources", "2026-11-22-메모.md")
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("# 메모\n\n본문이 바뀌었습니다. 승급 준비 뒤의 문서입니다.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	affected, err := Follow(wiki, []Move{{From: "inbox/2026-11-22-메모.md", To: "sources/2026-11-22-메모.md"}}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(affected) != 1 || affected[0].ID != "P0001" {
		t.Fatalf("근거로 둔 열린 일을 알리지 않았습니다: %+v", affected)
	}
	data, _ := os.ReadFile(filepath.Join(wiki, "pin", "items", "P0001.json"))
	text := string(data)
	if !strings.Contains(text, `"path": "sources/2026-11-22-메모.md"`) || !strings.Contains(text, `"stage": "sources"`) {
		t.Errorf("근거 경로와 단계가 따라가지 않았습니다:\n%s", text)
	}
	if !strings.Contains(text, `"rev": 1`) {
		t.Errorf("경로만 따라가는 변경이 rev 를 올렸습니다:\n%s", text)
	}
	// 인용문은 옮긴 문서에 더 없지만 거절하지 않는다(승급 준비로 본문이 바뀌는 것이 정상 절차다).
	if !strings.Contains(text, "김대리가 금요일까지 견적서를 보낸다.") {
		t.Error("인용문이 사라졌습니다")
	}
}

func TestFollowIgnoresUnreferencedMoves(t *testing.T) {
	wiki := newTestWiki(t)
	before := snapshotTree(t, filepath.Join(wiki, "pin"))
	affected, err := Follow(wiki, []Move{{From: "inbox/없는-문서.md", To: "sources/2026-11-22-없는-문서.md"}}, t0)
	if err != nil || len(affected) != 0 {
		t.Fatalf("%v %v", affected, err)
	}
	if snapshotTree(t, filepath.Join(wiki, "pin")) != before {
		t.Fatal("근거로 쓰이지 않는 문서의 이동이 원장을 바꿨습니다(로그에 변화 없음 줄이 쌓입니다)")
	}
}

func TestApplyWebRejectsUnsafeBodies(t *testing.T) {
	wiki := newTestWiki(t)
	for name, body := range map[string]string{
		"json 아님":     `not json`,
		"객체 아님":       `[]`,
		"id 실음":       `{"id":"x","op":"touch"}`,
		"via 실음":      `{"via":"command","op":"touch"}`,
		"화면이 못 내는 연산": `{"op":"move_path","args":{"from":"a","to":"b"}}`,
		"모르는 연산":      `{"op":"nope"}`,
	} {
		_, _, err := ApplyWeb(wiki, []byte(body), t0)
		var bad *BadRequest
		if !errors.As(err, &bad) {
			t.Errorf("%s: 400 으로 답할 에러가 아닙니다: %v", name, err)
		}
	}
	// web 은 extract 전용 연산을 못 낸다(상태를 바꾸거나 지울 수는 있지만 문서를 근거로 새 일을 만들 수는 없다는 뜻이 아니라 표의 차이다)
	if opsByVia["web"]["update_ai"] || opsByVia["web"]["move_path"] || opsByVia["web"]["propose"] {
		t.Error("web 이 extract 전용 연산을 낼 수 있습니다")
	}
}

func TestApplyWebWritesAndAnswersImmediately(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	id, at, err := ApplyWeb(wiki, []byte(`{"op":"accept","args":{"proposal":"Q0001"}}`), t0.Add(time.Hour))
	if err != nil || id == "" || !strings.HasSuffix(at, "+09:00") {
		t.Fatalf("%q %q %v", id, at, err)
	}
	l, _ := List(wiki)
	if l.Items[0].Proposed {
		t.Error("맞음이 받는 즉시 반영되지 않았습니다")
	}
	state, err := State(wiki, t0.Add(time.Hour))
	if err != nil || !strings.Contains(string(state), `"proposed": false`) {
		t.Errorf("상태가 원장을 따라가지 않았습니다: %v", err)
	}
}

func TestWebRevConflictIsRecordedNotWritten(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ApplyWeb(wiki, []byte(`{"op":"set_status","item":"P0001","base_rev":7,"args":{"status":"done"}}`), t0); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(wiki, "pin", "items", "P0001.json"))
	if strings.Contains(string(data), `"status": "done"`) {
		t.Fatal("rev 가 맞지 않는 요청이 반영됐습니다")
	}
	logs, _ := filepath.Glob(filepath.Join(wiki, "pin", "log", "*.jsonl"))
	text, _ := os.ReadFile(logs[0])
	if !strings.Contains(string(text), "rev_conflict") {
		t.Error("거절이 로그에 남지 않았습니다. 화면이 상태의 recent_changes 에서 읽습니다")
	}
}

func TestConcurrentWritersDoNotLoseUpdates(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := `{"op":"add","args":{"title":"동시에 넣은 일 ` + string(rune('A'+i)) + `"}}`
			if _, _, err := ApplyWeb(wiki, []byte(body), t0.Add(time.Duration(i)*time.Second)); err != nil {
				t.Errorf("요청 %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	l, err := List(wiki)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 13 {
		t.Fatalf("동시에 쓴 12건 가운데 잃은 것이 있습니다: 항목 %d개(기대 13)", len(l.Items))
	}
	ids := map[string]bool{}
	for _, it := range l.Items {
		if ids[it.ID] {
			t.Fatalf("항목 번호가 겹쳤습니다: %s", it.ID)
		}
		ids[it.ID] = true
	}
}

func TestStaleLockIsReclaimed(t *testing.T) {
	wiki := newTestWiki(t)
	lock := filepath.Join(wiki, "pin", lockName)
	if err := os.WriteFile(lock, []byte("죽은 프로세스"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := List(wiki); err != nil {
		t.Fatalf("오래된 잠금을 치우지 못했습니다: %v", err)
	}
}

func TestStateIsBuiltFromLedgerEveryTime(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	a, err := State(wiki, t0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(wiki, []any{map[string]any{"op": "touch", "item": "P0001", "args": map[string]any{"note": "확인함"}}}, t0); err != nil {
		t.Fatal(err)
	}
	b, _ := State(wiki, t0)
	if string(a) == string(b) {
		t.Fatal("CLI 가 쓴 것이 다음 상태에 보이지 않습니다")
	}
	if _, err := os.Stat(filepath.Join(wiki, "pin", "state.json")); err == nil {
		t.Fatal("상태 파일을 두지 않기로 했습니다")
	}
}

func TestEvidenceLinksOnlyForContextDocuments(t *testing.T) {
	cases := map[string]bool{
		"context/a.md": true, "inbox/a.md": false, "sources/a.md": false, "archive/a.md": false,
		"context/a.txt": false, "context/../inbox/a.md": false, "/context/a.md": false,
	}
	for path, want := range cases {
		if got := docURL(path) != nil; got != want {
			t.Errorf("%s 링크 여부 %v, 기대 %v", path, got, want)
		}
	}
}

func TestPageCSPHashesEveryInlineBlock(t *testing.T) {
	csp := PageCSP()
	if !strings.Contains(csp, "script-src 'sha256-") || !strings.Contains(csp, "style-src 'sha256-") {
		t.Fatalf("인라인 블록을 해시로 허용하지 않습니다: %s", csp)
	}
	for _, want := range []string{"default-src 'none'", "connect-src 'self'", "img-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("정책에 %q 가 없습니다", want)
		}
	}
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
		t.Error("unsafe 를 허용했습니다")
	}
	html := string(Page())
	if strings.Contains(html, "neocode24") || strings.Contains(html, "pin:site") {
		t.Error("사이트 구역이 지워지지 않았습니다")
	}
	if !strings.Contains(html, "var API='/pin/api/', SOURCES=['wiki'], AUTH='none';") {
		t.Error("설정 구역 값이 engram 값이 아닙니다")
	}
}

func TestOpsTablesMatchUpstreamShape(t *testing.T) {
	// 표를 줄이면 하네스가 견줄 때마다 예외를 적어야 한다. 줄지 않았는지 개수로 막는다(내용은 harness/pin 이 upstream 과 견준다).
	if len(opsByVia["web"]) != 18 || len(opsByVia["command"]) != 19 || len(opsByVia["extract"]) != 8 {
		t.Fatalf("허용 연산 표의 크기가 바뀌었습니다: web %d, command %d, extract %d",
			len(opsByVia["web"]), len(opsByVia["command"]), len(opsByVia["extract"]))
	}
}

func TestZeroProjectsIsAccepted(t *testing.T) {
	wiki := newTestWiki(t) // Init 의 설정은 프로젝트가 0개다
	if _, err := State(wiki, t0); err != nil {
		t.Fatalf("프로젝트 0개 원장이 거절됐습니다: %v", err)
	}
}

func TestForbiddenWordsAreNotChecked(t *testing.T) {
	wiki := newTestWiki(t)
	text := `{"requests":[{"op":"add","args":{"title":"스팸 차단 목록 정리","evidence":[{"section":"","quote":"김대리가 금요일까지 견적서를 보낸다."}]}}]}`
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, text), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatalf("정상 제목이 화면 문체 규칙으로 거절됐습니다: %v", err)
	}
}

func TestAddedAtRoundTripKeepsOffset(t *testing.T) {
	wiki := newTestWiki(t)
	if _, err := Run(Txn{Wiki: wiki, Now: t0, Decl: decl(t, declAdd), Doc: memoDoc(), Commit: func() error { return nil }}); err != nil {
		t.Fatal(err)
	}
	logs, _ := filepath.Glob(filepath.Join(wiki, "pin", "log", "*.jsonl"))
	data, _ := os.ReadFile(logs[0])
	if strings.Contains(string(data), "Z\"") {
		t.Errorf("시각이 Z 표기로 적혔습니다. 시스템 파이썬 3.9 가 읽지 못합니다:\n%s", data)
	}
	if !strings.Contains(string(data), "+09:00") {
		t.Errorf("시각이 오프셋 표기가 아닙니다:\n%s", data)
	}
}

// snapshotTree는 디렉터리의 모든 파일 이름과 내용을 한 글로 만든다. 잠금은 뺀다.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == lockName {
			return nil
		}
		data, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(root, p)
		b.WriteString(rel + "\n" + string(data) + "\n--\n")
		return nil
	})
	return b.String()
}
