#!/usr/bin/env python3
"""ad-hoc mermaid 렌더 검증. 정식 테스트가 아니다.
문서 안의 모든 ```mermaid 블록을 mermaid-cli로 렌더해 실패 블록을 보고한다.
사용: python3 scripts/check-mermaid.py docs/architecture.md [...]"""
import re, os, sys, subprocess, tempfile

files = sys.argv[1:] or ["README.md", "docs/architecture.md"]
env = dict(os.environ, npm_config_cache="/tmp/npmcache")
fail, total = [], 0

with tempfile.TemporaryDirectory() as td:
    for path in files:
        blocks = re.findall(r"```mermaid\n(.*?)```", open(path, encoding="utf-8").read(), re.S)
        for i, src in enumerate(blocks):
            total += 1
            mmd = os.path.join(td, f"b{i}.mmd")
            open(mmd, "w").write(src)
            cmd = ["npx", "-y", "@mermaid-js/mermaid-cli", "-i", mmd,
                   "-o", os.path.join(td, f"b{i}.svg")]
            # CI 러너처럼 샌드박스를 쓸 수 없거나 크롬 경로가 다른 환경에서는
            # PUPPETEER_CONFIG 로 설정 파일을 넘긴다. 없으면 기본 동작이다
            pcfg = os.environ.get("PUPPETEER_CONFIG")
            if pcfg:
                cmd += ["-p", pcfg]
            r = subprocess.run(cmd, capture_output=True, text=True, env=env)
            head = src.strip().split("\n")[0][:40]
            if r.returncode != 0 or not os.path.exists(os.path.join(td, f"b{i}.svg")):
                fail.append(f"{path} 블록{i+1} ({head}): {r.stderr.strip()[-200:]}")
            else:
                print(f"  ok  {path} 블록{i+1}  {head}")

print(f"\nmermaid 블록 {total}건, 실패 {len(fail)}건")
if fail:
    print("\n".join("  FAIL " + x for x in fail)); sys.exit(1)
print("전부 렌더 성공 (ad-hoc 검증)")
