# 로드맵

> 마일스톤별 범위는 [design.md](design.md)에, 여정 매핑은 [journeys.md](journeys.md)에 있다. 이 문서는 **지금 당장 무엇을 할 것인가**를 다룬다.

## 현재 상태

Go 구현이 시작되었다. ADR 19건, 동작 구조 도식 10종, 커맨드 체계, 여정 24개가 문서로 있고, 0.1 마일스톤의 커맨드 셋 중 `version`, `init`, `lint` 셋이 동작한다. upstream 쪽 선행 작업인 계약 변경 로그는 1차 구축을 마쳤고 기준 정정만 남았다.

## 끝난 것

로드맵 착수 항목 넷이 닫혔다.

| 항목 | 산출물 | 관련 ADR |
|---|---|---|
| Go 스캐폴드 | `cmd/engram/`, `internal/cli/`, `version`, 전역 `--json`과 `--now` | [0016](decisions/0016-cli-framework-and-global-flags.md) |
| `init`과 스키마 로더 | `internal/config/`(프리셋 3종, 축 on/off, 임계값), `internal/cli/init.go` | [0017](decisions/0017-yaml-for-config-and-frontmatter.md), [0018](decisions/0018-taxonomy-field-names.md) |
| `lint`와 게이트 | `internal/doc/`(프론트매터와 위키링크 파싱), `internal/lint/`(규칙 14종) | [0019](decisions/0019-index-documents-outside-the-gate.md) |
| harness 골격 | `harness/fixtures/golden-wiki/`(사례 12종), `harness/lint_golden_test.go` | [0005](decisions/0005-upstream-contract-and-harness.md) |

`init` 직후 `lint`가 위반 0건에 종료 코드 0을 낸다. 여정 0의 첫 화면이 거절이 아니어야 한다는 요구가 회귀 검사로 고정되었다.

## 즉시 다음

0.1 마일스톤에 남은 커맨드는 다섯이다. 순서는 아래로 한다.

1. **`capture`와 `source`** — 인테이크 두 경로. `capture`는 검증 없이 `inbox/`에 넣고, `source`는 `sources/`에 넣으며 원본 필드를 확정한다. `sources/`에 `updated`를 쓰지 않는다는 계약을 여기서 강제한다.
2. **`promote`와 `new`** — 승급 경로. 게이트 판정은 `internal/lint`가 이미 갖고 있으므로 그것을 호출한다. 게이트 로직을 두 벌 만들면 곧 갈라진다.
3. **`status`** — 위키 현황과 inbox 적체 압력. `lint`의 순회를 재사용한다.
4. **`doctor`** — 환경 점검. 사내 배포에서 지원 요청을 줄이는 장치이므로 각 항목에 복구 명령을 함께 낸다.
5. **upstream parity 대조** — 지금 harness가 덮는 것은 자기 출력의 골든 스냅샷뿐이다. ADR 0005가 정한 upstream 스크립트와의 실제 대조는 아직 없다. `docs/parity.md`는 그때 생긴다.

## upstream 선행 작업

ADR 0005의 전제다. 1차 구축은 끝났고 기준 정정이 남았다.

**완료.** upstream `meta/CHANGELOG.md` 신설과 git 이력 backfill 27건, `AGENTS.md`에 동시 커밋 계약 명시(upstream `05f7279`).

**남은 과제는 기록 대상 기준의 정정이다.** 현재 27건은 "계약 파일이 바뀐 커밋"을 기준으로 모았는데, 이 기준이 사용자 데이터와 스키마를 섞는다. 실제로 최다 항목이 `terminology-normalization` 6건이고 그 대부분은 조직 고유명사를 사전에 추가한 것이다. 이는 위키 소유자의 지식이지 downstream 구현이 따라야 할 규칙이 아니다.

ADR 0005는 이미 둘을 갈라 두었다. 치환 사전은 "코드로 옮기지 않고 데이터로 소비"한다고 적혀 있다. 즉 계약은 파일이 아니라 그 파일의 **스키마**다.

| 구분 | 예 | CHANGELOG 대상 |
|---|---|---|
| 스키마 | `Auto-correct?` 열의 값 집합, 필수 필드, 판정 임계값, 폐쇄 집합 | 대상 |
| 사용자 데이터 | 치환 사전 행 추가, `topic` 값 추가, 조직 고유명사 | 비대상 |

경계 사례 둘은 대상으로 남긴다. 값 집합이 늘어난 경우(사전에 `conditional`이 처음 등장)와 치환 규칙 자체가 바뀐 경우(canonical 표기를 영어에서 한국어로 전환)는 행 변경의 외형을 하지만 구현 동작을 바꾼다.

할 일 셋이다.

1. 기존 27건을 이 기준으로 재분류하고 사용자 데이터 항목을 제외한다.
2. upstream `AGENTS.md`의 계약 문구를 "파일이 바뀌면"에서 "스키마가 바뀌면"으로 정정하고 위 표를 넣는다.
3. `.githooks/pre-commit`에 게이트를 붙인다. 파일 경로만 보는 검사는 사전 행 추가마다 CHANGELOG를 요구해 로그를 노이즈로 채우므로, 데이터 파일과 규칙 파일을 구분하거나 커밋 메시지 탈출구를 함께 넣는다.

정정 전의 CHANGELOG는 delta 판정의 근거로 쓸 수 있으나 항목 수를 규칙 변경 빈도로 읽으면 안 된다.

## 진행 체제

Orca 오케스트레이션으로 coordinator와 worker를 붙인다.

| 역할 | 실행 | 비고 |
|---|---|---|
| coordinator | Hermes 또는 claude CLI(Opus) | 둘 중 하나를 쓴다 |
| worker | `glm` | `claude`가 아니다 |

worker는 반드시 `glm`으로 띄운다. `glm`은 claude CLI에 z.ai 인증과 모델 매핑을 주입하는 래퍼이므로, `claude`로 띄우면 구독 OAuth로 붙어 worker가 coordinator와 같은 모델이 된다. 모델 별칭도 래퍼가 재매핑하므로 worker에게 `--model opus`를 주면 GLM 5.3이 뜬다. dispatch 지시서에 실행 명령과 모델 별칭을 함께 적는다.

주의할 점 하나는 계획 파일 경로다. worker는 claude 기반이라 superpowers 규약을 따라 저장소의 `.superpowers/plans/`에 쓴다. coordinator가 claude CLI면 같은 경로를 보지만 Hermes면 `.hermes/plans/`를 본다. coordinator를 Hermes로 둘 때만 경로가 갈리므로, 그 경우 dispatch 지시서에 산출물 경로를 명시한다.

## 문서 부채

- **문체 정리.** AGENTS.md가 em dash와 화살표를 금지하는데 초기 문서 다수가 위반 상태다. 린터를 붙여 한 번에 정리한다.
- **`curriculum.md` 재작성.** 여정 24개와 마일스톤이 확정되었으므로 강의 단위를 다시 매핑해야 한다. 현재 내용은 여정 5개 시절 기준이다.
- **`docs/parity.md`** — harness가 돌기 시작하면 자동 생성된다. 지금은 없다.

## 미결정

- 교육용 데모 위키 내용. `engram init --preset education`으로 재생성 가능해야 하며 결과가 다르면 회귀로 본다.
- `serve` 웹 UI의 쓰기 범위.
- MCP 노출 시 도구 단위 분해. 쓰기가 `inbox`까지라는 경계는 확정이다.
- 강의 일정, 대상, 평가 방식.

## 검증 수단

Go 테스트 스위트가 생겼다. `go test ./...`가 정식 검증이다.

| 대상 | 검사 |
|---|---|
| `internal/config` | 프리셋 축 on/off, 병합 우선순위, 미정의 키 수집, 허용값 거절 |
| `internal/doc` | 프론트매터 분리와 파싱, BOM과 CRLF, 위키링크 추출, 코드 펜스 제외 |
| `internal/lint` | 규칙별 판정, 종료 코드, 출력 결정론 |
| `internal/cli` | `init` 생성물과 멱등성, `--now` 고정 시 바이트 동일 |
| `harness` | 골든 위키에 대한 lint 출력 스냅샷 비교. `go test ./harness -update`로 갱신 |

문서 쪽은 여전히 ad-hoc 스크립트 둘이다.

| 스크립트 | 검사 대상 |
|---|---|
| `scripts/check-adr.py` | ADR frontmatter 규약, 색인과 상태 일치, 상대링크 무결성 |
| `scripts/check-mermaid.py` | 문서 내 mermaid 블록 렌더 가능 여부 |

문서 검사 둘은 pre-commit에 붙인다. 아직 붙이지 않았다.

## 관련

- [architecture.md](architecture.md) 동작 구조
- [design.md](design.md) 커맨드 체계와 마일스톤
- [journeys.md](journeys.md) 사용자 여정
- [ADR 색인](decisions/README.md)
