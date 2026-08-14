# 로드맵

> 마일스톤별 범위는 [design.md](design.md)에, 여정 매핑은 [journeys.md](journeys.md)에 있다. 이 문서는 **지금 당장 무엇을 할 것인가**를 다룬다.

## 현재 상태

설계 문서가 확정되었고 Go 코드는 아직 없다. ADR 15건, 동작 구조 도식 10종, 커맨드 체계, 여정 24개가 문서로 존재한다.

## 즉시 다음

1. **Go 스캐폴드** — 루트에 `go.mod`(모듈명 `github.com/neocode24/engram`), `cmd/engram/`, `internal/`. CLI 프레임워크를 붙이고 `version`과 `--json` 전역 플래그, `--now` 전역 플래그를 먼저 뚫는다. `--now`는 0.3의 `resurface`를 위한 것이지만 처음부터 있어야 parity 측정이 성립한다.
2. **`init`과 스키마 로더** — 프리셋 3종, 설정 파일 파싱, 디렉토리 생성과 온보딩 문구 인쇄.
3. **`lint`와 게이트** — 프론트매터 검증, 위키링크 파싱, `min_wikilinks` 판정. 거절 메시지 품질이 제품 품질이므로 여기서 에러 출력 형식을 확정한다.
4. **harness 골격** — `harness/fixtures/`에 골든 위키를 만들고 lint 출력 비교를 붙인다. 코드가 늘어난 뒤에 시작하면 이미 늦다.

## upstream 선행 작업

ADR 0005의 전제이며 아직 미착수다.

- upstream 위키에 `meta/CHANGELOG.md`를 신설한다. 스키마와 규칙 변경만 기록하며 항목마다 영향 범위를 태그한다.
- upstream `AGENTS.md`에 "`meta/` 아래 규칙 파일을 고치면 CHANGELOG 항목을 추가한다"를 계약으로 명시한다.

현재 upstream의 운영 로그는 스펙 변경 로그가 아니므로 delta 판정의 근거로 쓸 수 없다.

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
