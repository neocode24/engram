# examples/

`engram` 커맨드를 순서대로 돌려서 만든 데모 위키. 조직 정보가 없는 깨끗한 예제다.

**이 디렉토리는 생성물이다.** 손으로 고치지 않는다. 내용을 바꾸려면 생성 시퀀스를 고치고 재생성한다. 재생성 결과가 커밋된 내용과 다르면 회귀로 간주한다. 즉 이 디렉토리는 예제인 동시에 테스트다.

| 항목 | 위치 |
|---|---|
| 생성 시퀀스 | `harness/examples/examples_test.go`의 `buildDemo` |
| 재생성 | `go test ./harness/examples -update` |
| 회귀 검사 | `go test ./harness/examples` |

생성 주체는 `init` 하나가 아니라 커맨드 시퀀스다. `init`만 돌리면 파일 셋짜리 빈 위키가 나와서 승급도 게이트도 링크도 보이지 않는다. `--now`를 고정하므로 몇 번을 돌려도 같은 결과가 나온다.

## 무엇이 들어 있나

`personal` 프리셋으로 만든 위키에 `context/` 문서 스물다섯이 있다.

| 단계 | 문서 | 왜 여기 있나 |
|---|---|---|
| `context/` | `promotion-pipeline`, `wikilink-graph` | `new`로 처음부터 검수된 지식으로 쓴 문서 |
| `context/` | `reading-note` | `inbox`에서 승급한 문서. 원본이 남지 않고 옮겨졌다 |
| `context/` | `km-primer` | `sources`에서 파생한 문서. 원본은 그대로 있다 |
| `context/` | 나머지 스물하나 | `examples/materials/`의 실습 재료에서 승급한 문서 |
| `sources/` | 다섯 | 보존되는 원본. 본문을 고치지 않는다 |
| `inbox/` | `2026-03-02-talk-note` | **일부러 남긴 미처리 메모** |
| 루트 | `index.md` | 색인 문서 |

**문서를 스물다섯까지 늘린 것은 재발견 때문이다.** 문서가 다섯인 위키에서는 `bridge`가 볼 쌍이 열 개뿐이고 `resurface`는 후보가 없다. 재발견은 위키가 커질수록 값이 나오는 기능이라 작은 위키로는 무엇을 하는지 보여줄 수 없다.

`examples/materials/`의 재료 스물다섯이 어떤 길로 들어와 어떤 문서가 되는지는 `harness/examples/examples_test.go`의 `materialDocs` 표에 있다. `created`가 붙은 넷은 남의 글이라 `source`로 들어와 파생을 만들고 원본이 남는다. 나머지는 우리 것이라 `capture`로 들어와 옮겨진다.

`inbox`에 하나를 남긴 것은 의도다. 밀린 것이 0인 위키로는 `status`가 무엇을 보는지, `lint`가 고아 문서를 어떻게 잡는지 보여줄 수 없다. **`error`와 `reject`는 0이다.**

**링크를 일부러 성기게 깔았다.** 문서마다 `[[index]]` 하나와 이웃 하나만 본문에서 가리킨다. 서로 다 이어 두면 `bridge`가 찾을 것이 남지 않는다. 이어져 있지 않은데 닮은 쌍을 찾는 것이 그 커맨드이기 때문이다. 그 대가로 `lint`의 `graph.orphan` 경고가 여럿 나오는데, 그 상태가 곧 재발견이 필요한 상태다.

## 돌려 보기

```
engram reindex examples/personal     # 색인을 만든다. .engram/ 은 커밋되지 않으므로 한 번 필요하다
engram status examples/personal
engram lint examples/personal
engram search --wiki examples/personal 승급
engram backlinks --wiki examples/personal promotion-pipeline
engram serve --wiki examples/personal
engram bridge --wiki examples/personal      # 재발견. 모델이 있으면 의미 축도 돈다
```

`bridge`는 모델이 없어도 단어 축으로 돈다. `engram model pull`로 모델을 받아 두면 의미 축이 함께 돌고, **낱말이 겹치지 않는데 같은 이야기를 하는 쌍**을 찾는다. 첫 실행은 문서 스물다섯을 인코딩하므로 이 기계에서 2분 45초가 걸렸고, 벡터가 `.engram/vectors.json`에 남아 다음부터는 즉시 끝난다.

`serve`를 띄우면 `context/`와 색인 문서만 보인다. `inbox/`와 `sources/`는 목록에도 URL에도 나오지 않는다. 승급 파이프라인이 웹에서도 그대로 지켜지는지 여기서 확인할 수 있다.

## 이 디렉토리가 아닌 것

검증용 골든 위키는 여기가 아니라 `harness/fixtures/`에 둔다. 그쪽은 upstream 스크립트와 Go 구현의 출력을 비교하기 위한 고정 입력이며 사람이 의도적으로 관리한다.

근거: [../docs/decisions/0011-repo-layout-and-module-name.md](../docs/decisions/0011-repo-layout-and-module-name.md)
