package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/neocode24/engram/internal/i18n"
	"github.com/neocode24/engram/internal/mcpserver"
)

// runMCP는 전사기를 MCP 서버로 노출한다.
//
// **CLI 만 두면 아무도 안 쓴다.** 사람이 회의 녹음을 손으로 전사하고
// 손으로 위키에 넣는 일은 실제로 일어나지 않는다. 에이전트가 도구로
// 부를 수 있어야 여정 2가 성립한다.
//
// 전송은 stdio 하나다. stdout 은 프로토콜 전용이므로 진행률과 안내를
// 전부 stderr 로 보낸다. 전사가 몇 분씩 걸리므로 진행률이 특히 중요한데
// 그것이 stdout 으로 새면 JSON-RPC 가 깨져 서버가 조용히 죽는다.
//
// **위키에 쓰지 않는다는 경계는 그대로다**(ADR 0079). 여기 있는 도구는
// 전사 결과를 돌려줄 뿐이고 위키에 넣는 것은 engram 의 capture 도구다.
// 에이전트가 둘을 이어 붙인다.
func runMCP(ctx context.Context, args []string) error {
	// instructions 는 initialize 응답에 실려 클라이언트가 모델 문맥에
	// 넣는다. engram 쪽 서버가 스킬 문서 전체를 보내므로 여기서는
	// 겹치지 않게 짧게 낸다. 규약의 진실원은 그 문서 하나다(ADR 0090).
	s := mcp.NewServer(&mcp.Implementation{
		Name:        "engram-voice",
		Title:       "engram-voice",
		Description: i18n.T("voice.mcp.desc"),
		Version:     version,
	}, &mcp.ServerOptions{Instructions: i18n.T("voice.mcp.instructions")})
	registerVoiceTools(s)
	fmt.Fprintln(os.Stderr, i18n.T("voice.mcp.starting"))
	return mcpserver.RunStdio(ctx, s)
}

// mcpTranscribeArgs는 transcribe 도구의 입력이다.
type mcpTranscribeArgs struct {
	Audio string `json:"audio" jsonschema:"오디오 파일 경로. m4a, mp3, wav 등"`
	// 화자 수는 아는 값을 받는다. 추정은 긴 녹음에서 무너진다(ADR 0082).
	Speakers int `json:"speakers,omitempty" jsonschema:"화자 수. 아는 값이 있으면 주세요. 생략하면 추정하며 그 값은 믿을 수 없습니다"`
	// 위키를 주면 용어 사전으로 교정한다. 안 주면 교정하지 않는다.
	Wiki string `json:"wiki,omitempty" jsonschema:"용어 사전을 읽을 위키 경로. 생략하면 교정하지 않습니다"`
	// 화자를 안 나누면 분할 시간을 아낀다. 혼자 말한 녹음에 쓴다.
	NoSpeakers bool   `json:"noSpeakers,omitempty" jsonschema:"화자 분할을 건너뜁니다. 혼자 말한 녹음에 씁니다"`
	Model      string `json:"model,omitempty" jsonschema:"모델 크기. large-v3(기본), medium, small"`
}

// mcpModelArgs는 model_status 도구의 입력이다.
type mcpModelArgs struct {
	Model string `json:"model,omitempty" jsonschema:"모델 크기. large-v3(기본), medium, small"`
}

// modelStatusResult는 모델이 준비되었는지 알린다. 전사를 부르기 전에
// 이것을 먼저 보게 하려고 둔다. 1.7GB 를 받는 동안 에이전트가 전사를
// 부르면 오래 기다리다 실패한다.
type modelStatusResult struct {
	Model   string   `json:"model"`
	Dir     string   `json:"dir"`
	Ready   bool     `json:"ready"`
	Missing []string `json:"missing,omitempty"`
	Hint    string   `json:"hint,omitempty"`
}

// registerVoiceTools는 도구를 등록한다.
//
// 도구가 둘뿐인 이유는 나머지가 에이전트의 일이 아니기 때문이다.
// 모델 내려받기는 1.7GB 를 받는 일이라 사람이 보고 있어야 하고,
// 위키에 넣는 것은 engram 쪽 도구다.
func registerVoiceTools(s *mcp.Server) {
	mcp.AddTool[mcpModelArgs, any](s, &mcp.Tool{
		Name:        "model_status",
		Description: i18n.T("voice.mcp.tool_status"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpModelArgs) (*mcp.CallToolResult, any, error) {
		size, dir, files, err := resolve(in.Model)
		if err != nil {
			return nil, nil, err
		}
		res := modelStatusResult{Model: string(size), Dir: dir, Ready: true}
		for _, f := range files {
			if fi, err := os.Stat(filepath.Join(dir, f.Name)); err != nil || fi.Size() != f.Size {
				res.Ready = false
				res.Missing = append(res.Missing, f.Name)
			}
		}
		if !res.Ready {
			var total int64
			for _, f := range files {
				total += f.Size
			}
			res.Hint = i18n.T("voice.mcp.pull_hint", size, humanBytes(total))
		}
		return nil, res, nil
	})

	mcp.AddTool[mcpTranscribeArgs, any](s, &mcp.Tool{
		Name:        "transcribe",
		Description: i18n.T("voice.mcp.tool_transcribe"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpTranscribeArgs) (*mcp.CallToolResult, any, error) {
		res, err := transcribeAudio(transcribeInput{
			Source:     in.Audio,
			Speakers:   in.Speakers,
			NoSpeakers: in.NoSpeakers,
			Wiki:       in.Wiki,
			RawModel:   in.Model,
		})
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	})
}
