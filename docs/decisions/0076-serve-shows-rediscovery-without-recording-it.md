---
number: 0076
title: serve는 재발견을 기록 없이 보여준다
date: 2026-08-21
status: accepted
---

# serve는 재발견을 기록 없이 보여준다

## 배경

[0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 `serve`의 노출 범위를 정하며 한 줄을 남겼다.

> 재발견 커맨드를 노출하지 않는다. 상태를 쓰거나 사람의 판단 기록을 건드린다([0028](0028-rediscovery-state-and-boundaries.md)).

그 판단의 근거는 옳았다. `resurface`는 제시 이력을 쓰고 `bridge --reject`는 기각 쌍을 위키에 남긴다. 인증 없는 네트워크 표면에 그것을 두면 읽기 전용 계약이 깨진다.

그러나 이 배제는 커맨드의 성질을 화면에 그대로 옮긴 것이다. **커맨드가 상태를 쓴다는 것과 화면이 상태를 써야 한다는 것은 다른 명제다.**

실측으로 확인했다. `resurface.Run`은 `dryRun` 인자를 이미 갖고 있고 참이면 제시 이력을 쓰지 않는다. `bridge.Run`은 기각 목록을 입력으로만 읽는다([0028](0028-rediscovery-state-and-boundaries.md)이 "기각 목록은 입력이지 출력이 아니다"라고 규정했다). 벡터는 `embed.Cached`가 캐시를 읽기만 한다. **셋 다 읽기 경로가 이미 있다.**

배제를 유지할 때 잃는 것도 실측했다. upstream은 `scripts/wiki_resurface.py --json`을 별도 렌더러에 물려 카드로 낸다. engram은 `resurface --json`과 `bridge --json`으로 같은 페이로드를 내므로 같은 렌더러가 붙지만, **바이너리만 받은 사람은 터미널 텍스트가 전부다.** 임베딩이 만든 쌍이 어디에 쓰이는지 눈으로 볼 자리가 없다.

## 판단 근거

### 화면은 제시가 아니다

`resurface`의 제시 이력은 "이 문서를 사용자 앞에 내놓았다"는 기록이고 그 기록의 쓸모는 21일 냉각이다([0028](0028-rediscovery-state-and-boundaries.md)). 같은 문서가 매일 나오지 않게 한다.

웹 화면을 여는 것은 그 의미의 제시가 아니다. **새로고침 한 번이 냉각을 소모하면 화면을 두 번 보는 것만으로 재발견 큐가 비어 버린다.** 기록하지 않는 것이 옳고, 기록하지 않으므로 읽기 전용 계약과 충돌하지 않는다.

대가는 CLI와 화면이 다른 후보를 낼 수 있다는 것이다. 받아들인다. 화면은 훑는 자리이고 CLI가 판단을 남기는 자리다. 두 자리의 성격이 다르다는 것이 이 결정의 내용이다.

### 계산하지 않는다

임베딩 계산은 문서당 12.6초다([0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md)). 요청 처리 중에 계산하면 문서 스물다섯인 위키에서 첫 요청이 2분 45초 걸리고 그동안 서버가 멈춘다.

**캐시에 있는 것만 쓴다.** 캐시를 채우는 것은 `bridge` 커맨드의 몫이다([0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md)가 계산 자리를 `bridge`로 정했고 MCP도 같은 규율을 따른다). 캐시가 비어 있으면 단어 축만으로 돈다.

**단어 축만 돌았다는 사실을 화면에 적는다.** 조용히 축소하면 읽는 사람은 의미 축까지 돌았다고 믿는다. 이 화면의 쓸모 절반이 의미 축이므로 그 부재는 알려야 한다.

### 노출 범위를 다시 거른다

`resurface`와 `bridge`는 위키 전체를 본다. 화면에서 다시 거르지 않으면 `private-local-only` 문서의 제목과 경로가 재발견 후보로 그대로 샌다. [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 백링크에 대해 이미 같은 것을 요구했다. "경로가 새면 제외가 무의미해진다."

판정을 두 벌 두지 않는다. 결과를 `view.byPath`와 `view.bySlug`로 걸러 노출 문서만 남긴다.

### 기각 버튼을 두지 않는다

카드 옆에 기각 버튼을 두고 싶어진다. 두지 않는다. 기각은 위키에 영구 기록되는 사람의 판단이고([0028](0028-rediscovery-state-and-boundaries.md)에서 `engram-state.yaml`은 git이 추적하는 유일한 재발견 상태다), 인증 없는 화면에 그 경로를 두면 [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)가 거절한 "제안 접수함"이 이름만 바꿔 돌아온다.

화면은 커맨드를 알려 주고 멈춘다.

## 결정

**`engram serve`에 `/resurface` 화면을 둔다. 읽기 전용이다.**

| 항목 | 값 |
|---|---|
| 제시 이력 | **쓰지 않는다.** `resurface.Run`을 `dryRun`으로 부른다 |
| 기각 | **화면에 경로가 없다.** `bridge --reject`를 안내한다 |
| 임베딩 | **캐시만 읽는다.** 계산하지 않는다 |
| 의미 축 부재 | 화면에 적는다 |
| 노출 범위 | `view`로 다시 거른다. 제외 문서는 카드에도 없다 |
| 내용 | 다시 볼 문서, 연결 후보 쌍(잡은 축과 점수), 아무도 안 가리키는 문서 |

[0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md)의 "재발견 커맨드를 노출하지 않는다"를 이 ADR이 대체한다. 나머지 절은 그대로다.

읽기 전용 계약의 시험은 위키 전체 해시 비교다. `TestServeDoesNotTouchWiki`가 `/resurface` 요청 전후로 `.engram/`을 포함한 모든 파일의 해시를 견준다. 화면이 상태를 쓰면 그 시험이 깨진다.

## 결과

- 임베딩이 만든 쌍을 눈으로 볼 자리가 생긴다. 잡은 축을 카드에 적으므로 의미 축만 잡은 쌍이 구별된다.
- 별도 렌더러 없이 카드가 나온다. `--json`으로 자기 렌더러를 붙이는 길은 그대로 있다.
- 화면과 CLI가 다른 재발견 후보를 낼 수 있다. 화면이 냉각을 소모하지 않기 때문이며 의도한 것이다.
- 캐시가 빈 위키에서는 연결 후보가 단어 축만으로 나온다. 화면이 그 사실을 밝힌다.

## 관련

- [0044 serve는 읽기 전용이고 검수된 지식만 보여준다](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md) 이 ADR이 한 절을 대체한다
- [0028 재발견 커맨드의 상태를 성격에 따라 두 곳에 나눠 둔다](0028-rediscovery-state-and-boundaries.md) 어느 상태가 입력이고 어느 것이 출력인지
- [0075 임베딩은 문서에 붙고 축마다 자기 하한을 갖는다](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md) 계산 자리를 bridge로 정한 결정
- [0043 MCP는 쓰기 도구를 하나만 노출하고 promote를 내보내지 않는다](0043-mcp-exposes-one-write-tool-and-omits-promote.md) 캐시만 읽는 같은 규율
