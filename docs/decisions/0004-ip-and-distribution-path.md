---
number: 0004
title: IP와 배포 경로
date: 2026-08-08
status: accepted
---

# 0004 IP와 배포 경로

## 결정

- 구현 과정은 본인 개인 GitHub(`neocode24`)에 둔다.
- 강의 시점에 필요하면 **내부 미러**로 공유(레포 미러/이관)한다.
- IP·영업비밀 gating은 이 경로로 해소된다.

## 근거

- llm-wiki 체계 자체(승급 파이프라인 설계)는 방법론이지 회사 고유 기반이 아니다.
- 교육용 데모 위키는 회사 정보가 없는 깨끗한 예시로 별도 제작한다 (`wiki/`).
- 사내 강의 배포는 내부 미러 경로로 회사 통제권을 확보한다.

## 열린 항목

- 사내 Enterprise 레포 구체적 명/위치 — 강의 확정 시 결정.
- 회사 특수 스키마 축(`sensitivity: restricted/private-local-only`)을 공개 오픈소스 스키마에 둘지, 보편화하여 제외할지 → [design.md](../design.md)의 스키마 매핑에서 결정.

## 관련

- [0001 프로젝트 목적과 범위](0001-purpose-and-scope.md)
