# AGENTS.md — engram 저장소 작업 계약

이 저장소에서 작업하는 에이전트(Hermes, Claude Code, Copilot)는 아래를 따른다.

## 첫 번째로 읽을 문서

**`docs/architecture.md`** — engram이 무엇을 하고 무엇을 하지 않는지, 어떤 순서로 도는지가 도식과 함께 들어 있다. 그다음 **`docs/spec-map.md`**(upstream 규칙 명세가 engram의 무엇이 되었는지. 이 프로젝트의 뼈대다), `docs/design.md`(커맨드 체계와 설정), `docs/decisions/README.md`(결정 이력과 개정 관계) 순으로 읽는다.

## 이 프로젝트가 무엇인가

`engram`은 운영 중인 위키(`llm-wiki`)의 지식관리 체계(승급 파이프라인, 다축 스키마, 재발견 루프)를 단일 Go 바이너리로 출판하는 프로젝트다. 목적은 교육 자료와 공개 산출물이다.

- upstream 진실원은 별도 저장소인 `llm-wiki`이며 이 저장소는 그 체계를 특정 시점에 얼려 출판한 산물이다. upstream은 비공개다.
- upstream을 건드릴 때는 반드시 그쪽 `AGENTS.md`의 계약을 먼저 읽는다.
- 1.0 마일스톤까지의 커맨드가 동작한다. 스물여덟이며 `engram --help`가 목록의 진실원이다. 넣기(`capture`, `source`), 올리기(`promote`, `new`, `demote`, `archive`), 조회(`search`, `recall`, `backlinks`, `lint`, `status`, `doctor`), 재발견(`resurface`, `bridge`, `digest`), 관리(`init`, `mv`, `update`, `reindex`, `migrate`, `sync`, `rules show`, `eject`, `skills install`, `mcp`, `serve`, `export`, `version`)로 나뉜다.

## 저장소 구조

### 문서

| 경로 | 역할 |
|---|---|
| `docs/architecture.md` | 동작 구조 전체. mermaid 도식 11종 |
| `docs/decisions/` | ADR. 번호순, 소급 수정 금지. `README.md`가 색인 |
| `docs/spec-map.md` | upstream 규칙 명세와 구현의 대응. 무엇을 코드가 강제하고 무엇을 사람에게 남겼는지 |
| `docs/design.md` | 커맨드 체계, 설정, 마일스톤 |
| `docs/voice.md` | `engram-voice` 동작 구조. 모델 넷과 단계 다섯, upstream 과의 차이 |
| `docs/parity.md` | upstream 스크립트와 판정을 대조한 결과. 사람이 옮겨 적는다 |
| `docs/journeys.md` | 사용자 여정과 마일스톤 매핑 |
| `docs/curriculum.md` | 교육 커리큘럼 |
| `docs/course/` | 강의 자료. `hands-on.md`(강사용)와 `agent-start.md`(자습용)가 원본이고 `index.html`, `agent.html`이 그 덱이다. 공개 자산이며 사내 사례를 담지 않는다 |

### 코드

| 경로 | 역할 |
|---|---|
| `cmd/engram/`, `internal/` | Go 구현. 저장소 루트가 모듈 루트 |
| `voice/` | `engram-voice` 의 **중첩 모듈**. 루트의 `./...` 에 안 잡히므로 따로 빌드하고 시험한다([0080](docs/decisions/0080-voice-is-a-nested-module-in-this-repository.md)) |
| `harness/` | 검증. `parity/`(upstream 대조), `eject/`(내보낸 린터 대조), `examples/`(데모 위키 재생성), `journey/`(여정 통합), `golden/`, `fixtures/`, `realdata/`, `upstream/`(vendored 명세 사본) |
| `examples/personal/` | 데모 위키. 커맨드 시퀀스의 생성물이므로 손으로 고치지 않는다. `go test ./harness/examples -update`로 재생성 |
| `examples/materials/` | 실습 재료. 수강생이 위키에 집어넣을 원재료이며 위키가 아니다. 합성 자료다 |
| `private/` | 공개 경계 밖 자료. **gitignore 대상이라 커밋되지 않는다** ([0024](docs/decisions/0024-public-boundary-and-private-directory.md)) |

### 스크립트

**`scripts/` 아래 파이썬은 제작 시점 도구다. `engram` 바이너리는 이 넷 중 무엇도 부르지 않는다.** 제품이 Go 인 것과 별개이며, 언어가 섞였다고 정리 대상으로 보지 말 것.

| 파일 | 무엇을 하나 | 언제 도나 |
|---|---|---|
| `scripts/check-boundary.py` | 공개 경계 검사. 금지 패턴이 커밋될 문서에 있는지 본다 | **커밋 훅** (`--require`), 수동 |
| `scripts/check-adr.py` | ADR 파일의 번호, 상태값, README 색인 일치, 상대링크 | **커밋 훅**, **CI**, 수동 |
| `scripts/check-mermaid.py` | 도식이 실제로 렌더되는지. `npx` 로 mermaid-cli 를 부른다 | **CI**, 수동 |
| `scripts/upstream-sync.py` | upstream 명세 vendoring 과 변화 감지 | **수동만.** 사람이 시작한다([0029](docs/decisions/0029-upstream-vendoring-and-parity-execution.md)) |
| `scripts/private-backup.sh` | `private/` 를 upstream `meta/engram/` 으로 백업 | 수동 |
| `scripts/windows-verify.ps1` | Windows 수동 검증 절차 | 수동 |

**`eject` 가 내보내는 `scripts/lint-frontmatter.py` 는 이것들과 다르다.** 그것은 사용자 손에 들어가는 **산출물**이며 언어 선택이 제품 결정이다([0039](docs/decisions/0039-eject-emits-rule-specs-and-a-python-linter.md)). engram 없이도 규칙이 돌아야 해서 표준 라이브러리 파이썬으로 낸다.

`.githooks/pre-commit` 이 `check-boundary.py --require` 와 `check-adr.py` 를 부른다. 첫 작업 전에 `git config core.hooksPath .githooks` 로 켠다.

## 작업 규칙

### 설계 질문

설계나 규칙을 묻는 질문에 추론으로 답하지 않는다. 순서대로 읽고 인용한다.

1. `docs/spec-map.md` 규칙 명세가 engram의 무엇이 되었는지
2. `docs/decisions/` 그 결정의 근거와 기각한 대안
3. upstream `llm-wiki`의 `meta/`와 단계 디렉토리 `README.md`

`spec-map` 4.1부터 4.7까지가 upstream이 계약 파일로 선언한 일곱이고 4.9가 그 선언 밖의 규범 문서다. **계약 밖 문서에도 규범문이 있으며 upstream의 `meta/CHANGELOG.md` 규율 대상이 아니라 바뀌어도 통보되지 않는다.**

셋 어디에도 없으면 없다고 말하고 새 ADR로 결정한다.

### 에이전트 경계

`internal/skills/SKILL.md`는 권고다. 스킬 경로의 에이전트는 셸과 파일 편집기를 쥐고 있어 문서로 막을 수 없다. **강제는 커맨드와 `lint`에서 한다.** 에이전트가 규약을 어기면 문구를 더하기 전에 커맨드 쪽에서 막을 수 있는지 먼저 본다.

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

`docs/course/hands-on.md`에 인용하는 커맨드 출력은 **전부 실측값**이다. 해당 단계의 위키 상태를 실제로 만들어 돌린 결과만 적는다. 손으로 지어낸 출력을 넣지 않는다.

큰 절을 교체하면 앞뒤 절과 어긋나는지 확인한다. 앞 절에서 옮긴 문서를 뒤 절에서 다시 쓰는 모순이 실제로 났다.

핸즈온은 4단계부터 **에이전트가 하고 사람은 승인만 한다**([0057](docs/decisions/0057-approval-attaches-to-content-not-to-the-command.md)). 뒤 단계에서 커맨드 나열로 돌아가지 않는다. 손으로 치는 것은 커맨드의 성격을 보여 줄 때뿐이며 그 이유를 그 자리에 밝힌다.

### 문체

문체는 독자에 따라 넷으로 나뉜다. 자세한 근거는 [0027](docs/decisions/0027-prose-register-by-audience.md)에 있다.

| 대상 | 문체 |
|---|---|
| `docs/`의 ADR과 설계 문서 | 보고체. 근거를 먼저 쓰고 결론을 뒤에 둔다 |
| 코드 주석 | 보고체 |
| **커맨드 출력과 에러 메시지** | **경어체.** 사용자가 매 실행마다 읽는다. 다만 문장을 늘리지 않는다 |
| `docs/course/` 교재 | 안내체. **구현 사정, 설계 자랑, "우리가 안 하는 것" 목록을 넣지 않는다. 수강생이 그것을 알아야 하는 이유가 없으면 뺀다** |
| `README.md` | 안내체. 구체 규칙은 재작성 시점에 정한다 |

아래는 네 층 전부에 적용된다.
- 이 저장소는 오픈소스 공개 예정이므로 **사내 식별자(실명, 조직명, 사내 제품명)를 쓰지 않는다.** upstream llm-wiki는 반대로 보존하므로, 규칙 명세를 가져올 때 익명화 경계를 넘지 않았는지 확인한다.
- 한국어 산문에 em dash, 화살표, 가운뎃점, 말줄임표, smart quotes를 쓰지 않는다. 코드 블록, 표, frontmatter, 기술 표기는 예외다.
