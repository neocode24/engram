---
type: concept
artifact_stage: context
status: promoted
indexable: true
tags:
  - databases
source_refs: []
derived_from: []
related:
  - "[[testing-pyramid]]"
  - "[[cli-flag-conventions]]"
source_channel: manual
derived_context: []
form: reference
topics:
  - sqlite
created: 2025-10-22
updated: 2025-11-30
---

# 외래 키 제약과 스키마 설계

관계형 스키마에서 외래 키 제약을 거는 규칙을 정리한 문서다.

- 참조 무결성이 필요한 모든 관계에 외래 키를 건다. 애플리케이션에서 지키는
  무결성은 애플리케이션 버그 하나로 사라진다.
- 제약 이름을 명시한다. 자동 생성 이름은 오류 메시지에서 알아볼 수 없다.
- 마이그레이션 순서를 문서화한다. 제약 추가는 데이터 검증 뒤에 둔다.

스키마 변경 절차를 자동화할 때 테스트 배치는 [[testing-pyramid]] 기준을 따른다.
도구의 플래그 규칙은 [[cli-flag-conventions]] 를 따른다.
