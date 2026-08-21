---
number: 0079
title: 음성은 별도 저장소의 별도 바이너리이고 위키는 용어 사전을 소유한다
date: 2026-08-21
status: accepted
---

# 음성은 별도 저장소의 별도 바이너리이고 위키는 용어 사전을 소유한다

## 배경

여정 2(음성)를 `docs/journeys.md:32`가 이렇게 규정한다.

> 전사, 화자 분리, 용어 사전 자동 치환, 사람이 오탈자 교정, 교정 내역이 사전에 피드백되어 다음 전사가 개선된다. 이 피드백 루프가 숨은 무기다. 원본 오디오는 커밋하지 않는다.

`design.md:179`가 이것을 "별도 트랙"으로 미뤄 두었고 [0068](0068-model-command-manages-embeddings-only.md)이 `engram model`의 관리 대상을 임베딩 하나로 못 박으며 STT를 범위 밖에 두었다. **무엇이 그 별도 트랙이 되는지는 정하지 않았다.** 이 ADR이 그것을 정한다.

upstream이 이미 이 파이프라인을 운영한다. 실측으로 확인한 구성이다.

| 역할 | 모델 | 런타임 |
|---|---|---|
| 전사 | `whisper-large-v3` | **MLX. Apple Silicon 전용** |
| 화자 분할 | `pyannote-segmentation-3.0` ONNX | sherpa-onnx (C++) |
| 화자 임베딩 | `3dspeaker campplus` ONNX | sherpa-onnx (C++) |

여기에 `uv`, Python 3.12 가상환경, `numpy`, `ffmpeg` 또는 `afconvert`가 붙는다. 그리고 **화자 분할 모델에 다운로드 경로가 없다.** upstream의 부트스트랩 스크립트는 그 모델을 특정 macOS 앱이 자기 컨테이너에 받아 둔 캐시에서 복사해 온다. 그 앱이 없으면 파이프라인이 서지 않는다.

## 판단 근거

### engram 안에 넣을 수 없다

[0007](0007-platform-and-distribution.md)이 코어를 순수 Go, `CGO_ENABLED=0`으로 못 박은 근거가 그대로 적용된다.

> cgo를 켜는 순간 "단일 바이너리, 설치 한 방"이라는 존재 이유가 무너진다. Windows에서는 DLL 배치, 백신 차단, MSVC 런타임 의존이 따라온다.

MLX는 Go로 갈 수 없고 Apple Silicon 밖으로도 못 나간다. 화자 분할은 순수 Go 대안이 존재하지 않는다. whisper를 ONNX로 순수 Go에서 돌리는 길은 계산량이 막는다. [0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md)의 실측에서 bge-m3가 2000자 문서 하나에 12.6초였고 그것은 인코더 한 번이다. whisper는 자기회귀 디코더라 자릿수가 다르며 한 시간 녹음이 대상이다.

**전사를 코어에 넣으면 여섯 플랫폼 매트릭스와 폐쇄망(여정 17)과 단일 바이너리가 한꺼번에 깨진다.**

### 같은 저장소에도 넣지 않는다

두 번째 모듈로 같은 저장소에 두는 길을 검토했다. 릴리스 파이프라인이 CGO 유무로 갈라져 복잡해지는 것이 표면적 비용이고, 그보다 큰 이유가 있다.

**"CGO 없음"은 [0007](0007-platform-and-distribution.md)부터 지켜 온 조항이고 이 저장소의 성격을 정하는 문장이다.** 같은 저장소에 CGO 모듈이 들어오면 그 조항은 모듈 수준의 세부 사항으로 격하되고, 저장소를 처음 여는 사람에게 더 이상 성립하지 않는다. 조항을 지키는 방법은 경계를 저장소로 긋는 것이다.

교재 쪽 비용은 클론 한 줄이다. 음성은 마지막 선택 항목이라 감당한다.

### 경계는 오디오와 텍스트 사이다

`engram-voice`는 **위키에 쓰지 않는다.** 오디오를 받아 전사 텍스트를 낸다. 그 텍스트를 위키에 넣는 것은 `engram capture`의 일이고 그 호출은 에이전트나 사람이 한다.

[0014](0014-llm-boundary-agent-drives-binary.md)가 LLM에 대해 정한 것과 같은 모양이다. 도구가 다른 도구를 부르지 않고 위에서 부른다. 이 경계를 지키면 결과가 둘이다. **음성 바이너리는 위키 규약을 하나도 알 필요가 없고**, 전사 결과가 위키로 들어가는 길이 다른 원문과 완전히 같아진다. `capture`는 이미 표준 입력을 받으므로 새로 만들 것이 없다.

### 용어 사전은 위키가 소유한다

여정 2가 "숨은 무기"라 부른 피드백 루프의 실체가 여기다. upstream은 사전을 두 자리에 쓴다. 전사 전에는 whisper의 `initial_prompt`로 넣어 인식을 유도하고, 전사 후에는 자동 교정 항목만 골라 치환한다. 실측으로 upstream 사전에 자동 교정 항목이 89개이며 `review`와 조건부 항목은 치환하지 않는다.

**사전이 위키 안에 산다.** 사람이 오탈자를 고치면 그 교정이 사전에 쌓이고 git이 추적하며 다음 전사가 개선된다. 사전이 도구 안에 있으면 이 루프가 끊긴다. 위키마다 용어가 다르므로 도구가 들고 다닐 수도 없다.

그래서 소유는 위키, 읽기는 양쪽이다. `engram-voice`가 위키 경로를 받아 사전을 읽고, engram 쪽에서는 `rules show`가 그것을 낸다. **읽기만 하므로 앞 절의 경계를 어기지 않는다.**

### 모델은 받아진다

upstream이 앱 캐시에서 복사하는 것은 sherpa-onnx의 공개 배포를 찾지 못했기 때문으로 보인다. 실측으로 전부 인증 없이 받아진다.

| 파일 | 크기 | 호스트 |
|---|---|---|
| pyannote segmentation 3.0 `model.onnx` | 5,992,913 | HF `csukuangfj/sherpa-onnx-pyannote-segmentation-3-0` |
| `3dspeaker_campplus_sv_zh_en_16k` | 28,281,164 | GitHub releases `speaker-recongition-models` |
| `large-v3-encoder.int8.onnx` | 766,671,985 | HF `csukuangfj/sherpa-onnx-whisper-large-v3` |
| `large-v3-decoder.int8.onnx` | 1,008,265,203 | 〃 |
| `large-v3-tokens.txt` | 816,730 | 〃 |

pyannote 원본은 MIT이나 HuggingFace에서 게이트되어 연락처 제공 동의와 토큰을 요구한다. **MIT가 재배포를 허용하므로 게이트 없는 변환본에서 받는다.** sherpa-onnx는 Apache-2.0, whisper 가중치는 MIT다.

**전사와 화자 분할을 sherpa-onnx 하나로 통일한다.** upstream이 MLX와 sherpa-onnx 둘을 쓰는 구성에서 MLX를 빼면 Python과 `uv`와 `numpy`가 함께 빠지고 Apple Silicon 제약도 사라진다. sherpa-onnx는 플랫폼별 사전 빌드 라이브러리를 담은 Go 바인딩을 배포하므로 사용자가 C++을 빌드하지 않는다.

받는 방식은 새로 만들지 않는다. `internal/embed/download.go`가 파일별 크기와 sha256 고정, Range 재개, 진행률, `--from` 오프라인 반입, `status --verify`를 이미 갖고 있다. **모델 표만 갈아 끼운다.** 다만 호스트가 둘이므로 기준 URL을 파일별로 두어야 하고, GitHub releases의 `.tar.bz2` 대신 HuggingFace의 단일 `model.onnx`를 받아 아카이브 처리를 두지 않는다.

### 기본 모델 크기는 지금 정하지 않는다

int8 기준으로 large-v3가 1.72GB, medium이 0.90GB, small이 0.36GB다. engram 본체가 이미 bge-m3 2.2GB를 사전 과제로 요구하므로 기본값 선택이 수강생의 내려받기 총량을 좌우한다.

한국어 회의 녹음에서 large-v3와 medium의 차이는 대체로 고유명사와 전문용어에 몰리는데 **그 부분은 용어 사전 교정이 다시 잡는다.** 자동 교정 항목이 89개인 이유가 그것이다. 반면 int8 large-v3 디코더 1GB를 CPU에서 자기회귀로 돌리는 비용은 크다.

**측정하지 않았으므로 정하지 않는다.** [0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md)가 모델을 하한별 쌍 개수로 실측해 고른 것과 같은 근거가 필요하다. 실제 한국어 녹음으로 두 모델을 돌려 비교하는 것을 구현의 첫 단계로 둔다.

## 결정

**`engram-voice`는 별도 저장소의 별도 바이너리다.**

| 항목 | 값 |
|---|---|
| 저장소 | 별도. `engram` 저장소에 두지 않는다 |
| CGO | **켠다.** sherpa-onnx 바인딩이 요구한다 |
| 런타임 | sherpa-onnx 하나. Python 과 MLX 를 쓰지 않는다 |
| 하는 일 | 오디오를 받아 전사 텍스트를 낸다. 화자 분할과 용어 치환을 포함한다 |
| **위키 쓰기** | **없다.** `capture` 를 부르지 않고 위키 파일을 만들지 않는다 |
| 용어 사전 | **위키가 소유한다.** 읽기만 한다 |
| 모델 배포 | `internal/embed/download.go` 의 규율을 그대로 쓴다. 크기와 sha256 고정, Range 재개, `--from` |
| 기본 모델 크기 | **미결.** 실측 후 정한다 |
| 교재 | 8세션 이후의 선택 항목. 필수 경로에 넣지 않는다 |

`engram` 본체는 바뀌지 않는다. `CGO_ENABLED=0`, 여섯 플랫폼, 폐쇄망, 단일 바이너리가 그대로다.

**engram 쪽에 남는 이음매가 하나 있다.** `type` 닫힌 집합에 `transcript`가 없다. 지금 값은 `source-raw`와 `source-summary`이고 화자가 갈린 전사는 둘 중 어느 쪽도 아니다. upstream은 `type: transcript`를 쓴다. 이 값을 더할지는 전사 산출물을 실제로 승급해 본 뒤 판단한다.

## 결과

- 코어의 "CGO 없음"이 저장소 수준에서 유지된다. 저장소를 여는 사람에게 그 문장이 그대로 성립한다.
- 음성 파이프라인이 Apple Silicon을 벗어난다. upstream은 MLX 때문에 Mac 전용이었다.
- upstream의 앱 캐시 의존이 사라진다. 모델 셋이 전부 공개 URL이다.
- 전사 결과가 위키로 들어가는 길이 다른 원문과 같다. `capture` 가 표준 입력을 받으므로 새 경로가 없다.
- 용어 사전이 위키 자산이 되어 git 이력에 교정 내역이 쌓인다.
- 수강생은 클론을 하나 더 한다. 선택 항목이므로 필수 경로의 길이는 그대로다.
- 저장소가 둘이 되어 릴리스와 이슈가 나뉜다. 버전도 각자 매긴다.

## 관련

- [0007 플랫폼과 배포, 코어와 시맨틱 층 분리](0007-platform-and-distribution.md) CGO 없음 조항의 출처
- [0068 model 커맨드는 임베딩만 관리한다](0068-model-command-manages-embeddings-only.md) STT를 범위 밖에 둔 결정
- [0014 LLM 호출을 바이너리에 두지 않고 에이전트가 바이너리를 부른다](0014-llm-boundary-agent-drives-binary.md) 도구가 도구를 부르지 않는 같은 경계
- [0074 임베딩은 순수 Go로 돌리고 모델은 bge-m3 fp32로 고정한다](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md) 모델 선택을 실측으로 닫는 방식
- [0051 sources는 원본과 정제본을 함께 담고 type이 그 둘을 가른다](0051-sources-holds-originals-and-refined-summaries.md) `transcript` 이음매가 닿는 자리
