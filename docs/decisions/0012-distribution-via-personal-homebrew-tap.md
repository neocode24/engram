---
number: 0012
title: 배포 경로를 개인 Homebrew tap으로 단일화한다
date: 2026-08-15
status: accepted
---

# 배포 경로를 개인 Homebrew tap으로 단일화한다

## 배경

ADR 0001은 Homebrew 배포를 범위에 넣었다. ADR 0007은 이를 뒤집어 Tier 1을 windows/amd64와 darwin/arm64로 잡고 Windows 쪽 winget과 scoop을 우선 경로로 두었다. 근거는 수강생 다수가 Windows 사용자라는 것이었다.

ADR 0004는 개인 GitHub에서 개발하고 사내 공개 시점에 Enterprise로 미러한다고 정했다. 그 결과 배포 채널이 개인과 사내 둘로 갈라지고, `self-update`가 채널을 리다이렉트해야 한다는 요구가 붙었다.

두 결정이 합쳐지면서 배포 경로가 넷이 되었다. Homebrew, winget, scoop, Enterprise 미러다. 아직 코드가 한 줄도 없는 시점에 릴리스 파이프라인이 넷으로 갈라져 있다.

## 판단 근거

**이미 동작하는 배포 인프라가 있다.** `neocode24/homebrew-tap`이 실재하며, 기존 프로젝트의 릴리스 워크플로가 태그 푸시 시 tap 저장소를 자동 갱신한다. 새로 만들 것은 Formula 정의뿐이다. 반면 winget과 scoop은 각각 별도 매니페스트 저장소와 심사 절차를 요구하며, 코드 서명이 없으면 SmartScreen 경고가 남는다.

**Enterprise 미러를 배포 채널로 삼으면 관리 지점이 는다.** 릴리스마다 두 곳에 올려야 하고, `self-update`가 환경에 따라 다른 곳을 보게 해야 하며, 버전 불일치가 생긴다. 사내 공개는 저장소 접근성의 문제이지 배포 채널의 문제가 아니다.

**저장소를 private에서 public으로 전환하면 이 문제가 사라진다.** 공개된 저장소의 릴리스는 사내에서도 그대로 받을 수 있다. 이 프로젝트는 형태와 방법론을 제공하며 사내 사례를 담지 않기로 했으므로(ADR 0011), 공개를 막을 이유가 없다.

**사내 사례가 필요해지는 시점은 따로 있다.** 강의가 실제로 굴러가면 사내 맥락을 담은 자료가 생긴다. 그것은 공개 저장소에 넣지 않고 `private/`로 분리하며, 필요하면 Enterprise를 추가 origin으로 붙여 그쪽에만 밀어 넣는다. 이는 배포가 아니라 자료 격리다.

## 결정

**배포는 `neocode24/homebrew-tap` 단일 체계로 한다.**

- Formula로 배포한다. 기존 프로젝트는 Cask(macOS 앱)이지만 engram은 CLI이므로 `Formula/engram.rb`에 놓인다.
- 릴리스 워크플로는 태그 푸시 시 아티팩트를 빌드하고 tap 저장소의 Formula를 자동 갱신한다. 기존 워크플로 구조를 재사용한다.
- Homebrew를 쓸 수 없는 환경을 위해 GitHub Releases의 아카이브 직접 다운로드 경로를 유지한다. `self-update`는 이 경로를 본다.
- winget과 scoop은 지금 하지 않는다. Windows 수강생 비중이 실제로 확인되고 코드 서명이 준비된 뒤 재검토한다. 그때까지 Windows는 아카이브 다운로드와 `%LOCALAPPDATA%` 설치 스크립트로 지원한다.

ADR 0007이 정한 플랫폼 매트릭스(Tier 1 = windows/amd64, darwin/arm64)는 유지한다. 바뀌는 것은 설치 채널뿐이다.

**저장소는 개인 GitHub `neocode24` 아래 private으로 시작하여 준비되면 public으로 전환한다.**

내부 미러는 배포 채널이 아니다. 사내 사례를 담은 자료가 생기면 `private/` 디렉토리로 분리하고, 그 시점에 Enterprise를 추가 origin으로 등록하여 해당 자료만 그쪽에 둔다. 공개 저장소의 이력에는 사내 자료가 들어가지 않는다.

## 결과

- ADR 0007의 배포 절을 개정한다. 0007의 플랫폼 결정은 유효하며 설치 채널만 바뀐다.
- ADR 0001이 언급한 Homebrew 배포가 복권된다.
- ADR 0004의 "Enterprise 미러로 배포 채널 이관" 전제가 바뀐다. Enterprise는 사내 자료 격리 용도로만 쓴다. `self-update`의 채널 리다이렉트 요구가 사라진다.
- 릴리스 파이프라인이 하나가 되어 0.1 착수 시 배포를 함께 준비할 수 있다.

## 열린 항목

- public 전환 시점. 0.1이 동작하는 시점과 강의 일정 중 늦은 쪽으로 본다.
- 코드 서명 인증서 확보 여부. 확보되면 winget 경로를 재검토한다.
