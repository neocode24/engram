---
number: 0099
title: upstream이 wiki-link 커밋에 updated 예외를 둬도 engram은 바꾸지 않는다
date: 2026-09-05
status: accepted
---

# upstream이 wiki-link 커밋에 updated 예외를 둬도 engram은 바꾸지 않는다

## 맥락

upstream `meta/frontmatter-schema.md`가 커밋 df07b29(재발견 link 판정 자동 반영, `wiki_apply_links.py` + 신선도 보호)에서 `updated` 필드 서술에 예외 한 줄을 더했다. 종전에는 pre-commit 훅이 스테이징된 문서 전부에 커밋 날짜를 무조건 썼다. 이제 `WIKI_LINK_ONLY=1` 환경에서 만드는 `wiki-link:` 커밋(`scripts/wiki_apply_links.py`가 재발견 link 판정을 자동 반영하며 `related` 한 줄만 추가하는 커밋)에서는 `updated`를 쓰지 않는다. 링크 한 줄 추가가 문서의 "마지막 내용 갱신"이 되면 재발견 신선도 신호가 죽기 때문이다.

필드의 의미, 대상 범위, 형식은 바뀌지 않았다. 바뀐 것은 upstream이 **재발견 링크 판정을 자동으로 위키에 반영하는 별도 스크립트(`wiki_apply_links.py`)를 새로 두고, 그 스크립트가 만드는 특정 종류의 커밋만 골라 예외를 준 것**이다.

## 결정

**engram 쪽 아무것도 고치지 않는다.**

이 예외가 다루는 대상 자체가 engram에 없다. engram의 재발견 커맨드(`resurface`, `bridge`, `digest`)는 후보와 근거만 반환하고 위키를 자동으로 고치지 않는다(README "Rediscovery commands return candidates and evidence only"). `related` 필드를 채우거나 커밋을 만드는 주체는 언제나 사람이거나 사람이 돌리는 `engram update --set related=...`이며, 그 경로는 [ADR 0055](0055-agents-change-the-wiki-only-through-commands.md)가 정한 대로 커맨드를 통과한다.

`internal/cli/update.go`의 `updated` 자동 채움 로직은 `--set`/`--unset`에 `updated`가 있는지와 `sources` 계층인지만 본다. 커밋 메시지의 접두사나 환경변수로 예외를 두는 개념 자체가 없다. `WIKI_LINK_ONLY=1` 같은 훅 레벨 신호는 upstream의 pre-commit 구조([ADR 0098](0098-upstream-updated-field-timing-change-needs-no-engram-change.md)이 이미 정리한 대로 engram은 애초에 그 구조를 안 씀)에서만 의미가 있다.

engram이 이 자동 반영 자체를 나중에 만든다면(재발견 판정을 커맨드가 위키에 직접 써넣는 기능) 그때 신선도 보호를 어떻게 할지가 새 설계 문제이지, 지금 이 예외를 옮겨 적을 문제가 아니다.

## 확인

- `internal/resurface`, `internal/bridge`, `internal/digest` 어디에도 위키 파일을 쓰는 코드가 없다. 읽기 전용이다.
- `internal/cli/update.go`의 `autoUpdated` 채움 조건에 커밋 메시지 접두사나 환경변수 분기가 없다.
- `harness/parity` lint 축과 resurface 축 모두 이 변경 전후로 동일하게 통과한다(`go test ./... -count=1`, `ENGRAM_UPSTREAM=~/Git/llm-wiki go test ./harness/parity/ -v`).
- `docs/spec-map.md` 6절 절차의 3~4번(lint 규칙, 프리셋 임계값)에 해당하는 변경점이 없다.
- 골든 스냅샷을 재생성할 필요가 없었다.

## 대안

**engram에 `wiki-link:` 류의 자동 반영 기능이 생길 것을 대비해 지금 예외 개념을 문서에 미리 적어 둔다** — 대비할 기능 자체가 설계되지 않았다. 없는 기능의 없는 예외를 문서에 옮기면 다음 사람이 실체 없는 규칙을 좇게 된다. 기능이 생기는 시점에 그 설계의 일부로 다룬다.
