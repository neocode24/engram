// Package i18n의 cli_ingest 카탈로그. ADR 0049.
package i18n

func init() {
	Register(LangKO, map[string]string{
		"cli.ingest.flag_wiki":        "대상 위키 경로",
		"cli.ingest.flag_title":       "문서 제목. 생략하면 본문 첫 줄에서 만듭니다",
		"cli.ingest.flag_slug":        "파일명 슬러그. 생략하면 제목에서 만듭니다",
		"cli.ingest.not_wiki":         "위키가 아닌 디렉토리입니다: %s\n먼저 engram init을 실행하세요",
		"cli.ingest.stat_fail":        "대상 경로를 확인할 수 없음",
		"cli.ingest.config_load_fail": "위키 설정을 읽을 수 없음",
		"cli.ingest.flag_read_fail":   "--%s 플래그를 읽을 수 없음",
		"cli.wiki_path.both_given":    "위키 경로를 위치 인자와 --wiki 둘 다로 주셨습니다\n어느 쪽을 뜻하는지 알 수 없으니 하나만 남겨 주세요",
		"cli.wiki_path.not_dir":       "위키 경로에 파일을 주셨습니다: %s\n이 커맨드는 위키 전체를 봅니다. 위키 디렉토리를 지정해 주세요",
		"cli.ingest.stdin_read_fail":  "표준 입력을 읽을 수 없음",
		"cli.ingest.no_content":       "내용을 받지 못했습니다\n인자로 내용을 주거나 파이프로 표준 입력을 넘기세요. 예: engram capture \"메모 내용\"",
		"cli.ingest.stage_put":        "%s에 넣었습니다: %s",
		"cli.ingest.next":             "다음: %s",
		"cli.ingest.doc_read_fail":    "문서를 읽을 수 없음: %s",
		"cli.ingest.doc_parse_fail":   "문서를 파싱할 수 없음: %s",
		"cli.ingest.doc_not_found":    "문서를 찾을 수 없습니다: %s\n위키 루트 상대 경로로 주세요. 예: engram promote inbox/2026-01-01-메모.md",
		"cli.ingest.rel_fail":         "위키 루트 기준 경로를 계산할 수 없음",
		"cli.ingest.walk_fail":        "위키를 순회할 수 없음",
		"cli.ingest.dest_exists":      "도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 슬러그를 다르게 지정하세요",
		"cli.ingest.type_invalid":     "--type 값이 허용값 밖입니다: %q (허용값: %s)",
		"cli.ingest.gate_line":        "게이트: 링크 %d개, 대상 %d개, 기준 %d개%s",
		"cli.ingest.deferred_note":    " (유예)",
		"cli.capture.short":           "새 메모를 검증 없이 inbox에 넣습니다",
		"cli.capture.long": `새 메모를 inbox에 넣습니다. 스키마 검증을 하지 않습니다.

내용을 인자로 받거나 파이프로 연결된 표준 입력으로 받습니다.
제목은 --title으로 주고, 생략하면 본문 첫 줄에서 만듭니다.
파일명은 <날짜>-<슬러그>.md이고 날짜는 전역 --now 기준입니다.`,
		"cli.capture.next_hint": "문서를 정리한 뒤 승급하세요. 지금은 engram lint로 위키 상태를 볼 수 있습니다",
		"cli.source.short":      "원본 자료를 sources에 넣고 출처를 확정합니다",
		"cli.source.long": `원본 자료를 sources에 넣고 원본 필드를 확정합니다.

내용을 인자로 받거나 파이프로 연결된 표준 입력으로 받습니다.
--created로 원본이 작성된 날을 줍니다. 하루(YYYY-MM-DD) 또는 연월(YYYY-MM)
정밀도를 허용하고 생략하면 전역 --now 기준 날짜를 씁니다.
이 계층은 원본 보존이 계약이므로 문서를 고치지 않고 updated 필드도 쓰지 않습니다.`,
		"cli.source.flag_created":     "원본이 작성된 날(YYYY-MM-DD 또는 YYYY-MM)",
		"cli.source.flag_channel":     "입력 경로. source_channel 속성 값",
		"cli.source.flag_ref":         "원본 출처(경로나 URL). 여러 번 쓸 수 있습니다",
		"cli.source.flag_type":        "문서 종류. 허용값은 위키 설정의 types입니다",
		"cli.source.created_invalid":  "--created 값이 YYYY-MM-DD 또는 YYYY-MM 형식이 아닙니다: %q",
		"cli.source.next_hint":        "정리가 끝나면 이 원본을 인용하는 맥락 문서를 만드세요",
		"cli.source.warn_channel_off": "경고: source_channel 속성이 꺼져 있어 --channel 값을 무시합니다. 켜려면 engram.yaml의 axes에서 source_channel을 true로 두세요",
		"cli.source.warn_refs_off":    "경고: source_refs 속성이 꺼져 있어 --ref 값을 무시합니다. 켜려면 engram.yaml의 axes에서 source_refs를 true로 두세요",
		"cli.promote.short":           "기존 문서를 context 단계로 올립니다",
		"cli.promote.long": `기존 문서를 승급 게이트로 검사해 context 단계로 올립니다.

inbox 문서는 이동합니다. 원본이 남지 않습니다. inbox는 임시 계층입니다.
sources 문서는 파생을 만듭니다. 원본이 그대로 남습니다. sources는 원본
보존 계층이고 이후 본문을 고치지 않는 것이 계약입니다.

게이트는 lint.EvaluateGate가 단일 진실원입니다. 링크가 부족하면
--related <슬러그>를 반복해 이 자리에서 채울 수 있습니다.`,
		"cli.promote.flag_slug":            "도착지 문서 슬러그. 생략하면 원본 파일명에서 날짜 접두사를 뗀 값입니다",
		"cli.promote.flag_related":         "related 필드에 추가할 슬러그. 여러 번 쓰거나 쉼표로 나눠 주세요",
		"cli.promote.flag_type":            "승급 문서의 문서 종류. 허용값은 위키 설정의 types입니다",
		"cli.promote.already_context":      "이미 context 단계입니다: %s\npromote는 inbox나 sources 문서를 올립니다",
		"cli.promote.unknown_stage":        "문서 단계를 알 수 없습니다: %s의 artifact_stage 값이 %q다 (허용값: inbox, source, context)",
		"cli.promote.warn_default_type":    "경고: 문서 종류가 %s 단계 기본값 %q 그대로입니다. --type으로 지정하세요 (허용값: %s)",
		"cli.promote.inbox_remove_fail":    "inbox 원본을 지울 수 없음: %s",
		"cli.promote.derived_context_fail": "원본의 derived_context를 갱신할 수 없음: %s",
		"cli.promote.done":                 "context로 올렸습니다: %s",
		"cli.promote.type_line":            "문서 종류: %s",
		"cli.promote.next":                 "다음: engram lint로 승급 문서의 스키마를 확인하세요",
		"cli.promote.plan":                 "올릴 예정입니다: %s에서 %s로",
		"cli.promote.flag_to":              "승급 대상 단계입니다. context(기본) 또는 sources. sources는 원본을 증거로 옮깁니다",
		"cli.promote.to_invalid":           "--to 값이 허용값 밖입니다: %q (허용값: %s)",
		"cli.promote.to_sources_from":      "sources로는 inbox 문서만 옮길 수 있습니다: %s (지금 단계: %s)\ncontext 문서를 내리려면 engram demote --to sources를 쓰세요",
		"cli.promote.to_sources_plan":      "sources로 옮길 예정입니다: %s에서 %s로",
		"cli.promote.to_sources_done":      "sources로 옮겼습니다: %s",
		"cli.promote.to_sources_next":      "원본은 이후 본문을 고치지 않습니다. 다음: 이 원본을 인용하는 맥락 문서를 만드세요",
		"cli.promote.flag_dry_run":         "쓰지 않고 게이트만 확인합니다. 통과하면 무엇이 만들어질지 냅니다",
		"cli.ingest.dry_run_note":          "시험 실행이라 아무것도 쓰지 않았습니다",
		"cli.new.short":                    "처음부터 검수된 지식으로 context에 씁니다",
		"cli.new.long": `검수된 지식을 곧바로 context 단계로 씁니다.

승급 게이트는 promote와 같은 함수로 판정합니다. 링크가 부족하면
--related <슬러그>를 반복해 이 자리에서 채울 수 있습니다.
본문은 upstream promotion-rules.md가 요구하는 절 제목의 골격만 넣습니다.`,
		"cli.new.flag_slug":    "문서 슬러그. 생략하면 제목에서 만듭니다",
		"cli.new.flag_type":    "문서 종류. 기본값은 context 단계 초기값입니다",
		"cli.new.flag_form":    "문서 형태. forms 폐쇄 집합의 값이어야 합니다",
		"cli.new.flag_topics":  "주제. 여러 번 쓸 수 있습니다",
		"cli.new.flag_tags":    "광범위 묶음 태그. 여러 번 쓸 수 있습니다",
		"cli.new.flag_related": "관련 문서 슬러그. 여러 번 쓰거나 쉼표로 나눠 주세요",
		"cli.new.done":         "context에 썼습니다: %s",
		"cli.new.next":         "다음: 골격의 절을 채우고 engram lint로 확인하세요",
		"cli.new.form_empty":   "--form 값을 받을 수 없습니다: %q\n위키 설정의 forms가 비어 있습니다. engram.yaml의 forms에 값을 정의하세요",
		"cli.new.form_invalid": "--form 값이 forms 폐쇄 집합에 없습니다: %q (허용값: %s)",
		"cli.demote.short":     "context 문서를 inbox나 sources로 되돌립니다",
		"cli.demote.long": `context 단계 문서를 inbox 또는 sources로 내립니다.

도착 단계의 기본값은 inbox입니다. inbox가 임시 계층이라 되돌리기의
도착지로 안전합니다.

문서를 내리면 파일명에 날짜 접두사가 붙어 슬러그가 바뀝니다. 그 문서를
가리키던 위키링크는 전부 깨지므로 실행 전에 목록을 경고로 냅니다.
되돌리기를 막지는 않습니다. 깨진 링크는 engram mv로 고치세요.

파생 문서라면 원본 sources 문서의 derived_context도 어긋납니다.
원본 되돌리기는 이 커맨드의 범위가 아니므로 경고로만 알립니다.`,
		"cli.demote.flag_to":          "도착 단계. 허용값: inbox, sources",
		"cli.demote.not_context":      "context 단계 문서만 되돌립니다: %s의 artifact_stage 값이 %q입니다",
		"cli.demote.to_invalid":       "--to 값이 허용값 밖입니다: %q (허용값: inbox, sources)",
		"cli.demote.warn_broken_link": "경고: 깨질 위키링크: %s %d줄이 [[%s]]를 가리킵니다. engram mv로 링크를 고치세요",
		"cli.demote.warn_derived":     "경고: 이 문서는 원본 %s에서 파생되었습니다. 원본의 derived_context 갱신은 이 커맨드가 하지 않습니다. engram update로 직접 고치세요",
		"cli.demote.remove_fail":      "context 원본을 지울 수 없음: %s",
		"cli.demote.done":             "%s로 내렸습니다: %s",
		"cli.demote.meta":             "날짜 접두사: %s, 슬러그: %s",
		"cli.demote.broken_count":     "깨질 링크 %d건. engram mv로 고치세요",
		"cli.demote.derived_count":    "파생 원본 %d건의 derived_context가 어긋났습니다",
		"cli.demote.next":             "다음: 정리가 끝나면 engram promote로 다시 올리세요",
		"cli.archive.short":           "수명이 끝난 context 문서를 보관합니다",
		"cli.archive.long": `context 단계 문서를 archive 디렉토리로 옮깁니다.

수명이 끝난 문서를 걷어 내되 링크는 깨지 않습니다. 슬러그를 바꾸지
않고 날짜 접두사도 붙이지 않으므로 이 문서를 가리키던 위키링크는
그대로 남습니다. 폐기된 결정을 가리키는 링크가 깨지면 안 되기
때문입니다. 승급이 잘못됐다면 engram demote를 쓰세요.

프론트매터의 artifact_stage를 archive로, status를 archived로 바꾸고
updated를 갱신합니다. 이 문서를 가리키는 링크가 있으면 개수를
알립니다. 막지는 않습니다.`,
		"cli.archive.already_archived":  "이미 보관 상태입니다: %s",
		"cli.archive.not_context":       "context 단계 문서만 보관합니다: %s의 artifact_stage 값이 %q입니다\n수명이 끝난 context 문서가 대상입니다. 승급이 잘못됐다면 engram demote를 쓰세요",
		"cli.archive.dest_exists":       "도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 보관 디렉토리에서 정리한 뒤 다시 시도하세요",
		"cli.archive.notice_incoming":   "알림: 이 문서를 가리키는 링크 %d개가 있습니다. 링크는 깨지지 않고 가리키는 대상이 폐기 상태가 됩니다",
		"cli.archive.remove_fail":       "보관 전 원본을 지울 수 없음: %s",
		"cli.archive.done":              "보관했습니다: %s",
		"cli.archive.slug_kept":         "슬러그 %s는 유지됩니다. 들어오는 링크는 깨지지 않습니다",
		"cli.archive.incoming_now_dead": "이 문서를 가리키던 링크 %d개가 폐기 상태를 가리키게 됩니다",
		"cli.mv.short":                  "문서 슬러그를 바꾸고 걸린 링크를 모두 고칩니다",
		"cli.mv.long": `문서의 슬러그를 바꾸고 그 문서를 가리키는 모든 링크를 고칩니다.

본문 위키링크의 표시 문자열과 헤딩은 보존하고 슬러그만 바꿉니다.
related, derived_from, derived_context, source_refs 필드도 같이 고칩니다.
코드 펜스와 인라인 코드 안의 링크 문법은 고치지 않습니다.

날짜 접두사 규칙을 지킵니다. 원본에 접두사가 있었으면 유지하고
슬러그 부분만 바꿉니다. 새 슬러그는 슬러그 규칙(ADR 0020)으로 정규화합니다.

링크를 먼저 다 고치고 파일 이동을 마지막에 합니다. 중간에 실패하면
링크가 옛 이름을 가리키는 상태로 끝나므로 mv를 다시 돌리면 수습됩니다.

--dry-run 은 무엇이 바뀌는지만 내고 아무것도 쓰지 않습니다.`,
		"cli.mv.flag_dry_run":     "무엇이 바뀌는지만 내고 아무것도 쓰지 않습니다",
		"cli.mv.old_empty":        "옛 슬러그가 비었습니다: %q",
		"cli.mv.same_slug":        "새 슬러그가 옛 슬러그와 같습니다: %s",
		"cli.mv.old_not_found":    "옛 슬러그에 해당하는 문서가 없습니다: %s\n문서 경로나 슬러그를 바로 줍니다. 예: engram mv note memo",
		"cli.mv.slug_taken":       "새 슬러그가 이미 쓰이고 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 다른 슬러그를 고르세요",
		"cli.mv.dest_exists":      "도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다",
		"cli.mv.rename_fail":      "문서를 옮길 수 없음: %s 로",
		"cli.mv.link_read_fail":   "링크 문서를 읽을 수 없음: %s",
		"cli.mv.link_parse_fail":  "링크 문서를 파싱할 수 없음: %s",
		"cli.mv.link_write_fail":  "링크 문서를 쓸 수 없음: %s",
		"cli.mv.verb_moved":       "옮겼습니다",
		"cli.mv.verb_will_change": "바꿀 예정입니다",
		"cli.mv.summary":          "%s: %s에서 %s로 (슬러그 %s)",
		"cli.mv.no_links":         "고칠 링크가 없습니다",
		"cli.mv.links_total":      "고친 링크 %d건:",
		"cli.mv.links_per_file":   "  %s: %d건",
		"cli.mv.rejections_fixed": "bridge 기각 쌍 %d건의 슬러그도 함께 고쳤습니다",
		"cli.mv.dry_run_note":     "시험 실행이라 아무것도 쓰지 않았습니다",
		"cli.mv.next":             "다음: engram lint로 링크 무결성을 확인하세요",
		"cli.update.short":        "문서의 프론트매터와 본문을 갱신합니다",
		"cli.update.long": `문서의 프론트매터 키와 본문을 갱신합니다.

--set key=value 로 키를 설정합니다. 반복할 수 있습니다. 배열 속성은 쉼표로
여러 값을 줍니다. 예: --set topics=go,cli
--unset key 로 키를 지웁니다. 반복할 수 있습니다.
--body-from <파일> 로 본문을 통째로 바꿉니다. - 를 주면 표준 입력을 읽습니다.

꺼진 속성의 키와 허용값 밖의 값은 거절합니다. artifact_stage는 여기서
바꾸지 못합니다. 단계 이동은 engram promote와 engram demote의 일입니다.
키 순서는 파싱이 보존한 그대로 유지됩니다.`,
		"cli.update.flag_set":              "프론트매터 키 설정. key=value. 반복 가능",
		"cli.update.flag_unset":            "프론트매터 키 제거. 반복 가능",
		"cli.update.flag_body_from":        "본문을 통째로 바꿉니다. 파일 경로 또는 - (표준 입력)",
		"cli.update.no_changes":            "갱신할 내용이 없습니다\n--set key=value, --unset key, --body-from <파일|-> 중 하나를 쓰세요",
		"cli.update.set_invalid":           "--set 값이 key=value 형식이 아닙니다: %q",
		"cli.update.unset_stage_forbidden": "artifact_stage는 update로 지울 수 없습니다. 단계 이동은 engram promote와 engram demote가 합니다",
		"cli.update.unset_axis_off":        "꺼진 속성의 키는 지울 것도 없습니다: %s",
		"cli.update.warn_sources_body":     "경고: sources는 원본 보존 계층입니다. 본문을 바꾼 사실을 기억해 두세요",
		"cli.update.write_fail":            "문서를 쓸 수 없음: %s",
		"cli.update.set_stage_forbidden":   "artifact_stage는 update로 바꿀 수 없습니다. 단계 이동은 engram promote와 engram demote가 합니다",
		"cli.update.set_axis_off":          "꺼진 속성의 키는 설정할 수 없습니다: %s (프리셋 %s). engram.yaml의 axes에서 켜거나 문서에서 쓰지 않습니다",
		"cli.update.set_empty_item":        "--set %s=%s 값에 빈 항목이 있습니다",
		"cli.update.set_not_bool":          "--set %s=%s 값이 참/거짓이 아닙니다 (true 또는 false)",
		"cli.update.set_not_allowed":       "--set %s=%s 값이 허용값 밖입니다 (허용값: %s)",
		"cli.update.body_read_fail":        "본문 파일을 읽을 수 없음: %s",
		"cli.update.done":                  "갱신했습니다: %s",
		"cli.update.set_line":              "설정: %s",
		"cli.update.unset_line":            "제거: %s",
		"cli.update.body_line":             "본문 교체: %s",
		"cli.update.updated_line":          "updated에 갱신 날짜 %s를 기록했습니다",
		"cli.update.next":                  "다음: engram lint로 갱신된 문서의 스키마를 확인하세요",
	})

	Register(LangEN, map[string]string{
		"cli.ingest.flag_wiki":        "Target wiki path",
		"cli.ingest.flag_title":       "Document title. Defaults to the first line of the body",
		"cli.ingest.flag_slug":        "File name slug. Defaults to the title",
		"cli.ingest.not_wiki":         "Not a wiki directory: %s\nRun engram init first",
		"cli.ingest.stat_fail":        "cannot check target path",
		"cli.ingest.config_load_fail": "cannot read wiki config",
		"cli.ingest.flag_read_fail":   "cannot read --%s flag",
		"cli.wiki_path.both_given":    "The wiki path was given both as a positional argument and as --wiki\nIt is unclear which one you mean; keep only one",
		"cli.wiki_path.not_dir":       "The wiki path points to a file: %s\nThis command works on the whole wiki. Give the wiki directory",
		"cli.ingest.stdin_read_fail":  "cannot read standard input",
		"cli.ingest.no_content":       "No content received\nPass content as arguments or pipe standard input. Example: engram capture \"note text\"",
		"cli.ingest.stage_put":        "Added to %s: %s",
		"cli.ingest.next":             "Next: %s",
		"cli.ingest.doc_read_fail":    "cannot read document: %s",
		"cli.ingest.doc_parse_fail":   "cannot parse document: %s",
		"cli.ingest.doc_not_found":    "cannot find document: %s\nPass a path relative to the wiki root. Example: engram promote inbox/2026-01-01-memo.md",
		"cli.ingest.rel_fail":         "cannot compute path relative to the wiki root",
		"cli.ingest.walk_fail":        "cannot walk the wiki",
		"cli.ingest.dest_exists":      "destination already has a document: %s\nExisting documents are not overwritten. Choose a different slug",
		"cli.ingest.type_invalid":     "--type value is not allowed: %q (allowed: %s)",
		"cli.ingest.gate_line":        "Gate: %d links, %d targets, min %d%s",
		"cli.ingest.deferred_note":    " (deferred)",
		"cli.capture.short":           "Add a new note to inbox without validation",
		"cli.capture.long": `Adds a new note to inbox. Schema validation is skipped.

Content comes from arguments or piped standard input.
Set the title with --title; when omitted it comes from the first line of the body.
The file name is <date>-<slug>.md with the date from the global --now.`,
		"cli.capture.next_hint": "Tidy up the document, then promote it. For now, engram lint shows the wiki state",
		"cli.source.short":      "Add source material to sources and record its origin",
		"cli.source.long": `Adds source material to sources and records its origin fields.

Content comes from arguments or piped standard input.
--created sets the day the original was written. Day (YYYY-MM-DD) or month
(YYYY-MM) precision is allowed; when omitted the global --now date is used.
This layer is preservation-by-contract: documents are never edited and the updated field is never written.`,
		"cli.source.flag_created":     "Day the original was written (YYYY-MM-DD or YYYY-MM)",
		"cli.source.flag_channel":     "Input channel. Value of the source_channel attribute",
		"cli.source.flag_ref":         "Origin of the source (path or URL). Can be repeated",
		"cli.source.flag_type":        "Document type. Allowed values are the types in the wiki config",
		"cli.source.created_invalid":  "--created value is not in YYYY-MM-DD or YYYY-MM format: %q",
		"cli.source.next_hint":        "When the tidy-up is done, write a context document citing this source",
		"cli.source.warn_channel_off": "Warning: the source_channel attribute is off, so --channel is ignored. Turn source_channel on under axes in engram.yaml to use it",
		"cli.source.warn_refs_off":    "Warning: the source_refs attribute is off, so --ref is ignored. Turn source_refs on under axes in engram.yaml to use it",
		"cli.promote.short":           "Promote an existing document to the context stage",
		"cli.promote.long": `Checks an existing document against the promotion gate and moves it up to the context stage.

inbox documents are moved. No original remains. inbox is a staging layer.
sources documents get a derivative. The original stays intact. sources is a
preservation layer and never editing its body afterwards is the contract.

The single source of truth of the gate is lint.EvaluateGate. When links are
short, repeat --related <slug> to fill them in here.`,
		"cli.promote.flag_slug":            "Destination document slug. Defaults to the original file name minus its date prefix",
		"cli.promote.flag_related":         "Slug to add to the related field. Repeat it or separate with commas",
		"cli.promote.flag_type":            "Document type of the promoted document. Allowed values are the types in the wiki config",
		"cli.promote.already_context":      "already at the context stage: %s\npromote moves inbox or sources documents",
		"cli.promote.unknown_stage":        "cannot determine the document stage: artifact_stage of %s is %q (allowed: inbox, source, context)",
		"cli.promote.warn_default_type":    "Warning: the type is still the %s stage default %q. Set it with --type (allowed: %s)",
		"cli.promote.inbox_remove_fail":    "cannot delete the inbox original: %s",
		"cli.promote.derived_context_fail": "cannot update derived_context of the original: %s",
		"cli.promote.done":                 "Promoted to context: %s",
		"cli.promote.type_line":            "Type: %s",
		"cli.promote.next":                 "Next: check the promoted document's schema with engram lint",
		"cli.promote.plan":                 "Would promote: %s to %s",
		"cli.promote.flag_to":              "Target stage. context (default) or sources. sources moves the original as evidence",
		"cli.promote.to_invalid":           "--to value is not allowed: %q (allowed: %s)",
		"cli.promote.to_sources_from":      "Only inbox documents can move to sources: %s (current stage: %s)\nTo move a context document down, use engram demote --to sources",
		"cli.promote.to_sources_plan":      "Would move to sources: %s to %s",
		"cli.promote.to_sources_done":      "Moved to sources: %s",
		"cli.promote.to_sources_next":      "The original body is never edited after this. Next: write a context document that cites it",
		"cli.promote.flag_dry_run":         "Check the gate without writing. Reports what would be created if it passes",
		"cli.ingest.dry_run_note":          "Dry run: nothing was written",
		"cli.new.short":                    "Write reviewed knowledge straight to context",
		"cli.new.long": `Writes reviewed knowledge directly to the context stage.

The promotion gate is judged by the same function as promote. When links
are short, repeat --related <slug> to fill them in here.
The body gets only the skeleton of the section titles that upstream promotion-rules.md requires.`,
		"cli.new.flag_slug":    "Document slug. Defaults to the title",
		"cli.new.flag_type":    "Document type. Defaults to the context stage initial value",
		"cli.new.flag_form":    "Document form. Must be a value of the closed forms set",
		"cli.new.flag_topics":  "Topics. Can be repeated",
		"cli.new.flag_tags":    "Broad grouping tags. Can be repeated",
		"cli.new.flag_related": "Related document slugs. Repeat them or separate with commas",
		"cli.new.done":         "Wrote to context: %s",
		"cli.new.next":         "Next: fill in the skeleton sections and check with engram lint",
		"cli.new.form_empty":   "--form cannot be accepted: %q\nforms is empty in the wiki config. Define values in forms of engram.yaml",
		"cli.new.form_invalid": "--form value is not in the closed forms set: %q (allowed: %s)",
		"cli.demote.short":     "Revert a context document to inbox or sources",
		"cli.demote.long": `Moves a context stage document down to inbox or sources.

The default destination stage is inbox. inbox is a staging layer, which
makes it a safe destination for a revert.

Demoting adds a date prefix to the file name, so the slug changes. Every
wikilink pointing at the document breaks, so the list is printed as a warning
before running. The revert is not blocked. Fix broken links with engram mv.

For derivative documents the derived_context of the original sources document
also drifts. Reverting the original is out of scope for this command, so it is
only warned about.`,
		"cli.demote.flag_to":          "Destination stage. Allowed: inbox, sources",
		"cli.demote.not_context":      "only context stage documents can be reverted: artifact_stage of %s is %q",
		"cli.demote.to_invalid":       "--to value is not allowed: %q (allowed: inbox, sources)",
		"cli.demote.warn_broken_link": "Warning: wikilink about to break: %s line %d points to [[%s]]. Fix it with engram mv",
		"cli.demote.warn_derived":     "Warning: this document derives from %s. This command does not update the original's derived_context. Fix it directly with engram update",
		"cli.demote.remove_fail":      "cannot delete the context original: %s",
		"cli.demote.done":             "Demoted to %s: %s",
		"cli.demote.meta":             "Date prefix: %s, slug: %s",
		"cli.demote.broken_count":     "%d link(s) will break. Fix them with engram mv",
		"cli.demote.derived_count":    "derived_context of %d source original(s) has drifted",
		"cli.demote.next":             "Next: once tidied up, promote again with engram promote",
		"cli.archive.short":           "Archive a context document whose life has ended",
		"cli.archive.long": `Moves a context stage document to the archive directory.

It retires documents whose life has ended without breaking links. The slug
is unchanged and no date prefix is added, so wikilinks pointing at the
document stay as they are. Links to retired decisions must not break.
If the promotion was wrong, use engram demote instead.

Sets artifact_stage to archive and status to archived in the frontmatter and
refreshes updated. If links point at the document, their count is reported.
Nothing is blocked.`,
		"cli.archive.already_archived":  "already archived: %s",
		"cli.archive.not_context":       "only context stage documents can be archived: artifact_stage of %s is %q\nThe target is a context document whose life has ended. If the promotion was wrong, use engram demote",
		"cli.archive.dest_exists":       "destination already has a document: %s\nExisting documents are not overwritten. Tidy up the archive directory and try again",
		"cli.archive.notice_incoming":   "Note: %d link(s) point at this document. The links stay intact while their target becomes retired",
		"cli.archive.remove_fail":       "cannot delete the pre-archive original: %s",
		"cli.archive.done":              "Archived: %s",
		"cli.archive.slug_kept":         "The slug %s is kept. Incoming links do not break",
		"cli.archive.incoming_now_dead": "%d link(s) pointing at this document now point at a retired document",
		"cli.mv.short":                  "Rename a document slug and fix every affected link",
		"cli.mv.long": `Renames a document's slug and fixes every link pointing at it.

Display text and headings of body wikilinks are preserved; only the slug
changes. The related, derived_from, derived_context and source_refs fields
are fixed as well. Link syntax inside code fences and inline code is left alone.

The date prefix rule is kept. If the original had a prefix it is preserved
and only the slug part changes. The new slug is normalized by the slug rule (ADR 0020).

Links are all fixed first and the file move comes last. If it fails midway,
links end up pointing at the old name, and re-running mv recovers.

--dry-run only reports what would change and writes nothing.`,
		"cli.mv.flag_dry_run":     "Only report what would change and write nothing",
		"cli.mv.old_empty":        "old slug is empty: %q",
		"cli.mv.same_slug":        "new slug is the same as the old slug: %s",
		"cli.mv.old_not_found":    "no document matches the old slug: %s\nPass the document path or slug directly. Example: engram mv note memo",
		"cli.mv.slug_taken":       "new slug is already in use: %s\nExisting documents are not overwritten. Choose a different slug",
		"cli.mv.dest_exists":      "destination already has a document: %s\nExisting documents are not overwritten",
		"cli.mv.rename_fail":      "cannot move document to %s",
		"cli.mv.link_read_fail":   "cannot read link document: %s",
		"cli.mv.link_parse_fail":  "cannot parse link document: %s",
		"cli.mv.link_write_fail":  "cannot write link document: %s",
		"cli.mv.verb_moved":       "Moved",
		"cli.mv.verb_will_change": "Would rename",
		"cli.mv.summary":          "%s: %s to %s (slug %s)",
		"cli.mv.no_links":         "No links to fix",
		"cli.mv.links_total":      "Fixed %d link(s):",
		"cli.mv.links_per_file":   "  %s: %d",
		"cli.mv.rejections_fixed": "Also renamed the slugs of %d bridge rejection pair(s)",
		"cli.mv.dry_run_note":     "Dry run: nothing was written",
		"cli.mv.next":             "Next: check link integrity with engram lint",
		"cli.update.short":        "Update a document's frontmatter and body",
		"cli.update.long": `Updates a document's frontmatter keys and body.

--set key=value sets a key. Repeatable. Array attributes take comma-separated
values. Example: --set topics=go,cli
--unset key removes a key. Repeatable.
--body-from <file> replaces the whole body. Pass - to read standard input.

Keys of disabled attributes and values outside the allowed set are rejected. artifact_stage
cannot be changed here. Stage moves belong to engram promote and engram demote.
Key order is kept exactly as parsing preserved it.`,
		"cli.update.flag_set":              "Set a frontmatter key. key=value. Repeatable",
		"cli.update.flag_unset":            "Remove a frontmatter key. Repeatable",
		"cli.update.flag_body_from":        "Replace the whole body. File path or - (standard input)",
		"cli.update.no_changes":            "Nothing to update\nUse one of --set key=value, --unset key, --body-from <file|->",
		"cli.update.set_invalid":           "--set value is not in key=value form: %q",
		"cli.update.unset_stage_forbidden": "artifact_stage cannot be removed with update. Stage moves belong to engram promote and engram demote",
		"cli.update.unset_axis_off":        "nothing to unset for a disabled attribute: %s",
		"cli.update.warn_sources_body":     "Warning: sources is a preservation layer. Remember that the body was changed",
		"cli.update.write_fail":            "cannot write document: %s",
		"cli.update.set_stage_forbidden":   "artifact_stage cannot be changed with update. Stage moves belong to engram promote and engram demote",
		"cli.update.set_axis_off":          "cannot set a disabled attribute: %s (preset %s). Turn it on under axes in engram.yaml, or stop using it in documents",
		"cli.update.set_empty_item":        "--set %s=%s value contains an empty item",
		"cli.update.set_not_bool":          "--set %s=%s value is not a boolean (true or false)",
		"cli.update.set_not_allowed":       "--set %s=%s value is not allowed (allowed: %s)",
		"cli.update.body_read_fail":        "cannot read body file: %s",
		"cli.update.done":                  "Updated: %s",
		"cli.update.set_line":              "Set: %s",
		"cli.update.unset_line":            "Removed: %s",
		"cli.update.body_line":             "Body replaced: %s",
		"cli.update.updated_line":          "Recorded the update date %s in updated",
		"cli.update.next":                  "Next: check the updated document's schema with engram lint",
	})
}
