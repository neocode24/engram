package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neocode24/engram/voice/internal/audio"
	"github.com/neocode24/engram/internal/glossary"
	"github.com/neocode24/engram/voice/internal/model"
	"github.com/neocode24/engram/voice/internal/stt"
)

// maxLine은 합친 줄 하나의 길이 상한이다. 초다. 같은 화자가 계속
// 말해도 이보다 길어지면 줄을 바꾼다. 읽기 편한 문단 길이일 뿐
// 정확성 주장이 아니다.
const maxLine = 30

// transcribeResult는 --json 이 내는 구조다.
type transcribeResult struct {
	Source        string       `json:"source"`
	Model         string       `json:"model"`
	AudioSeconds  float64      `json:"audioSeconds"`
	Speakers      int          `json:"speakers"`
	SpeakersGiven bool         `json:"speakersGiven"`
	Unknown       int          `json:"unknownLines"`
	Corrections   []correction `json:"corrections,omitempty"`
	Lines         []stt.Line   `json:"lines"`
}

// correction은 용어 사전이 바꾼 것 한 건이다. 무엇을 몇 번 바꿨는지
// 산출물에 남긴다. 조용히 바꾸면 검수하는 사람이 도구가 손댄 자리를
// 모른다(ADR 0083).
type correction struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
}

// applyGlossary는 위키의 용어 사전을 전사 줄에 적용한다. 위키를 주지
// 않았거나 사전이 없으면 아무것도 하지 않는다. 사전이 없는 것은
// 오류가 아니다.
func applyGlossary(wikiRoot string, lines []stt.Line) []correction {
	if wikiRoot == "" {
		return nil
	}
	g, err := glossary.Load(wikiRoot)
	if err != nil {
		if errors.Is(err, glossary.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "안내: %s 에 용어 사전이 없어 교정하지 않았습니다\n", wikiRoot)
			return nil
		}
		fmt.Fprintf(os.Stderr, "경고: 용어 사전을 읽지 못해 교정하지 않았습니다: %v\n", err)
		return nil
	}

	total := map[correction]int{}
	for i := range lines {
		fixed, applied := g.Apply(lines[i].Text)
		lines[i].Text = fixed
		for _, a := range applied {
			total[correction{From: a.Rule.Variant, To: a.Rule.Canonical}] += a.Count
		}
	}
	out := make([]correction, 0, len(total))
	for c, n := range total {
		c.Count = n
		out = append(out, c)
	}
	// 많이 바꾼 것을 앞에 둔다. 같으면 사전 순으로 고정한다.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].From < out[j].From
	})

	n := 0
	for _, c := range out {
		n += c.Count
	}
	// 읽은 규칙 수와 실제로 맞은 규칙 수를 함께 낸다. 맞은 수만
	// 내면 0 일 때 사전을 못 읽은 것으로 읽힌다.
	fmt.Fprintf(os.Stderr,
		"용어 사전 %s: 규칙 %d개 읽음, %d개가 맞아 %d건 교정, 검토 대상 %d개는 건드리지 않음\n",
		filepath.Base(g.Path), len(g.Rules), len(out), n, g.Reviewed)
	return out
}

// runTranscribe는 오디오 하나를 전사한다.
func runTranscribe(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("transcribe", flag.ContinueOnError)
	raw := sizeFlag(fs)
	nspk := fs.Int("speakers", 0, "화자 수. 생략하면 추정하며 그 값은 믿을 수 없습니다")
	noDiar := fs.Bool("no-speakers", false, "화자 분할을 건너뜁니다")
	asJSON := fs.Bool("json", false, "결과를 JSON 으로 냅니다")
	keep := fs.String("keep-wav", "", "변환한 wav 를 이 경로에 남깁니다")
	wiki := fs.String("wiki", "", "용어 사전을 읽을 위키 경로. 생략하면 교정하지 않습니다")
	// 표준 flag 는 첫 위치 인자에서 파싱을 멈춘다. 사람은
	// "transcribe 회의.m4a --speakers 3" 이라고 쓰므로 순서를 맞춰 준다.
	if err := fs.Parse(flagsFirst(args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("오디오 파일 하나가 필요합니다")
	}
	src := fs.Arg(0)

	size, dir, files, err := resolve(*raw)
	if err != nil {
		return err
	}
	// 모델이 다 있는지 먼저 본다. 오디오를 변환한 뒤에 모델이 없다고
	// 하면 그 변환 시간이 버려진다.
	if err := requireModel(dir, files, size); err != nil {
		return err
	}

	wav, cleanup, err := prepareWAV(src, *keep)
	if err != nil {
		return err
	}
	defer cleanup()

	samples, rate, err := stt.ReadWAV(wav)
	if err != nil {
		return fmt.Errorf("wav 를 읽을 수 없음: %w", err)
	}
	if rate != audio.SampleRate {
		return fmt.Errorf("%dHz 가 아닙니다: %d", audio.SampleRate, rate)
	}
	seconds := float64(len(samples)) / float64(rate)
	fmt.Fprintf(os.Stderr, "오디오 %s, 모델 %s\n", humanDuration(seconds), size)

	var speakers []stt.Speaker
	if !*noDiar {
		t0 := time.Now()
		speakers, err = stt.Diarize(samples, rate, dir, stt.DiarizeOptions{Speakers: *nspk})
		if err != nil {
			return fmt.Errorf("화자 분할 실패: %w", err)
		}
		n := stt.CountSpeakers(speakers)
		fmt.Fprintf(os.Stderr, "화자 분할 %s, 화자 %d명\n", humanDuration(time.Since(t0).Seconds()), n)
		if *nspk <= 0 {
			// ADR 0082. 추정한 값이라는 사실을 반드시 알린다.
			fmt.Fprintf(os.Stderr,
				"경고: 화자 수를 추정했습니다. 이 값은 믿을 수 없습니다. 아는 값이 있으면 --speakers 로 주세요\n")
		}
	}

	segs, err := stt.Segment(samples, rate, filepath.Join(dir, model.VadName))
	if err != nil {
		return fmt.Errorf("구간 나누기 실패: %w", err)
	}

	tr, err := stt.Open(dir)
	if err != nil {
		return err
	}
	defer tr.Close()

	t1 := time.Now()
	lines := tr.Transcribe(segs, speakers, transcribeProgress(os.Stderr))
	lines = stt.MergeAdjacent(lines, maxLine)
	fmt.Fprintf(os.Stderr, "전사 %s, 줄 %d개\n", humanDuration(time.Since(t1).Seconds()), len(lines))

	corrections := applyGlossary(*wiki, lines)

	res := transcribeResult{
		Source: filepath.Base(src), Model: string(size), AudioSeconds: seconds,
		Speakers: stt.CountSpeakers(speakers), SpeakersGiven: *nspk > 0, Lines: lines,
		Corrections: corrections,
	}
	for _, l := range lines {
		if l.Speaker == stt.Unknown {
			res.Unknown++
		}
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	writeTranscript(out, res, *noDiar)
	return nil
}

// flagsFirst는 플래그를 앞으로, 위치 인자를 뒤로 옮긴다. 값을 받는
// 플래그는 뒤따르는 값도 함께 옮겨야 하므로 어떤 플래그가 값을 받는지
// 알아야 한다. --name=value 형태는 값이 붙어 있으므로 그대로 옮긴다.
//
// "--" 뒤는 손대지 않는다. 파일 이름이 대시로 시작하는 경우가 있고
// 그때 사용자가 쓰는 탈출구다.
func flagsFirst(args []string) []string {
	takesValue := map[string]bool{"--model": true, "-model": true,
		"--speakers": true, "-speakers": true, "--keep-wav": true, "-keep-wav": true,
		"--wiki": true, "-wiki": true}
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			rest = append(rest, a)
			continue
		}
		flags = append(flags, a)
		if takesValue[a] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, rest...)
}

// requireModel은 모델이 다 있는지 본다. 없으면 무엇을 하면 되는지 낸다.
func requireModel(dir string, files []model.ModelFile, size model.Size) error {
	missing := 0
	for _, f := range files {
		if fi, err := os.Stat(filepath.Join(dir, f.Name)); err != nil || fi.Size() != f.Size {
			missing++
		}
	}
	if missing == 0 {
		return nil
	}
	return fmt.Errorf("모델 파일 %d개가 없거나 온전하지 않습니다\nengram-voice model pull --model %s 를 먼저 실행하세요", missing, size)
}

// prepareWAV는 전사에 넣을 wav 를 준비한다. 두 번째 반환값은 임시
// 파일을 지우는 함수이며 남기기로 했으면 아무것도 하지 않는다.
func prepareWAV(src, keep string) (string, func(), error) {
	noop := func() {}
	if _, err := os.Stat(src); err != nil {
		return "", noop, fmt.Errorf("오디오를 열 수 없음: %w", err)
	}
	conv, err := audio.FindConverter()
	if err != nil {
		// wav 를 그대로 받는 길은 남긴다. 변환기가 없어도 이미
		// 16kHz 모노 wav 를 가진 사용자는 쓸 수 있어야 한다.
		if audio.IsWAV(src) {
			fmt.Fprintln(os.Stderr, "안내: 변환기가 없어 준 wav 를 그대로 씁니다")
			return src, noop, nil
		}
		return "", noop, fmt.Errorf("%w\nwav 가 아닌 파일은 변환기가 있어야 합니다", err)
	}

	dst := keep
	if dst == "" {
		f, err := os.CreateTemp("", "engram-voice-*.wav")
		if err != nil {
			return "", noop, err
		}
		dst = f.Name()
		_ = f.Close()
		noop = func() { _ = os.Remove(dst) }
	}
	fmt.Fprintf(os.Stderr, "%s 로 변환 중\n", conv.Name)
	if err := audio.ToWAV(conv, src, dst); err != nil {
		noop()
		return "", func() {}, err
	}
	return dst, noop, nil
}

// transcribeProgress는 전사 진행률을 stderr 에 낸다. 표준 출력은
// 전사 결과만 담아야 파이프로 넘길 수 있다.
func transcribeProgress(w io.Writer) stt.Progress {
	interactive := false
	if fi, err := os.Stdout.Stat(); err == nil {
		interactive = fi.Mode()&os.ModeCharDevice != 0
	}
	var last time.Time
	lastStep := -1
	start := time.Now()
	return func(done, total int) {
		fin := done >= total
		pct := done * 100 / total
		// 남은 시간을 어림한다. 긴 녹음이 몇 분 걸리므로 사용자가
		// 멈춘 것인지 도는 것인지 알아야 한다.
		var eta string
		if done > 0 && !fin {
			per := time.Since(start) / time.Duration(done)
			eta = "  남은 시간 " + humanDuration((per * time.Duration(total-done)).Seconds())
		}
		line := fmt.Sprintf("  전사 %3d%%  %d/%d 구간%s", pct, done, total, eta)
		if interactive {
			if !fin && time.Since(last) < progressInterval {
				return
			}
			last = time.Now()
			fmt.Fprintf(w, "\r%s        ", line)
			if fin {
				fmt.Fprintln(w)
			}
			return
		}
		if step := pct / 10; fin || step > lastStep {
			lastStep = step
			fmt.Fprintln(w, line)
		}
	}
}

// writeTranscript는 사람이 읽을 전사를 낸다. 이 출력이 그대로
// engram capture 의 표준 입력이 된다(ADR 0079).
func writeTranscript(w io.Writer, res transcribeResult, noDiar bool) {
	fmt.Fprintf(w, "# 전사: %s\n\n", res.Source)
	fmt.Fprintf(w, "- 길이: %s\n", humanDuration(res.AudioSeconds))
	fmt.Fprintf(w, "- 모델: whisper %s\n", res.Model)
	switch {
	case noDiar:
		fmt.Fprintln(w, "- 화자: 나누지 않음")
	case res.SpeakersGiven:
		fmt.Fprintf(w, "- 화자: %d명 (사람이 지정)\n", res.Speakers)
	default:
		fmt.Fprintf(w, "- 화자: %d명 (추정. **이 값은 믿을 수 없습니다**)\n", res.Speakers)
	}
	if res.Unknown > 0 {
		fmt.Fprintf(w, "- 화자를 붙이지 못한 줄: %d개\n", res.Unknown)
	}
	fmt.Fprintln(w, "- 이름은 도구가 붙이지 않습니다. 번호를 사람 이름으로 바꾸세요")
	writeCorrections(w, res.Corrections)
	fmt.Fprint(w, "\n## 본문\n\n")
	for _, l := range res.Lines {
		who := ""
		if !noDiar {
			who = speakerLabel(l.Speaker) + ": "
		}
		fmt.Fprintf(w, "[%s -> %s] %s%s\n\n", clock(l.Start), clock(l.End), who, l.Text)
	}
}

// writeCorrections는 용어 사전이 바꾼 것을 산출물에 남긴다.
//
// 조용히 바꾸면 검수하는 사람이 도구가 손댄 자리를 모른다. 사전이
// 틀렸을 때 그것을 발견할 길이 이 목록뿐이다(ADR 0083).
func writeCorrections(w io.Writer, cs []correction) {
	if len(cs) == 0 {
		return
	}
	n := 0
	for _, c := range cs {
		n += c.Count
	}
	fmt.Fprintf(w, "\n### 용어 교정 %d건\n\n", n)
	fmt.Fprint(w, "사전이 바꾼 것입니다. 틀린 것이 있으면 사전을 고치세요.\n\n")
	for _, c := range cs {
		fmt.Fprintf(w, "- `%s` -> `%s` (%d회)\n", c.From, c.To, c.Count)
	}
}

// speakerLabel은 화자 번호를 표시 이름으로 바꾼다. 0부터 매긴 번호를
// 1부터 세어 낸다. 사람은 첫째를 1이라 부른다.
func speakerLabel(id int) string {
	if id == stt.Unknown {
		return "화자 미상"
	}
	return fmt.Sprintf("화자 %d", id+1)
}

// clock은 초를 시:분:초로 만든다.
func clock(sec float64) string {
	s := int(sec)
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

// humanDuration은 초를 사람이 읽는 시간으로 만든다.
func humanDuration(sec float64) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%.1f초", sec)
	case sec < 3600:
		return fmt.Sprintf("%d분 %d초", int(sec)/60, int(sec)%60)
	default:
		return fmt.Sprintf("%d시간 %d분", int(sec)/3600, (int(sec)%3600)/60)
	}
}
