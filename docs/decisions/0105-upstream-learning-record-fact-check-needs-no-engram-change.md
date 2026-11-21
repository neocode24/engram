---
number: 0105
title: upstream이 학습 기록의 사실 확인 행위를 신설해도 engram은 바꾸지 않는다
date: 2026-11-21
status: accepted
---

# upstream이 학습 기록의 사실 확인 행위를 신설해도 engram은 바꾸지 않는다

## 맥락

upstream 규칙 명세 사본이 lock 68cbdc0에서 48ee7a6 사이에 하나 바뀌었다. `promotion-rules.md`의 Sources Entry 절이 사실 확인 행위를 하나 더 인정한다(2026-11-20).

종전에는 `sources/` 진입의 사실 확인 행위가 교정 커밋(일반)과 다음 세션 참조(`source_channel: agent-session`) 둘이었다. 이제 `workflow: learning-alert` 초안(학습 기록)은 사용자가 그 주제의 학습 질문에 한 번 이상 답한 것을 사실 확인으로 본다. 답 원문과 시각이 문서에 남는 것이 증거다. 더 새 학습 주제가 시작되면 upstream의 학습 자료 동기화 스크립트가 승급 커밋을 만든다. 승급 커밋은 `sources/` 문서 생성, 초안의 `archive/web/` 이동, 자산의 `sources/assets/` 이동으로 이루어진다. 답이 없는 주제는 `inbox/`에 남는다.

같은 기간 upstream `meta/CHANGELOG.md`에는 할 일 원장 규칙(`pin-rules.md`) 항목이 둘 더 있다. 그 명세는 [0104](0104-upstream-pin-rules-are-held-out-of-the-harness.md)가 harness 추적에서 보류했으므로 이 결정의 대상이 아니다.

## 결정

**engram 쪽 아무것도 고치지 않는다.** 명세 사본(`harness/upstream/promotion-rules.md`)과 lock만 맞춘다.

[0100](0100-upstream-sources-entry-and-agent-session-channel-need-no-engram-change.md)이 `agent-session` 채널을 추가한 변경에 대해 내린 결정과 같은 구조다. 사실 확인 행위가 둘에서 셋으로 늘었을 뿐 경계를 넘지 않는다.

- `sources/` 진입 조건은 [spec-map.md 4.6](../spec-map.md)이 코드로 강제하지 않는다고 밝힌 영역이다. "학습 질문에 답했는가"는 문서에 남은 답의 유무를 사람이나 에이전트가 읽어 판단하며, `source` 커맨드가 그 판정을 하지 않는다.
- `workflow`는 개방 집합이다. `internal/config/config.go`의 `Schema` 주석이 "source_channel과 workflow는 개방 집합이라 여기 담지 않는다"고 밝히며, `schema.allowed-value`는 폐쇄 집합 축만 검사한다. `learning-alert`가 값으로 들어와도 lint는 반응하지 않는다.
- 승급 커밋을 만드는 주체는 upstream의 외부 스크립트다. engram 바이너리의 `promote`, `source`, `archive`는 그 경로에 관여하지 않는다.

## 확인

- `internal/lint/lint.go`의 `workflow` 참조는 `team` 프리셋의 필수 필드 존재 검사(399번 줄) 하나뿐이고 값을 대조하지 않는다.
- `internal/config/config.go` 172번 줄 주석이 `workflow`를 개방 집합으로 명시한다.
- `go test ./...` 전체 통과.
- `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ -v` 통과. lint 축과 resurface 축 모두 통과했고, 출력된 매핑 없음 3쌍과 resurface 첫 순위 차이는 이 변경과 무관한 기존 상태다.
- `docs/spec-map.md` 7절 절차의 3번부터 5번(lint 규칙, 프리셋 임계값, 골든 스냅샷)에 해당하는 변경점이 없다. 골든 스냅샷을 재생성하지 않았다.

## 대안

**`workflow` 허용값을 폐쇄 집합으로 만들어 `learning-alert`를 넣는다** 개방 집합을 닫으면 사용자가 정의한 값이 lint를 통과하지 못한다. 이번 변경이 요구하는 것이 아니다.

**승급 조건을 lint 규칙으로 만든다** "답이 문서에 남았는가"는 본문 의미를 읽어야 해서 결정론적으로 셀 수 없다. 4.6절이 이미 이 경계를 코드로 넘기지 않기로 정했다.
