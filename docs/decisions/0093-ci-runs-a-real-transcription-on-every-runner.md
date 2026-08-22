---
number: 0093
title: CI가 러너마다 실제 전사를 한 번 돌린다
date: 2026-08-22
status: accepted
---

# CI가 러너마다 실제 전사를 한 번 돌린다

## 배경

[0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md)이 voice 를 세 러너에서 빌드하고 시험하게 했다. `docs/voice.md` 의 플랫폼 표는 그 뒤에도 이렇게 적고 있었다.

> **실제 오디오를 넣어 전사까지 돌려 본 것은 darwin/arm64 하나다.** 나머지는 빌드와 단위 시험까지다. 단위 시험은 모델을 열지 않으므로 전사 경로 자체는 CI 가 못 본다.

빌드가 되고 시험이 통과해도 그 플랫폼에서 전사가 도는지는 모른다. 시험이 모델 파일을 요구하지 않기 때문이다. 실제로 [0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md)이 잡은 Windows 의 `0xc0000135` 는 시험 실행 파일이 DLL 을 못 찾은 것이었고, 그것은 모델 없이도 드러났다. **모델을 열어야만 드러나는 결함은 아직 아무도 안 봤다.**

## 측정

### 큰 모델은 러너마다 받기 무겁다

기본 크기 large-v3 는 encoder 731MB 와 decoder 962MB 다. 세 러너에서 매번 받으면 5GB 다.

`small` 이 있다. encoder 107MB, decoder 250MB, tokens 0.8MB, 화자 분할 둘 33MB, VAD 0.6MB 로 합계 392MB 다. **적재와 디코딩 경로는 크기와 무관하게 같다.** 크기별로 다른 것은 파일 이름 접두사뿐이다([0081](0081-default-whisper-model-is-large-v3.md)).

### 표본에 정답이 붙어 있다

[0092](0092-diarization-thresholds-are-measured-against-real-recordings.md)가 쓴 sherpa-onnx 의 실제 사람 녹음을 그대로 쓴다. `1-two-speakers-en.wav` 는 16초에 500KB 이고 이름에 화자 둘이라고 적혀 있다.

로컬에서 돌려 봤다. small 모델로 3.0초가 걸리고 화자 둘을 정확히 냈다.

    {"source":"1-two-speakers-en.wav","model":"small","audioSeconds":16,
     "speakers":2,"speakersGiven":false,"unknownLines":0,"lines":[2개]}

### 본문은 뜻이 없다

전사 언어가 한국어로 박혀 있다([0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md)). 자동 감지에 맡기지 않기로 한 결정이다. 영어 오디오를 넣으면 한국어 음절로 받아쓴 글이 나온다.

    화자 1: 블가 가장 잘 어요. 션은 을 넣어 을 넣어

**이것으로 충분하다.** 여기서 보는 것은 정확도가 아니라 모델이 열리고 VAD 가 구간을 나누고 디코더가 글자를 내고 화자 분할이 도는가다. 정확도는 [0081](0081-default-whisper-model-is-large-v3.md)이 실제 한국어 녹음으로 따로 쟀다.

## 판단 근거

### 무엇을 단언할지 고른다

본문 글자를 단언하면 흔들린다. int8 양자화 추론은 CPU 마다 미세하게 다를 수 있고 모델이 바뀌면 글자도 바뀐다.

세 가지만 본다.

| 단언 | 무엇이 깨지면 걸리나 |
|---|---|
| `speakers == 2` | 화자 분할 모델 적재, 세그먼트, 군집 |
| `lines` 가 비어 있지 않음 | VAD 구간 나누기 |
| 본문이 하나라도 비어 있지 않음 | whisper 인코더와 디코더 |

화자 수는 표본의 정답이라 값을 박아도 임의값이 아니다.

### 한국어 오디오를 커밋하지 않는다

한국어 표본을 두면 본문까지 볼 수 있다. 만들 방법은 `say -v Yuna` 뿐인데 [0085](0085-synthetic-speech-is-not-teaching-material.md)가 그 산출물을 저장소에 두지 않기로 했다. 누군가 교재로 쓰기 때문이다.

받아서 쓰는 표본은 그 문제가 없다. 저장소에 남지 않는다.

### 검사는 node 로 한다

세 러너 전부에 node 가 있다. Windows 러너는 `python3` 이 PATH 에 있다는 보장이 없다.

## 결정

| 항목 | 값 |
|---|---|
| 잡 | `voice-transcribe`. ubuntu, macos, windows 셋 |
| 모델 | **small.** 392MB 이며 `actions/cache` 로 재사용한다 |
| 캐시 키 | `voice/internal/model/model.go` 해시. 파일 목록이나 체크섬이 바뀌면 무효 |
| 표본 | `1-two-speakers-en.wav`. 실행할 때 받고 커밋하지 않는다 |
| 단언 | 화자 2명, 전사 줄 있음, 본문 비어 있지 않음 |
| 안 보는 것 | **본문 글자.** 언어가 한국어로 박혀 있어 뜻이 없다 |

## 결과

- `docs/voice.md` 의 "실제 전사를 돌려 봤나" 열이 darwin/arm64, linux/amd64, windows/amd64 셋에서 켜진다.
- 릴리스만 하는 셋(darwin/amd64, linux/arm64)은 그대로 비어 있다. 그 러너가 CI 에 없다.
- 첫 실행에서 러너마다 392MB 를 받는다. 그다음부터는 캐시가 받는다.
- 잰 것은 large-v3 가 아니라 small 이다. 큰 모델에서만 나는 결함이 있으면 이 잡이 못 본다.

## 관련

- [0087 Windows 는 재서 통과했고 포장이 CI 에서 돈다](0087-windows-is-measured-green-and-packaging-runs-in-ci.md) 세 러너에서 voice 를 돌리기 시작한 결정. 이 ADR 이 전사 한 번을 더한다
- [0092 화자 분할 임계값을 실제 사람 녹음으로 잰다](0092-diarization-thresholds-are-measured-against-real-recordings.md) 같은 표본을 쓴다
- [0085 합성 음성은 음성 세션의 교재가 될 수 없다](0085-synthetic-speech-is-not-teaching-material.md) 한국어 표본을 커밋하지 않는 이유
