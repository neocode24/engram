# 규칙 명세 대응 표

> upstream 위키의 규칙 명세 7종이 engram의 무엇이 되었는지를 빠짐없이 적은 문서다. 명세 사본은 [harness/upstream/](../harness/upstream/)에 고정되어 있다. 이 문서를 읽으면 각 규칙이 코드로 강제되는지, 설정으로 열려 있는지, 사람에게 남아 있는지를 한 번에 알 수 있다.

## 1. 이 문서가 답하는 것

이 문서는 세 종류의 독자를 상정한다.

강의에서 이 체계를 설명하는 사람에게는 **정확한 근거 목록**을 준다. "연결 없는 문서는 승급이 거절된다"고 말할 때 그 규칙이 어느 명세의 어느 줄에서 왔고 코드의 어느 규칙 ID로 판정되는지를 이 문서에서 바로 찾을 수 있다. 슬라이드 한 장을 만들 때 원전 인용이 필요한 사람을 위한 색인이다.

저장소를 처음 열어 "이게 뭘 하는 도구인가"를 3분에 알고 싶은 사람에게는 **차별점의 실체**를 준다. 다른 노트 도구와 다른 점이 기능이 아니라 강제에 있다면, 무엇을 강제하고 무엇을 강제하지 않는지가 곧 이 도구의 정체성이다. 3절의 세 방향 표와 5절의 승급 규칙 분석이 그 질문에 답한다.

새 규칙을 추가하려는 사람에게는 **갈림길의 판단 기준**을 준다. 새 규칙이 결정론적으로 판정 가능하면 코드로, 위키마다 다르게 정할 성격이면 설정으로, 판단이 필요하면 문서와 검토 절차로 간다. 4절의 대응표는 지금까지 그 판단이 어떻게 내려졌는지의 전례 모음이어서, 자기 규칙을 어디에 둘지 정할 때 비교할 수 있다.

## 2. 명세는 engram의 기능이 아니라 사양서다

가장 먼저 오해를 끊는다. **engram을 써도 명세 파일은 하나도 생기지 않는다.** `engram init`을 돌려면 다음이 만들어진다. 임시 디렉토리에서 실제로 실행해 확인한 목록이다.

```
$ engram init wiki
inbox/       sources/     context/     archive/
engram.yaml  index.md     .gitignore
```

`meta/` 디렉토리는 없다. engram 사용자는 명세를 평생 보지 않는다.

upstream에서 명세는 사람과 스크립트가 읽는 운용 문서였다. `meta/` 아래 마크다운으로 두고, 사람은 읽고 지켰고, shell과 python 스크립트는 자기 안에 하드코딩한 규칙으로 검사했다. 동등성 검증에서 실측한 바에 따르면 upstream의 lint 스크립트는 명세를 읽지 않고 허용값과 필수 필드를 스크립트 본문에 박아 두었다. 그래서 명세와 스크립트가 어긋나는 지점이 있었다.

engram은 같은 규칙을 두 곳으로 옮겼다. **결정론적으로 판정되는 규칙은 Go 코드**(`internal/lint`, `internal/cli`의 게이트)가 강제하고, **위키마다 다르게 정하는 값은 `engram.yaml`**이 담는다. 명세 사본 자체는 `harness/upstream/`에 원본 커밋 해시와 함께 고정되어 있으며 이 문서 7절의 동기화 대상이다. 산출물 위키에는 따라오지 않는다.

## 3. 규칙이 갈라지는 세 방향

| 방향 | 무엇 | 어긴 결과 |
|---|---|---|
| 코드로 강제 | lint 규칙 16종과 승급 게이트 | 종료 코드 1. 승급이 막힌다 |
| 설정으로 열어 둠 | `engram.yaml`의 속성과 임계값 | 위키마다 다르게 정한다 |
| 사람과 에이전트에 남김 | 판단이 필요한 것 | 도구가 관여하지 않는다 |

가르는 기준은 한 문장이다. **같은 입력에 같은 출력이 나오는가.** 결정론적으로 판정되는 것만 코드가 강제한다. 이 선은 [ADR 0014](decisions/0014-llm-boundary-agent-drives-binary.md)가 LLM 경계를 긋던 기준과 같다. 결정론적 연산은 바이너리가, 판단은 사람이나 에이전트가 맡는다.

설정 속성은 이 분류의 완충지대다. `sensitivity` 같은 축은 값의 종류는 코드가 검사하지만 그 값을 무엇으로 정할지는 사람의 판단이다. 그래서 축은 "켜고 끈다"로만 코드가 관여하고 값 자체는 문서를 쓰는 사람이 정한다.

## 4. 명세 7종 대응표

lint 규칙 16종은 이 절 전체에서 각각 정확히 한 번씩 등장한다. 규칙 ID는 `internal/lint/lint.go`에 실재하는 값만 쓴다.

### 4.1 frontmatter-schema.md

**이 명세가 선언하는 것.** 프론트매터를 기계가 읽는 제어면으로 삼는다. 단계(inbox, source, context)별 필수 필드를 정의하고, 필드별 의미와 허용값을 표로 정의한다. 날짜를 `created`, `sourced_at`, `updated` 셋으로 나누고 `sources/`에는 `updated`를 쓰지 않는다고 못 박는다. 검색 시스템의 인덱싱 자격을 `artifact_stage`와 `indexable`로 판정하라고 지시한다.

**코드로 강제하는 것.**

| 규칙 ID | 등급 | 판정 |
|---|---|---|
| `frontmatter.missing` | error | 프론트매터 블록이 아예 없음 |
| `frontmatter.unclosed` | error | 여는 `---`은 있으나 닫는 `---`이 없음 |
| `frontmatter.yaml` | error | 프론트매터 YAML 문법 오류 |
| `frontmatter.missing-field` | error | 단계별 필수 필드 누락 |
| `schema.allowed-value` | error | `artifact_stage`, `status`, `scope`, `sensitivity`, `trigger_mode` 값이 허용 집합 밖 |
| `schema.axis-off` | error | 설정이 끈 속성가 문서에 있음 |
| `sources.updated` | warn | `sources/` 문서가 `updated` 필드를 가짐 |

`promote`와 `new`는 이 명세의 단계별 필드를 채워 쓴다. 채우는 값의 진실원은 `internal/wiki`의 단계별 초기값이다.

**설정으로 열어 둔 것.** 속성 14종의 on/off(`axes`), 문서 종류 집합(`types`), 문서가 아닌 파일 목록(`ignore_files`, 기본 `README.md`). 꺼진 축의 필수성은 사라지고 `schema.axis-off`가 오히려 그 필드의 존재를 잡는다. 필수 필드 검사가 명세의 단계별 표를 그대로 옮기지 않고 프리셋에 따라 변형되는 것이 이 대응이다. 날짜 필드의 이름과 형식은 설정 키가 없이 코드에 고정되어 있다.

**사람에게 남긴 것.** 검토 상태(review state)의 운용, `meta/templates/`의 템플릿 관리(Template Rule), "indexable: true인데 restricted인 문서 찾기" 같은 검토 뷰의 구성이다. engram에는 검토 상태 속성이 없다.

### 4.2 wiki-artifact-schema.md

**이 명세가 선언하는 것.** 채널, 계기, 워크플로, 단계 네 속성으로 산출물을 분류하라고 정한다. 폴더를 늘리는 대신 축 값을 조합하라는 것이 핵심 주장이다. artifact 유형별 기본 위치와 인덱싱 기본값을 표로 정의하고, 품질 단계(inbox, source, context)를 나눈다.

**코드로 강제하는 것.**

| 규칙 ID | 등급 | 판정 |
|---|---|---|
| `location.stage-agreement` | error 또는 warn | 문서가 놓인 최상위 디렉토리와 `artifact_stage` 값이 어긋남. `context`를 선언했는데 `context/`에 없으면 `error`, 그 밖의 불일치는 `warn`([ADR 0035](decisions/0035-stage-mismatch-severity-by-direction.md)) |

이 규칙이 이 명세의 "유형별 기본 위치" 표를 판정으로 바꾼 것이다. 등급이 방향으로 갈리는 이유는 해악이 다르기 때문이다. `context`를 선언하고 `context/` 밖에 있는 문서는 검수된 지식의 필드 집합과 색인 자격을 얻어 게이트를 우회한다. 반대 방향은 필수 필드가 느슨하게 검사될 뿐 아무 관문도 지나치지 않는다. 단계와 디렉토리의 대응 표는 `internal/wiki`가 단일 진실원이고 lint는 그것을 읽는다([ADR 0031](decisions/0031-location-must-agree-with-stage.md)).

**설정으로 열어 둔 것.** `page_dirs`(단계별 디렉토리 이름), `root_files`, `source_channel`, `trigger_mode`, `workflow` 축. 셋은 개방 집합이라 `engram.yaml`의 폐쇄 집합에 담지 않고 축 on/off만 둔다. `types` 집합은 위키별로 정의한다.

**사람에게 남긴 것.** `sources/manifests/`, `context/decisions/` 같은 하위 디렉토리 구조. engram은 단계 디렉토리 아래를 사용자가 나누는 자유로 둔다. `moc` 유형과 `index` 단계는 6절에서 다룬다.

### 4.3 promotion-rules.md

**이 명세가 선언하는 것.** 승급 기준과 승급 금지 기준을 각각 다섯 줄과 네 줄로 적고, 승급 문서가 갖춰야 할 출력물 다섯을 적는다. 전문은 5절에서 분석한다.

**코드로 강제하는 것.**

| 규칙 ID | 등급 | 판정 |
|---|---|---|
| `gate.min-wikilinks` | reject | context 문서의 고유 위키링크 수가 `min_wikilinks` 미달 |

`promote`와 `new`는 이 판정을 승급 시점에 직접 돌리고 거절하면 종료 코드 1로 끝낸다. `lint.EvaluateGate`가 단일 진실원이라 커맨드로 통과한 문서를 lint가 다시 거절하지 않는다. `new`는 승급 문서의 절 골격을 "결론, 맥락, 현재 이해, 근거, 관련 링크" 순서로 만들어 두는데, 이 명세의 출력물 목록을 절 제목으로 옮긴 것이다. 골격은 비워 둔다. 내용을 지어내지 않는다.

**설정으로 열어 둔 것.** `min_wikilinks`. 0으로 두면 게이트가 꺼진다.

**사람에게 남긴 것.** 승급 기준 다섯 줄과 금지 기준 네 줄의 판단 전부. "재발할 질문에 답하는가", "결론이 재사용할 만큼 안정적인가"는 코드가 판정할 수 없어 검토자의 몫으로 남는다. 골격의 절을 채우는 일도 사람과 에이전트가 한다.

### 4.4 wiki-graph-policy.md

**이 명세가 선언하는 것.** 관계 필드 여섯(`source_refs`, `derived_from`, `derived_context`, `related`, `supersedes`, `superseded_by`)과 그 방향을 정의한다. 단계별 링크 요구사항을 정하고, context 단계는 인접 문서가 있으면 `related`를 요구하며, MOC 작성 규칙을 둔다. 링크는 의미 있을 때만 걸라고 못 박는다.

**코드로 강제하는 것.**

| 규칙 ID | 등급 | 판정 |
|---|---|---|
| `link.broken` | warn | 위키링크가 가리키는 문서가 위키에 없음 |
| `graph.orphan` | warn | 들어오는 관계와 나가는 관계가 모두 없음 |

고아 판정은 관계 필드(`derived_from`, `derived_context`, `source_refs`)를 위키링크와 같은 기준으로 센다. `promote`가 `sources/` 문서에서 파생을 만들 때 `derived_from`과 `derived_context`를 양방향으로 기록하므로([ADR 0022](decisions/0022-promote-moves-inbox-derives-sources.md)), 파이프라인의 산출물을 검사기가 못 보면 안 된다.

**설정으로 열어 둔 것.** `related`, `source_refs`, `derived_from`, `derived_context` 속성의 on/off.

**사람에게 남긴 것.** 링크를 걸 만한 의미가 있는지의 판단, MOC를 만들고 갱신하는 시점, 승급 뒤 원본 문서의 `derived_context`를 갱신하는 운용. `supersedes`와 `superseded_by`는 속성에 없어서 6절의 미이행 항목이다.

### 4.5 taxonomy.md

**이 명세가 선언하는 것.** 입력 채널, 지식 유형, 민감도, 스코프의 초기 분류를 정의한다. 오직 `public-reference`와 안전한 `internal` 자료만 승급 후보로 삼으라고 적혀 있다. 캡처 시점에 스코프를 억지로 분류하지 말고 모르면 `unknown`을 쓰라고 한다.

**코드로 강제하는 것.**

| 규칙 ID | 등급 | 판정 |
|---|---|---|
| `taxonomy.forms` | error | `form` 값이 `forms` 폐쇄 집합에 없음 |
| `taxonomy.topics` | warn | `topics` 값이 설정에 정의되지 않음. 개방 집합이라 경고만 |

**설정으로 열어 둔 것.** `topics`(개방 집합), `forms`(폐쇄 집합), `types` 집합. 미정의 값이 오류냐 경고냐로 갈리는 것이 두 집합의 차이고, 이 구분이 이 명세의 "초기 분류"가 위키마다 자라는 방식이다. 민감도와 스코프의 허용 집합은 `engram.yaml` 키가 없이 코드에 고정되어 있다.

**사람에게 남긴 것.** 민감도 판단. "오직 public-reference와 안전한 internal만 승급 후보"라는 문장은 코드가 강제하지 않는다. 승급 심사에서 사람이 민감도 속성 값을 읽고 판단한다. 스코프 미결정을 `unknown`으로 남기는 관행은 코드가 값으로 허용하는 것일 뿐 검사 대상이 아니다.

### 4.6 ingest-rules.md

**이 명세가 선언하는 것.** 기본 입력 절차 여섯 단계를 정한다. 채널을 식별하고, 원본을 `sources/`에 보존하고, 처리 노트를 `inbox/`에 만들고, 남을 사실을 뽑고, 중복과 충돌을 확인하고, 승급 대상과 제목을 제안한다. 채널별로 남길 메타데이터를 정하고, 음성 원본은 커밋하지 말고 전사와 요약만 남기라고 정한다.

**코드로 강제하는 것.** lint 규칙은 없다. 절차의 두 단계가 커맨드로 굳어 있다. `capture`가 검증 없이 `inbox/`에 받는 3단계이고, `source`가 원본 필드를 확정해 `sources/`에 두는 2단계다. 나머지 단계는 판단을 포함하므로 커맨드가 대신할 수 없다.

**설정으로 열어 둔 것.** `source_channel` 축. `source`의 `--channel` 플래그가 이 값을 받는다.

**사람에게 남긴 것.** 6단계 중 판단 단계 전부(남을 사실 뽑기, 중복 확인, 승급 대상 제안). 채널별 메타데이터 선호(Teams는 회의 제목과 참가자, Slack은 스레드와 permalink)도 남아 있다. 원본 음성을 보존할지 결정하는 일과 `sources/raw-private/` 운용도 이 명세의 범위다. upstream이 `inbox/<channel>/` 하위 디렉토리를 만드는 것과 달리 engram은 채널을 플래그 값으로만 남긴다.

### 4.7 security-rules.md

**이 명세가 선언하는 것.** 커밋 금지 항목(자격증명, API 토큰, 개인 키, 세션 쿠키, 불필요한 개인 식별 정보, 통제 안 된 원본 반출, 편집 없는 고객 민감 정보)을 나열한다. 민감 원본은 `sources/raw-private/`에 두고 개인 클라우드로 미러링하지 말라고 정한다.

**코드로 강제하는 것.** **없다.** 이 명세는 engram이 가장 적게 옮긴 명세다.

**설정으로 열어 둔 것.** `sensitivity` 축뿐이다. 값의 종류는 `schema.allowed-value`가 검사하지만 그 값이 무엇을 지키는지는 이 명세의 약속이다.

**사람에게 남긴 것.** 커밋 금지 목록의 실천 전부. 이 저장소 자체가 공개 경계를 지키는 장치(`private/` 디렉토리와 경계 검사 스크립트, git hook)를 갖고 있으나 그것은 저장소의 운용 규칙이지 engram 바이너리의 기능이 아니다. 위키 사용자의 커밋 내용은 engram이 검사하지 않는다.

### 4.8 어느 명세에도 없는 engram 고유 규칙

세 규칙은 7종 어디에도 선언되어 있지 않다. engram이 운영 경험에서 스스로 넣은 것이다.

| 규칙 ID | 등급 | 근거 |
|---|---|---|
| `gate.deferred` | warn | 링크 가능한 대상 문서 자체가 부족해 게이트를 유예한다. 갓 만든 위키의 고립은 결함이 아니라 시작이라는 판단이다([ADR 0021](decisions/0021-gate-deferral-when-targets-are-scarce.md)) |
| `body.max-lines` | warn | 문서 줄 수가 `max_lines`를 넘으면 경고한다 |
| `wiki.broad-topic` | warn | 한 주제가 전체 문서의 `broad_topic_pct` 퍼센트를 넘게 붙으면 위키 단위 진단을 낸다 |

## 5. 승급 규칙이 이 프로젝트의 핵심인 이유

`promotion-rules.md`를 열면 승급 기준이 열네 줄 남짓이다. 승급하라는 다섯 줄과 승급하지 말라는 네 줄, 그리고 승급 문서가 갖출 다섯이다.

승급하라는 줄은 이렇다.

> - the note answers a future question likely to recur
> - the source is known
> - the conclusion is stable enough to reuse
> - sensitive details are removed or scoped
> - duplicates/conflicts have been checked

이 중 기계가 판정할 수 있는 줄은 하나도 없다. "재발할 질문에 답하는가"는 판단이고 "결론이 안정적인가"도 판단이다. 기계가 확실히 셀 수 있는 것은 출력물 목록의 `related links` 한 줄뿐이다.

engram이 한 일이 정확히 이것이다. **열세 줄의 판단을 그대로 사람에게 남겨 두고, 셀 수 있는 한 줄만 골라 그것을 유일한 거절 사유로 만들었다.** `gate.min-wikilinks`가 거절 사유이고 나머지는 경고조차 되지 않는 이유다. 거절 조건을 늘리면 사용자가 게이트를 우회하는 순간 게이트가 무의미해진다.

세 갈래를 비교하면 이 선이 왜 핵심인지 드러난다.

| 도구 | 코드로 강제하는 것 | 결과 |
|---|---|---|
| 일반 노트 앱 | 없음 | 넣기만 하고 끝난다. 6개월 뒤 검색되지 않는 더미가 된다 |
| LLM으로 감싼 도구 | 승급 기준 전부를 코드 열에 넣으려 한다 | 같은 문서에 두 번 물으면 다른 답이 나온다. 게이트가 게이트가 아니게 된다 |
| engram | 셀 수 있는 한 줄만 | 결정론적인 것은 확실히 막고, 판단은 검토자에게 남는다 |

LLM으로 승급 기준 전부를 판정하게 하면 그 판정은 재현되지 않는다. 같은 문서를 두 번 심사해 다른 결론이 나오면 거절의 권위가 사라진다. 결정론 경계는 성능 문제가 아니라 게이트의 존재 조건이다.

## 6. 아직 옮기지 않은 것

숨기지 않고 적는다. 셋은 처음부터 확인된 것이고 넷째는 명세를 읽으며 확인했다.

| 항목 | upstream | engram |
|---|---|---|
| `supersedes` / `superseded_by` | wiki-graph-policy.md가 관계 필드로 선언 | 속성 14종에 없다 |
| `index` 단계와 MOC | `context/mocs/`를 별도 단계로 둔다 | 없다. 의도된 차이지만 실데이터에서 5건이 거절된다 |
| 단계 디렉토리 안의 비문서 파일 | 명세에 개념이 없다. 실데이터에는 `README.md`가 네 자리에 있다 | ~~없었다~~ `ignore_files`로 순회에서 뺀다([ADR 0036](decisions/0036-non-document-files-in-stage-dirs.md)) |
| security-rules.md 전체 | 커밋 금지 항목과 민감 자료 취급 | 코드로 강제하는 것이 하나도 없다. `sensitivity` 축만 있다 |
| `indexable` 단계 기본값 | wiki-artifact-schema.md가 유형별 인덱싱 기본값으로 선언. upstream lint는 위반을 검사 | 강제 규칙이 없다. 쓰기 커맨드가 단계별 초기값으로만 채운다 |

`supersedes`와 `superseded_by`는 지금 사람이 `related`나 본문 링크로 대신 표현한다. 결정할 시점은 대체 이력을 코드가 다뤄야 할 때다. 폐기와 개정의 이력 관리가 `archive` 너머로 확장되거나, 동등성 검증에서 이 필드를 쓰는 문서가 걸릴 때다.

`index` 단계는 지금 `root_files`(`index.md`)가 색인의 자리를 대신한다. 색인 문서는 게이트와 고아 판정에서 의도적으로 뺀다([ADR 0019](decisions/0019-index-documents-outside-the-gate.md)). 실데이터의 MOC 문서 5건은 `artifact_stage: index` 값을 갖고 있어 engram의 허용 집합에서 거절된다.

같은 자리에서 성격이 다른 것이 함께 나온다. 실운영 위키 306문서에 engram을 돌리면 `schema.allowed-value`가 열다섯이고 그중 다섯만 `index` 단계다. 나머지 열은 `status: draft` 다섯, `trigger_mode: manual` 넷, `sensitivity: public` 하나이며 **셋 다 upstream 명세가 선언한 적 없는 값이다.** 명세는 `trigger_mode`를 `manual-prompt` 등 다섯으로, `sensitivity`를 `public-reference` 등 넷으로 못 박았다. 즉 이 열 건은 engram과 명세의 차이가 아니라 **upstream의 실제 위키가 자기 명세에서 흘러내린 것**이고 engram이 그것을 잡아낸 것이다. upstream 자체 lint는 `archive/`를 스캔하지 않는 등 검사 범위가 더 좁다. 결정할 시점은 동등성 검증에서 upstream 전용 규칙으로 남아 있는 위치 규칙(upstream의 `location.type-agreement`)을 정리하며 MOC를 받아들일지 한 번에 정할 때다.

security-rules는 지금 전부 사람과 저장소 정책에 남아 있다. 이 저장소의 공개 경계 장치는 있지만 그것은 engram의 기능이 아니다. 결정할 시점은 민감 자료가 실제로 섞이는 운영 환경이 생겼을 때다. `doctor`의 점검 항목 후보가 된다.

`indexable`은 지금 `promote`가 context 문서에 참을, `capture`가 inbox 문서에 거짓을 초기값으로 쓰는 방식으로만 지켜진다. 사람이 프론트매터를 고쳐 값을 바꿔도 lint는 잡지 않는다. 결정할 시점은 검색 인덱스가 민감 자료를 걸러 내야 할 필요가 생겼을 때다. security-rules의 이행과 같은 때에 볼 문제다.

## 7. 명세가 바뀌면 무슨 일이 도는가

[ADR 0005](decisions/0005-upstream-contract-and-harness.md)가 정하고 [ADR 0029](decisions/0029-upstream-vendoring-and-parity-execution.md)가 다듬은 3층 harness가 명세 변화를 다룬다.

| 층 | 답하는 질문 | 도구 |
|---|---|---|
| 명세 사본 고정 | 바뀌기 전에 규칙이 무엇이었나 | `harness/upstream/`, `harness/upstream.lock` |
| 변경분 감지 | 무엇이 바뀌었나 | `scripts/upstream-sync.py`, `private/deltas/` |
| 동등성 검증 | 반영이 맞았나 | `harness/parity`, `docs/parity.md` |

실제 명령 순서는 다음과 같다. 두 문서의 사용법에서 가져왔다.

```
# 1. 명세 사본을 갱신한다. lock의 SHA부터 upstream HEAD까지의 차이를
#    harness/upstream/에 다시 만들고 변경분을 private/deltas/에 남긴다.
python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki
python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki --check   # 쓰지 않고 미리 보기

# 2. 변경분을 사람이 읽고 어느 규칙에 영향을 주는지 판단한다. 자동 반영은 없다.

# 3. 반영한 뒤 동등성 검증을 돌린다. 결과는 t.Log 로만 남으므로
#    사람이 docs/parity.md에 옮겨 적는다.
ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/... -v
```

동등성 검증은 지금 lint 위반 목록 하나의 축을 가진다. upstream 저장소가 비공개라 환경변수가 없으면 건너뛰고 **공개 저장소의 CI는 항상 건너뛴다.** 비교 축을 늘리는 작업이 남아 있다.

한 가지 표기를 밝혀 둔다. **upstream은 자기 문서에서 이 파일들을 여전히 "계약 파일"이라고 부른다.** `upstream-sync.py`가 upstream `AGENTS.md`의 "계약 파일" 선언 문구를 그대로 파싱해 대상 목록을 얻으므로 저쪽 표기는 바뀌지 않는다. 이 저장소가 산문에서 "규칙 명세"로 부르기로 한 것과 충돌하지 않는다. 다음 사람이 표기가 어긋났다고 오해하고 고치려 들지 않도록 하는 주석이다.

## 관련

- [architecture.md](architecture.md) 동작 구조와 도식
- [design.md](design.md) 커맨드 체계와 설정
- [parity.md](parity.md) 동등성 검증 측정 결과
- [ADR 색인](decisions/README.md)
