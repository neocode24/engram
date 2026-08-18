# upstream 미해결 항목

> 다섯 감사 보고서(A/B/C/D/E)에서 "없음" 또는 "다르게 구현됨" 판정을 받은 항목을 중복 제거하고 카테고리별로 정리한 문서다. 각 항목은 upstream 규정, engram 대응, 영향, 카테고리를 포함한다. 나중에 무엇을 만들지 고를 때 읽는 문서이므로 항목마다 판단 재료가 갖춰져야 한다.

## 카테고리

- **경계**: 공개 경계, 보안, 민감도, 접근 제어, 출처 관리 관련
- **재발견**: 재발견 루프, bridge, resurface, 색인 관련
- **기능**: 그 외의 기능 구현, 인터페이스, 운용 절차

## 자리

각 항목이 engram의 어디에 속하는지다. 기준은 `docs/spec-map.md` 3절의
"같은 입력에 같은 출력이 나오는가"다.

| 자리 | 뜻 | 건수 |
|---|---|---|
| 코드 | lint 규칙, 게이트, 판정, 커맨드 동작 | 46 |
| 설정 | `engram.yaml`의 값 | 10 |
| 데모 | `examples/personal` 데모 위키가 보여 줄 본보기 | 7 |
| 교재 | `docs/course/`가 가르칠 것 | 5 |
| 밖 | engram이 관여하지 않는다 | 34 |

**`코드`와 `설정`이 만들 것이다.** 나머지는 데모 위키와 교재의 몫이거나
engram이 관여하지 않는다.

한 항목에는 자리 하나만 붙는다. 둘에 걸치는 항목은 가벼운 쪽을 골랐고 그
이유를 자리 줄에 함께 적었다. 코드가 가장 무겁고 밖이 가장 가볍다. 코드로
만든 것은 되돌리기 어렵고 문서는 쉬우므로 확신이 서지 않는 항목은 가벼운
쪽으로 기울였다. 그렇게 기운 항목은 문서 끝의 "판정이 갈린 것"에 모았다.

제목에 `(닫힘, ADR NNNN)`이 붙은 항목은 이미 구현되어 커밋된 것이다.

## 선언과 구현이 어긋난 것

위 셋은 주제별 분류다. 그것과 별개로 **engram이 스스로 선언한 규칙과 실제 동작이 어긋나는 항목**을 따로 모은다. 이쪽은 "덜 만들었다"가 아니라 "말한 것과 다르게 동작한다"이므로 무엇을 더 만들지와 무관하게 닫아야 한다.

| 항목 | 무엇을 선언했나 | 실제 동작 |
|---|---|---|
| [G4](#g4-indexable-값-읽기-부재-닫힘-adr-0063) | `lint`가 `indexable`을 모든 문서의 필수 필드로 요구한다 | 그 값을 판정에 쓰는 코드가 없다 |
| [G5](#g5-status-superseded-필터-부재-닫힘-adr-0063) | [ADR 0044](decisions/0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 `serve`는 검수된 지식만 보여준다고 정했다 | 노출 판정이 `status`를 보지 않는다 |
| [G8](#g8-sensitivity-internal-반출-누락-닫힘-adr-0063) | `export`가 민감도로 반출을 거른다 | 승급 문서의 기본 민감도가 `internal`이고 그 값은 안 걸린다 |
| [G9](#g9-sources-원본-보존-위반-닫힘-adr-0064) | `source` 커맨드가 원본 보존을 계약이라 선언한다 | `update`가 같은 문서 본문을 경고만 내고 고친다 |
| [G1](#g1-indexmd-자동-갱신-부재) | [ADR 0055](decisions/0055-agents-change-the-wiki-only-through-commands.md)가 에이전트는 커맨드로만 위키를 바꾼다고 정했다 | `index.md`를 바꾸는 커맨드가 없다 |
| [F62](#f62-마크다운-링크-파싱-부재-닫힘-adr-0065) | `link.broken`과 `graph.orphan`이 문서 사이 관계를 판정한다 | 마크다운 링크를 아예 안 세므로 실측에서 링크 166개가 0으로 읽혔다 |

`sources`의 `source_refs` 값 검사 부재([G6](#g6-source_refs-값-검사-부재))는 이 목록에 넣지 않는다. upstream `scripts/lint-frontmatter.sh`도 존재만 보므로 engram 고유의 어긋남이 아니다.

---

## 경계

### G1. `index.md` 자동 갱신 부재

**upstream 규정**: `agents/workflows/context-node-add.md:45`, `scripts/sync_wiki_index.py:2`, `AGENTS.md:52-60`가 승급 완료 후 `index.md`에 type별 섹션에 wikilink 한 줄을 자동 추가하고 설명을 blockquote에서 자동으로 채운다. append-only 불변식을 유지한다.

**engram 대응**: 없다. `internal/cli/init.go:108`이 `index.md`를 한 번 만들고 이후 손대지 않는다. `docs/spec-map.md:232`가 "색인을 자동으로 고치지 않는 것은 의도된 선택"이라 적으나 ADR은 없다.

**영향**: 위키가 커질수록 새 문서가 색인에서 빠진 채 남는다. engram의 `graph.orphan`은 관계가 아예 없는 문서만 잡으므로 색인 누락은 걸리지 않는다. upstream의 드리프트 점검이 engram에서 성립하지 않는다.

**자리**: 코드

**출처**: A5, B15, D1, E22, E27

---

### G2. `log.md` 기록 부재

**upstream 규정**: `agents/workflows/context-node-add.md:46, :59`, `AGENTS.md:52-60`가 승급 완료 후 `log.md`에 `## YYYY-MM-DD | promote | <title>` 항목을 append한다. Source/Changed/Decision 세 줄을 기록한다.

**engram 대응**: 없다. `digest`가 기간 집계를 내지만 도움말이 "승급 집계는 promote가 승급 시각을 프론트매터에 남기지 않아 여기에 없습니다"라고 밝힌다. `docs/spec-map.md:232`가 미반영으로 적는다.

**영향**: 승급 이력 추적이 불가능하다. upstream은 새 문서가 생길 때마다 목차와 로그가 자동으로 갱신되는 것을 전제로 절차를 짰다.

**자리**: 코드. 파일 형태는 데모의 몫이나 승급 시각을 프론트매터에 남기는 데까지는 코드다

**출처**: A6, D2, E22, E27

---

### G3. 대량 커밋 필터 부재 (닫힘, ADR 0066)

**upstream 규정**: `scripts/wiki_resurface.py:99-133`, `scripts/sync_updated_field.py:32-34, 52`가 파일 수정 시각을 git 커밋 시각에서 얻되 `BULK_COMMIT_THRESHOLD`(15) 이상 파일을 건드린 커밋은 건너뛴다. 근거로 실제 사고를 적었다(2026-08-07 `quality_level` 폐기 커밋이 `context/` 79개를 오늘 날짜로 바꿔 resurface 결과가 0건이 됨).

**engram 대응**: `internal/resurface/resurface.go:51-62`는 프론트매터 `updated`/`created`를 읽는다. 벌크 커밋 개념이 없다. `internal/cli/sync.go:109`가 마지막 커밋 날짜를 그대로 쓴다. ADR 0037은 `sync_updated_field.py`를 대조 대상으로 지목하면서도 이 규칙을 옮기지 않았다.

**영향**: `migrate`는 정의상 문서 전체의 프론트매터를 한 번에 고치는 커맨드이고, 그 커밋 뒤에 `engram sync --apply`를 돌리면 모든 `context` 문서의 `updated`가 오늘이 된다. 그러면 `resurface`는 `stale_days`를 넘는 문서를 하나도 찾지 못하고, 같은 날짜를 읽는 `digest`의 노후 집계도 0이 된다. upstream이 겪은 사고가 engram에서 그대로 재현된다.

**자리**: 코드

**출처**: B3, C3, E7

---

### G4. `indexable` 값 읽기 부재 (닫힘, ADR 0063)

**upstream 규정**: `scripts/sync_wiki_index.py:171-177`, `scripts/lint-frontmatter.sh:271-282`, `context/systems/indexing-config.md:104, 106`가 `indexable: false`인 문서와 `artifact_stage: inbox`인 문서는 색인에서 빼고, `context` + `indexable != true`는 경고한다. 색인 자격 판정에 `indexable`과 `status: promoted`를 쓴다.

**engram 대응**: engram은 `indexable` 값을 읽는 코드가 없다. `promote`가 모든 문서에 `indexable: true`를 무조건 박는다(`internal/cli/promote.go:467`). 검색 색인은 `indexable: false`인 문서도 색인하고 `serve`/`mcp`/`export`도 그 값을 보지 않는다.

**영향**: 사용자가 `indexable: false`를 적으면 아무 일도 일어나지 않고 그 사실을 알려 주는 경고도 없다. upstream 색인 계약이 `indexable: false`와 `status: superseded`를 명시적 제외 조건으로 못 박은 이유가 여기 있다.

**자리**: 코드

**출처**: A8, B19, B29, C11, C12, E37

---

### G5. `status: superseded` 필터 부재 (닫힘, ADR 0063)

**upstream 규정**: `context/systems/indexing-config.md:106`가 `status`가 `inbox`, `archived`, `superseded`인 파일을 제외한다.

**engram 대응**: `archived`는 위치(`archive/`)로 걸리고 `inbox`도 위치로 걸린다. `superseded`는 스키마 허용값이지만(`internal/config/config.go:246`) 노출에서 거르지 않는다. `context/`에 남은 채 `status: superseded`인 문서는 `search`, `serve`, `export`, `mcp` 전부에 나온다.

**영향**: 폐기된 결정이 `context/`에 남아 있으면 그대로 검색에 나오고 MCP를 통해 에이전트에게 답변 근거로 간다.

**자리**: 코드

**출처**: A10, C12

---

### G6. `source_refs` 값 검사 부재

**upstream 규정**: `context/systems/indexing-config.md:130-141`가 색인 문서는 source 증거로 추적 가능해야 한다고 규정한다. "시스템은 출처 증명이 누락된 것을 보완하지 않는다. 승급 문서의 `source_refs` 누락은 검수 이슈로 다뤄야 한다."(`:141`)

**engram 대응**: `internal/lint/lint.go:384-388`의 `checkRequiredFields`가 `if _, ok := s.fields[f]; !ok`로 키 존재만 본다. `source_refs: []`는 통과한다. 값이 비었는지 검사하는 코드가 없다.

**영향**: `promote`가 만든 문서는 관계 필드가 빈 배열로 들어가므로 게이트를 통과한 문서 전부가 추적성 검사를 형식적으로만 통과한다. upstream이 "promoted 문서의 `source_refs` 누락은 검수 이슈로 다뤄야 한다"고 적은 그 상태가 engram에서는 정상 판정을 받는다.

**자리**: 코드. 값이 비었는지의 판정은 결정론적이다

**출처**: A2, C16

---

### G7. 민감도에 따른 승급 제한 부재 (닫힘, ADR 0069)

**upstream 규정**: `agents/workflows/context-node-add.md:53`, `AGENTS.md:12`(Core Principle 4), `promotion-review-checklist.md:52-57`가 민감정보(계정/토큰/개인정보/내부 보안)는 context 승급 금지한다.

**engram 대응**: `internal/cli/promote.go`에 `sensitivity` 문자열이 없다. 게이트가 민감도를 보지 않는다. 거르는 곳은 `internal/expose`뿐이고 그것은 `serve`와 `export`에만 걸린다.

**영향**: 민감 자료가 context에 올라가는 것 자체는 아무도 막지 않는다. `docs/spec-map.md`가 security-rules 전체를 미반영으로 적은 것과 같은 구멍이다.

**자리**: 코드. 무엇이 민감한지는 사람이 정하나 정해진 값으로 승급을 막는 것은 결정론적이다

**출처**: A9, E36

---

### G8. `sensitivity: internal` 반출 누락 (닫힘, ADR 0063)

**upstream 규정**: `agents/workflows/blog-publish.md:79-80`가 역류 시 `scope: work` / `sensitivity: internal` 맥락은 발행처에서 추상화하거나 제거한다. 실명, 계정, 토큰, 내부 보안 세부는 역류와 발행을 금지한다.

**engram 대응**: `export --help`가 빼는 것은 `private-local-only`와 `restricted` 둘뿐이다. `internal`은 그대로 나간다. `scope: work`는 반출 판정에 아예 쓰이지 않는다.

**영향**: 사내 회의에서 나온 `internal` 문서가 승급되면 `export`가 그것을 아무 경고 없이 내보낸다. `internal/wiki/wiki.go:315`가 inbox를 뺀 모든 단계의 기본 민감도를 `internal`로 주고 `promote`는 그 값을 손대지 않으므로, 사람이 손으로 올리지 않는 한 승급된 문서는 전부 반출 가능 상태로 시작한다.

**자리**: 코드

**출처**: A39

---

### G9. sources 원본 보존 위반 (닫힘, ADR 0064)

**upstream 규정**: `AGENTS.md:28-32`(Core Principles 2), `sources/README.md:16-23`가 "`sources/`는 원본과 출처를 보존한다. 가능한 한 수정하지 않는다"고 규정한다.

**engram 대응**: `engram update sources/2026-08-01-원본.md --set status=promoted`가 아무 경고 없이 통과했다. 단계별 쓰기 제한이 없다. ADR 0055는 "커맨드로만 바꾼다"를 정하나 그 커맨드가 `sources/`를 못 고치게 하지 않는다.

**영향**: 원본 보존 계약이 코드로 강제되지 않는다.

**자리**: 코드

**출처**: E9

---

### G10. 한국어 특수문자 규칙 강제 부재

**upstream 규정**: `confluence-authoring-draft.md:44-56`, `AGENTS.md:220-234`가 한국어 산문에 em dash, en dash, 화살표, 가운뎃점, 말줄임표, smart quotes를 쓰지 않는다. 단일 진실원은 `docs/korean-typography.md`와 `scripts/typography-check.mjs`다.

**engram 대응**: engram은 같은 규칙을 `CLAUDE.md`에 문장으로만 갖고 있다. 검사 스크립트가 없다. `ls scripts/` 결과는 `check-adr.py`, `check-boundary.py`, `check-mermaid.py`, `private-backup.sh`, `upstream-sync.py`, `windows-verify.ps1` 여섯이고 문체 검사는 없다.

**영향**: 한국어 특수문자 규칙이 강제되지 않는다.

**자리**: 밖. 설정으로 켜는 검사로도 가능하나 위키 안의 산문 문체는 조직마다 다르고 언어도 다를 수 있다

**출처**: D23, E8

---

### G11. 승급 문서 형태 강제 부재

**upstream 규정**: `context/README.md:20-32`, `promotion-review-checklist.md:31-34`가 승급 문서 형태를 `# Title`, `> One-line judgment`, `## Context`, `## Current Understanding`, `## Evidence`, `## Related`로 규정한다.

**engram 대응**: `new`는 다섯 절 골격을 넣는다(결론/맥락/현재 이해/근거/관련 링크, `internal/cli/new.go`). `promote`는 넣지 않는다. 실측: `capture` 한 줄짜리 메모를 `promote`하면 본문이 `테스트 메모` 한 줄 그대로이고 H1도 절도 없다. 형태 미강제는 ADR 0040의 의도된 선택이나, 두 커맨드가 다르게 동작하는 것은 그 ADR이 다루지 않는다.

**영향**: 승급 문서의 형태가 일정하지 않다.

**자리**: 데모. 본문의 절 구성이므로 본보기를 보여 주면 된다

**출처**: E11

---

### G12. 타입별 절 구성 부재

**upstream 규정**: `meta/templates/context-decision.md:21-31`, `context-procedure.md:21-33`, `context-system.md:21-31`가 타입마다 절 구성이 다르다. decision은 `Context/Decision/Rationale/Consequences/Evidence/Related`, procedure는 `Context/Preconditions/Procedure/Approval Boundaries/Evidence/Related`, system은 `Context/Current Understanding/Interfaces/Constraints/Evidence/Related`로 규정한다.

**engram 대응**: `internal/cli/new.go`의 골격은 타입과 무관하게 다섯 절 하나로 고정이다. `--type decision`으로 만들어도 `Decision`, `Rationale`, `Consequences` 절이 없다. `Approval Boundaries`, `Interfaces`, `Constraints`, `Preconditions`는 engram 어디에도 없다.

**영향**: 타입별 문서 형태 차별이 없다.

**자리**: 데모. 타입별 절 구성은 본보기의 몫이다

**출처**: E12

---

### G13. 시크릿 스캔 부재 (닫힘, ADR 0069)

**upstream 규정**: `AGENTS.md:69-89` Git Completion Policy가 pre-commit 게이트가 frontmatter lint, `index.md` 동기화, 시크릿 스캔 셋을 자동 수행한다(`.githooks/pre-commit:29,63,77`)고 규정한다.

**engram 대응**: engram은 lint만 제공한다. 색인 동기화와 시크릿 스캔이 없다. 시크릿 스캔 미구현은 `docs/spec-map.md` 4.7절이 "이 명세는 engram이 가장 적게 옮긴 명세다"로 이미 밝혔다.

**영향**: 시크릿이 커밋되는 것을 막는 게이트가 없다.

**자리**: 코드

**출처**: E23

---

### G14. 사람 문서 조건부 색인 부재

**upstream 규정**: `context/systems/context-indexing-boundary.md:52-58`가 기본 색인 대상은 `context/` 아래 여섯 하위 디렉토리다. `context/people/`만은 비민감하고 의도적으로 승급된 경우에만 색인한다.

**engram 대응**: engram은 `context/` 아래 하위 디렉토리를 구분하지 않는다(`internal/config/config.go:220`, `expose.go:Locate`). 사람 문서에 대한 조건부 색인 특례가 없다.

**영향**: 사람에 관한 문서는 승급 실수의 대가가 다른 문서와 다르다.

**자리**: 밖. 하위 디렉토리를 사용자 자유로 둔 spec-map 4.2를 따른다

**출처**: C20

---

### G15. 증거 표현 검토 부재

**upstream 규정**: `notebooklm-export-workflow.md:124-136`, `voice-memo-local-stt-diarization.md:147-165`가 반출/승급 전에 원자료의 표현을 검토한다. STT 용어 오류, 불확실한 직함과 조직명, 구어체, 사람 중심 서술을 고친다. "잘못은 원자료에서 고치고 팩을 다시 만든다. 최종 산출물만 손보지 않는다."

**engram 대응**: 그런 단계도 커맨드도 없다. `export`는 `--replacements` 치환 사전으로 문자열을 바꿀 뿐이고 그것은 익명화이지 표현 검토가 아니다.

**영향**: 원자료 표현 검토 절차가 없다.

**자리**: 교재. 언제 원자료를 검토하는지를 가르치면 된다

**출처**: D14

---

## 재발견

### R1. 임베딩/시맨틱 검색 부재

**upstream 규정**: `scripts/wiki_resurface.py:8-10, 289-292`, `context/decisions/wiki-resurface-loop.md:78-83`이 bridge는 단어 겹침과 bge-m3 임베딩을 둘 다 돌려 합집합을 본다. "실측상 상위 15쌍 중 겹치는 것이 1쌍뿐이었다." 자연어 질의에서 의미 방식이 어휘 방식을 압도했다는 실측이 있다.

**engram 대응**: `internal/bridge/bridge.go:74,107-122`가 TF 코사인 하나만 쓴다. `internal/` 전체에 임베딩 코드가 없다. `docs/design.md:102`에 `model pull`이 있으나 `engram --help`에 없다(미구현).

**영향**: 단어 방식과 임베딩이 잡는 쌍이 거의 겹치지 않는다는 실측에 따라, 공통 단어 없이 같은 이야기를 하는 문서 쌍을 구조적으로 못 잡는다. 재발견 세 축 중 관계 축이 절반만 도는 셈이다.

**자리**: 코드

**출처**: B1, B14, C1, C7, D21

---

### R2. 쿨다운 부재 (닫힘, ADR 0066)

**upstream 규정**: `scripts/wiki_resurface.py:47`, `context/decisions/wiki-resurface-loop.md:84`가 `COOLDOWN_DAYS = 21`이다. 한 번 노출된 페이지를 이 기간 동안 후보에서 제외한다.

**engram 대응**: `internal/resurface/resurface.go:121-142`는 제시 이력을 제외가 아니라 정렬 키로 쓴다. 미제시 문서 우선, 그다음 오래 전 제시 순. 쿨다운 상수 없다(`rg -rn 'cooldown\|쿨다운' internal/ docs/` 무결과).

**영향**: 최근 제시된 문서도 다시 후보로 나온다.

**자리**: 코드

**출처**: B4, C2

---

### R3. 인바운드 링크 가중치 부재 (닫힘, ADR 0066)

**upstream 규정**: `scripts/wiki_resurface.py:282-283`이 재발견 점수는 `age_days * (1.0 + 1.0/(1+인바운드 링크 수))`다. 링크가 적을수록 가중치를 준다.

**engram 대응**: `internal/resurface/resurface.go:128-142`의 정렬은 (미제시 여부, 마지막 제시, 경과일, 슬러그)뿐이다. 인바운드 링크 수가 정렬에 들어가지 않는다. `docs/parity.md:130-134`가 이미 실측으로 이 차이를 잡아 "따라갈 후보"로 남겨 두었다.

**영향**: 경과일만 같으면 잘 연결된 문서와 고립된 문서가 같은 순위다. 재발견의 목적이 "잊힌 것을 꺼내는 것"인데 잊힘의 강도를 재는 축 하나가 빠져 있다.

**자리**: 코드

**출처**: B4, B8, C4

---

### R4. 고아 정의 방향 차이 (닫힘, ADR 0066)

**upstream 규정**: `scripts/wiki_resurface.py:334`가 고아는 인바운드 링크 0이고 `type != "moc"`인 페이지다. 판정에 아웃바운드를 쓰지 않는다.

**engram 대응**: `internal/lint/lint.go:693-695`는 아웃바운드 위키링크, 관계 필드, 인바운드가 전부 0일 때만 고아로 본다. 다른 문서를 많이 링크하지만 아무도 걸어주지 않는 문서를 engram은 고아로 보지 않는다.

**영향**: 재발견 루프에서 고아 탐지의 목적이 "잊힐 위험"을 재는 것인데, 잊힐 위험은 인바운드가 결정한다. `digest`의 고아 집계와 `status`의 지표가 같은 함수를 쓰므로 세 곳이 함께 어긋난다.

**자리**: 코드

**출처**: B10

---

### R5. bridge 기본 임계값 차이

**upstream 규정**: `scripts/wiki_resurface.py:49`가 `BRIDGE_MIN_SIM = 0.18` 코사인 하한다.

**engram 대응**: `internal/cli/bridge.go:139` `--min` 기본값 `0.30`이다. 값이 다르다. ADR 0028 "열린 항목"이 기본 임계값을 미결로 남겼다.

**영향**: 기본 임계값이 다르다.

**자리**: 설정

**출처**: B7

---

### R6. 재발견 실행 주기 부재

**upstream 규정**: `scripts/wiki_resurface.py:251` 기본 개수 3, `context/decisions/wiki-resurface-loop.md:132` 실행 주기 주 1회 토요일 08시다.

**engram 대응**: `engram --help`에 `resurface --limit` 기본값 확인 안 함. engram에 스케줄 개념이 없다.

**영향**: 스케줄 실행이 없다.

**자리**: 교재. 스케줄은 조직의 운용이나 언제 돌리는지는 가르칠 몫이다

**출처**: B3절, C3절

---

### R7. 노후 판정 기준일 차이 (닫힘, ADR 0067)

**upstream 규정**: `scripts/wiki_resurface.py:48`가 `MIN_STALE_DAYS = 30`이다. "이보다 최근에 만진 페이지는 후보가 아니다."

**engram 대응**: `internal/config/config.go`의 `StaleDays`가 대응한다. 기본값이 90이었으나 [ADR 0067](decisions/0067-stale-days-default-is-thirty.md)이 30으로 바꿔 upstream과 같아졌다.

**영향**: 해소되었다. 90에서는 실운영 위키 308문서 기준 `resurface` 후보가 0건이었다.

**자리**: 설정

**출처**: B6

---

### R8. 용어 정규화 부재

**upstream 규정**: `voice-memo-terminology-normalization.md:37-50`가 원본 증거는 고치지 않는다. 정규화 대상은 inbox 초안, 승인된 `sources/summaries/*`, `sources/manifests/*`의 제목과 취급 주의, 승급된 `context/*`다. `meta/terminology-normalization.md`의 `Auto-correct? = yes` 행을 post-STT find/replace로 적용한다. `review`/`conditional` 행은 절대 자동 적용하지 않는다.

**engram 대응**: 치환 사전은 `export --replacements` 사용자 사전 하나뿐이고(ADR 0046) 반출 시점에만 돈다. 인입/정제 단계 정규화가 없고, 조건부 항목을 구분하는 개념도 없다.

**영향**: 위키 안에 잘못 인식된 용어가 그대로 쌓이고, 그 상태로 색인되면 `search`와 `recall`이 그 용어를 못 찾는다. upstream이 "자동 적용 가능한 것"과 "사람이 봐야 하는 것"을 나눈 구분도 engram에 없다.

**자리**: 밖. 치환 실행은 결정론적이나 무엇을 언제 정규화할지는 전사 도구와 조직 어휘에 매인다

**출처**: A28, D15

---

### R9. 토큰화 방식 차이

**upstream 규정**: `scripts/wiki_resurface.py:57-61`이 유사도 토큰화에서 한국어/영어 불용어 20개를 제거하고 한글 2자 이상, 영문 3자 이상만 토큰으로 본다. `df < 2`인 단어는 잡음으로 버린다(`:158`).

**engram 대응**: engram은 문자 bigram 토크나이저를 쓴다(ADR 0010). 불용어 목록도 df 하한도 없다(`rg -rn 'STOPWORD\|불용어' internal/` 무결과).

**영향**: 토큰화 방식이 다르다.

**자리**: 코드

**출처**: B11

---

### R10. 코드블록 제외 부재

**upstream 규정**: `scripts/wiki_resurface.py:137-138`이 유사도 계산 전에 코드블록과 위키링크를 본문에서 제외한다.

**engram 대응**: `internal/bridge/bridge.go:55`는 색인의 TF를 그대로 쓴다. ADR 0061이 색인 필드 가중치를 정했으나 코드블록 제외는 그 결정에 없다.

**영향**: 코드블록이 유사도 계산에 포함된다.

**자리**: 코드

**출처**: B12

---

## 기능

### F1. 트리거 표 부재

**upstream 규정**: `agents/workflows/context-node-add.md:33-37`, `voice-memo-intake.md:32-47`가 User-Facing Commands 표를 둔다. "context에 추가해줘", "L2로 올려줘", "이 내용 재사용 가능하게 정리해줘" 같은 자연어를 워크플로 실행으로 매핑한다. "The user should not need to mention script names."

**engram 대응**: `internal/skills/SKILL.md` 239줄에 트리거 표가 없다. "커맨드 갈래" 절이 커맨드를 성격별로 묶을 뿐 사용자 발화와 잇지 않는다.

**영향**: 사용자가 "이 회의록 정리해줘"라고 했을 때 에이전트가 어느 커맨드 순서로 들어가야 하는지가 문서에 없다.

**자리**: 교재. `internal/skills/SKILL.md`의 자리이나 그것은 권고이고 판정이 아니므로 가르치는 쪽에 둔다

**출처**: A12, A25

---

### F2. `inbox/_drop/` 부재

**upstream 규정**: `agents/workflows/text-drop-intake.md:50-58`, `:108`, `:116`가 `inbox/_drop/`에 아무 이름으로 던진다. 파일명 규칙은 없다. 처리 후 비운다. `_drop/`은 인덱싱 대상이 아니다.

**engram 대응**: `docs/spec-map.md:299-303`이 이 폴더를 만들지 않기로 하고 "커맨드가 없어서 생긴 우회로" 판정한다. 디렉토리 단위 제외 개념인 `ignore_dirs`가 없다(grep 무결과). 파일 단위 `ignore_files`만 있다(ADR 0036).

**영향**: 텍스트 드롭 인테이크 방식이 다르다.

**자리**: 밖. spec-map이 이미 우회로로 판정해 만들지 않기로 했다

**출처**: A17

---

### F3. 인입 시점 채널 추론 부재

**upstream 규정**: `agents/workflows/text-drop-intake.md:64-80`이 파일명, 크기, 첫 20줄을 읽어 채널, 날짜, slug, 민감도를 추론하고 먼저 제시해 확인을 받는다. 추론할 수 없으면 추측하지 말고 묻는다. `source_channel`을 본문의 화자 표기, "Teams 모임", Confluence 매크오 흔적, 메일 헤더에서 추론한다.

**engram 대응**: `internal/skills/SKILL.md`가 "제목과 슬러그를 사용자에게 묻지 마라. 내용에서 추론하고 확인만 받는다"고 적어 제목/슬러그는 같다. 채널과 날짜와 민감도의 추론과 확인은 없다. `engram capture --help`에 `--channel`이 없다. `source`와 `promote`에만 있다.

**영향**: 인입 시점에 채널을 줄 방법이 없다. `source_channel` 속성이 켜진 위키에서 inbox 문서는 전부 필수 필드를 빈 값으로 갖는다.

**자리**: 코드. 추론은 에이전트의 몫이나 값을 줄 플래그가 없는 것은 커맨드의 결손이다

**출처**: A18, A19

---

### F4. 채널 하위 디렉토리 부재

**upstream 규정**: `agents/workflows/text-drop-intake.md:100`이 draft를 `inbox/<channel>/YYYY-MM-DD-<slug>-meeting-note-draft.md`에 만든다. `inbox-processing-matrix.md:47-60`이 `inbox/<channel>/` 채널 디렉토리 여덟을 쓰고 워크플로가 생겼다고 새 inbox 폴더를 만들지 않는다.

**engram 대응**: `internal/cli/capture.go:70`이 `wiki.Create(root, cfg, wiki.StageInbox, date, slug, ...)`로 `inbox/` 평면에 놓는다. 채널 하위 디렉토리를 만들지 않는다. `--slug`로 흉내낼 수 없다(슬러그에 `/`가 못 들어간다, ADR 0045/0050).

**영향**: 채널별 하위 디렉토리 구조가 없다.

**자리**: 밖. 채널 여덟은 조직 어휘이고 단계 디렉토리 아래는 사용자의 자유다

**출처**: A20, D4

---

### F5. 회의록 절 구조 부재

**upstream 규정**: `agents/workflows/meeting-intake-promote.md:45`가 source 요약은 meeting-note 표준 헤딩 열을 갖는다. Metadata, Speaker Identification, One-Paragraph Summary, Key Discussions, Decisions/Alignment, Action Items, Open Questions, Promotion Candidates, Evidence Pointers, Review Notes. `slack-teams-meeting-intake.md:46-65`가 회의록 초안과 source 요약은 영어 절 제목 열 개를 고정으로 쓰고 본문만 한국어로 쓴다.

**engram 대응**: 본문 구조를 강제하거나 템플릿을 심는 것이 engram에 없다. lint 규칙 14종 중 본문을 보는 것은 `body.max-lines` 하나다. engram의 절 골격은 `new`가 만드는 승급 문서용 다섯(결론, 맥락, 현재 이해, 근거, 관련 링크)뿐이다. `capture`와 `source`는 골격을 만들지 않으므로 회의록 초안에 대응하는 골격이 없다.

**영향**: 회의록 구조가 표준화되지 않는다.

**자리**: 데모. 본문의 절 구성이다

**출처**: A14, D17

---

### F6. 직급 추상화 부재

**upstream 규정**: `agents/workflows/meeting-intake-promote.md:45`, `:48`가 직급은 역할 중심으로 추상화한다. 실명, 직함, 내부 의사결정은 L0에서만 보존하고 source/L2는 추상화한다.

**engram 대응**: 치환은 `export --replacements` 사용자 사전 하나뿐이고(ADR 0046) 반출 시점에만 돈다. 승급 시 추상화하는 경로가 없다.

**영향**: 승급 시점 추상화가 없다.

**자리**: 밖. 직급은 조직 어휘다

**출처**: A15

---

### F7. 회사 식별자 보존

**upstream 규정**: `agents/workflows/text-drop-intake.md:119`이 회사 식별자(실명, 조직명)는 llm-wiki 내부에서 보존한다. 익명화는 블로그 export 시점의 일이다.

**engram 대응**: ADR 0046. `export --replacements` 사용자 사전이 본문과 프론트매터와 파일명에 적용된다. 위키 안에서는 치환하지 않는다.

**영향**: 구현됨(ADR 0046)

**자리**: 밖. 회사 식별자의 취급은 조직마다 다르다

**출처**: A24

---

### F8. 음성 원본 커밋 금지

**upstream 규정**: `agents/workflows/voice-memo-intake.md:56`, `:124`가 Do not commit raw audio. 원시 오디오를 Git에 복사하지 않는다.

**engram 대응**: 비문서 바이너리를 다루는 개념이 없다. `walk`가 마크다운만 훑는다. `doctor`의 점검 항목에도 없다.

**영향**: 바이너리 파일이 혼입될 수 있다.

**자리**: 밖. 저장소에 무엇을 커밋하는지는 사용자의 운용이다

**출처**: A26

---

### F9. 승급 후 원본 이동 차이

**upstream 규정**: `agents/workflows/voice-memo-intake.md:88`이 promote 완료 후 원본 candidate를 `inbox/voice-memos/`에서 `archive/voice-memos/`로 `git mv` 한다. `status: promoted`는 유지한다. 단 inbox 회의록 초안은 원문 보존 차원에서 inbox에 둔다.

**engram 대응**: ADR 0022로 `promote`가 inbox 문서를 이동시켜 원본이 남지 않으므로 옮길 대상이 애초에 없다. `engram archive`는 도움말대로 context 단계 문서만 받고 `status`를 `archived`로 바꾼다. upstream처럼 `promoted`를 유지하는 보관이 없다.

**영향**: 승급 후 원본 취급이 다르다.

**자리**: 밖. 워크플로별 원본 취급은 조직 운용이다

**출처**: A30

---

### F10. 소스 준비 스크립트 거부

**upstream 규정**: `agents/workflows/voice-memo-intake.md:172`가 소스 준비 스크립트가 `inbox/voice-memos/` 밖 candidate를 거절하고, inbox가 아니거나 색인 대상이거나 `voice-memo-intake` workflow가 아닌 후보를 거절한다.

**engram 대응**: `promote`가 대상 문서의 `workflow`나 `source_channel` 값으로 승급 자격을 제한하지 않는다. 위치는 `location.stage-agreement`가 보지만 워크플로 소속은 보지 않는다.

**영향**: 워크플로별 승급 자격 제한이 없다.

**자리**: 밖. 어떤 워크플로가 있는지가 조직마다 다르다

**출처**: A32

---

### F11. 중복 감지 부재

**upstream 규정**: `agents/workflows/text-drop-intake.md:118`이 원본이 이미 `sources/`에 있는 중복이면 이동 대신 사용자에게 알리고 중단한다.

**engram 대응**: 파일명 중복만 거절한다(`internal/cli/promote.go:280`). 내용 기반 중복 감지가 없다.

**영향**: 내용 기반 중복이 검출되지 않는다.

**자리**: 코드. 내용 동일성 판정은 결정론적이다

**출처**: A23

---

### F12. 이슈 분류 부재

**upstream 규정**: `agents/workflows/arb-review.md:56`가 이슈를 CRITICAL/HIGH/MEDIUM/LOW 넷으로 분류한다. CRITICAL은 블로커다.

**engram 대응**: lint의 등급은 reject/error/warn 셋이다(`internal/lint/lint.go:103-134`). 대상이 리뷰 의견이 아니라 스키마 위반이라 성격이 다르다.

**영향**: 리뷰 의견 분류 체계가 없다.

**자리**: 밖. 리뷰 의견의 등급은 조직 운용이다

**출처**: A35

---

### F13. 제한/민감 페이지 읽기 제한 부재

**upstream 규정**: `agents/workflows/arb-review.md:62-64`가 제한/민감 페이지 읽기는 명시된 페이지만. 검토 결과의 외부 공유는 명시적 승인 후. 보안 판정은 최종 사람이 확정한다.

**engram 대응**: 외부로 나가는 경로에서만 대응이 있다. `internal/expose/expose.go:29`의 `HiddenSensitivities = {private-local-only, restricted}`가 `serve`와 `export`에서 걸린다. 읽기 경계와 공유 승인 절차는 없다.

**영향**: 읽기 경계 제어가 없다.

**자리**: 밖. 공유 승인과 보안 판정 확정은 사람의 절차다

**출처**: A36

---

### F14. 주간 뉴스 누적 부재

**upstream 규정**: `agents/workflows/weekly-news-intake.md:50`이 후보를 `inbox/weekly-news/YYYY-WW.md`에 모은다. 주차 단위 파일 하나에 누적한다.

**engram 대응**: `capture`의 파일명은 `<날짜>-<슬러그>.md` 고정이고(`engram capture --help`) 하위 디렉토리를 못 만든다. 주차 파일에 append하는 경로도 없다. `update --body-from`은 통째 교체다.

**영향**: 주간 누적 파일을 만들 수 없다.

**자리**: 밖. 주차 누적은 조직의 절차다

**출처**: A37

---

### F15. 발행 이력 기록 부재

**upstream 규정**: `agents/workflows/blog-publish.md:68`, `:92`가 발행 이벤트는 `log.md`에 자동 기록한다.

**engram 대응**: A6과 같다. 없다.

**출향**: 발행 이력 추적이 없다.

**자리**: 밖. `export`는 위키를 고치지 않는 것이 계약이라 이력을 쓰면 성격이 바뀐다

**출처**: A40

---

### F16. 에이전트 파일 경로 참조 차이

**upstream 규정**: `agents/README.md:13`이 Agent files should reference `context/` and `sources/` by path rather than embedding large copied context라고 규정한다.

**engram 대응**: ADR 0062로 `internal/skills/SKILL.md`가 "위키 내용은 recall로 꺼낸다. 파일을 직접 읽지 않는다"고 적는다. 경로 참조가 아니라 슬러그와 줄 범위와 원문 조각을 받는 방식이다.

**영향**: 에이전트 파일 참조 방식이 다르다.

**자리**: 밖. ADR 0062가 이미 다른 방식으로 같은 목적을 달성했다

**출처**: A41

---

### F17. 반출 시 링크 변환 부재

**upstream 규정**: `agents/references/blog/transform-rules.md:101-107`이 반출 시 기계적 변환을 규정한다. `[[링크]]`는 `링크`로, `[[원본|표시]]`는 `표시`로, `## Related`/`## References`/`## 관련 문서`/`## 참고` 섹션 전체 제거, `#tag` 인라인 태그 제거, 빈 줄 3줄 이상은 2줄로 축소한다.

**engram 대응**: `internal/export/export.go:69-72` 주석대로 "본문을 고치지 않으므로 링크는 그대로 나간다. 수만 알린다"(ADR 0046). 위키링크도 관련 섹션도 그대로 나간다.

**영향**: 반출 시 링크가 변환되지 않는다.

**자리**: 코드. upstream이 기계적 변환이라 못 박은 것이다

**출처**: A42

---

### F18. 승급 자격 여섯 가지

**upstream 규정**: `agents/workflows/context-node-add.md:41`이 승급 자격은 여섯이다. clear title, source reference, date/validity, **owner**, 결론/재사용 규칙, related 링크다.

**engram 대응**: 게이트의 거절 사유는 `gate.min-wikilinks` 하나뿐이다(`internal/i18n/catalog_engine.go:17`, ADR 0040/0054). `owner`는 속성 14종에 없고 코드 어디에도 없다.

**영향**: 승급 자격 기준이 다르다.

**자리**: 설정. 여섯 중 다섯은 판단이고 남는 것은 `owner` 축 하나다

**출처**: A1

---

### F19. blockquote 핵심 명제 부재

**upstream 규정**: `agents/workflows/context-node-add.md:43`이 본문 첫 줄에 `> ` 한 줄로 노드 핵심 명제를 쓴다. `sync_wiki_index.py`가 이 줄을 index.md 설명로 자동 추출하므로 이 blockquote가 index 설명의 원천이 된다.

**engram 대응**: lint 규칙 14종에 본문 구조를 보는 것이 없다. `capture`는 첫 줄을 제목으로 쓰고(`internal/cli/capture.go:189`) blockquote를 요구하지 않는다.

**영향**: index 설명 추출이 없다.

**자리**: 데모. 본문 첫 줄의 형태다

**출처**: A3

---

### F20. type별 위치 고정 부재

**upstream 규정**: `agents/workflows/context-node-add.md:44`가 type별 위치를 고정한다. decision은 `context/decisions/`, procedure는 `context/procedures/`, system은 `context/systems/`, concept은 `context/concepts/`, project은 `context/projects/`, moc는 `context/mocs/`다.

**engram 대응**: `docs/spec-map.md:83`이 단계 디렉토리 아래를 사용자의 자유로 둔다고 적는다. `promote`는 `context/<슬러그>.md`에 평면으로 놓는다. `moc` type은 `internal/config/config.go:241`의 types 11종에 없다.

**영향**: 타입별 위치 구분이 없다.

**자리**: 데모. 하위 디렉토리는 사용자 자유이므로 본보기만 보여 준다

**출처**: A4

---

### F21. inbox lint 범위 차이

**upstream 규정**: `scripts/lint-frontmatter.sh:9-19`, `AGENTS.md:114-123`이 `inbox/`는 기본 검사 범위 밖이고 pre-commit 게이트 밖이다. "붙여 넣는 시점에 스키마를 요구하면 capture-first 와 충돌한다. inbox lint 실패는 정상이며 버그가 아니다."

**engram 대응**: engram은 `inbox`를 `page_dirs`에 넣고(`internal/config/config.go:220`) 필수 필드를 요구한다. `docs/spec-map.md:302`가 프론트매터 없는 `inbox/` 파일이 `frontmatter.missing` error를 낸다고 실측으로 적었다.

**영향**: `capture`를 거치지 않는 유입 경로가 전부 lint 실패로 나타난다.

**자리**: 코드

**출처**: B22, E3

---

### F22. 폐기 필드 거부 부재

**upstream 규정**: `scripts/lint-frontmatter.sh:195-199`, `AGENTS.md:132`가 `quality_level`이 남아 있으면 FAIL. "폐기된 축이 다시 기어들어오지 못하게."

**engram 대응**: engram에 `quality_level`이라는 개념 자체가 없다(`rg -rn 'quality_level' internal/ docs/` 무결과).

**영향**: 폐기 필드가 들어와도 막지 못한다.

**자리**: 설정. 폐기 필드 목록은 위키마다 다르다

**출처**: B23, E4

---

### F23. `review_after` 폐기 거부 부재

**upstream 규정**: `scripts/lint-frontmatter.sh:201-206`, `AGENTS.md:185-187`이 `review_after`가 남아 있으면 FAIL. 근거: "예측이라 지킬 수 없고, 지나도 아무 일이 일어나지 않는다는 것이 실측으로 확인됐다."

**engram 대응**: engram에 `review_after` 개념이 없다(`rg -rn 'review_after' internal/ docs/` 무결과).

**영향**: 재도입 차단이 없다.

**자리**: 설정. 폐기 필드 목록은 위키마다 다르다

**출처**: B24, E5

---

### F24. `title` 필드 부재

**upstream 규정**: `scripts/lint-frontmatter.sh:228-234`가 `artifact_stage: context` 문서는 `title` 필드를 요구한다. 없으면 `index.md`가 영문 slug로만 보인다.

**engram 대응**: 대응 없음. `rg -n 'title' internal/lint/lint.go` 무결과. engram은 제목을 본문 첫 머리글에서 읽는다(`internal/resurface/resurface.go:242-250`).

**영향**: 제목 필드가 없다.

**자리**: 밖. engram은 제목을 본문 머리글에서 읽으므로 upstream이 든 이유가 성립하지 않는다

**출처**: B27, E2

---

### F25. 색인 자격 삼단 검사 부재

**upstream 규정**: `scripts/lint-frontmatter.sh:271-282`가 색인 자격 삼단 검사를 규정한다. `inbox` + `indexable != false`는 FAIL, `source` + `indexable != false`는 WARN, `context` + `indexable != true`는 WARN다.

**engram 대응**: 대응 없음. engram lint 규칙 17종에 `indexable` 값 검사가 없다(B19 참조).

**영향**: 색인 자격 검사가 없다.

**자리**: 코드

**출처**: B29

---

### F26. 단계별 provenance 차이

**upstream 규정**: `scripts/lint-frontmatter.sh:284-301`이 단계별 provenance를 규정한다. `context`/`agent-workflow`/`index`는 `source_refs` + `related` 필수, `source`는 `derived_context` 필수다.

**engram 대응**: `internal/lint/lint.go:349-363`. `source`는 `source_refs derived_from derived_context`, `context`는 `source_refs derived_from related`다. upstream보다 `derived_from`만큼 더 요구한다. 둘 다 존재만 보고 값이 비었는지는 안 본다.

**영향**: provenance 요구사항이 다르다.

**자리**: 코드

**출처**: B30

---

### F27. 검사 대상 디렉토리 차이

**upstream 규정**: `scripts/lint-frontmatter.sh:117-124`가 검사 대상 디렉토리 여섷을 규정한다. `context agents/workflows sources/manifests sources/transcripts sources/summaries meta/templates` + `index.md`다. `README.md`는 제외다.

**engram 대응**: `internal/config/config.go:220-222`이 `page_dirs: [inbox sources context archive]`, `root_files: [index.md]`, `ignore_files: [README.md]`다. `meta/templates` 개념이 없고 `agents/workflows`가 없다.

**영향**: 검사 대상 범위가 다르다.

**자리**: 설정. `page_dirs`와 `ignore_files`가 이미 그 자리다

**출처**: B31

---

### F28. `updated` 갱신 대상 차이

**upstream 규정**: `scripts/sync_updated_field.py:29-30`이 `updated` 갱신 대상은 `context`와 `agents/workflows`뿐이다. "sources/는 원본 보존이라 건드리지 않는다."

**engram 대응**: `internal/cli/sync.go:108`이 `stage != wiki.StageSource`로 sources만 뺀다. `inbox`와 `archive`에도 `updated`를 쓴다.

**영향**: `updated` 갱신 대상이 다르다.

**자리**: 코드

**출처**: B32

---

### F29. 벌크 커밋 날짜 비움 부재 (닫힘, ADR 0066)

**upstream 규정**: `scripts/sync_updated_field.py:110-111`이 벌크 커밋으로만 등장한 파일은 날짜를 비워 두고 건너뛴다. "다음 실제 편집 때 채워진다."

**engram 대응**: 대응 없음(B3의 결과).

**영향**: 벌크 커밋 파일 날짜가 비워있지 않는다.

**자리**: 코드

**출처**: B33

---

### F30. `created`/`sourced_at` 백필 대상 차이

**upstream 규정**: `scripts/backfill_source_dates.py:30`이 `created`/`sourced_at` 백필 대상은 `sources/summaries manifests transcripts`뿐이다.

**engram 대응**: `internal/cli/sync.go:113-114`가 `sourced_at`을 단계 구분 없이 모든 문서에 쓴다. `context` 문서에도 `sourced_at`이 들어간다. upstream context 문서는 이 필드를 갖지 않는다.

**영향**: 백필 대상이 다르다.

**자리**: 코드

**출처**: B34

---

### F31. 별칭 흡수 부재

**upstream 규정**: `scripts/backfill_source_dates.py:36-44`가 `created`를 다섯 별칭에서 흡수한다. `meeting_date recorded_at published_at page_created_at captured_at`다. "워크플로마다 이름이 갈라져 있어 여기서 흡수한다."

**engram 대응**: 대응 없음. `rg -rn 'meeting_date\|recorded_at\|captured_at' internal/` 무결과. ADR 0037은 `created`를 "건드리지 않는다"로 못박았다.

**영향**: 별칭 흡수가 없다.

**자리**: 밖. 별칭 목록이 조직의 워크플로마다 갈린다

**출처**: B35

---

### F32. 파일명 날짜 추론 부재

**upstream 규정**: `scripts/backfill_source_dates.py:32-34, 74-81`이 `created` 재료는 파일명 날짜 접두사다. `YYYY-MM-DD-slug`와 `YYYY-MM-slug` 둘 다 받는다. "연월까지만 아는 자료가 54개 있어."

**engram 대응**: engram은 `created` 형식으로 `2006-01-02`와 `2006-01`을 둘 다 허용한다(`internal/config/config.go:251`). 그러나 파일명에서 날짜를 뽑는 코드는 없다.

**영향**: 파일명에서 날짜 추론이 없다.

**자리**: 코드. engram이 파일명 규칙을 강제하므로 추출이 결정론적이다

**출처**: B36

---

### F33. 미커밋 파일 날짜 채움 부재

**upstream 규정**: `scripts/backfill_source_dates.py:124-127`이 아직 커밋되지 않은 파일은 `sourced_at`을 오늘 날짜로 채운다.

**engram 대응**: `internal/cli/sync.go` 계열은 커밋 없는 파일을 건너뛴다. ADR 0037이 "커밋되지 않은 파일은 건너뛰고 개수를 알린다"로 명시했다.

**영향**: 미커밋 파일 날짜가 비어 있다.

**자리**: 코드

**출처**: B37

---

### F34. 색인 자격 네 조건

**upstream 규정**: `scripts/list-index-candidates.sh:10-15, 68-76`이 색인 자격 네 조건 동시 충족을 규정한다. 위치가 `context/`/`agents/workflows/`/`index.md`, `artifact_stage in (context, agent-workflow, index)`, **`status == promoted`**, **`indexable == true`**, `sensitivity in (public-reference, internal)`이다.

**engram 대응**: `internal/expose/expose.go`는 위치(`context` + `root_files`)와 `sensitivity` 차단(`private-local-only`, `restricted`)만 본다. `status`와 `indexable`을 읽지 않는다. ADR 0044에 두 필드 언급이 없다.

**영향**: 색인 자격 판단이 다르다.

**자리**: 코드

**출처**: B38

---

### F35. 미러 포함 민감도 차이

**upstream 규정**: `scripts/mirror-to-icloud.sh:22-32`가 미러에서 제외하는 것은 `sensitivity: private-local-only` 하나뿐이고 `restricted`는 미러한다. "iCloud는 개인 계정 범위이고 리포는 이미 GitHub private."

**engram 대응**: `internal/expose/expose.go:29` `HiddenSensitivities = ["private-local-only", "restricted"]`로 둘 다 막는다. 목적지가 다르므로(웹 뷰어 대 개인 iCloud) 직접 대응은 아니다.

**영향**: 미러 민감도 포함 범위가 다르다.

**자리**: 밖. 미러는 engram 범위 밖이다

**출처**: B39

---

### F36. git hook 설치 부재

**upstream 규정**: `scripts/install-git-hooks.sh:16-17`이 `git config core.hooksPath .githooks`로 pre-commit 게이트를 켠다. clone 후 한 번 실행이 필요하다.

**engram 대응**: engram에 훅 설치 커맨드가 없다. `engram --help` 스물여덟에 없다. `eject`가 Python 린터를 내보내나(ADR 0039) 훅 등록은 하지 않는다.

**영향**: pre-commit 게이트 설정이 없다.

**자리**: 코드. 훅 등록은 사용자 운용이나 `eject`가 이미 린터를 내보내므로 같은 커맨드의 일이다

**출처**: B41

---

### F37. source 증거 선택 색인 부재

**upstream 규정**: `context/systems/indexing-config.md:108-128`이 source 증거의 선택 색인을 규정한다. 켜면 `artifact_stage: source` + `status: sourced` + `type: source-summary` + `indexable: true` 넷을 요구하고, raw transcript와 운영 메타데이터뿐인 manifest와 `source_refs` 없는 source는 그래도 뺀다. 기본 결정은 시스템 동작이 검증될 때까지 source 증거를 주 답변 코퍼스 밖에 두는 것이다.

**engram 대응**: `internal/expose/expose.go` `Exclude`가 `LocSources`를 무조건 제외한다. `IncludeArchive`는 있으나 `IncludeSources`에 해당하는 옵션이 없다. 기본 동작은 같고 여는 경로가 없다.

**영향**: source 증거를 색인에 포함할 수 없다.

**자리**: 코드

**출처**: C15

---

### F38. `agents/workflows/` 전체 부재

**upstream 규정**: `docs/spec-map.md:283`이 "이 절이 뒤늦게 생긴 경위를 적어 둔다"고 적는다. 대응표가 `meta/` 명세만 덮어서 아예 보지 않았다. 절차 명세는 규칙이 아니라 에이전트에게 주는 순서라 성격이 다르다고 보았는데, 그 안에 engram이 지켜야 할 계약이 섞여 있었다. 가장 분명한 것이 텍스트 드롭 인테이크다.

**engram 대응**: 없다.

**영향**: 에이전트 절차 명세가 구현되지 않았다.

**자리**: 밖. 조직의 절차 명세다

**출처**: C6절

---

### F39. 템플릿 부재

**upstream 규정**: `context/systems/obsidian-operating-layer.md:89-97`이 Obsidian이 쓸모 있으려면 문서에 안정된 제목, 위키링크, 프론트매터, 태그, MOC/index 페이지, **문서 타입별 템플릿**, 관계 필드가 있어야 한다고 규정한다. "This is not decoration."

**engram 대응**: 여섯은 있다. 템플릿이 없다. `new`, `capture`, `source`가 고정 골격을 쓸 뿐 사용자가 타입별 서식을 정의하는 자리가 없다. ADR 0029:28이 upstream `meta/templates/`를 보고 "서식"이라 분류해 계약 목록 밖에 두었다.

**영향**: 문서 타입별 템플릿이 없다.

**자리**: 데모. 설정 키로 열 여지가 있으나 서식은 조직 취향이라 본보기로 둔다

**출처**: C27

---

### F40. 부분 supersede 표기 차이

**upstream 규정**: upstream `context/decisions/`의 게이트웨이 운용 결정 문서(파일명은 사내 시스템 이름을 담아 옮기지 않는다. 원본은 `private/upstream-audit-2026-08-18/C-decisions.md` 참조)가 **부분 supersede를 조항 단위로 표기**한다고 규정한다. 대체된 문서 본문 상단에 "부분 supersede (2026-08-07): mobile-inbox 조항은 [[...]]로 대체되었다. ... 조항은 그대로 유효하다"를 넣고, 대체한 문서에도 `## Supersedes` 절을 둔다.

**engram 대응**: ADR 0015가 같은 개념을 다르게 실현했다. `amended` 상태가 "결론은 유효하나 일부 절이 후속 ADR로 대체되었다"(0015:23)이고, 어느 절이 대체됐는지는 `docs/decisions/README.md` 개정 그래프에 둔다. 본문 소급 수정을 금지하므로 upstream처럼 문서 안에 표기하지 않는다(0015:14).

**영향**: supersede 표기 방식이 다르다.

**자리**: 밖. 본문의 표기 방식이고 ADR 0015가 이미 다르게 실현했다

**출처**: C34

---

### F41. 프론트매터 우선 vs 경로 우선

**upstream 규정**: `context/systems/llm-wiki-architecture.md:97`가 "Selection should use **frontmatter first, path second**. A document should be indexed only when its metadata **agrees with** the path-level policy." 즉 둘 다 동의해야 색인한다고 규정한다.

**engram 대응**: ADR 0040이 정반대를 정했다. "게이트는 문서가 놓인 디렉토리로 발동한다. `artifact_stage` 선언을 보지 않는다." 두 값의 불일치는 `location.stage-agreement` 경고로만 보고되고(ADR 0031, 0035) 노출 여부를 바꾸지 않는다.

**영향**: 색인/노출 판단 기준이 다르다.

**자리**: 코드

**출처**: C22

---

### F42. corpus 크기 문턱 차이

**upstream 규정**: `context/decisions/wiki-resurface-loop.md:140-143`이 재검토 조건을 규정한다. 시맨틱 검색을 저장소 전체로 확장해야 하면 이 스크립트를 키우지 말고 검색 스택으로 옮긴다. `context/` 문서가 **300개를 넘으면** 쌍 비교가 제곱으로 늘어나 후보 사전 축소를 검토한다.

**engram 대응**: `internal/bridge/bridge.go:48-49` ponytail 주석이 같은 문제를 다르게 잡았다. "O(n^2) 쌍 비교. 문서 수 **2000** 규모까지는 감당한다. 그보다 커지면 토큰 역색인으로 교집합 쌍만 비교하게 바꾼다."

**영향**: corpus 크기 문턱이 다르다.

**자리**: 코드

**출처**: C8

---

### F43. AI 산출물 보존 등급 부재

**upstream 규정**: `automation-ai-output-review.md:43-51`가 자동화/AI 산출물을 보존 등급 다섴(`deliver-only`, `retained-source`, `promotion-proposal`, `workflow-rule`, `external-write`)으로 분류하고 등급마다 기본 목적지, 커밋 여부, 색인 노출을 정한다.

**engram 대응**: `rg -ni 'retention|deliver-only|retained-source' --type go internal/` 결과 0건. `types` 집합(`internal/config/config.go:241-243`)에 `agent-workflow`, `source-summary`가 있어 일부 목적지는 표현되지만 보존 등급이라는 축 자체가 없다.

**영향**: AI 산출물 보존 등급 구분이 없다.

**자리**: 밖. 보존 등급의 어휘가 조직마다 다르다

**출처**: D9

---

### F44. 산출물별 기본 보존 부재

**upstream 규정**: `web-news-intake.md:135-144`가 산출물별 기본 보존 표를 규정한다. 일간 브리핑은 보존하지 않고, 주간 다이제스트는 source summary로 보존하고, 기사 전문은 기본 미보존이다.

**engram 대응**: D9와 동일. engram에는 "보존하지 않는다"를 표현할 자리가 없다. `capture`는 넣기만 하고 지우는 커맨드가 없다(`engram --help` 28개 커맨드에 delete/purge 없음).

**영향**: 보존 등급 구분이 없다.

**자리**: 밖. 산출물별 보존 여부는 조직 운용이다

**출처**: D10

---

### F45. AI 산출물 승급 조건 부재

**upstream 규정**: `automation-ai-output-review.md:161-169`가 AI 산출물은 잘 쓰였다는 이유로 승급하지 않는다. 출처 근거, 재사용성, 스코프/민감도 분류, **기존 context와 중복 아님**, 증거 또는 사람의 결정에 연결됨 다섯을 모두 만족할 때만 승급한다.

**engram 대응**: `promote`/`new`가 돌리는 게이트는 `gate.min-wikilinks` 하나다. 중복 검사는 없다(`rg -ni 'duplicate|중복' internal/cli/promote.go` 결과는 목록 필드 병합 주석뿐). `bridge`가 색인 TF 벡터의 코사인 유사도로 비슷한데 링크 없는 쌍을 내지만 승급 경로와 이어져 있지 않다.

**영향**: AI 산출물 승급 조건 검사가 없다.

**자리**: 밖. 다섯 중 넷이 판단이다

**출처**: D11

---

### F46. 반출 적격성 삼분 차이

**upstream 규정**: `notebooklm-export-workflow.md:89-109`가 반출 적격성 삼분을 규정한다. 기본 허용은 `public-reference`/`internal`의 context 문서와 선택된 `sources/summaries`, `sources/manifests`다. `restricted`는 `--allow-restricted`가 있어야 한다. 절대 금지는 `private-local-only`, 원본 오디오, `.local/`, `inbox/`다.

**engram 대응**: `engram export`는 `restricted`와 `private-local-only`를 둘 다 무조건 제외하고 우회 플래그가 없다(`export --help`: "슬러그로 지목해도 이 제외는 뚫리지 않습니다"). `sources`도 통째로 제외라 증거를 함께 내보낼 방법이 없다.

**영향**: 반출 제어가 다르다.

**자리**: 코드

**출처**: D12

---

### F47. 반출 팩 구조 부재

**upstream 규정**: `notebooklm-export-workflow.md:47-67`가 반출 팩은 다섯 파일 형태를 갖고 그중 `manifest.md`가 선택 파일 목록, 민감도, 생성 시각, 취급 주의를 기록한다.

**engram 대응**: `engram export`는 마크다운 파일을 그대로 복사할 뿐이다. `rg -n 'manifest' internal/export/*.go internal/cli/export.go` 결과 0건.

**영향**: 반출 팩 구조가 없다.

**자리**: 코드

**출처**: D13

---

### F48. 원본 증거 불변 경계 위반 (닫힘, ADR 0064)

**upstream 규정**: `voice-memo-terminology-normalization.md:37-50`이 원본 증거(`.local/stt-runs/*`, 원본 오디오)는 고치지 않는다고 규정한다. 정규화 대상은 inbox 초안, 승인된 `sources/summaries/*`, `sources/manifests/*`의 제목과 취급 주석, 승급된 `context/*`다.

**engram 대응**: `sources` 계층 불변은 두 군데서 부분 강제된다. `source --help`가 "이 계층은 원본 보존이 계약이므로 문서를 고치지 않고 updated 필드도 쓰지 않습니다"라 밝히고, lint에 `sources.updated` 경고가 있다(`internal/lint/lint.go:119-120`). 다만 `update` 커맨드는 `sources` 문서의 본문과 프론트매터를 아무 제지 없이 고친다.

**영향**: `update`로 sources 문서를 고칠 수 있다.

**자리**: 코드

**출처**: D15

---

### F49. 음성 메모 후보 등록 승인

**upstream 규정**: `voice-memo-candidate-registration.md:96-100`이 등록된 inbox 후보는 `scripts/lint-frontmatter.sh --include-inbox`를 통과해야 한다고 규정한다.

**engram 대응**: engram lint는 inbox 문서에도 필수 필드 검사를 항상 적용한다(`internal/lint/lint.go:342-344`, 기본 필수 `type`, `artifact_stage`, `status`, `indexable`). 켜고 끄는 플래그가 없다(`lint --help`에 `--wiki`뿐).

**영향**: inbox 문서도 lint를 통과해야 한다.

**자리**: 코드

**출처**: D16

---

### F50. Teams 전사 메타데이터 부재

**upstream 규정**: `slack-teams-meeting-intake.md:156-162`가 Teams 전사가 Confluence를 경유할 때 `source_platform`, `source_artifact_type`, `transfer_method`를 프론트매터에 남긴다고 규정한다.

**engram 대응**: 이 세 키는 engram 속성 14종에 없다(`internal/cli/update.go:165-172`). `update --set`으로 임의 키를 넣는 것은 막히지 않지만(`axisOff` 주석: "속성 이름이 아닌 키는 여기서 가리지 않는다") lint가 검증하지 않고 `source` 커맨드에 해당 플래그도 없다.

**영향**: Teams 메타데이터 필드가 없다.

**자리**: 밖. 특정 도구와의 연동 절차다

**출처**: D18

---

### F51. Slack/Teams 최소 출처 부재

**upstream 규정**: `slack-teams-meeting-intake.md:180-191`가 모든 Slack/Teams 회의 아티팩트는 최소 출처 여덟(플랫폼, 아티팩트 종류, 회의 제목, 일시, 채널/회의 맥락, permalink, 전달 방법, 민감도 분류)를 기록한다고 규정한다.

**engram 대응**: `source` 커맨드가 받는 것은 `--channel`, `--created`, `--ref`, `--title`, `--type`, `--slug` 여섯이다(`source --help`). 플랫폼, 아티팩트 종류, 전달 방법에 해당하는 자리가 없다.

**영향**: 최소 출처 정보가 부족하다.

**자리**: 밖. 특정 도구와의 연동 절차다

**출처**: D19

---

### F52. Confluence 페이지 전체 복제 금지

**upstream 규정**: `atlassian-mcp-intake.md:129`, `:173`이 Confluence 페이지 전체를 `context/`로 복제하지 않는다고 규정한다. 승급된 노드는 지역적이고 지속되는 해석이어야 하고 원본은 증거로 남는다. Jira의 원문 설명과 긴 코멘트 스레드도 같다.

**engram 대응**: 승급 시 본문 길이나 원본 대비 파생 여부를 보는 판정이 없다. 근사치로 `body.max-lines` 경고가 있다(기본 1000줄, `internal/config/config.go:219`).

**영향**: 긴 본문 복제를 막는 판정이 없다.

**자리**: 교재. 특정 도구 연동이나 승급이 복제가 아니라는 것은 가르칠 몫이다

**출처**: D20

---

### F53. 시효 자료 필드 부재

**upstream 규정**: `inbox-processing-matrix.md:204`가 시효가 있는 자료는 유효 기간(validity window)이나 재검토 날짜를 표시한다고 규정한다. `atlassian-mcp-intake.md:196`도 같다.

**engram 대응**: 그런 필드가 없다. `rg -ni 'review_by|valid_until|validity|freshness|review_date' --type go internal/ cmd/`는 색인 신선도 주석만 잡는다. lint 규칙 17종에도 없다.

**영향**: 시효 자료 추적이 없다.

**자리**: 밖. upstream 안에서 규정이 엇갈리고 폐기 쪽이 실측을 근거로 든다

**출처**: D6, D7

---

### F54. 승급 요구사항 차이

**upstream 규정**: `legacy-obsidian-vault-migration.md:87-93`이 승급 요구사항 다섯을 규정한다. 명확한 제목과 현재 해석, 원자료 매니페스트 참조, 민감도 분류, 기술 내용이 낙을 수 있으면 신선도/재검토 창, 관련 context 링크다.

**engram 대응**: 다섯 중 넷은 대응이 있다(제목, `source_refs`, `sensitivity`, `gate.min-wikilinks`). 신선도/재검토 창만 없다.

**영향**: 신선도/재검토 추적이 없다.

**자리**: 밖. 빠진 하나가 F53과 같은 시효 필드다

**출처**: D8

---

### F55. 옛 노트 서식 보존 금지

**upstream 규정**: `legacy-obsidian-vault-migration.md:85`가 옛 노트를 승급할 때 과거 서식을 보존하지 말고 현재 위키 형태로 다시 쓴다고 규정한다.

**engram 대응**: `migrate`가 기존 문서를 지금의 설정과 규칙에 맞춘다(ADR 0038). 다만 대상은 프론트매터와 위치이고 본문 형태는 아니다.

**영향**: 본문 서식이 보존될 수 있다.

**자리**: 교재. 옛 노트를 다시 쓰는 일은 사람의 판단이다

**출처**: D24

---

### F56. 공개 문맥 통과 차이 (닫힘, ADR 0069)

**upstream 규정**: `blog-publishing-pipeline.md:149-159`가 공개 반출 전 공개 문맥 통과 다섯 항목을 검사한다. 회사명, 내부 URL, 내부 프로젝트명, 계정 식별자, 토큰, 사설 경로를 제거하고 내부 운영 맥락을 공개 개념으로 다시 쓴다.

**engram 대응**: `export --replacements`가 사용자 사전으로 문자열 치환을 한다(ADR 0046). 파일을 주지 않으면 치환하지 않는다. 검사(무엇이 남았는지 판정)는 없고 치환(무엇을 바꿀지 사람이 지정)만 있다.

**영향**: 공개 문맥 검사가 없다.

**자리**: 코드. 회사명은 사전이 필요하나 토큰과 사설 경로와 내부 URL은 패턴으로 잡힌다

**출처**: D26

---

### F57. 중복 갱신 vs 노드 생성

**upstream 규정**: `blog-publishing-pipeline.md:147`이 여러 자료가 같은 판단을 가리키면 새 context 노드를 만들지 말고 기존 노드를 갱신하고 `source_refs` 또는 `derived_from`에 새 출처를 더한다고 규정한다.

**engram 대응**: `promote`가 목록 필드를 중복 없이 병합하는 `mergeListField`를 갖고 있다(`internal/cli/promote.go:552-554`). 다만 "기존 노드가 있는지" 판정은 사람 몫이고 커맨드가 후보를 제시하지 않는다.

**영향**: 중복 감지가 사람에게 달려 있다.

**자리**: 코드. 판단은 사람이 하나 후보를 내는 것은 결정론적이다

**출처**: D27

---

### F58. 음성 아티팩트 분류 차이

**upstream 규정**: `voice-memo-stt-strategy.md:259-271`, `voice-memo-intake-trigger.md:71-81`이 승인된 텍스트 아티팩트를 셋으로 가른다. 구조화된 회의록은 `sources/summaries/`, 출처와 취급 메타데이터는 `sources/manifests/`, 전체 전사본은 명시 요청 시에만 `sources/transcripts/`다. 회의록과 매니페스트가 기본 증거이고 전사본은 선택이다.

**engram 대응**: engram은 `sources/`를 나누지 않고 `type`으로 가른다. `source-raw`가 원문, `source-summary`가 정제본이다(ADR 0051). 매니페스트에 해당하는 `type` 값이 없다. `types` 열한 개(`internal/config/config.go:241-243`)에 `manifest`가 없다.

**영향**: 아티팩트 분류가 다르다.

**자리**: 설정. `types` 집합이 이미 그 자리다

**출처**: D28

---

### F59. 토큰과 API 키 승급 금지 (닫힘, ADR 0069)

**upstream 규정**: `claude-code-operating-practices.md:53`이 토큰, API 키, 페어링 코드, 개인 설정값은 위키 context로 승급하지 않는다고 규정한다.

**engram 대응**: `docs/spec-map.md:149`가 `security-rules.md`에 대해 "코드로 강제하는 것. **없다**"고 이미 적었다. 시크릿 스캔은 위키가 아니라 이 저장소의 커밋 훅에 있다.

**영향**: 시크릿 스큔이 없다.

**자리**: 코드

**출처**: D29

---

### F60. 음성 메모 후보 민감도

**upstream 규정**: `voice-memo-candidate-registration.md:43`이 음성 메모 후보의 민감도는 `private-local-only`로 표시한다(기본 최고 등급으로 시작)라고 규정한다.

**engram 대응**: `private-local-only`는 engram의 `sensitivities` 허용값에 있다(`internal/config/config.go:248`). 다만 캡처 시점 기본값을 최고 등급으로 두는 규칙은 없고 `capture`가 민감도를 채우지 않는다.

**영향**: 캡처 시점 민감도 기본값이 없다.

**자리**: 설정. 캡처 시점 기본 민감도는 위키마다 다르게 정할 값이다

**출처**: D30

---

### F61. 파일명 한글 slug 고정 차이

**upstream 규정**: `AGENTS.md:165-167`이 **파일명은 영어 slug로 고정한다**고 규정한다. `context/`는 링크 대상이라 이름이 바뀌면 본문 위키링크, `meta/resurface-state.json`의 쿨다운 키, git 이력 추적이 함께 깨진다.

**engram 대응**: `new`가 `engram new "테스트 결정"`에 `context/테스트-결정.md`를 만든다. ADR 0020이 "한글은 보존한다. 음차하지 않는다"를 결정으로 못박았고 근거는 "upstream 파일명에 한글이 섞여 있다"인데, 실측하면 `context/` 81개 중 한글 파일명은 **0개**이고 한글은 `sources/` 151개 중 26개에만 있다. ADR이 읽은 것은 `sources/` 관례이고 `context/` 규칙이 아니다.

**영향**: 한글 파일명이 만들어진다.

**자리**: 코드

**출처**: E1

---

### F62. 마크다운 링크 파싱 부재 (닫힘, ADR 0065)

**upstream 규정**: `AGENTS.md:176-178`이 `index.md` 링크는 **표준 마크다운 형식**(`[한글제목]\(상대경로\)`)을 쓴다고 규정한다. 위키링크(`[[slug]]`)는 Obsidian에서만 클릭되고 GitHub 웹이나 편집기에서는 평문으로 보인다.

**engram 대응**: engram은 마크다운 링크를 아예 파싱하지 않는다. 정규식 전수(`grep -rn --include='*.go' MustCompile internal`) 6건 중 링크용은 `internal/doc/link.go:15`의 `\[\[([^[\]]+)\]\]` 하나뿐이다. 실측: upstream `index.md`는 마크다운 링크 166개와 위키링크 2개(둘 다 frontmatter `related`)를 갖는데, 마크다운 링크만 있는 문서를 engram lint에 넣으면 `graph.orphan` 경고가 난다.

**영향**: 마크다운 링크가 위키링크로 인식되지 않는다.

**자리**: 코드

**출처**: E6

---

### F63. 승급 완료 후 형태 차이

**upstream 규정**: `context/README.md:20-32`, `promotion-review-checklist.md:31-34`가 승급 문서 형태를 규정한다. `# Title`, `> One-line judgment`, `## Context`, `## Current Understanding`, `## Evidence`, `## Related`다.

**engram 대응**: `new`는 다섯 절 골격을 넣는다(결론/맥락/현재 이해/근거/관련 링크, `internal/cli/new.go`). `promote`는 넣지 않는다. 실측: `capture` 한 줄짜리 메모를 `promote`하면 본문이 `테스트 메모` 한 줄 그대로이고 H1도 절도 없다. 형태 미강제는 ADR 0040의 의도된 선택이나, 두 커맨드가 다르게 동작하는 것은 그 ADR이 다루지 않는다.

**영향**: `promote`와 `new`가 만드는 형태가 다르다.

**자리**: 데모. G11과 같은 본문 형태다

**출처**: E11

---

### F64. `context/people/` 부재

**upstream 규정**: `context/README.md:9-16`이 context 버킷 여섯을 규정한다. `concepts/ projects/ systems/ decisions/ procedures/ people/`다.

**engram 대응**: engram 기본 types 11종(`internal/config/config.go:242`)에 `people`이 없다. `grep -rn --include='*.go' '"people"' internal cmd` 0건. spec-map 4.9가 이미 "`people/`은 대응이 없다"고 적었다.

**영향**: people 타입이 없다.

**자리**: 설정. `types` 집합이 이미 그 자리다

**출처**: E14

---

### F65. `index.md` 타입/단계 차이

**upstream 규정**: `scripts/lint-frontmatter.sh:224-225`, `README.md:84`(`index.md`가 미러 대상)이 "root index must be type moc" / "root index must use artifact_stage index"라고 규정한다. upstream `index.md` 실물이 `type: moc`, `artifact_stage: index`다.

**engram 대응**: `engram init`이 만드는 `index.md`는 `type: system`, `artifact_stage: context`다(실측). 반대로 upstream 문법의 색인을 engram lint에 넣으면 `schema.allowed-value` **error**와 `location.stage-agreement` warn이 난다(실측). ADR 0019는 색인을 게이트와 고아 판정에서 뺐을 뿐 타입과 단계를 정하지 않는다.

**영향**: `index.md` 형태가 다르다.

**자리**: 코드. `init`이 만드는 값과 단계 허용값은 코드에 고정되어 있다

**출처**: E16

---

### F66. `sources.updated` 등급 차이

**upstream 규정**: `scripts/lint-frontmatter.sh:224-225`, CHANGELOG `2026-08-08`("source artifact must not carry `updated`")를 **FAIL**로 처리한다.

**engram 대응**: engram의 `sources.updated` 규칙은 `warn`이다(`internal/lint/lint.go:121`). 같은 규칙이 등급만 다르다.

**영향**: 등급이 다르다.

**자리**: 코드

**출처**: E17

---

### F67. 계약 파일 vendoring 부재

**upstream 규정**: `AGENTS.md:35-46`이 계약 파일 8종의 규칙을 고치면 같은 커밋에서 `meta/CHANGELOG.md`에 `impact` 붙은 항목을 추가한다고 규정한다. "이 로그가 필요한 이유는 downstream(`engram`)이 upstream 규칙과의 동등성을 harness로 검증하기 때문이다."

**engram 대응**: `scripts/upstream-sync.py`가 `meta/CHANGELOG.md` 변화만 delta로 뽑는다(`scripts/upstream-sync.py:169-180`). `AGENTS.md` 본문 자체는 계약 파일 목록을 추출하는 데만 쓰이고(`:64,76,79`) vendoring도 diff도 하지 않는다. `harness/upstream/`에 있는 것은 7개 명세뿐이고 `AGENTS.md`, 단계 README, `meta/templates/`는 없다.

**영향**: 계약 파일 전체가 vendoring 대상이 아니다.

**자리**: 코드. `harness/`의 검증 범위다

**출처**: E10

---

### F68. Superpowers 산물 격리 부재

**upstream 규정**: `AGENTS.md:48-50`이 Superpowers 산물 격리를 규정한다. brainstorming spec / plan / note 산물은 `.superpowers/`에 두고 위키 인덱싱과 git 추적 대상이 아니다. durable knowledge는 **승급 파이프라인으로만** 위키에 올린다.

**engram 대응**: engram에 대응이 없다. 단계 디렉토리 밖의 작업 산출물을 격리하는 개념이 없다. `ignore_files`(ADR 0036)는 단계 디렉토리 **안**의 비문서를 빼는 것이라 성격이 다르다.

**영향**: 단계 밖 산물 격리가 없다.

**자리**: 밖. 특정 도구의 산물 격리다

**출처**: E33

---

### F69. `_drop` 인테이크 차이

**upstream 규정**: `AGENTS.md:142-159`이 Natural-Language Intake Triggers를 규정한다. 사용자가 스크립트 이름을 대지 않아도 되게 한다. `inbox/_drop/`에 아무 이름으로 던지면 에이전트가 채널/날짜/슬러그를 추론하고 원본을 `sources/transcripts/`에 보존한 뒤 초안을 만들고 `_drop/`을 비운다. "**Never ask the user to pick a path or filename.**"

**engram 대응**: `_drop` 개념이 없다. 다만 같은 목적을 다른 방식으로 달성한다. `capture`/`source`/`new`가 슬러그를 제목에서 파생하므로 사용자가 경로와 파일명을 고르지 않는다(ADR 0020). 원본 보존은 ADR 0051의 `source-raw`와 `promote --to sources`(ADR 0058)가 맡는다.

**영향**: 인테이크 방식이 다르다.

**자리**: 밖. engram이 이미 다른 방식으로 같은 목적을 달성했다

**출처**: E34

---

### F70. 색인 제외 차이

**upstream 규정**: `AGENTS.md:14`(Core Principle 6)이 "RAG/시스템은 raw mixed inbox가 아니라 승인된 `context/`만 인덱싱한다"고 규정한다.

**engram 대응**: `serve`(ADR 0044)와 `export`(ADR 0046)가 `internal/expose`로 inbox와 sources를 닫는다. 다만 `engram search`/`recall`의 색인은 단계를 가리지 않는다. upstream lint는 추가로 `inbox artifacts must be indexable false`(`scripts/lint-frontmatter.sh:272-273`)를 FAIL로 잡는데 engram은 잡지 않는다(실측: `inbox/idx.md`에 `indexable: true`를 넣어도 위반 없음).

**영향**: 검색 색인에서 inbox/sources가 걸리지 않는다.

**자리**: 코드

**출처**: E37

---

### F71. 외부 발행물 특수문자 정제 부재

**upstream 규정**: `AGENTS.md:220-234`가 외부 발행물 산문에 em dash, 화살표, 가운뎃점, 말줄임, smart quotes를 쓰지 않는다. "llm-wiki 내부 마크다운을 외부로 export할 때 정제한다."

**engram 대응**: `engram export`는 사용자 치환 사전만 적용한다(ADR 0046, `internal/export/replacements.go`). 특수문자 정규화가 없다. `grep -rn --include='*.go' 가운뎃점 internal cmd` 0건.

**영향**: 특수문자 정제가 없다.

**자리**: 코드. 반출은 되돌릴 수 없고 변환이 기계적이다

**출처**: E8

---

### F72. 미러 포함 범위 차이

**upstream 규정**: `AGENTS.md:200-218`, `mirror/README.md`, `README.md:69-88`이 Mirror Rule 전체를 규정한다. 미러는 진실원이 아니고 쓰기 경로가 아니다. 미러 범위와 제외 3종(`sensitivity: private-local-only` 문서, `sources/raw-private|attachments|exports`, `inbox|private|.local|.cache`)다.

**engram 대응**: engram 범위 밖(spec-map 4.9가 명시). 다만 제외 규칙 첫째는 engram의 `expose`와 같은 판정이다. `internal/expose/expose.go:186`이 `sensitivity` 축이 켜져 있을 때 `private-local-only`와 `restricted`를 닫는다. upstream 미러는 `restricted`까지 **포함**한다(`AGENTS.md:212`).

**영향**: 미러 제외 범위가 다르다.

**자리**: 밖. 미러는 engram 범위 밖이다

**출처**: E29

---

### F73. `source` 필드 부재

**upstream 규정**: `README.md:58-67` Scope Metadata가 `scope`(4값), `sensitivity`(4값), `source`(8값), `status`(4값)을 규정한다.

**engram 대응**: `scope`, `sensitivity`, `status`는 있다. `source` 필드는 없다. engram은 `source_channel` 하나만 쓴다. upstream도 실제 문서에서는 `source_channel`을 쓰므로 `README.md`의 `source` 표기가 낡은 것으로 보인다.

**영향**: `source` 필드가 없다.

**자리**: 밖. upstream 자신의 표기가 낡은 것으로 보인다

**출처**: E31

---

### F74. `meta/weekly-news-sources` 계약 밖

**upstream 규정**: `meta/CHANGELOG.md`가 특정 워크플로의 소스 데이터 목록을 계약에서 명시 제외한다(`meta/CHANGELOG.md:29`).

**engram 대응**: engram 범위 밖. 대응 개념 없음. 다만 이 파일의 frontmatter가 `sensitivity: public`(taxonomy 4값 밖)와 `type: reference`(allowed_type 밖)를 쓰고 필수 필드 다수가 없다. `meta/`가 lint 스캔 대상이 아니라 드러나지 않는다.

**영향**: 특정 워크플로 소스가 계약 밖이다.

**자리**: 밖. engram 범위 밖이다

**출처**: E32

---

### F75. 타이밍 로딩 경로 부재

**upstream 규정**: `context-systems/nanoclaw-personal-agent-architecture.md:47-52`가 보안 경계 넷을 규정한다. credential을 container에 직접 주입하지 않는다, host/container 통신은 메시지와 결과 전달에 집중, group별 filesystem과 volume mount로 범위 제한, 외부 channel과 MCP/tool 호출은 별도 approval boundary다.

**engram 대응**: 마지막 하나만 engram과 겹친다. ADR 0043이 `mcp`의 쓰기 도구를 하나로 줄이고 `promote`를 뺐다. 나머지 셋은 에이전트 런타임 사안이라 engram 범위 밖.

**영향**: 일부 보안 경계가 구현되지 않았다.

**자리**: 밖. 에이전트 런타임 사안이다

**출처**: E40

---

### F76. artifact TTL 삭제 워크플로 부재

**upstream 규정**: `context/systems/brownfield-analysis-architecture.md:189-196`이 장기 저장 회피 목록을 규정한다. raw 소스, raw DB 스키마, raw graph 파일, 소스 조각을 담은 CLI 로그다. "If raw graph artifacts must be kept, store them as **encrypted project-scoped restricted artifacts with TTL and delete workflow**"라 규정한다.

**engram 대응**: engram에 `sensitivity: restricted` 값은 있으나(`config.go:249`) TTL도 삭제 워크플로도 없다. `archive`는 수명이 끝난 문서를 보관할 뿐 삭제하지 않는다(ADR 0028). 다만 이 규범이 위키를 겨눈 것인지 제품을 겨눈 것인지는 문서상 제품 쪽이다.

**영향**: TTL 삭제 워크플로가 없다.

**자리**: 밖. 이 규범이 겨눈 것이 위키가 아니라 제품이다

**출처**: C39

---

### F77. `context/mocs/`와 `type: moc` 부재

**upstream 규정**: `context/mocs/*.md` frontmatter(`llm-wiki-architecture.md:3,10`, `indexing-config.md:3,10`, `voice-memo-workflow.md:3,10`, `blog-publishing.md:3,10`)가 MOC 문서는 `type: moc`와 `artifact_stage: index`를 쓴다고 규정한다. 넷 모두 그렇다.

**engram 대응**: engram 허용 집합에 둘 다 없다(`internal/config/config.go:241-246`). `root_files`(기본 `index.md`)가 색인의 자리를 대신하고 게이트와 고아 판정에서 뺀다(ADR 0019). `docs/spec-map.md:288`이 이미 실데이터의 MOC 5건이 거절된다고 적어 두었다.

**영향**: MOC 타입과 index 단계가 없다.

**자리**: 설정. `types` 집합에 값을 더하면 되고 색인의 자리는 `root_files`가 대신한다

**출처**: C35

---

## 판정이 갈린 것

자리를 정하기 어려웠던 항목이다. 왜 갈렸는지와 어느 쪽으로 기울였는지를 적는다.

**G2 `log.md` 기록과 F15 발행 이력.** 둘은 같은 뿌리다. 커맨드가 한 일을 위키에
남기는 것이다. 그런데 자리를 다르게 뒀다. G2는 코드이고 F15는 밖이다. 가른 것은
커맨드의 성격이다. `promote`는 이미 위키를 고치는 커맨드이므로 승급 시각을
프론트매터에 남기는 것이 그 커맨드의 일이고, `digest`가 도움말에서 "승급 집계는
promote가 승급 시각을 프론트매터에 남기지 않아 여기에 없습니다"라고 스스로 결손을
선언했다. 반면 `export`는 위키를 고치지 않는 것이 계약이다. 반출 이력을 위키에
쓰면 읽기 전용 커맨드가 쓰기 커맨드가 된다. `log.md`라는 파일 형태 자체는 어느
쪽에서도 코드가 아니다. 데모 위키가 본보기를 보여 줄 몫이다.

**G10 한국어 특수문자 강제와 F71 반출 시 정제.** 같은 문체 규칙인데 G10은 밖이고
F71은 코드다. 위키 안의 산문 문체는 조직 취향이며 위키가 한국어가 아닐 수도 있다.
남의 위키 본문에 문체를 강제하는 것은 engram이 할 일이 아니다. 반면 반출은 파일이
기계 밖으로 나가는 행위라 되돌릴 수 없고, upstream 자신이 이 변환을 기계적인
것으로 규정했다. 같은 이유로 F17 반출 시 링크 변환도 코드에 뒀다. 다만 둘 다 기본
켜짐이 아니라 여는 옵션이라는 전제에서의 판정이다.

**F1 트리거 표.** 자리 다섯에 `internal/skills/SKILL.md`가 없다. SKILL.md는
바이너리가 배포하는 산물이지만 `CLAUDE.md`가 밝힌 대로 권고이지 강제가 아니고,
자연어 발화를 커맨드에 잇는 일은 결정론적 판정이 아니다. 가르치는 성격이 가장
가까워 교재로 기울였다. F16과 F69도 SKILL.md 계열이나 그 둘은 이미 다른 방식으로
같은 목적을 달성했다고 문서가 적었으므로 밖에 뒀다. **자리 축에 SKILL.md가 없는
것은 이 분류의 빈자리이며 나중에 축을 늘릴지가 남는다.**

**G6 `source_refs` 값 검사.** upstream이 "승급 문서의 `source_refs` 누락은 검수
이슈로 다뤄야 한다"고 적어 사람에게 넘긴 항목이다. 그 문장만 보면 밖이다. 그러나
빈 배열인지의 판정은 결정론적이고, 지금은 `promote`가 만든 문서 전부가 추적성
검사를 형식적으로만 통과한다. 검사를 코드가 하고 누락을 어떻게 처리할지를 사람이
정하는 것으로 갈라 코드에 뒀다.

**G14 사람 문서 조건부 색인.** `context/people/`이라는 버킷 자체가 조직 어휘라
밖으로 기울였다. 다만 "특정 하위 디렉토리를 색인에서 뺀다"는 설정으로 만들 수
있고, 그 설정은 F2가 말하는 `ignore_dirs`와 같은 자리다. 단계 디렉토리 아래를
사용자의 자유로 둔 `docs/spec-map.md` 4.2를 따라 밖에 뒀다.

**F36 git hook 설치.** 훅 등록은 사용자 저장소의 운용이라 밖으로 볼 수 있다.
그런데 `eject`가 이미 Python 린터를 내보내면서 훅 등록만 하지 않는다. 같은
커맨드가 절반만 하고 있는 상태이므로 코드로 기울였다.

**R8 용어 정규화.** 치환 실행 자체는 결정론적이고 `export --replacements`가 이미
그 코드를 갖고 있다. 인입 단계에 같은 것을 여는 것은 코드 변경이다. 그러나 무엇을
언제 정규화할지가 전사 도구와 조직 어휘에 매여 있고, upstream이 자동 적용과 사람
검토를 나눈 구분 자체가 그 조직의 용어표에 달려 있다. 밖으로 기울였다.

**F53과 F54 시효 자료 필드.** upstream 안에서 규정이 엇갈린다. 한쪽은 시효 있는
자료에 유효 기간이나 재검토 날짜를 표시하라고 하고, 다른 쪽은 `review_after`가
남아 있으면 FAIL이라고 하며 그 근거로 "예측이라 지킬 수 없고, 지나도 아무 일이
일어나지 않는다는 것이 실측으로 확인됐다"를 든다(F23). 폐기 쪽이 나중이고 근거가
실측이므로 밖에 뒀다. **upstream의 두 규정이 어긋난다는 사실 자체가 이 항목의
판단 재료다.**

**F56 공개 문맥 통과.** 회사명과 내부 프로젝트명은 사전이 필요해 사람의 몫이다.
그러나 토큰, 계정 식별자, 사설 경로, 내부 URL은 패턴으로 잡힌다. 반출이
되돌릴 수 없는 행위라 잡히는 것만이라도 코드가 잡는 쪽으로 기울였다. G13 시크릿
스캔, F59 토큰 승급 금지와 같은 뿌리이며 셋을 하나의 검사로 묶을 수 있다.

**F39 템플릿.** 템플릿 경로를 `engram.yaml` 키로 열면 설정이다. 그러나 서식은
조직 취향이고 ADR 0029가 upstream `meta/templates/`를 서식으로 분류해 계약 목록
밖에 뒀다. 그 판정을 뒤집을 근거가 없어 본보기를 보여 주는 데모로 기울였다.
G11, G12, F5, F19, F20, F63도 같은 이유로 데모다.

**F18 승급 자격 여섯.** 여섯 중 다섯은 "결론이 재사용할 만큼 안정적인가" 같은
판단이라 코드가 아니다. `docs/spec-map.md` 4.3이 이미 그 다섯을 사람에게 남긴
것으로 적었다. 남는 것은 `owner` 축 하나이고 축을 더하는 것은 설정의 일이라
설정에 뒀다.

**F34 색인 자격 네 조건.** ADR 0063이 노출 판정에 `indexable`과 `status`를
넣어 이 항목의 절반을 닫았다. 그러나 upstream은 `status == promoted`를 적극
조건으로 요구하고 engram은 `superseded` 하나만 뺀다. 남은 차이가 있어 닫힘으로
표시하지 않고 코드에 뒀다. **이 항목의 "engram 대응" 문장은 0063 이전 상태를
적은 것이라 낡았다.**

## 참고

- 이 문서는 다섯 감사 보고서의 "없음"과 "다르게 구현됨" 판정을 중복 제거하고 카테고리별로 정리한 것이다.
- 각 항목의 출처는 원본 보고서의 항목 번호를 참조한다.
- "의도적 제외"로 표기된 항목은 ADR이나 spec-map에 이미 명시된 의도적 차이를 말한다.
- "spec-map 기재済"로 표기된 항목은 `docs/spec-map.md`에 이미 기록된 항목을 말한다.
