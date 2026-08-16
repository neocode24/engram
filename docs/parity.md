# upstream parity

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
| 반영 | ADR 0031(`location.stage-agreement`)까지 |
| upstream 커밋 | `05f7279` |
| 픽스처 | `harness/fixtures/golden-wiki`, 문서 13개 |
| 프리셋 | `team` |
| 비교 축 | lint 위반 목록 |

**프리셋을 `team`으로 올려서 비교한다.** upstream 스크립트는 `engram.yaml`을 읽지 않고 자기 스키마를 하드코딩하며, 그 스키마는 `scope`, `sensitivity`, `source_channel`, `trigger_mode`, `workflow`를 전부 요구한다. 픽스처의 기본 프리셋인 `education`은 그 축들을 끄므로, 맞추지 않으면 비교 결과가 축 on/off 차이로 뒤덮여 실제 규칙 차이가 묻힌다. 실제로 `education`에서 error 4건이던 것이 `team`에서 48건이 되고 늘어난 44건이 전부 그 필드 누락이었다.

## 결과

| 갈래 | 쌍 |
|---|---|
| 양쪽 다 잡음 | **39** |
| upstream만 잡음 | 14 |
| engram만 잡음 | 15 |
| 정규화 매핑 없음 | 0 |

비교 단위는 (경로, 정규화한 규칙 이름) 쌍이다. 줄 번호는 넣지 않는다. 두 구현이 같은 위반을 다른 줄에서 가리킬 수 있기 때문이다.

## upstream만 잡은 것

engram이 따라가야 할 후보다. 셋으로 갈린다.

### 남은 차이

전부 의도했거나 판단이 끝난 것이다. 첫 측정에서 나온 진짜 갭 둘은 닫혔다.

| 규칙 | 건수 | 내용 |
|---|---|---|
| `frontmatter.missing-field:title` | 7 | **의도된 차이다.** 아래에서 설명한다 |
| `location.stage-agreement` | 1 | `index.md` 하나만 남았다. ADR [0019](decisions/0019-index-documents-outside-the-gate.md)에 따른 의도된 차이다 |
| `location.type-agreement` | 1 | **따라가지 않는다.** 적용 자리가 upstream에만 있는 `moc` 타입과 `index` 단계뿐이다 |

`inbox/wrong-stage.md`의 `location.stage-agreement`는 ADR [0031](decisions/0031-location-must-agree-with-stage.md)로 닫았다. 이제 양쪽이 함께 잡는다. `index.md`에 붙은 둘은 upstream이 색인 문서에 `moc` 타입과 `index` 단계를 요구하기 때문이며, engram에는 그 타입도 그 단계도 없다.

#### title은 따라가지 않는다

upstream 스크립트가 이 규칙의 이유를 주석에 적어 두었다.

> context 노드는 사람이 읽는 한글 제목을 title에 둔다. 파일명은 링크 안정성 때문에 영어 slug로 유지하므로, title이 없으면 index.md가 slug로만 보인다.

**upstream은 파일명을 영어 슬러그로 유지하기 때문에 한글 제목이 갈 곳이 없다.** 그래서 프론트매터에 둔다.

engram은 ADR [0020](decisions/0020-slug-and-filename-rules.md)에서 반대로 정했다. 한글 슬러그를 보존하고 음차하지 않는다. 한글 제목이 곧 파일명이므로 upstream이 `title`을 필요로 하는 이유가 성립하지 않는다.

다만 슬러그는 되돌려도 원문이 아니다. 공백이 하이픈이 되고 대소문자와 구두점을 잃는다. 그래서 제목을 문서 본문의 첫 헤딩으로 남긴다. `new`와 `init`이 이미 그렇게 했고, `capture`와 `source`도 `--title`을 받으면 `# 제목`을 붙이도록 맞췄다. `index`의 `docTitle`이 본문 첫 헤딩을 우선해서 읽으므로 검색 순위도 이 값을 쓴다.

프론트매터 축을 늘리지 않는 편을 택했다. 축을 늘리면 프리셋 세 벌과 lint 규칙이 함께 늘고, ADR [0018](decisions/0018-taxonomy-field-names.md)이 정한 스키마가 흔들린다. 제목이 파일명과 헤딩 두 곳에 이미 있는데 세 번째 자리를 만들 이유가 없다.

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

**engram 고유 규칙이 오탐은 아니다.** 게이트와 taxonomy 판정은 upstream `meta/`의 규칙 문서에 정의되어 있으나 upstream의 lint 스크립트가 구현하지 않은 것이다. 문서의 계약과 스크립트의 구현이 어긋나 있고, engram은 문서 쪽을 따랐다.

## 비교 축 밖

`wiki.broad-topic`은 문서 단위가 아니라 위키 단위 진단이라 쌍 비교에 넣지 않는다. upstream에 대응 개념이 없다. 측정 시점에 2건 나왔다.

## 다음

- 비교 축을 늘린다. `resurface` 선정 순위가 다음이다. 나머지 둘은 해당 커맨드가 생길 때.
- upstream `meta/CHANGELOG.md`에 `binary-affecting` 항목이 붙으면 이 비교를 다시 돌린다.
