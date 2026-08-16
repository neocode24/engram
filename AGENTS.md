# AGENTS.md — engram 저장소 작업 계약

이 저장소에서 작업하는 에이전트(Hermes, Claude Code, Copilot)는 아래를 따른다.

## 첫 번째로 읽을 문서

**`docs/architecture.md`** — engram이 무엇을 하고 무엇을 하지 않는지, 어떤 순서로 도는지가 도식과 함께 들어 있다. 그다음 **`docs/spec-map.md`**(upstream 규칙 명세가 engram의 무엇이 되었는지. 이 프로젝트의 뼈대다), `docs/design.md`(커맨드 체계와 설정), `docs/decisions/README.md`(결정 이력과 개정 관계) 순으로 읽는다.

## 이 프로젝트가 무엇인가

`engram`은 운영 중인 위키(`llm-wiki`)의 지식관리 체계(승급 파이프라인, 다축 스키마, 재발견 루프)를 단일 Go 바이너리로 출판하는 프로젝트다. 목적은 교육 자료와 공개 산출물이다.

- upstream 진실원은 별도 저장소인 `llm-wiki`이며 이 저장소는 그 체계를 특정 시점에 얼려 출판한 산물이다. upstream은 비공개다.
- upstream을 건드릴 때는 반드시 그쪽 `AGENTS.md`의 계약을 먼저 읽는다.
- 0.3 마일스톤까지의 커맨드가 동작한다. 스물이며 `engram --help`가 목록의 진실원이다. 넣기(`capture`, `source`), 올리기(`promote`, `new`, `demote`, `archive`), 조회(`search`, `recall`, `backlinks`, `lint`, `status`, `doctor`), 재발견(`resurface`, `bridge`, `digest`), 관리(`init`, `mv`, `update`, `reindex`, `version`)로 나뉜다.

## 저장소 구조

| 경로 | 역할 |
|---|---|
| `docs/architecture.md` | 동작 구조 전체. mermaid 도식 10종 |
| `docs/decisions/` | ADR. 번호순, 소급 수정 금지. `README.md`가 색인 |
| `docs/spec-map.md` | upstream 규칙 명세와 구현의 대응. 무엇을 코드가 강제하고 무엇을 사람에게 남겼는지 |
| `docs/design.md` | 커맨드 체계, 설정, 마일스톤 |
| `docs/journeys.md` | 사용자 여정과 마일스톤 매핑 |
| `docs/curriculum.md` | 교육 커리큘럼 |
| `docs/course/` | 강의 자료. 공개 자산이며 사내 사례를 담지 않는다 (placeholder) |
| `cmd/engram/`, `internal/` | Go 구현. 저장소 루트가 모듈 루트 |
| `harness/` | 골든 픽스처, 여정 통합 테스트, upstream 동등성 검증. lint 축까지 마쳤다 |
| `examples/` | 데모 위키. `init`의 생성물이므로 손으로 고치지 않는다 (placeholder) |
| `private/` | 공개 경계 밖 자료. **gitignore 대상이라 커밋되지 않는다** ([0024](docs/decisions/0024-public-boundary-and-private-directory.md)) |

## 작업 규칙

### ADR

- 결정은 `docs/decisions/NNNN-kebab-title.md`에 남긴다. frontmatter는 `number, title, date, status` 네 키만 쓴다.
- `status`는 `accepted`, `amended`, `superseded`, `proposed` 넷 중 하나다. 어휘 정의는 ADR 0015에 있다.
- **기존 ADR의 본문을 소급 수정하지 않는다.** 결정이 바뀌면 새 ADR을 쓰고, `docs/decisions/README.md`의 개정 그래프에 행을 추가한 뒤 대상 ADR의 `status`만 바꾼다. 명칭 변경 같은 표기 갱신은 "당시 명칭 X" 병기로 처리한다.
- 새 ADR을 추가하거나 상태를 바꿀 때 `docs/decisions/README.md` 색인을 함께 갱신한다. 색인이 개정 관계의 단일 진실원이다.

### git

- 기본 브랜치는 `main`. remote는 ssh-over-443(`ssh://git@ssh.github.com:443/neocode24/engram.git`), 개인 계정 `neocode24`.
- 작업 전 `git pull --rebase --autostash`, 작업 후 커밋하고 `git push origin`. origin push는 기본 동기화 행위이므로 확인 없이 진행한다.
- PR 생성, 외부 배포, 다른 remote push는 명시적 승인이 필요하다.

### 공개 경계

- **첫 작업 전에 훅을 켠다.** `git config core.hooksPath .githooks`. 커밋 시점에 경계 검사가 돈다.
- 조직 맥락, 동기, IP 경계 판단은 `private/`에 둔다. gitignore 대상이므로 커밋되지 않는다.
- `private/`는 백업되지 않는다. 실체는 upstream에 두고 여기에는 포인터와 발췌만 둔다.
- 공개 문서에는 기능과 기술적 판단만 쓴다. 왜 만들었는지의 조직적 맥락은 쓰지 않는다.

### 검증

`go test ./...`가 정식 검증이다. 문서 검사는 `scripts/check-adr.py`와 `scripts/check-mermaid.py`이며 아직 ad-hoc 스크립트다. 그 둘의 결과를 보고할 때는 **정식 테스트가 아니라 ad-hoc 검증**임을 명시한다.

### 문체

문체는 독자에 따라 셋으로 나뉜다. 자세한 근거는 [0027](docs/decisions/0027-prose-register-by-audience.md)에 있다.

| 대상 | 문체 |
|---|---|
| `docs/`의 ADR과 설계 문서 | 보고체. 근거를 먼저 쓰고 결론을 뒤에 둔다 |
| 코드 주석 | 보고체 |
| **커맨드 출력과 에러 메시지** | **경어체.** 사용자가 매 실행마다 읽는다. 다만 문장을 늘리지 않는다 |
| `README.md` | 안내체. 구체 규칙은 재작성 시점에 정한다 |

아래는 세 층 전부에 적용된다.
- 이 저장소는 오픈소스 공개 예정이므로 **사내 식별자(실명, 조직명, 사내 제품명)를 쓰지 않는다.** upstream llm-wiki는 반대로 보존하므로, 규칙 명세를 가져올 때 익명화 경계를 넘지 않았는지 확인한다.
- 한국어 산문에 em dash, 화살표, 가운뎃점, 말줄임표, smart quotes를 쓰지 않는다. 코드 블록, 표, frontmatter, 기술 표기는 예외다.
