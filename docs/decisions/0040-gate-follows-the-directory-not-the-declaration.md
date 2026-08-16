---
number: 0040
title: 게이트는 선언이 아니라 디렉토리를 따르고 artifact_stage 누락은 오류다
date: 2026-08-16
status: accepted
---

# 게이트가 무엇을 보고 발동하는가

## 배경

0.4를 마치고 경계를 손으로 때려 보다 구멍을 찾았다. `context/`에 아래 파일을 손으로 두면 `lint`가 통과시킨다.

```yaml
---
type: concept
status: promoted
---
# 링크 없음
```

```
검사한 파일 4개, error 0, warn 3, reject 0
```

위키링크가 0개인데 승급 게이트가 발동하지 않는다. 세 검사가 모두 빠진다.

| 검사 | 왜 안 도나 |
|---|---|
| `frontmatter.missing-field` | 단계를 모르면 무엇이 필수인지도 모르므로 이르게 반환한다 |
| `location.stage-agreement` | 비교할 값이 없어 이르게 반환한다 |
| `gate.min-wikilinks` | `artifact_stage`가 `context`일 때만 발동한다 |

ADR [0031](0031-location-must-agree-with-stage.md)이 두 번째 검사를 건너뛰기로 하면서 이렇게 적었다.

> `artifact_stage`가 아예 없는 문서는 `frontmatter.missing-field`가 이미 잡는다.

**그 전제가 성립하지 않는다.** 첫 번째 검사도 같은 조건에서 빠지기 때문이다. 서로가 서로를 믿고 둘 다 아무것도 안 한다.

구멍은 더 넓다. `artifact_stage: inbox`를 적고 `context/`에 두어도 게이트는 안 돈다. 선언이 `context`가 아니기 때문이다. 그런데 `resurface`와 `status`는 디렉토리로 단계를 세므로 **그 문서를 검수된 지식으로 취급한다.** 게이트만 지나지 않은 채로 재발견 대상이 되고 인용 대상이 된다.

## 판단 근거

**게이트만 선언을 보고 나머지는 위치를 본다.** 이것이 구멍의 뿌리다. `internal/resurface`와 `internal/status`가 디렉토리로 단계를 세고 그 자리에 "승급은 파일을 옮기는 행위이므로 위치가 운영의 진실원이다"라고 적혀 있다. ADR [0038](0038-migrate-conforms-documents-to-current-rules.md)도 같은 근거로 `migrate`가 프론트매터를 위치에 맞추게 했다. **판정 기준이 하나만 다르면 그 하나가 우회로가 된다.**

**게이트가 지키려는 것은 디렉토리다.** `promote`와 `new`가 게이트를 도는 시점은 문서를 `context/`에 쓰기 직전이다. 지키는 대상이 처음부터 위치였다. lint가 선언을 보는 것은 그 사실과 어긋난다.

**위치를 보면 ADR 0019의 예외가 구조로 바뀐다.** 색인 문서는 위키 루트의 `root_files`이고 `context/` 아래가 아니다. 게이트가 디렉토리를 보면 색인 문서는 자연히 대상 밖이 되므로 예외를 따로 둘 필요가 없다.

**누락은 그 자체로 오류다.** `artifact_stage`는 단계 판정의 입력이며 값이 없으면 뒤따르는 판정 대부분이 성립하지 않는다. 다른 검사가 잡아 주기를 기대하지 않고 여기서 막는다.

**누락일 때 다른 필수 필드는 보고하지 않는다.** 어느 단계인지 모르므로 무엇이 필수인지도 모른다. 디렉토리로 추정해 보고하면 사용자가 채운 뒤에 판정이 또 바뀔 수 있다. 한 번에 한 가지만 말한다. 값을 채우면 다음 실행이 나머지를 본다.

**영향이 작다.** 실운영 위키 306문서에서 `artifact_stage`가 없는 문서가 둘이고 둘 다 `inbox/`다. 게이트 대상이 되는 `context/` 문서 중 선언이 어긋난 것은 넷이며 전부 MOC라 링크가 많다. 그리고 `migrate`가 둘 다 정리한다.

## 결정

**게이트는 문서가 놓인 디렉토리로 발동한다. `artifact_stage` 선언을 보지 않는다.**

| 항목 | 값 |
|---|---|
| 게이트 대상 | `context` 단계에 대응하는 디렉토리 아래 문서 |
| 제외 | `root_files`(ADR [0019](0019-index-documents-outside-the-gate.md)). 루트에 있어 구조적으로 대상 밖이다 |
| 링크 대상 집계 | 그대로 유지한다(ADR [0023](0023-gate-targets-exclude-inbox.md)) |
| 유예 | 그대로 유지한다(ADR [0021](0021-gate-deferral-when-targets-are-scarce.md)) |

**`artifact_stage`가 없으면 `error`다.**

| 항목 | 값 |
|---|---|
| 규칙 ID | `frontmatter.missing-field` |
| 등급 | `error` |
| 조건 | `artifact_stage` 축이 켜져 있고 문서에 그 필드가 없다 |
| 함께 보고하지 않는 것 | 그 문서의 다른 필수 필드. 단계를 모르기 때문이다 |

- 단계와 디렉토리의 대응은 `internal/wiki`가 단일 진실원이다. 매핑을 두 벌 두지 않는다.
- `promote`와 `new`의 게이트 판정은 지금도 도착지 기준이므로 바뀌지 않는다.
- `location.stage-agreement`의 누락 시 건너뛰기는 그대로 둔다. 이제 `frontmatter.missing-field`가 실제로 잡으므로 0031의 전제가 성립한다.

## 결과

- `context/`에 손으로 둔 문서가 게이트를 지난다. 선언을 비우거나 낮춰도 우회할 수 없다. **게이트가 유일한 관문이라는 전제가 실제로 성립한다.**
- 실운영 위키에서 `error`가 둘 는다. `migrate`가 정리한다.
- `context/`에 있으면서 선언이 어긋난 문서 넷이 게이트 대상이 된다. 전부 링크 허브라 통과한다.
- ADR 0019의 색인 문서 예외가 구조적 결과가 된다. 명시적 제외를 지워도 동작이 같다.
- 골든 픽스처에 이 경로를 때리는 문서가 없다. 새로 넣는다.
- 내보낸 Python 린터가 함께 바뀐다. `harness/eject`의 대조가 그것을 강제한다.

## 관련

- [0031 문서가 놓인 디렉토리와 artifact_stage가 일치해야 한다](0031-location-must-agree-with-stage.md) 성립하지 않은 전제의 출처
- [0019 색인 문서를 승급 게이트와 고아 판정 대상에서 제외한다](0019-index-documents-outside-the-gate.md) 예외가 구조로 바뀐다
- [0021 링크 대상이 부족하면 승급 게이트를 유예한다](0021-gate-deferral-when-targets-are-scarce.md) 유예는 그대로다
- [0023 게이트의 링크 대상 집계에서 inbox 문서를 제외한다](0023-gate-targets-exclude-inbox.md) 집계는 그대로다
- [0035 위치와 단계의 불일치는 방향에 따라 등급을 나눈다](0035-stage-mismatch-severity-by-direction.md) 선언이 낮은 방향의 해악을 이 ADR이 다시 본다
