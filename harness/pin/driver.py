#!/usr/bin/env python3
"""harness/pin 이 upstream 의 pin 코드를 같은 입력에 돌리는 드라이버.

harness 시험 도구이며 engram 바이너리는 이것을 부르지 않는다. upstream 의 apply.run 을 직접
불러 미리알림 DB 와 동기화 상태를 건드리지 않는다(apply.main 은 reminders_sync.peek_seen 을 읽는다).

사용:
  driver.py apply   <upstream scripts/pin> <위키> <요청 파일> <시각 ISO> <상태 출력 파일>
  driver.py consts  <upstream scripts/pin>
  driver.py surface <upstream scripts/pin> <견본 디렉터리 harness/pin/fixtures>

surface 는 scripts/upstream-sync.py 가 부른다. 견본 위키에 요청 묶음을 순서대로 적용하고 상수, 게이트 결과,
상태의 형태와 값을 지문으로 떠서 JSON 한 줄로 낸다. upstream 의 STATE_SCHEMA 와 CHANGELOG 의 impact 는 읽지 않는다(ADR 0094).
"""
import hashlib
import json
import shutil
import sys
import tempfile
from datetime import datetime
from pathlib import Path

CONST_NAMES = ("STATUSES", "WAIT_REASONS", "SOURCES", "STAGES", "BASES", "ITEM_KEYS", "EVIDENCE_KEYS")


def shape(o, p=""):
    """상태의 키 경로 집합. 값은 보지 않고 값의 형(type)만 본다."""
    if isinstance(o, dict):
        out = set()
        for k, v in o.items():
            out |= shape(v, p + "." + k)
        return out or {p + ":{}"}
    if isinstance(o, list):
        out = set()
        for v in o:
            out |= shape(v, p + "[]")
        return out or {p + ":[]"}
    return {p + ":" + type(o).__name__}


def digest(o):
    return hashlib.sha256(json.dumps(o, sort_keys=True, ensure_ascii=False).encode()).hexdigest()[:16]


def tree_digest(root):
    h = hashlib.sha256()
    for p in sorted(root.rglob("*")):
        if p.is_file() and p.name != ".lock" and not p.name.endswith(".tmp"):
            h.update(p.relative_to(root).as_posix().encode() + b"\0" + p.read_bytes() + b"\0")
    return h.hexdigest()[:16]


def surface(apply, pinlib, fixtures):
    consts = {"ops_by_via": {v: sorted(pinlib.OPS_BY_VIA[v]) for v in ("web", "command", "extract")}}
    for name in CONST_NAMES:
        consts[name] = list(getattr(pinlib, name))
    tmp = Path(tempfile.mkdtemp(prefix="pin-surface-"))
    try:
        wiki = tmp / "wiki"
        shutil.copytree(fixtures / "wiki", wiki)
        (wiki / "pin").mkdir()
        shutil.copy(fixtures / "config.json", wiki / "pin" / "config.json")
        pinlib.init_pin(wiki / "pin")
        gate, values, paths = {}, {}, set()
        for batch in sorted((fixtures / "batches").glob("*.json")):
            data = json.loads(batch.read_text(encoding="utf-8"))
            reqs = tmp / (batch.stem + "-req.json")
            reqs.write_text(json.dumps({"requests": data["requests"]}), encoding="utf-8")
            state = tmp / (batch.stem + "-state.json")
            summary = apply.run(wiki, apply.load_request_files([reqs]), datetime.fromisoformat(data["now"]), state, use_intake=False)
            gate[batch.stem] = {
                "results": [[r["id"], r["result"], r.get("code"), r.get("message")] for r in summary["results"]],
                "archived": list(summary["archived"]),
                "files": tree_digest(wiki / "pin"),
            }
            st = json.loads(state.read_text(encoding="utf-8"))
            values[batch.stem] = digest(st)
            paths |= shape(st)
        return {"consts": consts, "gate": gate, "state_value": values, "state_shape": sorted(paths)}
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


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
    if mode == "surface":
        print(json.dumps(surface(apply, pinlib, Path(argv[3])), ensure_ascii=False, sort_keys=True))
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
