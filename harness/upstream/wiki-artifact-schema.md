<!-- upstream llm-wiki meta/wiki-artifact-schema.md 에서 가져왔다.
     원본 커밋 05f7279. 익명화 치환 4건이 적용되었습니다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Wiki Artifact Schema

> Classify artifacts by channel, trigger, workflow, and stage instead of making every variation a new inbox folder.

## Four Axes

Use these axes for LLM Wiki artifacts.

| Axis | Meaning | Examples |
| --- | --- | --- |
| `source_channel` | Where the material came from | `voice-memo`, `web`, `slack`, `teams`, `confluence`, `jira`, `mobile`, `manual`, `automation` |
| `trigger_mode` | How the work started | `manual-prompt`, `scheduled-automation`, `file-drop`, `mcp-fetch`, `script-import` |
| `workflow` | What processing flow is being run | `voice-memo-intake`, `tech-news-digest`, `meeting-note`, `issue-decision-extract`, `page-summarize` |
| `artifact_stage` | What layer this file belongs to | `inbox`, `source`, `context`, `agent-workflow`, `index` |

This avoids creating folders like `inbox/news` only because a workflow exists. News is usually `source_channel: web` plus `workflow: tech-news-digest`.

## Common Frontmatter

Use this shape for new managed markdown artifacts when practical:

```yaml
---
type: procedure
artifact_stage: context
status: promoted
scope: mixed
sensitivity: internal
source_channel: voice-memo
trigger_mode: manual-prompt
workflow: voice-memo-intake
artifact_stage: context
source_refs:
  - sources/manifests/example.md
derived_from:
  - sources/summaries/example.md
derived_context:
  - "[[example-decision]]"
related:
  - "[[inbox-processing-matrix]]"
tags:
  - llm-wiki
indexable: true
created: 2026-01-01
updated: 2026-01-01
---
```

The canonical frontmatter contract is `meta/frontmatter-schema.md`. This file defines the classification vocabulary; the frontmatter schema defines required fields by quality level.

## Artifact Types

| Type | Default location | Indexable by default |
| --- | --- | --- |
| `source-manifest` | `sources/manifests/` | no |
| `transcript` | `sources/transcripts/` | no |
| `source-summary` | `sources/summaries/` | no |
| `decision` | `context/decisions/` | yes when promoted |
| `procedure` | `context/procedures/` | yes when promoted |
| `system` | `context/systems/` | yes when promoted |
| `concept` | `context/concepts/` | yes when promoted |
| `project` | `context/projects/` | yes when promoted |
| `agent-workflow` | `agents/workflows/` | yes only when explicitly marked |
| `moc` | root or `context/mocs/` index pages | yes |

## Quality Levels

Use `artifact_stage` to separate capture, evidence, and reusable wiki knowledge:

| Level | Default stage | Meaning |
| --- | --- | --- |
| `inbox` | `inbox` | raw or unreviewed capture |
| `source` | `source` | source-backed processed artifact |
| `context` | `context`, selected `agent-workflow`, `index` | promoted reusable wiki node |

## 검색 시스템 Rule

검색 시스템 should select by metadata and path:

- include `artifact_stage: context`
- include promoted `context/`
- include selected `agents/workflows/`
- include MOC/index files
- optionally include selected source stage summaries for evidence retrieval
- exclude `artifact_stage: inbox`
- exclude raw and private sources
- exclude `indexable: false`

## Obsidian Rule

Obsidian-facing files should prefer stable titles and wikilinks. Tags are useful for broad grouping; wikilinks are better for specific relationships.

Use `meta/wiki-graph-policy.md` for relationship fields and link requirements by quality level.
Use `meta/frontmatter-schema.md` for required Obsidian Properties and 검색 시스템 eligibility fields.

## Related

- [[wiki-search-architecture]]
- [[obsidian-operating-layer]]
- [[wiki-graph-policy]]
- [[frontmatter-schema]]
