---
number: 0109
title: Pin이 upstream의 시작일, 선행 완료 시 대기 해제, 제안 카드 하나 규칙을 따라간다
date: 2026-11-28
status: accepted
---

# Pin이 upstream의 시작일, 선행 완료 시 대기 해제, 제안 카드 하나 규칙을 따라간다

## 맥락

[0107](0107-harness-judges-pin-surfaces-and-pulls-only-the-page.md)에 따라 주간 harness가 upstream Pin의 표면을 견본에 돌려 판정했다. lock 85b791b에서 b3c3092 사이에 `pin 판정: PR`이 나왔고 바뀐 표면은 상태 형태, 상태 값, 상수, 화면 요청, 규칙 문서 다섯이다. 게이트는 지문이 같았으나 `add_relation`과 `set_due`의 동작이 바뀌어 견본 요청에 걸리지 않았다. 반영 전 `harness/pin`은 TestPinParity 다섯 회차와 TestPinSurface가 모두 실패했다.

upstream `meta/CHANGELOG.md`의 이 구간 항목은 다섯이다.

- 새 일 제안이 열린 일은 담당자 확인 카드를 따로 만들지 않는다. 제안 카드가 담당자, 상태, 기한을 선택 줄로 갖고, 단추는 맞음과 아님 둘이다.
- 선행이 완료되면 그 일을 기다리던 대기(선행)를 할 일로 자동으로 푼다. 항목에 `freed_by`가 붙고 오늘 탭에 확인 카드가 생긴다.
- 사람(`web`, `command`)이 맺는 연관은 이유가 선택이다. 자동화가 맺는 연관은 종전대로 이유가 있어야 한다.
- 사람이 정하는 시작일 `start`와 연산 `set_start`가 생겼고, 예정의 자동 시작이 기한 30일 전에서 14일 전으로 바뀌었다.
- 기한이 먼 할 일은 예정으로 두고 프로젝트 신호와 오래 기록 없음 질문에서 뺀다. 프로젝트에 열린 일이 없으면 신호가 `none`(없음)이다.

## 결정

**Go 구현을 upstream에 맞춘다.** 위 다섯은 모두 원장 파일 형식, 거절 코드와 문장, 상태 값에 드러나는 변경이라 [spec-map 4.10](../spec-map.md)의 "같은 것"에 든다. 일부러 달리할 이유가 없다.

고친 곳은 `internal/pin`이다.

- `consts.go`: 선택 칸 `start`, `freed_by`, 변경 이력 필드 `start`, 연산 `set_start`(web, command), 신호 라벨 `none`, 기본값 `later_days` 14.
- `apply.go`: `opSetStart`, `releaseWaiters`, `releaseDonePriors`, `mark_seen`이 `freed_by`도 지운다, 사람이 맺는 연관의 이유 면제, 기한을 지우거나 앞당기면 시작을 지운다.
- `check.go`: `start`와 `freed_by` 형식 검사, 사람이 맺은 연관의 빈 이유 허용, `config.signal.later_days` 검사.
- `derive.go`: `laterFrom`(예정), 이번 주 끝(일요일) 기준 주의, 진행 중은 3일 안이어도 주의, `later_count`, 신호 `none`, 상태의 `later`, `start`, `freed_by`, `signal_rules.later_days`, 담당자 확인 카드 생략, 대기 해제 카드.
- 화면은 `--pull-page`로 가져왔다. 가져온 화면의 CSP 해시는 임베드한 파일에서 계산하므로 손댄 것이 없다.

**견본에 여섯째 회차를 더한다.** `harness/pin/fixtures/batches/06-start-release.json`이다. 다섯 회차는 `set_start`, 선행 완료 시 대기 해제, 이유 없는 연관을 지나지 않으므로 이 가지의 어긋남을 못 잡는다. 반영 전 Go는 이 요청에서 upstream과 결과가 달랐다.

**크롬 시험의 가정을 고친다.** `harness/pin/browser/probe.mjs`가 "맞음을 누르면 카드가 줄어든다"를 가정했다. 새 규칙에서는 담당자가 추정인 일에서 담당자를 고르지 않고 맞음을 누르면 제안이 닫힌 뒤 담당자 확인 카드가 따로 생겨 카드 수가 줄지 않는다. upstream 규칙 문서가 이 동작을 명시한다. 시험이 담당자를 `나`로 고른 뒤 맞음을 누르도록 바꿨다. Go 구현이 아니라 시험의 가정이 어긋난 것이다.

## 확인

- `go test ./...` 전체 통과.
- `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ ./harness/pin/ -v` 통과. TestPinParity 여섯 회차와 TestPinSurface 포함.
- `TestPinPageInChrome`이 `PASS`. 탭 넷, 카드 2에서 1, 콘솔 오류와 외부 요청 0, 서버를 내린 뒤 페이지 이동 0.
- `upstream-sync.py --check` 마지막 줄이 `pin 판정: 유지`.

## 대안

**크롬 시험을 그대로 두고 Go가 맞음 뒤 담당자 확인 카드를 만들지 않게 한다.** upstream 규칙 문서가 "담당자를 고르지 않고 맞음을 누르면 담당자 확인 카드가 따로 생긴다"고 적고 있어 upstream과 달라진다. 같은 것으로 두기로 한 영역이다.

**견본 다섯 회차의 요청을 고쳐 새 가지를 넣는다.** 앞 회차의 원장 상태에 뒤 회차가 기대고 있어 어느 회차를 바꿔도 뒤 회차 결과가 함께 바뀐다. 회차를 하나 더하는 쪽이 기존 기대값을 건드리지 않는다.
