// Package i18n의 cli_query 카탈로그. ADR 0049.
package i18n

func init() {
	Register(LangKO, map[string]string{
		// search
		"cli.search.short": "위키를 검색합니다",
		"cli.search.long": `검색 색인으로 위키를 검색합니다.

색인이 신선하면 색인으로 검색합니다. 낡았으면 낡은 색인 그대로 검색하되
경고를 냅니다. 색인이 없거나 깨졌으면 이번 실행에서만 문서를 읽어 검색합니다.
어느 쪽이든 색인 파일을 쓰지 않습니다. 색인을 만드는 것은 reindex뿐입니다.`,
		"cli.search.flag_read_fail":      "--%s 플래그를 읽을 수 없음",
		"cli.search.walk_fail":           "위키를 순회할 수 없음",
		"cli.search.warn_stale":          "경고: 색인이 낡았습니다. 낡은 색인으로 검색합니다. engram reindex로 갱신하세요",
		"cli.search.index_build_fail":    "즉석 색인을 만들 수 없음",
		"cli.search.notice_memory_index": "안내: 색인이 없어 이번 실행에서만 문서를 읽었습니다. engram reindex를 한 번 돌리면 검색이 빨라집니다",
		"cli.search.flag_limit":          "결과 상한",
		"cli.search.flag_wiki":           "대상 위키 경로",
		"cli.search.no_results":          "결과가 없습니다",
		"cli.search.stage_root":          "루트",
		"cli.search.query_tokens":        "질의 %q는 다음 토큰으로 검색했습니다: %s",

		// recall
		"cli.recall.short": "질의에 맞는 원문 조각을 출처와 함께 냅니다",
		"cli.recall.long": `검색 색인으로 질의에 맞는 문서를 찾고 헤딩 단위로 자른 원문 조각을 냅니다.

각 조각에는 슬러그, 헤딩 경로, 줄 범위, 원문이 함께 나오므로
조각을 컨텍스트에 넣고 [[슬러그]]로 인용할 수 있습니다.
요약을 만들지 않고 원문만 냅니다. 문서 목록이 필요하면 engram search를 쓰세요.

조회 커맨드이므로 색인 파일을 쓰지 않습니다. 색인이 없으면
engram reindex를 먼저 돌리라고 안내하고 종료합니다.`,
		"cli.recall.flag_read_fail": "--%s 플래그를 읽을 수 없음",
		"cli.recall.no_index":       "검색 색인이 없습니다: %s\nengram reindex를 먼저 실행하세요",
		"cli.recall.walk_fail":      "위키를 순회할 수 없음",
		"cli.recall.warn_stale":     "경고: 색인이 낡았습니다. 낡은 색인으로 검색합니다. engram reindex로 갱신하세요",
		"cli.recall.flag_limit":     "낼 조각 수",
		"cli.recall.flag_wiki":      "대상 위키 경로",
		"cli.recall.no_results":     "결과가 없습니다",
		"cli.recall.query_tokens":   "질의 %q는 다음 토큰으로 검색했습니다: %s",

		// backlinks
		"cli.backlinks.short": "슬러그를 가리키는 링크를 조회합니다",
		"cli.backlinks.long": `슬러그를 가리키는 링크를 조회합니다.

본문 링크와 related 필드와 관계 필드(derived_from, derived_context,
source_refs)를 종류별로 구분해 보여줍니다. 슬러그에 해당하는 문서가
위키에 없어도 동작합니다. 깨진 링크를 찾는 것이 이 커맨드의 용도 중 하나입니다.`,
		"cli.backlinks.walk_fail":      "위키를 순회할 수 없음",
		"cli.backlinks.flag_wiki":      "대상 위키 경로",
		"cli.backlinks.missing_notice": "알림: 슬러그 %q에 해당하는 문서가 위키에 없습니다. 아래는 깨진 링크입니다",
		"cli.backlinks.no_backlinks":   "백링크가 없습니다. 고아 문서일 수 있습니다",

		// lint
		"cli.lint.short": "위키의 스키마와 링크 무결성을 검사합니다",
		"cli.lint.long": `지정한 경로의 위키를 순회하며 스키마와 링크 무결성을 검사합니다.
경로를 생략하면 현재 디렉토리입니다.

등급은 error, warn, reject 셋입니다. error와 reject는 승급을 막아
종료 코드 1로 끝나고 warn은 통과시키되 알립니다.

기본 검사 범위는 inbox 디렉토리를 뺀 나머지입니다. inbox 문서도
검사하려면 --include-inbox를 주세요. 링크 그래프와 승급 게이트는
어느 쪽이든 inbox를 포함합니다.`,
		"cli.lint.config_read_fail":            "위키 설정을 읽을 수 없음",
		"cli.lint.blocking_violation":          "승급을 막는 위반이 있습니다",
		"cli.lint.flag_include_inbox":          "inbox 디렉토리 문서의 스키마 판정을 함께 돌립니다",
		"cli.lint.fix":                         "    고치는 법: %s",
		"cli.lint.wiki_findings_header":        "위키 진단:",
		"cli.lint.wiki_finding_line":           "  [%s] %s 주제 %q",
		"cli.lint.wiki_finding_ratio":          "    사용 비율이 %d%%(%d/%d 문서)로 broad_topic_pct %d를 넘습니다",
		"cli.lint.wiki_finding_paths":          "    해당 문서: %s",
		"cli.lint.summary_clean":               "검사한 파일 %d개, 위반 없음",
		"cli.lint.summary":                     "검사한 파일 %d개, error %d, warn %d, reject %d",
		"cli.lint.summary_clean_inbox_skipped": "검사한 파일 %d개, inbox 문서 %d개를 건너뛰었습니다(--include-inbox로 검사합니다), 위반 없음",
		"cli.lint.summary_inbox_skipped":       "검사한 파일 %d개, inbox 문서 %d개를 건너뛰었습니다(--include-inbox로 검사합니다), error %d, warn %d, reject %d",
		"cli.lint.more_paths":                  " 외 %d개",

		// status
		"cli.status.short": "위키 현황과 밀린 것을 보여줍니다",
		"cli.status.long": `대상 경로의 위키 현황과 아직 처리하지 않은 것을 보여줍니다. 경로를 생략하면 현재 디렉토리입니다.

현황, 밀린 것, 다음 행동 세 구획으로 나눠 냅니다.
나이 계산의 기준 시각은 전역 --now 입니다.

위키가 아닌 디렉토리에서는 거절하고 engram init 을 안내합니다.`,
		"cli.status.section_overview":      "현황",
		"cli.status.overview_counts":       "  inbox %d, source %d, context %d, archive %d 문서",
		"cli.status.overview_links":        "  위키링크 %d개, 고아 문서 %d개",
		"cli.status.overview_lint":         "  lint: 파일 %d개, error %d, warn %d, reject %d (상세는 engram lint)",
		"cli.status.section_backlog":       "밀린 것 (기준 %s)",
		"cli.status.backlog_inbox":         "  inbox 문서 %d개",
		"cli.status.backlog_inbox_today":   "  inbox 문서 %d개, 전부 오늘 들어왔습니다",
		"cli.status.backlog_inbox_unknown": "  inbox 문서 %d개, 가장 오래된 것 알 수 없음",
		"cli.status.backlog_inbox_oldest":  "  inbox 문서 %d개, 가장 오래된 것은 %d일 전",
		"cli.status.backlog_unknown_age":   "  날짜를 알 수 없는 inbox 문서 %d개",
		"cli.status.backlog_stale":         "  stale_days를 넘긴 context 문서 %d개",
		"cli.status.backlog_promotable":    "  지금 승급할 수 있는 문서 %d개",
		"cli.status.section_next":          "다음 행동",
		"cli.status.no_suggestions":        "  제안 없음",

		// doctor
		"cli.doctor.short": "환경과 위키 설정을 진단합니다",
		"cli.doctor.long": `대상 경로의 환경과 위키 설정을 진단합니다. 경로를 생략하면 현재 디렉토리입니다.

위키가 아닌 디렉토리에서는 환경 항목만 검사합니다.
각 항목은 상태(ok, warn, fail, skip)와 관측값을 한 줄로 출력하고
ok 가 아닌 항목에는 조치를 이어서 출력합니다.

fail 항목이 하나라도 있으면 종료 코드 1로 끝납니다. warn은 0입니다.`,
		"cli.doctor.has_fail": "진단 실패 항목이 있습니다",
		"cli.doctor.fix":      "    조치: %s",
		"cli.doctor.summary":  "항목 %d개, ok %d, warn %d, fail %d, skip %d",

		// resurface
		"cli.resurface.short": "오래 안 본 context 문서를 다시 꺼냅니다",
		"cli.resurface.long": `stale_days를 넘긴 context 문서를 골라 다시 보여줍니다.

제시 이력을 <위키>/.engram/resurface.json에 남겨 최근에 보여준 문서를
후보에서 뺍니다. 이 파일은 gitignore 대상이고 없어도 빈 이력으로
동작하므로 지워도 도구가 멈추지 않습니다.
순서는 점수 내림차순이고 동점은 슬러그 오름차순입니다. 점수는 경과일에
인바운드 링크 수의 역수를 곱한 값이라 아무도 안 가리키는 문서가 먼저 나옵니다.
최근 21일 안에 제시한 문서는 후보에서 빼되 후보가 모자라면 다시 넣습니다.
아무도 안 가리키는 문서는 따로 셉니다.
상태를 쓰는 유일한 조회 커맨드라 실행마다 결과가 달라지므로
--now로 기준 시각을 고정할 수 있습니다.
--dry-run은 이력을 기록하지 않습니다.`,
		"cli.resurface.flag_read_fail":    "--%s 플래그를 읽을 수 없음",
		"cli.resurface.flag_limit":        "낼 문서 수",
		"cli.resurface.flag_dry_run":      "제시 이력을 기록하지 않습니다",
		"cli.resurface.flag_wiki":         "대상 위키 경로",
		"cli.resurface.no_candidates":     "다시 꺼낼 문서가 없습니다",
		"cli.resurface.reason":            "  이유: %s",
		"cli.resurface.header":            "다시 꺼낼 문서 %d개 (stale_days %d일, 기준 %s)",
		"cli.resurface.never_shown":       "제시한 적 없음",
		"cli.resurface.last_shown":        "마지막 제시 %s",
		"cli.resurface.candidate_line":    "  - %s%s: 마지막 갱신 %d일 전, %s",
		"cli.resurface.cooldown_filled":   "후보가 모자라 최근 %[2]d일 안에 제시한 문서 %[1]d개를 다시 넣었습니다",
		"cli.resurface.no_inbound_header": "아무도 안 가리키는 문서 %d개",
		"cli.resurface.no_inbound_line":   "  - %s%s",
		"cli.resurface.skipped_no_date":   "기준 날짜를 알 수 없는 context 문서 %d개는 대상에서 뺐습니다",
		"cli.resurface.dry_run_note":      "--dry-run이라 제시 이력을 기록하지 않았습니다",

		// bridge
		"cli.bridge.short": "유사한데 링크가 없는 문서 쌍을 찾습니다",
		"cli.bridge.long": `검색 색인의 TF 벡터로 context 문서끼리 코사인 유사도를 재고,
유사도가 높은데 링크가 없는 쌍을 보여줍니다.

후보에서 기각한 쌍은 engram-state.yaml 에 영구 기록되어 다시 나오지 않습니다.
--reject 로 기각하고 --unreject 로 되돌립니다. 기각 기록은 git 이 추적합니다.

색인이 없으면 진행하지 않습니다. engram reindex 로 색인을 만드세요.
낡은 색인은 경고를 내고 그대로 진행합니다.`,
		"cli.bridge.flag_read_fail":     "--%s 플래그를 읽을 수 없음",
		"cli.bridge.no_positional_args": "bridge 는 위치 인자를 받지 않습니다. 기각은 engram bridge --reject <A> <B> 로 합니다",
		"cli.bridge.both_flags":         "--reject 와 --unreject 를 함께 쓸 수 없습니다. 한 번에 한 동작만 고르세요",
		"cli.bridge.reject_needs_two":   "--reject 는 슬러그 두 개를 받습니다. 지금 %d개입니다: %s",
		"cli.bridge.unreject_needs_two": "--unreject 는 슬러그 두 개를 받습니다. 지금 %d개입니다: %s",
		"cli.bridge.no_index":           "검색 색인이 없습니다. engram reindex 로 색인을 만든 뒤 다시 실행하세요",
		"cli.bridge.walk_fail":          "위키를 순회할 수 없음",
		"cli.bridge.warn_stale":         "경고: 색인이 낡았습니다. 낡은 색인으로 진행합니다. engram reindex로 갱신하세요",
		"cli.bridge.flag_min":           "코사인 유사도 하한",
		"cli.bridge.flag_limit":         "낼 쌍 수 상한",
		"cli.bridge.flag_reject":        "기각할 슬러그 둘 (예: --reject a b)",
		"cli.bridge.flag_unreject":      "기각을 되돌릴 슬러그 둘",
		"cli.bridge.flag_wiki":          "대상 위키 경로",
		"cli.bridge.reject_missing":     "위키에 없는 슬러그라 기각하지 못했습니다: %s\n슬러그를 확인하세요. 문서를 찾으려면 engram search 를 쓰세요",
		"cli.bridge.already_rejected":   "이미 기각된 쌍입니다: %s %s",
		"cli.bridge.reject_save_fail":   "기각을 저장할 수 없음",
		"cli.bridge.rejected":           "기각했습니다: %s %s\n%s 에 기록했습니다",
		"cli.bridge.not_rejected":       "기각 기록에 없는 쌍입니다: %s %s",
		"cli.bridge.unreject_save_fail": "기각 되돌리기를 저장할 수 없음",
		"cli.bridge.unrejected":         "기각을 되돌렸습니다: %s %s",
		"cli.bridge.no_pairs":           "후보가 없습니다",
		"cli.bridge.stats":              "  context 문서 %d개, 링크로 이어진 쌍 %d, 기각된 쌍 %d, min %.2f 미달 %d",
		"cli.bridge.header":             "유사도가 높은데 링크가 없는 문서 쌍 (min %.2f)",
		"cli.bridge.reject_hint":        "     기각하려면: engram bridge --reject %s %s",

		// digest
		"cli.digest.short": "기간 안의 위키 변화를 집계합니다",
		"cli.digest.long": `기간 안의 위키 변화를 집계합니다. 상태를 남기지 않으므로 몇 번을 돌려도 같은 결과가 나옵니다.

--days로 기간을 정합니다. 창은 [기준 시각 - N일, 기준 시각]이고 기준
시각은 전역 --now 다. 신규는 created가 창 안에 있는 문서, 오래된 문서는
stale_days를 넘긴 context 문서, 고아는 링크가 0개인 문서입니다.
승급 집계는 promote가 승급 시각을 프론트매터에 남기지 않아 여기에
없습니다.`,
		"cli.digest.flag_read_fail": "--%s 플래그를 읽을 수 없음",
		"cli.digest.days_negative":  "--%s 값은 0 이상이어야 합니다: %d",
		"cli.digest.flag_days":      "집계 기간(일)",
		"cli.digest.flag_wiki":      "대상 위키 경로",
		"cli.digest.header":         "기간 집계 (%s부터 %s까지, %d일)",
		"cli.digest.created":        "  신규 %d개%s",
		"cli.digest.stale":          "  오래된 문서 %d개%s",
		"cli.digest.orphans":        "  고아 %d개%s",
		"cli.digest.more_slugs":     " 외 %d개",

		// gate (promote와 new가 함께 쓰는 게이트 보조)
		"cli.gate.reject": "승급 게이트를 넘지 못했습니다: 이어지는 위키링크가 %d개로 min_wikilinks %d개에 못 미칩니다\n" +
			"위키에 있는 문서를 가리키는 링크를 %d개 더 추가하세요. 없는 슬러그와 inbox 문서는 세지 않습니다\n이 자리에서 채우려면 --related <슬러그>를 반복해 주세요",
		"cli.gate.deferred":        "경고: 링크 대상 문서가 %d개로 min_wikilinks %d개보다 적어 게이트를 유예했습니다. 위키가 자라면 게이트가 다시 적용됩니다",
		"cli.gate.unknown_related": "경고: --related 슬러그 %q에 해당하는 문서가 위키에 없습니다. 곧 만들 문서일 수 있습니다",
		"cli.gate.mkdir_fail":      "디렉토리를 만들 수 없음",
		"cli.gate.dest_exists":     "도착지에 이미 문서가 있습니다: %s\n기존 문서를 덮어쓰지 않습니다. 슬러그를 다르게 지정하세요",
		"cli.gate.create_fail":     "문서를 만들 수 없음: %s",
		"cli.gate.write_fail":      "문서를 쓸 수 없음: %s",
	})

	Register(LangEN, map[string]string{
		// search
		"cli.search.short": "Search the wiki",
		"cli.search.long": `Search the wiki with the search index.

With a fresh index, search runs on the index. When the index is stale, search
still uses it and prints a warning. With no index or a broken one, documents
are read for this run only. Either way no index file is written. Only reindex
builds the index.`,
		"cli.search.flag_read_fail":      "cannot read --%s flag",
		"cli.search.walk_fail":           "cannot walk the wiki",
		"cli.search.warn_stale":          "warning: the index is stale. Searching with the stale index. Run engram reindex to refresh",
		"cli.search.index_build_fail":    "cannot build an in-memory index",
		"cli.search.notice_memory_index": "note: no index found, documents were read for this run only. Run engram reindex once to speed up search",
		"cli.search.flag_limit":          "result limit",
		"cli.search.flag_wiki":           "target wiki path",
		"cli.search.no_results":          "No results",
		"cli.search.stage_root":          "root",
		"cli.search.query_tokens":        "Query %q was searched with these tokens: %s",

		// recall
		"cli.recall.short": "Return original passages matching the query, with sources",
		"cli.recall.long": `Find documents matching the query with the search index and return original
passages cut by heading.

Each passage comes with its slug, heading path, line range, and original text,
so it can be placed in context and cited as [[slug]].
No summary is generated; only the original text is returned. Use engram search
for a document list.

As a read command it writes no index file. Without an index it asks you to run
engram reindex first and exits.`,
		"cli.recall.flag_read_fail": "cannot read --%s flag",
		"cli.recall.no_index":       "no search index: %s\nRun engram reindex first",
		"cli.recall.walk_fail":      "cannot walk the wiki",
		"cli.recall.warn_stale":     "warning: the index is stale. Searching with the stale index. Run engram reindex to refresh",
		"cli.recall.flag_limit":     "number of passages to return",
		"cli.recall.flag_wiki":      "target wiki path",
		"cli.recall.no_results":     "No results",
		"cli.recall.query_tokens":   "Query %q was searched with these tokens: %s",

		// backlinks
		"cli.backlinks.short": "Look up links pointing to a slug",
		"cli.backlinks.long": `Look up links pointing to a slug.

Body links, the related field, and relation fields (derived_from,
derived_context, source_refs) are shown by kind. It works even when no document
in the wiki matches the slug. Finding broken links is one use of this command.`,
		"cli.backlinks.walk_fail":      "cannot walk the wiki",
		"cli.backlinks.flag_wiki":      "target wiki path",
		"cli.backlinks.missing_notice": "note: no document in the wiki matches slug %q. Below are broken links",
		"cli.backlinks.no_backlinks":   "No backlinks. This document may be an orphan",

		// lint
		"cli.lint.short": "Check the wiki for schema and link integrity",
		"cli.lint.long": `Walk the wiki at the given path and check schema and link integrity.
Without a path, the current directory is used.

Severities are error, warn, and reject. error and reject block promotion and
end with exit code 1; warn passes with a notice.

The default scope excludes the inbox directory. Pass --include-inbox to also
check inbox documents. The link graph and the promotion gate include inbox
either way.`,
		"cli.lint.config_read_fail":            "cannot read wiki config",
		"cli.lint.blocking_violation":          "There are violations that block promotion",
		"cli.lint.flag_include_inbox":          "also judge inbox documents against the schema",
		"cli.lint.fix":                         "    Fix: %s",
		"cli.lint.wiki_findings_header":        "Wiki findings:",
		"cli.lint.wiki_finding_line":           "  [%s] %s topic %q",
		"cli.lint.wiki_finding_ratio":          "    Usage %d%% (%d/%d documents) exceeds broad_topic_pct %d",
		"cli.lint.wiki_finding_paths":          "    Documents: %s",
		"cli.lint.summary_clean":               "Files checked: %d, no violations",
		"cli.lint.summary":                     "Files checked: %d, error %d, warn %d, reject %d",
		"cli.lint.summary_clean_inbox_skipped": "Files checked: %d, skipped %d inbox documents (check with --include-inbox), no violations",
		"cli.lint.summary_inbox_skipped":       "Files checked: %d, skipped %d inbox documents (check with --include-inbox), error %d, warn %d, reject %d",
		"cli.lint.more_paths":                  " and %d more",

		// status
		"cli.status.short": "Show wiki status and what is waiting",
		"cli.status.long": `Show wiki status and what is still waiting at the given path. Without a path,
the current directory is used.

Output is split into three sections: status, backlog, and next steps.
The reference time for age calculation is the global --now.

In a directory that is not a wiki, the command refuses and points to engram init.`,
		"cli.status.section_overview":      "Overview",
		"cli.status.overview_counts":       "  inbox %d, source %d, context %d, archive %d documents",
		"cli.status.overview_links":        "  wikilinks: %d, orphan documents: %d",
		"cli.status.overview_lint":         "  lint: %d files, error %d, warn %d, reject %d (details: engram lint)",
		"cli.status.section_backlog":       "Backlog (as of %s)",
		"cli.status.backlog_inbox":         "  inbox documents: %d",
		"cli.status.backlog_inbox_today":   "  inbox documents: %d, all arrived today",
		"cli.status.backlog_inbox_unknown": "  inbox documents: %d, oldest age unknown",
		"cli.status.backlog_inbox_oldest":  "  inbox documents: %d, oldest arrived %d days ago",
		"cli.status.backlog_unknown_age":   "  inbox documents with unknown date: %d",
		"cli.status.backlog_stale":         "  context documents past stale_days: %d",
		"cli.status.backlog_promotable":    "  documents promotable now: %d",
		"cli.status.section_next":          "Next steps",
		"cli.status.no_suggestions":        "  No suggestions",

		// doctor
		"cli.doctor.short": "Diagnose the environment and wiki config",
		"cli.doctor.long": `Diagnose the environment and wiki config at the given path. Without a path,
the current directory is used.

In a directory that is not a wiki, only environment checks run.
Each check prints its status (ok, warn, fail, skip) and observation on one line,
and anything not ok is followed by an action.

Any fail ends with exit code 1. warn stays 0.`,
		"cli.doctor.has_fail": "There are failed checks",
		"cli.doctor.fix":      "    Action: %s",
		"cli.doctor.summary":  "Checks: %d, ok %d, warn %d, fail %d, skip %d",

		// resurface
		"cli.resurface.short": "Resurface context documents not seen for a while",
		"cli.resurface.long": `Pick context documents past stale_days and show them again.

Presentation history is kept in <wiki>/.engram/resurface.json so recently
shown documents are dropped from the candidates. The file is gitignored and works as an empty
history when missing, so deleting it never breaks the tool.
Ordered by score descending, ties by slug ascending. The score is the age in
days times the reciprocal of the inbound link count, so documents nobody points
to come first. Documents shown within the last 21 days are excluded from the
candidates, but they are put back when there are not enough candidates.
Documents nobody points to are counted separately.
As the only query command that writes state, results change between runs;
pin the reference time with --now.
--dry-run records no history.`,
		"cli.resurface.flag_read_fail":    "cannot read --%s flag",
		"cli.resurface.flag_limit":        "number of documents to return",
		"cli.resurface.flag_dry_run":      "do not record presentation history",
		"cli.resurface.flag_wiki":         "target wiki path",
		"cli.resurface.no_candidates":     "No documents to resurface",
		"cli.resurface.reason":            "  Reason: %s",
		"cli.resurface.header":            "Documents to resurface: %d (stale_days %d days, as of %s)",
		"cli.resurface.never_shown":       "never shown",
		"cli.resurface.last_shown":        "last shown %s",
		"cli.resurface.candidate_line":    "  - %s%s: last updated %d days ago, %s",
		"cli.resurface.cooldown_filled":   "Refilled %[1]d documents shown within the last %[2]d days because there were not enough candidates",
		"cli.resurface.no_inbound_header": "Documents nobody points to: %d",
		"cli.resurface.no_inbound_line":   "  - %s%s",
		"cli.resurface.skipped_no_date":   "Excluded %d context documents whose date is unknown",
		"cli.resurface.dry_run_note":      "--dry-run: presentation history was not recorded",

		// bridge
		"cli.bridge.short": "Find similar document pairs without links",
		"cli.bridge.long": `Compute cosine similarity between context documents from the TF vectors of the
search index and show pairs with high similarity but no link.

Pairs rejected from the candidates are recorded permanently in
engram-state.yaml and never come back. Reject with --reject and undo with
--unreject. The rejection record is tracked by git.

Without an index the command stops. Build one with engram reindex.
A stale index warns and proceeds as is.`,
		"cli.bridge.flag_read_fail":     "cannot read --%s flag",
		"cli.bridge.no_positional_args": "bridge does not accept positional arguments. To reject, use engram bridge --reject <A> <B>",
		"cli.bridge.both_flags":         "--reject and --unreject cannot be used together. Pick one action at a time",
		"cli.bridge.reject_needs_two":   "--reject takes two slugs. Got %d: %s",
		"cli.bridge.unreject_needs_two": "--unreject takes two slugs. Got %d: %s",
		"cli.bridge.no_index":           "no search index. Build one with engram reindex, then run again",
		"cli.bridge.walk_fail":          "cannot walk the wiki",
		"cli.bridge.warn_stale":         "warning: the index is stale. Proceeding with the stale index. Run engram reindex to refresh",
		"cli.bridge.flag_min":           "cosine similarity lower bound",
		"cli.bridge.flag_limit":         "pair limit",
		"cli.bridge.flag_reject":        "two slugs to reject (e.g. --reject a b)",
		"cli.bridge.flag_unreject":      "two slugs to unreject",
		"cli.bridge.flag_wiki":          "target wiki path",
		"cli.bridge.reject_missing":     "cannot reject: slug not in the wiki: %s\nCheck the slug. Use engram search to find documents",
		"cli.bridge.already_rejected":   "Pair already rejected: %s %s",
		"cli.bridge.reject_save_fail":   "cannot save the rejection",
		"cli.bridge.rejected":           "Rejected: %s %s\nRecorded in %s",
		"cli.bridge.not_rejected":       "Pair not in the rejection record: %s %s",
		"cli.bridge.unreject_save_fail": "cannot save the unrejection",
		"cli.bridge.unrejected":         "Unrejected: %s %s",
		"cli.bridge.no_pairs":           "No candidates",
		"cli.bridge.stats":              "  context documents: %d, linked pairs: %d, rejected pairs: %d, below min %.2f: %d",
		"cli.bridge.header":             "Similar document pairs without links (min %.2f)",
		"cli.bridge.reject_hint":        "     To reject: engram bridge --reject %s %s",

		// digest
		"cli.digest.short": "Summarize wiki changes over a period",
		"cli.digest.long": `Summarize wiki changes within a period. No state is kept, so repeated runs
give the same result.

--days sets the period. The window is [reference time - N days, reference time],
and the reference time is the global --now. New means documents whose created
falls inside the window, stale means context documents past stale_days, and
orphans mean documents with zero links.
Promotion counts are absent because promote does not record promotion time in
frontmatter.`,
		"cli.digest.flag_read_fail": "cannot read --%s flag",
		"cli.digest.days_negative":  "--%s must be 0 or greater: %d",
		"cli.digest.flag_days":      "period in days",
		"cli.digest.flag_wiki":      "target wiki path",
		"cli.digest.header":         "Period summary (%s to %s, %d days)",
		"cli.digest.created":        "  new: %d%s",
		"cli.digest.stale":          "  stale: %d%s",
		"cli.digest.orphans":        "  orphans: %d%s",
		"cli.digest.more_slugs":     " and %d more",

		// gate (promote와 new가 함께 쓰는 게이트 보조)
		"cli.gate.reject": "Promotion gate not passed: %d resolving wikilinks fall short of min_wikilinks %d\n" +
			"Add %d more wikilinks in the related field or the body.\nTo add them here, repeat --related <slug>",
		"cli.gate.deferred":        "warning: linkable target documents: %d, fewer than min_wikilinks %d. The gate was deferred and applies again as the wiki grows",
		"cli.gate.unknown_related": "warning: no document in the wiki matches --related slug %q. It may be a document you are about to create",
		"cli.gate.mkdir_fail":      "cannot create the directory",
		"cli.gate.dest_exists":     "a document already exists at the destination: %s\nExisting documents are not overwritten. Pick a different slug",
		"cli.gate.create_fail":     "cannot create the document: %s",
		"cli.gate.write_fail":      "cannot write the document: %s",
	})
}
