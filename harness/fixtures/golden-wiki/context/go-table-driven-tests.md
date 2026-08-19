---
type: procedure
artifact_stage: context
status: promoted
indexable: true
tags:
  - go
  - testing
source_refs:
  - sources/tech-talk-summary.md
derived_from:
  - sources/tech-talk-summary.md
related:
  - "[[test-coverage-matrix]]"
source_channel: manual
derived_context: []
form: howto
topics:
  - go
  - testing
created: 2025-12-10
updated: 2026-01-20
---

# 테이블 주도 테스트

하나의 테스트 함수에서 여러 케이스를 표로 묶어 돌리는 패턴이다.
케이스가 늘어나도 함수가 늘지 않으므로 검토 부담이 작다.

작성 순서는 아래와 같다.

1. 케이스를 담을 구조체 슬라이스를 정의한다.
2. 각 케이스에 이름을 붙여 실패 시 무엇이 깨졌는지 알 수 있게 한다.
3. 케이스마다 기대값과 실제값을 비교해 한 줄로 보고한다.

커버리지 기준을 정하는 문서는 [[test-coverage-matrix]] 를 함께 본다.
