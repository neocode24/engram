# 세션 인계 노트

> 2026-08-13 작성. `~/Git/engram`에서 새 Hermes 세션을 열어 이어서 작업하기 위한 문서.
> 이 파일은 다음 세션이 첫 번째로 읽어야 할 문서다. 작업이 진행되면 갱신하고, ADR로 승격된 내용은 여기서 지운다.

## 다음 세션이 바로 할 일

아래 산출물을 작성하는 것이 이번 인계의 목적이다. 결정 내용은 이 문서 하단에 모두 정리되어 있으므로 추가 논의 없이 착수 가능하다.

1. `docs/decisions/0005-upstream-contract-and-harness.md` — llm-wiki 변경을 바이너리로 흘리는 harness 구조
2. `docs/decisions/0006-dual-mode-eject-seal.md` — easy/hard 듀얼 모드와 eject/seal/attach
3. `docs/decisions/0007-platform-and-distribution.md` — 지원 플랫폼, 배포, 업그레이드
4. `docs/decisions/0008-project-name-engram.md` — 이름 확정 근거
5. `docs/journeys.md` — 사용자 여정 15개와 마일스톤 매핑
6. `docs/design.md` 갱신 — engram 기준으로 커맨드 체계, 설정 2단 구조, 온보딩 반영
7. `README.md`의 "ADR 0008(작성 예정)" 문구를 실제 링크로 교체

### 착수 전 확인이 필요한 미결 사항

- **ADR 0002 처리 방식.** 0006이 0002의 전제(본인은 바이너리를 안 쓴다)를 바꾼다. 0002를 `superseded`로 내릴지, `accepted`를 유지하고 0006에 "결론은 유지, 근거 갱신"으로 적을지 사용자 확인 필요. 작성자 의견은 후자(이력 보존).

## 완료된 작업 (2026-08-13)

- 프로젝트 명칭을 `llm-wiki-edu`에서 `engram`으로 변경. 폴더 `~/Git/engram`, 리포 `neocode24/engram`(private), 기본 브랜치 `main`(구 `master` 삭제).
- ADR 0001, 0002는 본문을 소급 수정하지 않고 "당시 명칭 llm-wiki-edu" 병기만 추가.
- `~/Git/llm-wiki`의 `canopy.toml` 삭제 및 `.gitignore` 항목 정리(커밋 `730a092`). canopy는 설치 실험 부산물이며 upstream이 아니다.
- llm-wiki의 관련 문서(`inbox/manual/관련-문서.md`) 내 명칭 갱신.
- Hermes 스킬 `internal-tool-productization`의 reference 파일명을 `engram-design-notes.md`로 개명.

## 확정된 설계 결정

### 이름

`engram` = 기억이 뇌에 남기는 물리적 흔적. second brain을 표방하는 도구의 실체를 이름이 그대로 설명한다.

기각된 후보와 사유: `arbor`(AI 코딩 도구 인접 영역에 동명 프로젝트 다수), `loam`(건강식품 브랜드가 검색 점유), `bonsai`(동명 OSS 과다), `trellis`/`lattis`(격자 은유는 정확하나 사용자에게 와닿지 않음).

### 모드 전환 커맨드

| 명령 | 동작 | 비고 |
|---|---|---|
| `engram eject` | 내장 규칙을 실제 파일로 풀어놓는다 (easy에서 hard로) | 확인 프롬프트, drift 경고 |
| `engram seal` | 풀어놓은 파일을 거두고 내장 규칙으로 복귀 | 파괴적. 로컬 수정분 diff를 먼저 보여준다 |
| `engram attach` | 기존 hard mode 위키에 바이너리를 붙인다 | 파일 변경 없음. 후순위 |
| `engram pack` | 배포/공유용 번들 생성 | 예약어. 여정 14~15에서 사용 |

`pack`을 모드 전환에 쓰지 않은 이유는 배포 번들 자리에 그 이름이 필요하기 때문이다. `export`/`import`도 같은 이유로 기각했다(반출과 외부 자료 수집에 필요). `absorb`는 `seal`로 대체했다.

### easy / hard 모드

- **easy mode**: 바이너리가 전부 처리. 위키 폴더와 설정 파일만 있으면 되고 게이트는 바이너리 내부 검증으로 강제된다.
- **hard mode**: llm-wiki 원본 구조. `AGENTS.md` + `scripts/` + git hooks + lint. 게이트는 pre-commit hook과 에이전트 계약으로 강제된다.
- 둘은 별개 제품이 아니라 하나의 성장 경로이며, `eject`가 연결 고리다. 교육 서사에서 1~4주차는 easy, 마지막 주차에 eject하여 "이 강제력의 실체"를 열어 보이는 구성이 된다.
- 부수효과: ADR 0002의 열린 긴장("본인이 안 쓰는 도구를 가르친다")이 해소된다. 본인은 eject된 hard mode 사용자가 된다.
- 미결: eject를 단방향으로 못 박을지 여부. 작성자 의견은 단방향 + `doctor`가 drift를 알리는 쪽.

### Harness (upstream 계약 동기화)

upstream이 shell/python이고 downstream이 Go라 코드 재사용이 없다. 따라서 흘려야 하는 것은 코드가 아니라 **계약**이다.

실제 계약 원천은 llm-wiki의 `meta/` 아래 여섯 파일이다.

| 파일 | 계약의 성격 |
|---|---|
| `meta/frontmatter-schema.md` | 단계별 필수 필드 |
| `meta/promotion-rules.md` | 승급 게이트 판정 |
| `meta/ingest-rules.md` | 입력 수용 규칙 |
| `meta/taxonomy.md` | 태그 체계 |
| `meta/wiki-graph-policy.md` | 링크 불변식 |
| `meta/terminology-normalization.md` | 자동 치환 사전. 코드화하지 말고 데이터로 소비 |

여기에 `agents/workflows/*.md`(여정 정의)와 `scripts/*.py|sh`(알고리즘 재구현 대상)가 참고 자산으로 붙는다. `canopy.toml`은 계약이 아니다.

harness 3층 구조:

- **(a) upstream 스냅샷 vendoring** — `harness/upstream/`에 계약 파일만 pinned SHA로 복사하고 `harness/upstream.lock`에 커밋 해시 기록. llm-wiki가 private이고 사내 식별자를 포함하므로 전체 서브모듈은 금지하고, vendoring 스텝에 사내 식별자 grep 스캐너를 붙인다.
- **(b) 변경 감지와 spec-delta** — `make upstream-sync`가 lock의 SHA부터 HEAD까지 diff를 뽑아 `harness/deltas/YYYY-MM-DD.md` 생성.
- **(c) conformance 테스트** — `harness/fixtures/`의 깨끗한 골든 위키에 대해 upstream 스크립트와 Go 바이너리 출력을 비교. 비교 가능 항목은 `lint` 위반 목록, `resurface` 선정 순위, frontmatter 정규화 결과 세 가지. 결과는 `docs/parity.md`로 자동 갱신하며 홍보 자산이 된다. eject 산출물과 upstream 스냅샷의 diff가 네 번째 검증 축이다.

**전제조건 (llm-wiki 쪽 작업)**: llm-wiki에 `meta/CHANGELOG.md`를 신설하고(스키마/규칙 변경만, 항목마다 `impact: binary-affecting | wiki-only`), `AGENTS.md`에 "meta/ 아래 규칙 파일을 고치면 CHANGELOG 항목 추가"를 계약으로 명시해야 한다. 현재 `log.md`는 운영 로그이지 스펙 변경 로그가 아니다.

**함정**: resurface는 `meta/resurface-state.json`으로 상태를 들고 있어 비결정적이다. 골든 비교를 위해 바이너리에 `--now` 플래그를 처음부터 넣어야 한다.

### 플랫폼과 배포

가장 위험한 지점이다. 기존 design.md의 "bge-m3 ONNX"와 "Homebrew 배포"는 둘 다 Windows 전용 사용자를 배제한다. 사내 임직원 다수가 Windows만 사용한다.

**ONNX 문제**: onnxruntime Go 바인딩은 cgo와 플랫폼별 네이티브 라이브러리를 요구한다. cgo를 켜면 "단일 바이너리, 설치 한 방"이라는 존재 이유가 무너지고 Windows에서 DLL 배치, 백신 차단, MSVC 런타임 이슈가 따라온다.

층 분리:

- **코어(필수)**: 순수 Go, `CGO_ENABLED=0`. BM25 키워드 검색까지 포함. 이것만으로 강의 1~4단위가 완결된다.
- **시맨틱(선택)**: `engram model pull`로 런타임 다운로드하는 사이드카. 실패해도 코어는 동작하고 검색이 키워드 전용으로 degrade된다. 사내망 차단을 고려해 오프라인 번들 경로(`--from ./bge-m3.zip`)도 제공한다.

플랫폼 매트릭스는 windows/amd64, windows/arm64, darwin/arm64, darwin/amd64, linux/amd64, linux/arm64 여섯을 CI에서 빌드하되 **windows/amd64와 darwin/arm64를 tier 1**, 나머지를 tier 2로 명시한다.

| 플랫폼 | 설치 | 업그레이드 |
|---|---|---|
| Windows | winget, scoop, 관리자 권한 없는 환경용 zip + `%LOCALAPPDATA%` 설치 ps1 | `engram self-update` |
| macOS | Homebrew tap | brew upgrade 또는 self-update |
| Linux | curl 스크립트, tar.gz | self-update |

`self-update`는 GitHub Releases의 서명된 체크섬을 검증하고, 사내 프록시 환경을 위해 `ENGRAM_UPDATE_URL`로 내부 미러 미러를 가리킬 수 있어야 한다(ADR 0004의 배포 경로와 연결).

**Windows 고유 함정**: 경로 구분자, 대소문자 비구분 파일시스템(위키링크 해석), CRLF(autocrlf 켜진 사내 PC에서 lint 전량 실패. `.gitattributes` 템플릿 동봉 필수), 콘솔 UTF-8 한글 깨짐, 260자 경로 제한, 서명 없는 exe의 SmartScreen 차단(코드사이닝 없으면 winget/scoop 경유를 권장 경로로).

### 설치와 초기 구성

설정은 canopy가 쓰던 2단 구조를 따른다. 이 구조 자체는 유효하다.

- **사용자 전역** (`~/.config/engram`, Windows는 `%APPDATA%`): 관리 중인 위키 목록, 기본 에디터, 업데이트 채널, 모델 경로. 머신 고유이며 커밋 대상이 아니다.
- **위키별** (위키 홈의 설정 파일): 스키마 축, 임계값, 디렉토리 매핑, 모드. git에 커밋되어 팀이 공유한다.
- 우선순위: 전역 < 위키 < 환경변수/플래그. `config list --origin`으로 값의 출처를 보여준다(사내 지원 부담 절감).

커맨드:

- **`engram init`** — 대화형 마법사. 대상 폴더, 모드, 스키마 프리셋(personal/team/education), 언어, git 초기화, 시맨틱 검색 사용 여부를 순서대로 묻는다. 각 질문에 왜 묻는지 한 줄 설명을 붙이고, 폴더 생성 시 역할을 그 자리에서 인쇄한다(inbox는 처리 대기, sources는 원본 보존, context는 검수된 지식). 이 온보딩 텍스트가 곧 강의 1단위의 압축판이 된다.
- **첫 실행 감지** — 위키 루트를 못 찾으면 에러 대신 생성을 유도한다. 부모 디렉토리 역탐색, `ENGRAM_HOME` 환경변수, XDG 기본 위치 순으로 해석.
- **`engram config`** — 인자 없으면 대화형 편집, `config get/set`은 스크립트용.
- **`engram doctor`** — 사내 배포에서 필수. git 버전, autocrlf, 파일시스템 대소문자 구분, 콘솔 인코딩, 모델 파일, 프록시 도달성, 스키마 유효성, 인덱스 신선도, 권한을 점검하고 각 실패에 복구 명령을 함께 출력한다.
- **`engram tour`** — 샘플 문서로 한 사이클을 유도하는 튜토리얼. 0.2 이후.

초기 디렉토리: `inbox/ sources/ context/ archive/ meta/` + 설정 파일 + `index.md` + `log.md`. llm-wiki의 iCloud 미러와 회사 특수 축은 프리셋으로 분리하여, `team` 프리셋에서만 sensitivity 축이 켜지게 한다. 즉 **9축을 잘라내지 않고 프리셋으로 켜고 끈다**. 이것이 기존 design.md의 "스키마 보편화 범위" 미결정에 대한 답이다.

### 사용자 여정 15개

기존 curriculum.md의 여정 5개는 CLI 명령 나열에 가까워 실제 시나리오가 아니다. llm-wiki의 `agents/workflows/` 8개를 뼈대로 삼고 누락분을 채운 결과가 아래다.

**A. 입력**

1. **회의록** — 회의 중 러프 캡처, 회의 후 구조화(결정/액션/미결), 액션 아이템 외부 반출, 다음 회의 직전 이전 회의록 자동 회수. 마지막 고리가 핵심 가치다. upstream `meeting-intake-promote.md`.
2. **음성 녹음** — 전사, 화자 분리, 용어 사전 자동 치환, 사람이 오탈자 교정, 교정 내역이 사전에 피드백되어 다음 전사가 개선. 이 피드백 루프가 숨은 무기다. 원본 오디오는 커밋하지 않는다. 로컬 STT는 무거우므로 바이너리는 "외부 전사기 호출과 결과 수용" 인터페이스만 정의한다. upstream `voice-memo-intake.md`.
3. **텍스트 드롭** — 링크/클립보드 한 방 캡처. upstream `text-drop-intake.md`.
4. **주간 뉴스 수집** — 정기 수집, 요약, 관심사 필터. upstream `weekly-news-intake.md`.

**B. 승급**

5. **inbox 정리 세션** — 쌓인 inbox를 훑으며 폐기/원본화/승급 분류. 며칠째 몇 개 쌓였는지 압력 지표 필요.
6. **게이트 실패와 교정** — promote가 거절되고 무엇을 채워야 하는지 안내받아 통과. 거절 경험이 곧 교육이므로 에러 메시지 품질이 제품 품질이다.

**C. 회수**

7. **작업 중 회수** — 현재 작업과 관련된 과거 지식을 검색으로 회수.
8. **MCP 연결** — 위키를 MCP 서버로 노출. 쟁점은 쓰기 권한 범위다. 에이전트가 context에 직접 쓰면 승급 게이트 철학이 무너지므로 **MCP 쓰기는 inbox까지만 허용하고 승급은 사람 확인을 강제**한다. 이것이 다른 PKM MCP와의 차별점이 된다.

**D. 재발견 (제품 정체성)**

9. **오래 안 본 문서 역제안** — "90일째 안 봤습니다. 아직 유효한가요?" 유효/폐기/보강 세 갈래.
10. **연관 제안과 묶기** — 유사하지만 링크 없는 문서 쌍을 알리고 묶을지 질문. 수락 시 양방향 링크 삽입 또는 상위 개념 노드 생성 제안.
11. **정기 다이제스트** — 주간 요약. 신규, 승급, 노후, 고아 문서.

**E. 운영**

12. **드리프트 점검과 대량 마이그레이션** — 스키마 변경 시 기존 문서 수백 개를 따라오게 한다. `engram migrate --dry-run`. llm-wiki에서 실제 반복 겪은 문제이며 `sync_updated_field.py`, `backfill_source_dates.py`가 그 흔적이다.
13. **동기화와 충돌** — 2대 PC, git 기반. 프론트매터 병합 지원.
14. **외부 반출** — 보고서/블로그/발표자료로 내보내기. 익명화와 민감도 필터가 걸린다. 사내 교육 맥락에서 필수. upstream `weekly-report-draft.md`, `blog-publish.md`.
15. **팀 공유** — 개인 위키 일부를 팀과 공유. `serve` 웹 UI가 여기 붙는다.

마일스톤 매핑 초안: 1/3/5/6/7이 0.1~0.2, 9/10/11이 0.3, 8/14/15가 1.0. 2번(음성)은 외부 의존이라 별도 트랙.

## 작업 시 참고

- 설계 논의에는 Hermes 스킬 `internal-tool-productization`을 사용한다. 세션 기록은 `references/engram-design-notes.md`.
- upstream 진실원은 `~/Git/llm-wiki`이며, 그 저장소를 건드릴 때는 반드시 `~/Git/llm-wiki/AGENTS.md`의 계약을 먼저 읽는다.
- 이 저장소는 아직 테스트 스위트가 없다. 검증은 임시 스크립트로 수행하고 그 사실을 명시한다.
