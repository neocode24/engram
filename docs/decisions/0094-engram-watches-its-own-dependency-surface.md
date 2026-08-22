---
number: 0094
title: engram이 upstream 의존 표면을 스스로 감시하고 impact 표시에 기대지 않는다
date: 2026-08-22
status: accepted
---

# engram이 upstream 의존 표면을 스스로 감시하고 impact 표시에 기대지 않는다

## 배경

[0029](0029-upstream-vendoring-and-parity-execution.md)가 delta 를 upstream 의 `meta/CHANGELOG.md` 에서 만들고, 그 항목의 `impact: binary-affecting` 을 "engram 이 따라가야 할 항목" 으로 삼았다.

그 구조는 **upstream 이 engram 을 알아야 성립한다.** 실제로 upstream `AGENTS.md` 가 이렇게 적고 있었다.

> 이 로그가 필요한 이유는 downstream(`engram`)이 upstream 규칙과의 동등성을 harness로 검증하기 때문이다.

판단을 저쪽에 맡긴 것이다. upstream 이 `impact` 를 안 붙이거나 잘못 붙이면 engram 은 아무것도 모른다.

## 측정

### 파일별 유지/갱신 신호가 죽어 있었다

vendored 파일 머리말에 출처 커밋이 박혀 있었다.

    <!-- upstream llm-wiki meta/frontmatter-schema.md 에서 가져왔다.
         원본 커밋 05f7279. 익명화 치환 6건을 적용했다. -->

그래서 upstream HEAD 가 움직이면 **본문이 한 글자도 안 바뀌어도 일곱 개 전부 `갱신` 으로 나왔다.** 실측으로 확인했다. 18커밋을 sync 했을 때 `--check` 가 일곱 전부 `갱신` 이라고 냈는데, 본문 diff 는 이것뿐이었다.

    -     원본 커밋 05f7279. 익명화 치환 6건을 적용했다.
    +     원본 커밋 eacb0f6. 익명화 치환 6건을 적용했다.

**규칙 명세는 바이트 단위로 같았다.** 이미 갖고 있던 신호를 스스로 못 쓰게 만들어 두고 `impact` 를 읽고 있었던 것이다.

### 용어 사전의 형식 의존이 아무에게도 안 보였다

`internal/glossary` 가 upstream `meta/terminology-normalization.md` 의 표 구조에 기댄다. 네 칸짜리 표이고 셋째 칸이 `yes` 로 시작하는 행만 자동 치환한다.

그런데 그 파일은 사전 전체가 조직 어휘라 vendoring 대상이 아니다([0029](0029-upstream-vendoring-and-parity-execution.md)). 스냅샷이 없으므로 비교할 대상도 없다.

형식이 바뀌면 **파서가 조용히 망가진다.** 예외를 던지지 않고 전부 `review` 로 읽혀 치환 0건이 된다. 변이 시험으로 확인했다. 셋째 칸 어휘를 `yes` 에서 `auto` 로 바꾸면 자동 교정이 전부 사라진다.

### upstream 이 downstream 을 알고 있었다

위의 `AGENTS.md` 문장과 `meta/CHANGELOG.md` 서두 둘이다. upstream 은 그 자체로 완결되어야 하고, 참조 관계는 engram 쪽에만 있어야 한다.

## 판단 근거

### 판단을 이쪽으로 가져온다

무엇이 engram 에 영향을 주는지는 **engram 이 안다.** upstream 이 그것을 대신 판정하면 두 가지가 어긋난다. upstream 이 규칙 밖이라고 본 것이 engram 에는 영향을 줄 수 있고(용어 사전 형식이 그렇다), 반대도 마찬가지다.

그래서 트리거를 `impact` 에서 **engram 자신의 diff** 로 옮긴다. 머리말에서 커밋을 빼면 파일별 `유지/갱신` 이 곧 "이 규칙 명세가 실제로 바뀌었나" 가 된다. 커밋은 `harness/upstream.lock` 하나가 가진다.

### CHANGELOG 는 버리지 않고 강등한다

diff 는 **무엇이** 바뀌었는지만 알려주고 CHANGELOG 는 **왜** 바꿨는지를 알려준다. 사람이 반영 여부를 판단할 때 그 맥락이 필요하다. 계속 읽되 판정의 근거로 삼지 않는다.

### 사전은 내용이 아니라 형식만 뜬다

`terminology-normalization.md` 를 vendoring 할 수 없다는 [0029](0029-upstream-vendoring-and-parity-execution.md)의 판단은 그대로다. 사전 항목은 전부 조직 어휘다.

engram 이 기대는 것은 사전 내용이 아니라 **표 형식**이다. 형식만 뜨면 경계를 넘지 않는다. 칸 이름은 일반 명사이고, 셋째 칸은 첫 낱말만 본다. 값 전체에는 조직 식별자가 들어 있어 그대로 두면 안 된다.

행 수는 세지 않는다. 사전에 항목 하나만 늘어도 지문이 바뀌면 내용 변화와 형식 변화를 구분할 수 없게 된다. 실측으로 확인했다. 항목을 하나 더한 upstream 에서 지문이 같았고, 셋째 칸 어휘를 바꾼 upstream 에서 지문이 달랐다.

머리글은 이름으로 찾지 않고 **다음 줄이 구분줄인 행**으로 찾는다. `internal/glossary` 가 쓰는 규칙과 같아야 지문이 파서의 전제를 대변한다.

## 결정

| 항목 | 값 |
|---|---|
| 판정 근거 | **engram 자신의 diff.** `impact` 를 읽지 않는다 |
| vendored 머리말 | 출처 커밋을 **빼고** 치환 건수만 남긴다. 커밋은 lock 하나가 가진다 |
| 파일별 신호 | `유지` 는 규칙 명세가 그대로, `갱신` 은 실제로 바뀌었다는 뜻이 된다 |
| 용어 사전 | 내용은 그대로 제외. **표 형식 지문**을 `harness/upstream/terminology-format.md` 로 뜬다 |
| 지문에 담는 것 | 표 머리글, 자동 교정 칸의 첫 낱말 집합. **행 수와 값 전체는 담지 않는다** |
| `meta/CHANGELOG.md` | 계속 읽어 delta 에 남긴다. **판정 근거가 아니라 사람이 읽을 맥락이다** |
| upstream | engram 참조를 뺀다. 저쪽은 저쪽으로 완결된다 |

## 결과

- upstream 이 `impact` 를 잘못 붙여도 engram 이 알아챈다. 안 붙여도 마찬가지다.
- 머리말 형식이 바뀌므로 vendored 일곱 개가 한 번 전부 갱신된다. 그다음부터 `유지` 가 나온다.
- 용어 사전 형식 변경이 처음으로 감지 대상이 된다. 그 전에는 아무 장치도 없었다.
- Hermes 주간 잡의 판정 기준이 `binary-affecting N건` 에서 파일별 `유지/갱신` 으로 바뀐다.
- upstream `AGENTS.md` 와 `meta/CHANGELOG.md` 서두에서 downstream 참조를 뺐다. `impact` 값 이름은 그대로 두었다. 기존 항목들이 쓰고 있고 upstream 자신의 분류로 남는다.

## 열린 항목

- upstream `AGENTS.md` 본문의 규범문과 단계 디렉토리 `README.md` 는 여전히 아무도 안 본다. 계약 선언 밖이라 CHANGELOG 규율 대상이 아니고 vendoring 대상도 아니다([spec-map](../spec-map.md) 6.2절). 같은 방식으로 지문을 뜰 수 있으나 이번에는 하지 않았다.
- `impact` 를 안 읽게 되었으므로 upstream 이 그 값을 계속 쓸지는 저쪽이 정한다. engram 은 어느 쪽이든 상관없다.

## 관련

- [0029 upstream 계약을 vendoring 하고 parity 를 실행한다](0029-upstream-vendoring-and-parity-execution.md) delta 를 CHANGELOG 에서 만들기로 한 결정. 이 ADR 이 트리거를 옮긴다
- [0030 upstream delta 는 공개 산출물이 아니다](0030-upstream-delta-is-not-a-public-artifact.md) delta 가 `private/` 로 가는 이유
- [0083 용어 사전은 후처리만 하고 쓰는 모델에 붙어 자란다](0083-the-glossary-corrects-after-the-fact-and-grows-against-one-model.md) 형식에 기대는 파서가 사는 자리
