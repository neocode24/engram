---
number: 0029
title: upstream 계약을 치환 사전으로 익명화해 vendoring하고 parity는 로컬에서만 돈다
date: 2026-08-16
status: amended
---

# upstream vendoring의 익명화와 parity 실행 조건

## 배경

ADR 0005가 3층 harness를 정했다. 계약 파일 vendoring, delta 감지, conformance 비교다. 그러나 착수 시점에 세 가지가 실제와 어긋났다.

**계약 파일이 여섯이 아니라 여덟이다.** 0005는 `meta/` 아래 여섯을 들었다. upstream의 `AGENTS.md`가 그 뒤에 `wiki-artifact-schema.md`와 `security-rules.md`를 계약 파일 목록에 넣었다.

**스캐너가 실패한 다음이 없다.** 0005는 "익명화 경계를 넘은 문자열이 들어오면 실패시킨다"고 했다. 실패 후 무엇을 해야 vendoring이 성사되는지는 정하지 않았다. 실제로 여덟 파일을 스캔하니 네 파일에서 여섯 줄이 걸렸다.

**parity가 CI에서 돌 수 없다.** upstream 저장소가 비공개이고 로컬에만 있다. 공개 저장소의 CI는 그 저장소를 볼 수 없다.

공개 경계의 뜻도 좁혀졌다. 막아야 하는 것은 **공개 저장소에 커밋되는 것**이지 파일을 읽는 행위나 외부 도구에 넘기는 행위가 아니다.

## 판단 근거

### 계약 파일 목록의 진실원

**upstream이 자기 계약 파일을 선언한다.** upstream `AGENTS.md`가 "이 목록의 파일을 고치면 같은 커밋에서 `meta/CHANGELOG.md`에 항목을 추가한다"고 계약했다. 그 목록이 곧 downstream이 따라가야 할 대상이다. 이 저장소가 별도 목록을 들고 있으면 upstream이 파일을 추가할 때마다 두 목록이 어긋난다.

`meta/` 아래라고 전부 계약은 아니다. `resurface-state.json`은 상태이고 `weekly-news-sources.md`는 운영 자료이며 `templates/`는 서식이다. 셋 다 upstream의 계약 목록에 없다.

### 익명화

**치환하되 조용히 삼키지 않는다.** 걸린 여섯 줄을 사람이 매번 손으로 고치면 sync가 반복 노동이 된다. 반대로 자동으로 마스킹하면 패턴 목록에 없는 식별자가 그대로 통과한다. 둘 사이가 답이다. 치환 사전을 두고, 치환한 뒤 스캐너를 돌리고, **그래도 걸리면 실패시킨다.** 사람은 사전에 항목을 추가하는 일만 한다.

사전은 `private/`에 둔다. 사전 자체가 식별자 목록이기 때문이다. 이력 익명화에 쓴 `private/history-replacements.txt`와 같은 이유이고 같은 형식이다.

**대체어는 의미를 보존한다.** 계약 파일은 규칙 문서이고 규칙은 예시로 설명된다. 사내 제품명을 `제품-A`로 바꾸면 그 예시가 무엇을 말하는지 알 수 없게 된다. 일반명사로 바꾼다. 규칙의 형태가 남아야 vendoring의 목적이 성립한다.

**치환 사실을 vendored 파일에 남긴다.** 독자가 원문과 다르다는 것을 알아야 한다. 모르면 upstream 원문을 인용하는 줄 알고 쓴다.

### terminology-normalization.md

이 파일만 성격이 다르다. 15.7KB의 치환 사전이며 **사전 전체가 조직 어휘의 목록이다.** 걸린 줄이 둘뿐이라도, 어떤 용어를 정규화 대상으로 삼았는지가 그 조직이 무엇을 다루는지를 드러낸다. 여섯 줄을 치환한다고 해결되는 종류가 아니다.

지금 vendoring하지 않는다. 손해가 없다. 0005가 정한 비교 축 넷에 용어 정규화가 없고, 그 기능을 구현하는 마일스톤도 아직 오지 않았다. 필요해지는 시점에 사전의 공개 범위를 따로 판단한다.

### 픽스처

**upstream 문서를 픽스처로 쓰지 않는다.** 0005가 이미 "조직 정보 없는 깨끗한 예시"라고 못 박았다. 실제 문서를 가져오면 익명화 대상이 규칙 여덟 파일에서 위키 전체로 늘어난다. 규칙을 검증하는 데 실제 지식이 필요하지 않다.

픽스처는 규칙의 경계를 때리도록 새로 쓴다. 필수 필드 누락, 허용값 밖의 값, 깨진 링크, 고아 문서, 게이트 미달처럼 판정이 갈리는 자리를 담는다.

### 실행 조건

**parity는 upstream이 있을 때만 돈다.** 환경변수가 upstream 경로를 가리키면 돌고, 없으면 skip이다. CI는 항상 skip이다.

CI가 못 돈다고 가치가 없지는 않다. 이 비교는 매 커밋이 아니라 **upstream 계약이 바뀔 때** 필요하다. `meta/CHANGELOG.md`에 `binary-affecting` 항목이 붙는 순간이 그 시점이고, 그것은 사람이 판단해 시작하는 작업이다.

`docs/parity.md`는 로컬 실행 결과를 사람이 커밋한다. 자동 갱신이 아니다.

### 비교 축의 현재

0005가 넷을 들었으나 지금 성립하는 것은 둘이다.

| 축 | upstream | engram | 지금 |
|---|---|---|---|
| lint 위반 목록 | `scripts/lint-frontmatter.sh` | `engram lint` | 가능 |
| resurface 선정 순위 | `scripts/wiki_resurface.py` | `engram resurface` | 가능 |
| frontmatter 정규화 | `scripts/sync_updated_field.py` | 해당 커맨드 없음 | 불가 |
| eject 산출물 | 없음 | `eject` 미구현 | 불가 |

**축 하나로 시작한다.** 하니스 골격과 비교 방식이 옳은지는 축 하나로 확인된다. 넷을 한 번에 만들면 골격이 틀렸을 때 넷을 다시 만든다.

## 결정

### vendoring

- 대상은 upstream `AGENTS.md`가 선언한 계약 파일이다. 목록을 이 저장소에 복제하지 않고 매 sync마다 그 문서에서 읽는다.
- `terminology-normalization.md`는 제외한다. 사전 전체가 조직 어휘 목록이기 때문이다.
- 복사 위치는 `harness/upstream/`, 원본 커밋 해시는 `harness/upstream.lock`이다.
- 치환 사전은 `private/vendor-replacements.txt`다. 형식은 `private/history-replacements.txt`와 같다.
- 순서는 복사, 치환, `scripts/check-boundary.py` 검사다. 검사에 걸리면 **sync가 실패한다.** 사람이 사전에 항목을 추가하고 다시 돌린다.
- vendored 파일 머리에 출처 커밋과 치환 여부를 주석으로 남긴다.

### delta

- lock의 해시부터 upstream HEAD까지의 `meta/CHANGELOG.md` 변화를 `harness/deltas/`에 남긴다.
- 자동 반영하지 않는다. 사람이 읽고 판단한다.
- `impact: binary-affecting` 항목을 눈에 띄게 표시한다. 그것이 이 저장소가 따라가야 할 항목이다.

### parity

- 픽스처는 `harness/fixtures/` 아래에 새로 쓴다. upstream 문서를 가져오지 않는다.
- upstream 경로는 환경변수로 받는다. 없으면 비교를 skip한다. CI는 항상 skip이다.
- 첫 축은 `lint` 위반 목록이다. 나머지는 골격이 검증된 뒤에 붙인다.
- 결과는 `docs/parity.md`에 사람이 커밋한다.

## 결과

- upstream이 계약 파일을 추가해도 이 저장소의 목록을 고치지 않는다. upstream `AGENTS.md`가 단일 진실원이다.
- 익명화가 반복 노동이 아니라 사전 관리가 된다. 새 식별자는 한 번만 등록한다.
- 스캐너가 걸리면 sync가 멈추므로 익명화되지 않은 문자열이 커밋에 도달하지 않는다.
- parity가 CI 밖에 있으므로 그 실행 시점을 사람이 정해야 한다. `meta/CHANGELOG.md`의 `binary-affecting` 항목이 그 계기다.
- `terminology-normalization.md`를 뺐으므로 용어 정규화 기능을 만들 때 이 결정을 다시 본다.

## 열린 항목

- `terminology-normalization.md`의 공개 범위. 용어 정규화 기능을 만들 때 판단한다.
- upstream `scripts/lint-frontmatter.sh`가 픽스처에서 그대로 도는지. 위키 루트를 가정한 경로가 있으면 실행 방식을 조정해야 한다.
- 두 구현의 위반 목록 형식이 다르므로 비교 전에 정규화가 필요하다. 그 정규화가 차이를 감추지 않게 하는 방법.

## 관련

- [0005 upstream 계약과 harness](0005-upstream-contract-and-harness.md) 3층 harness의 원안
- [0024 공개 경계와 private 디렉토리](0024-public-boundary-and-private-directory.md) 경계 검사와 `private/`
- [0028 재발견 커맨드의 상태와 경계](0028-rediscovery-state-and-boundaries.md) `resurface`의 상태 파일
