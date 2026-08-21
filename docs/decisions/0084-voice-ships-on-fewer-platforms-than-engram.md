---
number: 0084
title: engram-voice는 본체보다 적은 플랫폼에 배포하고 릴리스 경로가 다르다
date: 2026-08-21
status: accepted
---

# engram-voice는 본체보다 적은 플랫폼에 배포하고 릴리스 경로가 다르다

## 배경

[0081](0081-default-whisper-model-is-large-v3.md)이 배포 형태를 정하며 이렇게 적었다.

> **따라서 `engram-voice`의 아카이브는 바이너리 하나가 아니라 바이너리와 `lib/` 둘이다.** engram 본체와 다른 점이며 교재와 릴리스 문서에 그대로 적는다.

형태만 정했고 어느 플랫폼에 내는지와 워크플로가 어떻게 갈리는지는 정하지 않았다. 본체는 [0007](0007-platform-and-distribution.md)이 `CGO_ENABLED=0`으로 여섯 조합을 내기로 했고 [0042](0042-release-artifacts-and-workflow.md)가 goreleaser 한 잡으로 그것을 만든다. **CGO 를 켜는 순간 그 방식이 성립하지 않는다.**

## 측정

### 지금 상태는 두 갈래로 갈린다

소스에서 빌드하는 길은 손대지 않아도 돈다. `go install ./cmd/engram-voice` 로 만든 바이너리가 다른 디렉토리에서 실행된다. rpath 에 박힌 모듈 캐시 경로가 빌드한 기계에는 실제로 있기 때문이다.

    path /Users/<홈>/go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-macos@v1.13.6/lib/aarch64-apple-darwin

**교재가 안내하는 길은 소스 빌드다.** 그러므로 수강생 경로는 지금도 막혀 있지 않다. 막힌 것은 미리 빌드한 아카이브를 내려받는 길이다.

### 플랫폼마다 rpath 사정이 다르다

바인딩의 cgo 지시자를 읽었다.

| 플랫폼 | rpath 를 박는가 | 라이브러리 찾는 법 |
|---|---|---|
| macOS | 박는다. `${SRCDIR}/lib/<트리플>` | rpath |
| Linux | 박는다. `${SRCDIR}/lib/<트리플>` | rpath |
| Windows | **박지 않는다** | exe 옆의 DLL |

Windows 는 포장할 때 DLL 을 exe 옆에 두면 되고 rpath 를 지울 일이 없다. 나머지 둘은 상대 rpath 를 더하고 모듈 캐시 경로를 지워야 한다.

### 포장한 아카이브를 실제로 검증했다

`voice/scripts/package.sh`를 만들어 darwin/arm64 로 돌리고 결과를 쟀다.

| 항목 | 값 |
|---|---|
| 아카이브 | 14,323,788 바이트 |
| 풀어서 | 39.8 MiB |
| 바이너리 | 8.6 MiB |
| `lib/` | dylib 셋. onnxruntime 27 MiB, sherpa-onnx-c-api 4.0 MiB, sherpa-onnx-cxx-api 159 KiB |

압축을 풀어 저장소 바깥의 임시 디렉토리에서 돌렸다.

- `LC_RPATH`가 `@executable_path/lib` 하나만 남았다. 모듈 캐시 경로가 없다.
- 바이너리에서 빌더의 홈 경로 문자열이 **0건**이다.
- `engram-voice version`과 `model status`가 돈다.

`install_name_tool`이 서명을 깨뜨리므로 다시 서명한다. 이것을 빠뜨리면 macOS 가 실행을 SIGKILL 로 죽인다. 이 세션에서 실제로 겪은 실패다.

### voice 는 아무 검사도 받고 있지 않았다

중첩 모듈이라([0080](0080-voice-is-a-nested-module-in-this-repository.md)) 루트의 `./...` 패턴에 잡히지 않는다. 워크스페이스가 켜져 있어도 그렇다는 것을 확인했다. 루트 CI 를 안 깨뜨린다는 점에서는 의도한 대로지만, **`ci.yml`에 voice 잡이 없어 이 모듈은 push 마다 아무 검사도 받지 않고 있었다.**

## 판단 근거

### 교차 빌드를 포기한다

CGO 는 대상 플랫폼의 C 툴체인을 요구하고, sherpa-onnx 는 플랫폼마다 다른 모듈로 미리 빌드된 라이브러리를 들고 온다. 한 러너에서 여섯 조합을 내는 [0042](0042-release-artifacts-and-workflow.md)의 방식이 성립하지 않는다.

대신 대상마다 그 플랫폼의 러너를 쓴다. 러너 수가 대상 수와 같아지므로 **대상을 늘리는 비용이 본체와 다르다.** 본체는 조합을 하나 더하는 데 빌드 한 번이고 여기서는 러너 하나다.

### Windows 를 넣지 않는다

`sherpa-onnx-go-windows`는 mingw 트리플(`x86_64-pc-windows-gnu`)의 라이브러리를 들고 온다. 러너 이미지에 cgo 가 쓸 수 있는 mingw gcc 가 있는지 **재지 않았다.**

**안 된다고 판단한 것이 아니라 재지 않은 것이다.** 태그를 밀어야 확인되는 것을 된다고 적어 두면 릴리스가 절반만 성공한다. 잰 뒤에 넣는다.

오디오 변환기는 걸림돌이 아니다. `ffmpeg`가 PATH 에 있으면 되고 그 조건은 Linux 와 같다.

### 본체 릴리스 뒤에 올린다

`gh release upload`는 이미 있는 릴리스에 파일을 더한다. 그 릴리스를 goreleaser 가 만들므로 순서가 있다. `needs: release`로 묶는다.

체크섬은 아카이브마다 `.sha256` 파일을 따로 낸다. goreleaser 의 `checksums.txt`는 자기가 만든 아카이브만 담으며 이 아카이브는 그 바깥에서 만들어진다. 하나로 합치려면 goreleaser 에 CGO 매트릭스를 밀어 넣어야 하는데 그것이 이 ADR 이 피하려는 것이다.

### 소스 빌드 경로는 건드리지 않는다

교재는 `go install` 하나만 안내한다. 그 길이 지금 돌고 이 변경이 그것을 바꾸지 않는다. 포장 스크립트는 배포 산출물을 만드는 자리이고 수강생에게 안내하지 않는다.

## 결정

| 항목 | 값 |
|---|---|
| CI | `ci.yml`에 voice 잡을 둔다. `ubuntu-latest`, `macos-latest` |
| 릴리스 대상 | darwin arm64, darwin amd64, linux amd64, linux arm64 **넷** |
| Windows | **넣지 않는다.** 재지 않았기 때문이며 재면 넣는다 |
| 만드는 법 | `voice/scripts/package.sh <버전> <출력>`. 대상 플랫폼의 러너에서 돈다 |
| 교차 빌드 | 막는다. 스크립트가 호스트와 대상이 다르면 멈춘다 |
| 아카이브 | 바이너리, `lib/`, README, LICENSE. Windows 는 DLL 을 exe 옆에 |
| rpath | macOS `@executable_path/lib`, Linux `$ORIGIN/lib`. 모듈 캐시 경로는 지운다 |
| 서명 | macOS 는 `install_name_tool` 뒤에 ad-hoc 재서명한다 |
| 체크섬 | 아카이브마다 `.sha256` 파일 |
| 순서 | `needs: release`. 본체 릴리스가 만든 릴리스에 올린다 |
| 소스 빌드 | 그대로다. `go install` 이 돌고 교재를 바꾸지 않는다 |

## 결과

- voice 모듈이 push 마다 두 러너에서 빌드와 시험을 받는다. 지금까지는 아무 검사도 없었다.
- 릴리스 아카이브가 본체 여섯에 voice 넷이 더해진다. 이름 규칙은 `engram-voice_<버전>_<os>_<arch>` 로 본체와 같은 꼴이다.
- **darwin/arm64 만 실측으로 검증했다.** 나머지 셋은 태그를 밀기 전까지 확인되지 않는다. 첫 태그에서 이 잡이 깨질 수 있고 그때 본체 릴리스는 이미 성공해 있다.
- Windows 사용자는 소스에서 빌드해야 한다. 그 길이 되는지도 재지 않았다.
- `lib/` 가 40 MiB 가까이 되므로 내려받기가 본체보다 크다. 모델 1.72GB 에 비하면 작다.

## 관련

- [0081 기본 whisper 모델은 large-v3이고 배포는 단일 바이너리가 아니다](0081-default-whisper-model-is-large-v3.md) 배포 형태를 정한 결정. 이 ADR 이 그 플랫폼과 경로를 채운다
- [0080 음성은 이 저장소의 중첩 모듈이다](0080-voice-is-a-nested-module-in-this-repository.md) 루트 패턴에 안 잡히는 이유
- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) 본체가 여섯을 내는 근거. CGO 를 끄기로 한 자리
- [0042 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다](0042-release-artifacts-and-workflow.md) 본체 릴리스 경로. 이 잡이 그 뒤에 붙는다
