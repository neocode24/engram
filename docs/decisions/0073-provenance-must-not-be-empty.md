---
number: 0073
title: 증거 필드는 비어 있으면 안 된다
date: 2026-08-19
status: accepted
---

# 증거 필드는 비어 있으면 안 된다

## 배경

결함이 두 겹이었다.

첫째, lint가 키 존재만 봤다. `checkRequiredFields`가 `if _, ok := s.fields[f]; !ok`로 판정하므로 `source_refs: []`가 통과한다. 값이 비었는지 검사하는 코드가 없었다.

둘째, `promote`가 빈 배열을 만들었다. `sources` 문서에서 파생을 만들 때 `derived_from`만 채우고 `source_refs`를 안 채웠다. 파생의 원본이 곧 증거인데 증거 필드에 안 들어간다. 즉 engram 자신이 빈 `source_refs`를 만들고 있었다.

검사만 넣으면 도구가 자기 산출물을 자기 규칙으로 잡는 상태가 된다. 그래서 채우는 쪽을 먼저 고치고 검사를 더한다.

upstream도 값을 보지 않는다는 것을 확인했다. `context/systems/indexing-config.md:141`이 "시스템은 출처 증명이 누락된 것을 보완하지 않는다. 승급 문서의 `source_refs` 누락은 검수 이슈로 다뤄야 한다"고 적었고 upstream lint도 존재만 본다. 그쪽은 사람에게 넘겼다. engram이 코드로 가는 근거는 하나다. 빈 배열인지의 판정은 결정론적이다. 무엇을 증거로 삼을지는 여전히 사람이 정한다.

## 판단

### promote가 파생의 원본을 source_refs에도 넣는다

`promote`가 `sources`에서 파생을 만들 때 원본 경로를 `source_refs`에도 넣는다. `derived_from`에 넣는 그 값이다. 두 필드의 뜻은 다르다. `derived_from`은 "무엇에서 나왔나"이고 `source_refs`는 "무엇이 이것을 뒷받침하나"다. 그런데 파생의 원본은 둘 다에 해당한다. 내용이 그 원본에서 나왔고 그 원본이 승급 문서의 근거를 뒷받침한다.

`inbox`에서 올린 문서는 `source_refs`를 안 채운다. 원본이 이동해 사라지므로 가리킬 대상이 없다. 증거를 남길 가치가 있는 원본은 `promote --to sources`가 그 자리다(ADR 0058). 사람이 그 경로로 증거를 먼저 남기고 슬러그를 채운다.

### lint 규칙 graph.empty-provenance를 더한다

대상은 `context` 단계 문서다. `source_refs` 축이 켜져 있고 값이 빈 배열이면 잡는다. 키가 아예 없는 경우는 기존 `frontmatter.missing-field`가 잡으므로 중복해서 잡지 않는다. 색인 문서는 승급 대상이 아니라 위키의 구조 자체이므로 뺀다(ADR 0019).

등급은 `warn`이다. 기존 위키의 승급 문서가 전부 걸린다. 무엇을 증거로 삼을지 고르는 데 판단이 필요하므로 한 번에 고칠 수 없어 경고가 맞다.

고치는 법 안내에는 `promote --to sources`로 증거를 먼저 남기는 경로를 적는다.

### sources.updated 등급을 error로 올린다

`sources.updated` 규칙의 등급을 `warn`에서 `error`로 올린다. 규칙 ID와 판정 조건은 그대로다. ADR 0064가 `update`를 거절로 바꿔 원본 보존을 계약으로 굳혔다. 그 계약이 깨진 흔적이 `updated` 필드인데 등급이 경고에 머물러 있었다. upstream `scripts/lint-frontmatter.sh`도 같은 규칙을 FAIL로 처리한다.

## 결과

- lint 규칙이 19종에서 20종이 된다. `docs/spec-map.md` 4.4의 대응 표에 들어간다.
- `promote`가 `sources`에서 만든 파생은 빈 `source_refs`로 나오지 않는다.
- `inbox`에서 올린 승급 문서는 빈 `source_refs`로 남고 lint가 경고로 알린다.
- `sources` 문서에 `updated`가 있으면 lint가 종료 코드 1로 끝낸다.
