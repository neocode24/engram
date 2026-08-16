---
number: 0042
title: 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다
date: 2026-08-17
status: accepted
---

# 릴리스 산출물과 워크플로

## 배경

ADR [0012](0012-distribution-via-personal-homebrew-tap.md)가 배포 체계를 정했다. `neocode24/homebrew-tap`의 Formula 하나이고, 태그 푸시가 아티팩트를 빌드하고 tap의 Formula를 자동 갱신한다. GitHub Releases의 아카이브 직접 다운로드 경로도 유지한다.

정해지지 않은 것은 그 워크플로를 무엇으로 만드느냐다. 아카이브 형식과 이름, 체크섬, 버전 주입, 재현 가능성도 함께 정해야 한다.

**설치가 이 프로젝트의 이탈 지점이다.** ADR [0001](0001-purpose-and-scope.md)이 스크립트 묶음을 버린 이유가 "수강생의 설치 진입장벽"이었고, ADR [0009](0009-schema-presets-and-thresholds.md)가 여정 0을 "사내 배포에서 이탈이 가장 많은 구간"으로 지목했다. 단일 바이너리를 만든 목적이 여기서 회수된다.

## 판단 근거

**goreleaser를 쓴다.** 손으로 만들면 여섯 플랫폼 조합의 빌드, 아카이브, 체크섬, 릴리스 노트, tap Formula 갱신을 YAML 백여 줄로 다시 짜야 한다. goreleaser는 그 전부를 설정 사십 줄로 하고 Homebrew tap 갱신을 내장 지원한다. **이 저장소가 지키는 무의존 원칙은 바이너리에 적용되는 것이지 CI 도구에 적용되는 것이 아니다.** 산출물에는 아무 흔적도 남지 않는다.

**태그 푸시가 유일한 계기다.** 수동 실행 버튼을 두면 태그 없는 릴리스가 생기고 `version`이 무엇을 가리키는지 모호해진다. 태그가 곧 버전이다.

**`-trimpath`와 `CGO_ENABLED=0`을 준다.** 후자는 ADR [0007](0007-platform-and-distribution.md)이 정한 것이고 이미 CI가 지킨다. 전자는 빌드 경로가 바이너리에 박히는 것을 막는다. 개인 기계의 홈 디렉토리 경로가 공개 산출물에 들어가면 안 된다. **공개 경계 문제이기도 하다.**

**체크섬을 낸다.** Homebrew Formula가 SHA256을 요구하고, 아카이브를 직접 받는 사용자가 검증할 수단이 필요하다. 파일 하나에 모은다.

**Formula 갱신에는 별도 토큰이 필요하다.** 기본 `GITHUB_TOKEN`은 자기 저장소에만 쓴다. tap은 다른 저장소이므로 토큰을 시크릿으로 넣어야 한다. **그 시크릿이 없으면 릴리스가 조용히 절반만 성공하면 안 된다.** 없으면 실패시킨다.

**tap 갱신은 공개 전환 뒤에 켠다.** 저장소가 private인 동안 Homebrew는 아카이브를 받을 수 없다. 지금 켜면 Formula가 설치되지 않는 것을 가리키게 된다. 워크플로에 자리를 만들되 켜는 시점을 사람이 정한다.

**`self-update`는 지금 만들지 않는다.** ADR 0012가 GitHub Releases 직접 다운로드 경로를 `self-update`의 근거로 들었으나, Homebrew 사용자는 `brew upgrade`로 충분하고 그 밖의 경로는 아카이브를 다시 받으면 된다. 여정 17(업그레이드와 사내망)이 오면 그때 만든다. **커맨드를 미리 만들어 두지 않는다.**

## 결정

**릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다.**

| 항목 | 값 |
|---|---|
| 계기 | `v*` 태그 푸시 |
| 플랫폼 | ADR 0007의 여섯 조합. `CGO_ENABLED=0` |
| 빌드 플래그 | `-trimpath`, `ldflags`로 `version` 주입 |
| 아카이브 | unix는 `tar.gz`, windows는 `zip` |
| 이름 | `engram_<version>_<os>_<arch>.<확장자>` |
| 체크섬 | `checksums.txt` 하나에 SHA256 |
| 동봉 | `README.md`, `LICENSE` |
| tap 갱신 | 설정에 두되 **공개 전환 전까지 끈다** |

- tap 토큰 시크릿이 없으면 **실패한다.** 절반만 성공하지 않는다.
- 릴리스 워크플로는 CI와 같은 파일에 두지 않는다. 계기와 권한이 다르다.
- 태그를 붙이기 전에 CI가 통과했는지 사람이 확인한다. 워크플로가 그것을 대신 판단하지 않는다.
- `self-update` 커맨드는 만들지 않는다. 여정 17에서 다시 본다.

## 결과

- Go 툴체인 없이 설치하는 경로가 생긴다. 여정 0의 이탈 지점이 닫힌다.
- `engram version`이 태그를 낸다. 지금은 `dev`다.
- tap 갱신을 켜는 것과 저장소를 공개로 돌리는 것이 한 묶음이 된다. 순서를 지켜야 Formula가 죽은 링크를 가리키지 않는다.
- goreleaser 설정이 저장소에 남으므로 릴리스 산출물의 구성이 문서 없이도 읽힌다.
- 코드 서명을 하지 않는다. macOS와 Windows에서 첫 실행 경고가 뜬다. ADR 0012가 winget과 scoop을 미룬 것과 같은 이유이며 서명 준비가 되면 다시 본다.

## 관련

- [0012 배포 경로를 개인 Homebrew tap으로 단일화한다](0012-distribution-via-personal-homebrew-tap.md) 배포 체계의 원안
- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) 당시 명칭. 플랫폼 매트릭스와 `CGO_ENABLED=0`
- [0001 프로젝트 목적과 범위](0001-purpose-and-scope.md) 설치 진입장벽이 단일 바이너리의 이유
