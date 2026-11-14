---
number: 0103
title: upstream이 설계 상태 필드 여섯을 신설해도 engram은 코드를 바꾸지 않고 미이행으로 기록한다
date: 2026-11-14
status: accepted
---

# upstream이 설계 상태 필드 여섯을 신설해도 engram은 코드를 바꾸지 않고 미이행으로 기록한다

## 맥락

upstream 규칙 명세 사본이 lock 3f93986에서 68cbdc0 사이에 하나 바뀌었다. `frontmatter-schema.md`가 특정 도메인의 설계 문서에만 쓰는 선택 필드 여섯을 신설했다(2026-11-11). `architecture_state`, `baseline`, `entity_types`, `domain`, `layer`, `biz_codes`다.

규칙은 셋이다. `architecture_state`는 폐쇄 집합이며 값은 `historical-design`, `observed-as-is`, `target-proposal`, `target-approved`, `target-rejected` 다섯이다. `architecture_state`가 있으면 `baseline`이 비어 있지 않은 문자열이어야 한다. `entity_types`가 있으면 `architecture_state`가 있어야 한다. 나머지 셋은 개방 집합이다. 과거 설계서가 현재 구현처럼 읽히는 것을 막으려는 필드다.

같은 기간 `CHANGELOG`의 `wiki-only` 항목 하나가 더 있다. 도메인 이름 대응 문서의 원본 소유권을 정리한 것이며 스키마 값은 바꾸지 않았다. 이 저장소가 따라갈 것이 없다.

upstream `scripts/lint-frontmatter.sh`가 같은 날 검사 15줄을 갖췄다.

## 결정

**engram 코드는 고치지 않는다.** `docs/spec-map.md` 4.1절과 6절에 **미이행**으로 적는다. 이유와 결정할 시점을 함께 적는다.

## 근거

임시 위키를 만들어 같은 입력을 engram `lint`와 upstream 스크립트에 돌렸다. 입력은 `context/` 문서 다섯이다. 잘못된 `architecture_state` 값, `baseline` 누락, `baseline` 빈 값, `architecture_state` 없는 `entity_types`, 올바른 조합이다.

| 입력 | upstream | engram |
|---|---|---|
| `architecture_state: bogus` | FAIL | 위반 없음 |
| `architecture_state` 있고 `baseline` 없음 | FAIL | 위반 없음 |
| `architecture_state` 있고 `baseline` 빈 값 | FAIL | 위반 없음 |
| `entity_types`만 있음 | FAIL | 위반 없음 |
| 올바른 조합 | 통과 | 위반 없음 |

판정이 갈리는 것은 앞의 네 줄이다. engram은 이 필드 여섯을 모르는 키로 지나친다. 그럼에도 코드를 바꾸지 않는 이유는 넷이다.

**필드를 쓰는 문서가 없다.** upstream CHANGELOG가 이 필드를 가진 기존 문서는 0건이라고 적었다. engram이 만든 위키에서도 `new`와 `promote`가 이 필드를 쓰지 않으므로 생기지 않는다. 사용자가 손으로 적어야 생긴다.

**특정 도메인의 선택 필드다.** 속성 14종은 모든 위키가 공유하는 어휘이고 `axes`로 켜고 끈다([0071](0071-lint-checks-indexable-stage-and-deprecated-fields.md)). 이 여섯은 한 도메인의 설계 문서에만 쓰이며 값 사전(`domain`, `layer`, `baseline`)의 원본도 upstream의 다른 문서에 있다. 범용 도구가 폐쇄 집합 하나와 조건부 필수 둘만 옮기면 값 사전 없이 반쪽 검사가 된다.

**넣는 비용이 크다.** 규칙이 하나 늘면 `rules show`의 규칙 수와 문서의 정량 주장 여럿이 함께 움직이고 `harness/docs`가 대조한다([0096](0096-docs-harness-checks-quantitative-claims.md)). 골든 스냅샷과 `eject`가 내보내는 린터도 같이 바뀐다([0039](0039-eject-emits-rule-specs-and-a-python-linter.md)). 검사 대상이 되는 문서가 0건인 규칙에 쓰기에는 크다.

**유지보수 방침과 맞는다.** [0102](0102-engram-stays-in-maintenance-and-adapts-to-the-learner-toolchain.md)가 정한 대로 engram은 수강생의 초기 구축 도구이고 도메인별 확장은 `eject` 이후 사용자가 자기 운영에 맞춰 붙이는 영역이다.

## 확인

- `go test ./...` 통과.
- `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ -v` 통과. lint 축은 양쪽 다 잡음 49쌍이다.
- **이것은 새 규칙이 동등하다는 증거가 아니다.** `harness/fixtures/golden-wiki`에 이 필드를 가진 문서가 없어 parity가 이 규칙을 비교하지 않는다. 위 임시 위키 실측이 실제 근거다.
- 골든 스냅샷을 재생성하지 않았다. 코드가 바뀌지 않았다.
- `docs/spec-map.md` 6절(정정: 7절 「규칙이 바뀌었을 때 무엇을 함께 고치나」) 절차의 3번(lint 규칙과 임계값)과 4번(프리셋 기본값)에 고칠 것이 없다.

parity의 resurface 축은 이번 실행 이전부터 실패하고 있었다. upstream `wiki_resurface.py`가 같은 디렉토리의 `wiki_link.py`를 부르도록 바뀌었는데 비교 하니스가 `wiki_resurface.py`만 임시 위키로 복사했기 때문이다(`ModuleNotFoundError`). `harness/parity/parity_resurface_test.go`가 `wiki_link.py`도 함께 복사하도록 고쳤다. 이 고침은 이 lock과 무관하게 upstream 스크립트 구조 변화를 따라간 것이다.

## 대안

**검사 셋을 지금 넣는다** — 검사 자체는 결정론적이고 upstream과 판정이 같아진다. 그러나 위 비용을 치르고도 걸릴 문서가 없다. 값 사전이 없는 반쪽 검사라는 점도 남는다.

**여섯 필드를 설정으로 켜고 끄는 속성으로 추가한다** — `axes` 구조에 맞는다. 그러나 속성 14종이 20종이 되어 프리셋, `rules show`, 교재의 수치가 모두 바뀐다. 한 도메인의 요구 때문에 모든 수강생이 보는 어휘를 늘리는 결정이며 harness 에이전트가 정할 크기가 아니다.

## 열린 항목

- 결정할 시점은 이 필드를 쓰는 위키를 engram으로 운영하는 사용자가 실제로 나타날 때, 또는 upstream이 같은 종류의 필드를 범용 어휘로 올릴 때다. 그때 픽스처에 사례를 더해 parity가 이 규칙을 비교하게 한다.
