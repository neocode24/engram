<!-- upstream llm-wiki meta/promotion-rules.md 에서 가져왔다. 치환 없음.
     출처 커밋은 harness/upstream.lock 에 있다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Promotion Rules

Promotion means moving reusable knowledge from `inbox/` into `context/`.

## Promote When

- the note answers a future question likely to recur
- the source is known
- the conclusion is stable enough to reuse
- sensitive details are removed or scoped
- duplicates/conflicts have been checked

## Do Not Promote When

- the input is raw conversation without a clear reusable point
- the facts are uncertain and no source exists
- the content is private, credential-like, or too sensitive
- it is only a temporary task reminder

## Promotion Output

Every promoted document should include:

- one-line conclusion
- context
- current understanding
- evidence
- related links
