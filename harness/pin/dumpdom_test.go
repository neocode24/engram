package pin_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 크롬 하나만으로 하는 화면 기동 시험이다. node 와 playwright-core 가 필요 없다.
// 크롬은 ENGRAM_CHROME 에서 찾고, 없으면 PATH 와 macOS 기본 위치에서 찾는다. 없으면 건너뛴다.

var tagRe = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9-]*)((?:"[^"]*"|'[^']*'|[^'">])*)>`)
var hiddenAttrRe = regexp.MustCompile(`(?:^|\s)hidden(?:\s|=|/|$)`)

var voidTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
	"input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true,
}

// visibleText는 DOM 직렬화에서 숨긴 요소(hidden 속성)의 글자를 빼고 읽는다.
// script 와 style 의 내용도 글자로 세지 않는다. 크롬 --dump-dom 은 hidden 요소의 글자도 그대로 내므로
// 이렇게 빼지 않으면 "로그인이 만료됐습니다" 같은 숨은 안내가 있는 것으로 읽힌다(설계 2.3절).
func visibleText(dom string) string {
	var out strings.Builder
	hiddenDepth := 0 // 숨긴 요소 안에 있을 때 그 요소의 깊이. 0 이면 숨기지 않음
	depth := 0
	skipUntil := "" // script, style 안
	pos := 0
	for _, m := range tagRe.FindAllStringSubmatchIndex(dom, -1) {
		text := dom[pos:m[0]]
		pos = m[1]
		if skipUntil == "" && hiddenDepth == 0 {
			out.WriteString(text)
		}
		closing := dom[m[2]:m[3]] == "/"
		name := strings.ToLower(dom[m[4]:m[5]])
		attrs := dom[m[6]:m[7]]
		if skipUntil != "" {
			if closing && name == skipUntil {
				skipUntil = ""
			}
			continue
		}
		if closing {
			if hiddenDepth > 0 && depth == hiddenDepth {
				hiddenDepth = 0
			}
			depth--
			continue
		}
		if voidTags[name] || strings.HasSuffix(strings.TrimSpace(attrs), "/") {
			if hiddenDepth == 0 {
				out.WriteString(" ")
			}
			continue
		}
		depth++
		if hiddenDepth == 0 && hiddenAttrRe.MatchString(attrs) {
			hiddenDepth = depth
		}
		if name == "script" || name == "style" {
			skipUntil = name
		}
		out.WriteString(" ")
	}
	if skipUntil == "" && hiddenDepth == 0 {
		out.WriteString(dom[pos:])
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func TestVisibleTextSkipsHidden(t *testing.T) {
	dom := `<html><body><div id="authbar" role="alert" hidden=""><div><b>로그인이 만료됐습니다.</b><br>다시</div></div>` +
		`<div id="view"><p>보이는 글</p><input value="x"><div hidden>숨김 둘<span>안쪽</span></div><p>뒤의 글</p></div>` +
		`<script>var 스크립트="글자 아님"</script><style>.a{content:"스타일"}</style><span title="a > b">끝</span></body></html>`
	got := visibleText(dom)
	for _, want := range []string{"보이는 글", "뒤의 글", "끝"} {
		if !strings.Contains(got, want) {
			t.Errorf("보여야 할 글 %q 가 없다: %q", want, got)
		}
	}
	for _, bad := range []string{"로그인이 만료됐습니다", "다시", "숨김 둘", "안쪽", "스크립트", "스타일"} {
		if strings.Contains(got, bad) {
			t.Errorf("숨긴 글 %q 가 읽혔다: %q", bad, got)
		}
	}
}

func findChrome() string {
	if p := os.Getenv("ENGRAM_CHROME"); p != "" {
		return p
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if runtime.GOOS == "darwin" {
		p := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// TestPinPageDumpDOM은 engram serve 가 낸 /pin 을 크롬 --headless --dump-dom 으로 열어
// 화면이 떠서 상태를 그렸는지 본다. 숨긴 요소의 글자는 빼고 읽는다.
func TestPinPageDumpDOM(t *testing.T) {
	chrome := findChrome()
	if chrome == "" {
		t.Skip("크롬이 없어 건너뜁니다(ENGRAM_CHROME 으로 경로를 줄 수 있습니다)")
	}
	addr, _ := startPinServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
		"--user-data-dir="+filepath.Join(t.TempDir(), "profile"), "--virtual-time-budget=8000", "--dump-dom", "http://"+addr+"/pin")
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("크롬이 화면을 열지 못했습니다: %v", err)
	}
	dom := string(raw)
	if !strings.Contains(dom, `id="authbar"`) {
		t.Fatalf("화면이 뜨지 않았습니다. DOM 앞부분: %.300s", dom)
	}
	text := visibleText(dom)
	for _, want := range []string{"견적서 보내기", "설치 초안 만들기", "오늘", "프로젝트별 현황", "연관 지도", "할 일"} {
		if !strings.Contains(text, want) {
			t.Errorf("보이는 글에 %q 가 없습니다", want)
		}
	}
	for _, bad := range []string{"상태를 읽지 못했습니다", "미리알림에서 들어온 일이 없습니다", "로그인이 만료됐습니다"} {
		if strings.Contains(text, bad) {
			t.Errorf("보이면 안 되는 글 %q 가 있습니다", bad)
		}
	}
	// 숨은 로그인 안내가 DOM 에는 남아 있고 보이는 글에서만 빠져야 시험이 숨긴 요소를 실제로 거른 것이다.
	inDOM := strings.Contains(dom, "로그인이 만료됐습니다")
	t.Logf("DOM %d바이트, 숨은 로그인 안내가 DOM 에 있음=%v, 보이는 글 %d자", len(dom), inDOM, len([]rune(text)))
	if !inDOM {
		t.Error("숨은 로그인 안내가 DOM 에 없습니다. 숨긴 요소를 거르는 시험이 성립하지 않으므로 화면이 바뀐 것인지 확인하십시오")
	}
}
