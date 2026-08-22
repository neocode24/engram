---
number: 0090
title: 스킬 문서가 진실원이고 MCP는 그것을 instructions로 보낸다
date: 2026-08-22
status: accepted
---

# 스킬 문서가 진실원이고 MCP는 그것을 instructions로 보낸다

## 배경

[0041](0041-skills-install-embeds-one-static-skill.md)이 스킬을 정적인 한 벌로 정하고 [0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md)이 MCP 도구 열을 정했다. **둘의 관계는 정하지 않았다.**

그 사이에 [0089](0089-voice-is-driven-by-an-agent-and-its-pipeline-is-documented.md)가 `engram-voice mcp`를 더하면서 같은 규칙을 두 곳에 적었다.

## 측정

### 같은 규칙이 두 곳에 있었다

화자 이름을 지어내지 말라는 규칙이 이렇게 갈려 있었다.

| 자리 | 문구 |
|---|---|
| `SKILL.md` | 약한 근거로 화자 이름을 지어내지 않는다 |
| MCP 도구 설명 | 결과의 화자는 번호이고 이름이 아닙니다. 약한 근거로 이름을 지어내지 마세요 |

화자 수를 사람에게 묻는 규칙, 위키에 넣는 것은 `capture`가 한다는 규칙도 같았다. **한쪽만 고치면 다른 쪽이 조용히 남는다.**

### MCP만 쓰는 클라이언트에는 스킬 문서가 안 닿았다

`skills install`은 `~/.claude/skills/` 같은 **파일 경로**에 심는다. 셸을 쥔 에이전트는 그 파일을 읽지만 MCP 프로토콜로만 붙은 클라이언트는 그 디렉토리를 보지 않는다.

그래서 도구 설명에 규약을 적을 수밖에 없었고, 그것이 위의 중복을 낳았다.

### 프로토콜에 그 자리가 있었다

MCP `initialize` 응답에 `instructions` 필드가 있다. 명세가 "서버와 그 기능을 어떻게 쓰는지 설명하는 지시" 라고 적어 두었고 클라이언트가 그것을 모델 문맥에 넣는다. Go SDK 는 `ServerOptions.Instructions` 로 받는다.

**쓰고 있지 않았다.** 확인해 보니 서버 능력이 `logging` 과 `tools` 뿐이고 `instructions` 는 비어 있었다.

    capabilities: {"logging": {}, "tools": {"listChanged": true}}
    prompts/list -> []
    resources/list -> []

### 크기를 쟀다

`skills.Body()` 가 15,252바이트, 7,682글자다. 대략 5천에서 6천 토큰이며 **셸 에이전트가 스킬 파일로 받는 것과 같은 양이다.** 전달 경로만 다르고 내용이 같으므로 새로 드는 비용이 아니다.

## 판단 근거

### 문서가 진실원이고 프로토콜은 배달 경로다

규약은 한 곳에 있어야 한다. 어느 쪽이 진실원인지 정하면 나머지는 배달이다.

`SKILL.md` 를 고른다. 이미 [0041](0041-skills-install-embeds-one-static-skill.md)이 한 벌로 정했고 [0077](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md)이 `eject` 로 위키에도 내보내게 했다. 세 번째 소비자가 MCP 다.

| 소비자 | 받는 법 |
|---|---|
| 셸을 쥔 에이전트 | `skills install` 이 심은 파일 |
| MCP 클라이언트 | `initialize` 의 `instructions` |
| 위키를 복제한 사람 | `eject` 가 낸 `meta/agent-contract.md` |

셋이 같은 `skills.Body()` 를 본다.

### 도구 설명은 도구만 설명한다

절차 규칙을 도구 설명에서 걷어낸다. 도구 설명에는 **그 도구가 무엇을 하고 무엇을 돌려주는지**만 남긴다.

절차와 경계는 `instructions` 가 나른다. 도구 설명은 도구를 고를 때 읽는 것이고 절차는 일을 시작하기 전에 읽는 것이라 자리가 다르다.

### voice 는 짧게 낸다

두 서버를 다 등록하면 같은 15KB 를 두 번 받는다. `engram` 서버가 스킬 문서 전체를 내고 `engram-voice` 서버는 **자기 경계만** 짧게 낸다. 462글자다.

거기 적는 것은 넷이다. 모델 확인, 화자 수를 사람에게 묻기, 이름을 지어내지 않기, 위키에 넣는 것은 `capture` 라는 것이다. 회의록 구조 같은 나머지는 engram 쪽 문서를 가리킨다.

**voice 서버만 등록한 클라이언트도 경계는 안다.** 그것이 이 넷을 중복이 아니라 최소 계약으로 두는 이유다.

### 스킬과 MCP 를 둘 다 남긴다

하나로 합칠 수 없다. 대상이 다르다.

| | 스킬 파일 | MCP |
|---|---|---|
| 필요한 것 | 셸 | 프로토콜 연결 |
| 도구 호출 | 에이전트가 커맨드를 친다 | 클라이언트가 함수로 부른다 |
| 없으면 | MCP 클라이언트가 규약을 모른다 | 셸 없는 클라이언트가 아무것도 못 한다 |

**중복은 규약이 두 벌인 것이었지 경로가 둘인 것이 아니다.** 경로는 둘이어야 한다.

## 결정

| 항목 | 값 |
|---|---|
| 진실원 | **`internal/skills/SKILL.md` 한 벌** |
| 배달 | 파일(`skills install`), MCP(`instructions`), 위키(`eject`) 셋 |
| `engram mcp` | `instructions` 에 `skills.Body()` 전체 |
| `engram-voice mcp` | **경계 넷만** 짧게. 나머지는 engram 문서를 가리킨다 |
| 도구 설명 | **그 도구가 하는 일만.** 절차 규칙을 적지 않는다 |
| 스킬과 MCP | **둘 다 남긴다.** 대상이 다르다 |

## 결과

- MCP 만 쓰는 클라이언트가 규약을 받는다. 그 전에는 도구 이름과 설명만 봤다.
- 같은 규칙이 한 곳에만 있다. 스킬 문서를 고치면 세 경로에 함께 반영된다.
- MCP 세션마다 5천에서 6천 토큰이 더 든다. 셸 에이전트가 이미 내던 비용과 같은 양이다.
- `mcpserver.New` 의 인자가 셋이 됐다. 다른 호출자가 생기면 무엇을 낼지 정해야 한다.

## 관련

- [0041 skills install은 정적인 스킬 한 벌을 심는다](0041-skills-install-embeds-one-static-skill.md) 스킬이 한 벌인 근거
- [0043 MCP는 쓰기 도구 하나만 노출하고 promote를 뺀다](0043-mcp-exposes-one-write-tool-and-omits-promote.md) 도구 경계. 이 ADR 이 그 위에 지시를 얹는다
- [0077 에이전트 계약은 위키에 남고 스킬은 한 벌이다](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md) 셋째 배달 경로
- [0089 음성은 에이전트가 도구로 부르고 그 동작 구조를 문서에 적는다](0089-voice-is-driven-by-an-agent-and-its-pipeline-is-documented.md) 중복이 생긴 자리
