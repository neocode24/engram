---
number: 0088
title: Homebrew 배포는 Formula가 아니라 Cask이고 tap 접근은 배포키로 한다
date: 2026-08-21
status: accepted
---

# Homebrew 배포는 Formula가 아니라 Cask이고 tap 접근은 배포키로 한다

## 배경

[0012](0012-distribution-via-personal-homebrew-tap.md)가 Homebrew 배포를 정하며 이렇게 적었다.

> Formula로 배포한다. 기존 프로젝트는 Cask(macOS 앱)이지만 engram은 CLI이므로 `Formula/engram.rb`에 놓인다.

같은 ADR 이 저장소 공개 전환을 tap 활성화의 선행 조건으로 두었고, `.goreleaser.yaml` 이 그 절차를 주석으로 남겨 두었다.

> 켜는 시점은 저장소 공개 전환과 한 묶음이다.

저장소를 공개로 바꿨으므로([0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md)의 결과) 그 절차를 밟는다. 밟아 보니 전제 둘이 어긋나 있었다.

## 측정

### goreleaser 가 Formula 생성을 접었다

로컬 goreleaser 2.17.1 로 `brews` 키를 넣고 검사했다.

    • DEPRECATED:  brews  should not be used anymore
    • error=configuration is valid, but uses deprecated properties
    ⨯ check failed

**설정은 유효하지만 `goreleaser check` 가 실패한다.** 릴리스 자체는 아직 돌 수 있으나 검사가 막히므로 설정을 검증할 길이 없어진다.

goreleaser 가 밝힌 이유는 이렇다. 예전에는 미리 컴파일한 바이너리를 설치하려면 Formula 를 편법으로 쓸 수밖에 없었는데 그 사정이 없어졌다는 것이다.

### Cask 가 이제 Linux 도 덮는다

[0012](0012-distribution-via-personal-homebrew-tap.md)가 Cask 를 "macOS 앱" 이라고 적은 것은 당시에는 맞았다. 지금은 아니다.

Homebrew 는 Cask 의 지원 대상을 아티팩트 종류가 아니라 `depends_on` 선언에서 판단한다. 아티팩트 중 `app` 과 `pkg` 는 macOS 전용이고 `app_image` 는 Linux 전용이지만, **`binary` 는 양쪽에서 돈다.**

CLI 는 `binary` 아티팩트다. 그러므로 Cask 로 내도 macOS 와 Linux 를 모두 덮는다. [0012](0012-distribution-via-personal-homebrew-tap.md)가 Formula 를 고른 근거였던 "CLI 이므로" 가 더 이상 Formula 를 가리키지 않는다.

### tap 은 이미 있고 배포키로 갱신되고 있다

`neocode24/homebrew-tap` 이 이미 공개 저장소로 존재하며 `Casks/veil.rb` 가 들어 있다. 같은 계정의 다른 프로젝트가 쓰고 있다.

그 프로젝트의 갱신 방식이 `.goreleaser.yaml` 주석이 가정한 것과 다르다.

| | `.goreleaser.yaml` 주석의 가정 | 실제로 쓰이는 방식 |
|---|---|---|
| 인증 | `HOMEBREW_TAP_TOKEN` (PAT) | **배포키** (`HOMEBREW_DEPLOY_KEY`) |
| 접근 | HTTPS + 토큰 | SSH |
| 권한 범위 | 계정의 모든 저장소 | **그 저장소 하나** |

tap 저장소에 쓰기 가능한 배포키가 이미 둘 등록되어 있다.

## 판단 근거

### Cask 로 바꾼다

[0012](0012-distribution-via-personal-homebrew-tap.md)의 결론을 뒤집는 것이 아니라 **그 결론의 근거가 사라진 것이다.** "CLI 이므로 Formula" 가 성립하려면 Cask 가 macOS 앱 전용이어야 하는데 그렇지 않다.

바꾸지 않으면 `goreleaser check` 를 릴리스 전에 돌릴 수 없다. 설정을 검증하지 못한 채 태그를 미는 것은 [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md)과 [0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md)에서 두 번 겪은 실패의 반복이다.

기존 사용자는 없다. 첫 릴리스이므로 이주 문제가 없다.

### 서명하지 않으므로 격리 속성을 지운다

코드 서명은 Apple 개발자 등록을 요구한다. 하지 않는다.

서명 없는 바이너리를 Cask 로 설치하면 Gatekeeper 가 격리 속성을 붙여 첫 실행이 막힌다. goreleaser 가 안내하는 설치 후 훅으로 그 속성을 지운다.

**이것이 macOS 보안 장치를 우회하는 것임을 밝혀 둔다.** 서명이 정답이고 이것은 차선이다. 서명을 시작하면 이 훅을 뺀다.

### 토큰이 아니라 배포키를 쓴다

근거 셋이다.

**권한이 좁다.** 배포키는 그 저장소 하나에만 쓴다. PAT 는 계정의 모든 저장소에 닿는다. 릴리스 워크플로가 유출되면 피해 범위가 다르다.

**같은 계정이 이미 그 방식이다.** 인증 방식이 프로젝트마다 다르면 키를 돌릴 때 어디를 봐야 하는지 헷갈린다.

**사람 손이 덜 간다.** 배포키는 API 로 만들고 등록할 수 있다. PAT 는 사람이 웹에서 만들어야 한다.

goreleaser 가 `repository.git.private_key` 로 SSH 를 받는다. 값이 아니라 **경로**를 받으므로 워크플로가 시크릿을 파일로 쓰고 그 경로를 넘긴다. 쓰고 나면 지운다.

### 시크릿이 없으면 멈춘다

[0042](0042-release-artifacts-and-workflow.md)가 적어 둔 것을 그대로 따른다. 시크릿이 없는 채로 릴리스를 돌리면 아카이브만 올라가고 tap 이 낡은 채로 남아 **절반만 성공한 릴리스**가 된다. 그 상태는 발견하기 어렵다.

키 준비 스텝이 시크릿을 확인하고 없으면 거기서 끝낸다.

## 결정

| 항목 | 값 |
|---|---|
| 종류 | **Cask.** `Casks/engram.rb` |
| 근거 | `binary` 아티팩트가 macOS 와 Linux 를 모두 덮는다 |
| goreleaser 키 | `homebrew_casks`. `brews` 는 쓰지 않는다 |
| 인증 | **배포키.** `HOMEBREW_DEPLOY_KEY` 시크릿, SSH |
| 키 전달 | 워크플로가 파일로 쓰고 경로를 `HOMEBREW_DEPLOY_KEY_PATH` 로 넘긴다. 뒤에 지운다 |
| 시크릿 없음 | **실패한다.** 조용히 건너뛰지 않는다 |
| 격리 속성 | 설치 후 훅으로 지운다. 서명을 시작하면 뺀다 |
| 설치 | `brew tap neocode24/tap` 뒤 `brew install --cask engram` |

## 결과

- [0012](0012-distribution-via-personal-homebrew-tap.md)의 Formula 결정이 Cask 로 바뀐다. 설치 커맨드가 `--cask` 를 포함하므로 README 와 교재의 설치 안내가 바뀐다.
- tap 하나에 Cask 둘이 놓인다. 디렉토리 구조가 그대로다.
- 서명하지 않는 동안 설치 후 훅이 Gatekeeper 를 우회한다. 서명은 미결로 남는다.
- 릴리스 워크플로가 시크릿 없이는 실패한다. 태그를 밀기 전에 시크릿이 있는지 확인해야 한다.
- **`goreleaser check` 가 통과한다.** 태그 전에 설정을 검증할 수 있다.

## 관련

- [0012 배포는 개인 Homebrew tap 단일 체계로 한다](0012-distribution-via-personal-homebrew-tap.md) Formula 로 정한 결정. 이 ADR 이 그것을 Cask 로 바꾼다
- [0042 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다](0042-release-artifacts-and-workflow.md) 절반만 성공한 릴리스를 막으라는 지침
- [0087 Windows는 재서 통과했고 포장까지 CI에서 돌린다](0087-windows-is-measured-green-and-packaging-runs-in-ci.md) 공개 전환이 이 절차의 선행 조건이었다
- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) 플랫폼 매트릭스. 설치 채널만 바뀐다
