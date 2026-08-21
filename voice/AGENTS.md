# AGENTS.md. voice 모듈 작업 계약

저장소 루트의 `CLAUDE.md`가 먼저다. 이 문서는 그것과 **다른 점만** 적는다.

## 이것이 무엇인가

`engram-voice`는 오디오를 전사 텍스트로 바꾼다. 루트 모듈과 별개인 Go 모듈이며 같은 저장소에 산다([0080](../docs/decisions/0080-voice-is-a-nested-module-in-this-repository.md)).

## 루트와 다른 것

| 항목 | 루트 | voice |
|---|---|---|
| CGO | **끈다** (`CGO_ENABLED=0`) | **켠다.** sherpa-onnx 바인딩이 요구한다 |
| 배포 | 단일 바이너리 | **바이너리와 `lib/` 둘.** dylib 이 따라온다([0081](../docs/decisions/0081-default-whisper-model-is-large-v3.md)) |
| 외부 프로세스 | 없다 | `afconvert` 또는 `ffmpeg` 를 부른다 |
| 모듈 | `github.com/neocode24/engram` | `github.com/neocode24/engram/voice` |

**루트 모듈을 깨뜨리지 마라.** 아래 셋이 통과해야 한다. 중첩 모듈은 루트의 `./...` 패턴에 잡히지 않으므로 그냥 돌리면 된다.

    CGO_ENABLED=0 go build ./...
    go vet ./...
    go test ./...

## 넘지 말 것

### 위키에 쓰지 않는다

**이 바이너리는 위키 파일을 만들거나 고치지 않는다.** `engram capture` 를 부르지도 않는다. 전사 결과를 표준 출력으로 낼 뿐이고 위키에 넣는 것은 사람이나 에이전트가 `engram capture` 로 한다([0079](../docs/decisions/0079-voice-is-a-separate-binary-in-a-separate-repository.md)).

그래서 이 모듈은 위키 규약을 알 필요가 없다. 프론트매터도 슬러그도 승급도 여기 없다. 그 코드를 여기 들이려는 생각이 들면 경계를 넘고 있는 것이다.

용어 사전만 예외이며 **읽기만 한다.** 사전은 위키가 소유한다. 파서는 루트의 `internal/glossary` 에 있고 `engram rules show` 도 같은 것을 쓴다. **여기에 사전 파서를 새로 쓰지 마라.**

### 표준 출력은 전사 결과만 담는다

진행률, 경고, 안내는 전부 표준 오류로 낸다. 아래가 성립해야 하기 때문이다.

    engram-voice transcribe 회의.m4a --speakers 3 | engram capture --title "회의"

### 추정치에 추정이라고 적는다

화자 수를 자동으로 잡으면 그 값은 믿을 수 없다([0082](../docs/decisions/0082-speaker-count-is-asked-not-guessed.md)). 실측으로 upstream 에서 120분 녹음이 화자 132명을 냈다. 경고를 지우지 마라. 산출물에도 남긴다. 이 전사가 그대로 위키 문서가 되므로 나중에 읽는 사람도 알아야 한다.

### 정답 없이 임계값을 만지지 않는다

화자 분할의 파편 필터 하한, 군집 임계값, 줄 병합 상한은 전부 검증되지 않은 값이다. 결과가 나아 보이게 숫자를 바꾸는 것은 측정이 아니다. 누가 언제 말했는지의 정답이 있는 자료로만 다시 잰다.

## 모델

크기와 sha256 을 고정한다. 표는 `internal/model/model.go` 에 있고 값은 전부 실제로 받아 계산했다. 받고 검증하는 일은 루트의 `internal/modelfetch` 가 한다. **여기에 다운로드 코드를 새로 쓰지 마라.**

호스트가 둘이라 `ModelFile` 마다 `Base` 가 있어야 한다. 비면 URL 이 `/경로` 가 되어 실패한다. 시험이 그것을 잡는다.

## 측정

`cmd/measure` 는 모델과 설정을 견주는 자리다. 정식 커맨드가 아니며 결과를 ADR 에 남기는 데 쓴다. 사용자에게 안내하지 않는다.

**커버리지를 재지 않으면 측정이 조용히 틀린다.** 구간 나누기가 오디오의 61%만 덮은 채로 모델 둘을 비교한 적이 있고, 놓친 대목이 어려운 쪽에 몰려 작은 모델에 유리하게 기울었다([0081](../docs/decisions/0081-default-whisper-model-is-large-v3.md)).

## 시험

모델을 여는 시험을 쓰지 마라. 1.8GB 를 받아야 돌고 CI 에서 성립하지 않는다. 군집 결과를 다듬는 판단, 인자 재배열, 출력 형식처럼 **모델과 무관한 순수 계산**에 시험을 건다.

## 공개 경계

측정에 쓰는 녹음은 사용자의 실제 회의이며 `private-local-only` 다. **전사 내용을 저장소에 남기지 마라.** ADR 과 커밋 메시지에 들어가는 것은 수치뿐이다. 파일 이름과 경로도 식별자다.
