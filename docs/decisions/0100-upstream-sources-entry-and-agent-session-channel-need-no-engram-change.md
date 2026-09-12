---
number: 0100
title: upstream이 sources 진입 기준과 agent-session 채널을 신설해도 engram은 바꾸지 않는다
date: 2026-09-12
status: accepted
---

# upstream이 sources 진입 기준과 agent-session 채널을 신설해도 engram은 바꾸지 않는다

## 맥락

upstream `meta/promotion-rules.md`와 `meta/wiki-artifact-schema.md`가 커밋 df07b29에서 bf401e4 사이(789d041, 1669ee5)에서 두 가지를 신설했다.

첫째, `promotion-rules.md`가 `inbox/`에서 `context/`로 가는 조건만 있던 것을 둘로 나눴다. `inbox/`에서 `sources/`로 가는 진입 조건(사실 교정 또는 확인 1회, `sensitivity` 결정, `source_refs` 존재)을 신설하고, 재사용 가치 판단은 `context/` 진입 조건으로만 남겼다. `source_channel: agent-session` 초안은 교정 대신 "다음 세션에서 근거로 참조됨"을 확인 행위로 본다.

둘째, `wiki-artifact-schema.md`가 `source_channel` 폐쇄 집합처럼 보이는 예시 목록에 `agent-session`을 추가했다. Hermes 또는 터미널 에이전트와의 대화에서 나온 결론을 가리키며, `source_refs`에 파일 경로 대신 `hermes-session:<session_id>` 식별자를 쓸 수 있다고 적었다.

## 결정

**engram 쪽 아무것도 고치지 않는다.**

두 변경 모두 engram이 이미 사람의 판단으로 남겨 둔 자리 안에 있다.

`sources/` 진입 조건은 [spec-map.md 4.6](../spec-map.md)이 "코드로 강제하는 것. 문서 단위 규칙은 없다"고 이미 밝혀 둔 영역이다. `source` 커맨드(`internal/cli/source.go`)는 검증 없이 필드를 채워 `sources/`에 쓸 뿐, "교정이 있었는가", "사실 확인이 됐는가"를 판정하지 않는다. 이 판정을 코드가 대신할 수 없다는 것은 이번 변경 이전부터의 설계이지, 이번에 생긴 공백이 아니다. 조건이 셋으로 늘어나도 여전히 사람(또는 대화 중인 에이전트)이 판단해서 `source` 커맨드를 실행하는 구조는 그대로다.

`source_channel`은 [config.go](../../internal/config/config.go)의 주석대로 "개방 집합이라 여기 담지 않는다." `AxisSourceChannel`은 켜고 끄는 축일 뿐 값의 허용 집합을 코드가 검사하지 않는다. `agent-session`이라는 새 값이 예시 목록에 추가되어도 lint의 `schema.allowed-value`(`internal/lint/lint.go`)는 `valueFields()`가 나열한 다섯 축(`artifact_stage`, `status`, `scope`, `sensitivity`, `trigger_mode`)만 검사하고 `source_channel`은 애초에 포함하지 않는다. `--channel` 플래그도 임의 문자열을 그대로 받는다(`internal/cli/source.go`의 `applyChannel`). `hermes-session:<id>` 형식의 `source_refs` 값도 마찬가지로, `source_refs`는 코드가 형식을 검증하지 않는 자유 문자열 목록이다(`internal/graph/graph.go`의 `KindSourceRefs`는 필드 존재와 위키링크 카운트만 본다).

## 확인

- `internal/cli/source.go`에 사실 교정 여부나 확인 이력을 판정하는 코드가 없다.
- `internal/lint/lint.go`의 `valueFields()`에 `source_channel` 축이 없다.
- `internal/config/config.go` 172번 줄 주석이 `source_channel`을 개방 집합으로 명시한다.
- `go test ./...` 전체 통과.
- `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ -v` 통과. lint 축, resurface 축 모두 이 변경 전후로 갈림이 없다.
- `docs/spec-map.md` 6절 절차의 3~4번(lint 규칙, 프리셋 임계값)에 해당하는 변경점이 없다.
- 골든 스냅샷을 재생성할 필요가 없었다.

## 대안

**`agent-session`을 `engram.yaml` 문서나 예시에 미리 적어 둔다** — 개방 집합의 예시 값 하나를 문서에 옮겨도 코드 동작은 달라지지 않는다. 오히려 폐쇄 집합처럼 보이게 하는 오해를 만들 수 있어 적지 않는다.

**`sources/` 진입 조건 셋을 lint 규칙으로 만든다** — "교정이 있었는가"는 git 이력 대조가 필요하고 "사실 확인이 됐는가"는 사람의 판단이라 결정론적으로 셀 수 없다. spec-map 4.6이 이미 이 경계를 코드로 넘기지 않기로 정했고, 이번 변경은 그 경계 안에서 조건을 세분화한 것일 뿐 경계 자체를 넘지 않는다.
