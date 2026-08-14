# AGENTS.md — engram 저장소 작업 계약

이 저장소에서 작업하는 에이전트(Hermes, Claude Code, Copilot)는 아래를 따른다.

## 첫 번째로 읽을 문서

**`docs/handoff.md`** — 직전 세션에서 확정된 설계 결정과 다음 할 일이 전부 들어 있다. 작업 시작 전 반드시 읽는다.

## 이 프로젝트가 무엇인가

`engram`은 운영 중인 `~/Git/llm-wiki`의 지식관리 체계(승급 파이프라인, 다축 스키마, 재발견 루프)를 단일 Go 바이너리로 출판하는 프로젝트다. 목적은 사내 교육강좌와 공개 산출물이다.

- upstream 진실원은 `~/Git/llm-wiki`이며 이 저장소는 그 체계를 특정 시점에 얼려 출판한 산물이다.
- upstream을 건드릴 때는 반드시 `~/Git/llm-wiki/AGENTS.md`의 계약을 먼저 읽는다.
- 현재 상태는 설계/계획 단계다. Go 코드는 아직 없다.

## 저장소 구조

| 경로 | 역할 |
|---|---|
| `docs/handoff.md` | 세션 인계 노트. 가장 먼저 읽는다 |
| `docs/decisions/` | ADR. 번호순, 소급 수정 금지 |
| `docs/design.md` | 아키텍처와 커맨드 체계 |
| `docs/journeys.md` | 사용자 여정과 마일스톤 매핑 |
| `docs/curriculum.md` | 교육 커리큘럼 |
| `docs/course/` | 강의 자료. 공개 자산이며 사내 사례를 담지 않는다 (placeholder) |
| `cmd/engram/`, `internal/` | Go 구현. 저장소 루트가 모듈 루트 (미생성) |
| `harness/` | upstream 계약 스냅샷, 골든 픽스처, parity (미생성) |
| `examples/` | 데모 위키. `init`의 생성물이므로 손으로 고치지 않는다 (placeholder) |

## 작업 규칙

### ADR

- 결정은 `docs/decisions/NNNN-kebab-title.md`에 남긴다. frontmatter는 `number, title, date, status`.
- **기존 ADR의 본문을 소급 수정하지 않는다.** 결정이 바뀌면 새 ADR을 쓰고 이전 것의 `status`만 바꾼다. 명칭 변경 같은 표기 갱신은 "당시 명칭 X" 병기로 처리한다.

### git

- 기본 브랜치는 `main`. remote는 ssh-over-443(`ssh://git@ssh.github.com:443/neocode24/engram.git`), 개인 계정 `neocode24`.
- 작업 전 `git pull --rebase --autostash`, 작업 후 커밋하고 `git push origin`. origin push는 기본 동기화 행위이므로 확인 없이 진행한다.
- PR 생성, 외부 배포, 다른 remote push는 명시적 승인이 필요하다.

### 검증

이 저장소에는 아직 테스트 스위트가 없다. 검증은 임시 스크립트로 수행하고, 결과를 보고할 때 **정식 테스트가 아니라 ad-hoc 검증**임을 명시한다.

### 문서 문체

- 보고체. 근거를 먼저 쓰고 결론을 뒤에 둔다.
- 이 저장소는 오픈소스 공개 예정이므로 **사내 식별자(실명, 조직명, 사내 제품명)를 쓰지 않는다.** upstream llm-wiki는 반대로 보존하므로, 계약 파일을 가져올 때 익명화 경계를 넘지 않았는지 확인한다.
- 한국어 산문에 em dash, 화살표, 가운뎃점, 말줄임표, smart quotes를 쓰지 않는다. 코드 블록, 표, frontmatter, 기술 표기는 예외다.
