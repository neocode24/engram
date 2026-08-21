package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/lint"
	"github.com/spf13/cobra"
)

// runRules는 rules 커맨드를 루트 등록 없이 시험한다.
// 커맨드 등록은 coordinator 소관이므로 테스트 안에서 부모 커맨드를
// 조립한다. 전역 플래그는 실제 루트와 같은 PersistentPreRunE 로 흉내낸다.
func runRules(t *testing.T, args ...string) (string, error) {
	t.Helper()
	parent := &cobra.Command{Use: "engram", SilenceUsage: true}
	parent.PersistentFlags().Bool(flagJSON, false, "결과를 JSON으로 출력합니다")
	parent.PersistentFlags().String(flagNow, "", "기준 시각(RFC3339)")
	parent.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		raw, err := cmd.Flags().GetString(flagNow)
		if err != nil {
			return err
		}
		parsed, err := parseNow(raw)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), nowKey{}, parsed))
		return nil
	}
	parent.AddCommand(newRulesCmd())
	var out bytes.Buffer
	parent.SetOut(&out)
	parent.SetErr(&out)
	parent.SetArgs(args)
	err := parent.Execute()
	return out.String(), err
}

// makeRulesWiki는 지정한 프리셋과 설정 내용으로 위키를 만든다.
func makeRulesWiki(t *testing.T, yaml string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "context"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "engram.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRulesShowCmd(t *testing.T) {
	t.Run("프리셋 셋마다 켜진 축 수가 다릅니다", func(t *testing.T) {
		counts := map[string]int{}
		for _, preset := range []string{"minimal", "personal", "team"} {
			wiki := makeRulesWiki(t, "preset: "+preset+"\n")
			out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
			if err != nil {
				t.Fatalf("preset %s 실행 실패: %v\n%s", preset, err, out)
			}
			var res rulesReport
			jsonPart := out[strings.Index(out, "{"):]
			if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
				t.Fatalf("preset %s JSON 파싱 실패: %v\n%s", preset, err, out)
			}
			for _, on := range res.Axes {
				if on {
					counts[preset]++
				}
			}
		}
		if counts["minimal"] != 8 || counts["personal"] != 10 || counts["team"] != 14 {
			t.Errorf("속성 수 = %v, want minimal 8, personal 10, team 14", counts)
		}
	})

	t.Run("사용자가 바꾼 임계값이 제품 기본값 대신 나옵니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: personal\nmin_wikilinks: 5\nstale_days: 60\n")
		out, err := runRules(t, "rules", "show", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		// 정렬 공백에 영향받지 않도록 임계값 행만 뽑아 비교한다.
		lines := []string{}
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "min_wikilinks") || strings.Contains(line, "stale_days") {
				fields := strings.Fields(line)
				lines = append(lines, strings.Join(fields, " "))
			}
		}
		joined := strings.Join(lines, "\n")
		for _, want := range []string{"min_wikilinks 5", "stale_days 60"} {
			if !strings.Contains(joined, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
		if strings.Contains(joined, "min_wikilinks 2") || strings.Contains(joined, "stale_days 30") {
			t.Errorf("제품 기본값이 나옴:\n%s", out)
		}
	})

	t.Run("꺼진 축이 꺼졌다고 나옵니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: personal\n")
		out, err := runRules(t, "rules", "show", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "꺼짐") {
			t.Fatalf("꺼진 축 안내가 없음:\n%s", out)
		}
		for _, ax := range []string{"scope", "sensitivity", "trigger_mode", "workflow"} {
			if !strings.Contains(out, ax) {
				t.Errorf("꺼진 축 %s가 출력에 없음:\n%s", ax, out)
			}
		}
	})

	t.Run("두 번 돌리면 바이트까지 같습니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: team\n")
		first, err1 := runRules(t, "rules", "show", "--wiki", wiki)
		second, err2 := runRules(t, "rules", "show", "--wiki", wiki)
		if err1 != nil || err2 != nil {
			t.Fatalf("실행 에러: %v %v", err1, err2)
		}
		if first != second {
			t.Fatalf("두 실행의 출력이 다릅니다:\n%s\n===\n%s", first, second)
		}
	})

	t.Run("파일을 쓰지 않습니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: personal\n")
		before := snapshotFiles(t, wiki)
		out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		after := snapshotFiles(t, wiki)
		if before != after {
			t.Fatalf("조회 커맨드가 파일을 만들거나 지웠습니다:\n%s\n===\n%s", before, after)
		}
	})

	t.Run("--json 이 유효한 JSON 입니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: team\n")
		out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		var res rulesReport
		jsonPart := out[strings.Index(out, "{"):]
		if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
		}
		if res.Preset != "team" || len(res.Rules) == 0 || res.Gate.OnlyRejectRule != "gate.min-wikilinks" {
			t.Errorf("JSON 내용이 잘못됨: %+v", res)
		}
	})

	t.Run("위키가 아닌 디렉토리에서는 init 을 안내합니다", func(t *testing.T) {
		out, err := runRules(t, "rules", "show", "--wiki", t.TempDir())
		if err == nil || !strings.Contains(out, "engram init") {
			t.Fatalf("거절되어야 함: %v\n%s", err, out)
		}
	})

	t.Run("rules 만 치면 사용법을 냅니다", func(t *testing.T) {
		out, err := runRules(t, "rules")
		// 하위 커맨드 없이 치면 cobra 가 사용법을 인쇄한다.
		if !strings.Contains(out, "show") {
			t.Errorf("사용법에 show 가 없음:\n%s", out)
		}
		if err != nil {
			t.Errorf("사용법 인쇄는 에러가 아니어야 함: %v", err)
		}
	})
}

// snapshotFiles는 디렉토리 아래 파일 목록을 상대 경로로 모아 반환한다.
func snapshotFiles(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return strings.Join(paths, "\n")
}

// TestRulesListMatchesLint은 rules show 의 규칙 목록이 internal/lint 가
// 내보내는 목록과 하나인지 확인한다. 규칙 메타데이터의 진실원은
// lint.Rules 하나이므로 이 비교는 조립이 빠뜨리는 것을 잡는다.
func TestRulesListMatchesLint(t *testing.T) {
	wiki := makeRulesWiki(t, "preset: personal\n")
	out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
	if err != nil {
		t.Fatalf("실행 실패: %v\n%s", err, out)
	}
	// lint.Rule 은 등급과 설명을 메시지 ID 로 담고 JSON 에는 푼 문자열을
	// 낸다(ADR 0049). 그래서 lint.Rule 로 되받지 않고 낸 형태 그대로 읽는다.
	var res struct {
		Rules []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
			Desc     string `json:"desc"`
		} `json:"rules"`
	}
	jsonPart := out[strings.Index(out, "{"):]
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
		t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
	}
	want := lint.Rules()
	if len(res.Rules) != len(want) {
		t.Fatalf("규칙 수가 다름: 출력 %d, lint.Rules %d", len(res.Rules), len(want))
	}
	// Rule 은 등급과 설명을 메시지 ID 로 담고 JSON 에는 푼 문자열을 낸다.
	// 그래서 값 비교가 아니라 푼 결과끼리 대조한다. ADR 0049.
	for i, r := range want {
		got := res.Rules[i]
		if got.ID != r.ID || got.Severity != r.Severity() || got.Desc != r.Desc() {
			t.Errorf("규칙 %d번이 다름: 출력 %+v, want {ID:%s Severity:%s Desc:%s}",
				i, got, r.ID, r.Severity(), r.Desc())
		}
	}
}

// TestDisplayWidth는 표시 폭 계산이 동아시아 넓은 문자를 두 칸으로
// 세는지 검증한다.
func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"error", 5},
		{"0123", 4},
		{"또는", 4},
		{"error 또는 warn", 15},
		{"설명이 여기 있습니다", 20},
		{"가나다123", 9},
		{"漢字", 4},   // 한중일 통합 한자
		{"ひらがな", 8}, // 히라가나
		{"カタカナ", 8}, // 가타카나
		{"！", 2},    // 전각 형식
		{"　", 2},    // 전각 공백
	}
	for _, c := range cases {
		if got := displayWidth(c.in); got != c.want {
			t.Errorf("displayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestPadRight는 한글이 섞인 문자열을 표시 폭 기준으로 채우는지 검증한다.
func TestPadRight(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"또는", 6, "또는  "},
		{"error", 5, "error"},
		{"error", 7, "error  "},
		{"가", 6, "가    "},
		{"넓은 값", 4, "넓은 값"},
	}
	for _, c := range cases {
		if got := padRight(c.in, c.width); got != c.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

// displayCol은 문자열의 처음부터 byteIdx 까지의 표시 폭을 센다.
// 테스트가 열 시작 위치를 표시 폭으로 잰다.
func displayCol(s string, byteIdx int) int {
	n := 0
	for i, r := range s {
		if i >= byteIdx {
			break
		}
		if isWide(r) {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// colAfterToken은 줄에서 tok 토큰 뒤의 공백을 건너뛴 위치의 표시 폭을 반환한다.
func colAfterToken(line, tok string) (int, bool) {
	i := strings.Index(line, tok)
	if i < 0 {
		return 0, false
	}
	i += len(tok)
	for i < len(line) && line[i] == ' ' {
		i++
	}
	return displayCol(line, i), true
}

// TestRulesShowTableColumns은 rules show 의 네 표에서 마지막 열의 시작
// 위치가 모든 행에서 같은지 검증한다. fmt 의 룬 수 폭으로는 한글이 섞인
// 열이 화면에서 어긋난다.
func TestRulesShowTableColumns(t *testing.T) {
	wiki := makeRulesWiki(t, "preset: personal\n")
	out, err := runRules(t, "rules", "show", "--wiki", wiki)
	if err != nil {
		t.Fatalf("실행 실패: %v\n%s", err, out)
	}

	var ruleCols, threshCols, stageCols, valueCols, openValueCols []int
	section := ""
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "":
			continue
		case !strings.HasPrefix(line, " "):
			section = line
			continue
		}
		switch {
		case strings.HasPrefix(line, "  ["):
			// 규칙 표. [등급] ID 설명 순서고 ID 에는 공백이 없다.
			bracket := strings.Index(line, "] ")
			if bracket < 0 {
				t.Fatalf("규칙 줄에서 등급 닫기를 못 찾음: %q", line)
			}
			id := strings.Fields(line[bracket+2:])
			if len(id) == 0 {
				t.Fatalf("규칙 줄에서 ID 를 못 찾음: %q", line)
			}
			if col, ok := colAfterToken(line, id[0]); ok {
				ruleCols = append(ruleCols, col)
			}
		case strings.HasPrefix(section, "임계값"):
			f := strings.Fields(line)
			if len(f) < 3 {
				continue
			}
			if col, ok := colAfterToken(line, f[1]); ok {
				threshCols = append(threshCols, col)
			}
		case strings.HasPrefix(section, "단계별"):
			f := strings.Fields(line)
			if len(f) < 2 {
				continue
			}
			if col, ok := colAfterToken(line, f[0]); ok {
				stageCols = append(stageCols, col)
			}
		case strings.HasPrefix(section, "허용값. 폐쇄"):
			f := strings.Fields(line)
			if len(f) < 2 {
				continue
			}
			if col, ok := colAfterToken(line, f[0]); ok {
				valueCols = append(valueCols, col)
			}
		case strings.HasPrefix(section, "허용값. 개방"):
			f := strings.Fields(line)
			if len(f) < 2 {
				continue
			}
			if col, ok := colAfterToken(line, f[0]); ok {
				openValueCols = append(openValueCols, col)
			}
		}
	}
	for name, cols := range map[string][]int{
		"규칙 표":  ruleCols,
		"임계값 표": threshCols,
		"단계별 표": stageCols,
		"허용값 표": valueCols,
	} {
		if len(cols) == 0 {
			t.Errorf("%s의 행을 못 찾았습니다", name)
			continue
		}
		for _, c := range cols[1:] {
			if c != cols[0] {
				t.Errorf("%s의 마지막 열 시작 위치가 행마다 다릅니다: %v", name, cols)
				break
			}
		}
	}
}

// TestRulesShowGlossary는 사전이 있을 때와 없을 때를 본다. 없는 것은
// 오류가 아니라 아직 안 채운 상태이며 출력이 그렇게 말해야 한다(ADR 0083).
func TestRulesShowGlossary(t *testing.T) {
	const table = `| 정규형 | 변형 | 자동 교정 | 설명 |
|---|---|---|---|
| 임베딩 | 임배딩, 임베딩모델 | yes | |
| 승급 | 승격 | review | 문맥에 따라 다름 |
`
	t.Run("사전이 없으면 없다고 밝힙니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: personal\n")
		out, err := runRules(t, "rules", "show", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		if !strings.Contains(out, "용어 사전") || !strings.Contains(out, "없습니다") {
			t.Errorf("사전이 없다는 안내가 없습니다:\n%s", out)
		}
	})

	t.Run("사전이 있으면 경로와 수를 냅니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: personal\n")
		if err := os.MkdirAll(filepath.Join(wiki, "meta"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(wiki, "meta", "terminology.md"), []byte(table), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
		if err != nil {
			t.Fatalf("실행 실패: %v\n%s", err, out)
		}
		var res rulesReport
		if err := json.Unmarshal([]byte(strings.TrimSpace(out[strings.Index(out, "{"):])), &res); err != nil {
			t.Fatalf("JSON 파싱 실패: %v", err)
		}
		// 변형 둘에 검토 항목 하나다.
		if res.Glossary.Rules != 2 || res.Glossary.Reviewed != 1 {
			t.Errorf("규칙 2개 검토 1개를 기대했으나 %+v", res.Glossary)
		}
		// 절대 경로가 새면 출력에 사용자 홈이 섞인다.
		if res.Glossary.Path != filepath.Join("meta", "terminology.md") {
			t.Errorf("위키 루트 기준 상대 경로여야 하는데 %q", res.Glossary.Path)
		}
	})
}
