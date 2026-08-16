# ADR 색인

`engram`의 설계 결정 기록이다. 번호순으로 쌓이며 본문은 소급 수정하지 않는다. 결정이 바뀌면 새 ADR을 쓰고 이 색인에 개정 관계를 추가한 뒤 대상 ADR의 `status`만 바꾼다. 규칙은 [0015](0015-adr-status-vocabulary-and-amendment-index.md)에 있다.

## 상태 어휘

| 값 | 의미 |
|---|---|
| `accepted` | 유효하다 |
| `amended` | 결론은 유효하나 일부 절이 대체되었다. 아래 개정 그래프를 함께 읽는다 |
| `superseded` | 전체가 대체되었다. 이력으로만 읽는다 |
| `proposed` | 확정 전이다 |

## 목록

| # | 제목 | 날짜 | 상태 |
|---|---|---|---|
| [0001](0001-purpose-and-scope.md) | 프로젝트 목적과 범위 | 2026-08-08 | accepted |
| [0002](0002-operating-vs-product-separation.md) | 운영과 산출물을 분리한다 | 2026-08-08 | accepted |
| [0003](0003-approach-B-fullspec-with-promotion.md) | 접근법 B, 풀스펙에 승급 파이프라인 | 2026-08-08 | amended |
| [0004](0004-ip-and-distribution-path.md) | IP와 배포 경로 | 2026-08-08 | amended |
| [0005](0005-upstream-contract-and-harness.md) | upstream 계약을 harness로 동기화한다 | 2026-08-13 | amended |
| [0006](0006-dual-mode-eject-seal.md) | easy/hard 듀얼 모드와 모드 전환 커맨드 | 2026-08-13 | amended |
| [0007](0007-platform-and-distribution.md) | 플랫폼과 배포, 코어와 시맨틱 층 분리 | 2026-08-13 | amended |
| [0008](0008-project-name-engram.md) | 프로젝트 이름을 engram으로 확정한다 | 2026-08-13 | accepted |
| [0009](0009-schema-presets-and-thresholds.md) | 스키마 프리셋과 게이트 임계값 | 2026-08-14 | amended |
| [0010](0010-storage-index-and-korean-search.md) | 저장, 인덱스, 한국어 검색 | 2026-08-14 | accepted |
| [0011](0011-repo-layout-and-module-name.md) | 저장소 구성과 Go 모듈명 | 2026-08-14 | amended |
| [0012](0012-distribution-via-personal-homebrew-tap.md) | 배포 경로를 개인 Homebrew tap으로 단일화한다 | 2026-08-14 | accepted |
| [0013](0013-eject-redefined-seal-removed.md) | eject를 규칙 소유권 이양으로 재정의하고 seal을 폐기한다 | 2026-08-14 | accepted |
| [0014](0014-llm-boundary-agent-drives-binary.md) | LLM 호출을 바이너리에 두지 않고 에이전트가 바이너리를 부른다 | 2026-08-15 | accepted |
| [0015](0015-adr-status-vocabulary-and-amendment-index.md) | ADR 상태 어휘와 개정 관계를 색인으로 관리한다 | 2026-08-15 | accepted |
| [0016](0016-cli-framework-and-global-flags.md) | CLI 프레임워크를 cobra로 하고 전역 플래그 계약을 고정한다 | 2026-08-15 | accepted |
| [0017](0017-yaml-for-config-and-frontmatter.md) | 설정과 프론트매터를 모두 YAML로 하고 파서를 하나만 둔다 | 2026-08-15 | accepted |
| [0018](0018-taxonomy-field-names.md) | taxonomy 두 facet의 문서 필드명을 topics와 form으로 확정한다 | 2026-08-15 | accepted |
| [0019](0019-index-documents-outside-the-gate.md) | 색인 문서를 승급 게이트와 고아 판정 대상에서 제외한다 | 2026-08-15 | accepted |
| [0020](0020-slug-and-filename-rules.md) | 슬러그와 파일명 규칙을 upstream 실물에서 확정한다 | 2026-08-15 | accepted |
| [0021](0021-gate-deferral-when-targets-are-scarce.md) | 링크 대상이 부족하면 승급 게이트를 유예한다 | 2026-08-15 | amended |
| [0022](0022-promote-moves-inbox-derives-sources.md) | promote는 inbox를 옮기고 sources에서는 파생을 만든다 | 2026-08-15 | accepted |
| [0023](0023-gate-targets-exclude-inbox.md) | 게이트의 링크 대상 집계에서 inbox 문서를 제외한다 | 2026-08-15 | accepted |
| [0024](0024-public-boundary-and-private-directory.md) | 공개 경계를 gitignore된 private 디렉토리로 긋고 이력을 익명화한다 | 2026-08-15 | amended |
| [0025](0025-index-storage-and-staleness.md) | 인덱스를 JSON으로 저장하고 조회는 인덱스를 갱신하지 않는다 | 2026-08-15 | accepted |
| [0026](0026-windows-console-utf8.md) | Windows 콘솔에 출력할 때 코드페이지를 UTF-8로 바꾼다 | 2026-08-15 | accepted |
| [0027](0027-prose-register-by-audience.md) | 문체를 독자에 따라 세 층으로 나눈다 | 2026-08-15 | accepted |
| [0028](0028-rediscovery-state-and-boundaries.md) | 재발견 커맨드의 상태를 성격에 따라 두 곳에 나눠 둔다 | 2026-08-16 | accepted |
| [0029](0029-upstream-vendoring-and-parity-execution.md) | upstream 계약을 치환 사전으로 익명화해 vendoring하고 parity는 로컬에서만 돈다 | 2026-08-16 | amended |
| [0030](0030-upstream-delta-is-not-a-public-artifact.md) | upstream 변경 로그에서 뽑은 delta는 공개하지 않는다 | 2026-08-16 | accepted |
| [0031](0031-location-must-agree-with-stage.md) | 문서가 놓인 디렉토리와 artifact_stage가 일치해야 한다 | 2026-08-16 | amended |
| [0032](0032-update-writes-the-updated-field.md) | 문서를 바꾸는 커맨드가 updated를 그 자리에서 채운다 | 2026-08-16 | accepted |
| [0033](0033-private-backup-and-fail-closed-boundary.md) | private 자료는 upstream에 백업하고 경계 검사는 닫히는 쪽으로 실패한다 | 2026-08-16 | accepted |
| [0034](0034-rule-spec-terminology.md) | upstream 규칙 문서를 계약이 아니라 규칙 명세라 부른다 | 2026-08-16 | accepted |
| [0035](0035-stage-mismatch-severity-by-direction.md) | 위치와 단계의 불일치는 방향에 따라 등급을 나눈다 | 2026-08-16 | accepted |
| [0036](0036-non-document-files-in-stage-dirs.md) | 단계 디렉토리 안의 비문서 마크다운을 순회에서 제외한다 | 2026-08-16 | accepted |
| [0037](0037-sync-corrects-dates-from-git.md) | sync는 git 이력에서 날짜를 정정하고 프론트매터 병합은 맡지 않는다 | 2026-08-16 | accepted |
| [0038](0038-migrate-conforms-documents-to-current-rules.md) | migrate는 기존 문서를 지금의 설정과 규칙에 맞춘다 | 2026-08-16 | accepted |
| [0039](0039-eject-emits-rule-specs-and-a-python-linter.md) | eject는 규칙 명세와 표준 라이브러리 Python 린터를 내보낸다 | 2026-08-16 | accepted |

## 공개 범위 밖

[0004](0004-ip-and-distribution-path.md)는 본문이 공개 대상이 아니라 스텁만 남아 있다. 번호 연속성을 위해 자리는 유지한다. 근거는 [0024](0024-public-boundary-and-private-directory.md)에 있다.

## 개정 그래프

| 개정된 ADR | 대체된 절 | 대체한 ADR |
|---|---|---|
| 0003 | v1 범위에 웹 UI를 포함한다 | [0013](0013-eject-redefined-seal-removed.md) 마일스톤 분할 |
| 0003 | 에이전트 skill 내장 | [0014](0014-llm-boundary-agent-drives-binary.md) 스킬 문서 배포로 확정 |
| 0004 | 내부 미러를 배포 경로로 쓴다 | [0012](0012-distribution-via-personal-homebrew-tap.md) 격리 용도로 재정의 |
| 0006 | `seal` 커맨드 | [0013](0013-eject-redefined-seal-removed.md) 폐기 |
| 0006 | `eject`의 정의와 왕복 전제 | [0013](0013-eject-redefined-seal-removed.md) 규칙 소유권 이양, 단방향 |
| 0006 | `attach`를 별도 커맨드 후순위로 둔다 | [0013](0013-eject-redefined-seal-removed.md) 기본 동작으로 전환 |
| 0007 | Windows는 winget과 scoop을 권장 설치 경로로 한다 | [0012](0012-distribution-via-personal-homebrew-tap.md) Homebrew tap 단일화, winget 보류 |
| 0007 | `ENGRAM_UPDATE_URL`로 Enterprise 미러를 가리킨다 | [0012](0012-distribution-via-personal-homebrew-tap.md) 공개 릴리스를 사내에서도 그대로 받는다 |
| 0011 | 공개 전환은 `private/` 경로를 이력에서 제거하는 방식으로 처리한다 | [0024](0024-public-boundary-and-private-directory.md) gitignore로 애초에 커밋하지 않는다 |
| 0021 | 대상은 `page_dirs` 아래 문서와 `root_files`다 | [0023](0023-gate-targets-exclude-inbox.md) inbox 단계와 단계 불명 문서를 제외 |
| 0005 | 계약 파일은 `meta/` 아래 여섯이다 | [0029](0029-upstream-vendoring-and-parity-execution.md) upstream AGENTS.md의 선언을 진실원으로 삼는다 |
| 0005 | 스캐너가 걸리면 실패시킨다 | [0029](0029-upstream-vendoring-and-parity-execution.md) 치환 사전을 거친 뒤 실패시킨다 |
| 0005 | `docs/parity.md`로 자동 갱신한다 | [0029](0029-upstream-vendoring-and-parity-execution.md) upstream이 로컬에 있을 때만 돌고 사람이 커밋한다 |
| 0029 | delta를 `harness/deltas/`에 남긴다 | [0030](0030-upstream-delta-is-not-a-public-artifact.md) `private/deltas/`로 옮겨 공개하지 않는다 |
| 0009 | `updated`는 git 이력에서만 채운다 | [0032](0032-update-writes-the-updated-field.md) 문서를 바꾸는 커맨드가 그 자리에서 채우고 `sync`가 나중에 정정한다 |
| 0024 | `private/`의 실체는 upstream에 두고 여기에는 포인터와 발췌만 둔다 | [0033](0033-private-backup-and-fail-closed-boundary.md) 실체를 양쪽에 두고 `meta/engram/`으로 백업한다 |
| 0024 | 패턴 목록이 없으면 경계 검사를 건너뛴다 | [0033](0033-private-backup-and-fail-closed-boundary.md) 커밋 훅에서는 `--require`로 막는다 |
| 0031 | 등급은 `error`다 | [0035](0035-stage-mismatch-severity-by-direction.md) 불일치 방향에 따라 `error`와 `warn`으로 나눈다 |

## 근거만 갱신된 경우

결론이 유지되어 `status`를 바꾸지 않았으나 근거가 달라진 사례다.

| ADR | 갱신 내용 | 기록 위치 |
|---|---|---|
| 0002 | 분리의 이유가 "본인이 안 쓰기 때문"에서 "같은 체계를 두 강제 방식으로 제공하기 때문"으로 바뀌었다 | [0006](0006-dual-mode-eject-seal.md) 본문 |
| 0001 | Homebrew 배포가 0007에서 후퇴했다가 0012에서 복권되어 원문과 일치한다 | [0012](0012-distribution-via-personal-homebrew-tap.md) 본문 |
| 0005 | "계약 파일"이라는 명칭이 "규칙 명세"로 바뀌었다. 결론은 그대로다 | [0034](0034-rule-spec-terminology.md) 본문 |
| 0029 | 상동. 제목의 "계약"은 당시 명칭으로 읽는다 | [0034](0034-rule-spec-terminology.md) 본문 |
