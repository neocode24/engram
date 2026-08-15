#!/usr/bin/env python3
"""공개 경계 검사 (ADR 0024).

저장소 본문과 커밋 이력에서 금지 패턴을 찾는다. 이 저장소는 공개를 전제하므로
조직 고유 명칭이 섞이면 안 된다.

패턴 목록 자체가 식별자이므로 저장소에 두지 않는다. `private/boundary-patterns.txt`에
한 줄에 하나씩 정규표현식을 적는다. `#`으로 시작하는 줄과 빈 줄은 무시한다.
목록 파일이 없으면 검사를 건너뛴다. CI에는 목록이 없으므로 항상 건너뛴다.
CI 로그는 저장소와 함께 공개되므로 거기에 패턴이 찍히면 가드가 유출 경로가 된다.

사용법:
    python3 scripts/check-boundary.py            워킹트리만 검사
    python3 scripts/check-boundary.py --history  커밋 이력까지 검사 (느리다)
"""

import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PATTERNS = os.path.join(ROOT, "private", "boundary-patterns.txt")
SKIP_DIRS = {".git", "private", "node_modules"}
SKIP_FILES = {"check-boundary.py"}


def load_patterns():
    if not os.path.exists(PATTERNS):
        return None
    out = []
    with open(PATTERNS, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                out.append(re.compile(line, re.IGNORECASE))
    return out


def scan_worktree(pats):
    hits = []
    for dirpath, dirnames, filenames in os.walk(ROOT):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            if name in SKIP_FILES or name.endswith((".png", ".jpg", ".pdf", ".sum", ".exe")):
                continue
            path = os.path.join(dirpath, name)
            try:
                text = open(path, encoding="utf-8").read()
            except (UnicodeDecodeError, OSError):
                continue
            for i, line in enumerate(text.splitlines(), 1):
                for p in pats:
                    if p.search(line):
                        rel = os.path.relpath(path, ROOT)
                        # 적중한 내용을 출력하지 않는다. 로그가 유출 경로가 된다
                        hits.append(f"{rel}:{i} 금지 패턴 적중")
                        break
    return hits


def scan_history(pats):
    try:
        blob = subprocess.run(
            ["git", "-C", ROOT, "log", "--all", "-p", "--format=%H%n%s%n%b"],
            capture_output=True, text=True, timeout=300,
        ).stdout
    except (subprocess.SubprocessError, OSError) as e:
        return [f"이력 검사 실패: {e}"]
    hits = []
    commit = "?"
    for line in blob.splitlines():
        if re.fullmatch(r"[0-9a-f]{40}", line):
            commit = line[:12]
            continue
        for p in pats:
            if p.search(line):
                hits.append(f"이력 {commit} 금지 패턴 적중")
                break
    return sorted(set(hits))


def main():
    pats = load_patterns()
    if pats is None:
        print("패턴 목록이 없어 건너뛴다 (private/boundary-patterns.txt)")
        return 0
    if not pats:
        print("패턴 목록이 비어 있다")
        return 0

    hits = scan_worktree(pats)
    if "--history" in sys.argv:
        hits += scan_history(pats)

    if hits:
        print(f"공개 경계 위반 {len(hits)}건")
        for h in hits:
            print(f"  {h}")
        print("적중 내용은 출력하지 않는다. 해당 위치를 직접 열어 확인한다")
        return 1
    print(f"패턴 {len(pats)}건 검사, 위반 없음")
    return 0


if __name__ == "__main__":
    sys.exit(main())
