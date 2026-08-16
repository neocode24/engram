# 로드맵

> 마일스톤별 범위는 [design.md](design.md)에, 여정 매핑은 [journeys.md](journeys.md)에 있다. 이 문서는 **지금 당장 무엇을 할 것인가**를 다룬다.

## 현재 상태

**0.4까지 끝났고 1.0은 `pack` 하나가 남았다.** 커맨드 스물일곱, ADR 45건, 동작 구조 도식 10종, 여정 24개가 문서로 있고, 코드는 패키지 열다섯이다. upstream 동등성 검증을 lint 축에서 측정했고 결과는 [parity.md](parity.md)에 있다. Windows 실환경 2차 검증까지 마쳤으나 콘솔 직결 항목은 검증 스크립트 결함으로 아직 미확인이다.

## 끝난 것

| 커맨드 | 하는 일 | 관련 ADR |
|---|---|---|
| `search` | 한국어 bigram과 BM25. 인덱스 상태를 알린다 | [0010](decisions/0010-storage-index-and-korean-search.md), [0025](decisions/0025-index-storage-and-staleness.md) |
| `reindex` | 인덱스를 만드는 유일한 커맨드 | [0025](decisions/0025-index-storage-and-staleness.md) |
| `backlinks` | 들어오는 링크. 종류를 구분해 보여준다 | |
| `mv` | 이름 변경. 모든 링크를 함께 고친다 | [0020](decisions/0020-slug-and-filename-rules.md) |
| `demote` | 승급을 되돌린다. 깨질 링크를 알린다 | [0022](decisions/0022-promote-moves-inbox-derives-sources.md) |
| `update` | 본문과 프론트매터 갱신 | |
| `version` | 버전과 빌드 정보 | [0016](decisions/0016-cli-framework-and-global-flags.md) |
| `init` | 위키 생성. 프리셋 3종 | [0017](decisions/0017-yaml-for-config-and-frontmatter.md), [0018](decisions/0018-taxonomy-field-names.md), [0020](decisions/0020-slug-and-filename-rules.md) |
| `capture` | 검증 없이 `inbox/`에 받는다 | |
| `source` | `sources/`에 원본을 확정한다. `updated`를 쓰지 않는다 | [0009](decisions/0009-schema-presets-and-thresholds.md) |
| `promote` | inbox는 이동, sources는 파생 | [0021](decisions/0021-gate-deferral-when-targets-are-scarce.md), [0022](decisions/0022-promote-moves-inbox-derives-sources.md), [0023](decisions/0023-gate-targets-exclude-inbox.md) |
| `new` | 처음부터 검수된 지식으로 `context/`에 쓴다 | 상동 |
| `lint` | 규칙 16종. 스키마, 링크 무결성, 위치와 단계의 일치 | [0019](decisions/0019-index-documents-outside-the-gate.md) |
| `status` | 현황과 적체 압력, 다음 행동 제안 | |
| `doctor` | 환경과 위키 점검 12종. 항목마다 복구 조치 | |
| `resurface` | 오래 안 본 문서를 제시 이력 기준으로 꺼낸다 | [0028](decisions/0028-rediscovery-state-and-boundaries.md) |
| `bridge` | 유사한데 링크가 없는 쌍. 기각은 영구 기록 | [0028](decisions/0028-rediscovery-state-and-boundaries.md) |
| `digest` | 기간 안의 신규, 노후, 고아 집계. 상태를 남기지 않는다 | [0028](decisions/0028-rediscovery-state-and-boundaries.md) |
| `recall` | 헤딩 단위 청크 원문과 출처 | [0028](decisions/0028-rediscovery-state-and-boundaries.md) |
| `archive` | 수명이 끝난 문서를 보관한다. 슬러그를 유지해 링크가 안 깨진다 | [0028](decisions/0028-rediscovery-state-and-boundaries.md) |
| `migrate` | 기존 문서를 지금의 설정과 규칙에 맞춘다. 파일을 옮기지 않고 승급시키지 않는다 | [0038](decisions/0038-migrate-conforms-documents-to-current-rules.md) |
| `sync` | git 이력에서 `updated`와 `sourced_at`을 정정한다 | [0037](decisions/0037-sync-corrects-dates-from-git.md) |
| `rules show` | 이 위키에 적용되는 규칙 전부를 읽기 전용으로 낸다 | [0013](decisions/0013-eject-redefined-seal-removed.md) |
| `eject` | 규칙을 명세 문서와 Python 린터로 풀어 소유권을 넘긴다. 연산은 engram에 남는다 | [0039](decisions/0039-eject-emits-rule-specs-and-a-python-linter.md), [0013](decisions/0013-eject-redefined-seal-removed.md) |
| `skills install` | 에이전트에 정적 스킬 문서를 심는다. 위키별 규칙은 `rules show`가 낸다 | [0041](decisions/0041-skills-install-embeds-one-static-skill.md), [0014](decisions/0014-llm-boundary-agent-drives-binary.md) |
| `mcp` | 위키를 MCP 서버로 노출한다. 도구 열 중 쓰기는 `capture` 하나이고 `promote`는 없다 | [0043](decisions/0043-mcp-exposes-one-write-tool-and-omits-promote.md) |
| `serve` | 읽기 전용 웹 뷰어. `context/`와 색인만 보여주고 민감도 선언을 뒤집지 않는다 | [0044](decisions/0044-serve-is-read-only-and-shows-only-vetted-knowledge.md) |

패키지는 열다섯이다. `config`(설정과 프리셋), `doc`(파싱과 직렬화), `walk`(순회), `graph`(링크 관계), `lint`(규칙과 게이트), `wiki`(문서 쓰기), `index`(색인과 BM25), `chunk`(헤딩 청킹), `state`(영구 상태), `resurface`, `digest`, `bridge`, `status`, `doctor`, `cli`다.

**같은 판정을 두 벌 두지 않는다.** 게이트 판정, 링크 대상 집계, 고아 판정, 노후 판정은 각각 단일 함수이며 커맨드가 그것을 부른다. 같은 판정이 두 곳에 생기면 커맨드로 통과한 문서를 `lint`가 거절하거나, `status`와 `digest`가 서로 다른 고아 수를 말하는 상태가 된다.

**도구가 자기 산출물로 자기 검사를 통과한다.** `init`부터 `archive`까지 순서대로 돌린 위키에서 `lint`의 `error`와 `reject`가 0이다. 구현 중 네 번 이 원칙을 어겼고 네 번 다 고쳤다. 앞의 셋이 ADR 0019, 0021, 0023이고 넷째가 `archive`의 `artifact_stage` 허용값 누락이다.

**도구가 자기 산출물을 자기 커맨드로 읽을 수 있어야 한다.** 같은 계열의 다섯째 결함이 0.3에서 나왔다. `capture`가 날짜를 파일명에만 남기고 `promote`가 그 접두사를 떼면서, 승급한 모든 문서가 `resurface`와 `digest`의 대상에서 빠졌다. 커맨드가 늘 때마다 기존 산출물이 그 커맨드의 입력으로 성립하는지 확인한다.

**게이트가 유일한 관문이라는 전제가 0.4까지 실제로는 성립하지 않았다.** `context/`에 손으로 둔 문서는 `artifact_stage`를 비우거나 낮춰 적으면 게이트를 지나지 않았다. 게이트만 프론트매터 선언을 보고 나머지 판정은 디렉토리를 봤기 때문이다. ADR [0040](decisions/0040-gate-follows-the-directory-not-the-declaration.md)이 게이트를 디렉토리 기준으로 바꿔 닫았다. 픽스처에 그 경로를 때리는 문서가 하나도 없어서 여태 안 잡혔다.

**다섯 결함 전부를 사람이 손으로 여정을 돌려서 찾았다.** 단위 테스트 팔천 줄이 하나도 못 잡았다. 커맨드를 하나씩만 보기 때문이다. `harness/journey`가 그 빈자리를 메운다. 실제 바이너리를 빌드해 열한 단계를 순서대로 돌고 각 변경 뒤에 `lint`를 다시 검사한다. 이 하니스는 **변이 시험으로 검증한다.** 과거 결함을 코드에 되돌려 놓고 테스트가 실제로 실패하는지 본다. 통과해 버리면 단언이 느슨한 것이다. 첫 판이 실제로 그랬다.

## 즉시 다음

1. **저장소 공개 전환과 첫 릴리스.** 둘이 한 묶음이다. private인 동안 Homebrew가 아카이브를 받을 수 없으므로 tap 갱신을 먼저 켜면 Formula가 죽은 링크를 가리킨다(ADR [0042](decisions/0042-release-artifacts-and-workflow.md)).
2. **`pack`.** 1.0에서 남은 하나다. 여정 14(외부 반출)이며 익명화와 민감도 필터가 걸린다. 민감도 제외 규칙은 [0044](decisions/0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)를 진실원으로 삼는다.
2. **비교 축을 늘린다.** 동등성 검증이 지금 보는 것은 lint 위반 목록 하나다. `resurface` 선정 순위가 다음이다. 나머지 둘은 해당 커맨드가 생길 때.
3. **Windows 재검증** — 다음 버전 검증 때 함께 한다. 콘솔 직결 항목은 2차 검증 당시 스크립트가 출력을 캡처해 버려 실제로는 파이프를 검증하고 있었다. 스크립트를 고쳐 두었으나 실행하지 않았다. `autocrlf` 항목은 VM에 git이 없어 여전히 미검증이다.

## 다음 트랙 (1.1)

1.0을 출하한 뒤에 연다. 지금 여는 것은 출하를 미루는 일이다.

### 용어 정규화와 사전 갱신 루프

**방향은 확정이다. 사전을 빈 채로 출하하고 쓰는 사람이 채워 간다.** `topics`와 `forms`를 다루는 방식 그대로다. upstream의 `meta/terminology-normalization.md`는 156줄짜리 조직 어휘 목록이라 내용을 공개할 수 없다([0029](decisions/0029-upstream-vendoring-and-parity-execution.md)). engram은 **메커니즘만** 준다.

왜 이것이 다음인가. 게이트가 지키는 것은 쌓이는 쪽이고 용어 일관성이 지키는 것은 꺼내는 쪽이다. 같은 개념이 여러 표기로 흩어지면 `search`도 `bridge`도 `taxonomy`도 무너진다. `taxonomy.forms`를 폐쇄 집합으로 둔 근거([0009](decisions/0009-schema-presets-and-thresholds.md))가 그대로 확장되는 자리다.

**upstream 명세에 피드백 루프는 없다.** 실물을 열어 확인했다. 정적 사전과 검토 체크리스트뿐이고, 교정 내역이 사전에 반영되는 과정은 사람이 손으로 돌리고 있다. 그중 기계가 할 수 있는 것만 만든다.

| 만들 것 | 성격 |
|---|---|
| 사전을 문서에 적용 | 결정론적 텍스트 치환. `--dry-run` 기본. `migrate`, `sync`와 같은 계열 |
| 사전과 어긋난 표기를 잡는 lint 규칙 | `taxonomy.forms`와 같은 성격 |
| 사전에 없는데 반복 등장하는 표기 제안 | 세어서 낼 뿐 **등재는 사람이 한다** |

셋 다 순수 Go 텍스트 처리다. 모델도 음성도 필요 없다.

전사 결과 수용은 그다음이다. `capture`가 외부 전사 파일을 받아 위 첫째를 태우면 음성 없이도 루프가 닫힌다.

### 시맨틱 층과 model 커맨드

[0007](decisions/0007-platform-and-distribution.md)이 `engram model pull`을 사이드카로 정해 두었으나 구현이 없고 어느 마일스톤에도 없다. **사이드카이므로 바이너리는 순수 Go로 남는다.** CGO나 LLM 경계와 충돌하지 않는다.

범위가 미결이다. 아래 미결정 절에 있다.

## 알려진 빈틈

- `page_dirs`는 디렉토리 이름 목록이라 단계와 디렉토리를 다르게 매핑할 수 없다. `inbox` 단계는 `inbox`라는 이름의 디렉토리를 요구한다. design.md는 이 키를 "디렉토리 매핑"이라 불렀으므로 표기가 어긋나 있다. 이름을 바꾸고 싶다는 요구가 실제로 나오면 맵으로 바꾼다.
- `inbox/`와 `sources/` 문서의 슬러그에는 날짜 접두사가 들어간다. 그 문서를 위키링크로 가리키려면 접두사까지 써야 한다. `context/` 문서는 접두사가 없으므로 실사용에서 걸리는 일은 드물다.
- `status`의 다음 행동 제안이 `reject`를 계기로 삼지 않는다. 현황 줄에는 `reject` 건수가 나오므로 정보 자체는 있다.
- `digest`가 승급을 집계하지 않는다. `promote`가 승급 시각을 프론트매터에 남기지 않기 때문이다. 필드를 새로 만드는 것은 스키마 축이 느는 일이라 요구가 나올 때 결정한다.
- ~~`updated` 필드를 아무도 채우지 않는다.~~ `update`가 채운다(ADR [0032](decisions/0032-update-writes-the-updated-field.md)). 손으로 고친 파일은 여전히 `sync`가 와야 정정된다.
- `resurface`의 제시 이력이 무한히 쌓인다. 문서 수만큼이므로 지금 규모에서는 문제가 아니다.
- **실운영 위키 306문서를 돌리면 `error`가 25건 남는다.** 열다섯은 upstream 위키가 자기 명세에 없는 값(`status: draft`, `trigger_mode: manual`, `sensitivity: public`)을 쓰거나 engram에 없는 `index` 단계를 쓴 것이고, 여덟은 프론트매터가 빠진 실제 문서, 둘은 `artifact_stage`가 없는 문서다. 전부 engram이 옳게 잡은 것이고 뒤의 둘은 `migrate`가 정리한다. 근거는 `docs/spec-map.md` 6절에 있다.
- **위키 경로를 받는 방식이 커맨드마다 다르다.** `lint`, `status`, `doctor`, `init`, `reindex`는 위치 인자로 받고 나머지는 `--wiki`로 받는다. 자기 위치 인자가 따로 있는 커맨드는 `--wiki`가 맞지만 `resurface`, `bridge`, `digest`는 위치 인자가 없는데도 `--wiki`만 받는다. `engram status ~/wiki`를 익힌 사용자가 `engram resurface ~/wiki`를 치면 `unknown command` 에러를 본다. 0.4에서 커맨드가 더 늘기 전에 정한다.

## 공개 경계 밖

upstream 명세 동기화 과제와 coordinator/worker 실행 체제는 `private/roadmap-internal.md`에 있다. upstream 저장소가 비공개라 공개 독자가 확인할 수 없는 항목이며, 실행 체제는 개인 작업 환경이라 이 저장소의 결정과 무관하다. 근거는 [0024](decisions/0024-public-boundary-and-private-directory.md)에 있다.

## 문서 부채

- **문체 정리.** AGENTS.md가 em dash와 화살표를 금지하는데 초기 문서 다수가 위반 상태다. 린터를 붙여 한 번에 정리한다. 사용자 대면 문자열의 경어체 전환(ADR 0027)은 끝났으나 같은 린터가 이 규칙도 지켜야 재발하지 않는다.
- ~~**규칙 명세와 구현의 대응표.**~~ `docs/spec-map.md`로 만들었다. 명세 7종, lint 규칙 16종, 설정 축 14종의 대응이 전부 들어 있다.
- **`curriculum.md` 재작성.** 여정 24개와 마일스톤이 확정되었으므로 강의 단위를 다시 매핑해야 한다. 현재 내용은 여정 5개 시절 기준이다.
- ~~**`docs/parity.md`**~~ lint 축 측정을 마쳤다. 자동 생성이 아니라 로컬 실행 결과를 사람이 옮겨 적는다(ADR 0029).
- ~~**README 재작성.**~~ 0.3 완료 시점으로 다시 썼다. 커맨드 스물을 다섯 갈래 표로 정리했고 실린 출력은 전부 실제 실행 결과다. 남은 축은 둘이다.
  - **언어.** 영어를 기본으로 두고 한국어를 `docs/ko-KR/README.md`로 분리한다. 오픈소스 공개 시 접근 범위가 달라진다.
  - **시각 자산.** 히어로 영역, 배지, 동작 캡처가 없다. CLI이므로 스크린샷 대신 터미널 녹화가 맞다.
- **여정 2가 어느 마일스톤에도 없다.** journeys.md가 그 피드백 루프를 "숨은 무기"라 부르는데 0.1부터 1.0까지 매핑에서 빠져 있다. 1.1의 용어 정규화가 그 절반을 덮으므로 그때 다시 앉힌다.
- **journeys.md가 없는 동작을 적고 있다.** 여정 0이 "`init`은 대상 폴더, 스키마 프리셋, 언어, git 초기화, 시맨틱 검색 사용 여부를 순서대로 묻는다"고 하는데 실제 `init`은 묻지 않는다. 시맨틱 층이 없어 물을 대상이 없다. 문서와 실제 중 어느 쪽을 고칠지 시맨틱 층을 정할 때 함께 본다.

## 미결정

- 교육용 데모 위키 내용. `engram init --preset education`으로 재생성 가능해야 하며 결과가 다르면 회귀로 본다.
- ~~`serve` 웹 UI의 쓰기 범위.~~ [0044](decisions/0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 읽기 전용으로 정했다.
- ~~MCP 노출 시 도구 단위 분해.~~ [0043](decisions/0043-mcp-exposes-one-write-tool-and-omits-promote.md)이 도구 열로 정했다.
- **`engram model`이 관리할 모델의 범위.** 커맨드 구조는 [0007](decisions/0007-platform-and-distribution.md)에 이미 있고 사이드카라 근간과 충돌하지 않는다. 정할 것은 목록 하나다.

  | 선택 | 뜻 |
  |---|---|
  | 안쪽만 | 임베딩 하나. 검색 랭킹 보조. STT와 TTS는 사용자가 준비하고 engram은 결과만 받는다 |
  | 앞뒤까지 | 셋 다 관리. 설치 한 번으로 파이프라인 전체가 뜬다. engram의 성격에 모델 런처가 한 겹 붙는다 |

  판단 근거는 모델이 붙는 자리다. 임베딩은 engram **안**(검색에 직접), STT는 **앞**(파일이 만들어지기 전), TTS는 **뒤**(꺼낸 다음)다. 어느 쪽이든 [0007](decisions/0007-platform-and-distribution.md)을 뒤집지 않고 목록을 넓히는 ADR 하나면 된다.
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
| `harness/journey` | 실제 바이너리로 `init`부터 `reindex`까지 열한 단계. 각 변경 뒤 `lint`의 `error`와 `reject`가 0인지, 승급 문서에 `created`가 남는지, `resurface` 후보에 승급 문서가 드는지 |
| `harness/parity` | upstream 스크립트와의 대조. lint 위반 목록과 `resurface` 선정 순위 두 축. `ENGRAM_UPSTREAM`이 있을 때만 돈다 |
| `harness/eject` | 내보낸 Python 린터와 `engram lint`의 판정 대조. 여덟 상황에서 출력과 종료 코드를 본다. **어긋나면 실패다.** CI에서 돈다 |
| `harness/realdata` | 실운영 위키 306문서 스모크. 생존, 조회 커맨드의 무변경, 판정의 결정론, 시간 상한 넷만 단언하고 위반 수는 로그로 남긴다. `ENGRAM_UPSTREAM`이 있을 때만 돈다 |
| 릴리스 산출물 | `goreleaser release --snapshot --clean --skip=publish`로 아카이브 여섯과 `checksums.txt`를 만든다. 현재 플랫폼 바이너리의 `engram version`이 `dev`가 아니고 `strings`로 `/Users/` 경로가 안 박혔는지 본다 |

문서 쪽은 여전히 ad-hoc 스크립트 둘이다.

| 스크립트 | 검사 대상 |
|---|---|
| `scripts/check-adr.py` | ADR frontmatter 규약, 색인과 상태 일치, 상대링크 무결성 |
| `scripts/check-mermaid.py` | 문서 내 mermaid 블록 렌더 가능 여부 |

`check-adr.py`는 pre-commit에 붙어 있고 공개 경계 검사도 함께 돈다. 경계 검사는 커밋 훅에서 `--require`로 도므로 패턴 목록이 없는 기계에서는 커밋이 막힌다(ADR [0033](decisions/0033-private-backup-and-fail-closed-boundary.md)). `check-mermaid.py`는 렌더에 브라우저를 띄워 느리므로 훅에 붙이지 않고 CI에서만 돈다.

## 관련

- [architecture.md](architecture.md) 동작 구조
- [design.md](design.md) 커맨드 체계와 마일스톤
- [journeys.md](journeys.md) 사용자 여정
- [ADR 색인](decisions/README.md)
