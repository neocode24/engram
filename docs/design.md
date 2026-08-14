# 설계 개요 (초안)

> 2026-08-08 초기 논의 기반. 구체화는 다음 세션부터.

## 정체성

"**판단은 사람이, 불변식은 코드가**" — llm-wiki 운영 철학을 단일 Go 바이너리가 강제한다. 다른 PKM/위키 도구와의 차별점은 **inbox → sources → context 승급 파이프라인을 코드가 강제**한다는 점이다.

## 형태 (단일 Go 바이너리)

구현 형태:

- 단일 Go 바이너리, XDG Base Directory 준수
- 위키 내 스키마 파일(`*.toml`)
- 쓰기 명령 = 스키마 검증 게이트
- BM25 + bge-m3 ONNX 하이브리드 검색 (완전 로컬)
- resurface / bridge / recall / digest 루프
- `serve` 웹 UI
- `skills install` 에이전트 연동
- Homebrew tap 배포

## 핵심 diff: 승급 파이프라인

llm-wiki의 3단계 계층을 명령으로 노출한다:

- `capture` (inbox): 처리 대기 입력
- `source`: 원본·출처 보존 (append-only)
- `promote`: 검수된 지식을 context로 승급 (스키마 검증 게이트)

## 커맨드 체계 초안

| 분류 | 명령 |
|---|---|
| 초기화 | `init`, `lint`, `status` |
| 승급 | `capture`, `source`, `promote`, `archive` |
| 쓰기 (context) | `new`, `update`, `mv`, `rm` (스키마 강제) |
| 검색 | `search` (hybrid/keyword/semantic), `recall`, `backlinks` |
| 재발견 | `resurface`, `bridge`, `digest` |
| 운영 | `serve`, `sync`, `reindex`, `skills install` |

## 스키마 매핑 (llm-wiki 9축 → 교육 오픈소스)

llm-wiki: `type, status, scope, sensitivity, source_channel, trigger_mode, workflow, artifact_stage, indexable`

- **핵심 보존**: `type, status, artifact_stage, source_channel, tags, indexable`
- **보편화 검토**: `scope, sensitivity, trigger_mode, workflow` — 회사 특수성이 강함. 교육용으로는 핵심 subset + 확장 가능 축으로 재설계. **미결정(다음 설계 세션).**

## 마일스톤 (접근법 B, 단계적 출하)

| 버전 | 범위 |
|---|---|
| 0.1 | init, capture/source/promote, 스키마 강제, lint, status |
| 0.2 | keyword/hybrid 검색, reindex |
| 0.3 | resurface/bridge/recall |
| 1.0 | serve 웹 UI, skills install, Homebrew 배포 |

## 미결정

- ~~하위 repo 구성~~ 완료. [ADR 0011](decisions/0011-repo-layout-and-module-name.md)
- ~~GitHub remote 생성 시점~~ 완료
- 스키마 보편화 범위
- 교육용 데모 위키 내용
