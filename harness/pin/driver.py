#!/usr/bin/env python3
"""harness/pin 이 upstream 의 pin 코드를 같은 입력에 돌리는 드라이버.

harness 시험 도구이며 engram 바이너리는 이것을 부르지 않는다. upstream 의 apply.run 을 직접
불러 미리알림 DB 와 동기화 상태를 건드리지 않는다(apply.main 은 reminders_sync.peek_seen 을 읽는다).

사용:
  driver.py apply  <upstream scripts/pin> <위키> <요청 파일> <시각 ISO> <상태 출력 파일>
  driver.py consts <upstream scripts/pin>
"""
import json
import sys
from datetime import datetime
from pathlib import Path


def main(argv):
    mode, up = argv[1], Path(argv[2])
    sys.path.insert(0, str(up))
    sys.dont_write_bytecode = True
    import apply  # noqa: E402
    import pinlib  # noqa: E402

    if mode == "consts":
        print(json.dumps({
            "ops_by_via": {v: sorted(pinlib.OPS_BY_VIA[v]) for v in ("web", "command", "extract")},
            "STATUSES": list(pinlib.STATUSES), "WAIT_REASONS": list(pinlib.WAIT_REASONS),
            "SOURCES": list(pinlib.SOURCES), "STAGES": list(pinlib.STAGES), "BASES": list(pinlib.BASES),
            "ITEM_KEYS": list(pinlib.ITEM_KEYS), "EVIDENCE_KEYS": list(pinlib.EVIDENCE_KEYS),
        }, ensure_ascii=False))
        return 0
    if mode == "init":
        created = pinlib.init_pin(Path(argv[3]) / "pin")
        print(json.dumps(sorted(created), ensure_ascii=False))
        return 0
    wiki, reqs, now, state = Path(argv[3]), Path(argv[4]), datetime.fromisoformat(argv[5]), Path(argv[6])
    summary = apply.run(wiki, apply.load_request_files([reqs]), now, state, use_intake=False)
    out = [{"id": r["id"], "result": r["result"], "code": r.get("code"), "message": r.get("message")} for r in summary["results"]]
    print(json.dumps({"results": out, "archived": summary["archived"], "written": summary["written"]}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
