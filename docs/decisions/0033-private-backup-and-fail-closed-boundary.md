---
number: 0033
title: private 자료는 upstream에 백업하고 경계 검사는 닫히는 쪽으로 실패한다
date: 2026-08-16
status: accepted
---

# private의 백업과 경계 검사의 실패 방향

## 배경

ADR [0024](0024-public-boundary-and-private-directory.md)가 `private/`를 gitignore 대상으로 두고 "실체는 upstream에 두고 여기에는 포인터와 발췌만 둔다"고 했다. 0.3까지 오면서 그 전제가 깨졌다. 지금 `private/`에 upstream 어디에도 없는 파일이 셋 있다.

| 파일 | 성격 |
|---|---|
| `boundary-patterns.txt` | 경계 검사가 쓰는 금지 패턴 목록 |
| `vendor-replacements.txt` | upstream 계약 vendoring의 익명화 사전([0029](0029-upstream-vendoring-and-parity-execution.md)) |
| `history-replacements.txt` | 이력 익명화에 쓴 치환 사전 |

셋 다 잃으면 손으로 다시 만들어야 하고, 그 과정에서 빠뜨린 항목이 곧 공개 저장소로 나가는 식별자가 된다. **공개 경계의 방어선이 백업되지 않는 로컬 파일에 걸려 있다.**

두 번째 문제가 같은 자리에 있다. `scripts/check-boundary.py`는 패턴 목록이 없으면 건너뛰고 종료 코드 0을 낸다. `.githooks/pre-commit`이 그 종료 코드를 그대로 받으므로 **패턴 파일이 없는 기계에서는 커밋이 검사 없이 통과한다.** 첫 문제의 결과가 곧 두 번째 문제의 발동 조건이다. 디스크를 잃거나 새 기계에 저장소만 받으면 경계 검사가 조용히 꺼진 채로 커밋이 쌓인다.

## 판단 근거

### 백업 위치

**upstream이 이미 백업된 비공개 저장소다.** `llm-wiki`는 개인 GitHub의 private 저장소이고 원격에 push된다. 별도 백업 수단을 새로 만들 이유가 없다.

**세 파일 다 upstream 어휘를 다룬다.** 익명화 사전은 upstream의 실제 식별자와 대체어의 대응이고, 금지 패턴은 그 식별자들의 정규표현식이다. upstream은 실제 식별자를 보존하는 저장소이므로 이 내용이 그곳에 있는 것이 자연스럽다. 공개 저장소로 나가지 않는다는 조건은 그대로 지켜진다.

**upstream `private/`에는 둘 수 없다.** upstream의 gitignore가 `private/`를 이미 막고 있다. 백업이 목적인데 커밋되지 않으면 의미가 없다. `meta/engram/`에 둔다. upstream lint 스크립트의 스캔 범위는 `context/`, `sources/`, `inbox/`이고 `meta/`는 대상이 아니므로 부작용이 없다.

**심볼릭 링크로 잇지 않는다.** `private/`의 파일을 upstream으로 향하는 링크로 만들면 upstream이 없는 기계에서 링크가 끊긴다. 끊긴 링크는 `os.path.exists`가 거짓이므로 경계 검사가 건너뛰기로 넘어간다. 백업을 붙이려다 두 번째 문제의 발동 조건을 하나 더 만드는 셈이다. 실체를 양쪽에 두고 명령 하나로 맞춘다.

**경로를 저장소에 박지 않는다.** upstream 위치는 기계마다 다르다. `harness/parity`가 이미 쓰는 `ENGRAM_UPSTREAM` 환경변수를 그대로 쓴다. 같은 저장소를 가리키는 변수를 둘 두지 않는다.

**`private/deltas/`는 제외한다.** upstream `meta/CHANGELOG.md`에서 파생한 산물이라 upstream만 있으면 다시 만들 수 있다([0030](0030-upstream-delta-is-not-a-public-artifact.md)). 파생물을 원본 옆에 두지 않는다.

### 실패 방향

**CI는 건너뛰어야 하고 커밋 훅은 막아야 한다.** 두 요구가 다르다. 0024가 건너뛰기를 택한 이유는 CI 로그가 저장소와 함께 공개되므로 거기에 패턴이 찍히면 가드 자체가 유출 경로가 되기 때문이다. 그 판단은 CI에서 유효하다. 반면 커밋 훅에서 목록이 없다는 것은 "검사할 필요가 없다"가 아니라 **"검사할 수 없다"**이며, 그 상태로 커밋을 통과시키면 가드가 없는 것과 같다.

**따라서 실패 방향을 부르는 쪽이 정한다.** 스크립트에 플래그를 두고 훅만 그것을 준다. CI는 주지 않는다. 기본값은 지금 동작을 유지하므로 손으로 돌리는 경우가 깨지지 않는다.

**빈 목록은 막지 않는다.** 파일이 있는데 항목이 없는 것은 사람이 그렇게 둔 상태다. 없는 것과 다르다.

## 결정

### 백업

- `private/`의 실체를 upstream `meta/engram/`에 사본으로 둔다. upstream은 비공개 저장소이며 원격에 push된다.
- 맞추는 수단은 `scripts/private-backup.sh` 하나다. upstream 위치는 `ENGRAM_UPSTREAM`으로 받고 없으면 실패한다.
- `private/deltas/`는 제외한다. upstream에서 다시 만들 수 있다.
- 심볼릭 링크를 쓰지 않는다. 양쪽에 실체를 두고 명령으로 맞춘다.
- `private/`를 고친 뒤 이 명령을 돌린다. 자동화하지 않는다. upstream 저장소에 커밋을 남기는 행위이므로 사람이 시작한다.

### 실패 방향

- `scripts/check-boundary.py`에 `--require`를 둔다. 패턴 목록이 없으면 종료 코드 1이다.
- `.githooks/pre-commit`이 `--require`를 준다. 패턴 파일이 없는 기계에서 커밋이 막힌다.
- 플래그가 없으면 지금과 같이 건너뛴다. CI와 손으로 돌리는 경우가 그대로다.
- 막을 때 복구 방법을 안내한다. 패턴 자체는 출력하지 않는다.

## 결과

- 세 사전이 원격에 백업된다. 디스크를 잃어도 upstream을 받으면 복구된다.
- 새 기계에서 engram만 받고 커밋하면 훅이 막는다. 경계 검사가 조용히 꺼진 채로 커밋이 쌓이지 않는다.
- 백업이 수동이므로 `private/`를 고치고 명령을 안 돌리면 사본이 낡는다. 자동화 대신 이 ADR과 AGENTS.md에 절차로 남긴다.
- upstream에 engram 전용 디렉토리가 하나 생긴다. upstream의 계약 파일 목록은 파일명을 명시하므로 vendoring 대상이 늘지 않는다.

## 관련

- [0024 공개 경계와 private 디렉토리](0024-public-boundary-and-private-directory.md) `private/`와 경계 검사의 원안. 이 ADR이 백업 경로와 실패 방향을 더한다
- [0029 upstream vendoring과 parity 실행 조건](0029-upstream-vendoring-and-parity-execution.md) `vendor-replacements.txt`와 `ENGRAM_UPSTREAM`
- [0030 upstream delta는 공개하지 않는다](0030-upstream-delta-is-not-a-public-artifact.md) `private/deltas/`가 파생물인 근거
