---
type: concept
artifact_stage: context
status: promoted
indexable: true
tags:
  - documentation
source_refs: []
derived_from: []
related: []
source_channel: manual
derived_context: []
form: reference
topics:
  - documentation
created: 2026-01-05
updated: 2026-01-05
---

# 위키링크 문법

위키 문서를 연결하는 문법을 정리한 문서다. 이 문서는 문법을 설명하기 위해
코드 블록에 링크처럼 보이는 문자열을 담는다. 코드 블록 안의 링크는 링크가 아니다.

## 문법

기본 형태는 대괄표 두 개로 슬러그를 감싼다.

```
예: [[가짜링크]] 는 코드 블록 안이므로 링크로 세면 안 된다.
```

인라인 코드 안도 마찬가지다. `[[인라인가짜링크]]` 처럼 감싼 경우 링크가 아니다.

## 왜 중요한가

코드 블록 안의 링크를 세는 도구는 이 문서를 링크가 있는 문서로 오판한다.
문법 설명 문서가 게이트를 통과하는 것은 이 규칙이 지켜질 때만 성립한다.
