<!-- upstream llm-wiki meta/wiki-graph-policy.md 에서 가져왔다.
     원본 커밋 05f7279. 익명화 치환 10건을 적용했다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Wiki Graph Policy

> The wiki graph is managed from processed sources through promoted context; 검색 시스템 consumes the governed graph, not only isolated documents.

## Purpose

The LLM Wiki should preserve relationships between evidence, summaries, decisions, procedures, systems, projects, and agent workflows. Obsidian backlinks and graph navigation are useful only when those relationships are intentionally written into the files.

This policy defines where graph links are required, where they are optional, and how they relate to 검색 시스템 indexing.

## Quality Levels

| Level | Meaning | Typical locations | 검색 시스템 default |
| --- | --- | --- | --- |
| `inbox` | raw or unreviewed capture | `inbox/`, raw exports, raw transcripts, local-only candidates | excluded |
| `source` | source-backed processed artifact | `sources/manifests/`, `sources/summaries/`, reviewed transcripts | evidence-only or excluded by default |
| `context` | promoted reusable wiki node | `context/`, selected `agents/workflows/`, MOC/index pages | included when indexable |

Quality level is not a judgment of importance. It is a statement of how safe the artifact is for reuse.

## Relationship Fields

Use these fields in frontmatter. They are not only documentation metadata; they are the machine-readable graph surface for Obsidian review and 검색 시스템 governance:

```yaml
artifact_stage: context
source_refs:
  - sources/manifests/example.md
derived_from:
  - sources/summaries/example.md
derived_context:
  - "[[example-decision]]"
related:
  - "[[example-procedure]]"
supersedes:
  - "[[old-rule]]"
superseded_by:
```

Field meanings:

| Field | Direction | Use |
| --- | --- | --- |
| `source_refs` | context/agent -> source | Evidence backing this reusable node |
| `derived_from` | artifact -> previous artifact | Direct source artifact this file was made from |
| `derived_context` | source -> context | Context nodes produced from this source |
| `related` | peer relationship | Conceptual or operational relation |
| `supersedes` | new -> old | This document replaces an older one |
| `superseded_by` | old -> new | This document is replaced by a newer one |

## Link Requirements By Stage

### inbox stage. Inbox And Raw Capture

Required:

- minimum metadata when available: source channel, trigger mode, workflow, sensitivity

Optional:

- rough related links

Do not spend time building a graph for low-quality or sensitive raw material. The goal is triage.

### source stage. Sources And Processed Artifacts

Required:

- source manifest links back to raw or stable source reference
- summary links to its manifest
- transcript links to its manifest when approved
- manifest records `derived_context` after promotion happens

Optional:

- links to candidate context nodes before promotion

source stage is where provenance becomes reliable. It should allow a reviewer to trace an context stage node back to evidence.

### context stage. Context And Agent Workflows

Required:

- `source_refs`
- `related` links when nearby context exists
- stable title
- `artifact_stage: context`
- `indexable: true` unless there is a reason to exclude

Recommended:

- tags for broad grouping
- `supersedes` / `superseded_by` for changing decisions or procedures
- MOC links for high-value clusters

context stage is the main Obsidian graph layer and the default 검색 시스템 corpus.

## MOC Rules

MOC files are human navigation maps. They should not duplicate every backlink automatically.

Create or update a MOC when:

- a topic has three or more promoted nodes
- a workflow needs a stable entry point
- 검색 시스템 retrieval needs a high-signal overview document

Useful MOC candidates:

- `index.md`
- `context/systems/wiki-search-architecture.md`
- future `context/mocs/voice-memos.md`
- future `context/mocs/tech-news.md`

## 검색 시스템 Relationship Rule

검색 시스템 indexing should favor context stage documents, but answers should remain traceable to source stage evidence through `source_refs`.

Default policy:

- index context stage by default
- optionally index selected source stage summaries as evidence retrieval
- exclude inbox stage
- exclude any artifact with `indexable: false`
- exclude `private-local-only` and `restricted` unless explicitly overridden

## Obsidian Relationship Rule

Obsidian should show the working knowledge graph:

- context stage nodes link to each other with wikilinks
- source stage artifacts link upward to derived context
- MOC pages provide topic-level entry points

Backlinks are useful only if links are meaningful. Do not add low-signal links just to increase graph density.

## Frontmatter Rule

Frontmatter is mandatory for managed artifacts except truly raw imported material. Use it to make graph and quality state visible to Obsidian Properties and available to agents.

Required:

- source stage and context stage artifacts must carry `artifact_stage`, `source_refs` or `derived_from`, and `indexable`.
- context stage artifacts must carry `related` when meaningful neighboring context exists.
- source stage source manifests should be updated with `derived_context` after promotion.
- Templates must include the relevant frontmatter block from `meta/frontmatter-schema.md`.

## Related

- [[wiki-search-architecture]]
- [[obsidian-operating-layer]]
- [[context-indexing-boundary]]
- [[frontmatter-schema]]
