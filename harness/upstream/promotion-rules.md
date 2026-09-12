<!-- upstream llm-wiki meta/promotion-rules.md 에서 가져왔다. 치환 없음.
     출처 커밋은 harness/upstream.lock 에 있다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Promotion Rules

Promotion has two stages. Verified evidence moves from `inbox/` into `sources/`,
while reusable knowledge moves from `sources/` or `inbox/` into `context/`.

*(llm-wiki 승급 구조. 원본 그림은 upstream 의 assets/promotion-layers.svg 에 있다)*

세 계층은 역할이 다르다. `inbox/`는 담는 곳이라 스키마 검사를 받지 않고,
`sources/`는 남기는 곳이라 read-mostly이며, `context/`는 쓰는 곳이라 RAG
인덱싱 대상이다. `sources/` 진입은 사람의 사실 확인에서 시작하고,
`context/` 진입은 사람의 명시적 판단이 필요하다. `context/` 진입만
pre-commit 게이트가 지킨다.

## Sources Entry

다음 세 조건을 모두 만족하면 `inbox/` 초안을 `sources/`로 옮긴다.

- 사람이 초안의 사실(인명, 용어, 화자, 날짜)을 한 번 이상 교정했거나
  "맞다"고 확인했다. 교정 커밋이 그 증거다.
- `sensitivity`가 정해졌다. 기본값은 `restricted`이며, 사용자가
  `private-local-only`라고 말하면 그대로 둔다.
- `source_refs`가 candidate, transcript, 원문 드롭 같은 근거를 가리킨다.

재사용 가치, 중복 여부, 결론의 안정성은 `sources/` 진입 조건이 아니다.
그 판단은 `context/` 진입 조건에만 적용한다.

`source_channel: agent-session` 초안은 교정 대신 "다음 세션에서 근거로
참조됨"을 사실 확인 행위로 본다.

## Context Entry

### Promote When

- the note answers a future question likely to recur
- the source is known
- the conclusion is stable enough to reuse
- sensitive details are removed or scoped
- duplicates/conflicts have been checked

### Do Not Promote When

- the input is raw conversation without a clear reusable point
- the facts are uncertain and no source exists
- the content is private, credential-like, or too sensitive
- it is only a temporary task reminder

## Promotion Output

Every document promoted to `context/` should include:

- one-line conclusion
- context
- current understanding
- evidence
- related links
