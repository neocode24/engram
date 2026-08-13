# engram

사내 교육강좌 및 공개 산출물을 위해, 운영 중인 `llm-wiki`의 철학(승급 파이프라인, 다축 스키마, 재발견 루프)을 **단일 Go 바이너리 오픈소스**로 출판하는 프로젝트.

> **2026-08-13 현재 상태: 설계/계획 단계.** 이 저장소에는 논의된 방향과 의사결정을 문서로 남긴 커밋만 있다. Go 코드는 아직 없다.
> 프로젝트 명칭이 `llm-wiki-edu`에서 `engram`으로 변경되었다. 확정 근거는 ADR 0008(작성 예정)에 남긴다.

## 핵심 전제

- 본인 실사용은 기존 `llm-wiki`의 script/shell 구조를 그대로 유지한다 (유연성 최우선).
- 이 프로젝트의 바이너리는 사내 교육 가치성과 홍보 산출물을 목적으로 한다.
- 코드는 처음부터 Go로 새로 구현한다 (외부 도구 fork 없음).

## 이름

engram = 기억이 뇌에 남기는 물리적 흔적. second brain을 표방하는 도구의 실체가 무엇인지 이름이 그대로 말한다.

## 구조

| 경로 | 역할 | 상태 |
|---|---|---|
| `docs/` | 설계, 의사결정(ADR), 커리큘럼, 다음 스텝 | 초기 작성됨 |
| `binary/` | Go 바이너리 (별도 repo 후보) | placeholder |
| `wiki/` | 교육용 데모 위키 예시 (회사 정보 없는 깨끗한 예제) | placeholder |
| `curriculum/` | 강의 자료 | placeholder |

## 커맨드 개요 (설계 중)

| 명령 | 동작 |
|---|---|
| `engram eject` | 내장 규칙을 실제 파일로 풀어놓는다 (easy mode에서 hard mode로) |
| `engram seal` | 풀어놓은 파일을 거두고 내장 규칙으로 복귀한다 |
| `engram attach` | 기존 hard mode 위키에 바이너리를 붙인다 |
| `engram pack` | 배포/공유용 번들을 만든다 (후순위) |

## 문서 둘러보기

- [docs/handoff.md](docs/handoff.md) — **세션 인계 노트. 작업 재개 시 가장 먼저 읽는다**
- [AGENTS.md](AGENTS.md) — 에이전트 작업 계약
- [docs/design.md](docs/design.md) — 아키텍처, 커맨드 체계, 스키마 매핑, 마일스톤
- [docs/decisions/](docs/decisions/) — 의사결정 기록
- [docs/curriculum.md](docs/curriculum.md) — 교육 커리큘럼 뼈대
- [docs/next-steps.md](docs/next-steps.md) — 다음 스텝과 미결정 사항

## 원천 참조

운영 진실원: `~/Git/llm-wiki`. 이 프로젝트는 그 체계를 특정 시점에 "정식 얼림"으로 출판한 산물이다.
