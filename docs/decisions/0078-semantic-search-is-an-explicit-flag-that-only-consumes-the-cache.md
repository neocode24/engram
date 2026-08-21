---
number: 0078
title: 의미 검색은 명시 플래그이고 캐시를 소비만 한다
date: 2026-08-21
status: accepted
---

# 의미 검색은 명시 플래그이고 캐시를 소비만 한다

## 배경

[0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md)와 [0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md)로 임베딩이 들어왔다. 실제로 닿는 커맨드를 세어 보면 하나다.

| 커맨드 | 임베딩 |
|---|---|
| `bridge` | 쓴다 |
| `search`, `serve`의 검색 | 안 쓴다. 단어 축만 |
| `resurface`, `digest` | 안 쓴다 |

upstream `scripts/wiki_resurface.py`도 임베딩을 bridges 계산에만 쓰므로 동등성은 지켜져 있다. 그러나 2.2 GB 모델을 내려받아 문서 여든에 15분을 쓴 대가가 가끔 치는 커맨드 하나라면 임베딩이 어디에 쓰였는지 사용자가 알 길이 없다.

`search`는 매일 쓴다. 에이전트가 승급할 이을 곳을 찾을 때 부르는 것도 `search`다([0057](0057-approval-attaches-to-content-not-to-the-command.md)의 승급 준비 절차 1번). **의미 축이 가장 쓸모 있을 자리가 비어 있다.**

막고 있던 것은 비용 추정이었다. 문서 하나 인코딩이 12.6초이므로 질의도 그러리라 보였다. 실측했다.

| 대상 | 시간 |
|---|---|
| 모델 적재 | 530 ms |
| 질의 "응답 지연" (5자) | 167 ms |
| 질의 "배치가 데이터베이스를 잡는다" (15자) | 441 ms |
| 문서 하나 (2000자) | 12.6 s |

**질의 검색 한 번이 모델 적재까지 1초 안쪽이다.** 12.6초는 2000자를 밀어 넣었을 때의 값이고 질의는 그 자릿수가 아니다. `search --semantic`의 실측 총 소요는 0.96초다.

## 판단 근거

### 계산하지 않는다

문서 벡터는 캐시에서 읽기만 한다. 없는 것을 여기서 계산하면 검색 한 번이 문서 수에 12.6초를 곱한 시간이 된다.

캐시를 채우는 자리는 `bridge`다([0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md)). 그 결정을 여기서 뒤집지 않는다. **캐시가 비어 있으면 계산하지 않고 `bridge`를 안내하고 멈춘다.** 조용히 단어 축으로 떨어지지 않는다. 사용자가 `--semantic`을 명시했으므로 무엇이 없어서 안 되는지 말하는 것이 옳다.

`serve`와 MCP도 같은 규율이다([0076](0076-serve-shows-rediscovery-without-recording-it.md), [0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md)). 캐시를 채우는 것은 `bridge` 하나다.

### 기본값이 아니다

`--semantic` 없는 `search`는 지금과 같다. 0.58초이고 모델을 열지 않는다.

기본으로 켜면 셋을 잃는다. 모델이 없는 환경에서 매번 경고가 나고, 검색이 0.58초에서 0.96초가 되며, 골든 스냅샷과 교재의 실측 출력이 모델 유무에 따라 달라진다. 셋 다 받아들일 이유가 없다.

### 두 축을 섞지 않는다

단어 축은 BM25 점수이고 실측 범위가 2.5에서 14.9다. 의미 축은 코사인이고 0.42에서 0.70이다. **척도가 다르므로 하나로 합치려면 세 번째 순위 규칙이 필요하다.**

역순위 융합 같은 방법이 있으나 도입하면 `bridge`가 쓰는 규율과 어긋난다. `bridge`는 축마다 하한을 따로 두고 합집합을 내며 어느 축이 잡았는지 밝힌다([0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md)). 같은 위키의 같은 두 축이 커맨드마다 다르게 합쳐지면 사용자가 점수의 뜻을 배울 수 없다.

**축을 고르게 한다.** `--json`에 `axis`를 실어 소비자가 점수의 뜻을 알게 한다. 두 축을 다 보려면 두 번 부른다.

실측이 이 선택을 지지한다. 질의 "데이터베이스 경합"에서 단어 축 1위는 그 낱말이 본문에 있는 `response-latency-incident`이고 의미 축 1위는 실제로 그 주제인 `batch-db-contention`이다. 섞으면 이 차이가 보이지 않는다.

### 대상이 context로 좁다

벡터는 `bridge`가 만들고 `bridge`는 `context/`만 본다. 따라서 `search --semantic`도 `context/`만 찾는다. `sources/` 원본은 이 축에 없다.

넓히면 비용이 곱절 이상이다. upstream 실측으로 `context/` 81문서에 `sources/` 153문서다. 지금 15분이 45분이 된다.

받아들이고 밝힌다. 이을 곳을 찾을 때는 게이트가 `sources/`도 세므로([0023](0023-gate-targets-exclude-inbox.md)) 단어 축을 함께 돌린다. 커맨드 도움말과 계약 문서에 적는다.

## 결정

**`search --semantic`을 둔다. 낱말 대신 의미로 순위를 매긴다.**

| 항목 | 값 |
|---|---|
| 기본 | 꺼져 있다. 켠 적 없는 `search`는 그대로다 |
| 문서 벡터 | **캐시만 읽는다.** 없으면 `bridge`를 안내하고 멈춘다 |
| 질의 벡터 | 실행 시점에 만든다. 모델 적재까지 1초 안쪽 |
| 순위 | 코사인 단독. 단어 축과 섞지 않는다 |
| 동점 | 슬러그 오름차순([0028](0028-rediscovery-state-and-boundaries.md)) |
| 대상 | `context/`만 |
| `--json` | `axis`에 `term` 또는 `semantic`을 싣는다 |
| MCP | `search` 도구가 `semantic` 인자를 받는다. 같은 코드 경로다 |

## 결과

- 임베딩이 매일 쓰는 커맨드에 닿는다. 모델을 받은 대가가 `bridge` 하나가 아니게 된다.
- 에이전트가 이을 곳을 의미로 찾을 수 있다. 같은 낱말을 안 쓴 문서가 후보에 오른다.
- `--semantic`은 `bridge`를 한 번도 안 돌린 위키에서 실패한다. 실패 메시지가 무엇을 하면 되는지 말한다.
- `sources/`는 이 축에 안 잡힌다. 도움말이 그 사실을 밝히고 단어 축을 함께 쓰라고 안내한다.
- 점수의 뜻이 축마다 다르다. `--json`의 `axis`가 그것을 알린다.

## 관련

- [0075 임베딩은 문서에 붙고 축마다 자기 하한을 갖는다](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md) 두 축을 섞지 않는 규율의 출처
- [0074 임베딩은 순수 Go로 돌리고 모델은 bge-m3 fp32로 고정한다](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md) 문서 인코딩 비용의 근거
- [0025 인덱스를 JSON으로 저장하고 조회는 인덱스를 갱신하지 않는다](0025-index-storage-and-staleness.md) 조회가 파일을 쓰지 않는다는 계약
- [0076 serve는 재발견을 기록 없이 보여준다](0076-serve-shows-rediscovery-without-recording-it.md) 캐시만 읽는 같은 규율
