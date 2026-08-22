<!-- upstream llm-wiki meta/security-rules.md 에서 가져왔다. 치환 없음.
     출처 커밋은 harness/upstream.lock 에 있다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Security Rules

This repository may contain company-related operational knowledge. Treat it conservatively.

## Never Commit

- credentials
- API tokens
- private keys
- session cookies
- personal identity data not required for the knowledge task
- uncontrolled raw exports from DMs or sensitive meetings
- customer-sensitive details without redaction

## Sensitive Source Handling

Use `sources/raw-private/` for local-only staging. This path is gitignored.

If a source is sensitive but produces a reusable operational lesson, promote only the lesson and keep the source reference minimal.

## Mirror Handling

Do not mirror private or raw source folders to iCloud.

The iCloud mirror is for mobile reading and quick capture, not for sensitive source storage.
