#!/usr/bin/env python3
"""upstream 계약 vendoring 과 delta 감지 (ADR 0029).

upstream llm-wiki 의 계약 파일을 익명화해 `harness/upstream/` 에 옮기고,
sync 상태를 `harness/upstream.lock` 에, 계약 변화를 `private/deltas/` 에 남긴다.

계약 파일 목록은 이 스크립트에 두지 않는다. upstream `AGENTS.md` 의
"meta 계약 변경 로그" 절이 선언하는 목록을 매 실행마다 다시 읽는다.
upstream 이 목록을 바꿔도 이 쪽은 고칠 것이 없다는 것이 ADR 0029 의 결정이다.

읽는 것은 upstream 의 커밋 객체다. 작업 폴더가 더러워도 sync 결과가 흔들리지
않도록 하기 위해서다. lock 의 해시와 vendored 내용이 늘 한 커밋을 가리킨다.

사용법:
    python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki
    python3 scripts/upstream-sync.py --upstream ~/Git/llm-wiki --check

--check 는 아무것도 쓰지 않고 무엇이 바뀔지만 낸다.
"""

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
UPSTREAM_DIR = os.path.join(ROOT, "harness", "upstream")
# delta 는 upstream CHANGELOG 원문을 인용하므로 공개하지 않는다. ADR 0030.
DELTAS_DIR = os.path.join(ROOT, "private", "deltas")
LOCK_PATH = os.path.join(ROOT, "harness", "upstream.lock")
REPLACEMENTS = os.path.join(ROOT, "private", "vendor-replacements.txt")
CHECK_BOUNDARY = os.path.join(ROOT, "scripts", "check-boundary.py")

# 사전 전체가 조직 어휘 목록이라 vendoring 하지 않는다 (ADR 0029).
EXCLUDED = "terminology-normalization.md"


def die(msg):
    print(f"실패: {msg}")
    sys.exit(1)


def git(upstream, *args, check=True):
    """upstream 저장소에서 git 명령을 돌리고 stdout 을 돌려준다."""
    r = subprocess.run(
        ["git", "-C", upstream, *args],
        capture_output=True, text=True, check=False,
    )
    if check and r.returncode != 0:
        die(f"git {' '.join(args)} 실패: {r.stderr.strip()}")
    return r


def read_contract_list(upstream, head):
    """upstream AGENTS.md 의 계약 파일 선언에서 파일명을 뽑는다.

    "계약 파일(`a.md`, `b.md`, ...)" 괄호 안의 백틱 파일명이 선언이다.
    절을 못 찾거나 선언이 비면 에러로 끝낸다. 조용히 넘어가면 upstream 이
    계약을 바꿨을 때 알아챌 수 없다.
    """
    agents = git(upstream, "show", f"{head}:AGENTS.md").stdout
    section = []
    in_section = False
    for line in agents.splitlines():
        if line.strip() == "## meta 계약 변경 로그":
            in_section = True
            continue
        if in_section and line.startswith("## "):
            break
        if in_section:
            section.append(line)
    if not in_section:
        die('upstream AGENTS.md 에 "## meta 계약 변경 로그" 절이 없습니다')
    m = re.search(r"계약 파일\(([^)]*)\)", "\n".join(section))
    if not m:
        die("upstream AGENTS.md 에서 계약 파일 선언 괄호를 찾지 못했습니다")
    names = re.findall(r"`([^`]+)`", m.group(1))
    names = [n for n in names if n.endswith(".md") and "/" not in n]
    if not names:
        die("계약 파일 선언에서 파일명을 뽑지 못했습니다")
    missing = [n for n in names
               if git(upstream, "cat-file", "-e", f"{head}:meta/{n}").returncode != 0]
    if missing:
        die(f"선언된 계약 파일이 upstream meta/ 에 없습니다: {', '.join(missing)}")
    return names


def load_replacements():
    """치환 사전을 읽는다. 파일이 없으면 치환 없이 간다.

    형식은 private/history-replacements.txt 와 같다. 한 줄에
    `원문==>대체어` 하나이며 `regex:` 접두사를 붙이면 정규표현식으로
    치환한다. `#` 줄과 빈 줄은 무시한다.
    """
    if not os.path.exists(REPLACEMENTS):
        print("치환 사전이 없어 익명화 없이 진행합니다 (private/vendor-replacements.txt)")
        return [], 0
    entries = []
    with open(REPLACEMENTS, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "==>" not in line:
                die(f"치환 사전의 줄을 해석하지 못했습니다: {line.split('==>')[0][:20]}...")
            if line.startswith("regex:"):
                body = line[len("regex:"):]
                src, dst = body.split("==>", 1)
                entries.append((re.compile(src), dst, src, True))
            else:
                src, dst = line.split("==>", 1)
                if not src:
                    die("치환 사전에 빈 원문이 있습니다")
                entries.append((None, dst, src, False))
    # 긴 원문을 먼저 치환한다. 짧은 원문이 긴 원문의 일부이면
    # 순서가 틀릴 때 긴 쪽이 깨진다.
    entries.sort(key=lambda e: len(e[2]), reverse=True)
    return entries, len(entries)


def apply_replacements(text, entries):
    """치환을 적용하고 적용 건수를 센다."""
    count = 0
    for pattern, dst, src, is_regex in entries:
        if is_regex:
            text, n = pattern.subn(dst, text)
        else:
            n = text.count(src)
            if n:
                text = text.replace(src, dst)
        count += n
    return text, count


def vendor_header(short, count):
    """치환 사실을 독자에게 알리는 머리말. 원문과 다르다는 것을 알아야 한다."""
    line2 = f"익명화 치환 {count}건을 적용했다" if count else "치환 없음"
    return (
        f"<!-- upstream llm-wiki meta/NAME 에서 가져왔다.\n"
        f"     원본 커밋 {short}. {line2}.\n"
        f"     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->\n"
    )


def parse_changelog(text):
    """CHANGELOG 를 항목 단위로 나눈다.

    항목은 (날짜, 파일, 본문 줄) 하나다. 날짜는 `## `, 파일은 `### `,
    본문은 `- ` 로 시작하는 줄에서 나온다. 날짜 절이 아니면 서두이므로
    항목으로 세지 않는다.
    """
    date = file = None
    out = []
    for line in text.splitlines():
        s = line.strip()
        if re.match(r"## \d{4}-\d{2}-\d{2}$", s):
            date, file = s[3:].strip(), None
        elif date is not None and s.startswith("### "):
            file = s[4:].strip()
        elif date is not None and s.startswith("- "):
            out.append((date, file, s))
    return out


def build_delta(upstream, old_head, head, entries):
    """lock 커밋과 HEAD 사이의 CHANGELOG 변화를 delta 문서로 만든다."""
    old = git(upstream, "show", f"{old_head}:meta/CHANGELOG.md", check=False)
    old_text = old.stdout if old.returncode == 0 else ""
    old_lines = {s for _, _, s in parse_changelog(old_text)}
    fresh = [e for e in parse_changelog(
        git(upstream, "show", f"{head}:meta/CHANGELOG.md").stdout)
        if e[2] not in old_lines]

    binary = [e for e in fresh if "binary-affecting" in e[2]]
    wiki = [e for e in fresh if "binary-affecting" not in e[2]]
    parts = [
        "<!-- upstream llm-wiki meta/CHANGELOG.md 의 변화. scripts/upstream-sync.py 가 만들었다.",
        "     손으로 고치지 않는다. 자동 반영하지 않는다. 사람이 읽고 판단한다. -->",
        "",
        f"# upstream delta {old_head[:7]} 에서 {head[:7]} 까지",
        "",
        f"- 이전 sync 커밋: {old_head}",
        f"- 이번 HEAD 커밋: {head}",
        f"- binary-affecting {len(binary)}건, wiki-only {len(wiki)}건",
        "",
        "## binary-affecting, 구현이 따라가야 할 항목",
        "",
    ]

    def add_entries(items):
        if not items:
            parts.append("(해당 없음)")
            parts.append("")
            return
        last_key = None
        for date, file, body in items:
            if (date, file) != last_key:
                if last_key is not None:
                    parts.append("")
                parts.append(f"### {date} {file}")
                last_key = (date, file)
            parts.append(body)
        parts.append("")

    add_entries(binary)
    parts.append("## wiki-only, 참고")
    parts.append("")
    add_entries(wiki)
    text, count = apply_replacements("\n".join(parts) + "\n", entries)
    return text, len(binary), len(wiki), count


def run_boundary():
    """경계 검사를 그대로 부른다. 출력은 경로와 줄번호만 있으므로 그대로 넘긴다."""
    r = subprocess.run(
        [sys.executable, CHECK_BOUNDARY], capture_output=True, text=True, check=False,
    )
    return r.returncode, r.stdout.strip()


def write_if_changed(path, text):
    """내용이 다를 때만 쓴다. 같은 상태를 두 번 써도 바이트가 흔들리지 않는다."""
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            if f.read() == text:
                return False
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(text)
    return True


def main():
    ap = argparse.ArgumentParser(description="upstream 계약 vendoring 과 delta 감지")
    ap.add_argument("--upstream", required=True, help="upstream llm-wiki 경로")
    ap.add_argument("--check", action="store_true", help="쓰지 않고 변화만 낸다")
    args = ap.parse_args()

    upstream = os.path.abspath(os.path.expanduser(args.upstream))
    r = git(upstream, "rev-parse", "--verify", "HEAD")
    head = r.stdout.strip()
    short = head[:7]

    names = read_contract_list(upstream, head)
    if EXCLUDED in names:
        names.remove(EXCLUDED)
        print(f"{EXCLUDED} 는 사전 전체가 조직 어휘 목록이라 제외합니다 (ADR 0029)")
    entries, dict_size = load_replacements()

    old_lock = None
    if os.path.exists(LOCK_PATH):
        with open(LOCK_PATH, encoding="utf-8") as f:
            old_lock = json.load(f)
        if git(upstream, "cat-file", "-e",
               f"{old_lock.get('commit', '')}^{{commit}}").returncode != 0:
            die(f"lock 의 커밋 {old_lock.get('commit', '')[:7]} 을 upstream 에서 찾을 수 없습니다")

    changed_files = []
    total_subs = 0
    for name in names:
        src = git(upstream, "show", f"{head}:meta/{name}").stdout
        body, n = apply_replacements(src, entries)
        total_subs += n
        text = vendor_header(short, n).replace("NAME", name) + body
        target = os.path.join(UPSTREAM_DIR, name)
        if args.check:
            exists = os.path.exists(target)
            same = exists and open(target, encoding="utf-8").read() == text
            state = "유지" if same else ("갱신" if exists else "신규")
            print(f"  {state} {name} (치환 {n}건)")
        else:
            if write_if_changed(target, text):
                changed_files.append(name)
    if args.check:
        print(f"총 치환 {total_subs}건 (사전 {dict_size}항목)")
        if old_lock is None:
            print(f"lock: 신규 (커밋 {short})")
        elif old_lock.get("commit") == head:
            print(f"lock: 유지 (커밋 {short})")
        else:
            print(f"lock: 갱신 ({old_lock['commit'][:7]} 에서 {short})")
            _, b, w, _ = build_delta(upstream, old_lock["commit"], head, entries)
            print(f"delta: private/deltas/{short}.md 생성 (binary-affecting {b}건, wiki-only {w}건)")
        return 0

    written = [n for n in names]
    print(f"harness/upstream/ 에 {len(written)}개를 맞췄습니다 (치환 {total_subs}건)")
    if changed_files:
        print("  갱신: " + ", ".join(changed_files))

    # 목록에서 빠진 파일은 계약이 아니게 된 것이므로 남기지 않는다.
    stale = []
    if os.path.isdir(UPSTREAM_DIR):
        for name in os.listdir(UPSTREAM_DIR):
            if name not in names:
                os.remove(os.path.join(UPSTREAM_DIR, name))
                stale.append(name)
    if stale:
        print("  제거(계약 목록에서 빠짐): " + ", ".join(stale))

    delta_path = None
    if old_lock is None:
        print("lock 이 없어 delta 를 만들지 않습니다 (첫 sync)")
    elif old_lock.get("commit") == head:
        print("upstream HEAD 가 lock 과 같아 delta 를 만들지 않습니다")
    else:
        text, b, w, _ = build_delta(upstream, old_lock["commit"], head, entries)
        delta_path = os.path.join(DELTAS_DIR, f"{short}.md")
        if write_if_changed(delta_path, text):
            print(f"delta 를 남겼습니다: private/deltas/{short}.md "
                  f"(binary-affecting {b}건, wiki-only {w}건)")
        else:
            print(f"delta 는 이미 같은 내용입니다: private/deltas/{short}.md")

    code, out = run_boundary()
    if code != 0:
        # 걸린 문자열을 화면에 찍지 않는다. check-boundary 출력은 경로와
        # 줄번호만 있으므로 그것만 넘긴다.
        print(out)
        mine = any(l.lstrip().startswith(("harness/upstream/", "private/deltas/"))
                   for l in out.splitlines())
        if mine:
            for name in names:
                p = os.path.join(UPSTREAM_DIR, name)
                if os.path.exists(p):
                    os.remove(p)
            if delta_path and os.path.exists(delta_path):
                os.remove(delta_path)
            print("익명화되지 않은 문자열이 남아 sync 를 실패시킵니다. "
                  "private/vendor-replacements.txt 에 항목을 추가하고 다시 실행하십시오")
        else:
            print("위반은 vendoring 결과 밖의 파일에서 났습니다. "
                  "해당 파일을 정리한 뒤 다시 실행하십시오")
        return 1

    same_state = (old_lock is not None
                  and old_lock.get("commit") == head
                  and sorted(old_lock.get("files", [])) == sorted(names))
    if same_state:
        print(f"lock 은 같은 상태라 그대로 둡니다 (커밋 {short})")
    else:
        lock = {
            "commit": head,
            "files": sorted(names),
            "synced_at": datetime.now().isoformat(timespec="seconds"),
        }
        with open(LOCK_PATH, "w", encoding="utf-8") as f:
            json.dump(lock, f, ensure_ascii=False, indent=2, sort_keys=True)
            f.write("\n")
        print(f"lock 을 썼습니다 (커밋 {short})")
    print("경계 검사 통과")
    return 0


if __name__ == "__main__":
    sys.exit(main())
