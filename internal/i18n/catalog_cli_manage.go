// Package i18n의 cli_manage 카탈로그. 관리 커맨드(root, init, reindex,
// migrate, sync, rules, eject, skills, version)의 사용자 대면 문자열을
// 담는다. ADR 0049.
package i18n

func init() {
	Register(LangKO, map[string]string{
		// root
		"cli.root.short": "지식관리 위키의 승급 파이프라인을 다루는 CLI",
		"cli.root.long": `engram은 지식관리 위키의 문서 상태와 승급 파이프라인을 다루는 CLI입니다.

모든 조회 커맨드는 --json으로 JSON 출력을 지원하고, --now로 기준 시각을
고정해 결정론적인 결과를 얻을 수 있습니다.`,
		"cli.root.flag_json": "결과를 JSON으로 출력합니다",
		"cli.root.flag_now":  "기준 시각(RFC3339). 빈 값이면 현재 시각",
		"cli.root.flag_lang": "출력 언어(ko, en). 빈 값이면 %s 환경변수, 그다음 ko",

		// version
		"cli.version.short":     "버전과 빌드 정보를 출력합니다",
		"cli.version.commit_at": "commit 시각: %s",

		// init
		"cli.init.short": "새 위키를 만듭니다",
		"cli.init.long": `지정한 경로에 새 위키를 만듭니다. 경로를 생략하면 현재 디렉토리입니다.

디렉토리 구성, engram.yaml 설정, 첫 문서 index.md, .gitignore를 만듭니다.
이미 engram.yaml이 있으면 기존 위키를 보존하기 위해 거절합니다.`,
		"cli.init.flag_preset":          "스키마 프리셋. minimal, personal, team 중 하나",
		"cli.init.preset_invalid":       "--preset 값이 허용값이 아님: %q (허용값: minimal, personal, team)",
		"cli.init.already_wiki":         "대상이 이미 engram 위키입니다: %s\n기존 위키를 덮어쓰지 않습니다. 다른 경로를 지정하거나 기존 %s을 손으로 고치세요",
		"cli.init.path_check_fail":      "대상 경로를 확인할 수 없음",
		"cli.init.root_mkdir_fail":      "위키 루트를 만들 수 없음",
		"cli.init.config_load_fail":     "초기 설정을 읽을 수 없음",
		"cli.init.dir_mkdir_fail":       "디렉토리를 만들 수 없음: %s",
		"cli.init.file_create_fail":     "파일을 만들 수 없음: %s",
		"cli.init.file_write_fail":      "파일을 쓸 수 없음: %s",
		"cli.init.gitignore_read_fail":  ".gitignore을 읽을 수 없음",
		"cli.init.gitignore_write_fail": ".gitignore을 갱신할 수 없음",
		"cli.init.config_yaml": `# engram 위키 설정. 프론트매터 속성, 임계값, 디렉토리 매핑을 정의합니다.
preset: %s

# 프론트매터 속성. 프리셋(minimal < personal < team)이 시작점이며
# 개별 속성을 아래에서 따로 켜고 끌 수 있습니다.
# 사용 가능한 속성: type, artifact_stage, status, indexable, tags, source_refs,
# derived_from, related, source_channel, derived_context, scope, sensitivity,
# trigger_mode, workflow
# axes:
#   scope: true

# 문서 종류(type 속성의 허용값). 위키에 맞게 추가합니다.
# types: [concept, project, system, decision, procedure, incident,
#   meeting-summary, agent-workflow, source-summary, inbox-note]

# taxonomy. topics는 개방 집합이고 forms는 폐쇄 집합입니다.
# topics: [go, cli]
# forms: [memo, report]

# 임계값. min_wikilinks만 승급 거절 사유이고 나머지는 경고에 쓰입니다.
min_wikilinks: 2    # promote 게이트. 0으로 두면 게이트가 꺼집니다
stale_days: 30      # 재발견 대상 판정 기준 일수
max_lines: 1000     # 문서 길이 경고 상한
broad_topic_pct: 25 # 광범위 주제 비율 경고 상한(퍼센트)

# 문서가 놓이는 디렉토리와 루트에 있어야 하는 파일
page_dirs: [inbox, sources, context, archive]
root_files: [index.md]

# 문서가 아닌 마크다운. 같은 파일명이면 깊이와 무관하게 순회에서 뺍니다.
# 기본값은 README.md 하나입니다. 비워 두면 README.md도 문서로 검사합니다.
# ignore_files: [README.md]

# 이 위키가 폐기한 프론트매터 키. 목록에 있는 키가 문서에 있으면 lint가
# error로 잡습니다. 기본값은 빈 목록이고 마이그레이션할 때 채웁니다.
# deprecated_fields: [quality_level, review_after]
`,
		"cli.init.index_title":     "# engram 위키",
		"cli.init.index_intro":     "이 문서는 위키의 첫 문서입니다. 위키를 소개하는 안내로 바꿉니다.",
		"cli.init.index_guide":     "새 자료는 inbox에 넣고 승급 파이프라인을 따라 context로 옮깁니다.",
		"cli.init.dir_inbox":       "새 자료가 들어오는 곳",
		"cli.init.dir_sources":     "원본을 보존하는 곳",
		"cli.init.dir_context":     "정리된 문서가 사는 곳",
		"cli.init.dir_archive":     "승급에서 물러난 문서가 가는 곳",
		"cli.init.dir_other":       "문서 디렉토리",
		"cli.init.file_config":     "위키 설정. 속성과 임계값을 여기서 조정하세요",
		"cli.init.file_index":      "첫 문서. 위키 소개로 채우세요",
		"cli.init.file_gitignore":  ".engram/ 캐시 디렉토리를 git에서 제외합니다",
		"cli.init.done":            "위키를 초기화했습니다: %s (프리셋: %s)",
		"cli.init.dirs_header":     "디렉토리:",
		"cli.init.files_header":    "파일:",
		"cli.init.next_header":     "다음 단계:",
		"cli.init.step_inbox":      "inbox에 첫 자료를 넣으세요",
		"cli.init.step_config":     "%s을 열어 속성과 임계값을 위키에 맞게 조정하세요",
		"cli.init.step_fill_index": "index.md를 위키 소개로 채우세요",

		// reindex
		"cli.reindex.short": "검색 색인을 만듭니다",
		"cli.reindex.long": `위키를 순회해 검색 색인을 만들고 .engram/index.json에 씁니다.

reindex가 인덱스를 만드는 유일한 커맨드입니다. 조회 커맨드는 색인 파일을
갱신하지 않습니다. 경로를 생략하면 현재 디렉토리입니다.`,
		"cli.reindex.walk_fail":        "위키를 순회할 수 없음",
		"cli.reindex.build_fail":       "색인을 만들 수 없음",
		"cli.reindex.save_fail":        "색인을 쓸 수 없음",
		"cli.reindex.stat_fail":        "색인 파일을 확인할 수 없음",
		"cli.reindex.done":             "색인을 만들었습니다: %s",
		"cli.reindex.summary":          "문서 %d개, 토큰 %d개, 크기 %d 바이트",
		"cli.reindex.not_wiki":         "위키가 아닌 디렉토리입니다: %s\n먼저 engram init을 실행하세요",
		"cli.reindex.path_check_fail":  "대상 경로를 확인할 수 없음",
		"cli.reindex.config_load_fail": "위키 설정을 읽을 수 없음",

		// migrate
		"cli.migrate.short": "기존 문서를 지금의 설정과 규칙에 맞게 정리합니다",
		"cli.migrate.long": `기존 문서를 지금의 engram.yaml과 지금의 규칙에 맞춥니다.

켜진 속성의 필수 필드를 단계별 초기값으로 채우고, 꺼진 속성의 필드를 지우고,
문서가 놓인 디렉토리에 맞게 artifact_stage를 고칩니다.
파일을 옮기지 않고 슬러그를 바꾸지 않습니다. 문서를 승급시키지도 않습니다.
inbox에 있으면서 context라고 선언한 문서는 선언이 inbox로 내려갈 뿐입니다.
올리려면 engram promote를 쓰세요.

기본은 시험 실행입니다. 실제로 쓰려면 --apply를 주세요.
꺼진 속성의 필드에 값이 있으면 --force 없이는 지우지 않습니다. 값을 비운
필드는 --force 없이도 지웁니다.
승급 게이트 위반과 깨진 위키링크는 고치지 않고 보고합니다. 어떤 문서에
이어야 하는지는 판단이므로 engram promote나 demote로 직접 정리하세요.`,
		"cli.migrate.flag_read_fail":  "--%s 플래그를 읽을 수 없음",
		"cli.migrate.flag_apply":      "변경을 파일에 씁니다. 기본은 시험 실행입니다",
		"cli.migrate.flag_force":      "값이 있는 꺼진 속성 필드도 지웁니다",
		"cli.migrate.flag_wiki":       "대상 위키 경로",
		"cli.migrate.all_ok":          "검사한 문서 %d개가 규칙에 맞습니다.",
		"cli.migrate.applied_summary": "검사한 문서 %d개 중 %d개가 규칙에 맞지 않아 %d개를 고쳤습니다.",
		"cli.migrate.applied_partial": "%d개는 migrate로도 완전히 맞추지 못했고 남은 필드를 아래에 적습니다.",
		"cli.migrate.dry_summary":     "검사한 문서 %d개 중 %d개가 규칙에 맞지 않습니다.",
		"cli.migrate.dry_partial":     "이 중 %d개는 migrate로도 완전히 맞추지 못합니다. 남은 필드를 아래에 적습니다.",
		"cli.migrate.dry_notice":      "시험 실행이므로 파일을 쓰지 않았습니다. 적용하려면 --apply를 주세요.",
		"cli.migrate.unparsed":        "프론트매터를 읽을 수 없어 건너뛴 문서 %d개가 있습니다. 먼저 프론트매터를 고치세요.",
		"cli.migrate.more_detail":     "... 외 %d개 문서의 상세는 --json으로 볼 수 있습니다.",
		"cli.migrate.change_stage":    "artifact_stage: %s -> %s (문서가 놓인 디렉토리에 맞춥니다)",
		"cli.migrate.change_fill":     "%s: (없음) -> %s",
		"cli.migrate.change_remove":   "%s: %s -> (삭제)",
		"cli.migrate.blocked":         "%s: %s (지우면 이 값을 잃습니다. 지우려면 --force를 주세요)",
		"cli.migrate.gate_advice":     "승급 게이트 위반 문서 %d개는 migrate가 고치지 않습니다. 어떤 문서에 이어야 하는지는 판단입니다. engram promote나 demote로 정리하세요.",
		"cli.migrate.broken_advice":   "깨진 위키링크가 있는 문서 %d개는 migrate가 고치지 않습니다. 슬러그를 고치거나 대상 문서를 만드세요.",
		"cli.migrate.empty_value":     "(빈 값)",

		// sync
		"cli.sync.short": "git 이력과 파일명에서 날짜 필드를 정정합니다",
		"cli.sync.long": `git 이력에서 날짜 필드를 정정합니다.

updated 는 context 단계 문서에만 쓰고 마지막 커밋 날짜로 채웁니다.
sourced_at 은 sources 단계 문서에만 쓰고 최초 커밋 날짜로 정정합니다.
created 는 비어 있을 때만 채웁니다. 재료는 git 최초 커밋 날짜를 우선하고
이력이 없으면 파일명의 날짜 접두사(YYYY-MM-DD- 또는 YYYY-MM-)를 봅니다.
이미 있는 값은 지우지 않습니다. 지우는 것은 migrate 의 일입니다.

기본은 dry-run 입니다. 실제로 쓰려면 --apply 를 주세요.
값이 이미 같으면 쓰지 않으므로 두 번 돌려도 같은 결과입니다.
커밋되지 않은 파일 중 파일명에 재료가 없는 것은 건너뛰고 개수를 알립니다.
워킹트리가 더러워도 판정 근거가 커밋 이력이라 그대로 돕니다.`,
		"cli.sync.flag_read_fail": "--%s 플래그를 읽을 수 없음",
		"cli.sync.walk_fail":      "위키를 순회할 수 없음",
		"cli.sync.flag_apply":     "변경을 파일에 씁니다. 기본은 dry-run 입니다",
		"cli.sync.flag_wiki":      "대상 위키 경로",
		"cli.sync.doc_write_fail": "문서를 쓸 수 없음: %s",
		"cli.sync.none":           "정정할 문서가 없습니다",
		"cli.sync.applied":        "정정했습니다. 문서 %d개, 필드 %d개",
		"cli.sync.dry_run":        "정정 대상 문서 %d개, 필드 %d개. 아직 dry-run 입니다. 적용하려면 --apply 를 주세요",
		"cli.sync.field_absent":   "(없음)",
		"cli.sync.current":        "현재 %s",
		"cli.sync.uncommitted":    "커밋되지 않은 문서 %d개는 건너뛰었습니다",
		"cli.sync.bulk_only":      "문서 %[2]d개 이상을 한 번에 고친 커밋에만 등장하는 문서 %[1]d개는 날짜를 채우지 않았습니다",
		"cli.sync.filename_dates": "파일명 접두사에서 날짜를 채운 필드는 %d개입니다",

		// rules
		"cli.rules.short": "이 위키에 적용되는 규칙을 다룹니다",
		"cli.rules.long": `이 위키에 지금 적용되는 규칙을 다룹니다.

rules show가 규칙 전부를 읽기 전용으로 보여줍니다. eject 없이
규칙을 확인하려는 경우를 위한 커맨드입니다.`,
		"cli.rules.show_short": "이 위키에 적용되는 규칙 전부를 보여줍니다",
		"cli.rules.show_long": `이 위키에 지금 적용되는 규칙 전부를 보여줍니다.

프리셋과 프론트매터 속성, 단계별 필수 필드, 허용값, 임계값, 승급 게이트, lint 규칙,
디렉토리를 절로 나눠 보여줍니다. 위키의 engram.yaml을 읽어 프리셋과
사용자 설정이 합쳐진 결과를 냅니다. 제품 기본값이 아니라 이 위키의
값입니다.

무엇도 바꾸지 않습니다. 규칙은 읽는 것입니다.`,
		"cli.rules.flag_wiki":          "대상 위키 경로",
		"cli.rules.header":             "이 위키의 규칙 (프리셋: %s)",
		"cli.rules.header_note":        "프리셋과 engram.yaml 설정이 합쳐진 결과입니다. 제품 기본값이 아니라 이 위키의 값입니다.",
		"cli.rules.axes_header":        "프론트매터 속성 %d종 (켜짐 %d, 꺼짐 %d)",
		"cli.rules.axes_on":            "켜짐",
		"cli.rules.axes_off":           "꺼짐",
		"cli.rules.axes_off_note":      "꺼진 속성은 필수가 아니고, 문서에 있으면 schema.axis-off가 잡습니다",
		"cli.rules.required_header":    "단계별 필수 필드",
		"cli.rules.closed_header":      "허용값. 폐쇄 집합은 정의되지 않은 값이 오류입니다",
		"cli.rules.closed_empty":       "정의되지 않아 이 위키는 값을 검사하지 않습니다",
		"cli.rules.open_header":        "허용값. 개방 집합은 정의되지 않은 값이 경고입니다",
		"cli.rules.open_empty":         "정의되지 않아 모든 값이 경고 대상입니다",
		"cli.rules.thresholds_header":  "임계값",
		"cli.rules.th_min_wikilinks":   "승급 게이트의 거절 기준입니다",
		"cli.rules.th_stale_days":      "resurface가 다시 꺼낼 문서를 고르는 기준 일수입니다",
		"cli.rules.th_max_lines":       "body.max-lines 경고의 상한 줄 수입니다",
		"cli.rules.th_broad_topic_pct": "wiki.broad-topic 진단의 상한 비율입니다",
		"cli.rules.gate_header":        "승급 게이트",
		"cli.rules.gate_only_one":      "게이트가 승급을 거절하는 조건은 하나뿐입니다.",
		"cli.rules.gate_condition":     "문서가 위키에 실제로 있는 문서를 가리키는 고유 위키링크 수가 min_wikilinks %d개에 못 미치는 것입니다.\n  없는 슬러그, 자기 자신, inbox 문서를 가리키는 링크는 세지 않습니다.",
		"cli.rules.gate_scope":         "promote와 new는 이것만 묻습니다. 길이도 형식도 주제도 막지 않습니다.",
		"cli.rules.lint_header":        "lint 규칙 %d종",
		"cli.rules.lint_severity_note": "등급에서 error와 reject는 lint를 종료 코드 1로 끝냅니다. warn은 알리고 통과시킵니다.",
		"cli.rules.lint_reject_note":   "승급 시점에 문서를 거절하는 것은 reject 하나뿐입니다.",
		"cli.rules.dirs_header":        "디렉토리",

		// eject
		"cli.eject.short": "규칙 명세와 Python 린터를 위키로 내보냅니다",
		"cli.eject.long": `내장 규칙을 사용자가 고칠 수 있는 파일로 내보냅니다.

규칙 명세 문서(meta/), 문서 단위 규칙을 판정하는 Python 린터
(scripts/lint-frontmatter.py), 커밋 훅(.githooks/pre-commit),
에이전트 계약(AGENTS.md), 줄바꿈 설정(.gitattributes)을 만듭니다.
전부 이 위키의 engram.yaml 을 반영해 생성합니다. 린터는 값을 박지
않고 engram.yaml 을 실행 시점에 읽으므로 설정을 바꾸면 따라갑니다.

제품에서 나가는 문이 아닙니다. 규칙만 사용자 것이 되고 연산은
engram 에 남습니다. 이후에도 search, recall, resurface, bridge,
digest, backlinks 가 그대로 동작합니다.

단방향입니다. 되돌리는 커맨드가 없으므로 이미 있는 파일을 덮지
않습니다. 충돌하면 무엇이 충돌하는지 전부 알리고 멈춥니다.
--force 를 주면 덮되 무엇을 덮는지 먼저 알립니다.
--dry-run 은 무엇이 만들어질지 봅니다. 쓰지는 않습니다.

린터와 훅에는 python3 가 필요합니다. Windows 는 기본 제공되지
않으므로 따로 설치해야 합니다.`,
		"cli.eject.flag_read_fail":      "--%s 플래그를 읽을 수 없음",
		"cli.eject.conflict":            "이미 있는 파일을 덮지 않습니다. 덮으려면 --force 를 주세요",
		"cli.eject.flag_force":          "이미 있는 파일을 덮어 씁니다",
		"cli.eject.flag_dry_run":        "무엇이 만들어질지 봅니다. 파일을 쓰지 않습니다",
		"cli.eject.flag_wiki":           "대상 위키 경로",
		"cli.eject.dir_mkdir_fail":      "디렉토리를 만들 수 없음",
		"cli.eject.artifact_write_fail": "산출물을 쓸 수 없음: %s",
		"cli.eject.dry_run":             "만들 예정인 파일 %d개 (dry-run. 아직 쓰지 않았습니다)",
		"cli.eject.done":                "만들었습니다. 파일 %d개",
		"cli.eject.overwritten":         "덮어 쓴 파일 %d개",
		"cli.eject.guide_header":        "안내:",
		"cli.eject.hook_enable":         "훅을 켜려면: git config core.hooksPath .githooks",
		"cli.eject.still_works":         "eject 이후에도 search, recall, resurface, bridge, digest, backlinks 가 그대로 동작합니다",
		"cli.eject.python_note":         "린터와 훅에는 python3 이 필요합니다. Windows 는 기본 제공되지 않으므로 설치해야 합니다",
		"cli.eject.conflicts_count":     "이미 있는 파일이 %d개 있습니다",

		// skills
		"cli.skills.short": "에이전트 스킬을 다룹니다",
		"cli.skills.long": `에이전트가 engram을 다루는 법을 가르치는 스킬 문서를 다룹니다.

skills install이 바이너리에 임베드된 스킬 문서를 감지된 에이전트의
스킬 디렉토리에 심습니다. 이것이 LLM 통합의 전부입니다(ADR 0014).

이 커맨드는 위키가 아니라 에이전트를 다룹니다. 위키 밖에서 실행해도
동작합니다.`,
		"cli.skills.install_short": "스킬 문서를 에이전트 스킬 디렉토리에 심습니다",
		"cli.skills.install_long": `임베드된 스킬 문서를 에이전트의 스킬 디렉토리에 심습니다.

문서는 정적입니다. 위키의 임계값이나 허용값을 담지 않습니다. 그 위키에
적용되는 규칙은 에이전트가 engram rules show로 얻습니다. 그래서 한 번
심으면 모든 위키에 통합니다.

--dir이 없으면 홈 디렉토리에서 실제로 존재하는 에이전트 스킬 디렉토리를
찾아 전부에 심습니다. 없는 도구를 위해 디렉토리를 만들지 않습니다.
하나도 찾지 못하면 실패하니 --dir로 직접 지정하세요. --dir은 스킬
루트(스킬 디렉토리들이 있는 곳)를 받고 그 아래 engram/SKILL.md를
만듭니다. 이미 있는 디렉토리여야 합니다.

이미 있는 파일은 덮지 않습니다. 충돌하면 무엇이 충돌하는지 알리고
멈춥니다. 덮으려면 --force를 주세요.`,
		"cli.skills.flag_read_fail":  "--%s 플래그를 읽을 수 없음",
		"cli.skills.dir_not_dir":     "--dir 경로가 디렉토리가 아닙니다: %s\n실제로 있는 스킬 루트 디렉토리를 주세요",
		"cli.skills.home_fail":       "홈 디렉토리를 알 수 없음",
		"cli.skills.home_fail_hint":  "--dir로 설치 위치를 직접 지정하세요",
		"cli.skills.detect_fail":     "설치 대상이 될 에이전트 스킬 디렉토리를 찾지 못했습니다\n찾아본 곳(홈 디렉토리 기준):\n  %s\n없는 도구를 위해 디렉토리를 만들지 않습니다. 설치 위치를 --dir로 직접 지정하세요",
		"cli.skills.conflict":        "이미 있는 파일을 덮지 않습니다. 덮으려면 --force를 주세요",
		"cli.skills.dir_mkdir_fail":  "스킬 디렉토리를 만들 수 없음",
		"cli.skills.doc_write_fail":  "스킬 문서를 쓸 수 없음: %s",
		"cli.skills.flag_dir":        "설치 위치를 직접 지정합니다. 감지를 건너뜁니다",
		"cli.skills.flag_force":      "이미 있는 파일을 덮어 씁니다",
		"cli.skills.flag_dry_run":    "어디에 무엇을 심을지만 냅니다. 쓰지 않습니다",
		"cli.skills.dry_run":         "심을 예정인 파일 %d개 (dry-run. 아직 쓰지 않았습니다)",
		"cli.skills.done":            "심었습니다. 파일 %d개",
		"cli.skills.overwritten":     "덮어 쓴 파일 %d개",
		"cli.skills.restart_note":    "에이전트를 다시 시작해야 스킬이 잡힐 수 있습니다",
		"cli.skills.conflicts_count": "이미 있는 파일이 %d개 있습니다",
	})

	Register(LangEN, map[string]string{
		// root
		"cli.root.short": "CLI to manage the promotion pipeline of a knowledge wiki",
		"cli.root.long": `engram is a CLI that manages document state and the promotion pipeline of a knowledge wiki.

Every query command supports JSON output with --json, and --now pins the
reference time for deterministic results.`,
		"cli.root.flag_json": "Print the result as JSON",
		"cli.root.flag_now":  "Reference time (RFC3339). Empty means the current time",
		"cli.root.flag_lang": "Output language (ko, en). Empty means the %s environment variable, then ko",

		// version
		"cli.version.short":     "Print version and build info",
		"cli.version.commit_at": "commit time: %s",

		// init
		"cli.init.short": "Create a new wiki",
		"cli.init.long": `Create a new wiki at the given path. Without a path, the current directory.

Creates the directory layout, the engram.yaml config, the first document
index.md, and .gitignore. Refuses if engram.yaml already exists, to preserve
the existing wiki.`,
		"cli.init.flag_preset":          "Schema preset. One of minimal, personal, team",
		"cli.init.preset_invalid":       "--preset value is not allowed: %q (allowed: minimal, personal, team)",
		"cli.init.already_wiki":         "Target is already an engram wiki: %s\nExisting wikis are not overwritten. Choose another path or edit the existing %s by hand",
		"cli.init.path_check_fail":      "Cannot check target path",
		"cli.init.root_mkdir_fail":      "Cannot create wiki root",
		"cli.init.config_load_fail":     "Cannot read initial config",
		"cli.init.dir_mkdir_fail":       "Cannot create directory: %s",
		"cli.init.file_create_fail":     "Cannot create file: %s",
		"cli.init.file_write_fail":      "Cannot write file: %s",
		"cli.init.gitignore_read_fail":  "Cannot read .gitignore",
		"cli.init.gitignore_write_fail": "Cannot update .gitignore",
		"cli.init.config_yaml": `# engram wiki config. Defines frontmatter attributes, thresholds, and directory mapping.
preset: %s

# Frontmatter attributes. A preset (minimal < personal < team) is the starting
# point and each attribute can be turned on or off below.
# Available attributes: type, artifact_stage, status, indexable, tags, source_refs,
# derived_from, related, source_channel, derived_context, scope, sensitivity,
# trigger_mode, workflow
# axes:
#   scope: true

# Document types (allowed values of the type attribute). Add ones that fit the wiki.
# types: [concept, project, system, decision, procedure, incident,
#   meeting-summary, agent-workflow, source-summary, inbox-note]

# Taxonomy. topics is an open set and forms is a closed set.
# topics: [go, cli]
# forms: [memo, report]

# Thresholds. Only min_wikilinks rejects promotion; the rest feed warnings.
min_wikilinks: 2    # promote gate. Set 0 to turn the gate off
stale_days: 30      # Days used to pick resurface candidates
max_lines: 1000     # Document length warning ceiling
broad_topic_pct: 25 # Broad topic ratio warning ceiling (percent)

# Directories where documents live, and files that must sit at the root
page_dirs: [inbox, sources, context, archive]
root_files: [index.md]

# Markdown that is not a document. With the same filename it is excluded from
# the walk regardless of depth. The default is README.md alone. Empty means
# README.md is also checked as a document.
# ignore_files: [README.md]

# Frontmatter keys this wiki has retired. A listed key present in a document
# is a lint error. The default is an empty list; fill it when migrating.
# deprecated_fields: [quality_level, review_after]
`,
		"cli.init.index_title":     "# engram wiki",
		"cli.init.index_intro":     "This is the first document of the wiki. Replace it with an introduction.",
		"cli.init.index_guide":     "Put new material in inbox and move it to context through the promotion pipeline.",
		"cli.init.dir_inbox":       "Where new material arrives",
		"cli.init.dir_sources":     "Where originals are preserved",
		"cli.init.dir_context":     "Where organized documents live",
		"cli.init.dir_archive":     "Where documents retired from promotion go",
		"cli.init.dir_other":       "Document directory",
		"cli.init.file_config":     "Wiki config. Adjust attributes and thresholds here",
		"cli.init.file_index":      "First document. Fill it with a wiki introduction",
		"cli.init.file_gitignore":  "Excludes the .engram/ cache directory from git",
		"cli.init.done":            "Initialized wiki: %s (preset: %s)",
		"cli.init.dirs_header":     "Directories:",
		"cli.init.files_header":    "Files:",
		"cli.init.next_header":     "Next steps:",
		"cli.init.step_inbox":      "Put your first material in inbox",
		"cli.init.step_config":     "Open %s and adjust attributes and thresholds for your wiki",
		"cli.init.step_fill_index": "Fill index.md with a wiki introduction",

		// reindex
		"cli.reindex.short": "Build the search index",
		"cli.reindex.long": `Walks the wiki, builds the search index, and writes .engram/index.json.

reindex is the only command that builds the index. Query commands never
update the index file. Without a path, the current directory.`,
		"cli.reindex.walk_fail":        "Cannot walk the wiki",
		"cli.reindex.build_fail":       "Cannot build the index",
		"cli.reindex.save_fail":        "Cannot write the index",
		"cli.reindex.stat_fail":        "Cannot check the index file",
		"cli.reindex.done":             "Built the index: %s",
		"cli.reindex.summary":          "%d documents, %d tokens, %d bytes",
		"cli.reindex.not_wiki":         "Not a wiki directory: %s\nRun engram init first",
		"cli.reindex.path_check_fail":  "Cannot check target path",
		"cli.reindex.config_load_fail": "Cannot read wiki config",

		// migrate
		"cli.migrate.short": "Align existing documents with the current config and rules",
		"cli.migrate.long": `Aligns existing documents with the current engram.yaml and the current rules.

Fills required fields of enabled attributes with stage defaults, removes fields
of disabled attributes, and fixes artifact_stage to match the document's directory.
It moves no files and changes no slugs. It does not promote documents.
A document sitting in inbox while declaring context only gets its declaration
lowered to inbox. Use engram promote to raise it.

The default is a dry run. Pass --apply to write.
Fields of disabled attributes that hold a value are not removed without --force.
Empty fields are removed even without --force.
Promotion gate violations and broken wikilinks are reported, not fixed. Which
document should link where is a judgment, so settle it with engram promote
or demote.`,
		"cli.migrate.flag_read_fail":  "Cannot read the --%s flag",
		"cli.migrate.flag_apply":      "Write changes to files. Default is a dry run",
		"cli.migrate.flag_force":      "Also remove disabled-attribute fields that hold a value",
		"cli.migrate.flag_wiki":       "Target wiki path",
		"cli.migrate.all_ok":          "All %d checked documents conform to the rules.",
		"cli.migrate.applied_summary": "Of %d checked documents, %d did not conform and %d were fixed.",
		"cli.migrate.applied_partial": "Migrate could not fully align %d of them; remaining fields are listed below.",
		"cli.migrate.dry_summary":     "Of %d checked documents, %d do not conform to the rules.",
		"cli.migrate.dry_partial":     "Migrate cannot fully align %d of them. Remaining fields are listed below.",
		"cli.migrate.dry_notice":      "This was a dry run, so no files were written. Pass --apply to apply.",
		"cli.migrate.unparsed":        "documents skipped because their frontmatter could not be read: %d. Fix the frontmatter first.",
		"cli.migrate.more_detail":     "... details of %d more documents are in --json.",
		"cli.migrate.change_stage":    "artifact_stage: %s -> %s (matches the document's directory)",
		"cli.migrate.change_fill":     "%s: (none) -> %s",
		"cli.migrate.change_remove":   "%s: %s -> (removed)",
		"cli.migrate.blocked":         "%s: %s (removing loses this value. Pass --force to remove)",
		"cli.migrate.gate_advice":     "Migrate does not fix %d documents violating the promotion gate. Which document should link where is a judgment; settle it with engram promote or demote.",
		"cli.migrate.broken_advice":   "Migrate does not fix %d documents with broken wikilinks. Fix the slug or create the target document.",
		"cli.migrate.empty_value":     "(empty)",

		// sync
		"cli.sync.short": "Corrects date fields from git history and filenames",
		"cli.sync.long": `Corrects date fields from git history.

updated is written only to context-stage documents, with the file's last
commit date. sourced_at is written only to sources-stage documents,
corrected to the first commit date. created is filled only when empty:
the first commit date comes first, and when there is no history the date
prefix of the filename (YYYY-MM-DD- or YYYY-MM-) is used.
Existing values are never erased. Erasing is migrate's job.

The default is a dry run. Pass --apply to write.
Values already equal are not written, so running twice gives the same result.
Uncommitted files without a usable filename prefix are skipped and counted.
A dirty worktree is fine: the verdict rests on commit history.`,
		"cli.sync.flag_read_fail": "Cannot read the --%s flag",
		"cli.sync.walk_fail":      "Cannot walk the wiki",
		"cli.sync.flag_apply":     "Write changes to files. Default is a dry run",
		"cli.sync.flag_wiki":      "Target wiki path",
		"cli.sync.doc_write_fail": "Cannot write document: %s",
		"cli.sync.none":           "No documents to correct",
		"cli.sync.applied":        "Corrected. %d documents, %d fields",
		"cli.sync.dry_run":        "%d documents, %d fields to correct. Still a dry run. Pass --apply to apply",
		"cli.sync.field_absent":   "(none)",
		"cli.sync.current":        "now %s",
		"cli.sync.uncommitted":    "Skipped %d uncommitted documents",
		"cli.sync.bulk_only":      "Left dates unfilled for %[1]d documents that appear only in commits touching %[2]d or more documents at once",
		"cli.sync.filename_dates": "Filled %d fields from filename prefixes",

		// rules
		"cli.rules.short": "Manage the rules applied to this wiki",
		"cli.rules.long": `Manage the rules currently applied to this wiki.

rules show lists every rule read-only. It is for checking the rules
without eject.`,
		"cli.rules.show_short": "Show every rule applied to this wiki",
		"cli.rules.show_long": `Shows every rule currently applied to this wiki.

Preset and frontmatter attributes, per-stage required fields, allowed values,
thresholds, the promotion gate, lint rules, and directories, each in its own
section. Reads the wiki's engram.yaml and reports the merged result of the
preset and user settings, not the product defaults but this wiki's values.

Changes nothing. Rules are for reading.`,
		"cli.rules.flag_wiki":          "Target wiki path",
		"cli.rules.header":             "Rules of this wiki (preset: %s)",
		"cli.rules.header_note":        "The merged result of the preset and engram.yaml settings. Not the product defaults but this wiki's values.",
		"cli.rules.axes_header":        "%d frontmatter attributes (%d on, %d off)",
		"cli.rules.axes_on":            "on",
		"cli.rules.axes_off":           "off",
		"cli.rules.axes_off_note":      "Disabled axes are not required; schema.axis-off flags them when present in a document",
		"cli.rules.required_header":    "Required fields by stage",
		"cli.rules.closed_header":      "Allowed values. In a closed set, an undefined value is an error",
		"cli.rules.closed_empty":       "undefined, so this wiki does not check values",
		"cli.rules.open_header":        "Allowed values. In an open set, an undefined value is a warning",
		"cli.rules.open_empty":         "undefined, so every value draws a warning",
		"cli.rules.thresholds_header":  "Thresholds",
		"cli.rules.th_min_wikilinks":   "The rejection criterion of the promotion gate",
		"cli.rules.th_stale_days":      "Days used to pick resurface candidates",
		"cli.rules.th_max_lines":       "Line ceiling of the body.max-lines warning",
		"cli.rules.th_broad_topic_pct": "Ratio ceiling of the wiki.broad-topic diagnostic",
		"cli.rules.gate_header":        "Promotion gate",
		"cli.rules.gate_only_one":      "Only one condition makes the gate reject a promotion.",
		"cli.rules.gate_condition":     "The document's count of unique wikilinks resolving to an existing document falls short of min_wikilinks %d.\n  Missing slugs, self-links, and inbox documents do not count.",
		"cli.rules.gate_scope":         "promote and new ask only this. Length, format, and topic never block.",
		"cli.rules.lint_header":        "%d lint rules",
		"cli.rules.lint_severity_note": "Of the severities, error and reject end lint with exit code 1. warn informs and passes.",
		"cli.rules.lint_reject_note":   "Only reject refuses a document at promotion time.",
		"cli.rules.dirs_header":        "Directories",

		// eject
		"cli.eject.short": "Export the rule spec and the Python linter into the wiki",
		"cli.eject.long": `Exports built-in rules as files the user can edit.

Creates the rule spec documents (meta/), the Python linter that judges
per-document rules (scripts/lint-frontmatter.py), the commit hook
(.githooks/pre-commit), the agent contract (AGENTS.md), and the line-ending
config (.gitattributes). All are generated from this wiki's engram.yaml.
The linter bakes no values and reads engram.yaml at run time, so it follows
config changes.

This is not an exit from the product. Only the rules become yours; the
computation stays in engram. search, recall, resurface, bridge, digest, and
backlinks keep working afterwards.

One-way. There is no command to undo it, so existing files are not
overwritten. On conflict it reports every conflict and stops.
--force overwrites but first reports what it overwrites.
--dry-run shows what would be created. It writes nothing.

The linter and hook need python3. Windows does not ship it, so install it
separately.`,
		"cli.eject.flag_read_fail":      "Cannot read the --%s flag",
		"cli.eject.conflict":            "Existing files are not overwritten. Pass --force to overwrite",
		"cli.eject.flag_force":          "Overwrite existing files",
		"cli.eject.flag_dry_run":        "Show what would be created. Writes nothing",
		"cli.eject.flag_wiki":           "Target wiki path",
		"cli.eject.dir_mkdir_fail":      "Cannot create directory",
		"cli.eject.artifact_write_fail": "Cannot write artifact: %s",
		"cli.eject.dry_run":             "Files to create: %d (dry-run. Nothing written yet)",
		"cli.eject.done":                "Created. Files: %d",
		"cli.eject.overwritten":         "Overwritten files: %d",
		"cli.eject.guide_header":        "Notes:",
		"cli.eject.hook_enable":         "To enable the hook: git config core.hooksPath .githooks",
		"cli.eject.still_works":         "After eject, search, recall, resurface, bridge, digest, and backlinks keep working",
		"cli.eject.python_note":         "The linter and hook need python3. Windows does not ship it, so install it",
		"cli.eject.conflicts_count":     "Files that already exist: %d",

		// skills
		"cli.skills.short": "Manage agent skills",
		"cli.skills.long": `Manages the skill documents that teach agents how to use engram.

skills install plants the skill documents embedded in the binary into the
skill directories of detected agents. That is the whole LLM integration
(ADR 0014).

This command deals with agents, not wikis. It works outside a wiki.`,
		"cli.skills.install_short": "Plant the skill documents into agent skill directories",
		"cli.skills.install_long": `Plants the embedded skill documents into agent skill directories.

The documents are static. They carry no wiki thresholds or allowed values.
The rules for a given wiki are obtained by the agent via engram rules show.
So once planted, they serve every wiki.

Without --dir, it finds agent skill directories that actually exist under
the home directory and plants into all of them. It creates no directories
for tools you do not have. If it finds none it fails, so point --dir at one.
--dir takes a skill root (where skill directories live) and creates
engram/SKILL.md under it. It must already exist.

Existing files are not overwritten. On conflict it reports what conflicts
and stops. Pass --force to overwrite.`,
		"cli.skills.flag_read_fail":  "Cannot read the --%s flag",
		"cli.skills.dir_not_dir":     "--dir path is not a directory: %s\nGive a skill root directory that actually exists",
		"cli.skills.home_fail":       "Cannot determine the home directory",
		"cli.skills.home_fail_hint":  "Point --dir at the install location directly",
		"cli.skills.detect_fail":     "No agent skill directories found to install into\nSearched (relative to the home directory):\n  %s\nNo directories are created for tools you do not have. Point --dir at the install location",
		"cli.skills.conflict":        "Existing files are not overwritten. Pass --force to overwrite",
		"cli.skills.dir_mkdir_fail":  "Cannot create skill directory",
		"cli.skills.doc_write_fail":  "Cannot write skill document: %s",
		"cli.skills.flag_dir":        "Point at the install location directly. Skips detection",
		"cli.skills.flag_force":      "Overwrite existing files",
		"cli.skills.flag_dry_run":    "Show where and what would be planted. Writes nothing",
		"cli.skills.dry_run":         "Files to plant: %d (dry-run. Nothing written yet)",
		"cli.skills.done":            "Planted. Files: %d",
		"cli.skills.overwritten":     "Overwritten files: %d",
		"cli.skills.restart_note":    "Restart the agent for the skill to be picked up",
		"cli.skills.conflicts_count": "Files that already exist: %d",
	})
}
