<!-- upstream llm-wiki meta/promotion-rules.md 에서 가져왔다. 치환 없음.
     출처 커밋은 harness/upstream.lock 에 있다.
     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->
# Promotion Rules

Promotion has two stages. Verified evidence is created in `sources/` from
confirmed `inbox/` content, while reusable knowledge moves from `sources/` or
`inbox/` into `context/`. The processed inbox draft moves to
`archive/<channel>/` in the same promotion commit.

*(llm-wiki 승급 구조. 원본 그림은 upstream 의 assets/promotion-layers.svg 에 있다)*

세 계층은 역할이 다르다. `inbox/`는 담는 곳이라 스키마 검사를 받지 않고,
`sources/`는 남기는 곳이라 read-mostly이며, `context/`는 쓰는 곳이라 RAG
인덱싱 대상이다. `sources/` 진입은 사람의 사실 확인에서 시작하고,
`context/` 진입은 사람의 명시적 판단이 필요하다. `context/` 진입만
pre-commit 게이트가 지킨다.

## Inbox Exit

초안이 `inbox/`를 떠나는 사건은 셋뿐이다.

1. **승급 커밋.** 초안의 내용이 `sources/` 또는 `context/` 문서에 들어가고,
   그 문서가 초안을 `source_refs` 또는 `derived_from`으로 가리킨다. 같은
   커밋에서 초안을 `archive/<channel>/<같은 파일명>`으로 `git mv`한다.
   승급을 수행한 주체가 이동까지 맡는다.
2. **재배치.** 사람이 지식이 아니라고 판단한 문서는 `exports/docs/`로,
   채널이 틀린 초안은 맞는 `inbox/<channel>/`로 옮긴다.
3. **삭제.** 사람이 "승급하지 않는다"고 정한 초안은 아무도 참조하지
   않으므로 `git rm`한다. `archive/`에 두지 않는다.

기한은 없다. 나이만으로 나가는 문서는 없으며, 사람이 손대지 않은 초안은
무기한 `inbox/`에 머문다. 이것은 결함이 아니라 "처리 대기" 상태의 정의다.

`archive/`는 폐기장이 아니라 근거 보존소다. `sources/`와 `context/` 문서가
`derived_from`으로 가리키는 근거를 보존하므로 지울 수 없다. 참조를 받지 않는
파일은 `archive/`에 넣지 않는다.

pre-commit lint는 `sources/`와 `context/` 문서의 `source_refs`와
`derived_from` 중 `inbox/`로 시작하는 경로가 워킹트리에 아직 존재하면
FAIL해야 한다. `/assets/`를 포함하는 경로는 제외한다. 근거 자산은 원문 자리에
남기기 때문이다. 이 검사는 다음 build 카드에서 구현한다.

승급 대상이 `context/`이고 초안 본문을 담은 `sources/` 문서가 없다면,
`archive/` 이동 전에 `sources/summaries/`에 본문을 옮긴 source summary를 먼저
만든다. 그렇지 않으면 context는 증류본인데 원문은 어디에도 남지 않는다.

채널마다 실행 시점이 다르다. `voice-memos`는 교정 턴에 sources 생성과 archive
이동을 한 번에 한다. `agent-session`은 "참조 시 sources 이동" 규칙을 따르되,
그때 만드는 source summary의 `derived_from`에 원래 초안 경로를 반드시 적는다.

## Sources Entry

다음 세 조건을 모두 만족하면 `inbox/` 초안의 내용으로 `sources/` 문서를
만들고, 같은 커밋에서 초안을 `archive/<channel>/`로 옮긴다.

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
