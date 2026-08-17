// Package i18n의 engine 영역 카탈로그. internal/lint, internal/doctor,
// internal/eject 의 사용자 대면 문자열. ADR 0049.
//
// eject.* 는 내보내는 산출물이 시점 언어로 굶는 문자열이다. Python 이
// 채우는 %s, %d 포맷을 그대로 담으므로 Go 에서 fmt 인자로 채우면 안 된다.
package i18n

func init() {
	Register(LangKO, map[string]string{
		// lint 규칙 설명. rules show 와 meta/lint-rules.md 이 낸다.
		"lint.rule.gate_min_wikilinks":        "context 디렉토리 아래 문서의 고유 위키링크 수가 min_wikilinks 미달. 게이트의 유일한 거절 사유(ADR 0040)",
		"lint.rule.frontmatter_missing":       "프론트매터 블록이 아예 없는 문서",
		"lint.rule.frontmatter_unclosed":      "닫는 --- 없이 끝난 프론트매터",
		"lint.rule.frontmatter_yaml":          "프론트매터 YAML 문법 오류",
		"lint.rule.frontmatter_missing_field": "단계별 필수 필드 누락",
		"lint.rule.schema_allowed_value":      "허용 집합 밖의 필드 값. artifact_stage, status, scope, sensitivity, trigger_mode",
		"lint.rule.schema_axis_off":           "설정이 끈 속성이 문서에 있음",
		"lint.rule.location_stage_agreement":  "문서가 놓인 디렉토리와 artifact_stage 값의 불일치. context를 선언했는데 context/ 밖에 있으면 error, 그 밖의 불일치는 warn(ADR 0035)",
		"lint.rule.taxonomy_forms":            "forms 폐쇄 집합에 없는 form 값",
		"lint.rule.sources_updated":           "sources 문서의 원본 보존에 어긋나는 updated 필드",
		"lint.rule.taxonomy_topics":           "설정에 정의되지 않은 topics 값. 개방 집합이라 경고",
		"lint.rule.body_max_lines":            "문서 줄 수가 max_lines 초과",
		"lint.rule.link_broken":               "위키링크가 가리키는 문서가 위키에 없음",
		"lint.rule.graph_orphan":              "들어오는 관계와 나가는 관계가 모두 없는 문서",
		"lint.rule.gate_deferred":             "링크 가능한 대상 문서가 부족해 게이트 유예",
		"lint.rule.wiki_broad_topic":          "한 주제가 전체 문서의 broad_topic_pct를 넘게 붙음",

		// lint 위반 문구. 골든 스냅샷이 바이트로 비교한다.
		"lint.violation.frontmatter_missing.message":  "프론트매터가 없습니다",
		"lint.violation.frontmatter_missing.fix":      "문서 첫 줄에 --- 로 여는 구분자를 두고 필드를 채운 뒤 --- 로 닫으세요",
		"lint.violation.frontmatter_unclosed.message": "프론트매터가 닫는 --- 구분자 없이 끝났습니다",
		"lint.violation.frontmatter_unclosed.fix":     "프론트매터 끝에 --- 줄을 추가하세요",
		"lint.violation.frontmatter_yaml.message":     "프론트매터 YAML 파싱 실패: %s",
		"lint.violation.frontmatter_yaml.fix":         "프론트매터의 YAML 문법을 고치세요",
		"lint.violation.stage_missing.message":        "artifact_stage 필드가 없습니다",
		"lint.violation.stage_missing.fix":            "프론트매터에 artifact_stage 필드를 채우세요",
		"lint.violation.required_missing.message":     "단계 %s의 필수 필드 %s가 없습니다",
		"lint.violation.required_missing.fix":         "프론트매터에 %s 필드를 추가하세요",
		"lint.violation.allowed_value.message":        "%s 값이 허용값 밖입니다: %q (허용값: %s)",
		"lint.violation.allowed_value.fix":            "%s 값을 허용값 중 하나로 바꿉니다",
		"lint.violation.stage_agreement.message":      "문서가 %s 디렉토리에 있지만 artifact_stage가 %q입니다",
		"lint.violation.stage_agreement.fix":          "문서를 artifact_stage에 맞는 디렉토리로 옮기거나 artifact_stage를 %s로 고치세요. 문서를 옮길 때는 engram promote, demote, archive를 쓰세요",
		"lint.violation.axis_off.message":             "설정에서 꺼진 속성이 문서에 있습니다: %s (프리셋 %s)",
		"lint.violation.axis_off.fix":                 "engram.yaml의 axes에서 %s를 켜거나 문서에서 %s 필드를 지웁니다",
		"lint.violation.forms.message":                "form 값이 forms 폐쇄 집합에 없습니다: %q (허용값: %s)",
		"lint.violation.forms.fix":                    "form 값을 허용값 중 하나로 바꿉니다",
		"lint.violation.topics.message":               "topics 값이 설정에 정의되지 않았습니다: %q (topics는 개방 집합입니다)",
		"lint.violation.topics.fix":                   "engram.yaml의 topics 목록에 %q를 추가하세요",
		"lint.violation.sources_updated.message":      "sources 계층 문서에 updated 필드가 있습니다",
		"lint.violation.sources_updated.fix":          "updated 필드를 지우세요. sources는 원본 보존 계층이라 갱신하지 않습니다",
		"lint.violation.max_lines.message":            "문서가 %d줄로 max_lines %d줄을 넘습니다",
		"lint.violation.max_lines.fix":                "문서를 나누세요. 상한은 engram.yaml의 max_lines로 조정하세요",
		"lint.violation.link_broken.message":          "깨진 위키링크: [[%s]]에 해당하는 문서가 없습니다",
		"lint.violation.link_broken.fix":              "슬러그를 고치거나 [[%s]] 문서를 만드세요",
		"lint.violation.orphan.message":               "들어오는 관계와 나가는 관계가 모두 없습니다",
		"lint.violation.orphan.fix":                   "다른 문서의 related나 본문에서 [[%s]]로 연결하거나 관계 필드로 잇으세요",
		"lint.violation.gate_deferred.message":        "링크 가능한 대상 문서가 %d개로 min_wikilinks %d개보다 적어 게이트를 유예합니다. 대상 문서가 %d개가 되면 게이트가 동작합니다",
		"lint.violation.gate_deferred.fix":            "연결할 문서를 만들어 대상을 늘리세요. 기준은 engram.yaml의 min_wikilinks로 조정하세요",
		"lint.violation.gate_reject.message":          "위키링크가 %d개로 min_wikilinks %d개에 못 미칩니다",
		"lint.violation.gate_reject.fix":              "related 필드나 본문에 위키링크를 %d개 더 추가하세요",
		"lint.wiki.broad_topic.fix":                   "주제를 더 세분하세요. 기준은 engram.yaml의 broad_topic_pct로 조정하세요",

		// doctor 항목 문구.
		"doctor.env.git.unavailable":                     "git을 실행할 수 없습니다",
		"doctor.env.git.unavailable_fix":                 "engram sync만 git을 요구하고 나머지 커맨드는 git 없이 동작하므로 당장은 문제가 아닙니다. 날짜 필드를 git 이력에서 정정하려면 설치하세요. macOS는 xcode-select --install, Windows는 Git for Windows 설치",
		"doctor.env.git_autocrlf.skip_no_git":            "git이 없어 확인할 수 없습니다",
		"doctor.env.git_autocrlf.skip_not_repo":          "대상 경로가 git 저장소가 아닙니다",
		"doctor.env.git_autocrlf.read_fail":              "core.autocrlf 값을 읽을 수 없습니다",
		"doctor.env.git_autocrlf.read_fail_fix":          "git config core.autocrlf input으로 직접 확인하세요",
		"doctor.env.git_autocrlf.no_value":               "설정 없음",
		"doctor.env.git_autocrlf.true_detail":            "core.autocrlf가 true입니다. 줄바꿈이 자동 변환되어 프론트매터와 골든 비교가 틀어집니다",
		"doctor.env.fs_case.fail_mkdir":                  "임시 디렉토리를 만들 수 없어 확인에 실패했습니다",
		"doctor.env.fs_case.fail_mkdir_fix":              "TMPDIR 환경변수와 임시 디렉토리 권한을 확인하세요",
		"doctor.env.fs_case.fail_write":                  "임시 파일을 만들 수 없어 확인에 실패했습니다",
		"doctor.env.fs_case.fail_write_fix":              "임시 디렉토리 쓰기 권한을 확인하세요",
		"doctor.env.fs_case.ignore_case":                 "파일시스템이 대소문자를 무시합니다. 대소문자만 다른 슬러그가 서로 겹칩니다",
		"doctor.env.fs_case.ignore_case_fix":             "슬러그에 대소문자만 다른 이름을 쓰지 마세요",
		"doctor.env.fs_case.ok":                          "파일시스템이 대소문자를 구분합니다",
		"doctor.env.console_encoding.ok_not_windows":     "Windows가 아니므로 콘솔은 UTF-8을 씁니다",
		"doctor.env.console_encoding.ok_not_console":     "stdout이 콘솔이 아닙니다. UTF-8 바이트를 그대로 냅니다. 받는 쪽이 UTF-8로 해석해야 합니다",
		"doctor.env.console_encoding.ok_not_console_fix": "PowerShell 이라면 [Console]::OutputEncoding = [Text.Encoding]::UTF8를 먼저 실행하세요",
		"doctor.env.console_encoding.ok_switched":        "출력 코드페이지를 65001 (UTF-8) 로 전환했습니다",
		"doctor.env.console_encoding.warn":               "출력 코드페이지가 %d 입니다. 콘솔 인코딩 전환에 실패했습니다",
		"doctor.env.console_encoding.warn_fix":           "콘솔에서 chcp 65001을 실행하세요",
		"doctor.env.write_perm.skip_no_path":             "대상 경로가 없습니다",
		"doctor.env.write_perm.skip_not_dir":             "대상 경로가 디렉토리가 아닙니다",
		"doctor.env.write_perm.fail":                     "위키 루트에 쓸 수 없습니다: %v",
		"doctor.env.write_perm.fail_fix":                 "chmod u+w %s 또는 디렉토리 소유자를 확인하세요",
		"doctor.env.write_perm.ok":                       "위키 루트에 쓸 수 있습니다",
		"doctor.wiki.config.skip":                        "engram.yaml이 없어 위키가 아닙니다. 환경 점검만 진행합니다",
		"doctor.wiki.config.fail":                        "engram.yaml을 파싱할 수 없습니다: %v",
		"doctor.wiki.config.fail_fix":                    "engram.yaml의 YAML 문법을 고치세요",
		"doctor.wiki.config.ok":                          "engram.yaml을 읽었습니다",
		"doctor.wiki.skip":                               "engram.yaml이 없어 위키가 아닙니다",
		"doctor.wiki.unknown_keys.warn":                  "알 수 없는 키: %s",
		"doctor.wiki.unknown_keys.warn_fix":              "이 키들을 지우거나 맞는 이름으로 고치세요. 지원 키는 engram config list --origin에서 봅니다",
		"doctor.wiki.unknown_keys.ok":                    "알 수 없는 키가 없습니다",
		"doctor.wiki.min_wikilinks.warn":                 "min_wikilinks가 0 이라 승급 게이트가 꺼져 있습니다",
		"doctor.wiki.min_wikilinks.warn_fix":             "engram.yaml에 min_wikilinks: 2를 지정하세요",
		"doctor.wiki.page_dirs.fail":                     "없는 디렉토리: %s",
		"doctor.wiki.page_dirs.ok":                       "page_dirs %d개가 모두 있습니다",
		"doctor.wiki.root_files.fail":                    "없는 루트 파일: %s",
		"doctor.wiki.root_files.fail_fix":                "%s 파일을 위키 루트에 만드세요",
		"doctor.wiki.root_files.ok":                      "root_files %d개가 모두 있습니다",
		"doctor.wiki.bridge_rejections.read_fail":        "%s을 읽을 수 없습니다: %v",
		"doctor.wiki.bridge_rejections.read_fail_fix":    "%s의 YAML 형식을 고치세요. 손으로 고치기 어려우면 파일을 지우면 기각이 전부 사라집니다",
		"doctor.wiki.bridge_rejections.ok_none":          "기각된 쌍이 없습니다",
		"doctor.wiki.bridge_rejections.walk_fail":        "위키를 순회할 수 없어 확인하지 못했습니다",
		"doctor.wiki.bridge_rejections.dangling_item":    "%s %s (%s 없음)",
		"doctor.wiki.bridge_rejections.dangling":         "실재하지 않는 슬러그를 가리키는 기각 쌍 %d건: %s",
		"doctor.wiki.bridge_rejections.dangling_fix":     "engram bridge --unreject <A> <B>로 지우거나 engram mv로 슬러그를 맞추세요",
		"doctor.wiki.bridge_rejections.ok":               "기각 쌍 %d건이 모두 실재하는 문서를 가리킵니다",
		"doctor.wiki.engram_gitignore.skip_no_git":       "git이 없어 확인할 수 없습니다",
		"doctor.wiki.engram_gitignore.skip_not_repo":     "위키가 git 저장소가 아닙니다",
		"doctor.wiki.engram_gitignore.ok":                ".engram이 gitignore 됩니다",
		"doctor.wiki.engram_gitignore.warn":              ".engram 캐시 디렉토리가 gitignore 되지 않았습니다",
		"doctor.wiki.engram_gitignore.warn_fix":          "위키 루트 .gitignore에 .engram/ 줄을 추가하세요",

		// eject 산출물 문구. 내보내는 시점 언어로 굳는다.
		"eject.excluded_note": `이 문서와 린터가 내보내지 않는 것:

- wiki.broad-topic. 위키 전체의 통계로 판정하는 진단이라 문서 하나를 보는
  훅의 자리가 아니다. engram lint 가 계속 판정한다.
- 검색 색인, 재발견, 링크 그래프 계산, 다이제스트. 연산은 파일로 표현되지
  않는다. engram search, recall, resurface, bridge, digest, backlinks 가
  계속 수행한다.`,

		"eject.schema_doc.heading":          "# 프론트매터 스키마\n\n",
		"eject.schema_doc.intro":            "이 위키는 %s 프리셋을 쓴다. 프리셋은 속성의 시작점이고 engram.yaml 의 axes 로 개별 속성을 켜고 끌 수 있다.\n\n",
		"eject.schema_doc.on_heading":       "## 켜진 속성\n\n",
		"eject.schema_doc.off_heading":      "\n## 꺼진 속성\n\n꺼진 속성을 문서에 두면 위반이다.\n\n",
		"eject.schema_doc.required_heading": "\n## 단계별 필수 필드\n\n",
		"eject.schema_doc.dates": `
## 날짜 필드

- created. 원본이 처음 기록된 날. 사람이나 인테이크가 채운다. 연월까지만
  알면 YYYY-MM 을 허용한다.
- sourced_at. 이 위키에 편입된 날. 도구가 채운다.
- updated. 마지막으로 내용이 갱신된 날. 도구가 채운다. 손으로 쓰지 않는다.
- ` + "`sources/`" + ` 계층 문서에는 updated 를 두지 않는다. 원본 보존 계층이라
  갱신되지 않으며 신선도를 오해하게 만든다.

`,

		"eject.values_doc.heading":        "# 값 집합\n\n",
		"eject.values_doc.intro":          "닫힌 집합(집합 밖 값은 위반)과 열린 집합(집합 밖 값은 경고)을 구분한다.\n\n",
		"eject.values_doc.closed_heading": "## 닫힌 집합\n\n",
		"eject.values_doc.open_heading":   "\n## 열린 집합\n\n- topics: %s\n",
		"eject.values_doc.open_note":      "\ntopics 에 정의되지 않은 값을 쓰면 경고다. 값 자체는 허용된다.\n\n",

		"eject.promotion_doc.heading":          "# 승급 게이트와 위치 규칙\n\n",
		"eject.promotion_doc.intro":            "min_wikilinks 는 %d다. context 디렉토리 아래 문서의 고유 위키링크 수가 이 값에 못 미치면 게이트가 문서를 거절한다. 0이면 게이트가 꺼진다. 게이트는 선언이 아니라 문서가 놓인 디렉토리로 발동한다.\n\n",
		"eject.promotion_doc.sole_reason":      "거절 사유는 gate.min-wikilinks 하나뿐이다. 게이트를 통과하려면 관련 문서에 위키링크로 연결되어 있어야 한다.\n\n",
		"eject.promotion_doc.deferred":         "링크 가능한 대상은 문서 수가 충분해야 센다. 대상 문서가 min_wikilinks 보다 적으면 게이트를 유예하고 경고만 낸다. 위키가 자라면 게이트가 다시 동작한다. inbox 단계 문서는 대상에서 뺀다. promote 되면 슬러그가 바뀌어 링크가 깨지기 때문이다.\n\n",
		"eject.promotion_doc.location_heading": "## 위치와 단계의 일치\n\n문서가 놓인 최상위 디렉토리와 artifact_stage 값이 일치해야 한다.\n\n",
		"eject.promotion_doc.location_row":     "- %s 단계 문서는 %s/ 디렉토리에 둔다\n",
		"eject.promotion_doc.context_note":     "\ncontext 를 선언했는데 context 디렉토리 밖에 있으면 오류이다. 검수된 지식의 필드 집합을 갖추고 색인 자격을 주장하며 파이프라인을 우회하기 때문이다. 그 밖의 불일치는 경고다.\n\n",

		"eject.lint_rules_doc.heading":        "# lint 규칙\n\n규칙 ID 는 점 표기 소문자다. 등급의 의미는 아래와 같다.\n\n",
		"eject.lint_rules_doc.severities":     "- error: 승급을 막는다\n- warn: 통과시키되 알린다\n- reject: 승급 게이트 거절\n\n",
		"eject.lint_rules_doc.table_heading":  "## 규칙 목록\n\n",
		"eject.lint_rules_doc.table_header":   "| 규칙 ID | 등급 | 판정 |\n|---|---|---|\n",
		"eject.lint_rules_doc.script_heading": "\n## scripts/lint-frontmatter.py 가 판정하는 규칙\n\n",
		"eject.lint_rules_doc.script_note":    "문서 단위 규칙과 링크 무결성, 고아 판정, 승급 게이트를 판정한다. 위 표에서 wiki.broad-topic 만 빠진다. 위키 전체의 통계로 판정하는 진단이라 문서 하나를 보는 훅의 자리가 아니다.\n\n",

		"eject.layout_doc.heading":       "# 위키 배치\n\n",
		"eject.layout_doc.stage_heading": "## 단계 디렉토리\n\n",
		"eject.layout_doc.stage_row":     "- %s/ (%s 단계)\n",
		"eject.layout_doc.root_files":    "\n## 루트 파일\n\nroot_files 로 정의한다: %s\n",
		"eject.layout_doc.root_note":     "\n루트 파일은 색인이다. 승급 게이트와 고아 판정에서 빠지고 스키마 검사는 그대로 받는다.\n\n",
		"eject.layout_doc.ignore_files":  "## 문서가 아닌 파일\n\nignore_files 로 정의한다: %s\n",
		"eject.layout_doc.ignore_note":   "\n같은 파일명이면 깊이와 무관하게 문서에서 뺀다. 디렉토리를 설명하는 README 같은 파일이 해당한다.\n\n",

		"eject.hook.comment":   "# 커밋 전 문서 규칙 검사. engram eject 가 만들었다.\n# 활성화: git config core.hooksPath .githooks\n",
		"eject.hook.blocked":   "문서 규칙 위반으로 커밋을 막았습니다. 위반 목록을 고친 뒤 다시 커밋하세요",
		"eject.hook.no_python": "python3 가 없다면 scripts/lint-frontmatter.py 상단의 안내를 확인하세요",

		"eject.agents_doc.heading":         "# AGENTS.md\n\n이 위키에서 작업하는 에이전트는 아래를 따른다.\n\n",
		"eject.agents_doc.rules_heading":   "## 규칙은 이 위키의 것이다\n\n",
		"eject.agents_doc.rules_body":      "문서 규칙의 소유권은 이 위키에 있다. 규칙 명세는 meta/ 디렉토리의 문서에 있고 판정은 scripts/lint-frontmatter.py 가 한다. 규칙을 바꾸려면 그 둘을 직접 고친다. 속성, 허용값, 임계값, 디렉토리는 engram.yaml 이 진실원이므로 스크립트는 그 파일을 실행 시점에 읽는다.\n\n",
		"eject.agents_doc.hook_body":       "커밋 전 검사는 .githooks/pre-commit 이 돌린다. 한 번 활성화한다.\n\n    git config core.hooksPath .githooks\n\n",
		"eject.agents_doc.compute_heading": "## 연산은 engram 이 계속 수행한다\n\n",
		"eject.agents_doc.compute_body":    "eject 는 규칙만 내보냈다. 검색 색인, 재발견, 링크 그래프 계산, 다이제스트는 파일로 표현되지 않는 연산이므로 engram 이 계속 맡는다. 아래 커맨드는 그대로 동작한다.\n\n",
		"eject.agents_doc.lint_body":       "engram lint 를 돌리면 내장 규칙으로 검사한다. 위키 단위 진단 wiki.broad-topic 은 스크립트가 내보내지 않으므로 engram lint 만 판정한다.\n\n",
		"eject.agents_doc.dirs_heading":    "## 디렉토리\n\n",
		"eject.agents_doc.dirs_row":        "- %s/. %s 단계 문서가 사는 곳\n",
		"eject.agents_doc.root_files":      "- 루트 파일(root_files): %s\n",
		"eject.agents_doc.ignore_files":    "- 문서가 아닌 파일(ignore_files): %s\n",
		"eject.agents_doc.preset_heading":  "\n## 프리셋\n\n",
		"eject.agents_doc.preset_body":     "이 위키는 %s 프리셋을 쓴다.\n",
		"eject.agents_doc.oneway_heading":  "\n## 단방향\n\n",
		"eject.agents_doc.oneway_body":     "eject 는 되돌리는 커맨드가 없다. 규칙 파일을 고쳤다면 engram 이 다시 만들면 덮어 쓴다. 되돌리고 싶은 부분은 git 이력으로 본다.\n",

		"eject.gitattributes.comment": "# 줄바꿈을 LF 로 통일한다. 린터는 CRLF 도 인식하지만 저장소는\n# 단일 형태를 유지하는 것이 비교와 병합에 좋다.\n",

		"eject.linter.docstring": `문서 단위 규칙을 판정하는 린터. engram eject 가 만들었다.

속성, 허용값, 임계값, 디렉토리는 engram.yaml 을 실행 시점에 읽는다.
설정을 바꾸면 이 스크립트를 다시 만들지 않아도 판정이 따라간다.
engram.yaml 이 고칠 수 없는 값은 아래 상수로 받았다. 고정 허용 집합과
단계와 디렉토리의 대응이 그것이다.

이 린터가 내보내지 않는 것:
- wiki.broad-topic. 위키 전체의 통계로 판정하는 진단이라 문서 하나를
  보는 훅의 자리가 아니다. engram lint 가 계속 판정한다.
- 검색 색인, 재발견, 링크 그래프 계산, 다이제스트. 연산은 파일로
  표현되지 않는다. engram search, recall, resurface, bridge, digest,
  backlinks 가 계속 수행한다.

이 위키는 eject 시점에 %s 프리셋을 썼다.

사용법: python3 scripts/lint-frontmatter.py [위키루트]
종료 코드는 engram lint 와 같다. error 나 reject 가 있으면 1, 그 밖에는
0 이다. 경고는 종료 코드에 영향을 주지 않는다. 이 스크립트를 부르는
커밋 훅이 경고만 있는 위키를 막으면 안 되기 때문이다.
`,
		"eject.linter.console_comment": `# Windows 콘솔의 기본 인코딩은 UTF-8 이 아니라 cp949 나 cp1252 다. 한글
# 메시지를 그대로 내면 UnicodeEncodeError 로 죽는다. engram 본체는 콘솔
# 코드페이지를 UTF-8 로 바꿔서 푸는데, 내보낸 이 스크립트는 그 처리를
# 받지 못하므로 여기서 스트림을 다시 연다.
`,
		"eject.linter.newline_comment": `# newline 도 함께 고정한다. Windows 의 텍스트 모드는 \n 을 \r\n 으로
# 바꿔 내보내는데, engram lint 는 \n 만 내므로 그대로 두면 두 린터의
# 출력이 줄바꿈에서 갈린다. 판정이 같아야 한다는 계약이 깨진다.
`,
		"eject.linter.constants_comment":   "# --- 고정 상수. engram.yaml 이 바꿀 수 없는 값이다. ---\n",
		"eject.linter.stage_dirs_comment":  "# 단계와 디렉토리의 대응.\n",
		"eject.linter.preset_axes_comment": "# 프리셋별 속성 기본값. minimal 이 personal 에, personal 이 team 에\n# 포함된다. engram.yaml 의 axes 가 개별 속성을 덮어 쓴다.\n",
		"eject.linter.defaults_comment":    "# engram.yaml 에 키가 없을 때의 기본값. eject 시점의 설정에서 왔다.\n",
		"eject.linter.backtick_comment":    "# 백틱 문자. 코드 펜스와 인라인 코드 판정에 쓴다.\n",
		"eject.linter.config_section":      "# --- 설정 읽기 ---\n",
		"eject.linter.yaml_docstring": `프론트매터와 engram.yaml 이 쓰는 YAML 부분집합을 파싱한다.

    키: 값, 흐름 목록 [a, b], 블록 목록, 한 단계 맵을 다룬다.
    PyYAML 을 쓰지 않는다. 표준 라이브러리만 쓴다는 eject 의 계약이다.
    `,
		"eject.linter.empty_key_comment":   "                # 값이 없는 키는 빈 값으로 둔다. 키의 존재 자체가 판정\n                # 대상이다(source_channel: 처럼).\n",
		"eject.linter.parse_section":       "# --- 문서 파싱 ---\n",
		"eject.linter.walk_section":        "# --- 순회 ---\n",
		"eject.linter.judge_section":       "# --- 판정 ---\n",
		"eject.linter.int_stage_comment":   "        # YAML 이 숫자로 읽는 값도 engram 은 문자열로 다룬다.\n",
		"eject.linter.stage_input_comment": "            # artifact_stage 는 단계 판정의 입력이다. 없으면 그 자체가\n            # 오류이고 어느 단계인지 모르므로 다른 필수 필드는 보고하지\n            # 않는다(ADR 0040).\n",
		"eject.linter.related_comment":     "        # related 필드의 링크. 파싱된 값을 진실원으로 쓰고 줄 번호만 원문에서 잡는다.\n",
		"eject.linter.gate_dir_comment":    "        # 게이트는 문서가 놓인 디렉토리로 발동한다(ADR 0040). 선언을 보면\n        # 값을 비우거나 낮춰 우회할 수 있다.\n",
		"eject.linter.gate_line_comment":   "            # 줄 번호는 related 키가 있는 줄로 잡는다. engram lint 와 같다.\n",
		"eject.linter.exit_code_comment":   "    # 종료 코드의 규칙은 engram lint 의 HasBlocking 과 같다. error 나\n    # reject 가 있어야 1 이다. 경고만으로 커밋을 막지 않는다.\n",

		// 아래는 Python 이 실행 시점에 %s, %d 로 채운다.
		"eject.linter.unclosed.message":         "프론트매터가 닫는 --- 구분자 없이 끝났습니다",
		"eject.linter.unclosed.fix":             "프론트매터 끝에 --- 줄을 추가하세요",
		"eject.linter.missing.message":          "프론트매터가 없습니다",
		"eject.linter.missing.fix":              "문서 첫 줄에 --- 로 여는 구분자를 두고 필드를 채운 뒤 --- 로 닫으세요",
		"eject.linter.stage_missing.message":    "artifact_stage 필드가 없습니다",
		"eject.linter.stage_missing.fix":        "프론트매터에 artifact_stage 필드를 채우세요",
		"eject.linter.required_missing.message": "단계 %s의 필수 필드 %s가 없습니다",
		"eject.linter.required_missing.fix":     "프론트매터에 %s 필드를 추가하세요",
		"eject.linter.allowed_value.message":    `%s 값이 허용값 밖입니다: "%s" (허용값: %s)`,
		"eject.linter.allowed_value.fix":        "%s 값을 허용값 중 하나로 바꿉니다",
		"eject.linter.stage_agreement.message":  `문서가 %s 디렉토리에 있지만 artifact_stage가 "%s"입니다`,
		"eject.linter.stage_agreement.fix":      "문서를 artifact_stage에 맞는 디렉토리로 옮기거나 artifact_stage를 %s로 고치세요. 문서를 옮길 때는 engram promote, demote, archive를 쓰세요",
		"eject.linter.axis_off.message":         "설정에서 꺼진 속성이 문서에 있습니다: %s (프리셋 %s)",
		"eject.linter.axis_off.fix":             "engram.yaml의 axes에서 %s를 켜거나 문서에서 %s 필드를 지웁니다",
		"eject.linter.forms.message":            `form 값이 forms 폐쇄 집합에 없습니다: "%s" (허용값: %s)`,
		"eject.linter.forms.fix":                "form 값을 허용값 중 하나로 바꿉니다",
		"eject.linter.topics.message":           `topics 값이 설정에 정의되지 않았습니다: "%s" (topics는 개방 집합입니다)`,
		"eject.linter.topics.fix":               `engram.yaml의 topics 목록에 "%s"를 추가하세요`,
		"eject.linter.sources_updated.message":  "sources 계층 문서에 updated 필드가 있습니다",
		"eject.linter.sources_updated.fix":      "updated 필드를 지우세요. sources는 원본 보존 계층이라 갱신하지 않습니다",
		"eject.linter.max_lines.message":        "문서가 %d줄로 max_lines %d줄을 넘습니다",
		"eject.linter.max_lines.fix":            "문서를 나누세요. 상한은 engram.yaml의 max_lines로 조정하세요",
		"eject.linter.link_broken.message":      "깨진 위키링크: [[%s]]에 해당하는 문서가 없습니다",
		"eject.linter.link_broken.fix":          "슬러그를 고치거나 [[%s]] 문서를 만드세요",
		"eject.linter.orphan.message":           "들어오는 관계와 나가는 관계가 모두 없습니다",
		"eject.linter.orphan.fix":               "다른 문서의 related나 본문에서 [[%s]]로 연결하거나 관계 필드로 잇으세요",
		"eject.linter.gate_deferred.message":    "링크 가능한 대상 문서가 %d개로 min_wikilinks %d개보다 적어 게이트를 유예합니다. 대상 문서가 %d개가 되면 게이트가 동작합니다",
		"eject.linter.gate_deferred.fix":        "연결할 문서를 만들어 대상을 늘리세요. 기준은 engram.yaml의 min_wikilinks로 조정하세요",
		"eject.linter.gate_reject.message":      "위키링크가 %d개로 min_wikilinks %d개에 못 미칩니다",
		"eject.linter.gate_reject.fix":          "related 필드나 본문에 위키링크를 %d개 더 추가하세요",
		"eject.linter.print_fix":                "    고치는 법: %s",
		"eject.linter.summary":                  "검사한 파일 %d개, error %d, warn %d, reject %d",
	})

	Register(LangEN, map[string]string{
		// lint rule descriptions, shown by rules show and meta/lint-rules.md.
		"lint.rule.gate_min_wikilinks":        "Unique wikilinks in a document under the context directory fall below min_wikilinks. The gate's only rejection reason (ADR 0040)",
		"lint.rule.frontmatter_missing":       "Document with no frontmatter block at all",
		"lint.rule.frontmatter_unclosed":      "Frontmatter that ends without a closing ---",
		"lint.rule.frontmatter_yaml":          "Frontmatter YAML syntax error",
		"lint.rule.frontmatter_missing_field": "Missing stage-required field",
		"lint.rule.schema_allowed_value":      "Field value outside the allowed set: artifact_stage, status, scope, sensitivity, trigger_mode",
		"lint.rule.schema_axis_off":           "Document carries an axis the config turned off",
		"lint.rule.location_stage_agreement":  "Mismatch between the document's directory and its artifact_stage value. Declaring context while outside context/ is an error; other mismatches are warns (ADR 0035)",
		"lint.rule.taxonomy_forms":            "form value not in the closed forms set",
		"lint.rule.sources_updated":           "updated field that breaks source preservation in a sources document",
		"lint.rule.taxonomy_topics":           "topics value not defined in config. Warned because the set is open",
		"lint.rule.body_max_lines":            "Document exceeds max_lines",
		"lint.rule.link_broken":               "Wikilink points to a document that is not in the wiki",
		"lint.rule.graph_orphan":              "Document with neither incoming nor outgoing relations",
		"lint.rule.gate_deferred":             "Gate deferred because there are too few linkable target documents",
		"lint.rule.wiki_broad_topic":          "One topic is attached to more than broad_topic_pct of all documents",

		// lint violation texts.
		"lint.violation.frontmatter_missing.message":  "No frontmatter",
		"lint.violation.frontmatter_missing.fix":      "Open with --- on the first line, fill in the fields, and close with ---",
		"lint.violation.frontmatter_unclosed.message": "Frontmatter ends without a closing --- delimiter",
		"lint.violation.frontmatter_unclosed.fix":     "Add a --- line at the end of the frontmatter",
		"lint.violation.frontmatter_yaml.message":     "Frontmatter YAML parse failure: %s",
		"lint.violation.frontmatter_yaml.fix":         "Fix the YAML syntax in the frontmatter",
		"lint.violation.stage_missing.message":        "No artifact_stage field",
		"lint.violation.stage_missing.fix":            "Fill in the artifact_stage field in the frontmatter",
		"lint.violation.required_missing.message":     "Stage %s requires field %s, which is missing",
		"lint.violation.required_missing.fix":         "Add the %s field to the frontmatter",
		"lint.violation.allowed_value.message":        "%s value is outside the allowed set: %q (allowed: %s)",
		"lint.violation.allowed_value.fix":            "Change the %s value to one of the allowed values",
		"lint.violation.stage_agreement.message":      "Document is in the %s directory but artifact_stage is %q",
		"lint.violation.stage_agreement.fix":          "Move the document to the directory matching its artifact_stage, or change artifact_stage to %s. Use engram promote, demote, or archive to move documents",
		"lint.violation.axis_off.message":             "The document carries an axis the config turned off: %s (preset %s)",
		"lint.violation.axis_off.fix":                 "Turn on %s in the axes of engram.yaml, or remove the %s field from the document",
		"lint.violation.forms.message":                "form value is not in the closed forms set: %q (allowed: %s)",
		"lint.violation.forms.fix":                    "Change the form value to one of the allowed values",
		"lint.violation.topics.message":               "topics value is not defined in config: %q (topics is an open set)",
		"lint.violation.topics.fix":                   "Add %q to the topics list in engram.yaml",
		"lint.violation.sources_updated.message":      "A sources-tier document has an updated field",
		"lint.violation.sources_updated.fix":          "Remove the updated field. sources is a preservation tier and is never updated",
		"lint.violation.max_lines.message":            "Document is %d lines, exceeding the max_lines of %d",
		"lint.violation.max_lines.fix":                "Split the document. Adjust the limit with max_lines in engram.yaml",
		"lint.violation.link_broken.message":          "Broken wikilink: no document matches [[%s]]",
		"lint.violation.link_broken.fix":              "Fix the slug or create the [[%s]] document",
		"lint.violation.orphan.message":               "No incoming or outgoing relations",
		"lint.violation.orphan.fix":                   "Link to it from another document's related or body with [[%s]], or connect via relation fields",
		"lint.violation.gate_deferred.message":        "Only %d linkable target documents versus min_wikilinks of %d, so the gate is deferred. The gate activates once there are %d target documents",
		"lint.violation.gate_deferred.fix":            "Create documents to link to. Adjust the threshold with min_wikilinks in engram.yaml",
		"lint.violation.gate_reject.message":          "%d wikilinks fall short of min_wikilinks %d",
		"lint.violation.gate_reject.fix":              "Add %d more wikilinks in the related field or the body",
		"lint.wiki.broad_topic.fix":                   "Split the topic further. Adjust the threshold with broad_topic_pct in engram.yaml",

		// doctor finding texts.
		"doctor.env.git.unavailable":                     "Cannot run git",
		"doctor.env.git.unavailable_fix":                 "Only engram sync requires git; the other commands run without it, so this is not an immediate problem. Install git to correct date fields from history. macOS: xcode-select --install, Windows: Git for Windows",
		"doctor.env.git_autocrlf.skip_no_git":            "Cannot check without git",
		"doctor.env.git_autocrlf.skip_not_repo":          "The target path is not a git repository",
		"doctor.env.git_autocrlf.read_fail":              "Cannot read the core.autocrlf value",
		"doctor.env.git_autocrlf.read_fail_fix":          "Check it yourself with git config core.autocrlf input",
		"doctor.env.git_autocrlf.no_value":               "not set",
		"doctor.env.git_autocrlf.true_detail":            "core.autocrlf is true. Automatic line-ending conversion breaks frontmatter and golden comparisons",
		"doctor.env.fs_case.fail_mkdir":                  "Cannot create a temporary directory to run the check",
		"doctor.env.fs_case.fail_mkdir_fix":              "Check the TMPDIR environment variable and temp directory permissions",
		"doctor.env.fs_case.fail_write":                  "Cannot create a temporary file to run the check",
		"doctor.env.fs_case.fail_write_fix":              "Check write permission on the temp directory",
		"doctor.env.fs_case.ignore_case":                 "The filesystem ignores case. Slugs differing only in case collide",
		"doctor.env.fs_case.ignore_case_fix":             "Do not use names that differ only in case",
		"doctor.env.fs_case.ok":                          "The filesystem distinguishes case",
		"doctor.env.console_encoding.ok_not_windows":     "Not Windows, so the console uses UTF-8",
		"doctor.env.console_encoding.ok_not_console":     "stdout is not a console. Raw UTF-8 bytes are emitted; the receiving side must decode them as UTF-8",
		"doctor.env.console_encoding.ok_not_console_fix": "In PowerShell, run [Console]::OutputEncoding = [Text.Encoding]::UTF8 first",
		"doctor.env.console_encoding.ok_switched":        "Output code page switched to 65001 (UTF-8)",
		"doctor.env.console_encoding.warn":               "Output code page is %d. Console encoding switch failed",
		"doctor.env.console_encoding.warn_fix":           "Run chcp 65001 in the console",
		"doctor.env.write_perm.skip_no_path":             "The target path does not exist",
		"doctor.env.write_perm.skip_not_dir":             "The target path is not a directory",
		"doctor.env.write_perm.fail":                     "Cannot write to the wiki root: %v",
		"doctor.env.write_perm.fail_fix":                 "Run chmod u+w %s or check the directory owner",
		"doctor.env.write_perm.ok":                       "The wiki root is writable",
		"doctor.wiki.config.skip":                        "No engram.yaml, so this is not a wiki. Environment checks only",
		"doctor.wiki.config.fail":                        "Cannot parse engram.yaml: %v",
		"doctor.wiki.config.fail_fix":                    "Fix the YAML syntax in engram.yaml",
		"doctor.wiki.config.ok":                          "engram.yaml loaded",
		"doctor.wiki.skip":                               "No engram.yaml, so this is not a wiki",
		"doctor.wiki.unknown_keys.warn":                  "Unknown keys: %s",
		"doctor.wiki.unknown_keys.warn_fix":              "Remove these keys or fix their names. See engram config list --origin for supported keys",
		"doctor.wiki.unknown_keys.ok":                    "No unknown keys",
		"doctor.wiki.min_wikilinks.warn":                 "min_wikilinks is 0, so the promotion gate is off",
		"doctor.wiki.min_wikilinks.warn_fix":             "Set min_wikilinks: 2 in engram.yaml",
		"doctor.wiki.page_dirs.fail":                     "Missing directories: %s",
		"doctor.wiki.page_dirs.ok":                       "All %d page_dirs entries exist",
		"doctor.wiki.root_files.fail":                    "Missing root files: %s",
		"doctor.wiki.root_files.fail_fix":                "Create the %s files in the wiki root",
		"doctor.wiki.root_files.ok":                      "All %d root_files entries exist",
		"doctor.wiki.bridge_rejections.read_fail":        "Cannot read %s: %v",
		"doctor.wiki.bridge_rejections.read_fail_fix":    "Fix the YAML format of %s. If editing by hand is hard, deleting the file clears all rejections",
		"doctor.wiki.bridge_rejections.ok_none":          "No rejected pairs",
		"doctor.wiki.bridge_rejections.walk_fail":        "Could not walk the wiki to verify",
		"doctor.wiki.bridge_rejections.dangling_item":    "%s %s (no %s)",
		"doctor.wiki.bridge_rejections.dangling":         "%d rejected pairs point at missing slugs: %s",
		"doctor.wiki.bridge_rejections.dangling_fix":     "Remove with engram bridge --unreject <A> <B> or fix the slug with engram mv",
		"doctor.wiki.bridge_rejections.ok":               "All %d rejected pairs point at existing documents",
		"doctor.wiki.engram_gitignore.skip_no_git":       "Cannot check without git",
		"doctor.wiki.engram_gitignore.skip_not_repo":     "The wiki is not a git repository",
		"doctor.wiki.engram_gitignore.ok":                ".engram is gitignored",
		"doctor.wiki.engram_gitignore.warn":              "The .engram cache directory is not gitignored",
		"doctor.wiki.engram_gitignore.warn_fix":          "Add a .engram/ line to .gitignore in the wiki root",

		// eject artifact texts. Frozen to the language at eject time.
		"eject.excluded_note": `What this document and the linter do not export:

- wiki.broad-topic. A diagnosis judged from whole-wiki statistics, not a
  per-document hook. engram lint keeps judging it.
- Search index, resurfacing, link graph computation, digests. These are
  computations, not files. engram search, recall, resurface, bridge, digest,
  backlinks keep performing them.`,

		"eject.schema_doc.heading":          "# Frontmatter schema\n\n",
		"eject.schema_doc.intro":            "This wiki uses the %s preset. The preset is the starting point for axes; engram.yaml's axes can turn each one on or off.\n\n",
		"eject.schema_doc.on_heading":       "## Enabled axes\n\n",
		"eject.schema_doc.off_heading":      "\n## Disabled axes\n\nA document carrying a disabled axis is a violation.\n\n",
		"eject.schema_doc.required_heading": "\n## Required fields by stage\n\n",
		"eject.schema_doc.dates": `
## Date fields

- created. When the original was first recorded. Filled by a person or intake.
  YYYY-MM is allowed when only the year and month are known.
- sourced_at. When it was incorporated into this wiki. Filled by the tool.
- updated. When the content was last updated. Filled by the tool. Never written by hand.
- Documents in the ` + "`sources/`" + ` tier carry no updated. The preservation tier is never
  updated, and an updated field would mislead about freshness.

`,

		"eject.values_doc.heading":        "# Value sets\n\n",
		"eject.values_doc.intro":          "Closed sets (values outside are violations) are distinguished from open sets (values outside are warnings).\n\n",
		"eject.values_doc.closed_heading": "## Closed sets\n\n",
		"eject.values_doc.open_heading":   "\n## Open sets\n\n- topics: %s\n",
		"eject.values_doc.open_note":      "\nA topics value not defined here is a warning. The value itself is allowed.\n\n",

		"eject.promotion_doc.heading":          "# Promotion gate and location rules\n\n",
		"eject.promotion_doc.intro":            "min_wikilinks is %d. The gate rejects a document under the context directory whose unique wikilink count falls below it. 0 disables the gate. The gate triggers on the document's directory, not its declaration.\n\n",
		"eject.promotion_doc.sole_reason":      "gate.min-wikilinks is the only rejection reason. To pass the gate, the document must be wikilinked to related documents.\n\n",
		"eject.promotion_doc.deferred":         "Linkable targets are counted only when there are enough documents. If target documents are fewer than min_wikilinks, the gate is deferred and only a warning is issued. The gate reactivates as the wiki grows. inbox-stage documents are excluded because promote changes their slug and breaks links.\n\n",
		"eject.promotion_doc.location_heading": "## Location and stage agreement\n\nThe document's top-level directory must match its artifact_stage value.\n\n",
		"eject.promotion_doc.location_row":     "- %s-stage documents go in the %s/ directory\n",
		"eject.promotion_doc.context_note":     "\nDeclaring context while sitting outside the context directory is an error: the document claims the field set of reviewed knowledge and index eligibility while bypassing the pipeline. Other mismatches are warnings.\n\n",

		"eject.lint_rules_doc.heading":        "# lint rules\n\nRule IDs are lower-case dot notation. Severity meanings are below.\n\n",
		"eject.lint_rules_doc.severities":     "- error: blocks promotion\n- warn: passes with a notice\n- reject: promotion gate rejection\n\n",
		"eject.lint_rules_doc.table_heading":  "## Rule list\n\n",
		"eject.lint_rules_doc.table_header":   "| Rule ID | Severity | What it checks |\n|---|---|---|\n",
		"eject.lint_rules_doc.script_heading": "\n## Rules judged by scripts/lint-frontmatter.py\n\n",
		"eject.lint_rules_doc.script_note":    "It judges per-document rules, link integrity, orphan detection, and the promotion gate. Only wiki.broad-topic is missing from the table above: a whole-wiki statistic has no place in a per-document hook.\n\n",

		"eject.layout_doc.heading":       "# Wiki layout\n\n",
		"eject.layout_doc.stage_heading": "## Stage directories\n\n",
		"eject.layout_doc.stage_row":     "- %s/ (stage %s)\n",
		"eject.layout_doc.root_files":    "\n## Root files\n\nDefined by root_files: %s\n",
		"eject.layout_doc.root_note":     "\nRoot files are indexes. They are exempt from the promotion gate and orphan detection but still undergo schema checks.\n\n",
		"eject.layout_doc.ignore_files":  "## Non-document files\n\nDefined by ignore_files: %s\n",
		"eject.layout_doc.ignore_note":   "\nAny file with a matching name is excluded regardless of depth, such as a README describing a directory.\n\n",

		"eject.hook.comment":   "# Pre-commit document rule check. Created by engram eject.\n# Activate: git config core.hooksPath .githooks\n",
		"eject.hook.blocked":   "Commit blocked by document rule violations. Fix them and commit again",
		"eject.hook.no_python": "If python3 is missing, see the notes at the top of scripts/lint-frontmatter.py",

		"eject.agents_doc.heading":         "# AGENTS.md\n\nAgents working in this wiki follow the rules below.\n\n",
		"eject.agents_doc.rules_heading":   "## The rules belong to this wiki\n\n",
		"eject.agents_doc.rules_body":      "Document rules are owned by this wiki. The rule specs live in the documents under meta/ and scripts/lint-frontmatter.py does the judging. To change a rule, edit those directly. Axes, allowed values, thresholds, and directories come from engram.yaml, so the script reads that file at run time.\n\n",
		"eject.agents_doc.hook_body":       "Pre-commit checks run from .githooks/pre-commit. Activate it once.\n\n    git config core.hooksPath .githooks\n\n",
		"eject.agents_doc.compute_heading": "## engram keeps doing the computation\n\n",
		"eject.agents_doc.compute_body":    "eject exported rules only. Search indexing, resurfacing, link graph computation, and digests are computations that cannot live in files, so engram keeps them. The commands below keep working.\n\n",
		"eject.agents_doc.lint_body":       "Running engram lint checks with the built-in rules. The whole-wiki diagnosis wiki.broad-topic is not exported, so only engram lint judges it.\n\n",
		"eject.agents_doc.dirs_heading":    "## Directories\n\n",
		"eject.agents_doc.dirs_row":        "- %s/. Where %s-stage documents live\n",
		"eject.agents_doc.root_files":      "- Root files (root_files): %s\n",
		"eject.agents_doc.ignore_files":    "- Non-document files (ignore_files): %s\n",
		"eject.agents_doc.preset_heading":  "\n## Preset\n\n",
		"eject.agents_doc.preset_body":     "This wiki uses the %s preset.\n",
		"eject.agents_doc.oneway_heading":  "\n## One way\n\n",
		"eject.agents_doc.oneway_body":     "eject has no undo. Re-running it overwrites the rule files. Recover from git history if needed.\n",

		"eject.gitattributes.comment": "# Normalize line endings to LF. The linter also accepts CRLF, but a single\n# form is better for diffing and merging.\n",

		"eject.linter.docstring": `Linter that judges per-document rules. Created by engram eject.

Axes, allowed values, thresholds, and directories are read from engram.yaml
at run time. Changing the config takes effect without regenerating this
script. Only values engram.yaml cannot change are baked into the constants
below: the fixed allowed sets and the stage-to-directory mapping.

What this linter does not export:
- wiki.broad-topic. A diagnosis judged from whole-wiki statistics, not a
  per-document hook. engram lint keeps judging it.
- Search index, resurfacing, link graph computation, digests. These are
  computations, not files. engram search, recall, resurface, bridge, digest,
  backlinks keep performing them.

This wiki used the %s preset at eject time.

Usage: python3 scripts/lint-frontmatter.py [wiki-root]
Exit codes match engram lint. 1 when there is an error or a reject, 0
otherwise. Warnings do not affect the exit code, because the commit hook
calling this script must not block a wiki that only has warnings.
`,
		"eject.linter.console_comment": `# The default console encoding on Windows is not UTF-8 but cp949 or cp1252.
# Emitting non-ASCII messages as-is dies with UnicodeEncodeError. The engram
# binary fixes this by switching the console code page to UTF-8, but this
# exported script does not get that treatment, so re-open the streams here.
`,
		"eject.linter.newline_comment": `# Also pin newline. Windows text mode rewrites \n to \r\n on output, while
# engram lint emits only \n; leaving this alone would make the two linters
# differ in line endings and break the same-verdict contract.
`,
		"eject.linter.constants_comment":   "# --- Fixed constants. Values engram.yaml cannot change. ---\n",
		"eject.linter.stage_dirs_comment":  "# Stage-to-directory mapping.\n",
		"eject.linter.preset_axes_comment": "# Default axes per preset. minimal is contained in personal, personal in\n# team. engram.yaml's axes override individual axes.\n",
		"eject.linter.defaults_comment":    "# Defaults when engram.yaml omits a key. Taken from the config at eject time.\n",
		"eject.linter.backtick_comment":    "# The backtick character. Used to detect code fences and inline code.\n",
		"eject.linter.config_section":      "# --- Config reading ---\n",
		"eject.linter.yaml_docstring": `Parses the YAML subset used by frontmatter and engram.yaml.

    Handles key: value, flow lists [a, b], block lists, and one-level maps.
    No PyYAML; the eject contract is standard library only.
    `,
		"eject.linter.empty_key_comment":   "                # Keys with no value stay empty. The key's presence is what\n                # gets judged (as with source_channel:).\n",
		"eject.linter.parse_section":       "# --- Document parsing ---\n",
		"eject.linter.walk_section":        "# --- Walk ---\n",
		"eject.linter.judge_section":       "# --- Judging ---\n",
		"eject.linter.int_stage_comment":   "        # Values YAML reads as numbers are still strings to engram.\n",
		"eject.linter.stage_input_comment": "            # artifact_stage is the input to stage judgment. Its absence is\n            # itself an error, and since the stage is unknown no other required\n            # fields are reported (ADR 0040).\n",
		"eject.linter.related_comment":     "        # Links from the related field. Parsed values are the source of truth; only line numbers come from the raw text.\n",
		"eject.linter.gate_dir_comment":    "        # The gate triggers on the document's directory (ADR 0040). Judging\n        # by declaration would allow gaming it with empty or low values.\n",
		"eject.linter.gate_line_comment":   "            # The line number points at the related key, same as engram lint.\n",
		"eject.linter.exit_code_comment":   "    # Exit code rule matches engram lint's HasBlocking. 1 only when there\n    # is an error or a reject. Warnings alone never block a commit.\n",

		// The strings below are filled by Python at run time.
		"eject.linter.unclosed.message":         "Frontmatter ends without a closing --- delimiter",
		"eject.linter.unclosed.fix":             "Add a --- line at the end of the frontmatter",
		"eject.linter.missing.message":          "No frontmatter",
		"eject.linter.missing.fix":              "Open with --- on the first line, fill in the fields, and close with ---",
		"eject.linter.stage_missing.message":    "No artifact_stage field",
		"eject.linter.stage_missing.fix":        "Fill in the artifact_stage field in the frontmatter",
		"eject.linter.required_missing.message": "Stage %s requires field %s, which is missing",
		"eject.linter.required_missing.fix":     "Add the %s field to the frontmatter",
		"eject.linter.allowed_value.message":    `%s value is outside the allowed set: "%s" (allowed: %s)`,
		"eject.linter.allowed_value.fix":        "Change the %s value to one of the allowed values",
		"eject.linter.stage_agreement.message":  `Document is in the %s directory but artifact_stage is "%s"`,
		"eject.linter.stage_agreement.fix":      "Move the document to the directory matching its artifact_stage, or change artifact_stage to %s. Use engram promote, demote, or archive to move documents",
		"eject.linter.axis_off.message":         "The document carries an axis the config turned off: %s (preset %s)",
		"eject.linter.axis_off.fix":             "Turn on %s in the axes of engram.yaml, or remove the %s field from the document",
		"eject.linter.forms.message":            `form value is not in the closed forms set: "%s" (allowed: %s)`,
		"eject.linter.forms.fix":                "Change the form value to one of the allowed values",
		"eject.linter.topics.message":           `topics value is not defined in config: "%s" (topics is an open set)`,
		"eject.linter.topics.fix":               `Add "%s" to the topics list in engram.yaml`,
		"eject.linter.sources_updated.message":  "A sources-tier document has an updated field",
		"eject.linter.sources_updated.fix":      "Remove the updated field. sources is a preservation tier and is never updated",
		"eject.linter.max_lines.message":        "Document is %d lines, exceeding the max_lines of %d",
		"eject.linter.max_lines.fix":            "Split the document. Adjust the limit with max_lines in engram.yaml",
		"eject.linter.link_broken.message":      "Broken wikilink: no document matches [[%s]]",
		"eject.linter.link_broken.fix":          "Fix the slug or create the [[%s]] document",
		"eject.linter.orphan.message":           "No incoming or outgoing relations",
		"eject.linter.orphan.fix":               "Link to it from another document's related or body with [[%s]], or connect via relation fields",
		"eject.linter.gate_deferred.message":    "Only %d linkable target documents versus min_wikilinks of %d, so the gate is deferred. The gate activates once there are %d target documents",
		"eject.linter.gate_deferred.fix":        "Create documents to link to. Adjust the threshold with min_wikilinks in engram.yaml",
		"eject.linter.gate_reject.message":      "%d wikilinks fall short of min_wikilinks %d",
		"eject.linter.gate_reject.fix":          "Add %d more wikilinks in the related field or the body",
		"eject.linter.print_fix":                "    How to fix: %s",
		"eject.linter.summary":                  "Files checked %d, error %d, warn %d, reject %d",
	})
}
