---
type: procedure
artifact_stage: context
status: promoted
indexable: true
tags:
  - cli
source_refs: []
derived_from: []
related:
  - "[[testing-pyramid]]"
  - "[[go-table-driven-tests]]"
source_channel: manual
derived_context: []
form: cheatsheet
topics:
  - cli
  - documentation
created: 2026-01-12
updated: 2026-01-12
---

# CLI 플래그 작성 규칙

명령줄 도구의 플래그를 정하는 규칙이다.

- 단일 문자 플래그와 긴 플래그를 함께 제공한다. 예를 들어 `-v` 와 `--verbose` 다.
- 값을 받는 플래그는 이름 뒤에 붙이지 않고 공백으로 가른다.
- 플래그 이름은 케밥 케이스로 쓴다. `--output-format` 처럼 쓴다.

테스트 전략의 큰 틀은 [[testing-pyramid]] 에 맞춘다.
구체적인 테스트 코드 작성법은 [[go-table-driven-tests]] 를 따른다.
