<!-- upstream llm-wiki meta/wiki-artifact-schema.md 에서 가져왔다. 익명화 치환 5건을 적용했다.
     출처 커밋은 harness/upstream.lock 에 있다.
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

## Assets

자산은 마크다운이 아닌 파일(SVG, PNG, PDF, 첨부 문서)이다. 스스로 지식이
되지 못하고 **어떤 문서의 일부로만 존재한다.**

### 배치 규칙

자산은 **그것을 참조하는 문서와 같은 레이어의 `assets/`**에 둔다.

| 참조하는 문서 | 자산 위치 |
| --- | --- |
| `inbox/teams/2026-08-06-....md` | `inbox/teams/assets/` |
| `sources/summaries/....md` | `sources/assets/` |
| `context/systems/....md` | `context/assets/` |
| `meta/promotion-rules.md` | `meta/assets/` |

**원문이 없는 자산은 존재할 수 없다.** 어느 마크다운도 참조하지 않는
`*/assets/` 파일은 고아이며 lint가 FAIL로 막는다. 자산을 먼저 만들고 문서를
나중에 붙이는 순서를 허용하지 않는다는 뜻이다.

### 근거 자산과 저작 자산

승급 시 자산을 옮길지는 자산의 성격이 정한다. 둘을 섞으면 링크가 깨진다.

| 구분 | 정체 | 예 | 승급 시 |
| --- | --- | --- | --- |
| **근거 자산** | 밖에서 받은 것. 원문의 일부 | 회의 첨부, 받은 보고서, 원본 캡처 | **옮기지 않는다** |
| **저작 자산** | 이 위키가 만든 것. 문서의 표현 | 직접 그린 개념도, 구조 SVG | **문서를 따라 옮긴다** |

근거 자산을 옮기지 않는 이유는 `sources/`를 append-only로 두는 이유와 같다.
근거는 그것이 있던 자리에 있어야 증거가 된다. 옮기면 이미 그 경로를 적어 둔
`source_refs`가 전부 무효가 된다. 실제 사례가
`inbox/teams/assets/legacy-ui-migration-report.md`로, 이를 참조하는
`context/concepts/sap-integration-architecture-basics.md`가 승급된 뒤에도
자산은 `inbox/`에 남아 있고 그것이 옳다.

저작 자산은 반대다. 문서의 표현이므로 문서가 `context/`로 올라가면 함께
올라간다. 남겨두면 `inbox/`가 비워질 때 참조가 끊긴다.

### 판단이 갈리는 자리

**원문이 이 위키 밖에만 있는 자산은 어느 레이어에도 두지 않는다.** 외부
시스템(Confluence, 블로그)에 붙일 목적으로만 그린 도식이 여기 해당한다.
그것은 발행 부산물이지 위키 지식이 아니다. 위키에 남길 값어치가 있다면
먼저 그 내용을 설명하는 문서를 위키 안에 만들고, 자산은 그 문서의 저작
자산으로 붙인다. 문서 없이 자산만 커밋하지 않는다.

2026-08-28에 `context/assets/`의 AIDx 도식 3종을 이 규칙으로 정리했다.
Confluence 페이지 전용 1회성 산출물이었고 위키 안에 원문이 없었다.


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
