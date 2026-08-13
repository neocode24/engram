---
number: 0007
title: 플랫폼과 배포, 코어와 시맨틱 층 분리
date: 2026-08-13
status: accepted
---

# 0007 플랫폼, 배포, 업그레이드

## 결정

- 바이너리를 **코어(필수)**와 **시맨틱(선택)** 두 층으로 나눈다. 코어는 순수 Go, `CGO_ENABLED=0`으로 빌드한다.
- 지원 플랫폼은 여섯이며 그중 windows/amd64와 darwin/arm64를 tier 1로 명시한다.
- 배포는 Homebrew 단독이 아니라 플랫폼별 표준 경로를 각각 제공한다.

## 근거

가장 위험한 지점이다. 이전 설계의 "bge-m3 ONNX 내장"과 "Homebrew 배포"는 둘 다 Windows 전용 사용자를 배제한다. 교육 대상 다수가 Windows만 사용하므로 그 전제로는 강의가 성립하지 않는다.

### ONNX 문제

onnxruntime의 Go 바인딩은 cgo와 플랫폼별 네이티브 라이브러리를 요구한다. cgo를 켜는 순간 "단일 바이너리, 설치 한 방"이라는 존재 이유가 무너진다. Windows에서는 DLL 배치, 백신 차단, MSVC 런타임 의존이 따라온다. 첫날 환경 공사를 없애려고 만든 도구가 첫날 환경 공사를 만든다.

### 층 분리

- **코어**: 순수 Go. BM25 키워드 검색까지 포함한다. 이것만으로 강의 1단위부터 4단위까지가 완결된다.
- **시맨틱**: `engram model pull`로 런타임에 내려받는 사이드카다. 실패해도 코어는 정상 동작하고 검색이 키워드 전용으로 degrade된다. 폐쇄망을 고려해 오프라인 번들 경로(`--from ./bge-m3.zip`)도 제공한다.

시맨틱 검색의 부재는 기능 결손이 아니라 성능 저하로 나타나야 한다. 이 원칙을 어기면 모델 다운로드가 사실상 필수가 되어 층 분리가 무의미해진다.

## 플랫폼 매트릭스

CI에서 여섯을 빌드한다. windows/amd64, windows/arm64, darwin/arm64, darwin/amd64, linux/amd64, linux/arm64.

tier 1은 windows/amd64와 darwin/arm64다. 릴리스 차단 기준은 tier 1 실패에만 적용하고, tier 2는 best effort로 둔다.

| 플랫폼 | 설치 | 업그레이드 |
|---|---|---|
| Windows | winget, scoop, 관리자 권한 없는 환경용 zip과 `%LOCALAPPDATA%` 설치 ps1 | `engram self-update` |
| macOS | Homebrew tap | brew upgrade 또는 self-update |
| Linux | curl 설치 스크립트, tar.gz | self-update |

`self-update`는 GitHub Releases의 서명된 체크섬을 검증한다. 사내 프록시 환경을 위해 `ENGRAM_UPDATE_URL`로 내부 미러 미러를 가리킬 수 있어야 한다. ADR 0004가 정한 배포 경로가 여기에 연결된다.

## Windows 고유 함정

배포 전 반드시 처리해야 하는 항목이다.

- **경로 구분자**: 위키링크 해석과 파일 경로 비교를 전부 슬래시로 정규화한다.
- **대소문자 비구분 파일시스템**: `[[Context/Foo]]`와 `[[context/foo]]`가 같은 파일을 가리킨다. 링크 무결성 검사가 플랫폼에 따라 다른 결론을 내면 안 된다.
- **CRLF**: autocrlf가 켜진 PC에서 lint가 전량 실패한다. `.gitattributes` 템플릿 동봉이 필수다.
- **콘솔 인코딩**: 한글 출력이 깨진다. UTF-8 코드페이지 설정 또는 출력 계층에서 처리한다.
- **260자 경로 제한**: 긴 문서 제목을 파일명으로 쓰는 경우 걸린다.
- **SmartScreen**: 서명 없는 exe가 차단된다. 코드사이닝 인증서를 확보하기 전까지는 winget과 scoop 경유를 권장 설치 경로로 안내한다.

이 항목들은 `engram doctor`의 점검 대상에 그대로 들어간다.

## 결과

- 릴리스 파이프라인은 tier 1 두 플랫폼의 빌드와 스모크 테스트를 게이트로 삼는다.
- 모델 파일은 릴리스 아티팩트에 포함하지 않는다. 바이너리 크기와 배포 대역폭이 층 분리의 실질적 이득이다.
- 첫 배포에 코드사이닝이 없다는 사실을 릴리스 노트에 명시한다.

## 관련

- [0004 IP와 배포 경로](0004-ip-and-distribution-path.md)
- [0006 듀얼 모드와 모드 전환 커맨드](0006-dual-mode-eject-seal.md)
