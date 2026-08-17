---
number: 0051
title: sources는 원본과 정제본을 함께 담고 type이 그 둘을 가른다
date: 2026-08-17
status: accepted
---

# sources는 원본과 정제본을 함께 담고 type이 그 둘을 가른다

## 배경

[0009](0009-schema-presets-and-thresholds.md)가 문서 종류 열을 기본 허용값으로 정했다. `sources/` 단계에 쓸 수 있는 값은 `source-summary` 하나이며 `source` 커맨드의 기본값도 그것이다.

그런데 `sources/`는 원본 보존 계층이고 이후 본문을 고치지 않는 것이 계약이다([0022](0022-promote-moves-inbox-derives-sources.md)). 회의 전사나 외부 글 전문을 그대로 넣으면 문서 종류가 "요약"으로 붙는다. 무엇을 보존했는지가 프론트매터에서 거짓이 된다.

upstream 실물을 확인했다. `sources/` 아래 마크다운 151건의 분포다.

| 자리 | 건수 | 성격 |
|---|---|---|
| `summaries/` | 97 | 정제본 |
| `manifests/` | 42 | 출처 메타데이터 |
| `exports/` | 6 | 외부 도구 export |
| `transcripts/` | 4 | 원본 전사 |
| `raw-private/` | 1 | 로컬 전용 |

**원본과 정제본이 함께 산다.** upstream `sources/README.md`가 그 성격을 이렇게 적었다. "These files are not necessarily good knowledge documents. They exist so that promoted context can be checked later." 지식이 아니라 증거라는 것이다.

upstream `meta/wiki-artifact-schema.md`는 문서 종류 열을 표로 선언한다. engram의 기본값과 대조하면 셋이 어긋난다.

| upstream에만 | engram에만 |
|---|---|
| `source-manifest`, `transcript`, `moc` | `incident`, `meeting-summary`, `inbox-note` |

`moc`은 [spec-map](../spec-map.md) 6절이 `index` 단계와 묶어 이미 기록했다. 나머지 둘은 기록되지 않았다.

## 판단 근거

**`transcript`를 그대로 가져오지 않는다.** 전사만 덮기 때문이다. 외부 글 전문과 도구 export는 전사가 아닌데 같은 문제를 갖는다. upstream도 `exports/`에 쓸 종류를 선언하지 않았다. 좁은 이름을 가져오면 같은 구멍이 남는다.

**일반형 하나를 만든다.** 필요한 구분은 "사람이나 에이전트가 손댄 결과인가, 원문 그대로인가" 하나다. 매체가 음성이든 웹 문서든 export든 이 구분은 같다. `source-raw`가 그 자리다. `transcript`는 이것의 좁은 경우이며 그 구분이 필요한 위키는 `engram.yaml`의 `types`로 켠다.

**`source-manifest`는 더하지 않는다.** manifest는 별도 문서가 원본의 출처 메타데이터를 담는 구조인데 engram에는 그 개념이 없다. engram은 `source_refs`와 `created`와 `source_channel`을 같은 문서의 프론트매터에 담는다. 개념 없이 종류만 더하면 껍데기가 된다.

**기본값은 `source-summary`로 둔다.** 실물 분포에서 정제본이 97건이고 원본이 열 건이다. 더 흔한 쪽을 기본값으로 둔다. 원본을 넣을 때 `--type source-raw`를 명시하게 하는 편이, 정제본에 매번 종류를 붙이게 하는 것보다 손이 덜 간다.

## 결정

기본 문서 종류에 `source-raw`를 더한다. 열하나가 된다.

| 종류 | 뜻 | 언제 |
|---|---|---|
| `source-raw` | 원문 그대로. 손대지 않았다 | 회의 전사, 외부 글 전문, 도구 export |
| `source-summary` | 정제본. 읽고 추린 결과 | 회의록, 구조화 요약. `source` 커맨드의 기본값 |

`sources/`에 둘 다 온다. 어느 쪽인지는 `type`이 말한다. 디렉토리를 나누지 않는다. 단계 디렉토리는 성숙도를 뜻하며 종류를 뜻하지 않는다([0009](0009-schema-presets-and-thresholds.md)의 축 개념과 같은 자리다).

**정제는 사람이나 에이전트가 하고 engram은 하지 않는다.** engram에 요약 기능이 없다([0014](0014-llm-boundary-agent-drives-binary.md)). 정제본을 `sources/`에 넣는 것은 그 결과물을 증거로 보존하는 행위이며, 승급 대상인 `context/`와 다르다.

## 결과

- `engram source --type source-raw`가 동작한다. 실습 자료가 이것을 쓴다.
- upstream의 `transcript` 4건은 engram 기본값에서 여전히 거절된다. 그 위키가 `types`에 더하면 통과한다. 이 사실을 [spec-map](../spec-map.md) 6절에 적는다.
- `source-manifest` 42건도 같다. 개념 자체를 옮길지는 열린 항목이다.

## 열린 항목

- `source-manifest`에 해당하는 개념을 engram이 가질지. 지금은 출처 메타데이터가 같은 문서의 프론트매터에 있다. 원본 하나에 출처가 여럿 붙는 경우가 실제로 나오면 그때 본다.
- `type`을 폐쇄 집합으로 둔 것이 맞는지. upstream `meta/frontmatter-schema.md`는 "such as"로 열어 두었고 `wiki-artifact-schema.md`만 표로 닫았다. engram은 닫힌 쪽을 따랐다. 위키마다 `types`로 넓힐 수 있으므로 지금은 문제가 되지 않는다.
