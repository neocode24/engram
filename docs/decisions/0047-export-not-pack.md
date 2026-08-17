---
number: 0047
title: 반출 커맨드의 이름을 export로 정정한다
date: 2026-08-17
status: accepted
---

# 반출 커맨드의 이름

## 배경

[0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md)이 여정 14의 반출 커맨드를 `pack`으로 구현했다. 근거로 [0006](0006-dual-mode-eject-seal.md)의 예약을 들었다. 그 ADR의 표에 이렇게 적혀 있다.

> | `engram pack` | 배포와 공유용 번들 생성 | 예약어. 여정 14, 15에서 사용 |

**같은 문서의 본문은 다르게 적고 있다.** "커맨드 이름 선택" 절이다.

> `pack`을 모드 전환에 쓰지 않은 이유는 배포 번들 자리에 그 이름이 필요하기 때문이다. `export`와 `import`도 같은 이유로 기각했다. **반출(여정 14)과 외부 자료 수집(여정 3)이 그 이름을 먼저 차지한다.**

본문은 모드 전환 후보에서 `export`를 뺀 이유로 "여정 14의 반출이 그 이름을 쓸 것"을 들었다. 즉 0006 본문 기준으로 **여정 14는 `export`이고 `pack`은 배포 번들이라는 별개 자리**다. 0046은 표만 보고 `pack`을 골랐다.

## 판단 근거

### 이름이 자기 결정과 어긋난다

표와 본문의 불일치보다 이쪽이 무겁다. [0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md)의 결정 첫 줄이다.

> 병합도 압축도 포맷 변환도 하지 않는다

그런데 `pack`은 관례상 **하나의 아카이브로 묶는 동작**이다. `npm pack`은 tarball을 만들고 `git archive`는 tar와 zip을 만들고 `cargo package`는 `.crate` 파일을 만든다. `engram pack`을 처음 보는 사람은 압축 파일을 기대하는데 디렉토리가 나온다.

**이름이 매번 해명을 요구하면 이름을 잘못 고른 것이다.**

### export는 이 동작의 표준 명칭이다

위키와 노트 도구가 같은 동작을 전부 export라 부른다. Confluence의 space export, Notion과 Obsidian과 Roam의 export가 모두 "고른 문서를 파일 묶음으로 낸다"이며 압축 여부는 선택이다. 학습 비용이 없다.

### eject와 가까워지는 것은 감수한다

`eject`가 이미 있고 `export`와 앞 글자가 겹친다. 그러나 대상이 갈린다. **`eject`는 규칙 소유권을 넘기고 `export`는 문서를 내보낸다**([0013](0013-eject-redefined-seal-removed.md), [0039](0039-eject-emits-rule-specs-and-a-python-linter.md)). 셸 완성도 두 글자면 갈린다.

"압축한다는 기대를 매번 배신하는 비용"이 "앞 두 글자가 겹치는 비용"보다 크다.

### 지금이 마지막 기회다

1.0을 출하하면 커맨드 이름은 호환성 대상이 된다. 아직 태그를 밀지 않았고 릴리스가 나가지 않았으므로 **비용이 커밋 하나다.**

### pack이라는 이름은 비워 둔다

[0006](0006-dual-mode-eject-seal.md) 본문이 `pack`에 준 자리는 배포 번들이었다. 그 요구가 실제로 나오면 그때 쓴다. 지금 반출에 그 이름을 쓰면 진짜 압축 번들이 필요해졌을 때 이름이 없다.

### internal/expose는 그대로 둔다

노출 판정 패키지의 이름은 유지한다. CLI에 나오지 않는 내부 이름이고 `serve`와 `export`가 "무엇이 밖에 보이는가"를 함께 판정하는 자리라 뜻이 정확하다.

## 결정

**반출 커맨드의 이름은 `engram export`다.**

| 항목 | 값 |
|---|---|
| 커맨드 | `engram pack` -> `engram export` |
| 패키지 | `internal/pack` -> `internal/export` |
| 동작 | [0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md)이 정한 그대로다. **하나도 바뀌지 않는다** |
| `pack` | 배포 번들 자리로 다시 비워 둔다. 별칭을 두지 않는다 |
| `internal/expose` | 유지 |

- [0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md)의 상태를 `amended`로 바꾼다. 바뀐 것은 이름 한 축이고 반출 범위, 익명화 방식, 출력 형태는 전부 그대로다.
- `pack`을 별칭으로 남기지 않는다. 출하 전이라 쓰던 사람이 없고, 별칭을 두면 문서가 두 이름을 설명해야 한다.

## 결과

- 커맨드가 하는 일과 이름이 일치한다. "왜 압축이 안 되냐"는 질문이 생기지 않는다.
- Confluence나 Notion에서 오는 사용자가 이름만 보고 동작을 짐작할 수 있다.
- `eject`와 `export`가 나란히 선다. 문서에서 둘의 대상 차이를 분명히 적어야 한다.
- [0006](0006-dual-mode-eject-seal.md)의 표와 본문이 어긋난 채로 남는다. 소급 수정 금지이므로 이 ADR이 그 불일치의 해석을 확정한다. **본문이 맞고 표가 틀렸다.**

## 관련

- [0046 pack은 파일을 그대로 내보내고 익명화는 사용자 사전으로 한다](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md) 동작을 정한 ADR. 이름만 이 문서가 바꾼다
- [0006 easy/hard 듀얼 모드와 모드 전환 커맨드](0006-dual-mode-eject-seal.md) 표와 본문이 어긋난 자리
- [0013 eject를 규칙 소유권 이양으로 재정의하고 seal을 폐기한다](0013-eject-redefined-seal-removed.md) `eject`가 무엇을 내보내는지
