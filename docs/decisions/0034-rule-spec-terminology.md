---
number: 0034
title: upstream 규칙 문서를 계약이 아니라 규칙 명세라 부른다
date: 2026-08-16
status: accepted
---

# 규칙 명세라는 이름

## 배경

ADR [0005](0005-upstream-contract-and-harness.md)가 upstream `meta/` 아래 규칙 문서를 "계약 파일"이라 이름 붙였다. 그 표기가 ADR 16건과 설계 문서 전반, harness 문서, sync 스크립트로 퍼졌다.

같은 자리에 이름이 둘인 경우도 생겼다. 0005는 세 번째 층을 "conformance 테스트"라 부르면서 그 산출물 파일은 `docs/parity.md`로 지었다. 이후 일상 표기는 parity가 이겼는데 0009, 0010, 0011은 아직 conformance라고 쓴다. 같은 것을 가리키는 두 단어가 문서에 공존한다.

`delta`도 마찬가지다. 문서는 영어를 쓰는데 이 저장소를 설명하는 자리에서는 자연히 "변경분"이라는 말이 나온다.

## 판단 근거

**계약은 양방향 의무를 함의한다.** 이 용어가 소프트웨어에서 쓰이는 자리는 둘이다. Design by Contract는 함수의 선행조건과 후행조건을 양쪽이 지키는 구조이고, consumer-driven contract testing은 소비자가 "나는 이 필드를 쓴다"를 제공자에게 되돌려 선언하기 때문에 계약이 성립한다.

여기는 그렇지 않다. upstream이 규칙을 선언하고 engram이 따라갈 뿐이며 **engram이 upstream에 요구하는 것은 없다.** 단방향인데 계약이라 부르니 말의 무게가 실물과 어긋난다.

**명세는 0005 본문에 이미 있었다.** 변경 감지 절의 제목이 "변경 감지와 spec-delta"다. spec이라는 단어가 계약 옆에 처음부터 같이 있었고 계약이 이겼을 뿐이다. 새로 들여오는 말이 아니다.

**이 저장소는 공개 예정이고 교육 자료다.** 독자가 단어의 뜻을 먼저 배워야 하면 그만큼 진입이 늦어진다. 명세는 설명 없이 통한다.

**같은 층에 이름을 둘 두지 않는다.** conformance와 parity 중 하나를 버린다. 산문에서는 무엇을 하는 층인지가 드러나는 편이 낫다.

**모든 "계약"을 바꾸지는 않는다.** ADR [0016](0016-cli-framework-and-global-flags.md)의 "전역 플래그 계약"은 다른 용례다. CLI가 사용자에게 하는 약속이고 사용자가 그 약속에 기대어 스크립트를 짜므로 양방향이다. 그 자리의 계약은 맞는 말이다. 이 결정은 **upstream 규칙 문서를 가리키는 용례에만** 적용된다.

**`scripts/upstream-sync.py`는 바꾸지 않는다.** 이 스크립트는 upstream `AGENTS.md`의 `## meta 계약 변경 로그` 절 제목과 `계약 파일(...)` 문자열을 그대로 파싱해 명세 목록을 읽는다. 그 표기는 upstream의 것이고 저장소가 다르다. 여기서 바꾸면 sync가 깨진다.

## 결정

**upstream `meta/` 아래 규칙 문서를 "규칙 명세" 또는 줄여서 "명세"라 부른다.**

| 옛 표기 | 새 표기 | 적용 범위 |
|---|---|---|
| 계약 파일 | 규칙 명세 (명세) | upstream 규칙 문서를 가리키는 용례만 |
| delta | 변경분 | 산문 |
| conformance, parity | 동등성 검증 | 산문 |

- 파일명과 패키지명은 영어를 유지한다. `docs/parity.md`, `harness/parity`, `harness/upstream.lock`이 그대로다. 이 저장소는 이미 `min_wikilinks`와 "위키링크"를 같은 방식으로 나눠 쓴다.
- **기존 ADR의 본문과 제목을 소급 수정하지 않는다.** ADR [0015](0015-adr-status-vocabulary-and-amendment-index.md)의 규칙이다. 제목에 "계약"이 남는 것은 0005, 0016, 0029 셋이다. 0016은 다른 용례라 그대로가 맞고, 나머지 둘은 당시 명칭으로 읽는다.
- `docs/`의 설계 문서, `AGENTS.md`, `README.md`, `harness/`의 문서는 새 표기로 갱신한다.
- `scripts/upstream-sync.py`는 갱신하지 않는다. upstream 표기를 파싱하는 코드다. 그 사실을 주석 한 줄로 밝힌다.
- 3층 harness의 층 이름을 확정한다. **명세 사본 고정, 변경분 감지, 동등성 검증**이다.

## 결과

- upstream은 자기 문서에서 계속 "계약 파일"이라 부른다. 저장소가 다르고 그쪽에는 그쪽의 작업 계약이 있다. 두 표기가 만나는 자리는 `upstream-sync.py` 하나뿐이며 거기에 주석을 남긴다.
- 0005를 읽는 사람이 "계약 파일"과 "규칙 명세"가 같은 것임을 알아야 한다. 이 ADR과 개정 그래프가 그 연결이다.
- 3층 harness를 층 이름으로 부를 수 있게 된다. 지금까지는 층 이름 없이 세부 용어만 오갔고, 그래서 "delta 위치"나 "parity 갱신 방식" 같은 말이 어느 층 이야기인지 알 수 없었다.

## 관련

- [0005 upstream 계약과 harness](0005-upstream-contract-and-harness.md) 당시 명칭 "계약 파일"의 출처. 3층 harness의 원안
- [0015 ADR 상태 어휘와 개정 색인](0015-adr-status-vocabulary-and-amendment-index.md) 소급 수정 금지와 표기 갱신 처리
- [0027 독자에 따른 문체](0027-prose-register-by-audience.md) 독자별 문체 구분
- `docs/spec-map.md` 명세와 구현의 대응표
