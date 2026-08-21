---
number: 0087
title: Windows는 재서 통과했고 포장까지 CI에서 돌린다
date: 2026-08-21
status: accepted
---

# Windows는 재서 통과했고 포장까지 CI에서 돌린다

## 배경

[0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md)이 Windows 를 CI 에 넣으며 이렇게 적었다.

> 녹색이 되면 `continue-on-error` 를 떼고 릴리스 대상에 넣는다. 그때 이 ADR 을 개정한다.

쟀다. 이 ADR 이 그 개정이다.

같은 자리에서 CI 자체가 나흘 동안 죽어 있었다는 것도 드러났다.

## 측정

### CI 가 나흘 동안 아무것도 검증하지 않고 있었다

최근 100건 중 88건이 실패였고 마지막 성공이 2026-08-17 09:52 다. 실패는 전부 같은 이유다.

    The job was not started because recent account payments have failed
    or your spending limit needs to be increased

**코드와 무관하다.** 모든 잡이 2초에서 6초 만에 죽었다. 비공개 저장소라 Actions 분이 과금 대상인데 결제 쪽에서 막힌 것이다.

즉 `voice/` 모듈을 세운 뒤의 모든 커밋이 **어떤 CI 검증도 받지 못했다.** [0084](0084-voice-ships-on-fewer-platforms-than-engram.md)가 "voice 모듈이 push 마다 두 러너에서 빌드와 시험을 받는다"고 적은 결과가 실제로는 일어나지 않았다.

저장소를 공개로 바꾸자 표준 러너가 무료가 되어 그 자리에서 풀렸다.

### Windows 는 빌드되고 시험만 죽었다

공개 전환 뒤 첫 실행 결과다.

| 잡 | 결과 |
|---|---|
| voice (ubuntu-latest) | 통과 |
| voice (macos-latest) | 통과 |
| voice (windows-latest) | 빌드 통과, vet 통과, **시험 실패** |

실패 코드가 `0xc0000135` 다. `STATUS_DLL_NOT_FOUND` 다.

깨진 것은 sherpa 를 링크하는 `cmd/engram-voice` 와 `internal/stt` 둘이고, 순수 Go 인 `internal/model` 과 `internal/glossary` 는 통과했다.

**[0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md)이 예측한 그대로다.** Windows 는 rpath 가 없어 exe 옆의 DLL 만 보는데 `go test` 가 만드는 임시 실행 파일 옆에는 DLL 이 없다.

모듈의 `lib/x86_64-pc-windows-gnu` 를 `GITHUB_PATH` 에 넣자 **통과했다.** 러너의 gcc 15.2.0 으로 cgo 빌드가 되고 시험도 돈다.

## 판단 근거

### Windows 를 정식 대상으로 올린다

빌드, 정적 검사, 시험 셋이 다 통과한다. [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md)이 건 조건이 충족됐다.

`continue-on-error` 를 뗀다. 지원한다고 선언한 이상 깨지면 막아야 한다.

배포 아카이브 쪽에는 이 문제가 없다. `package.sh` 가 DLL 을 exe 옆에 두므로 rpath 가 필요 없다. **CI 의 문제는 `go test` 가 만드는 임시 파일에만 있었다.**

### 포장을 CI 에서 돌린다

이번 일에서 배운 것이다. [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md)이 이미 같은 논지를 폈다.

> 릴리스 워크플로는 태그를 밀 때만 돈다. 거기서 처음 실패하면 본체 릴리스는 이미 성공해 있고 반쪽짜리 릴리스가 남는다.

그 논지를 빌드에만 적용하고 포장에는 적용하지 않았다. **포장 스크립트도 태그 때 처음 도는 코드다.** darwin/arm64 에서 손으로 한 번 돌려 본 것이 전부였고 linux 와 windows 경로는 아무도 실행한 적이 없다.

`package.sh` 를 CI 의 voice 잡에 넣는다. 릴리스가 부르는 것과 같은 스크립트를 같은 플랫폼에서 push 마다 돌린다. patchelf 설치도 함께 옮겨 릴리스와 CI 의 전제를 같게 둔다.

산출물은 버린다. 여기서 확인하려는 것은 **스크립트가 끝까지 도는가**이지 아카이브 자체가 아니다.

### zip 이 없을 수 있다

Windows 러너에 `zip` 이 있다고 가정할 수 없다. PowerShell 의 `Compress-Archive` 로 물러선다. PowerShell 은 Windows 에 늘 있다.

### 올릴 파일을 확장자로 나열하지 않는다

릴리스 업로드가 `dist/*.tar.gz dist/*.sha256` 이었다. Windows 아카이브는 `.zip` 이라 **조용히 빠진다.** `dist/engram-voice_*` 로 바꿔 접두사로 고른다.

`bash` 를 명시한다. Windows 러너의 기본 셸은 PowerShell 이고 스크립트는 bash 다.

## 결정

| 항목 | 값 |
|---|---|
| Windows CI | **정식 대상.** `continue-on-error` 를 뗀다 |
| Windows 릴리스 | **넣는다.** windows/amd64 |
| 릴리스 대상 | darwin arm64/amd64, linux amd64/arm64, windows amd64 **다섯** |
| DLL 경로 | CI 시험 전에 모듈 lib 디렉토리를 `GITHUB_PATH` 에 넣는다 |
| 포장 | **CI 의 voice 잡에서 돌린다.** 릴리스와 같은 스크립트, 같은 플랫폼 |
| 압축 | `zip` 이 없으면 PowerShell `Compress-Archive` |
| 업로드 | `engram-voice_*` 접두사. 확장자로 나열하지 않는다 |
| 셸 | 포장과 업로드 스텝에 `shell: bash` 를 명시한다 |

## 결과

- 릴리스 대상이 넷에서 다섯이 된다. Windows 사용자가 미리 빌드한 아카이브를 받는다.
- 포장 스크립트가 push 마다 세 플랫폼에서 돈다. 태그 때 처음 도는 코드가 줄었다.
- **저장소가 공개로 바뀌었다.** 표준 러너가 무료가 되어 CI 가 살아났다. 공개 전환의 첫 실질 효과가 이것이다.
- [0084](0084-voice-ships-on-fewer-platforms-than-engram.md)가 적은 "push 마다 검사를 받는다"가 이제 사실이다. 그 전까지는 아니었다.
- 결제로 CI 가 죽으면 **모든 잡이 초 단위로 실패한다.** 빨간불의 원인이 코드가 아닐 수 있다는 것을 기억한다.

## 관련

- [0086 러너 라벨을 실제로 확인하고 Windows는 CI에서 먼저 잰다](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md) 이 ADR 이 그 조건을 충족해 개정한다
- [0084 engram-voice는 본체보다 적은 플랫폼에 배포하고 릴리스 경로가 다르다](0084-voice-ships-on-fewer-platforms-than-engram.md) 대상 목록의 출처. 다섯으로 늘어난다
- [0042 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다](0042-release-artifacts-and-workflow.md) 태그가 유일한 계기라는 것이 이 위험의 근원
