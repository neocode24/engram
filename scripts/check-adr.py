#!/usr/bin/env python3
"""ad-hoc ADR 검증. 정식 테스트 스위트가 아니다.
1) frontmatter 4키 + 상태 어휘 준수  2) 파일번호와 number 일치
3) README 색인의 상태표와 실제 status 일치  4) 상대링크 무결성"""
import re, os, glob, sys

D = os.path.join(os.path.dirname(__file__), "..", "docs", "decisions")
D = os.path.normpath(D)
ROOT = os.path.normpath(os.path.join(D, "..", ".."))
VOCAB = {"accepted", "amended", "superseded", "proposed"}
KEYS = {"number", "title", "date", "status"}
fail = []

actual = {}
for f in sorted(glob.glob(os.path.join(D, "0*.md"))):
    b = os.path.basename(f)
    m = re.match(r"---\n(.*?)\n---\n", open(f).read(), re.S)
    if not m:
        fail.append(f"{b}: frontmatter 없음"); continue
    fm = dict(re.findall(r"^(\w+):\s*(.+)$", m.group(1), re.M))
    if set(fm) != KEYS:
        fail.append(f"{b}: 키 불일치 {sorted(set(fm) ^ KEYS)}")
    if fm.get("status") not in VOCAB:
        fail.append(f"{b}: 상태 어휘 위반 {fm.get('status')}")
    if fm.get("number", "").zfill(4) != b[:4]:
        fail.append(f"{b}: number 불일치 {fm.get('number')}")
    actual[b[:4]] = fm.get("status")

idx = open(os.path.join(D, "README.md")).read()
listed = dict(re.findall(r"\|\s*\[(\d{4})\]\([^)]+\)\s*\|[^|]+\|[^|]+\|\s*(\w+)\s*\|", idx))
for n, st in actual.items():
    if n not in listed:
        fail.append(f"색인 누락: {n}")
    elif listed[n] != st:
        fail.append(f"색인 불일치 {n}: 색인={listed[n]} 실제={st}")
for n in listed:
    if n not in actual:
        fail.append(f"색인에만 있음: {n}")

broken = 0
for f in glob.glob(os.path.join(ROOT, "**", "*.md"), recursive=True):
    if "/.git/" in f:
        continue
    d = os.path.dirname(f)
    for m in re.finditer(r"\]\((?!https?:|mailto:|#)([^)#]+)", open(f).read()):
        if not os.path.exists(os.path.normpath(os.path.join(d, m.group(1).strip()))):
            fail.append(f"깨진 링크 {os.path.relpath(f, ROOT)} -> {m.group(1)}")
            broken += 1

print(f"ADR {len(actual)}건, 색인 {len(listed)}건, 링크 깨짐 {broken}건")
if fail:
    print("\n".join("  FAIL " + x for x in fail)); sys.exit(1)
print("전부 통과 (ad-hoc 검증)")
