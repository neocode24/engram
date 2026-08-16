---
number: 0031
title: 문서가 놓인 디렉토리와 artifact_stage가 일치해야 한다
date: 2026-08-16
status: amended
---

# 위치와 단계의 일치

## 배경

upstream과의 첫 parity 비교(`docs/parity.md`)에서 engram에 없는 규칙 하나가 드러났다. upstream `lint-frontmatter.sh`는 문서가 놓인 디렉토리와 프론트매터 `artifact_stage` 값이 어긋나면 실패로 기록한다. engram에는 그 검사가 없다.

engram이 `inbox/wrong-stage.md`를 잡기는 한다. 그러나 `schema.allowed-value:artifact_stage`로 잡는다. 값이 허용 집합 밖이라서 걸린 것이지 위치가 어긋나서 걸린 것이 아니다. **값이 허용 집합 안에 있으면서 디렉토리와 어긋나는 문서는 engram이 통과시킨다.** `inbox/foo.md`에 `artifact_stage: context`를 적으면 아무도 막지 않는다.

## 판단 근거

**`artifact_stage`가 모든 단계 판정의 입력이다.** 단계별 필수 필드, 승급 게이트 대상, 고아 판정 제외, `resurface`의 후보 선정이 전부 이 값을 읽는다. 디렉토리와 어긋나면 그 판정들이 전부 엉뚱한 기준으로 돈다. `inbox/`에 있으면서 `context`라고 적힌 문서는 게이트를 지나지 않고 검수된 지식 취급을 받는다. **게이트가 유일한 관문이라는 전제가 이 한 줄로 무너진다.**

**등급은 error다.** 이 저장소의 `error`는 승급을 막고 종료 코드 1을 만든다. 위치와 선언이 어긋난 문서는 뒤따르는 모든 판정을 오염시키므로 경고로 통과시킬 수 없다. upstream도 실패로 기록한다.

**최상위 디렉토리로 판정한다.** `context/서브/문서.md`는 `context` 단계다. 하위 디렉토리는 사용자가 나누는 자유이고 단계를 바꾸지 않는다. upstream은 `context/mocs/`를 별도 취급하지만 그것은 upstream에만 있는 `index` 단계 때문이다.

**색인 문서는 제외한다.** ADR [0019](0019-index-documents-outside-the-gate.md)가 `root_files`를 게이트와 고아 검사 밖에 두기로 했다. 같은 이유가 여기에도 적용된다. `index.md`는 위키 루트에 있어 어떤 `page_dirs`에도 속하지 않으므로 비교할 디렉토리가 없다.

**`type-agreement`는 따라가지 않는다.** upstream은 디렉토리와 `type` 값의 일치도 본다. 그러나 그 규칙이 적용되는 자리가 `context/mocs/`와 `index.md` 둘뿐이고, 둘 다 upstream에만 있는 `moc` 타입과 `index` 단계를 요구한다. engram에는 그 타입도 그 단계도 없다. 없는 개념을 위해 규칙을 만들지 않는다.

**매핑을 두 벌 두지 않는다.** 단계와 디렉토리의 대응은 이미 `internal/wiki`에 있다. `source` 단계가 `sources` 디렉토리에 대응한다는 관례가 그 안에 들어 있고, 이것이 유일하게 이름이 어긋나는 자리다. lint가 같은 표를 다시 만들면 한쪽만 고치는 사고가 난다.

**값이 없으면 건너뛴다.** `artifact_stage`가 아예 없는 문서는 `frontmatter.missing-field`가 이미 잡는다. 비교할 값이 없는데 위치 불일치를 보고하면 같은 결함을 두 번 말하는 것이다. 값이 있고 어긋나면 보고한다. 그 값이 허용 집합 밖이더라도 보고한다. 허용값 위반과 위치 불일치는 다른 결함이고 upstream도 둘 다 보고한다.

## 결정

**`page_dirs` 아래 문서의 최상위 디렉토리가 프론트매터 `artifact_stage`와 일치해야 한다.**

| 항목 | 값 |
|---|---|
| 규칙 ID | `location.stage-agreement` |
| 등급 | `error` |
| 대상 | `page_dirs` 아래 문서 |
| 제외 | `root_files`(ADR 0019), `artifact_stage`가 없는 문서 |

- 디렉토리와 단계의 대응은 `internal/wiki`가 단일 진실원이다. lint는 그 표를 부른다.
- 판정은 최상위 디렉토리로 한다. 하위 디렉토리는 단계를 바꾸지 않는다.
- 위반 메시지는 어느 디렉토리에 있고 무엇이라 적혀 있는지를 함께 낸다. 고치는 법은 둘이므로(파일을 옮기거나 값을 고치거나) 둘 다 알린다.
- `type-agreement`는 구현하지 않는다.

## 결과

- `inbox/`에 있으면서 `context`라고 선언한 문서가 막힌다. 게이트를 우회하는 가장 단순한 경로가 닫힌다.
- `promote`, `demote`, `archive`가 만든 문서는 전부 이 규칙을 통과한다. 세 커맨드가 파일 이동과 프론트매터 갱신을 함께 하기 때문이다. 도구가 자기 산출물로 자기 검사를 통과한다는 원칙이 유지된다.
- 손으로 파일을 옮긴 사용자가 이 규칙에 걸린다. 그것이 이 규칙의 목적이다. `engram promote`나 `engram demote`를 쓰라고 안내한다.
- lint 규칙이 열여섯이 된다. 골든 스냅샷이 갱신된다.
- parity의 `upstream만 잡음`이 세 쌍 줄어든다. `index.md`의 두 쌍은 ADR 0019에 따른 의도된 차이로 남는다.

## 관련

- [0019 색인 문서는 게이트 밖에 둔다](0019-index-documents-outside-the-gate.md) `root_files` 제외의 근거
- [0022 promote는 inbox를 옮기고 sources에서 파생한다](0022-promote-moves-inbox-derives-sources.md) 단계 이동의 정상 경로
- `docs/parity.md` 이 규칙이 드러난 비교 결과
