package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRoundTrip는 뼈대가 실제 프로토콜 왕복을 하는지 본다. SDK 를 믿고
// 넘어가지 않기 위한 최소 확인이다. 실제 engram 도구의 왕복은 cli 계층
// 테스트가 담당한다.
func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := New("engram-test", "dev", "시험용 지시")
	s.AddTool(&mcp.Tool{
		Name:        "echo",
		Description: "입력을 그대로 돌려주는 시험 도구",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var in struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &in); err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: in.Text}},
		}, nil
	})

	session, done, err := Connect(ctx, s)
	if err != nil {
		t.Fatalf("서버 접속 실패: %v", err)
	}
	defer done()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list 실패: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "echo" {
		t.Fatalf("도구 목록이 틀림: %+v", tools.Tools)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"text": "안녕"},
	})
	if err != nil {
		t.Fatalf("tools/call 실패: %v", err)
	}
	if res.IsError {
		t.Fatalf("도구 호출이 에러로 돌아옴: %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatal("응답에 내용이 없음")
	}
	if text, ok := res.Content[0].(*mcp.TextContent); !ok || text.Text != "안녕" {
		t.Errorf("왕복 결과가 틀림: %+v", res.Content[0])
	}
}

// TestInstructionsReachClient는 initialize 응답에 지시가 실리는지 본다.
// MCP 만 쓰는 클라이언트에는 이것이 규약을 받는 유일한 경로다(ADR 0090).
func TestInstructionsReachClient(t *testing.T) {
	const want = "이 서버를 이렇게 쓴다"
	ctx := context.Background()
	s := New("engram-test", "dev", want)
	cs, done, err := Connect(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	if got := cs.InitializeResult().Instructions; got != want {
		t.Errorf("지시가 %q 여야 하는데 %q", want, got)
	}
}
