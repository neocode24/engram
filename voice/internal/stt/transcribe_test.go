package stt

import "testing"

func TestMergeAdjacentBoundsLineLength(t *testing.T) {
	// 구간 사이 간격이 0 이다. Segment 가 오디오 전체를 덮기 때문이며
	// 그래서 길이로 끊는다. 간격으로 끊으면 전부 한 줄이 된다.
	lines := []Line{
		{Start: 0, End: 20, Speaker: 0, Text: "가"},
		{Start: 20, End: 25, Speaker: 0, Text: "나"},
		{Start: 25, End: 45, Speaker: 0, Text: "다"},
		{Start: 45, End: 50, Speaker: 1, Text: "라"},
	}
	got := MergeAdjacent(lines, 30)
	if len(got) != 3 {
		t.Fatalf("줄이 셋이어야 함: %d", len(got))
	}
	if got[0].Text != "가 나" || got[0].End != 25 {
		t.Errorf("첫 줄이 상한 안에서 합쳐져야 함: %+v", got[0])
	}
	if got[1].Text != "다" {
		t.Errorf("상한을 넘으면 새 줄이어야 함: %+v", got[1])
	}
	if got[2].Speaker != 1 {
		t.Errorf("화자가 바뀌면 새 줄이어야 함: %+v", got[2])
	}
}

func TestMergeAdjacentKeepsSpeakerBoundary(t *testing.T) {
	lines := []Line{
		{Start: 0, End: 2, Speaker: 0, Text: "가"},
		{Start: 2, End: 4, Speaker: 1, Text: "나"},
		{Start: 4, End: 6, Speaker: 0, Text: "다"},
	}
	got := MergeAdjacent(lines, 30)
	if len(got) != 3 {
		t.Errorf("화자가 번갈면 합치면 안 됨: %d줄", len(got))
	}
}

func TestMergeAdjacentEmpty(t *testing.T) {
	if got := MergeAdjacent(nil, 30); len(got) != 0 {
		t.Errorf("빈 입력은 빈 출력이어야 함: %d", len(got))
	}
}
