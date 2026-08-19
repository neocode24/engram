---
type: concept
artifact_stage: context
status: promoted
source_channel:
tags: []
source_refs:
  - https://example.com/km-primer
  - sources/2026-02-10-km-primer.md
derived_from:
  - sources/2026-02-10-km-primer.md
derived_context: []
related:
  - "[[promotion-pipeline]]"
  - "[[wikilink-graph]]"
indexable: true
created: 2026-02-10
sourced_at: 2026-03-02
updated: 2026-03-02
---
# 지식관리 입문 자료

## 결론

원본을 보존하는 계층을 따로 두면, 요약이 틀렸을 때 되돌아갈 자리가 남는다.

## 맥락

이 문서는 `sources` 에 있는 원본에서 파생되었다. 원본은 지워지지 않고
제자리에 남아 있으며, 프론트매터의 `derived_from` 과 `derived_context` 가
두 문서를 양방향으로 잇는다.

## 현재 이해

`promote` 는 출발지에 따라 다르게 동작한다.

| 출발지 | 동작 |
|---|---|
| inbox | 문서를 `context` 로 **옮긴다**. 원본이 남지 않는다 |
| sources | 파생 문서를 **새로 만든다**. 원본은 그대로 있다 |

원본 보존 계층에서 문서를 빼내면 보존이라는 약속이 그 순간 깨지기
때문이다. 같은 이유로 `sources` 문서에는 `updated` 필드를 쓰지 않는다.
오타 하나 고친 것이 자료의 신선도를 오해하게 만든다.

## 근거

요약은 시간이 지나면 틀린다. 무엇을 근거로 그렇게 요약했는지 돌아갈 수
있어야 고칠 수 있다. [[promotion-pipeline]] 의 게이트가 지키는 것이
"쌓이는 쪽"이라면, 원본 보존이 지키는 것은 "거슬러 올라가는 쪽"이다.

## 관련 링크

- [[promotion-pipeline]]
- [[wikilink-graph]]
