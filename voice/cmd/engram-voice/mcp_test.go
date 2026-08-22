package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neocode24/engram/internal/mcpserver"
)

// TestVoiceMCPTools는 도구가 실제 프로토콜 왕복으로 보이는지 본다.
// 모델을 열지 않는 model_status 만 부른다.
func TestVoiceMCPTools(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "engram-voice", Version: "test"}, nil)
	registerVoiceTools(s)
	ctx := context.Background()
	cs, done, err := mcpserver.Connect(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	lt, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, tool := range lt.Tools {
		got[tool.Name] = true
	}
	for _, want := range []string{"transcribe", "model_status"} {
		if !got[want] {
			t.Errorf("도구 %q 가 없습니다: %v", want, got)
		}
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "model_status"})
	if err != nil {
		t.Fatalf("model_status 실패: %v", err)
	}
	if res.IsError {
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				t.Errorf("model_status 가 오류를 냈습니다: %s", tc.Text)
			}
		}
		t.FailNow()
	}
}
