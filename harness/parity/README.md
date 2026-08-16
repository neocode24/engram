# parity

ADR 0029 가 정한 upstream llm-wiki 스크립트와 engram 구현의 parity 비교
러너다. 비교 축은 ADR 0005 가 넷을 들었으나 지금 여기에 있는 것은
**lint 위반 목록 하나**다. 나머지 축(resurface 선정 순위, 프론트매터
정규화, eject 산출물)은 이 골격이 검증된 뒤에 붙는다.

## 무엇을 비교하는가

upstream `scripts/lint-frontmatter.sh --include-inbox` 와 `engram lint` 를
같은 픽스처 위키(`../fixtures/golden-wiki`, 골든 러너와 공유)에 돌려
위반 목록을 **(경로, 정규화 규칙)** 쌍 단위로 비교한다.

두 구현의 출력 형식은 완전히 다르다. upstream 은 `FAIL 경로: 메시지`
줄을 내고 engram 은 규칙 ID 와 등급을 가진 위반 목록을 낸다. 그래서
양쪽을 normalize.go 의 매핑 표로 공통 규칙 이름에 옮긴 뒤 비교한다.

## 어떻게 돌리는가

```
ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/... -v
```

- `ENGRAM_UPSTREAM` 이 없으면 skip 한다. **CI 는 항상 skip 이다.**
  upstream 저장소가 비공개라 공개 저장소의 CI 에서는 볼 수 없고, 비교
  시점도 매 커밋이 아니라 upstream 계약이 바뀔 때 사람이 정한다(ADR 0029).
- bash 가 없거나 `<루트>/scripts/lint-frontmatter.sh` 가 없으면 이유와
  함께 skip 한다.
- 종료 코드는 0(통과), 1(위반 있음) 둘 다 정상 출력으로 본다. 그 밖의
  종료 코드는 실패로 취급한다.
- 결과는 `t.Log` 로만 남는다. 사람이 `docs/parity.md` 에 옮겨 적는다.
  자동 갱신이 아니다.

### 실행 방식

upstream 스크립트는 위키 루트를 인자로 받지 않고 자기 파일 위치의 상위
디렉토리(`scripts/..`)를 루트로 본다. 그래서 러너는 임시 디렉토리에
픽스처를 통째로 복사하고 그 안의 `scripts/` 아래에 스크립트 사본을 둔 뒤
돌린다. engram lint 는 같은 임시 사본에 대해 `lint.Run` 을 직접
호출한다. 바이너리를 exec 하지 않는 이유는 골든 러너와 같다.

스크립트가 픽스처의 `engram.yaml` 을 읽지 않는다는 점도 실측으로
확인했다. 허용값과 필수 필드는 스크립트 안에 하드코딩되어 있다. 그래서
임시 디렉토리 조정만으로 그대로 돌고, 픽스처를 upstream `meta/` 에
맞춰 바꿀 필요가 없다.

## 결과 읽기

비교 결과는 세 갈래로 나온다. **차이가 나도 테스트는 실패하지 않는다.**
지금은 차이를 측정하는 단계다. 실패 조건은 양쪽 모두 위반 0건, 곧 비교가
성립하지 않은 경우뿐이다.

| 갈래 | 뜻 |
|---|---|
| 양쪽 다 잡음 | 일치 |
| upstream 만 잡음 | engram 이 놓친 규칙 |
| engram 만 잡음 | engram 이 더 엄격하거나 오탐 |

같은 쌍이 양쪽에 있되 개수가 다르면(예: 깨진 위키링크가 한 문서에 여러
개) 그 사실을 함께 적는다.

### 2026-08 측정값

골든 위키 픽스처에서는 upstream 53건, engram 10건이었다. 일치 2쌍
(`frontmatter.missing`, `schema.allowed-value:artifact_stage`), upstream
만 51쌍, engram 만 7쌍. upstream 만 잡은 것의 대부분은 픽스처가
`scope`, `sensitivity`, `trigger_mode`, `workflow` 필드를 갖지 않는
스키마 세대 차이다. engram 은 education 프리셋에서 이 축들을 요구하지
않는다. 매핑 표에 없는 규칙(unmapped)은 0건이었다.

## 무엇을 비교하지 않는가

- **줄 번호.** 두 구현이 같은 줄을 가리키리라는 보장이 없다.
- **등급.** upstream 의 FAIL/WARN 과 engram 의 error/warn/reject 는
  대응을 세우지 않는다. 같은 규칙이라도 등급이 갈릴 수 있다.
  `sources.updated` 는 upstream 이 FAIL, engram 이 warn 이다.
- **위키 단위 진단.** `wiki.broad-topic` 은 파일 위반이 아니라 위키
  통계 판정이라 쌍 비교에서 빼고 로그에 따로 남긴다.
- **스캔 범위 차이.** upstream 은 `context/`, `agents/workflows/`,
  `sources/manifests|transcripts|summaries/`, `meta/templates/`,
  `index.md`(그리고 `--include-inbox` 로 `inbox/`)만 보고 모든 경로의
  `README.md` 를 뺀다. engram 은 `page_dirs`(inbox, sources, context,
  archive)와 `root_files` 를 본다. 그래서 `sources/` 바로 아래 문서는
  engram 만 검사한다. 픽스처의 `sources/tech-talk-summary.md` 는 양쪽
  모두 위반이 없어 지금은 왜곡이 없다. 문서를 늘릴 때 이 차이를 기억한다.
  stderr 에 find 오류가 남는 것은 픽스처에 없는 스캔 루트 때문이고
  정상이다.
- **CRLF.** upstream 스크립트는 첫 줄이 정확히 `---` 인지 바이트로
  보므로 CRLF 문서를 프론트매터 없음으로 판정하고, engram 은 CRLF 를
  정규화한 뒤 판정한다. 픽스처의 `crlf-meeting-note.md` 는 이 차이의
  측정 사례로 upstream 만 잡힌다.

## 정규화 매핑 표

normalize.go 의 표와 같은 내용이다. 표를 고치면 이 문서도 함께 고친다.
"대응 없음"은 매핑이 실패했다는 뜻이 아니라, 그 정규화 이름을 이쪽
구현이 내지 않는다는 뜻이다. 짝이 없는 이름은 "한쪽만 잡음"으로
측정된다. **표에 없는 이름은 `unmapped:` 접두사와 함께 결과에 남는다.**
조용히 버리지 않는다.

| 정규화 규칙 | upstream 메시지(접두사) | engram 규칙 |
|---|---|---|
| `frontmatter.missing` | `missing frontmatter block at top of file` | `frontmatter.missing` |
| `frontmatter.malformed` | `empty or malformed frontmatter block` | 대응 없음 |
| `frontmatter.unclosed` | 대응 없음 | `frontmatter.unclosed` |
| `frontmatter.yaml` | 대응 없음 | `frontmatter.yaml` |
| `frontmatter.missing-field:<필드>` | `missing required field 'F'`, `source artifact requires created/sourced_at`, `context artifact missing title`, `<단계> artifact missing source_refs/related/derived_context` | `frontmatter.missing-field`(메시지에서 필드 추출) |
| `schema.allowed-value:<필드>` | `invalid <필드> '<값>'` | `schema.allowed-value`(메시지에서 필드 추출) |
| `schema.retired-field:quality_level` | `quality_level is retired` | 대응 없음 |
| `schema.retired-field:review_after` | `review_after is retired` | 대응 없음 |
| `schema.axis-off:<축>` | 대응 없음 | `schema.axis-off`(메시지에서 축 추출) |
| `sources.updated` | `source artifact must not carry updated` | `sources.updated` |
| `location.stage-agreement` | `context/mocs documents must use artifact_stage index`, `context documents must use artifact_stage context`, `agent workflows must use artifact_stage agent-workflow`, `source artifacts must use artifact_stage source`, `inbox documents must use artifact_stage inbox`, `root index must use artifact_stage index` | 대응 없음 |
| `location.type-agreement` | `context/mocs documents must be type moc`, `root index must be type moc` | 대응 없음 |
| `indexable.policy` | `inbox artifacts must be indexable false`, `source artifacts are normally indexable false`, `context artifacts are normally indexable true` | 대응 없음 |
| `taxonomy.forms` | 대응 없음 | `taxonomy.forms` |
| `taxonomy.topics` | 대응 없음 | `taxonomy.topics` |
| `body.max-lines` | 대응 없음 | `body.max-lines` |
| `link.broken` | 대응 없음 | `link.broken` |
| `graph.orphan` | 대응 없음 | `graph.orphan` |
| `gate.deferred` | 대응 없음 | `gate.deferred` |
| `gate.min-wikilinks` | 대응 없음 | `gate.min-wikilinks` |
| `wiki.broad-topic` | 대응 없음 | 위키 단위 진단, 쌍 비교 밖 |

필드 단위 규칙(`frontmatter.missing-field`, `schema.allowed-value`)에
필드 이름을 붙이는 이유는 정규화가 차이를 감추지 않기 위해서다. 규칙
이름을 뭉뚱그리면 한쪽만 잡은 위반이 같은 규칙으로 보인다.

## 디렉토리

| 경로 | 역할 |
|---|---|
| `parity_lint_test.go` | lint 축 비교 러너 |
| `normalize.go` | 위반 정규화와 매핑 표 |
