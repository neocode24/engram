---
number: 0004
title: IP와 배포 경로
date: 2026-08-08
status: amended
---

# 0004 IP와 배포 경로

## 공개 범위 밖

이 결정의 본문은 조직의 IP 경계와 내부 배포 경로에 대한 판단이라 공개하지 않는다. 근거와 처리 방식은 [0024](0024-public-boundary-and-private-directory.md)에 있다.

번호와 제목만 남기는 이유는 결정 이력의 번호 연속성 때문이다. 이 자리에 결정이 있었다는 사실은 남기고 내용만 뺀다.

공개 대상인 부분은 다른 ADR이 이미 담고 있다.

- 저장소 구성과 공개 경계를 폴더로 긋는 원칙은 [0011](0011-repo-layout-and-module-name.md)에 있다.
- 배포 경로는 [0012](0012-distribution-via-personal-homebrew-tap.md)가 개인 Homebrew tap 단일화로 대체했다.
- 스키마의 조직 특수 축을 프리셋으로 다루는 결정은 [0009](0009-schema-presets-and-thresholds.md)에 있다.

## 관련

- [0011 저장소 구성과 Go 모듈명](0011-repo-layout-and-module-name.md)
- [0012 배포 경로를 개인 Homebrew tap으로 단일화한다](0012-distribution-via-personal-homebrew-tap.md)
- [0024 공개 경계를 gitignore된 private 디렉토리로 긋고 이력을 익명화한다](0024-public-boundary-and-private-directory.md)
