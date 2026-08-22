package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/voice/internal/stt"
	_ "github.com/neocode24/engram/voice/internal/vi18n"
)

func TestFlagsFirstReordersArgs(t *testing.T) {
	// 표준 flag 는 첫 위치 인자에서 멈춘다. 사람은 파일을 먼저 쓴다.
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"파일이 먼저", []string{"a.m4a", "--speakers", "3"},
			[]string{"--speakers", "3", "a.m4a"}},
		{"플래그가 먼저", []string{"--speakers", "3", "a.m4a"},
			[]string{"--speakers", "3", "a.m4a"}},
		{"섞임", []string{"--json", "a.m4a", "--speakers", "2"},
			[]string{"--json", "--speakers", "2", "a.m4a"}},
		{"등호 형태", []string{"a.m4a", "--speakers=3"},
			[]string{"--speakers=3", "a.m4a"}},
		{"불리언 뒤 파일", []string{"--no-speakers", "a.m4a"},
			[]string{"--no-speakers", "a.m4a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := flagsFirst(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("%v 여야 함: %v", c.want, got)
			}
		})
	}
}

func TestFlagsFirstStopsAtDoubleDash(t *testing.T) {
	// 파일 이름이 대시로 시작할 때 쓰는 탈출구다. 그 뒤는 손대지 않는다.
	got := flagsFirst([]string{"--json", "--", "-이상한이름.m4a"})
	want := []string{"--json", "-이상한이름.m4a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%v 여야 함: %v", want, got)
	}
}

func TestWriteTranscriptWarnsOnEstimatedSpeakers(t *testing.T) {
	// ADR 0082. 추정치라는 사실이 산출물에 남아야 한다. 이 전사가
	// 그대로 위키 문서가 되므로 나중에 읽는 사람도 알아야 한다.
	res := transcribeResult{
		Source: "a.m4a", Model: "large-v3", AudioSeconds: 100,
		Speakers: 4, SpeakersGiven: false,
		Lines: []stt.Line{{Start: 0, End: 5, Speaker: 0, Text: "가"}},
	}
	// 문구가 아니라 카탈로그 항목으로 견준다. 리터럴로 견주면 언어를
	// 바꾸거나 문구를 다듬을 때마다 시험이 깨진다.
	estimated := i18n.T("voice.out.speakers_est", 4)
	given := i18n.T("voice.out.speakers", 4)

	var b bytes.Buffer
	writeTranscript(&b, res, false)
	if !strings.Contains(b.String(), estimated) {
		t.Errorf("추정 경고가 있어야 함:\n%s", b.String())
	}

	res.SpeakersGiven = true
	b.Reset()
	writeTranscript(&b, res, false)
	if strings.Contains(b.String(), estimated) {
		t.Errorf("사람이 지정했으면 경고가 없어야 함:\n%s", b.String())
	}
	if !strings.Contains(b.String(), given) {
		t.Errorf("지정했다는 사실을 적어야 함:\n%s", b.String())
	}
}

func TestWriteTranscriptReportsUnknownLines(t *testing.T) {
	res := transcribeResult{
		Source: "a.m4a", Model: "small", AudioSeconds: 10, Speakers: 1,
		SpeakersGiven: true, Unknown: 2,
		Lines: []stt.Line{
			{Start: 0, End: 3, Speaker: stt.Unknown, Text: "가"},
			{Start: 3, End: 6, Speaker: 0, Text: "나"},
		},
	}
	var b bytes.Buffer
	writeTranscript(&b, res, false)
	got := b.String()
	if !strings.Contains(got, "붙이지 못한 줄: 2개") {
		t.Errorf("미상 줄 수를 알려야 함:\n%s", got)
	}
	if !strings.Contains(got, "화자 미상:") {
		t.Errorf("미상 줄에 표시가 있어야 함:\n%s", got)
	}
	// 도구가 이름을 지어내지 않는다는 것을 산출물에 남긴다.
	if !strings.Contains(got, "이름은 도구가 붙이지 않습니다") {
		t.Errorf("이름 규약을 적어야 함:\n%s", got)
	}
}

func TestWriteTranscriptWithoutDiarizationHasNoSpeakers(t *testing.T) {
	res := transcribeResult{
		Source: "a.m4a", Model: "small", AudioSeconds: 10,
		Lines: []stt.Line{{Start: 0, End: 3, Speaker: stt.Unknown, Text: "가"}},
	}
	var b bytes.Buffer
	writeTranscript(&b, res, true)
	got := b.String()
	if strings.Contains(got, "화자 미상") {
		t.Errorf("화자를 안 나눴으면 화자 표시가 없어야 함:\n%s", got)
	}
	if !strings.Contains(got, "나누지 않음") {
		t.Errorf("나누지 않았다고 적어야 함:\n%s", got)
	}
}

func TestSpeakerLabelCountsFromOne(t *testing.T) {
	// 내부 번호는 0부터지만 사람은 첫째를 1이라 부른다.
	if got := speakerLabel(0); got != "화자 1" {
		t.Errorf("화자 1 이어야 함: %s", got)
	}
	if got := speakerLabel(stt.Unknown); got != "화자 미상" {
		t.Errorf("화자 미상 이어야 함: %s", got)
	}
}

func TestClock(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{{0, "00:00:00"}, {59.9, "00:00:59"}, {61, "00:01:01"}, {3661, "01:01:01"}} {
		if got := clock(c.in); got != c.want {
			t.Errorf("%v 는 %s 여야 함: %s", c.in, c.want, got)
		}
	}
}
