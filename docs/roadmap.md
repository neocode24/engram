# 로드맵

> 마일스톤별 범위는 [design.md](design.md)에, 여정 매핑은 [journeys.md](journeys.md)에 있다. 이 문서는 **지금 당장 무엇을 할 것인가**를 다룬다.

## 현재 상태

설계 문서가 확정되었고 Go 코드는 아직 없다. ADR 15건, 동작 구조 도식 10종, 커맨드 체계, 여정 24개가 문서로 존재한다. upstream 쪽 선행 작업인 계약 변경 로그는 1차 구축을 마쳤고 기준 정정만 남았다.

## 즉시 다음

1. **Go 스캐폴드** — 루트에 `go.mod`(모듈명 `github.com/neocode24/engram`), `cmd/engram/`, `internal/`. CLI 프레임워크를 붙이고 `version`과 `--json` 전역 플래그, `--now` 전역 플래그를 먼저 뚫는다. `--now`는 0.3의 `resurface`를 위한 것이지만 처음부터 있어야 parity 측정이 성립한다.
2. **`init`과 스키마 로더** — 프리셋 3종, 설정 파일 파싱, 디렉토리 생성과 온보딩 문구 인쇄.
3. **`lint`와 게이트** — 프론트매터 검증, 위키링크 파싱, `min_wikilinks` 판정. 거절 메시지 품질이 제품 품질이므로 여기서 에러 출력 형식을 확정한다.
4. **harness 골격** — `harness/fixtures/`에 골든 위키를 만들고 lint 출력 비교를 붙인다. 코드가 늘어난 뒤에 시작하면 이미 늦다.

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

coordinator는 Hermes, worker는 claude CLI를 Orca 오케스트레이션으로 붙인다. 주의할 점 하나는 계획 파일 경로다. worker 쪽은 superpowers 규약을 따라 저장소의 `.superpowers/plans/`에 쓰고 coordinator는 `.hermes/plans/`를 본다. 두 경로가 갈리므로 dispatch 지시서에 산출물 경로를 명시한다.

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

이 저장소에는 아직 테스트 스위트가 없다. 현재 있는 것은 ad-hoc 스크립트 둘이다.

| 스크립트 | 검사 대상 |
|---|---|
| `scripts/check-adr.py` | ADR frontmatter 규약, 색인과 상태 일치, 상대링크 무결성 |
| `scripts/check-mermaid.py` | 문서 내 mermaid 블록 렌더 가능 여부 |

Go 코드가 들어오면 정식 테스트로 대체하고, 문서 검사 둘은 pre-commit에 붙인다.

## 관련

- [architecture.md](architecture.md) 동작 구조
- [design.md](design.md) 커맨드 체계와 마일스톤
- [journeys.md](journeys.md) 사용자 여정
- [ADR 색인](decisions/README.md)
