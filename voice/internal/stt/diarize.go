package stt

import (
	"errors"
	"path/filepath"
	"sort"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/neocode24/engram/voice/internal/model"
)

// Speaker는 화자가 말한 구간 하나다. Start 와 End 는 초이고 ID 는
// 0부터 매긴 화자 번호다. 이름이 아니라 번호인 이유는 도구가 누구인지
// 알 수 없기 때문이다. 이름을 붙이는 것은 사람의 일이다.
type Speaker struct {
	Start float64
	End   float64
	ID    int
}

// 화자 분할 상수.
const (
	// diarThreads는 분할과 임베딩에 쓰는 스레드 수다.
	diarThreads = 4
	// clusterThreshold는 화자를 가르는 코사인 거리 하한이다. 낮을수록
	// 화자를 잘게 쪼갠다. sherpa-onnx 예제의 기본값이다.
	clusterThreshold = 0.5
	// minDurationOn은 이보다 짧은 발화를 버린다. 초다.
	minDurationOn = 0.3
	// minDurationOff는 이보다 짧은 침묵은 같은 발화로 잇는다. 초다.
	minDurationOff = 0.5
	// defaultMinSpeechRatio는 자동 추정에서 군집을 살려 둘 최소
	// 발화 비율이다. 전체 발화 시간에 대한 비율이며 이보다 적게
	// 말한 군집은 화자 미상이 된다. 근거는 dropFragments 에 있다.
	defaultMinSpeechRatio = 0.02
)

// Unknown은 화자를 알 수 없다는 표시다.
const Unknown = -1

// DiarizeOptions는 화자 분할 선택지다.
type DiarizeOptions struct {
	// Speakers는 화자 수다. 0 이하면 자동으로 추정한다.
	//
	// 아는 값이 있으면 주는 편이 낫다. 자동 추정은 임계값 하나에
	// 기대므로 목소리가 비슷한 둘을 하나로 합치거나 한 사람의 톤
	// 변화를 둘로 가르는 일이 있다.
	Speakers int
	// Threshold는 자동 추정에서 화자를 가르는 코사인 거리 하한이다.
	// 0 이면 clusterThreshold 를 쓴다. Speakers 를 주면 무시된다.
	Threshold float32
	// MinSpeechRatio는 자동 추정에서 군집을 살려 둘 최소 발화 비율이다.
	// 0 이면 defaultMinSpeechRatio 를 쓴다. Speakers 를 주면 무시된다.
	MinSpeechRatio float64
}

// Diarize는 표본에서 화자별 구간을 낸다. dir 은 model.Dir 이 낸
// 모델 디렉토리다.
//
// 표본 전체를 한 번에 넘긴다. 전사와 달리 30초 창 제약이 없고,
// 오히려 나눠 넘기면 조각마다 화자 번호를 새로 매겨 이어 붙일 수
// 없게 된다.
func Diarize(samples []float32, rate int, dir string, opts DiarizeOptions) ([]Speaker, error) {
	if rate != 16000 {
		return nil, errors.New("화자 분할은 16000Hz 만 받습니다")
	}
	if len(samples) == 0 {
		return nil, errors.New("표본이 비었습니다")
	}

	cfg := sherpa.OfflineSpeakerDiarizationConfig{}
	cfg.Segmentation.Pyannote.Model = filepath.Join(dir, model.SegmentName)
	cfg.Segmentation.NumThreads = diarThreads
	cfg.Embedding.Model = filepath.Join(dir, model.SpeakerName)
	cfg.Embedding.NumThreads = diarThreads
	// 화자 수를 모르면 NumClusters 를 음수로 두고 임계값으로 가른다.
	// 0 을 주면 군집이 하나도 안 생긴다.
	if opts.Speakers > 0 {
		cfg.Clustering.NumClusters = opts.Speakers
	} else {
		cfg.Clustering.NumClusters = -1
		cfg.Clustering.Threshold = clusterThreshold
		if opts.Threshold > 0 {
			cfg.Clustering.Threshold = opts.Threshold
		}
	}
	cfg.MinDurationOn = minDurationOn
	cfg.MinDurationOff = minDurationOff

	sd := sherpa.NewOfflineSpeakerDiarization(&cfg)
	if sd == nil {
		return nil, errors.New("화자 분할기를 만들 수 없습니다. 모델 경로를 확인하세요")
	}
	defer sherpa.DeleteOfflineSpeakerDiarization(sd)

	segs := sd.Process(samples)
	out := make([]Speaker, 0, len(segs))
	for _, s := range segs {
		out = append(out, Speaker{Start: float64(s.Start), End: float64(s.End), ID: s.Speaker})
	}
	// 시각 순으로 고정한다. 뒤에서 전사 구간과 겹치는 정도로 화자를
	// 붙일 때 순서가 정해져 있어야 한다.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start != out[j].Start {
			return out[i].Start < out[j].Start
		}
		return out[i].ID < out[j].ID
	})
	if opts.Speakers <= 0 {
		out = dropFragments(out, opts.MinSpeechRatio)
	}
	return renumber(out), nil
}

// dropFragments는 발화 시간이 너무 적은 군집을 화자 미상으로 돌린다.
//
// 자동 추정이 긴 녹음에서 무너지기 때문에 있다. 실측으로 6.7분 녹음이
// 화자 열일곱을 냈는데 그중 하나가 163.6초를 말하고 나머지 열여섯은
// 0.3초에서 18초짜리 파편이었다. 파편의 시간 분포폭 중앙값이 전체의
// 0.7%였다. 대화 내내 등장하는 사람이 아니라 한 지점에서 떨어져 나온
// 조각이라는 뜻이다.
//
// 원인은 같은 사람의 목소리가 시간에 따라 변하는데 코사인 임계값
// 하나로 자르면 시간이 갈수록 새 군집이 계속 생기는 것이다. upstream
// 실측에서도 화자 수가 녹음 길이에 비례했다. 120분 녹음이 132명이다.
//
// **잘못된 화자를 붙이느니 모른다고 하는 편이 낫다.** 걸러진 구간은
// Unknown 이 되고 전사에는 화자 없이 붙는다.
func dropFragments(segs []Speaker, ratio float64) []Speaker {
	if ratio <= 0 {
		ratio = defaultMinSpeechRatio
	}
	talk := map[int]float64{}
	var total float64
	for _, s := range segs {
		d := s.End - s.Start
		talk[s.ID] += d
		total += d
	}
	floor := total * ratio
	out := make([]Speaker, 0, len(segs))
	for _, s := range segs {
		if talk[s.ID] < floor {
			s.ID = Unknown
		}
		out = append(out, s)
	}
	return out
}

// renumber는 화자 번호를 처음 말한 순서로 0부터 다시 매긴다.
//
// 군집 번호는 내부 값이라 걸러내고 나면 구멍이 생긴다. 전사에
// "화자 12" 와 "화자 47" 이 나오면 읽는 사람은 그 사이 서른넷이
// 어디 있는지 묻는다. Unknown 은 그대로 둔다.
func renumber(segs []Speaker) []Speaker {
	next := 0
	seen := map[int]int{}
	out := make([]Speaker, 0, len(segs))
	for _, s := range segs {
		if s.ID == Unknown {
			out = append(out, s)
			continue
		}
		id, ok := seen[s.ID]
		if !ok {
			id = next
			seen[s.ID] = id
			next++
		}
		s.ID = id
		out = append(out, s)
	}
	return out
}

// AssignSpeaker는 전사 구간 하나에 화자를 붙인다. 겹치는 시간이 가장
// 긴 화자를 고르고 겹치는 것이 없으면 -1 을 낸다.
//
// 전사 구간과 화자 구간은 서로 다른 모델이 다른 규칙으로 나눈 것이라
// 경계가 맞지 않는다. 겹침으로 붙이는 것이 그 어긋남을 다루는 가장
// 단순한 방법이고, 한 구간에 화자가 둘 섞여 있으면 더 많이 말한
// 쪽으로 간다. 그것이 틀렸다고 보이면 사람이 고친다.
func AssignSpeaker(start, end float64, speakers []Speaker) int {
	best, bestID := 0.0, -1
	for _, s := range speakers {
		o := overlap(start, end, s.Start, s.End)
		if o > best {
			best, bestID = o, s.ID
		}
	}
	return bestID
}

// overlap은 두 구간이 겹치는 시간을 낸다. 겹치지 않으면 0 이다.
func overlap(aStart, aEnd, bStart, bEnd float64) float64 {
	lo := aStart
	if bStart > lo {
		lo = bStart
	}
	hi := aEnd
	if bEnd < hi {
		hi = bEnd
	}
	if hi <= lo {
		return 0
	}
	return hi - lo
}

// CountSpeakers는 구간 목록에 나온 화자 수를 센다. 화자 미상은
// 세지 않는다. 모르는 것은 사람이 아니다.
func CountSpeakers(speakers []Speaker) int {
	seen := map[int]bool{}
	for _, s := range speakers {
		if s.ID != Unknown {
			seen[s.ID] = true
		}
	}
	return len(seen)
}

// CountUnknown은 화자를 붙이지 못한 구간 수를 센다. 호출자가 그
// 사실을 사용자에게 알려야 하므로 따로 낸다.
func CountUnknown(speakers []Speaker) int {
	n := 0
	for _, s := range speakers {
		if s.ID == Unknown {
			n++
		}
	}
	return n
}
