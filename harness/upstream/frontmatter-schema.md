<!-- upstream llm-wiki meta/frontmatter-schema.md 에서 가져왔다.
     원본 커밋 05f7279. 익명화 치환 6건을 적용했다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Frontmatter Schema

> Frontmatter is the control plane for Obsidian, agents, and 검색 시스템.

## Purpose

Obsidian-facing markdown should not rely only on folder paths and body text. Frontmatter should carry the machine-readable state that agents and future indexers need:

- artifact identity
- quality level
- sensitivity
- source and workflow provenance
- graph relationships
- indexing eligibility
- review state

This lets the same markdown files work for human review in Obsidian, agent workflows in Codex/Claude, and retrieval governance in 검색 시스템.

## Required Fields By Level

### inbox stage. Inbox / Raw Capture

```yaml
---
type: inbox-note
artifact_stage: inbox
status: inbox
scope: unknown
sensitivity: private-local-only
source_channel:
trigger_mode:
workflow:
artifact_stage: inbox
indexable: false
---
```

### source stage. Source / Processed Evidence

```yaml
---
type: source-summary
artifact_stage: source
status: sourced
scope: unknown
sensitivity: internal
source_channel:
trigger_mode:
workflow:
artifact_stage: source
source_refs: []
derived_from: []
derived_context: []
related: []
tags: []
indexable: false
---
```

### context stage. Promoted Wiki Node

```yaml
---
type: procedure
artifact_stage: context
status: promoted
scope: mixed
sensitivity: internal
source_channel:
trigger_mode:
workflow:
artifact_stage: context
source_refs:
  - sources/manifests/example.md
derived_from:
  - sources/summaries/example.md
related:
  - "[[example-related-node]]"
tags:
  - llm-wiki
indexable: true
created: 2026-01-01
updated: 2026-01-01
---
```

## Field Definitions

| Field | Required | Meaning |
| --- | --- | --- |
| `type` | yes | Artifact type such as `decision`, `procedure`, `source-summary`, `agent-workflow` |
| `artifact_stage` | yes | `inbox`, `source`, or `context` |
| `status` | yes | `inbox`, `sourced`, `promoted`, `archived`, `superseded` |
| `scope` | yes | `work`, `personal`, `mixed`, `unknown` |
| `sensitivity` | yes | `public-reference`, `internal`, `restricted`, `private-local-only` |
| `source_channel` | yes | Origin channel such as `voice-memo`, `web`, `jira` |
| `trigger_mode` | yes | `manual-prompt`, `scheduled-automation`, `file-drop`, `mcp-fetch`, `script-import` |
| `workflow` | yes | Processing workflow such as `voice-memo-intake` |
| `artifact_stage` | yes | `inbox`, `source`, `context`, `agent-workflow`, `index` |
| `source_refs` | source stage/context stage | Evidence references |
| `derived_from` | source stage/context stage | Direct upstream artifact |
| `derived_context` | source stage | Context nodes produced from this source |
| `related` | context stage | Meaningful wiki links |
| `tags` | recommended | Broad Obsidian grouping |
| `indexable` | yes | Whether 검색 시스템 may index this artifact |
| `created` | source stage 필수 / 그 외 recommended | 원본이 처음 기록된 날. 회의한 날, 문서가 쓰인 날이다. `YYYY-MM-DD` 또는 연월까지만 아는 경우 `YYYY-MM` |
| `sourced_at` | source stage 필수 | 이 wiki에 source로 편입된 날. git 최초 커밋일이 진실원이다 |
| `updated` | auto | 마지막으로 내용이 갱신된 날. `scripts/sync_updated_field.py`가 git 이력에서 채운다. 손으로 쓰지 않는다 |

### sources의 날짜를 두 개로 나눈 이유

`created`와 `sourced_at`은 서로 다른 사실이다. 이전 노트 앱에서 이관해 온 2021~2024년
자료가 54개 있는데, 편입일 하나만 남기면 "4년 전 자료"라는 사실이 사라진다.
반대로 원본 작성일만 남기면 언제부터 이 wiki가 그 자료를 알고 있었는지 모른다.

`sources/`에는 `updated`를 넣지 않는다. 원본 보존 계층이라 갱신되지 않으며,
오타 수정으로 `updated`가 최신이 되면 오히려 신선도를 오해하게 만든다.
`scripts/sync_updated_field.py`도 같은 이유로 `sources/`를 스캔하지 않는다.

백필과 신규 채움은 `scripts/backfill_source_dates.py`가 한다(멱등). `created`는
파일명 날짜 접두사에서, `sourced_at`은 git 최초 커밋일에서 가져온다.
`recorded_at`, `fetched_at` 같은 워크플로 고유 필드는 부가 정보로 그대로 두되,
날짜 질의의 기준은 `created`/`sourced_at` 두 개로 본다.

`review_after`(언제까지 유효한가)는 쓰지 않는다. 예측이라 지킬 수 없다.
실제로 2026-08-07 기준 값이 있던 63개 중 22개가 같은 날짜에 몰려 있었고,
이미 지난 것 2건에 대해 아무 조치도 일어나지 않았다. 대신 `updated`로
"최근까지 유효했다"는 사실을 남기고, 무의미해진 문서는 `archive/`로 옮긴다.

## Obsidian Usage

Obsidian Properties should expose these fields for filtering and review. Tags are useful for broad navigation, but relationship fields should use wikilinks where possible.

Useful Obsidian views later:

- promoted but missing `source_refs`
- source stage summaries with empty `derived_context`
- `updated`가 오래된 context stage 문서 (archive 후보)
- `indexable: true` documents with restricted sensitivity
- workflow clusters by `workflow`

## 검색 시스템 Usage

검색 시스템 should treat frontmatter as the primary eligibility signal:

- include `artifact_stage: context` and `indexable: true`
- optionally include selected `artifact_stage: source` summaries for evidence retrieval
- exclude `artifact_stage: inbox`
- exclude `indexable: false`
- exclude sensitive material unless explicitly allowed

## Template Rule

Every new template in `meta/templates/` should include the required frontmatter fields for its quality level.

## Related

- [[wiki-artifact-schema]]
- [[wiki-graph-policy]]
- [[obsidian-operating-layer]]
