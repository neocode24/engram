---
number: 0098
title: upstream이 updated 필드를 채우는 시점을 바꿔도 engram은 바꾸지 않는다
date: 2026-08-28
status: accepted
---

# upstream이 updated 필드를 채우는 시점을 바꿔도 engram은 바꾸지 않는다

## 맥락

upstream `meta/frontmatter-schema.md`가 커밋 93096a0에서 `updated` 필드 서술을 바꿨다. 예전에는 pre-commit 훅이 `scripts/sync_updated_field.py`(인자 없음)로 지나간 git 이력에서 날짜를 읽었는데, 그 시점의 이력에는 방금 만들어지는 커밋이 아직 없어 값이 한 세대 늦게 박혔다. 이제 pre-commit이 `sync_updated_field.py --precommit`으로 스테이징된 문서에 지금 만들어지는 커밋의 커미터 날짜를 직접 쓴다. `--apply`는 사후 정리 도구로 남는다.

필드의 의미(마지막으로 내용이 갱신된 날), 대상 범위(`context`, `agents/workflows`. `sources`는 제외), 임계값(벌크 커밋 필터), 형식(`YYYY-MM-DD`)은 전혀 바뀌지 않았다. 바뀐 것은 upstream 쪽 훅이 그 값을 **몇 번째 커밋에서** 쓰느냐는 타이밍뿐이다.

## 결정

**engram 쪽 아무것도 고치지 않는다.**

engram은 애초에 upstream처럼 "지나간 git 이력에서 읽어 한 세대 늦게 쓰는" 방식을 따른 적이 없다. [ADR 0037](0037-sync-corrects-dates-from-git.md)이 정한 대로 `engram sync`는 실행 시점에 `internal/gitdate`를 통해 커밋 이력을 직접 읽어 프론트매터를 정정하며, pre-commit 훅에 끼워 넣는 구조가 아니다. upstream이 이번에 고친 "몇 세대 늦게 박히는" 결함 자체가 engram 구현에는 없었다. `internal/config`의 `Updated` 필드 정의, `internal/lint`의 `checkSourcesUpdated`(sources 계층 검사), `internal/resurface`의 `BaseDate` 우선순위 어느 것도 이 변경이 건드리는 지점을 참조하지 않는다.

## 확인

- `harness/parity` lint 축과 resurface 축 모두 이 변경 전후로 동일하게 통과한다.
- `docs/spec-map.md` 6절 절차의 3~4번(lint 규칙, 프리셋 임계값)에 해당하는 변경점이 없다.
- 골든 스냅샷을 재생성할 필요가 없었다(내용에 영향이 없으므로).

## 대안

**upstream 표현을 그대로 문서에 옮겨 적는다** — engram 문서 어디에도 "pre-commit이 지나간 이력에서 읽는다"는 서술이 없었으므로 옮길 대상이 없다. 오히려 이번 변경은 engram이 이미 택한 방식(실행 시점 직접 읽기)이 upstream이 뒤늦게 겪은 문제를 처음부터 피했다는 것을 확인해 줄 뿐이다.
