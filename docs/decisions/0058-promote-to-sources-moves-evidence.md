---
number: 0058
title: promote --to sources가 inbox의 증거를 옮긴다
date: 2026-08-17
status: accepted
---

# inbox의 탈출 경로가 셋인데 둘만 있었다

## 배경

핸즈온 4단계를 실제 에이전트로 검증하다가 upstream `inbox/README.md`를 읽었다. 계약 파일 선언 밖에 있어 [spec-map](../spec-map.md)이 대응시키지 않은 문서였다.

> After processing:
>
> - delete the inbox item if it has no long-term value
> - **move evidence to `sources/` if it should be traceable**
> - promote durable knowledge to `context/`

**`inbox`가 나가는 길은 셋이고 engram에는 둘뿐이었다.** 삭제는 파일을 지우면 되고 `promote`가 `context`로 올린다. **가운데가 비어 있었다.**

`source`는 표준 입력으로 새 문서를 만들 뿐 `inbox` 문서를 옮기지 않고, `demote`는 `context`에서만 내려간다.

**실제 에이전트가 이 자리에서 잘못된 우회를 두 번 만들어 냈다.**

```
engram source --title "..." --created 2026-03-04 --ref "회의록" < inbox/2026-08-17-test.md
```

돌려 봤다. `inbox` 문서의 프론트매터가 본문 안에 그대로 박히고, `inbox` 원본이 안 지워져 같은 내용이 두 곳에 남는다. **에이전트가 없는 길을 지어낸 것이 아니라 있어야 할 길이 없어서 우회한 것이다.**

그 결과가 승급 문서에 남는다. 전사를 `capture`로 받아 `context`로 올리면 원문이 사라지고 `source_refs: []`가 된다. `context/README.md`가 요구하는 `source-backed`가 성립하지 않는다.

## 판단 근거

**`promote`에 붙인다.** upstream `AGENTS.md`가 파이프라인을 `inbox→sources→context`로 표기한다. `sources`는 `inbox`보다 위이므로 올리기다. 새 동사를 만들면 커맨드 스물여덟이 스물아홉이 되고 사용자가 외울 것이 하나 는다. `demote`가 이미 `--to`로 도착 단계를 고르므로 플래그 이름도 재사용한다.

**게이트를 적용하지 않는다.** 게이트는 지식 노드가 고립된 채 `context`로 올라가는 것을 막는 장치다([0019](0019-index-documents-outside-the-gate.md)). `sources`는 지식이 아니라 증거다. 링크 없는 전사도 증거로서 값어치가 있다.

**이동이다. 파생이 아니다.** `inbox`는 임시 계층이므로 원본이 남으면 같은 내용이 두 벌이 된다([0022](0022-promote-moves-inbox-derives-sources.md)의 근거와 같다). `sources`에서 `context`로 가는 것이 파생인 것과 다르다.

**본문을 고치지 않는다.** `sources`는 이후 본문을 고치지 않는 것이 계약이다([0009](0009-schema-presets-and-thresholds.md)). 프론트매터만 새로 만들고 본문은 그대로 옮긴다.

**`type` 기본값이 `source` 커맨드와 다르다.** `source`는 `source-summary`다. 실물 분포에서 정제본이 97건이고 원본이 열 건이기 때문이다([0051](0051-sources-holds-originals-and-refined-summaries.md)). 이쪽은 upstream이 **"move evidence"**라 부르는 자리이고 증거는 원문이다. **`source-raw`가 기본값이다.**

**`created`를 지어내지 않는다.** `--created`가 없으면 문서가 이미 갖고 있는 값을 쓴다. `capture`가 넣은 날짜라 입수일에 가깝지만 없는 날짜를 만드는 것보다 낫다. 값을 확정하는 것은 사람의 일이다([0052](0052-agent-prepares-the-promotion-and-the-human-decides-it.md)).

**반대 논거를 검토했다.** `capture` 단계에서 `source`로 바로 넣게 하면 이 커맨드가 필요 없다는 것이다.

기각한다. **넣는 시점에는 그 자료가 증거로 남을 값어치가 있는지 모른다.** `inbox`의 정의가 "분류, 검증, 중복 제거, 승급 넷 중 하나라도 안 된 것"이고, 그 판단을 넣는 자리에서 요구하면 관문이 두 번이 된다. 3단계에서 `capture`를 마찰 없이 둔 이유와 같다.

## 결정

`promote`에 `--to`를 더한다. 값은 `context`(기본)와 `sources` 둘이다.

```
engram promote inbox/2026-08-17-회의전사.md --to sources \
  --created 2026-03-04 --ref "회의 전사" --channel teams
```

| 항목 | 동작 |
|---|---|
| 대상 | `inbox` 문서만. `sources`나 `context` 문서는 거절하고 `demote`를 안내한다 |
| 이동 | 파일을 옮긴다. `inbox` 원본이 남지 않는다 |
| 본문 | 그대로 둔다 |
| 게이트 | 적용하지 않는다 |
| `type` 기본값 | `source-raw` |
| 파일명 | `<created>-<슬러그>.md`. `created`가 없으면 문서의 기존 값, 그것도 없으면 오늘 |
| `--dry-run` | 지원한다([0056](0056-promote-has-a-dry-run.md)) |

`--created`, `--ref`, `--channel`은 `source` 커맨드와 같은 플래그를 쓴다. `capture`가 채워 둔 `source_channel`은 `--channel`이 없으면 살린다.

## 결과

- `inbox`의 탈출 경로 셋이 전부 커맨드로 열린다.
- 전사를 `capture`로 받은 뒤에도 증거로 남길 수 있다. 그다음 `promote`로 파생을 만들면 `derived_from`이 자동으로 채워져 `source-backed`가 성립한다.
- 회귀 시험 다섯을 더했다. 이동과 원본 제거, 게이트 미적용, `--dry-run`, `inbox` 아닌 단계 거절, `--to` 허용값이다.
- 핸즈온 4단계에서 "이 길이 없다"고 밝히던 절이 실제 절차로 바뀐다.

## 열린 항목

- `context/README.md`의 `source-backed` 요구가 여전히 강제되지 않는다. `source_refs` 키가 있는지만 보고 값이 비었는지는 안 본다. 이 커맨드는 그 값을 채우기 **쉽게** 만들 뿐 강제하지 않는다.
- `archive`로 가는 `--to`는 두지 않았다. 폐기는 `archive` 커맨드가 이미 맡는다.
