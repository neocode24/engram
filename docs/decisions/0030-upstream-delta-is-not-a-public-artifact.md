---
number: 0030
title: upstream 변경 로그에서 뽑은 delta는 공개하지 않는다
date: 2026-08-16
status: accepted
---

# delta 문서의 공개 여부

## 배경

ADR 0029가 delta를 `harness/deltas/`에 남기기로 정했다. 공개 저장소의 경로다. 그 결정 뒤에 upstream `meta/CHANGELOG.md`의 실제 내용을 읽었고, 전제가 틀렸다는 것이 드러났다.

delta는 lock 커밋부터 upstream HEAD까지의 CHANGELOG 변화를 옮겨 담는다. 즉 **CHANGELOG 원문을 인용한다.** 그 원문에는 사내 제품명, 조직 부서명, 사내 프로세스 명칭, 직급 호칭이 들어 있다. 그것이 규칙 변경을 설명하는 방식이기 때문이다. 어떤 용어를 정규화 대상으로 삼았는지 적으려면 그 용어를 적어야 한다.

`scripts/check-boundary.py`의 패턴 목록은 열두 개다. CHANGELOG에 나오는 조직 어휘는 그보다 훨씬 많고 계속 는다. 패턴 목록이 걸러 주기를 기대할 수 없다.

이 문제를 vendoring 워커가 완료 보고에서 스스로 지적했다. "패턴 목록이 경계의 정의이므로 설계대로다"라고 판단했는데, 그 판단이 절반만 맞다. 패턴 목록은 **알려진** 식별자에 대한 경계의 정의이지 경계 그 자체가 아니다.

## 판단 근거

**delta는 공개 독자에게 쓸모가 없다.** delta의 독자는 upstream 규칙 변경을 이 구현에 반영할지 판단하는 유지보수자 한 명이다. 문서는 upstream 커밋 해시를 인용하는데 공개 독자는 그 저장소를 볼 수 없으므로 해시를 따라갈 수 없다. 공개해서 얻는 것이 없다.

**치환으로 해결하려 하면 사전이 CHANGELOG를 따라 자란다.** upstream이 새 용어를 다룰 때마다 항목을 추가해야 하고, 추가를 빠뜨린 회차가 곧 유출이다. 익명화 사전은 여덟 개 계약 파일이라는 닫힌 집합에 대해서는 관리 가능하지만, 앞으로 계속 쓰일 로그에 대해서는 관리 부담이 무한하다.

**얻을 것이 없는 위험은 감수하지 않는다.** 공개해서 얻는 것이 없고 실수 하나가 유출인 문서라면 애초에 공개 경로에 두지 않는다. `private/`는 gitignore 대상이라 커밋 자체가 불가능하고, 이것이 패턴 목록보다 강한 보장이다.

**반대 논거를 검토했다.** delta를 공개하면 이 구현이 upstream을 추적한다는 증거가 된다. 그러나 그 증거는 `docs/parity.md`가 더 정확하게 낸다. parity는 수치이고 delta는 산문이다. 수치는 익명화 부담이 없다.

**계약 파일 vendoring은 그대로 둔다.** 여덟 개라는 닫힌 집합이고, 규칙 문서 자체는 downstream 구현자가 읽어야 할 공개 자산이다. delta와 성격이 다르다.

## 결정

**upstream CHANGELOG에서 뽑은 delta는 `private/deltas/`에 남긴다.** 공개 저장소에 커밋하지 않는다.

- `scripts/upstream-sync.py`의 출력 경로를 바꾼다. ADR 0029의 delta 절에서 경로만 개정된다. 생성 방식, `impact` 분리, 치환 적용은 그대로다.
- `private/`는 백업되지 않는다(ADR 0024). delta는 upstream 저장소에서 언제든 다시 만들 수 있으므로 손실이 문제가 아니다.
- 경계 검사는 여전히 delta에 치환을 적용한 뒤 돈다. `private/`에 있으니 통과가 강제는 아니지만, 사전이 최신인지를 알려 주는 신호로 남긴다.

`docs/parity.md`는 공개 산출물로 그대로 둔다. 수치와 규칙 이름만 담기 때문이다.

## 결과

- upstream이 새 조직 어휘를 다뤄도 공개 저장소가 영향을 받지 않는다. 익명화 사전을 CHANGELOG 속도에 맞춰 늘릴 필요가 없다.
- 유지보수자는 delta를 로컬에서만 읽는다. 원래 그 문서의 독자가 한 명이므로 잃는 것이 없다.
- `terminology-normalization.md`를 vendoring에서 뺀 ADR 0029의 결정이 옳았음이 확인됐다. CHANGELOG 항목의 다수가 그 파일에 관한 것이고, 사내 어휘가 그대로 나열된다.

## 관련

- [0024 공개 경계와 private 디렉토리](0024-public-boundary-and-private-directory.md) `private/`와 경계 검사
- [0029 upstream vendoring과 parity 실행](0029-upstream-vendoring-and-parity-execution.md) delta 생성 방식
