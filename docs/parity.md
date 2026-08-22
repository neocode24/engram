# upstream 동등성 검증

upstream `llm-wiki`의 스크립트와 이 저장소의 Go 구현이 같은 입력에 같은 판정을 내리는지 비교한 결과다. ADR [0005](decisions/0005-upstream-contract-and-harness.md)가 정한 3층 harness의 세 번째 층이다.

## 실행 방법

```
ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/... -v
```

upstream 저장소는 비공개이며 로컬에만 있다. 환경변수가 없으면 비교를 건너뛴다. **CI는 항상 건너뛴다**(ADR [0029](decisions/0029-upstream-vendoring-and-parity-execution.md)). 이 문서는 자동 생성물이 아니라 로컬 실행 결과를 사람이 옮겨 적은 것이다.

## 측정 기준

| 항목 | 값 |
|---|---|
| 측정일 | 2026-08-16 |
| 반영 | ADR 0035(불일치 방향에 따른 등급), ADR 0036(`ignore_files`)까지 |
| upstream 커밋 | `05f7279` |
| 픽스처 | `harness/fixtures/golden-wiki`, 문서 14개 |
| 프리셋 | `team` |
| 비교 축 | lint 위반 목록 |

위 표는 lint 축의 조건이다. resurface 축의 조건은 아래 절에 따로 적는다.

**프리셋을 `team`으로 올려서 비교한다.** upstream 스크립트는 `engram.yaml`을 읽지 않고 자기 스키마를 하드코딩하며, 그 스키마는 `scope`, `sensitivity`, `source_channel`, `trigger_mode`, `workflow`를 전부 요구한다. 픽스처의 기본 프리셋인 `personal`은 그 속성들을 끄므로, 맞추지 않으면 비교 결과가 속성 on/off 차이로 뒤덮여 실제 규칙 차이가 묻힌다. 실제로 `personal`에서 error 4건이던 것이 `team`에서 48건이 되고 늘어난 44건이 전부 그 필드 누락이었다.

## 결과

| 갈래 | 쌍 |
|---|---|
| 양쪽 다 잡음 | **44** |
| upstream만 잡음 | 15 |
| engram만 잡음 | 15 |
| 정규화 매핑 없음 | 0 |

비교 단위는 (경로, 정규화한 규칙 이름) 쌍이다. 줄 번호는 넣지 않는다. 두 구현이 같은 위반을 다른 줄에서 가리킬 수 있기 때문이다.

## upstream만 잡은 것

engram이 따라가야 할 후보다. 셋으로 갈린다.

### 남은 차이

전부 의도했거나 판단이 끝난 것이다. 첫 측정에서 나온 진짜 갭 둘은 닫혔다.

| 규칙 | 건수 | 내용 |
|---|---|---|
| `frontmatter.missing-field:title` | 8 | **의도된 차이다.** 아래에서 설명한다 |
| `location.stage-agreement` | 1 | `index.md` 하나만 남았다. ADR [0019](decisions/0019-index-documents-outside-the-gate.md)에 따른 의도된 차이다 |
| `location.type-agreement` | 1 | **따라가지 않는다.** 적용 자리가 upstream에만 있는 `moc` 타입과 `index` 단계뿐이다 |

`inbox/wrong-stage.md`의 `location.stage-agreement`는 ADR [0031](decisions/0031-location-must-agree-with-stage.md)로 닫았다. 이제 양쪽이 함께 잡는다. 등급은 ADR [0035](decisions/0035-stage-mismatch-severity-by-direction.md)로 방향에 따라 갈렸으나 비교 단위가 (경로, 규칙 이름) 쌍이라 대조 결과는 바뀌지 않는다.

픽스처에 `inbox/misplaced-context.md`를 더했다. `artifact_stage: context`를 선언하고 `inbox/`에 남은 문서이며 0035가 `error`로 잡는 방향을 때린다. 이 문서 때문에 일치가 다섯 늘고 `title` 차이가 하나 늘었다. `index.md`에 붙은 둘은 upstream이 색인 문서에 `moc` 타입과 `index` 단계를 요구하기 때문이며, engram에는 그 타입도 그 단계도 없다.

#### title은 따라가지 않는다

upstream 스크립트가 이 규칙의 이유를 주석에 적어 두었다.

> context 노드는 사람이 읽는 한글 제목을 title에 둔다. 파일명은 링크 안정성 때문에 영어 slug로 유지하므로, title이 없으면 index.md가 slug로만 보인다.

**upstream은 파일명을 영어 슬러그로 유지하기 때문에 한글 제목이 갈 곳이 없다.** 그래서 프론트매터에 둔다.

engram은 ADR [0020](decisions/0020-slug-and-filename-rules.md)에서 반대로 정했다. 한글 슬러그를 보존하고 음차하지 않는다. 한글 제목이 곧 파일명이므로 upstream이 `title`을 필요로 하는 이유가 성립하지 않는다.

다만 슬러그는 되돌려도 원문이 아니다. 공백이 하이픈이 되고 대소문자와 구두점을 잃는다. 그래서 제목을 문서 본문의 첫 헤딩으로 남긴다. `new`와 `init`이 이미 그렇게 했고, `capture`와 `source`도 `--title`을 받으면 `# 제목`을 붙이도록 맞췄다. `index`의 `docTitle`이 본문 첫 헤딩을 우선해서 읽으므로 검색 순위도 이 값을 쓴다.

프론트매터 속성을 늘리지 않는 편을 택했다. 속성을 늘리면 프리셋 세 벌과 lint 규칙이 함께 늘고, ADR [0018](decisions/0018-taxonomy-field-names.md)이 정한 스키마가 흔들린다. 제목이 파일명과 헤딩 두 곳에 이미 있는데 세 번째 자리를 만들 이유가 없다.

`location.*` 두 규칙 중 `index.md`에 붙은 것은 의도된 차이다. ADR [0019](decisions/0019-index-documents-outside-the-gate.md)가 색인 문서를 게이트와 고아 검사 밖에 두기로 했다. 그러나 `inbox/wrong-stage.md`에 붙은 `location.stage-agreement`는 의도한 적이 없는 갭이다. engram은 그 문서를 `schema.allowed-value:artifact_stage`로만 잡아 위치 불일치를 별도로 보고하지 않는다.

### upstream 파서의 한계

| 대상 | 내용 |
|---|---|
| `inbox/crlf-meeting-note.md` | upstream이 `frontmatter.missing`으로 본다. **CRLF 줄바꿈 프론트매터를 인식하지 못한다.** engram은 정규화 후 정상 파싱한다 |
| `inbox/unclosed-frontmatter.md` | upstream이 닫히지 않은 프론트매터를 부분 파싱해 필드 누락 4건으로 보고한다. engram은 `frontmatter.unclosed` 하나로 보고한다 |

앞의 것은 engram이 옳다. 뒤의 것은 표현 방식의 차이이며 어느 쪽도 틀리지 않았다.

## engram만 잡은 것

| 규칙 | 건수 | 내용 |
|---|---|---|
| `frontmatter.missing-field:*` (`sources/`) | 4 | **upstream 스크립트가 `sources/`를 스캔 범위에 넣지 않는다** |
| `frontmatter.missing-field:*` (CRLF 문서) | 4 | 위 CRLF 파싱 차이의 뒷면. engram만 파싱에 성공해 필드를 검사했다 |
| `gate.min-wikilinks` | 2 | 승급 게이트. upstream은 `promotion-rules.md`에 규칙을 두지만 lint 스크립트가 검사하지 않는다 |
| `taxonomy.forms`, `taxonomy.topics` | 2 | 폐쇄 집합과 개방 집합 판정 |
| `graph.orphan` | 1 | 고아 문서 |
| `frontmatter.unclosed` | 1 | 위 표현 차이의 뒷면 |

**engram 고유 규칙이 오탐은 아니다.** 게이트와 taxonomy 판정은 upstream `meta/`의 규칙 문서에 정의되어 있으나 upstream의 lint 스크립트가 구현하지 않은 것이다. 규칙 명세와 스크립트의 구현이 어긋나 있고, engram은 명세 쪽을 따랐다.

## 비교 축 밖

`wiki.broad-topic`은 문서 단위가 아니라 위키 단위 진단이라 쌍 비교에 넣지 않는다. upstream에 대응 개념이 없다. 측정 시점에 2건 나왔다.

## resurface 선정 순위

둘째 축이다. 같은 위키에서 두 구현이 어떤 문서를 어떤 순서로 다시 꺼내는지 본다. 비교 단위는 후보 슬러그의 목록과 그 순서다. 점수 계산식이 달라도 순서가 같으면 같은 판정으로 본다.

측정 조건이 lint 축과 다르다.

| 항목 | 값 |
|---|---|
| upstream 커밋 | `8bc3f41` |
| 상태 | 양쪽 모두 빈 상태에서 시작하고 쓰지 않는다. upstream `--no-state`, engram `--dry-run` |
| 날짜 진실원 | upstream은 git 커밋 시각, engram은 프론트매터. 커밋 시각을 문서의 기준 날짜(`updated` 우선)로 맞춰 같은 사실을 가리키게 했다 |
| 후보 수 상한 | 6. 픽스처 context 문서 전부가 후보가 되는 값 |
| 프리셋 | 이 축은 임계값만 읽으므로 무관하다. 사본은 픽스처의 `personal` 그대로다 |

**비교가 성립한다.** 관문은 둘이었고 둘 다 통과했다. 첫째, upstream은 실행 시각을 고정할 수 없다. 커밋 시각을 고정해 순위가 초 단위 흔들림에 뒤집히지 않게 했다. 경과일 격차가 며칠 이상이므로 측정으로 안정성을 확인했다. 둘째, 상태 파일이다. upstream은 `meta/resurface-state.json`을 쓰지만 `--no-state`로 끄면 읽기만 하는 빈 상태로 돈다. engram은 `--dry-run`이 같은 역할을 한다.

### 결과

후보는 양쪽 모두 6건이고 집합이 같다. 순위는 6위만 같다. 2026-08-19 재측정 결과다.

| 순위 | upstream | engram |
|---|---|---|
| 1 | legacy-import-notes | sqlite-foreign-keys |
| 2 | sqlite-foreign-keys | markdown-link-syntax |
| 3 | markdown-link-syntax | legacy-import-notes |
| 4 | cli-flag-conventions | go-table-driven-tests |
| 5 | go-table-driven-tests | cli-flag-conventions |
| 6 | testing-pyramid | testing-pyramid |

### 차이의 이유

engram도 인바운드 가중을 따라갔다. [0066](decisions/0066-rediscovery-reads-inbound-links-and-ignores-bulk-commits.md)이 정렬을 upstream과 같은 식으로 바꿨다. 점수는 `경과일 * (1 + 1/(1 + 인바운드 링크 수))`이고 정렬 키는 점수 내림차순과 슬러그 오름차순 둘뿐이다. 제시 이력은 정렬 키에서 빠져 쿨다운 필터로 옮겼다. 그래서 이전 측정에서 5위였던 `legacy-import-notes`가 3위로 올라왔다.

**남은 차이는 인바운드를 세는 범위다.** 식은 같은데 대입하는 인바운드 값이 다르다.

| 슬러그 | upstream 인바운드 | engram 인바운드 | engram이 더 세는 것 |
|---|---|---|---|
| sqlite-foreign-keys | 1 | 1 | 없음 |
| markdown-link-syntax | 1 | 1 | 없음 |
| legacy-import-notes | 0 | 1 | `index.md`의 `related` |
| go-table-driven-tests | 2 | 3 | `index.md`의 본문 링크 |
| cli-flag-conventions | 2 | 5 | `inbox/` 문서 둘, `context/missing-stage.md` |
| testing-pyramid | 2 | 5 | `index.md`, `inbox/` 문서 하나, `sources/` 문서 하나 |

upstream `wiki_resurface.py`는 `context/`만 훑으므로 `index.md`와 `inbox/`와 `sources/`가 거는 링크를 세지 않는다. engram의 링크 그래프는 위키 전체를 보고 관계 필드도 링크로 센다([0065](decisions/0065-markdown-links-count-as-relations.md)). 그래서 engram의 인바운드가 항상 크거나 같고, 가중이 그만큼 작아진다. 이 차이는 upstream의 색인 경계 판단과 engram의 그래프 정의가 다른 데서 오므로 재발견 쪽에서 좁힐 문제가 아니다.

측정의 사소한 차이 하나. 커밋 시각을 그날 정오로 맞추는 바람에 upstream의 경과일이 engram보다 하루씩 크게 나온다. 모든 문서가 같은 방향으로 치우치므로 순위에는 영향을 주지 않는다.

### 이 축이 비교하지 않는 것

- 점수 값 자체. 식은 같아졌으나 인바운드를 세는 범위가 달라 값이 다르다. 위에서 설명했다
- upstream 스크립트가 함께 내는 bridges와 orphans. 별개 기능이다
- 제시 이력이 쌓인 상태의 재정렬. 빈 상태만 비교한다

## 다음

- 비교 축을 늘린다. 프론트매터 정규화는 `engram sync` 가 생겨 막힌 자리가 풀렸고 붙이는 일만 남았다. eject 산출물은 upstream 에 대응하는 것이 없어 성립하지 않는다.
- `upstream-sync.py --check` 가 규칙 명세 파일을 `갱신` 으로 내면 이 비교를 다시 돌린다([0094](decisions/0094-engram-watches-its-own-dependency-surface.md)).
