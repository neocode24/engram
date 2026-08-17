# 1.0 교차 검증 보고

> 2026-08-17. 개발을 맡은 세션과 다른 세션, 다른 모델이 저장소 `bbe07bc` 시점의 코드를 처음부터 빌드하고 커맨드 스물여덟을 실제로 돌려 확인한 결과다. 정식 검증(`go test ./...`)과 별개로 사람이 손으로 여정을 돌리며 찾은 것을 적는다. [windows-verification.md](windows-verification.md)와 같은 성격의 문서다.

## 한 줄 요약

**정식 검증은 전부 통과했고 설계 문서가 주장하는 경계(게이트 단일 사유, MCP 쓰기 도구 하나, serve와 export의 노출 규칙, 결정론)는 실제 동작과 일치한다.** 손으로 돌려서 결함 셋과 설계 빈틈 셋을 찾았다. 셋 중 하나(슬러그 정규화 충돌)는 `promote`의 기본 경로에서 생기고 `mv`가 그 위에서 링크를 잘못 고치므로 릴리스 전에 닫는 편이 좋다.

## 범위와 방법

| 항목 | 내용 |
|---|---|
| 대상 | 저장소 `bbe07bc` (`fix: 파이썬이 소스를 읽을 때 인코딩을 명시한다`). `.git`, `private/`, 이미지 원본을 뺀 소스 전체 |
| 환경 | Linux amd64. Go 1.26.6을 소스에서 빌드해 썼다. `golang.org/x/*`와 `go.yaml.in`은 사내망처럼 막힌 환경이라 GitHub 미러로 `replace`했다. 검증용 변경이며 저장소에는 넣지 않았다 |
| 정식 검증 | `go build ./...`, `go vet ./...`, `go test ./...` (harness 여섯 포함), `go test -race ./internal/...`, 교차 빌드 windows/amd64, darwin/arm64, linux/arm64 (`CGO_ENABLED=0`), `scripts/check-adr.py` |
| 기능 검증 | 빈 디렉토리에서 위키 넷을 만들어 커맨드 스물여덟을 전부 실행. README의 5분 시나리오, 게이트 유예, sources 파생, demote, archive 링크 유지, mv, update, lint 규칙 16종 유발, 프리셋 상향과 하향 migrate, git 위 sync, eject 산출물과 Python 린터 판정 대조, skills install, MCP stdio 핸드셰이크와 도구 호출, serve 노출 범위와 경로 탈출과 POST, team 프리셋 sensitivity 제외, export 치환, `--json` 전 조회 커맨드 5회 반복 해시 비교 |
| 검증하지 않은 것 | Homebrew tap 갱신과 실제 릴리스 워크플로(비공개 저장소라 실행 불가), Windows 콘솔 직결 항목(기존 문서와 같은 상태), upstream 동등성(`ENGRAM_UPSTREAM` 없음), 시맨틱 검색 층(없음) |

## 통과한 것

- `go build`, `go vet` 깨끗하다. `go test ./...` 전 패키지 ok, `-race` 깨끗하다. `harness/journey`, `harness/eject`, `harness/examples`, `harness/realdata`가 실제 바이너리로 통과한다.
- README의 5분 시나리오는 출력 한 글자까지 실제와 같다.
- 게이트: 거절 사유는 `min_wikilinks` 하나다. 빈 위키에서 유예되고(`gate.deferred`), 대상이 늘면 켜진다. `context/` 문서를 다시 `promote`하면 거절한다. 도착지에 문서가 있으면 덮어쓰지 않는다.
- `promote`: `inbox/`는 이동하고 `sources/`는 파생을 만들며 원본은 그대로 남는다. `derived_from`과 `derived_context`가 양방향으로 기록된다.
- `demote`는 파생 문서를 내릴 때 원본의 `derived_context`가 어긋난다고 알린다. `archive`는 슬러그를 유지하고 들어오는 링크 수를 알린다. `mv --dry-run`이 바꿀 것을 미리 보여 준다.
- lint 규칙 16종이 전부 유발되고 메시지마다 고치는 법이 붙는다. `location.stage-agreement`는 방향에 따라 error와 warn으로 갈린다(ADR 0035). `sources.updated`, `taxonomy.forms`(error), `taxonomy.topics`(warn), `body.max-lines`, `link.broken`, `frontmatter.*`, `schema.allowed-value` 확인.
- `migrate`: 프리셋 상향은 기본값을 채우고, 하향은 값이 있는 필드를 `--force` 없이는 지우지 않으며 그동안 lint가 `schema.axis-off`로 알린다. `sync`는 git이 없으면 거절하고 있으면 dry-run이 기본이다.
- `eject`: 파일 아홉을 만들고 이미 있으면 `--force` 없이 덮지 않는다. 내보낸 `scripts/lint-frontmatter.py`가 정상 위키에서 0, 깨진 프론트매터에서 1, 게이트 미달에서 reject를 낸다. `engram lint`와 같다.
- MCP: 도구 열. 쓰기는 `capture` 하나이고 `inbox`에만 쓴다. `promote`를 부르면 `unknown tool`. 도구는 `wiki` 인자를 받지 않는다(`additional properties` 거절). `resurface`를 MCP로 불러도 제시 이력 파일이 바뀌지 않는다.
- serve: `context/`와 `index.md`만 200이고 `inbox/`, `sources/`, `archive/`는 404다. `/doc/../engram.yaml`, `%2e%2e` 경로 탈출 404. POST 405. team 프리셋에서 `sensitivity: restricted` 문서는 404이고 시작 로그에 제외 수가 찍힌다. serve의 검색은 제외 문서를 내지 않는다.
- export: 같은 노출 규칙을 쓰고, 치환 파일이 파일명과 본문에 함께 적용되며, 번들 밖을 가리키는 링크를 알린다.
- 결정론: `--now`를 고정하면 lint, status, digest, resurface --dry-run, bridge, search, recall, backlinks, rules show, doctor, migrate, export --dry-run, eject --dry-run의 `--json` 출력이 반복 실행에서 바이트 단위로 같다.
- `skills install`은 감지 실패 시 디렉토리를 만들지 않고 `--dir`을 안내한다. `--now` 형식 오류, 없는 위키, 없는 문서 경로에서 다음 행동이 붙은 에러가 난다.

## 찾은 것

심각도는 셋이다. **높음**은 데이터를 잘못 고치거나 핵심 약속이 깨지는 것, **중간**은 동작이 문서나 이름과 다른 것, **낮음**은 다듬을 것이다.

### 1. 슬러그 정규화 충돌과 mv의 오기록 (높음)

`graph.Normalize`가 비교 키에서 날짜 접두사를 뗀다. 그래서 `sources/2026-08-게이트웨이-벤치마크.md`, `context/게이트웨이-벤치마크.md`, `inbox/2026-08-17-게이트웨이-벤치마크.md`가 모두 `게이트웨이-벤치마크` 하나로 뭉친다. **`promote`가 `sources/` 문서를 기본 슬러그(파일명에서 접두사를 뗀 값)로 파생시키면 이 충돌이 기본 경로에서 생긴다.** 원본은 남으므로 늘 두 문서가 같은 키를 갖게 된다.

재현. 빈 위키에서

```
engram source --title "게이트웨이 벤치마크" --created 2026-08 "원문"
engram promote sources/2026-08-게이트웨이-벤치마크.md --type concept --related index --related <다른 문서>
engram capture --title "게이트웨이 벤치마크" "메모"       # inbox 에 하나 더
engram mv 게이트웨이-벤치마크 벤치마크-원본
```

결과.

- `mv`는 순회 순서에서 처음 만나는 문서(`archive/`, `context/`, `inbox/`, `sources/` 순)를 옮긴다. 사용자가 어느 문서를 뜻했는지 묻지 않는다.
- 다른 문서에서 **원본 source를 가리키던** `[[2026-08-게이트웨이-벤치마크]]`가 `[[벤치마크-원본]]`으로 바뀌어 context 문서를 가리키게 된다. 링크의 대상이 조용히 바뀐다.
- 옮긴 context 문서 자신의 `derived_from: sources/2026-08-게이트웨이-벤치마크.md`가 `sources/2026-08-벤치마크-원본.md`로 바뀐다. 존재하지 않는 파일이다. lint는 관계 필드의 경로 존재를 검사하지 않아 이 파손을 잡지 못한다.
- `backlinks 게이트웨이-벤치마크`가 세 문서를 향한 링크를 한데 섞어 낸다.
- 충돌을 알리는 lint 규칙이 없다. `error 0`인 채로 위 상태가 유지된다.

제안. 셋 중 하나 이상.

1. 정규화 키가 같은 문서가 둘 이상이면 lint가 error를 낸다(`slug.duplicate`). 도구가 만든 위키가 lint를 통과해야 한다는 불변식과 맞물리므로, `promote`가 `sources/`에서 파생할 때 충돌하는 기본 슬러그를 거절하고 `--slug`를 요구하거나 접미사를 붙인다.
2. `mv`는 옛 슬러그가 둘 이상에 걸리면 거절하고 경로로 지정하게 한다.
3. 장기적으로는 `[[2026-08-슬러그]]`처럼 접두사를 포함한 링크를 정확 일치로 먼저 풀고, 접두사 없는 링크만 정규화로 푼다.

### 2. mv가 bare 슬러그 관계 필드에 `.md`를 붙인다 (중간)

충돌과 무관하게 재현된다. `sources/2026-08-원본.md`를 `--slug 결론`으로 파생시킨 뒤 `engram mv 결론 최종-결론`을 하면 원본의 `derived_context`가 `최종-결론`이 아니라 **`최종-결론.md`**가 된다. `replacedValue`가 `[[` 로 시작하지 않는 값을 전부 경로로 보고 `.md`를 붙인다. `graph.Normalize`가 `.md`를 떼므로 링크는 계속 풀리지만, 도구가 자기 규칙(`derived_context`는 bare 슬러그)과 다른 형식을 쓰게 된다. `derived_context`와 `derived_from`의 형식을 구분해 고치면 된다.

### 3. sources 파생 시 원본의 `derived_context`가 새 문서로 복사된다 (중간)

같은 원본에서 두 번 파생하면 두 번째 context 문서가 첫 번째 파생 문서를 자기 `derived_context`로 갖고 태어난다. 원본의 프론트매터를 통째로 물려받기 때문이다. `tags` 같은 값도 함께 넘어온다. 파생 문서의 `derived_context`는 빈 목록으로 시작해야 한다.

### 4. `indexable` 필드는 쓰기만 하고 아무도 읽지 않는다 (중간, 설계)

`capture`가 `false`, `promote`가 `true`를 쓰지만 `internal/index`, `search`, `recall`, `serve`, `export`, `expose` 어디에도 이 필드를 읽는 코드가 없다. `sources/` 문서는 `indexable: false`인데 `search`에 정상적으로 나온다(README의 5분 시나리오가 그렇다). spec-map 6절이 "lint가 검사하지 않는다"고만 적었는데 실제로는 **검색도 존중하지 않는다.** 이름이 동작과 다르므로 셋 중 하나를 고른다. 색인이 `indexable`을 존중하게 하거나, 필드를 빼거나, 문서에 "예약 필드"라고 명시한다.

### 5. 게이트가 세는 링크의 자격 (중간, 설계)

게이트는 고유 위키링크 수만 센다. 그래서 다음이 전부 통과한다.

- 존재하지 않는 문서 둘: `--related 없는1 --related 없는2` (경고만 내고 통과)
- 자기 자신 링크 하나와 index 하나: 본문에 `[[자기]]`, `--related index`
- inbox 문서를 가리키는 링크

ADR 0023이 게이트의 **대상 수**에서 inbox를 뺐지만 **세는 링크**에서는 아무것도 빼지 않는다. "연결 없는 고립 노드만 막는다"는 약속이 자기 링크로 충족되는 것은 의도가 아닐 것이다. 최소한 자기 링크는 빼고, 없는 대상을 세는 관행은 ADR로 명시하거나 바꾼다.

### 6. 한글 파일명의 유니코드 정규화 (낮음에서 중간, macOS)

파일명이 NFD로 저장된 `한글문서.md`를 `[[한글문서]]`(NFC)로 가리키면 `link.broken`이 나고 `backlinks`가 못 찾는다. 코드에 정규화가 없다. macOS APFS는 입력 형태를 보존하므로 engram이 만든 파일은 문제가 없지만, 다른 도구가 만든 NFD 파일이나 오래된 볼륨에서 온 파일이 섞이면 한글 슬러그가 조용히 끊긴다. 슬러그와 링크를 비교하는 자리에서 NFC로 맞추면 된다.

### 7. 다듬을 것 (낮음)

- `update --set tags=[a,b]`가 `["[a", "b]"]`를 쓴다. 문서화된 문법은 `tags=a,b`이지만 대괄호를 벗기거나 거절하는 편이 안전하다.
- 유예로 올라간 문서는 위키가 자라면 lint에서 `reject`로 바뀐다(ADR 0021대로다). 그런데 `status`의 다음 행동이 `reject`를 계기로 삼지 않아 사용자가 알기 어렵다. roadmap의 알려진 빈틈이며 1.0에서 닫히지 않았다.
- 위키 경로를 받는 방식이 커맨드마다 다르다. `engram resurface <경로>`는 `unknown command`다. roadmap이 0.4에서 정한다고 했으나 남아 있다.
- 사용자 대면 에러 중 보고체가 남아 있다. 예: `--now 값이 RFC3339 형식이 아님`. ADR 0027의 경어체 규칙 대상이다. `~없음: %w` 형태의 감싸기 접두는 명사구라 대상이 아니라고 볼 수 있으나 최상위 메시지로 그대로 노출되는 것이 99건이다.
- `body.max-lines`가 프론트매터 줄을 포함해 센다. 의도라면 메시지에 밝힌다.
- 강의 덱(`docs/course/index.html`)이 커맨드 스물여섯, `serve` 설계만 확정, `pack` 미구현 시점의 서술이다. 스물여덟, `serve` 완료, `export`로 현행화해야 한다.

## 다시 확인할 것

결함 1과 2, 3을 고친 뒤 아래를 하니스에 넣으면 같은 계열의 회귀를 막을 수 있다.

- 같은 원본에서 기본 슬러그로 파생한 뒤 `mv`를 돌리고, 원본을 가리키던 링크와 `derived_from` 경로가 그대로인지 단언한다.
- `derived_context`가 `.md` 없이 남는지 단언한다.
- 두 번째 파생 문서의 `derived_context`가 비어 있는지 단언한다.
- NFD 파일명 픽스처 하나를 골든 위키에 둔다.
- 자기 링크만 있는 문서가 게이트에서 거절되는지 단언한다.

## 판정

1.0의 커맨드 스물여덟은 문서가 말하는 대로 동작하고 경계는 지켜진다. 결함 1은 데이터를 잘못 고치는 경로가 기본 사용 흐름 안에 있으므로 첫 릴리스 전에 닫기를 권한다. 나머지는 1.0.x에서 다뤄도 된다.
