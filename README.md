# engram

사내 교육강좌 및 공개 산출물을 위해, 운영 중인 `llm-wiki`의 철학(승급 파이프라인, 다축 스키마, 재발견 루프)을 **단일 Go 바이너리 오픈소스**로 출판하는 프로젝트.

> **2026-08-15 현재 상태: 설계/계획 단계.** 이 저장소에는 논의된 방향과 의사결정을 문서로 남긴 커밋만 있다. Go 코드는 아직 없다.
> 프로젝트 명칭이 `llm-wiki-edu`에서 `engram`으로 변경되었다. 확정 근거는 [ADR 0008](docs/decisions/0008-project-name-engram.md)에 있다.

## 핵심 전제

- 본인 실사용은 기존 `llm-wiki`의 script/shell 구조를 그대로 유지한다 (유연성 최우선).
- 이 프로젝트의 바이너리는 사내 교육 가치성과 홍보 산출물을 목적으로 한다.
- 코드는 처음부터 Go로 새로 구현한다 (외부 도구 fork 없음).

## 이름

engram = 기억이 뇌에 남기는 물리적 흔적. second brain을 표방하는 도구의 실체가 무엇인지 이름이 그대로 말한다.

## 구조

단일 저장소이며 저장소 루트가 Go 모듈 루트다. 근거는 [ADR 0011](docs/decisions/0011-repo-layout-and-module-name.md).

| 경로 | 역할 | 상태 |
|---|---|---|
| `docs/` | 설계, 의사결정(ADR), 로드맵 | 초기 작성됨 |
| `docs/course/` | 강의 자료. 공개 자산 | placeholder |
| `cmd/engram/` | 실행 파일 진입점 | 미생성 |
| `internal/` | 구현 | 미생성 |
| `harness/` | upstream 계약 스냅샷, 골든 픽스처, parity | 미생성 |
| `examples/` | 데모 위키. `init`의 생성물이며 손으로 고치지 않는다 | placeholder |

## 커맨드 개요 (설계 중)

| 명령 | 동작 |
|---|---|
| `engram eject` | 내장 규칙을 실제 파일로 풀어 사용자에게 넘긴다. 단방향 |
| `engram rules show` | eject 없이 내장 규칙을 읽기 전용으로 출력한다 |
| `engram skills install` | LLM 에이전트에 스킬 문서를 설치한다 |
| `engram pack` | 배포와 공유용 번들을 만든다 (후순위) |

`attach`는 별도 커맨드가 아니라 기본 동작이다. 위키 루트에 `.engram/`이 있으면 자동으로 붙는다.

## 문서 둘러보기

- [docs/architecture.md](docs/architecture.md) — **동작 구조와 역할 경계. 가장 먼저 읽는다**
- [AGENTS.md](AGENTS.md) — 에이전트 작업 계약
- [docs/design.md](docs/design.md) — 커맨드 체계, 설정, 마일스톤
- [docs/journeys.md](docs/journeys.md) — 사용자 여정 24개
- [docs/decisions/README.md](docs/decisions/README.md) — 의사결정 색인과 개정 관계
- [docs/curriculum.md](docs/curriculum.md) — 교육 커리큘럼 뼈대
- [docs/course/](docs/course/) — 강의 자료
- [docs/roadmap.md](docs/roadmap.md) — 로드맵과 미결정 사항

## 원천 참조

운영 진실원: `~/Git/llm-wiki`. 이 프로젝트는 그 체계를 특정 시점에 "정식 얼림"으로 출판한 산물이다.
