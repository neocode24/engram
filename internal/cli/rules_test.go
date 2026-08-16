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
		for _, preset := range []string{"personal", "education", "team"} {
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
		if counts["personal"] != 8 || counts["education"] != 10 || counts["team"] != 14 {
			t.Errorf("축 수 = %v, want personal 8, education 10, team 14", counts)
		}
	})

	t.Run("사용자가 바꾼 임계값이 제품 기본값 대신 나옵니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: education\nmin_wikilinks: 5\nstale_days: 30\n")
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
		for _, want := range []string{"min_wikilinks 5", "stale_days 30"} {
			if !strings.Contains(joined, want) {
				t.Errorf("출력에 %q 없음:\n%s", want, out)
			}
		}
		if strings.Contains(joined, "min_wikilinks 2") || strings.Contains(joined, "stale_days 90") {
			t.Errorf("제품 기본값이 나옴:\n%s", out)
		}
	})

	t.Run("꺼진 축이 꺼졌다고 나옵니다", func(t *testing.T) {
		wiki := makeRulesWiki(t, "preset: education\n")
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
		wiki := makeRulesWiki(t, "preset: education\n")
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
	wiki := makeRulesWiki(t, "preset: education\n")
	out, err := runRules(t, "rules", "show", "--json", "--wiki", wiki)
	if err != nil {
		t.Fatalf("실행 실패: %v\n%s", err, out)
	}
	var res rulesReport
	jsonPart := out[strings.Index(out, "{"):]
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonPart)), &res); err != nil {
		t.Fatalf("JSON 파싱 실패: %v\n출력: %s", err, out)
	}
	want := lint.Rules()
	if len(res.Rules) != len(want) {
		t.Fatalf("규칙 수가 다름: 출력 %d, lint.Rules %d", len(res.Rules), len(want))
	}
	for i, r := range want {
		if res.Rules[i] != r {
			t.Errorf("규칙 %d번이 다름: 출력 %+v, want %+v", i, res.Rules[i], r)
		}
	}
}
