// Package i18n의 cli_pin 카탈로그. pin 커맨드, --pin 선언, serve 의 Pin 화면 안내. ADR 0049, 0106.
package i18n

func init() {
	Register(LangKO, map[string]string{
		"cli.pin.short": "위키 문서에서 나온 할 일 원장을 다룹니다",
		"cli.pin.long": `위키 문서에서 나온 할 일 원장(pin/)을 다룹니다.

원장은 engram init 이 위키 루트에 만듭니다. capture, source, promote --to sources
는 --pin 선언 없이는 아무것도 쓰지 않습니다. 할 일인지의 판단은 명령을 부른
에이전트가 하고, 확정은 사람이 serve 의 /pin 화면에서 합니다. pin/ 이 없는 위키는
아직 갱신하지 않은 위키이니 engram init 을 다시 실행하세요.`,
		"cli.pin.flag_pin": "할 일 선언(JSON 파일 경로, - 는 표준 입력) 또는 none. 필수입니다",
		"cli.pin.required": "%s 에는 --pin 이 필요합니다. 아무것도 쓰지 않았습니다.\n" +
			"engram pin list --json 으로 열린 일을 보고 선언을 만드세요. 할 일이 없으면 --pin none 입니다",
		"cli.pin.required_mcp": "pin 인자가 필요합니다. 아무것도 쓰지 않았습니다. 할 일이 없으면 none 을 주세요",
		"cli.pin.not_updated": "이 위키에는 할 일 원장(pin/)이 없습니다. 아직 갱신하지 않은 위키입니다. 아무것도 쓰지 않았습니다.\n" +
			"engram init <위키 경로> 를 다시 실행하면 pin/ 이 채워집니다. 있는 문서와 설정은 바뀌지 않습니다",
		"cli.pin.not_for_context": "context 문서는 할 일 판단 대상이 아닙니다. --pin 은 inbox 와 sources 에 문서를 넣을 때만 씁니다",
		"cli.pin.decl_invalid":    "선언이 올바르지 않습니다",
		"cli.pin.decl_read_fail":  "선언을 읽을 수 없음: %s",
		"cli.pin.rejected":        "선언이 원장 검사에서 거절되어 문서도 원장도 쓰지 않았습니다.%s",
		"cli.pin.follow_fail":     "문서는 옮겼으나 원장의 근거 경로를 따라 고치지 못했습니다",
		"cli.pin.done_none":       "Pin: 할 일이 없다고 판단했습니다. pin/log 에 남겼습니다",
		"cli.pin.done_items":      "Pin: 새 일 %s 이 제안 표시로 들어갔습니다(제안 %s). 확정은 serve 의 /pin 화면에서 사람이 합니다",
		"cli.pin.done_updated":    "Pin: 기존 일의 근거와 맥락을 갱신했습니다",
		"cli.pin.affected_head":   "Pin: %s 를 근거로 둔 열린 일 %d건의 경로를 따라 고쳤습니다",
		"cli.pin.affected_item":   "  %s %s (%s)",
		"cli.pin.affected_tail":   "  이 문서가 옮겨졌다고 일이 끝난 것은 아닙니다. 닫을지는 사람이 /pin 화면에서 정합니다",

		"cli.pin.list.short": "열린 일과 프로젝트와 사람이 버린 것을 냅니다",
		"cli.pin.list.long": `에이전트가 문서에서 할 일을 찾기 전에 읽는 색인입니다. 열린 일, 프로젝트,
사람이 버린 일을 냅니다. 인용문이 문서에서 사라진 근거의 수도 함께 냅니다.`,
		"cli.pin.list.items":         "열린 일 %d건",
		"cli.pin.list.item":          "  %s %s [%s, %s, %s]%s",
		"cli.pin.list.proposed":      " 제안 중",
		"cli.pin.list.works":         "프로젝트 %d개",
		"cli.pin.list.work":          "  %s %s",
		"cli.pin.list.rejected":      "사람이 버린 것 %d건(같은 일을 다시 내지 마세요)",
		"cli.pin.list.rejected_item": "  %s",
		"cli.pin.list.stale":         "인용문이 문서에서 사라진 근거 %d건",

		"cli.pin.apply.short": "사람이 시킨 변경을 원장에 넣습니다",
		"cli.pin.apply.long": `요청 파일의 변경을 화면과 같은 게이트로 원장에 넣습니다. 에이전트는 이것으로 사람이
시킨 상태 변경, 수락, 삭제를 합니다. 사람이 시키지 않은 수락과 변경과 삭제는 하지 마세요.

요청 파일은 {"requests":[{"op":"set_status","item":"P0001","args":{"status":"done"}}]} 모양입니다.
id 와 at 은 생략하면 채웁니다. 거절된 요청이 있으면 종료 코드 1 입니다.`,
		"cli.pin.apply.flag_requests": "변경 요청 JSON 파일(- 는 표준 입력)",
		"cli.pin.apply.no_requests":   "--requests 가 필요합니다",
		"cli.pin.apply.line":          "%-9s %s %s %s",
		"cli.pin.apply.summary":       "반영 %d, 변화 없음 %d, 거절 %d, 건너뜀 %d, 보관 %d",
		"cli.pin.apply.rejected":      "요청 %d건이 거절되었습니다",

		"cli.serve.notice_pin":         "Pin 화면: http://%s/pin (쓰는 곳은 pin/ 뿐입니다)",
		"cli.serve.notice_pin_missing": "Pin 화면을 열지 않았습니다: 이 위키에는 pin/ 이 없습니다. engram init 을 다시 실행하세요",
		"cli.serve.notice_pin_off":     "Pin 화면을 열지 않았습니다: %s 는 이 컴퓨터 밖에서 닿는 주소입니다. 인증 없는 쓰기를 열 수 없습니다. 127.0.0.1 에 묶으세요",

		"core.serve.pin_bad_host":   "Host 가 이 컴퓨터의 주소가 아닙니다: %s",
		"core.serve.pin_bad_origin": "다른 출처에서 온 요청입니다",
		"core.serve.pin_no_header":  "X-Pin: 1 헤더가 없습니다",
		"core.serve.pin_bad_type":   "Content-Type 이 application/json 이 아닙니다",
		"core.serve.pin_too_large":  "요청 본문이 %d바이트를 넘습니다",
	})
	Register(LangEN, map[string]string{
		"cli.pin.short": "Manage the to-do ledger derived from wiki documents",
		"cli.pin.long": `Manage the to-do ledger (pin/) derived from wiki documents.

The ledger is created by engram init at the wiki root. capture, source and
promote --to sources write nothing without a --pin declaration. The agent that
runs the command judges whether something is a to-do; a person confirms it on
the /pin page of serve. A wiki without pin/ has not been updated yet: run
engram init again.`,
		"cli.pin.flag_pin": "to-do declaration (JSON file path, - for stdin) or none. Required",
		"cli.pin.required": "%s needs --pin. Nothing was written.\n" +
			"Read the open items with engram pin list --json and build a declaration. Use --pin none when there is no to-do",
		"cli.pin.required_mcp": "The pin argument is required. Nothing was written. Pass none when there is no to-do",
		"cli.pin.not_updated": "This wiki has no to-do ledger (pin/). It has not been updated yet. Nothing was written.\n" +
			"Run engram init <wiki path> again and pin/ will be filled in. Existing documents and config are not changed",
		"cli.pin.not_for_context": "context documents are not judged for to-dos. --pin is only for putting documents into inbox and sources",
		"cli.pin.decl_invalid":    "The declaration is invalid",
		"cli.pin.decl_read_fail":  "cannot read the declaration: %s",
		"cli.pin.rejected":        "The declaration was rejected by the ledger check, so neither the document nor the ledger was written.%s",
		"cli.pin.follow_fail":     "The document was moved but the ledger's evidence paths could not be followed",
		"cli.pin.done_none":       "Pin: judged that there is no to-do. Recorded in pin/log",
		"cli.pin.done_items":      "Pin: new items %s entered as proposals (proposals %s). A person confirms them on the /pin page of serve",
		"cli.pin.done_updated":    "Pin: updated the evidence and context of existing items",
		"cli.pin.affected_head":   "Pin: followed the paths of %[2]d open items that cite %[1]s",
		"cli.pin.affected_item":   "  %s %s (%s)",
		"cli.pin.affected_tail":   "  Moving this document does not mean the work is done. A person decides on the /pin page",

		"cli.pin.list.short": "List open items, projects and what people discarded",
		"cli.pin.list.long": `The index an agent reads before looking for to-dos in a document. It lists open items,
projects and items people discarded, plus the number of evidence quotes that vanished from their documents.`,
		"cli.pin.list.items":         "%d open items",
		"cli.pin.list.item":          "  %s %s [%s, %s, %s]%s",
		"cli.pin.list.proposed":      " proposed",
		"cli.pin.list.works":         "%d projects",
		"cli.pin.list.work":          "  %s %s",
		"cli.pin.list.rejected":      "%d discarded by people (do not propose the same again)",
		"cli.pin.list.rejected_item": "  %s",
		"cli.pin.list.stale":         "%d evidence quotes vanished from their documents",

		"cli.pin.apply.short": "Apply changes a person asked for to the ledger",
		"cli.pin.apply.long": `Apply the changes in a request file to the ledger through the same gate as the page. Agents use this
for status changes, acceptances and deletions a person asked for. Do not accept, change or delete on your own.

A request file looks like {"requests":[{"op":"set_status","item":"P0001","args":{"status":"done"}}]}.
id and at are filled in when omitted. The exit code is 1 when any request is rejected.`,
		"cli.pin.apply.flag_requests": "change request JSON file (- for stdin)",
		"cli.pin.apply.no_requests":   "--requests is required",
		"cli.pin.apply.line":          "%-9s %s %s %s",
		"cli.pin.apply.summary":       "applied %d, no change %d, rejected %d, skipped %d, archived %d",
		"cli.pin.apply.rejected":      "%d requests were rejected",

		"cli.serve.notice_pin":         "Pin page: http://%s/pin (writes go to pin/ only)",
		"cli.serve.notice_pin_missing": "The Pin page is not open: this wiki has no pin/. Run engram init again",
		"cli.serve.notice_pin_off":     "The Pin page is not open: %s is reachable from outside this computer. Writes without authentication cannot be opened. Bind to 127.0.0.1",

		"core.serve.pin_bad_host":   "Host is not an address of this computer: %s",
		"core.serve.pin_bad_origin": "The request came from another origin",
		"core.serve.pin_no_header":  "The X-Pin: 1 header is missing",
		"core.serve.pin_bad_type":   "Content-Type is not application/json",
		"core.serve.pin_too_large":  "The request body exceeds %d bytes",
	})
}
