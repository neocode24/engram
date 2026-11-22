---
number: 0104
title: upstream이 할 일 원장 규칙을 계약으로 선언해도 harness는 한동안 따라가지 않는다
date: 2026-11-20
status: amended
---

# upstream이 할 일 원장 규칙을 계약으로 선언해도 harness는 한동안 따라가지 않는다

## 맥락

upstream은 지식 계층(`inbox`, `sources`, `context`) 옆에 할 일 원장을 운영한다. 위키 문서와 다른 출처에서 할 일을 모아 상태를 관리하고, 원장을 고치는 곳은 적용 스크립트 하나뿐이다. 그 규칙을 `meta/pin-rules.md`로 정리하고 계약 파일로 선언한다(2026-11-20).

계약 파일로 선언되면 `scripts/upstream-sync.py`가 다음 실행에서 그 파일을 읽고, 주간 harness 잡은 `신규` 또는 `갱신`을 보고 브랜치를 파고 PR을 연다([0095](0095-hermes-is-the-harness-agent-and-the-pr-is-the-approval-point.md)). 지금까지의 계약 파일은 engram이 이미 구현한 규칙의 변경분이었지만, 할 일 원장은 engram에 없는 기능 전체다. 사람이 없는 회차가 기능 하나를 통째로 설계하고 구현하게 된다.

## 결정

**`pin-rules.md`는 vendoring하지 않고 유지와 갱신도 판정하지 않는다.** `scripts/upstream-sync.py`의 `HELD`에 파일명과 이 ADR 번호를 둔다. 계약 선언에서 읽은 목록에서 빼고, 뺐다는 사실을 출력에 한 줄 남긴다.

engram에 할 일 원장을 둘지, 둔다면 어떤 범위로 둘지는 별도 세션에서 정한다. 그때 새 ADR을 쓰고 `HELD`에서 이 파일을 지운다.

## 근거

**판단할 것이 많아 harness가 정할 크기가 아니다.** 원장 규칙은 출처마다 다른 연결(미리알림, 채팅 반응, 위키 추출 에이전트)을 전제로 한다. engram 바이너리는 LLM을 부르지 않는다([0014](0014-llm-boundary-agent-drives-binary.md)). 그래서 어느 부분이 결정론적 연산이고 어느 부분이 에이전트의 판단인지부터 갈라야 한다. 쓰기 게이트를 어디에 둘지도 정해야 한다. [0103](0103-upstream-design-state-fields-are-recorded-as-unimplemented.md)은 필드 여섯을 넣는 일조차 harness 에이전트가 정할 크기가 아니라고 적었다.

**미이행 기록으로는 부족하다.** [0101](0101-upstream-inbox-exit-rule-is-recorded-as-unimplemented.md)과 [0103](0103-upstream-design-state-fields-are-recorded-as-unimplemented.md)처럼 PR마다 미이행으로 적는 길도 있다. 그러나 원장 규칙은 운영하며 자주 바뀔 문서라 매주 같은 미이행 PR이 온다. [0095](0095-hermes-is-the-harness-agent-and-the-pr-is-the-approval-point.md)가 막으려던 "볼 것이 없는 PR"이다.

**제외 목록을 코드에 둔다.** 잡 프롬프트에 "이 파일은 무시하라"를 적으면 에이전트가 지키는지 확인할 길이 없다. `upstream-sync.py`가 목록에서 빼면 `--check` 출력에 그 파일이 나오지 않으므로 잡이 판정할 거리 자체가 없다. `terminology-normalization.md`를 빼는 방식([0029](0029-upstream-vendoring-and-parity-execution.md))과 같다.

## 대안

**upstream 계약 선언에 넣지 않는다.** 그러면 upstream의 `meta/CHANGELOG.md` 규율이 원장 규칙에 걸리지 않아, 규칙이 바뀌어도 기록이 남지 않는다. 나중에 engram이 따라갈 때 변화 이력을 잃는다. 기각한다.

**harness 잡 프롬프트에서 제외한다.** 위 근거대로 강제되지 않는다. 기각한다.

## 확인

- 가짜 upstream(계약 선언에 `pin-rules.md`를 더한 임시 저장소)으로 `--check`를 돌려 `pin-rules.md`가 판정 목록에 없고 제외 줄이 나오는 것을 봤다.
- `go test ./...` 통과.
