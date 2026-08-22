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
| [0012](0012-distribution-via-personal-homebrew-tap.md) | 배포 경로를 개인 Homebrew tap으로 단일화한다 | 2026-08-14 | amended |
| [0013](0013-eject-redefined-seal-removed.md) | eject를 규칙 소유권 이양으로 재정의하고 seal을 폐기한다 | 2026-08-14 | accepted |
| [0014](0014-llm-boundary-agent-drives-binary.md) | LLM 호출을 바이너리에 두지 않고 에이전트가 바이너리를 부른다 | 2026-08-15 | accepted |
| [0015](0015-adr-status-vocabulary-and-amendment-index.md) | ADR 상태 어휘와 개정 관계를 색인으로 관리한다 | 2026-08-15 | accepted |
| [0016](0016-cli-framework-and-global-flags.md) | CLI 프레임워크를 cobra로 하고 전역 플래그 계약을 고정한다 | 2026-08-15 | accepted |
| [0017](0017-yaml-for-config-and-frontmatter.md) | 설정과 프론트매터를 모두 YAML로 하고 파서를 하나만 둔다 | 2026-08-15 | accepted |
| [0018](0018-taxonomy-field-names.md) | taxonomy 두 facet의 문서 필드명을 topics와 form으로 확정한다 | 2026-08-15 | accepted |
| [0019](0019-index-documents-outside-the-gate.md) | 색인 문서를 승급 게이트와 고아 판정 대상에서 제외한다 | 2026-08-15 | accepted |
| [0020](0020-slug-and-filename-rules.md) | 슬러그와 파일명 규칙을 upstream 실물에서 확정한다 | 2026-08-15 | amended |
| [0021](0021-gate-deferral-when-targets-are-scarce.md) | 링크 대상이 부족하면 승급 게이트를 유예한다 | 2026-08-15 | amended |
| [0022](0022-promote-moves-inbox-derives-sources.md) | promote는 inbox를 옮기고 sources에서는 파생을 만든다 | 2026-08-15 | amended |
| [0023](0023-gate-targets-exclude-inbox.md) | 게이트의 링크 대상 집계에서 inbox 문서를 제외한다 | 2026-08-15 | accepted |
| [0024](0024-public-boundary-and-private-directory.md) | 공개 경계를 gitignore된 private 디렉토리로 긋고 이력을 익명화한다 | 2026-08-15 | amended |
| [0025](0025-index-storage-and-staleness.md) | 인덱스를 JSON으로 저장하고 조회는 인덱스를 갱신하지 않는다 | 2026-08-15 | accepted |
| [0026](0026-windows-console-utf8.md) | Windows 콘솔에 출력할 때 코드페이지를 UTF-8로 바꾼다 | 2026-08-15 | accepted |
| [0027](0027-prose-register-by-audience.md) | 문체를 독자에 따라 세 층으로 나눈다 | 2026-08-15 | accepted |
| [0028](0028-rediscovery-state-and-boundaries.md) | 재발견 커맨드의 상태를 성격에 따라 두 곳에 나눠 둔다 | 2026-08-16 | amended |
| [0029](0029-upstream-vendoring-and-parity-execution.md) | upstream 계약을 치환 사전으로 익명화해 vendoring하고 parity는 로컬에서만 돈다 | 2026-08-16 | amended |
| [0030](0030-upstream-delta-is-not-a-public-artifact.md) | upstream 변경 로그에서 뽑은 delta는 공개하지 않는다 | 2026-08-16 | accepted |
| [0031](0031-location-must-agree-with-stage.md) | 문서가 놓인 디렉토리와 artifact_stage가 일치해야 한다 | 2026-08-16 | amended |
| [0032](0032-update-writes-the-updated-field.md) | 문서를 바꾸는 커맨드가 updated를 그 자리에서 채운다 | 2026-08-16 | accepted |
| [0033](0033-private-backup-and-fail-closed-boundary.md) | private 자료는 upstream에 백업하고 경계 검사는 닫히는 쪽으로 실패한다 | 2026-08-16 | accepted |
| [0034](0034-rule-spec-terminology.md) | upstream 규칙 문서를 계약이 아니라 규칙 명세라 부른다 | 2026-08-16 | accepted |
| [0035](0035-stage-mismatch-severity-by-direction.md) | 위치와 단계의 불일치는 방향에 따라 등급을 나눈다 | 2026-08-16 | accepted |
| [0036](0036-non-document-files-in-stage-dirs.md) | 단계 디렉토리 안의 비문서 마크다운을 순회에서 제외한다 | 2026-08-16 | accepted |
| [0037](0037-sync-corrects-dates-from-git.md) | sync는 git 이력에서 날짜를 정정하고 프론트매터 병합은 맡지 않는다 | 2026-08-16 | amended |
| [0038](0038-migrate-conforms-documents-to-current-rules.md) | migrate는 기존 문서를 지금의 설정과 규칙에 맞춘다 | 2026-08-16 | accepted |
| [0039](0039-eject-emits-rule-specs-and-a-python-linter.md) | eject는 규칙 명세와 표준 라이브러리 Python 린터를 내보낸다 | 2026-08-16 | accepted |
| [0040](0040-gate-follows-the-directory-not-the-declaration.md) | 게이트는 선언이 아니라 디렉토리를 따르고 artifact_stage 누락은 오류다 | 2026-08-16 | accepted |
| [0041](0041-skills-install-embeds-one-static-skill.md) | skills install은 정적 스킬 문서 하나를 심고 위키별 규칙은 rules show로 넘긴다 | 2026-08-17 | amended |
| [0042](0042-release-artifacts-and-workflow.md) | 릴리스는 goreleaser로 만들고 태그 푸시가 유일한 계기다 | 2026-08-17 | accepted |
| [0043](0043-mcp-exposes-one-write-tool-and-omits-promote.md) | MCP는 쓰기 도구를 하나만 노출하고 promote를 내보내지 않는다 | 2026-08-17 | accepted |
| [0044](0044-serve-is-read-only-and-shows-only-vetted-knowledge.md) | serve는 읽기 전용이고 검수된 지식만 보여준다 | 2026-08-17 | amended |
| [0045](0045-explicit-slug-must-still-be-filesystem-safe.md) | 명시한 슬러그도 파일시스템 안전 검사를 받는다 | 2026-08-17 | amended |
| [0046](0046-pack-exports-files-and-anonymizes-by-user-dictionary.md) | pack은 파일을 그대로 내보내고 익명화는 사용자 사전으로 한다 | 2026-08-17 | amended |
| [0047](0047-export-not-pack.md) | 반출 커맨드의 이름을 export로 정정한다 | 2026-08-17 | accepted |
| [0048](0048-preset-names-follow-attribute-sets.md) | 프리셋 이름을 속성 집합에 맞추고 축을 속성으로 부른다 | 2026-08-17 | accepted |
| [0049](0049-cli-output-language.md) | 출력 언어는 사용자 환경이 정하고 카탈로그는 바이너리에 묶는다 | 2026-08-17 | accepted |
| [0050](0050-slugs-must-be-wikilink-safe.md) | 슬러그는 파일시스템뿐 아니라 위키링크 문법에도 안전해야 한다 | 2026-08-17 | accepted |
| [0051](0051-sources-holds-originals-and-refined-summaries.md) | sources는 원본과 정제본을 함께 담고 type이 그 둘을 가른다 | 2026-08-17 | accepted |
| [0052](0052-agent-prepares-the-promotion-and-the-human-decides-it.md) | 에이전트가 승급을 준비하고 사람은 승급을 결정한다 | 2026-08-17 | amended |
| [0053](0053-wiki-path-accepts-both-positional-and-flag.md) | 위키 경로는 네 커맨드에서도 --wiki로 받는다 | 2026-08-17 | accepted |
| [0054](0054-gate-counts-only-links-that-resolve.md) | 승급 게이트는 실제로 이어지는 링크만 센다 | 2026-08-17 | accepted |
| [0055](0055-agents-change-the-wiki-only-through-commands.md) | 에이전트는 커맨드로만 위키를 바꾼다 | 2026-08-17 | accepted |
| [0056](0056-promote-has-a-dry-run.md) | promote에 --dry-run을 둔다 | 2026-08-17 | accepted |
| [0057](0057-approval-attaches-to-content-not-to-the-command.md) | 승인은 커맨드가 아니라 문서 내용에 붙는다 | 2026-08-17 | accepted |
| [0058](0058-promote-to-sources-moves-evidence.md) | promote --to sources가 inbox의 증거를 옮긴다 | 2026-08-17 | accepted |
| [0059](0059-recall-candidate-pool-is-independent-of-limit.md) | recall의 문서 후보 수는 --limit과 무관하게 고정한다 | 2026-08-18 | accepted |
| [0060](0060-search-line-shows-stage-not-path.md) | search 목록은 경로 대신 단계를 내고 제목은 --json에만 담는다 | 2026-08-18 | accepted |
| [0061](0061-field-weights-and-what-the-index-is-not.md) | 색인 필드에 가중치를 주고 슬러그는 색인하지 않는다 | 2026-08-18 | accepted |
| [0062](0062-agents-read-the-wiki-through-recall.md) | 에이전트는 위키 내용을 recall로 꺼낸다 | 2026-08-18 | accepted |
| [0063](0063-exposure-reads-indexable-status-and-scopes-internal.md) | 노출 판정이 indexable과 status를 읽고 internal은 반출에서 뺀다 | 2026-08-19 | accepted |
| [0064](0064-update-refuses-to-change-sources.md) | update는 sources 문서를 거절한다 | 2026-08-19 | accepted |
| [0065](0065-markdown-links-count-as-relations.md) | 마크다운 링크도 문서 사이의 관계로 센다 | 2026-08-19 | accepted |
| [0066](0066-rediscovery-reads-inbound-links-and-ignores-bulk-commits.md) | 재발견은 인바운드 링크를 보고 대량 커밋을 신호에서 뺀다 | 2026-08-19 | accepted |
| [0067](0067-stale-days-default-is-thirty.md) | 노후 기준일 기본값을 30일로 한다 | 2026-08-19 | accepted |
| [0068](0068-model-command-manages-embeddings-only.md) | model 커맨드는 임베딩만 관리한다 | 2026-08-19 | amended |
| [0069](0069-secrets-and-sensitivity-block-promotion-and-export.md) | 시크릿과 민감도가 승급과 반출을 막는다 | 2026-08-19 | accepted |
| [0070](0070-lint-skips-inbox-by-default.md) | lint는 기본으로 inbox를 검사하지 않는다 | 2026-08-19 | amended |
| [0071](0071-lint-checks-indexable-stage-and-deprecated-fields.md) | lint가 색인 자격과 폐기 필드를 검사한다 | 2026-08-19 | accepted |
| [0072](0072-date-fields-are-written-only-where-they-are-read.md) | 날짜 필드는 그 값을 읽는 단계에만 쓴다 | 2026-08-19 | accepted |
| [0073](0073-provenance-must-not-be-empty.md) | 증거 필드는 비어 있으면 안 된다 | 2026-08-19 | accepted |
| [0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md) | 임베딩은 순수 Go로 돌리고 모델은 bge-m3 fp32로 고정한다 | 2026-08-19 | accepted |
| [0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md) | 임베딩은 문서에 붙고 축마다 자기 하한을 갖는다 | 2026-08-19 | amended |
| [0076](0076-serve-shows-rediscovery-without-recording-it.md) | serve는 재발견을 기록 없이 보여준다 | 2026-08-21 | accepted |
| [0077](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md) | 에이전트 계약은 위키 안에 남고 스킬 문서는 한 벌이다 | 2026-08-21 | accepted |
| [0078](0078-semantic-search-is-an-explicit-flag-that-only-consumes-the-cache.md) | 의미 검색은 명시 플래그이고 캐시를 소비만 한다 | 2026-08-21 | accepted |
| [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md) | 음성은 별도 저장소의 별도 바이너리이고 위키는 용어 사전을 소유한다 | 2026-08-21 | amended |
| [0080](0080-voice-is-a-nested-module-in-this-repository.md) | 음성은 이 저장소의 중첩 모듈이다 | 2026-08-21 | accepted |
| [0081](0081-default-whisper-model-is-large-v3.md) | 기본 whisper 모델은 large-v3이고 배포는 단일 바이너리가 아니다 | 2026-08-21 | amended |
| [0082](0082-speaker-count-is-asked-not-guessed.md) | 화자 수는 사람에게 묻고 추정치에는 신뢰할 수 없다고 적는다 | 2026-08-21 | amended |
| [0083](0083-the-glossary-corrects-after-the-fact-and-grows-against-one-model.md) | 용어 사전은 후처리만 하고 쓰는 모델에 붙어 자란다 | 2026-08-21 | accepted |
| [0084](0084-voice-ships-on-fewer-platforms-than-engram.md) | engram-voice는 본체보다 적은 플랫폼에 배포하고 릴리스 경로가 다르다 | 2026-08-21 | amended |
| [0085](0085-synthetic-speech-is-not-teaching-material.md) | 합성 음성은 음성 세션의 교재가 될 수 없고 수강생은 자기 목소리를 녹음한다 | 2026-08-21 | amended |
| [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md) | 러너 라벨을 실제로 확인하고 Windows는 CI에서 먼저 잰다 | 2026-08-21 | amended |
| [0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md) | Windows는 재서 통과했고 포장까지 CI에서 돌린다 | 2026-08-21 | accepted |
| [0088](0088-homebrew-ships-a-cask-and-the-tap-is-reached-with-a-deploy-key.md) | Homebrew 배포는 Formula가 아니라 Cask이고 tap 접근은 배포키로 한다 | 2026-08-21 | accepted |
| [0089](0089-voice-is-driven-by-an-agent-and-its-pipeline-is-documented.md) | 음성은 에이전트가 도구로 부르고 그 동작 구조를 문서에 적는다 | 2026-08-22 | accepted |
| [0090](0090-the-skill-document-is-the-source-and-mcp-delivers-it-as-instructions.md) | 스킬 문서가 진실원이고 MCP는 그것을 instructions로 보낸다 | 2026-08-22 | accepted |
| [0091](0091-metrics-count-only-what-the-user-can-act-on.md) | 지표는 사용자가 손댈 수 있는 것만 세고 센 것은 다음 행동으로 잇는다 | 2026-08-22 | accepted |
| [0092](0092-diarization-thresholds-are-measured-against-real-recordings.md) | 화자 분할 임계값을 실제 사람 녹음으로 재고 합성 음성 경로를 닫는다 | 2026-08-22 | accepted |
| [0093](0093-ci-runs-a-real-transcription-on-every-runner.md) | CI가 러너마다 실제 전사를 한 번 돌린다 | 2026-08-22 | accepted |

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
| 0020 | 사용자는 언제든 슬러그를 명시해 파생을 덮어쓸 수 있다 | [0045](0045-explicit-slug-must-still-be-filesystem-safe.md) 파생은 덮어쓰되 파일시스템 안전 검사는 못 덮는다 |
| 0031 | 등급은 `error`다 | [0035](0035-stage-mismatch-severity-by-direction.md) 불일치 방향에 따라 `error`와 `warn`으로 나눈다 |
| 0046 | 커맨드 이름을 `pack`으로 한다 | [0047](0047-export-not-pack.md) `export`로 정정. 동작은 그대로다 |
| 0009 | 프리셋 이름은 `personal`, `education`, `team`이다 | [0048](0048-preset-names-follow-attribute-sets.md) `minimal`, `personal`, `team`으로 정정. 속성 구성은 그대로다 |
| 0045 | 슬러그 안전 검사는 파일시스템 예약 문자를 본다 | [0050](0050-slugs-must-be-wikilink-safe.md) 위키링크 문법을 깨는 문자를 더한다 |
| 0009 | 문서 종류 기본값은 열이다 | [0051](0051-sources-holds-originals-and-refined-summaries.md) `source-raw`를 더해 열하나가 된다 |
| 0041 | 스킬 문서가 반드시 담는 것은 다섯이다 | [0052](0052-agent-prepares-the-promotion-and-the-human-decides-it.md) 승급 준비 지시 셋을 더해 여덟이 된다 |
| 0041 | 쓰기 경계는 단계로 말한다 | [0055](0055-agents-change-the-wiki-only-through-commands.md) 도구로도 말한다. 파일 직접 편집을 금지한다 |
| 0052 | promote는 제안만 하고 사람이 실행한다 | [0057](0057-approval-attaches-to-content-not-to-the-command.md) 승인을 받은 뒤 에이전트가 실행한다. 스킬 경로에 한한다 |
| 0022 | promote는 inbox를 옮기고 sources에서 파생한다 | [0058](0058-promote-to-sources-moves-evidence.md) --to sources로 inbox에서 sources로 옮기는 길을 더한다 |
| 0009 | 게이트는 문서의 고유 위키링크 수를 센다 | [0054](0054-gate-counts-only-links-that-resolve.md) 대상이 실재하는 링크만 센다 |
| 0052 | 스킬 문서가 반드시 담는 것은 여덟이다 | [0062](0062-agents-read-the-wiki-through-recall.md) 위키 내용을 recall로 꺼내라는 지시를 더해 아홉이 된다 |
| 0009 | 게이트는 문서의 고유 위키링크 수를 센다 | [0065](0065-markdown-links-count-as-relations.md) 마크다운 링크도 관계로 세므로 대상이 위키링크에 한정되지 않는다 |
| 0044 | 노출 판정은 위치와 민감도를 본다 | [0063](0063-exposure-reads-indexable-status-and-scopes-internal.md) indexable 과 status 를 더하고 internal 을 반출에서 뺀다 |
| 0028 | 재발견 선정은 경과일과 제시 이력으로 정렬한다 | [0066](0066-rediscovery-reads-inbound-links-and-ignores-bulk-commits.md) 인바운드 가중 점수로 정렬하고 제시 이력은 쿨다운 필터로 옮긴다 |
| 0009 | 노후 기준일 기본값은 90일이다 | [0067](0067-stale-days-default-is-thirty.md) 30일로 바꾼다. 실측에서 90 이면 후보가 0 건이다 |
| 0007 | 시맨틱은 model pull 로 받는 선택적 사이드카다 | [0068](0068-model-command-manages-embeddings-only.md) 관리 대상을 임베딩 하나로 못 박는다 |
| 0063 | 노출 판정이 반출에서 internal 을 뺀다 | [0069](0069-secrets-and-sensitivity-block-promotion-and-export.md) 승급과 반출에서 시크릿과 높은 민감도를 막는다 |
| 0009 | 필수 필드 검사는 단계별로 모든 문서에 적용된다 | [0070](0070-lint-skips-inbox-by-default.md) inbox 는 기본 범위 밖이고 --include-inbox 로 연다 |
| 0037 | sync 는 git 이력에서 날짜를 정정한다 | [0072](0072-date-fields-are-written-only-where-they-are-read.md) 이력이 없으면 파일명 접두사를 보조로 쓰고 대상 단계를 좁힌다 |
| 0022 | promote 는 sources 에서 파생을 만들며 derived_from 을 채운다 | [0073](0073-provenance-must-not-be-empty.md) source_refs 도 함께 채운다 |
| 0068 | 넷째 논거가 int8 양자화판 585MB 를 전제한다 | [0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md) 그 판은 순수 Go 에서 돌지 않는다. 실제 대상은 fp32 2.27GB 하나다 |
| 0068 | 순수 Go 의 실제 동작과 속도는 미측정이다 | [0074](0074-embedding-runs-in-pure-go-and-the-model-is-bge-m3-fp32.md) 측정해 닫는다. fp32 에 한해 돌고 계산 시점을 bridge 로 옮겨 쓴다 |
| 0028 | bridge 는 코사인 하한 하나로 후보를 거른다 | [0075](0075-embedding-attaches-to-the-document-and-each-axis-has-its-own-floor.md) 단어 축과 임베딩 축이 각자 하한을 갖고 합집합을 본다 |
| 0044 | 재발견 커맨드를 노출하지 않는다 | [0076](0076-serve-shows-rediscovery-without-recording-it.md) 기록하지 않는 화면으로 노출한다. 제시 이력도 벡터 캐시도 쓰지 않는다 |
| 0041 | 스킬 문서는 사용자 홈에만 심긴다 | [0077](0077-the-agent-contract-stays-in-the-wiki-and-the-skill-is-one-copy.md) eject 가 같은 본문을 위키 안 meta/agent-contract.md 로도 남긴다 |
| 0075 | 임베딩을 쓰는 곳은 bridge 뿐이다 | [0078](0078-semantic-search-is-an-explicit-flag-that-only-consumes-the-cache.md) search --semantic 과 serve 의 재발견 화면이 그 캐시를 읽는다. 계산 자리는 그대로 bridge 다 |
| 0068 | STT 는 범위 밖이라고만 적고 어디로 가는지는 열어 둔다 | [0079](0079-voice-is-a-separate-binary-in-a-separate-repository.md) engram-voice 로 간다. 코어의 CGO 없음은 그대로 지킨다 |
| 0079 | 사전을 whisper 의 initial_prompt 로 넣어 인식을 유도한다 | [0083](0083-the-glossary-corrects-after-the-fact-and-grows-against-one-model.md) sherpa-onnx 의 whisper 에 프롬프트 자리가 없다. 후처리 치환만 한다 |
| 0082 | 임계값을 더 만지지 않는다. 정답이 없기 때문이다 | [0092](0092-diarization-thresholds-are-measured-against-real-recordings.md) 정답이 붙은 실제 사람 녹음이 생겼다. 군집 임계값을 0.5 에서 0.70 으로 바꾼다 |
| 0085 | 목소리 셋을 받으면 make.sh 가 화자 정답 자료를 만든다 | [0092](0092-diarization-thresholds-are-measured-against-real-recordings.md) 목소리를 바꿔도 화자가 안 갈린다. 그 길을 닫고 스크립트를 지운다 |
| 0087 | 전사 경로는 CI 가 못 본다. 모델을 받아야 하기 때문이다 | [0093](0093-ci-runs-a-real-transcription-on-every-runner.md) small 모델 392MB 를 캐시해 러너 셋에서 한 번씩 전사한다 |
| 0070 | 링크 그래프 판정과 게이트는 지금대로 inbox 를 담는다 | [0091](0091-metrics-count-only-what-the-user-can-act-on.md) graph.orphan 은 inbox 를 판정하지 않는다. link.broken 과 게이트는 그대로다 |
| 0070 | 요약은 inbox 문서를 건너뛰었다고 알린다 | [0091](0091-metrics-count-only-what-the-user-can-act-on.md) 건너뛴 것은 스키마 판정뿐이라 문구를 그렇게 고친다 |
| 0012 | Homebrew 는 Formula 로 배포한다 | [0088](0088-homebrew-ships-a-cask-and-the-tap-is-reached-with-a-deploy-key.md) goreleaser 가 brews 를 접었고 Cask 의 binary 가 Linux 도 덮는다 |
| 0086 | Windows 는 재는 중이라 릴리스에 넣지 않는다 | [0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md) 쟀고 통과했다. 릴리스 대상이 다섯이 된다 |
| 0084 | 릴리스 대상은 넷이다 | [0087](0087-windows-is-measured-green-and-packaging-runs-in-ci.md) windows/amd64 를 더해 다섯 |
| 0084 | darwin/amd64 를 macos-13 러너에서 빌드한다 | [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md) 그 라벨은 2025년 12월에 없어졌다. macos-15-intel 로 바꾼다 |
| 0084 | Windows 는 재지 않았으므로 넣지 않는다 | [0086](0086-runner-labels-are-checked-and-windows-is-measured-in-ci.md) 러너에 gcc 가 있다. CI 에 넣어 잰 뒤 릴리스에 올린다 |
| 0081 | 배포 형태만 정하고 플랫폼과 워크플로는 남겼다 | [0084](0084-voice-ships-on-fewer-platforms-than-engram.md) 대상 넷을 정하고 러너를 나눈다. Windows 는 재지 않아 넣지 않는다 |
| 0081 | 용어 사전이 모델 차이를 메운다는 가설은 미검증이다 | [0083](0083-the-glossary-corrects-after-the-fact-and-grows-against-one-model.md) 메우지 못한다. 사전은 자기가 쌓인 모델의 오인식만 잡는다 |
| 0005 | upstream 과 같은 판정을 내는 것이 동등성이다 | [0082](0082-speaker-count-is-asked-not-guessed.md) 화자 분할은 규칙이 아니라 추정이라 동등성을 확인만 하고 따르지 않는다 |
| 0079 | 기본 whisper 크기는 미결이고 용어 사전이 모델 차이를 메운다고 본다 | [0081](0081-default-whisper-model-is-large-v3.md) large-v3 로 정한다. 사전 가설은 신호 부족으로 검증도 반증도 못 했다 |
| 0079 | 음성은 별도 저장소에 둔다 | [0080](0080-voice-is-a-nested-module-in-this-repository.md) 이 저장소의 중첩 모듈로 둔다. 별도 저장소에서는 internal/embed 재사용이 성립하지 않는다 |

## 근거만 갱신된 경우

결론이 유지되어 `status`를 바꾸지 않았으나 근거가 달라진 사례다.

| ADR | 갱신 내용 | 기록 위치 |
|---|---|---|
| 0002 | 분리의 이유가 "본인이 안 쓰기 때문"에서 "같은 체계를 두 강제 방식으로 제공하기 때문"으로 바뀌었다 | [0006](0006-dual-mode-eject-seal.md) 본문 |
| 0001 | Homebrew 배포가 0007에서 후퇴했다가 0012에서 복권되어 원문과 일치한다 | [0012](0012-distribution-via-personal-homebrew-tap.md) 본문 |
| 0005 | "계약 파일"이라는 명칭이 "규칙 명세"로 바뀌었다. 결론은 그대로다 | [0034](0034-rule-spec-terminology.md) 본문 |
| 0029 | 상동. 제목의 "계약"은 당시 명칭으로 읽는다 | [0034](0034-rule-spec-terminology.md) 본문 |
| 0010 | 자체 구현의 둘째 근거인 parity 보증이 성립하지 않는다. upstream에 `search`가 없어 비교할 대상이 없다. 첫째 근거인 한국어 토크나이저와 0007의 CGO 제약으로 결정은 유지된다 | [0061](0061-field-weights-and-what-the-index-is-not.md) 본문 |
| 0074 | upstream MPS 속도를 주석의 "12초"가 아니라 실제로 돌려 쟀다. 전수 인코딩 13.27초이며 순수 Go와의 배율이 84배가 아니라 76배다. 결정은 유지된다 | [upstream-gap.md](../upstream-gap.md) R1 |
| 0077 | 본문이 인용한 `SKILL.md` 11.7KB는 결정 시점 값이다. 같은 작업에서 의미 축 검색 지시를 더해 12.9KB가 되었고 `eject` 산출물은 13KB다. 규범 총량 비교(22.4KB 대 35KB)의 결론은 유지된다 | [0078](0078-semantic-search-is-an-explicit-flag-that-only-consumes-the-cache.md) 본문 |
| 0074 | 판별 간격 0.0737은 평균 풀링으로 잰 값이다. bge-m3의 실제 방식인 CLS 풀링으로 다시 재면 0.1217이며 e5-small과의 차이가 3배가 아니라 4.8배다. 모델 선택은 유지된다 | [upstream-gap.md](../upstream-gap.md) R1 |
