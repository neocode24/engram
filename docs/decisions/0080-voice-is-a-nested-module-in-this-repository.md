---
number: 0080
title: 음성은 이 저장소의 중첩 모듈이다
date: 2026-08-21
status: accepted
---

# 음성은 이 저장소의 중첩 모듈이다

## 배경

[0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)가 `engram-voice`를 별도 저장소로 정했다. 그 ADR이 같은 본문에서 이렇게 적었다.

> 받는 방식은 새로 만들지 않는다. `internal/embed/download.go`가 파일별 크기와 sha256 고정, Range 재개, 진행률, `--from` 오프라인 반입, `status --verify`를 이미 갖고 있다. **모델 표만 갈아 끼운다.**

**두 문장이 동시에 성립하지 않는다.** Go의 `internal` 규칙은 `github.com/neocode24/engram/internal/embed`를 그 경로 아래가 아닌 코드에서 import 할 수 없게 한다. 별도 저장소에서는 그 450줄을 복사하거나, `pkg/`로 승격해 공개 API로 만들어야 한다.

**engram에는 `pkg/`가 없다. 전부 `internal/`이다.** 이것은 사고가 아니라 자세다. 공개 API 표면이 없으면 라이브러리 소비자에게 지킬 약속이 생기지 않는다. 단일 바이너리 하나를 파는 물건에 그 약속이 붙으면 버전마다 시그니처를 지켜야 하고, 그 비용은 영구적이다. 저장소를 나누기 위해 그 자세를 버리는 것은 값이 맞지 않는다.

## 판단 근거

### 0079의 근거를 실측으로 다시 봤다

0079가 저장소를 나눈 근거는 이것이었다.

> "CGO 없음"은 이 저장소의 성격을 정하는 문장이다. 같은 저장소에 CGO 모듈이 들어오면 그 조항은 모듈 수준의 세부 사항으로 격하된다.

이 주장이 실제로 성립하는지 재 봤다. cgo 패키지와 순수 Go 커맨드를 한 저장소에 두고 측정한 결과다.

| 측정 | 결과 |
|---|---|
| `CGO_ENABLED=0 go build ./cmd/<코어>` (같은 모듈에 cgo 패키지 있음) | **통과** |
| `CGO_ENABLED=0 go build ./...` (외부 cgo 의존이 실재할 때) | **실패.** cgo 패키지에서 심볼 미정의 |
| 코어만 빌드했을 때 무거운 의존을 받는가 | **안 받는다.** 빈 모듈 캐시에 `github.com/` 아래가 하나도 안 생겼다 |

셋째가 중요하다. `go install .../cmd/engram@latest`가 sherpa-onnx를 끌고 오지 않는다. **코어 사용자는 음성 의존의 존재를 모른다.**

둘째는 CI 문구의 문제다. `go build ./...`의 범위를 좁히면 해결되며 그것은 격하가 아니라 명시다.

### 중첩 모듈이면 그 문제도 사라진다

`voice/`에 자기 `go.mod`를 두는 구성을 측정했다.

| 측정 | 결과 |
|---|---|
| 중첩 모듈이 루트의 `internal/`을 import 할 수 있는가 | **된다.** import 경로가 루트 모듈 경로를 접두사로 가지면 허용된다 |
| 루트의 `go build ./...`가 중첩 모듈을 건드리는가 | **안 건드린다.** `go list ./...`에 나오지 않는다 |

**루트 모듈이 완전히 그대로다.** `CGO_ENABLED=0`, `go build ./...`, `go test ./...`, CI 네 러너, goreleaser 여섯 플랫폼이 한 줄도 안 바뀐다. 그러면서 `voice/`는 `internal/embed`를 그대로 쓴다.

0079가 저장소 경계로 얻으려던 것을 모듈 경계가 전부 준다. 저장소를 나눌 이유가 남지 않는다.

### 같은 저장소여서 얻는 것

- **`internal/embed/download.go`를 복사하지 않는다.** 크기와 sha256 고정, Range 재개, 진행률, `--from`, `--verify`가 한 벌이다. 두 벌이면 갈라진다.
- **용어 사전 파싱이 `internal/doc`와 `internal/config`를 그대로 쓴다.** 사전은 위키 안에 살고 그 파싱은 이미 있는 것이다([0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)가 사전 소유를 위키로 정했다).
- **ADR이 한 줄기다.** 경계에 걸친 결정(사전 소유, `transcript` 타입)이 같은 번호 계열에 남는다. 저장소를 나누면 상호 참조가 저장소를 넘어간다.
- **교재의 클론이 하나다.** 0079가 "선택 항목이라 감당한다"고 적었으나 감당할 이유가 없어졌다.

### 남는 비용과 대응

**릴리스 태그가 둘이다.** 하위 디렉토리 모듈의 Go 태그 규약은 `voice/v0.1.0`이다. 루트는 `v1.0.0`을 그대로 쓴다.

**`voice/go.mod`에 `replace`를 두지 않는다.** `go install github.com/neocode24/engram/voice/cmd/engram-voice@latest`가 `replace`가 있는 모듈을 거절하기 때문이다. 루트를 정식 태그로 `require`하고, 로컬 개발은 루트의 `go.work`로 잇는다.

**CGO 교차 빌드는 어렵다.** 러너 하나에서 여섯 플랫폼을 cgo로 빌드할 수 없다. 플랫폼별 러너나 교차 툴체인이 필요하다. **이 비용은 저장소를 나눠도 같으므로 이 결정의 차이가 아니다.**

## 결정

**`engram-voice`는 이 저장소의 중첩 모듈이다.** [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)의 저장소 조항을 이 ADR이 대체한다. 나머지 조항은 그대로다.

| 항목 | 값 |
|---|---|
| 저장소 | **이 저장소.** 별도 저장소를 두지 않는다 |
| 모듈 | `voice/go.mod`. 루트와 별개 모듈 |
| 루트 모듈 | **바뀌지 않는다.** `CGO_ENABLED=0`, `go build ./...`, CI, goreleaser 그대로 |
| 코드 위치 | `voice/cmd/engram-voice/`, `voice/internal/` |
| 공유 | 루트의 `internal/`을 import 한다. `pkg/`를 만들지 않는다 |
| `replace` | **두지 않는다.** 정식 태그로 require 하고 로컬은 `go.work` |
| 릴리스 태그 | 루트 `vX.Y.Z`, 음성 `voice/vX.Y.Z` |

0079가 정한 나머지는 유효하다. 위키에 쓰지 않는 것, 용어 사전을 위키가 소유하는 것, 모델 배포 방식, 기본 whisper 크기가 미결인 것, 교재의 선택 세션이다.

## 결과

- 다운로드 기반이 한 벌로 유지된다. 복사본이 갈라지는 일이 없다.
- `pkg/`가 생기지 않는다. 공개 API 표면 없음이라는 자세가 유지된다.
- 코어 사용자는 음성 의존을 받지 않는다. 실측으로 모듈 캐시가 비어 있다.
- `go build ./...`가 루트에서 그대로 통과한다. 중첩 모듈이 패턴에 안 잡힌다.
- 태그가 둘이 되어 릴리스 워크플로가 둘로 갈린다. 계기는 여전히 태그 푸시다([0042](0042-release-artifacts-and-workflow.md)).
- `go.work`가 저장소에 생긴다. 커밋할지는 구현 때 정한다.

## 관련

- [0079 음성은 별도 저장소의 별도 바이너리이고 위키는 용어 사전을 소유한다](0079-voice-is-a-separate-binary-in-a-separate-repository.md) 저장소 조항만 이 ADR이 대체한다
- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) 루트 모듈이 지키는 조항
- [0011 저장소 구성과 Go 모듈명](0011-repo-layout-and-module-name.md) 저장소 루트를 모듈 루트로 삼은 결정
- [0042 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다](0042-release-artifacts-and-workflow.md) 태그가 둘이 되는 자리
