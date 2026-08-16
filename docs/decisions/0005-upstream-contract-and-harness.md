---
number: 0005
title: upstream 계약을 harness로 동기화한다
date: 2026-08-13
status: amended
---

# 0005 upstream 계약과 harness

## 결정

upstream 위키(`llm-wiki`)와 이 저장소의 Go 바이너리 사이를 잇는 것은 코드가 아니라 **계약 파일**이다. 계약을 vendoring하고, 변경을 delta로 감지하고, 동작 동등성을 conformance 테스트로 증명하는 3층 harness를 둔다.

## 근거

upstream은 shell과 python으로 되어 있고 downstream은 Go다. 언어가 다르므로 코드 재사용 경로가 없다. 반면 두 구현이 지켜야 하는 규칙(스키마, 승급 게이트, 링크 불변식)은 동일해야 한다. 따라서 동기화 대상은 구현이 아니라 규칙의 문서적 정의다.

계약의 실제 원천은 upstream `meta/` 아래 여섯 파일이다.

| 파일 | 계약의 성격 |
|---|---|
| `meta/frontmatter-schema.md` | 단계별 필수 필드 |
| `meta/promotion-rules.md` | 승급 게이트 판정 |
| `meta/ingest-rules.md` | 입력 수용 규칙 |
| `meta/taxonomy.md` | 태그 체계 |
| `meta/wiki-graph-policy.md` | 링크 불변식 |
| `meta/terminology-normalization.md` | 자동 치환 사전 |

`agents/workflows/*.md`는 여정 정의로, `scripts/*.py|sh`는 알고리즘 재구현 대상으로 참고 자산에 포함한다. 설치 실험 부산물인 `canopy.toml`은 계약이 아니며 upstream에서 이미 삭제되었다.

`meta/terminology-normalization.md`는 코드로 옮기지 않고 **데이터로 소비**한다. 치환 사전은 사람이 계속 늘리는 자산이므로 바이너리에 하드코딩하면 즉시 노후한다.

## 구조

### (a) upstream 스냅샷 vendoring

`harness/upstream/`에 계약 파일만 복사하고 `harness/upstream.lock`에 원본 커밋 해시를 기록한다. upstream 저장소는 private이고 조직 고유 식별자를 포함하므로 서브모듈 전체 참조는 금지한다. vendoring 스텝에는 식별자 grep 스캐너를 붙여, 익명화 경계를 넘은 문자열이 들어오면 실패시킨다.

### (b) 변경 감지와 spec-delta

`make upstream-sync`가 lock의 SHA부터 upstream HEAD까지 diff를 뽑아 `harness/deltas/YYYY-MM-DD.md`를 생성한다. 사람이 그 delta를 읽고 바이너리 반영 여부를 판단한다. 자동 반영은 하지 않는다.

### (c) conformance 테스트

`harness/fixtures/`의 골든 위키(조직 정보 없는 깨끗한 예시)에 대해 upstream 스크립트와 Go 바이너리의 출력을 비교한다. 비교 축은 네 가지다.

1. `lint` 위반 목록
2. `resurface` 선정 순위
3. frontmatter 정규화 결과
4. `eject` 산출물과 upstream 스냅샷의 diff

결과는 `docs/parity.md`로 자동 갱신한다. 이 문서는 검증 산출물인 동시에 "원본 체계와 동일하게 동작한다"는 홍보 자산이 된다.

## 전제조건

upstream 쪽에 선행 작업이 필요하다.

- `meta/CHANGELOG.md`를 신설한다. 스키마와 규칙 변경만 기록하며, 항목마다 `impact: binary-affecting | wiki-only`를 붙인다.
- upstream `AGENTS.md`에 "`meta/` 아래 규칙 파일을 고치면 CHANGELOG 항목을 추가한다"를 계약으로 명시한다.

현재 upstream의 `log.md`는 운영 로그이지 스펙 변경 로그가 아니므로 delta 판정의 근거로 쓸 수 없다.

## 함정

resurface는 `meta/resurface-state.json`으로 상태를 들고 있어 실행 시각에 따라 결과가 달라진다. 골든 비교가 성립하려면 바이너리에 `--now` 플래그를 **처음부터** 넣어야 한다. 나중에 넣으면 그 전까지의 parity 수치가 전부 무의미해진다.

## 관련

- [0002 운영과 산출물의 분리](0002-operating-vs-product-separation.md)
- [0003 접근 B, 전체 스펙과 승급](0003-approach-B-fullspec-with-promotion.md)
