---
type: concept
artifact_stage: context
status: promoted
indexable: true
tags:
  - testing
  - wiki
source_refs:
  - sources/tech-talk-summary.md
derived_from:
  - sources/tech-talk-summary.md
related:
  - "[[go-table-driven-tests]]"
  - "[[markdown-link-syntax]]"
source_channel: manual
derived_context: []
form: reference
topics:
  - testing
created: 2025-12-03
updated: 2026-02-10
---

# 테스트 피라미드

팀에서 테스트를 배치하는 기준을 정리한 문서다. 기본 원칙은 아래와 같다.

- 단위 테스트가 가장 많고, 통합 테스트는 그 절반 이하로 유지한다.
- 사용자 여정을 검증하는 end to end 테스트는 가장 적게 둔다.
- 느린 테스트는 빠른 테스트보다 실패 원인을 특정하기 어려우므로 수를 제한한다.

구체적인 단위 테스트 작성법은 [[go-table-driven-tests]] 문서에서 다룬다.
위키 문서 안에서 링크 문법을 다루는 규칙은 [[markdown-link-syntax#문법]] 섹션을 참고한다.

## 운영 규칙

피라미드 비율은 분기마다 한 번 측정한다. 측정 결과는 회고 문서에 남긴다.
