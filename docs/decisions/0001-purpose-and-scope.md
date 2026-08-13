---
number: 0001
title: 프로젝트 목적과 범위
date: 2026-08-08
status: accepted
---

# 0001 프로젝트 목적과 범위

## 결정

`engram`(당시 명칭 `llm-wiki-edu`)은 운영 중인 `llm-wiki`의 철학(승급 파이프라인·다축 스키마·재발견 루프)을 **단일 Go 바이너리 오픈소스**로 출판하여, 사내 교육강좌와 공개 산출물로 사용한다.

## 배경

2026년 7월 기술 전문가로 선정되며 2026년도 활동 계획을 수립해야 했다. 교육 콘텐츠 후보로 `llm-wiki` 운영 체계 구축 방법론을 검토했다.

문제는 산출물의 형태다. 현재 `llm-wiki`는 shell/python script 모음으로, 본인에게는 "빠르게 고쳐 쓰는 유연성"이 핵심 가치다. 하지만 사내 교육·홍보 맥락에서 script 모음은 "가치성을 말하기 어렵다" — 수강생의 설치 진입장벽(Python venv, bge-m3 다운로드, git hooks)이 너무 높고, "스크립트 묶음"은 제품으로 인지되지 않는다.

## 범위

- **하는 것**: llm-wiki 구조를 단일 Go 바이너리로 정식 구현. Homebrew 배포. 교육용 데모 위키 + 커리큘럼.
- **안 하는 것**: 본인 실운영을 이 바이너리로 전환하지 않는다. `llm-wiki` script 운영은 현행 유지.

## 관련

- [0002 운영과 산출물의 분리](0002-operating-vs-product-separation.md)
- [0003 접근법 B](0003-approach-B-fullspec-with-promotion.md)
