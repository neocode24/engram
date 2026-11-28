// Package pin은 위키 문서에서 나온 할 일 원장(`pin/`)을 읽고 쓰는 계층이다.
//
// 원장의 파일 이름과 JSON 모양은 upstream 의 pin 과 같다. 그래야 같은 요청
// 묶음을 upstream 파이썬과 이 패키지에 돌려 원장 파일을 바이트 단위로
// 견줄 수 있다(harness/pin). upstream 과 일부러 달리한 점은 ADR 0106 과
// docs/spec-map.md 의 Pin 절에 있다.
//
// 원장을 쓰는 곳은 이 패키지의 Apply 하나다. CLI, MCP, serve 화면이 같은
// 문을 지난다. 요청 하나를 적용한 직후 원장 전체를 검사하고, 어긋나면 그
// 요청만 되돌린다. 어긋난 원장은 저장되지 않는다.
package pin

import "regexp"

// 상태와 대기 사유 같은 값의 목록이다. 이름과 순서는 upstream 의 pinlib 과
// 같다. 하네스가 이 표면을 upstream 과 견준다.
var (
	Statuses     = []string{"todo", "doing", "waiting", "hold", "done"}
	WaitReasons  = []string{"prior", "reply", "decision"}
	RelationKind = []string{"before", "related", "duplicate"}
	// Sources는 항목의 출처 값 전부다. engram 이 쓰는 것은 wiki 와 command 뿐이고
	// 나머지 둘은 upstream 과 상태의 이름표를 같게 두려고 적어 둔다.
	Sources = []string{"wiki", "reminder", "command", "hermes"}
	// Stages는 근거의 단계다. 경로의 첫 폴더와 같아야 한다.
	Stages        = []string{"inbox", "sources", "context", "exports", "archive"}
	RecordKinds   = []string{"wiki", "pin", "reminder"}
	Bases         = []string{"stated", "inferred"}
	ProposalKinds = []string{"new_item", "status", "relation", "parent", "work", "delete"}
	// Vias는 요청 경로다. web 은 화면, command 는 `engram pin apply`, extract 는
	// 문서를 넣는 세 명령의 선언이다.
	Vias = []string{"web", "command", "extract"}
)

var (
	statusLabels = map[string]string{"todo": "할 일", "doing": "진행 중", "waiting": "대기", "hold": "보류", "done": "완료"}
	waitLabels   = map[string]string{"prior": "선행", "reply": "회신", "decision": "결정"}
	sourceLabels = map[string]string{"wiki": "위키", "reminder": "미리알림", "command": "직접 입력", "hermes": "Hermes"}
	recordLabels = map[string]string{"wiki": "위키", "pin": "Pin", "reminder": "미리알림"}
	signalLabels = map[string]string{"risk": "위험", "caution": "주의", "normal": "정상", "none": "없음"} // none: 열린 일이 없다
)

const (
	// Unsorted는 프로젝트가 정해지지 않은 항목의 work 값이다.
	Unsorted = "unsorted"
	// NotePathPrefix는 사람이 남긴 한 줄 메모가 언급된 곳에 들어갈 때의 경로 접두사다.
	NotePathPrefix = "note:"
	// ReminderPathPrefix는 미리알림 경로의 접두사다. 이 패키지는 미리알림을
	// 다루지 않으므로 문서 근거로 쓸 수 없다.
	ReminderPathPrefix = "reminder:"
	noteMax            = 500
	maxTitle           = 200
	maxWorkDesc        = 200
	maxWorkName        = 60
	markSeenMax        = 50
	relatedLimit       = 3
	defaultDueRiskDays = 3
	// defaultLaterDays는 config.signal.later_days 가 없을 때 예정이 되는 기한 전 날수다.
	defaultLaterDays = 14
	// StateSchema는 상태 JSON 의 schema 값이다. 형태가 바뀌어도 올리지 않는 것이
	// upstream 의 관례라, 하네스는 이 값이 아니라 형태 자체를 견준다.
	StateSchema     = 1
	workRecentLimit = 8
	doneDailyDays   = 14
	// DefaultPageURL은 상태의 page_url 과 항목 url 의 바탕이다. serve 화면 경로다.
	DefaultPageURL = "/pin"
)

// 항목의 칸 목록이다. upstream pinlib 과 같다.
var (
	itemKeys = []string{
		"id", "rev", "source", "title", "work", "owner", "status", "wait", "since", "due", "parent",
		"reminder", "evidence", "mentions", "context", "last_record", "locked", "created", "closed_at",
	}
	optionalItemKeys = []string{"rev_by", "changes", "origin", "guess", "unseen", "start", "freed_by"}
	changeKeys       = []string{"date", "field", "from", "to", "note", "via"}
	changeFields     = []string{"status", "due", "owner", "work", "title", "start"}
	evidenceKeys     = []string{"path", "section", "date", "stage", "quote"}
	mentionKeys      = []string{"date", "path", "quote"}
	contextKeys      = []string{"why", "now", "next", "written_at", "from_commit"}
	lockable         = []string{"owner", "work", "due", "title", "context", "context.why", "context.now", "context.next"}
	requestKeys      = []string{"id", "at", "via", "op", "item", "base_rev", "args", "received_at", "client"}
	emptyDirs        = []string{"items", "archive", "intake", "log", "prompts"}
)

var (
	humanOps = []string{
		"add", "set_status", "touch", "set_owner", "set_work", "set_due", "set_start", "set_parent", "set_title", "accept", "reject",
		"add_relation", "remove_relation", "confirm_relation", "delete", "cancel_delete", "set_work_order", "mark_seen",
	}
	// opsByVia는 요청 경로마다 허용된 연산이다. upstream 의 세 경로(web, command,
	// extract)를 글자 그대로 옮겼다. engram 에서 쓸 일이 없는 연산도 표에 두고
	// 변화 없음으로 답한다. 표를 줄이면 하네스가 견줄 때마다 예외를 적어야 한다.
	opsByVia = map[string]map[string]bool{
		"web":     setOf(humanOps...),
		"command": setOf(append(append([]string{}, humanOps...), "set_work_desc")...),
		"extract": setOf("add", "update_ai", "set_owner", "set_work", "set_due", "add_relation", "propose", "move_path"),
	}
	itemRequiredOps = setOf("set_status", "touch", "set_owner", "set_work", "set_due", "set_start", "set_parent", "update_ai", "set_title", "cancel_delete")
	baseRevOps      = setOf("set_status", "touch", "set_owner", "set_work", "set_due", "set_start", "set_parent", "set_title", "delete")
)

var (
	itemIDRe     = regexp.MustCompile(`^P\d{4}$`)
	proposalIDRe = regexp.MustCompile(`^Q\d{4}$`)
	clientRe     = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	dateRe       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// OpsByVia는 요청 경로마다 허용된 연산 이름을 정렬해 돌려준다. 하네스가 upstream
// 의 표와 견주는 데 쓴다.
func OpsByVia() map[string][]string {
	out := map[string][]string{}
	for via, ops := range opsByVia {
		out[via] = sortedKeys(ops)
	}
	return out
}

// Constants는 upstream pinlib 의 상수와 견줄 값을 낸다. 하네스가 쓴다.
func Constants() map[string]any {
	return map[string]any{
		"STATUSES":      toAnyList(Statuses),
		"WAIT_REASONS":  toAnyList(WaitReasons),
		"SOURCES":       toAnyList(Sources),
		"STAGES":        toAnyList(Stages),
		"BASES":         toAnyList(Bases),
		"ITEM_KEYS":     toAnyList(itemKeys),
		"EVIDENCE_KEYS": toAnyList(evidenceKeys),
	}
}
