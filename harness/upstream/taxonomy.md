<!-- upstream llm-wiki meta/taxonomy.md 에서 가져왔다.
     원본 커밋 eacb0f6. 치환 없음.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Taxonomy

Initial classification for company LLM Wiki material.

## Input Channels

- Teams
- Slack
- Confluence
- Jira
- Voice Memos
- Web
- Manual

## Knowledge Types

- concept
- project
- system
- decision
- procedure
- incident
- meeting-summary
- agent-workflow

## Sensitivity

- public-reference
- internal
- restricted
- private-local-only

Only `public-reference` and safe `internal` material should be considered for `context/` promotion.

## Scope

- work
- personal
- mixed
- unknown

Do not force scope classification at capture time. Use `unknown` when unsure and resolve during processing.
