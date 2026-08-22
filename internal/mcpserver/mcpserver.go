// Package mcpserver는 engram 을 MCP 서버로 노출하는 뼈대를 담는다.
// SDK 어댑터 역할만 한다. 도구 등록과 결과 구조는 커맨드 계층이 맡는다.
// 도구 결과는 CLI 의 --json 출력 구조체를 그대로 쓰는데 그 구조체들이
// cli 패키지 안에 있으므로 이 패키지가 cli 를 알면 순환이 생긴다.
// 그래서 나누었다(ADR 0043).
package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neocode24/engram/internal/i18n"
)

// New는 engram MCP 서버를 만든다. 도구 등록은 호출자가 한다.
// stdout 은 프로토콜 전용이므로 어떤 옵션도 stdout 으로 로그를 보내지
// 않는다. 진단이 필요한 호출자는 stderr 로 낸다.
//
// instructions 는 initialize 응답에 실려 클라이언트가 모델 문맥에
// 넣는다. **스킬 문서를 여기로 보낸다.** 셸을 쥔 에이전트는
// skills install 이 심은 파일로 규약을 읽지만 MCP 만 쓰는
// 클라이언트에는 그 파일이 닿지 않는다. 도구 설명에 규약을 다시
// 적으면 같은 규칙이 두 곳에 살고 한쪽만 고쳐진다(ADR 0090).
func New(name, version, instructions string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:        name,
		Title:       name,
		Description: i18n.T("core.mcpserver.description"),
		Version:     version,
	}, &mcp.ServerOptions{Instructions: instructions})
}

// RunStdio는 stdio 전송으로 서버를 돌린다. HTTP 전송은 두지 않는다.
func RunStdio(ctx context.Context, s *mcp.Server) error {
	return s.Run(ctx, &mcp.StdioTransport{})
}

// Connect는 서버에 붙은 클라이언트 세션을 돌려준다. 테스트가 실제
// 프로토콜 왕복으로 도구를 확인하는 데 쓴다. in-memory 전송을 쓰므로
// 프로세스 밖으로 나가지 않는다. 반환된 정리 함수로 세션과 서버 연결을
// 닫는다.
func Connect(ctx context.Context, s *mcp.Server) (*mcp.ClientSession, func(), error) {
	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, t1, nil)
	if err != nil {
		return nil, nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "engram-client", Version: "dev"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		ss.Close()
		return nil, nil, err
	}
	return cs, func() {
		cs.Close()
		ss.Close()
	}, nil
}
