---
number: 0089
title: 음성은 에이전트가 도구로 부르고 그 동작 구조를 문서에 적는다
date: 2026-08-22
status: accepted
---

# 음성은 에이전트가 도구로 부르고 그 동작 구조를 문서에 적는다

## 배경

[0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)가 음성을 별도 바이너리로 정하고 [0080](0080-voice-is-a-nested-module-in-this-repository.md)이 자리를 정했으나, **에이전트가 그것을 어떻게 부르는지는 어느 ADR 도 정하지 않았다.** 그래서 CLI 만 만들어졌다.

같은 이유로 동작 구조를 적은 문서도 없었다. `docs/architecture.md` 에 `voice` 라는 낱말이 0건이고 `docs/journeys.md` 와 `docs/spec-map.md` 에도 없다. `voice/AGENTS.md` 가 유일한 문서인데 그것은 작업 계약이지 동작 설명이 아니다.

## 측정

### 부를 방법이 셸밖에 없었다

| | `engram` | `engram-voice` |
|---|---|---|
| 스킬 문서 | `skills install` 로 위키에 넣는다 | **없음** |
| MCP 도구 | 열 개 | **없음** |
| 에이전트 경로 | 도구 호출 | 셸 명령뿐 |

`internal/skills/SKILL.md` 에 `engram-voice` 언급이 아예 없다. 셸을 쥔 에이전트는 커맨드를 칠 수 있으나 **MCP 만 붙은 클라이언트는 부를 길이 없다.**

### upstream 은 마지막 단계가 프롬프트다

upstream `scripts/voice_memo_local_stt.py` 를 읽었다. 557줄이며 마지막에 `meeting-note-draft-prompt.md` 를 만든다. 아홉 항목의 구조와 규칙 넷이 적혀 있고 **노트는 그 프롬프트를 읽은 에이전트가 쓴다.**

> - Do not infer real speaker names from weak evidence.
> - Keep Speaker IDs when names are unknown.
> - Do not promote to `sources/` or `context/` without user review.

우리 구현에는 이 단계가 통째로 없었다. 전사 텍스트를 내고 끝이었다.

### 문서가 없어 구조를 물어볼 데가 없었다

모델이 몇 개이고 어떤 순서로 도는지가 ADR 여섯에 흩어져 있었다. [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)에 소유, [0081](0081-default-whisper-model-is-large-v3.md)에 모델 크기, [0082](0082-speaker-count-is-asked-not-guessed.md)에 화자 수, [0083](0083-the-glossary-corrects-after-the-fact-and-grows-against-one-model.md)에 사전이다.

**ADR 은 결정의 근거를 남기는 자리이지 지금 무엇이 어떻게 도는지를 설명하는 자리가 아니다.** 본체에는 `docs/architecture.md` 가 그 일을 하는데 음성에는 없었다.

## 판단 근거

### MCP 를 만든다

스킬 문서만으로는 부족하다. 스킬은 셸을 쥔 에이전트에게 "이 커맨드를 쳐라" 고 알려 주는 것이고, 셸이 없는 클라이언트에는 통하지 않는다.

MCP 는 본체가 이미 쓰는 방식이고([0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md)) `internal/mcpserver` 가 루트에 있어 voice 가 그대로 가져다 쓴다. 새로 만들 것이 없다.

**도구는 둘만 둔다.** `transcribe` 와 `model_status` 다. 모델 내려받기는 1.7GB 라 사람이 보고 있어야 하고, 위키에 넣는 것은 engram 쪽 도구다. [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)의 경계가 그대로 유지된다.

`model_status` 를 둔 이유는 실패를 앞당기기 위해서다. 모델이 없는 채로 전사를 부르면 오디오를 변환한 뒤에 실패한다. 먼저 물어보게 한다.

### 진입점이 둘이어도 절차는 하나다

`transcribeAudio` 하나를 CLI 와 MCP 가 함께 부른다. 절차를 두 벌 두면 한쪽만 고쳐지고 **그 차이를 아무도 못 본다.**

이 원칙이 바로 결함 하나를 드러냈다. CLI 는 플래그 기본값이 모델 크기를 채우는데 MCP 는 인자를 생략하면 빈 문자열이 온다. `resolve` 가 빈 값을 거절해 도구가 즉시 실패했다. 빈 값을 기본으로 치도록 고쳤다.

### 회의록 구조는 스킬 문서에 둔다

upstream 은 프롬프트를 파일로 남기지만 우리는 스킬 문서에 적는다. 파일로 남기면 그 파일이 위키에 생기고 그것은 [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)가 그은 "위키에 쓰지 않는다" 를 어긴다.

스킬 문서는 이미 에이전트가 읽는 자리이고 `eject` 가 위키로 내보내는 대상이기도 하다([0077](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md)). 구조를 거기 두면 사용자가 자기 것으로 고칠 수 있다.

upstream 의 규칙 셋을 그대로 옮긴다. 약한 근거로 이름을 지어내지 않는 것, 모르면 번호를 두는 것, 사용자 검토 없이 승급하지 않는 것이다.

### 동작 문서를 따로 만든다

`docs/architecture.md` 에 절을 더하지 않고 `docs/voice.md` 를 만든다. 근거 둘이다.

**모듈이 다르다.** 본체 문서에 CGO 와 외부 프로세스 이야기가 섞이면 본체가 그런 것을 쓴다고 읽힌다.

**선택 사항이다.** 음성을 안 쓰는 독자가 본체 구조를 읽다가 이것을 만나야 할 이유가 없다.

문서에 **무엇을 안 해 봤는지도 적는다.** 실제 오디오로 전사까지 돌려 본 것은 darwin/arm64 하나이고 나머지는 빌드와 단위 시험까지다. 단위 시험이 모델을 열지 않으므로 전사 경로 자체는 CI 가 못 본다. 이것을 안 적으면 다섯 플랫폼이 다 검증된 것으로 읽힌다.

## 결정

| 항목 | 값 |
|---|---|
| 에이전트 접점 | **MCP.** `engram-voice mcp`, stdio |
| 도구 | `transcribe`, `model_status` **둘** |
| 위키 쓰기 | **없다.** engram 의 `capture` 가 한다 |
| 공유 | `transcribeAudio` 하나를 CLI 와 MCP 가 부른다 |
| 회의록 구조 | **스킬 문서에 둔다.** 파일로 위키에 남기지 않는다 |
| 구조 문서 | `docs/voice.md`. `architecture.md` 에 섞지 않는다 |
| 문서에 적을 것 | 모델 넷, 단계 다섯, upstream 과의 차이, **검증 안 된 플랫폼** |
| i18n | **미결.** 지금은 한국어 리터럴뿐이다 |

## 결과

- MCP 만 붙은 클라이언트에서도 전사가 된다. 이 기능의 실제 동선이 열렸다.
- 회의록 초안의 구조가 스킬 문서에 있어 에이전트가 전사 원문을 그대로 붓지 않는다.
- 모델과 단계를 물어볼 곳이 생겼다. ADR 여섯을 뒤지지 않아도 된다.
- 다섯 플랫폼 중 넷이 실제 전사를 안 돌려 봤다는 사실이 문서에 남는다.
- **`engram-voice` 에 i18n 이 없다.** 본체는 카탈로그 여섯에 ko/en 을 갖췄는데 여기는 한국어 리터럴뿐이다. 결정한 적이 없고 그냥 안 한 것이며 미결로 남긴다.

## 관련

- [0079 음성은 별도 바이너리이고 위키는 용어 사전을 소유한다](0079-voice-is-a-separate-binary-in-a-separate-repository.md) 위키에 쓰지 않는다는 경계. 이 ADR 이 그것을 지키며 도구를 연다
- [0043 MCP는 쓰기 도구 하나만 노출하고 promote를 뺀다](0043-mcp-exposes-one-write-tool-and-omits-promote.md) 본체의 MCP 방식. 같은 뼈대를 쓴다
- [0077 에이전트 계약은 위키에 남고 스킬은 한 벌이다](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md) 스킬 문서가 사용자 것이 되는 경로
- [0014 LLM 호출을 바이너리에 두지 않고 에이전트가 바이너리를 부른다](0014-llm-boundary-agent-drives-binary.md) 회의록을 도구가 안 쓰고 에이전트가 쓰는 이유
