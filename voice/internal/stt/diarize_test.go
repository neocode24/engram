package stt

import "testing"

// 아래 시험은 모델을 열지 않는다. 군집 결과를 어떻게 다듬는지가
// 이 파일의 대상이고, 그 판단은 모델과 무관한 순수 계산이다.

func TestDropFragmentsTurnsSmallClustersUnknown(t *testing.T) {
	// 실측에서 본 모양이다. 한 군집이 대부분을 말하고 나머지는
	// 0.3초에서 몇 초짜리 파편이다(ADR 0082).
	segs := []Speaker{
		{Start: 0, End: 100, ID: 7},
		{Start: 100, End: 101, ID: 3},
		{Start: 101, End: 130, ID: 7},
		{Start: 130, End: 130.4, ID: 9},
	}
	got := dropFragments(segs, 0.02)
	want := []int{7, Unknown, 7, Unknown}
	for i := range got {
		if got[i].ID != want[i] {
			t.Errorf("구간 %d 의 화자가 %d 여야 함: %d", i, want[i], got[i].ID)
		}
	}
	// 구간 자체는 버리지 않는다. 시간대는 남고 화자만 모른다.
	if len(got) != len(segs) {
		t.Errorf("구간 수가 %d 여야 함: %d", len(segs), len(got))
	}
}

func TestDropFragmentsKeepsBalancedSpeakers(t *testing.T) {
	// 셋이 고르게 말하면 아무도 걸러지면 안 된다. 필터가 정상적인
	// 대화를 망가뜨리는지가 이 시험의 요점이다.
	segs := []Speaker{
		{Start: 0, End: 30, ID: 0},
		{Start: 30, End: 60, ID: 1},
		{Start: 60, End: 90, ID: 2},
		{Start: 90, End: 120, ID: 0},
	}
	got := dropFragments(segs, 0.02)
	for i, s := range got {
		if s.ID == Unknown {
			t.Errorf("구간 %d 가 걸러지면 안 됨", i)
		}
	}
	if n := CountSpeakers(got); n != 3 {
		t.Errorf("화자가 셋이어야 함: %d", n)
	}
}

func TestRenumberClosesGaps(t *testing.T) {
	// 군집 번호는 내부 값이라 걸러내면 구멍이 생긴다. 처음 말한
	// 순서로 0부터 다시 매긴다.
	segs := []Speaker{
		{Start: 0, End: 1, ID: 12},
		{Start: 1, End: 2, ID: Unknown},
		{Start: 2, End: 3, ID: 47},
		{Start: 3, End: 4, ID: 12},
	}
	got := renumber(segs)
	want := []int{0, Unknown, 1, 0}
	for i := range got {
		if got[i].ID != want[i] {
			t.Errorf("구간 %d 의 번호가 %d 여야 함: %d", i, want[i], got[i].ID)
		}
	}
}

func TestAssignSpeakerPicksLongestOverlap(t *testing.T) {
	speakers := []Speaker{
		{Start: 0, End: 10, ID: 0},
		{Start: 10, End: 20, ID: 1},
	}
	// 구간 8-18 은 0 과 2초, 1 과 8초 겹친다. 더 많이 말한 쪽이다.
	if got := AssignSpeaker(8, 18, speakers); got != 1 {
		t.Errorf("화자 1 이어야 함: %d", got)
	}
	// 겹치는 것이 없으면 모른다고 한다. 가까운 쪽으로 밀지 않는다.
	if got := AssignSpeaker(30, 40, speakers); got != Unknown {
		t.Errorf("겹침이 없으면 Unknown 이어야 함: %d", got)
	}
	// 경계가 맞닿기만 하면 겹친 것이 아니다.
	if got := AssignSpeaker(20, 30, speakers); got != Unknown {
		t.Errorf("맞닿기만 하면 Unknown 이어야 함: %d", got)
	}
}

func TestCountsIgnoreUnknown(t *testing.T) {
	segs := []Speaker{{ID: 0}, {ID: Unknown}, {ID: 1}, {ID: Unknown}}
	if n := CountSpeakers(segs); n != 2 {
		t.Errorf("화자 수가 2 여야 함: %d", n)
	}
	if n := CountUnknown(segs); n != 2 {
		t.Errorf("미상 수가 2 여야 함: %d", n)
	}
}
