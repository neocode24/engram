---
number: 0086
title: 러너 라벨을 실제로 확인하고 Windows는 CI에서 먼저 잰다
date: 2026-08-21
status: accepted
---

# 러너 라벨을 실제로 확인하고 Windows는 CI에서 먼저 잰다

## 배경

[0084](0084-voice-ships-on-fewer-platforms-than-engram.md)가 `engram-voice`의 릴리스 대상 넷과 Windows 제외를 정하며 이렇게 적었다.

> **안 된다고 판단한 것이 아니라 재지 않은 것이다.** 태그를 밀어야 확인되는 것을 된다고 적어 두면 릴리스가 절반만 성공한다. 잰 뒤에 넣는다.

그 ADR 은 러너 라벨 자체를 확인하지 않았다. 확인해 보니 하나가 존재하지 않는 라벨이었다.

## 측정

`actions/runner-images` 저장소의 이미지 목록과 이미지별 소프트웨어 명세를 읽었다.

### `macos-13` 은 없다

[0084](0084-voice-ships-on-fewer-platforms-than-engram.md)가 darwin/amd64 에 `macos-13` 을 썼다. **2025년 9월 22일부터 폐기가 시작되어 12월에 완전히 없어졌다.** 태그를 밀면 그 잡은 이미지를 못 찾고 즉시 실패한다.

| 대체 라벨 | 아키텍처 | 성격 |
|---|---|---|
| `macos-15-intel` | x64 | **표준 러너.** 공개 저장소 무료 |
| `macos-15-large`, `macos-latest-large` | x64 | larger runner. 공개 저장소라도 과금 |
| `macos-latest`, `macos-15` | arm64 | 표준 러너 |

GitHub 이 오픈소스 수요를 이유로 `macos-15-intel` 을 표준 러너로 따로 냈다. `-large` 계열만 있었으면 공개 저장소에서 Intel 빌드를 무료로 못 돌린다.

**Intel 자체가 2027년 가을에 끝난다.** macOS 15 이미지가 마지막 x86_64 이며 그 뒤로 Actions 에 x86_64 macOS 가 없다.

### Windows 러너에 gcc 가 있다

| 이미지 | gcc | MSYS2 |
|---|---|---|
| `windows-2025`(=`windows-latest`) | **15.2.0** (Chocolatey) | `C:\msys64`. **PATH 에 없음** |
| `windows-2022` | 14.2.0 | `C:\msys64`. PATH 에 없음 |

`sherpa-onnx-go-windows` 가 쓰는 트리플이 `x86_64-pc-windows-gnu` 이고 Chocolatey 의 gcc 가 mingw-w64 계열이므로 **ABI 가 맞을 법하다.**

다만 빌드가 되는 것과 시험이 도는 것은 다르다. Windows 는 rpath 가 없어 DLL 을 exe 옆에서 찾는데, `go test` 가 만드는 임시 실행 파일 옆에는 DLL 이 없다. 링크는 되고 실행이 안 될 수 있다.

### windows/arm64 는 라이브러리가 없다

모듈에 트리플이 둘뿐이다.

    i686-pc-windows-gnu
    x86_64-pc-windows-gnu

arm64 가 없다. 루트 모듈은 `windows-11-arm` 에서 도는데([0007](0007-platform-and-distribution.md)) voice 는 못 돈다.

## 판단 근거

### 문서를 읽는 것도 재는 것이다

[0084](0084-voice-ships-on-fewer-platforms-than-engram.md)가 Windows 를 뺀 이유는 "재지 않았다"였는데, 그 ADR 은 러너 명세를 읽는 것조차 하지 않았다. **읽었으면 gcc 가 있다는 것과 `macos-13` 이 없다는 것을 그 자리에서 알았다.**

돌려 보는 것만 측정이라고 여기면 읽으면 알 수 있는 것을 미결로 남긴다. 그 결과가 존재하지 않는 라벨을 워크플로에 넣은 것이다.

### 그래도 릴리스에 바로 넣지 않는다

명세에 gcc 가 있다는 것은 **빌드가 될 법하다**까지다. 링크와 DLL 해결은 돌려 봐야 안다.

릴리스 워크플로는 태그를 밀 때만 돈다. 거기서 처음 실패하면 본체 릴리스는 이미 성공해 있고 반쪽짜리 릴리스가 남는다.

**CI 는 push 마다 돈다.** 같은 것을 훨씬 싸게 확인할 수 있다. CI 에 넣고 결과를 본 뒤 릴리스 대상에 올린다.

### 실패해도 main 을 막지 않는다

`continue-on-error` 로 둔다. 아직 지원한다고 선언한 플랫폼이 아니므로 그것이 빨갛다고 다른 작업이 멈출 이유가 없다. **결과를 보려는 것이지 게이트를 세우려는 것이 아니다.**

녹색이 되면 `continue-on-error` 를 떼고 릴리스 대상에 넣는다. 그때 이 ADR 을 개정한다.

### darwin/amd64 는 남긴다

`macos-15-intel` 이 표준 러너라 공개 저장소에서 무료다. 라벨만 고치면 된다.

2027년 가을에 없어지지만 그때 빼면 된다. 지금 미리 빼면 Intel Mac 사용자가 그때까지 소스 빌드를 해야 한다.

## 결정

| 항목 | 값 |
|---|---|
| darwin/amd64 러너 | `macos-13` 에서 **`macos-15-intel`** 로. 표준 러너이며 무료 |
| Intel 종료 | 2027년 가을. 그때 대상에서 뺀다 |
| Windows | **CI 에 넣어 잰다.** `windows-latest`, `continue-on-error` |
| Windows 릴리스 | CI 가 녹색이 된 뒤에 넣는다. 지금은 아니다 |
| windows/arm64 | **넣지 않는다.** 라이브러리가 없다. 재고의 여지가 없다 |
| 라벨 확인 | 워크플로에 라벨을 쓰기 전에 `actions/runner-images` 목록에서 확인한다 |

## 결과

- [0084](0084-voice-ships-on-fewer-platforms-than-engram.md)의 릴리스 매트릭스가 첫 태그에서 절반 실패할 뻔한 것을 막았다.
- voice CI 가 러너 셋에서 돈다. 그중 하나는 실패해도 넘어간다.
- Windows 지원 여부가 다음 push 의 CI 결과로 정해진다.
- 2027년 가을에 darwin/amd64 를 뺄 일정이 생겼다.

## 관련

- [0084 engram-voice는 본체보다 적은 플랫폼에 배포하고 릴리스 경로가 다르다](0084-voice-ships-on-fewer-platforms-than-engram.md) 이 ADR 이 그 라벨을 고치고 Windows 항목을 연다
- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) 루트가 도는 러너. voice 가 못 따라가는 자리
- [0081 기본 whisper 모델은 large-v3이고 배포는 단일 바이너리가 아니다](0081-default-whisper-model-is-large-v3.md) CGO 때문에 러너를 나눠야 하는 근원
