// Package i18n의 cli_net 카탈로그. mcp, serve, export 커맨드의
// 사용자 대면 문자열. ADR 0049.
package i18n

func init() {
	Register(LangKO, map[string]string{
		// mcp 커맨드
		"cli.mcp.short": "위키를 MCP 서버로 노출합니다",
		"cli.mcp.long": `위키를 MCP 서버로 stdio 로 노출합니다.

도구는 열이고 쓰기는 capture 하나뿐이다. capture 는 inbox 에만 쓴다.
승급(promote, demote, archive)은 사람이 확정하므로 도구로 내보내지
않는다. 이것이 다른 PKM MCP 와의 차별점이다.

위키 경로는 --wiki 로 시작 시 고정한다. 도구는 경로를 인자로 받지
않는다. 도구 인자로 경로를 받으면 에이전트가 임의의 디렉토리를 읽을
수 있기 때문이다.

도구 결과는 각 커맨드의 --json 출력과 같은 구조다. 조회 도구는 재료를
낼 뿐 요약하지 않는다. 기준 시각은 실제 시계를 쓰고 --now 를 받지
않는다. 재현이 필요하면 CLI 를 쓰세요.`,
		"cli.mcp.starting":  "engram MCP 서버를 띄웁니다: %s",
		"cli.mcp.flag_wiki": "대상 위키 경로",
		"cli.mcp.tool_capture": "새 메모를 inbox 에 넣는다. 이 도구는 inbox 에만 쓴다. " +
			"context 로 올리는 승급은 게이트를 지나야 하고 확정은 사람이 한다. " +
			"승급을 제안하려면 사용자에게 engram promote 실행을 맡겨라.",
		"cli.mcp.tool_search":    "위키를 검색해 문서 목록을 재료로 낸다. 요약하지 않는다. 문장은 네가 만든다.",
		"cli.mcp.tool_recall":    "질의에 맞는 원문 조각을 출처와 함께 낸다. 요약하지 않는다. 조각을 인용해 네가 문장을 만든다.",
		"cli.mcp.tool_backlinks": "슬러그를 가리키는 링크를 본문 링크와 관계 필드로 구분해 낸다.",
		"cli.mcp.tool_status":    "위키 현황을 수치로 낸다. 단계별 문서 수, 링크 수, 고아 수, lint 요약.",
		"cli.mcp.tool_lint":      "규칙 위반 목록을 낸다. 규칙 ID, 등급, 고치는 법이 함께 나온다. 재료일 뿐 판단하지 않는다.",
		"cli.mcp.tool_rules": "이 위키에 지금 적용되는 규칙 전부를 낸다. 임계값과 허용값을 지어내지 말고 " +
			"이 도구로 얻어라. 위키마다 값이 다르다.",
		"cli.mcp.tool_resurface":      "오래 안 본 context 문서 후보를 재료로 낸다. 제시 이력을 쓰지 않는다.",
		"cli.mcp.tool_bridge":         "연결되지 않은 문서 쌍을 유사도와 함께 낸다. 기각을 기록하지 않는다.",
		"cli.mcp.tool_digest":         "기간 집계를 재료로 낸다. 신규 문서, 노후 문서, 고아 목록. 다이제스트 산문은 네가 쓴다.",
		"cli.mcp.index_missing_build": "검색 색인이 없습니다. engram reindex 로 색인을 만든 뒤 다시 실행하세요",
		"cli.mcp.index_missing_first": "검색 색인이 없습니다. engram reindex 를 먼저 실행하세요",

		// serve 커맨드
		"cli.serve.short": "위키를 읽기 전용 웹 뷰어로 노출합니다",
		"cli.serve.long": `위키를 읽기 전용 웹 뷰어로 노출합니다.

쓰기 경로가 없습니다. 제안 접수도 두지 않으므로 GET 과 HEAD 외의
요청은 전부 거절합니다.

검수를 지난 context 문서와 색인 문서만 보입니다. inbox 와 sources 는
목록에도 URL 에도 없습니다. archive 는 --include-archive 로 엽니다.
sensitivity 속성이 켜진 위키에서는 private-local-only 와 restricted 문서를
빼며 이 제외를 뒤집는 플래그는 없습니다. 내보내야 하면 문서의 값을
고치세요.

기본 바인딩은 127.0.0.1 입니다. --host 로 외부에 노출하면 인증이 없으므로
접근할 수 있는 모두가 위 문서를 읽습니다.`,
		"cli.serve.flag_wiki":                "대상 위키 경로",
		"cli.serve.flag_host":                "바인딩 주소",
		"cli.serve.flag_port":                "바인딩 포트",
		"cli.serve.flag_include_archive":     "archive 문서도 노출합니다",
		"cli.serve.flag_read_fail":           "--%s 플래그를 읽을 수 없음",
		"cli.serve.port_range":               "--%s 값이 포트 범위 밖입니다: %d (0부터 65535까지)",
		"cli.serve.listen_fail":              "주소 %s 를 열 수 없습니다",
		"cli.serve.notice_start":             "engram serve 를 시작합니다: http://%s",
		"cli.serve.notice_root":              "대상 위키: %s",
		"cli.serve.notice_exposed":           "노출: context 문서 %d개, 색인 문서 %d개",
		"cli.serve.notice_exposed_archive":   ", archive 문서 %d개",
		"cli.serve.notice_exposed_total":     " (모두 %d개)",
		"cli.serve.notice_excluded":          "제외: inbox %d개, sources %d개",
		"cli.serve.notice_excluded_archive":  ", archive %d개(--include-archive 로 엽니다)",
		"cli.serve.notice_excluded_outside":  "제외: 네 단계 밖에 있는 문서 %d개",
		"cli.serve.notice_excluded_unparsed": "제외: 프론트매터를 읽을 수 없어 판정하지 못한 문서 %d개 (engram lint 로 확인하세요)",
		"cli.serve.notice_sensitivity_on":    "민감도: private-local-only 와 restricted 문서 %d개를 제외했습니다. 뒤집는 플래그는 없습니다",
		"cli.serve.notice_sensitivity_off":   "민감도: 이 위키는 sensitivity 속성이 꺼져 있어 거를 값이 없습니다",
		"cli.serve.notice_readonly":          "쓰기 경로가 없습니다. GET 과 HEAD 외의 요청은 거절합니다",
		"cli.serve.notice_expose_warning":    "경고: %s 에 바인딩했습니다. 인증이 없으므로 이 주소에 닿는 모두가 위 문서를 읽습니다",
		"cli.serve.notice_stop":              "멈추려면 Ctrl+C 를 누르세요",

		// export 커맨드
		"cli.export.short": "검수된 문서를 익명화해 반출 번들로 내보냅니다",
		"cli.export.long": `검수를 지난 문서를 --out 디렉토리에 마크다운 그대로 내보냅니다.

병합도 압축도 포맷 변환도 하지 않습니다. 보고서나 발표자료로 만들려면
pandoc 같은 도구를 뒤에 붙이세요.

나가는 것은 serve 와 같은 규칙입니다. context 문서와 색인 문서만 나가고
inbox 와 sources 는 나가지 않습니다. archive 는 --include-archive 로
엽니다. sensitivity 속성이 켜진 위키에서는 private-local-only 와 restricted
문서를 뺍니다. 슬러그로 지목해도 이 제외는 뚫리지 않습니다. 반출해야
하면 문서의 값을 고치세요.

슬러그를 주면 그 문서만 나갑니다. 링크를 따라가지 않으므로 함께
내보낼 문서는 함께 적으세요.

익명화는 --replacements 파일로 합니다. 한 줄에 하나씩 원문==>대체어
형식이며 # 으로 시작하는 줄은 건너뜁니다. 본문과 프론트매터와 파일명
전부에 적용합니다. 파일을 주지 않으면 치환하지 않습니다.

--dry-run 은 무엇이 나갈지 봅니다. 파일을 쓰지 않습니다.`,
		"cli.export.flag_wiki":             "대상 위키 경로",
		"cli.export.flag_out":              "번들을 내보낼 디렉토리",
		"cli.export.flag_replacements":     "익명화 치환 파일. 한 줄에 원문==>대체어",
		"cli.export.flag_include_archive":  "archive 문서도 반출합니다",
		"cli.export.flag_dry_run":          "무엇이 나갈지 봅니다. 파일을 쓰지 않습니다",
		"cli.export.out_required":          "--%s 로 내보낼 디렉토리를 지정하세요",
		"cli.export.flag_read_fail":        "--%s 플래그를 읽을 수 없음",
		"cli.export.no_files":              "반출할 문서가 없습니다",
		"cli.export.repl_read_fail":        "치환 파일을 읽을 수 없음",
		"cli.export.repl_parse_fail":       "치환 파일 %s",
		"cli.export.repl_empty":            "치환 파일에 규칙이 없습니다: %s",
		"cli.export.outdir_check_fail":     "출력 디렉토리를 확인할 수 없음",
		"cli.export.outdir_not_empty":      "출력 디렉토리가 비어 있지 않습니다: %s",
		"cli.export.outdir_not_empty_hint": "이전 반출물이 섞이지 않도록 비우고 다시 실행하세요",
		"cli.export.mkdir_fail":            "디렉토리를 만들 수 없음",
		"cli.export.write_fail":            "번들 파일을 쓸 수 없음: %s",
		"cli.export.outcome_dryrun":        "내보낼 문서 %d개 (dry-run. 아직 쓰지 않았습니다)",
		"cli.export.outcome_done":          "내보냈습니다. 문서 %d개 -> %s",
		"cli.export.excluded_summary":      "제외: inbox %d개, sources %d개, archive %d개",
		"cli.export.excluded_filter":       "제외: 지목한 슬러그에 들지 않은 문서 %d개",
		"cli.export.excluded_outside":      "제외: 단계 디렉토리 밖에 있는 문서 %d개",
		"cli.export.excluded_unparsed":     "제외: 프론트매터를 읽을 수 없어 판정하지 못한 문서 %d개 (engram lint 로 확인하세요)",
		"cli.export.sensitivity_on":        "민감도: private-local-only 와 restricted 문서 %d개를 제외했습니다. 뒤집는 플래그는 없습니다",
		"cli.export.sensitivity_off":       "민감도: 이 위키는 sensitivity 속성이 꺼져 있어 거를 값이 없습니다",
		"cli.export.anonymized_count":      "익명화: %d건을 치환했습니다",
		"cli.export.unused_rules_warning":  "경고: 한 번도 걸리지 않은 치환 규칙이 %d건 있습니다. 사전의 오타를 확인하세요",
		"cli.export.anonymized_none":       "익명화: 치환 파일을 주지 않아 원문 그대로 나갑니다",
		"cli.export.dangling_links":        "번들 밖을 가리키는 위키링크 %d개 (문서 %d개). 본문은 고치지 않았습니다",
	})

	Register(LangEN, map[string]string{
		// mcp command
		"cli.mcp.short": "Expose the wiki as an MCP server",
		"cli.mcp.long": `Expose the wiki as an MCP server over stdio.

Tools are read-only and the only write is capture. capture writes to inbox
only. Promotion (promote, demote, archive) is confirmed by a human, so it
is not exposed as a tool. This is what sets it apart from other PKM MCP
servers.

The wiki path is fixed at startup with --wiki. Tools do not take a path
argument, because a path argument would let the agent read arbitrary
directories.

Tool results use the same structure as each command's --json output.
Query tools deliver material and do not summarize. The reference time is
the real clock; --now is not accepted. Use the CLI when you need
reproducibility.`,
		"cli.mcp.starting":  "Starting engram MCP server: %s",
		"cli.mcp.flag_wiki": "Target wiki path",
		"cli.mcp.tool_capture": "Put a new memo into inbox. This tool writes to inbox only. " +
			"Promotion to context must pass gates and is confirmed by a human. " +
			"To propose a promotion, ask the user to run engram promote.",
		"cli.mcp.tool_search":    "Search the wiki and return document hits as material. Do not summarize. You write the sentences.",
		"cli.mcp.tool_recall":    "Return source passages matching the query, with their sources. Do not summarize. Quote the passages and write your own sentences.",
		"cli.mcp.tool_backlinks": "Return links pointing to a slug, split into body links and relation fields.",
		"cli.mcp.tool_status":    "Return wiki status as numbers: document count per stage, link count, orphan count, lint summary.",
		"cli.mcp.tool_lint":      "Return rule violations: rule ID, grade, and how to fix each. Material only; it makes no judgment.",
		"cli.mcp.tool_rules": "Return every rule currently in effect for this wiki. Do not invent thresholds or allowed " +
			"values; obtain them with this tool. They differ per wiki.",
		"cli.mcp.tool_resurface":      "Return candidates among context documents not seen for a while, as material. Does not record presentation history.",
		"cli.mcp.tool_bridge":         "Return unconnected document pairs with similarity scores. Does not record rejections.",
		"cli.mcp.tool_digest":         "Return period statistics as material: new documents, aging documents, orphan list. You write the digest prose.",
		"cli.mcp.index_missing_build": "No search index. Run engram reindex to build one, then run again",
		"cli.mcp.index_missing_first": "No search index. Run engram reindex first",

		// serve command
		"cli.serve.short": "Serve the wiki as a read-only web viewer",
		"cli.serve.long": `Serve the wiki as a read-only web viewer.

There is no write path and no suggestion intake, so every request other
than GET and HEAD is rejected.

Only context documents that passed review and index documents are visible.
inbox and sources appear in neither listings nor URLs. archive opens with
--include-archive. In wikis with the sensitivity attribute on,
private-local-only and restricted documents are excluded, and no flag
reverses this exclusion. If a document must go out, change its value.

The default binding is 127.0.0.1. Exposing it with --host means no
authentication, so everyone who can reach the address reads the documents
above.`,
		"cli.serve.flag_wiki":                "Target wiki path",
		"cli.serve.flag_host":                "Bind address",
		"cli.serve.flag_port":                "Bind port",
		"cli.serve.flag_include_archive":     "Also expose archive documents",
		"cli.serve.flag_read_fail":           "cannot read --%s flag",
		"cli.serve.port_range":               "--%s value is outside the port range: %d (0 to 65535)",
		"cli.serve.listen_fail":              "cannot open address %s",
		"cli.serve.notice_start":             "Starting engram serve: http://%s",
		"cli.serve.notice_root":              "Target wiki: %s",
		"cli.serve.notice_exposed":           "Exposed: %d context documents, %d index documents",
		"cli.serve.notice_exposed_archive":   ", %d archive documents",
		"cli.serve.notice_exposed_total":     " (%d total)",
		"cli.serve.notice_excluded":          "Excluded: %d inbox, %d sources",
		"cli.serve.notice_excluded_archive":  ", %d archive (open with --include-archive)",
		"cli.serve.notice_excluded_outside":  "Excluded: %d documents outside the four stages",
		"cli.serve.notice_excluded_unparsed": "Excluded: %d documents whose frontmatter could not be parsed (check with engram lint)",
		"cli.serve.notice_sensitivity_on":    "Sensitivity: excluded %d private-local-only and restricted documents. No flag reverses this",
		"cli.serve.notice_sensitivity_off":   "Sensitivity: this wiki has the sensitivity attribute off, so there is nothing to filter",
		"cli.serve.notice_readonly":          "There is no write path. Requests other than GET and HEAD are rejected",
		"cli.serve.notice_expose_warning":    "Warning: bound to %s. There is no authentication, so everyone who reaches this address reads the documents above",
		"cli.serve.notice_stop":              "Press Ctrl+C to stop",

		// export command
		"cli.export.short": "Export reviewed documents, anonymized, as a release bundle",
		"cli.export.long": `Export documents that passed review as markdown, unchanged, into the
--out directory.

No merging, compression, or format conversion. Attach a tool like pandoc
afterward to build reports or slides.

What goes out follows the same rules as serve: context documents and index
documents go out, inbox and sources do not. archive opens with
--include-archive. In wikis with the sensitivity attribute on,
private-local-only and restricted documents are excluded. Naming a slug
does not punch through this exclusion. If a document must go out, change
its value.

Given slugs, only those documents go out. Links are not followed, so list
every document to export together.

Anonymization uses the --replacements file: one original==>replacement
per line; lines starting with # are skipped. It applies to body,
frontmatter, and file names alike. Without a file, nothing is replaced.

--dry-run shows what would go out. It writes no files.`,
		"cli.export.flag_wiki":             "Target wiki path",
		"cli.export.flag_out":              "Directory to export the bundle into",
		"cli.export.flag_replacements":     "Anonymization replacement file. One original==>replacement per line",
		"cli.export.flag_include_archive":  "Also export archive documents",
		"cli.export.flag_dry_run":          "Show what would go out. Writes no files",
		"cli.export.out_required":          "Specify the export directory with --%s",
		"cli.export.flag_read_fail":        "cannot read --%s flag",
		"cli.export.no_files":              "No documents to export",
		"cli.export.repl_read_fail":        "cannot read replacement file",
		"cli.export.repl_parse_fail":       "replacement file %s",
		"cli.export.repl_empty":            "replacement file has no rules: %s",
		"cli.export.outdir_check_fail":     "cannot inspect output directory",
		"cli.export.outdir_not_empty":      "output directory is not empty: %s",
		"cli.export.outdir_not_empty_hint": "Empty it and run again so earlier exports do not mix in",
		"cli.export.mkdir_fail":            "cannot create directory",
		"cli.export.write_fail":            "cannot write bundle file: %s",
		"cli.export.outcome_dryrun":        "%d documents to export (dry-run. Nothing written yet)",
		"cli.export.outcome_done":          "Exported. %d documents -> %s",
		"cli.export.excluded_summary":      "Excluded: %d inbox, %d sources, %d archive",
		"cli.export.excluded_filter":       "Excluded: %d documents not among the named slugs",
		"cli.export.excluded_outside":      "Excluded: %d documents outside the stage directories",
		"cli.export.excluded_unparsed":     "Excluded: %d documents whose frontmatter could not be parsed (check with engram lint)",
		"cli.export.sensitivity_on":        "Sensitivity: excluded %d private-local-only and restricted documents. No flag reverses this",
		"cli.export.sensitivity_off":       "Sensitivity: this wiki has the sensitivity attribute off, so there is nothing to filter",
		"cli.export.anonymized_count":      "Anonymized: replaced %d occurrences",
		"cli.export.unused_rules_warning":  "Warning: %d replacement rules never matched. Check the dictionary for typos",
		"cli.export.anonymized_none":       "Anonymized: no replacement file given, exporting text as-is",
		"cli.export.dangling_links":        "%d wikilinks pointing outside the bundle (%d documents). Bodies were not modified",
	})
}
