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

Pin 화면과 생성 코드(scripts/pin)는 계약 파일이 아니라서 위 목록과 별개로 견준다(ADR 0107).
lock 커밋과 HEAD 의 upstream scripts/pin 을 git archive 로 꺼내 harness/pin/fixtures 견본에
돌리고 표면별로 유지/갱신을 낸다. 마지막 줄 `pin 판정: <유지|화면만|PR|실패>` 이 잡이 읽는 신호다.
화면만 바뀐 회차에만 화면 파일을 가져온다. 가져올지는 이 스크립트가 정하고 잡 지시문에 맡기지 않는다.
--pull-page 는 판정과 무관하게 화면을 가져온다. PR 브랜치에서 Go 구현을 고친 뒤 쓰는 수동 스위치다.
"""

import argparse
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
from datetime import datetime

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
UPSTREAM_DIR = os.path.join(ROOT, "harness", "upstream")
# delta 는 upstream CHANGELOG 원문을 인용하므로 공개하지 않는다. ADR 0030.
DELTAS_DIR = os.path.join(ROOT, "private", "deltas")
LOCK_PATH = os.path.join(ROOT, "harness", "upstream.lock")
REPLACEMENTS = os.path.join(ROOT, "private", "vendor-replacements.txt")
CHECK_BOUNDARY = os.path.join(ROOT, "scripts", "check-boundary.py")
PIN_FIXTURES = os.path.join(ROOT, "harness", "pin", "fixtures")
PIN_DRIVER = os.path.join(ROOT, "harness", "pin", "driver.py")
# 가져온 화면을 쓰는 자리. 바이너리가 go:embed 하는 파일이다(ADR 0106).
PAGE_TARGET = os.path.join(ROOT, "internal", "pin", "web", "template.html")
PAGE_REL = "internal/pin/web/template.html"

# 사전 전체가 조직 어휘 목록이라 vendoring 하지 않는다 (ADR 0029).
EXCLUDED = "terminology-normalization.md"
# 사전 내용 대신 표 형식만 뜬 지문이 사는 파일이다.
FORMAT_NAME = "terminology-format.md"
# upstream 이 계약으로 선언했지만 engram 이 한동안 따라가지 않는 명세다.
# 값은 근거 ADR 이다. 여기 든 파일은 vendoring 도, 유지/갱신 판정도 하지 않으므로
# 주간 harness 가 이 파일의 변화로 PR 을 열지 않는다. 풀 때는 이 줄을 지우고 새 ADR 을 쓴다.
# 지금은 비어 있다. pin-rules.md 를 ADR 0107 이 풀었다.
HELD = {}

# engram 이 기대는 절만 떼어 사본으로 둔다. 파일 전체를 가져오지 않으므로 미리알림과
# Hermes 절이 바뀐 주는 `유지` 다(ADR 0107). 값이 None 이면 절 전체, 숫자면 그 수만큼의 첫 문단.
PIN_RULES = "pin-rules.md"
PIN_RULES_SECTIONS = (
    ("위키 문서에서 할 일을 찾을 때", None),
    ("직접 입력과 원장 쓰기", None),
    ("항목·관계·제안", None),
    ("삭제", 1),
)

# 화면을 engram 값으로 바꾼다. 설정 구역의 값 한 줄과 사이트 구역 둘뿐이다.
PAGE_MARKS = ("// pin:config begin", "// pin:config end", "<!-- pin:site begin -->", "<!-- pin:site end -->")
PAGE_CONFIG_RE = re.compile(r"var API='[^']*', SOURCES=\[[^\]]*\], AUTH='[^']*';")
PAGE_CONFIG_ENGRAM = "var API='/pin/api/', SOURCES=['wiki'], AUTH='none';"
# 견본에서 뜬 표면 가운데 상태와 게이트와 상수가 하나라도 다르면 화면을 가져오지 않는다.
PIN_SURFACE_ORDER = ("상태 형태", "상태 값", "게이트", "상수", "화면 요청")


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


IMAGE_LINK_RE = re.compile(r"!\[([^\]]*)\]\(([^)\s]+)\)")


def annotate_asset_links(text):
    """그림 링크를 글 설명으로 바꾼다.

    vendoring 대상은 계약 선언에 든 마크다운 일곱뿐이다. upstream 이 그
    문서 안에서 assets/ 자산을 그림으로 걸면 사본에는 파일이 없어 링크가
    깨진다(check-adr 의 링크 검사). 자산을 함께 가져오는 선택은 하지
    않는다. 계약 선언 밖의 파일을 경계 심사 없이 공개 저장소에 들이는
    일이 되기 때문이다. 대체 텍스트를 산문으로 남겨 무엇을 뜻하는지는
    지킨다.
    """
    def repl(m):
        alt, target = m.group(1).strip(), m.group(2).strip()
        if not alt:
            alt = "그림"
        return f"*({alt}. 원본 그림은 upstream 의 {target} 에 있다)*"
    return IMAGE_LINK_RE.sub(repl, text)


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


def vendor_header(count):
    """치환 사실을 독자에게 알리는 머리말. 원문과 다르다는 것을 알아야 한다.

    **출처 커밋을 여기에 적지 않는다.** 적으면 upstream HEAD 가 움직일 때마다
    본문이 한 글자도 안 바뀌어도 파일이 바뀌어, 파일별 유지/갱신 신호가
    죽는다. 그 신호가 곧 "이 규칙 명세가 실제로 바뀌었나" 이므로 죽이면
    안 된다. 커밋은 harness/upstream.lock 하나가 가진다(ADR 0094).
    """
    line2 = f"익명화 치환 {count}건을 적용했다" if count else "치환 없음"
    return (
        f"<!-- upstream llm-wiki meta/NAME 에서 가져왔다. {line2}.\n"
        f"     출처 커밋은 harness/upstream.lock 에 있다.\n"
        f"     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->\n"
    )


def is_separator_row(line):
    """표의 구분줄인지 모양으로 판정한다. engram 의 glossary 파서와 같은 규칙이다."""
    if not line.startswith("|"):
        return False
    body = line.strip("|")
    if not body.strip():
        return False
    return all(c in "-: |" for c in body)


def terminology_fingerprint(upstream, head):
    """용어 사전의 표 형식만 뽑는다. 사전 내용은 가져오지 않는다.

    engram 의 internal/glossary 가 이 표 구조에 기댄다. 칸이 늘거나 순서가
    바뀌거나 셋째 칸 어휘가 바뀌면 파서가 **조용히** 0건 치환으로 떨어진다.
    그런데 이 파일은 사전 전체가 조직 어휘라 vendoring 대상이 아니므로
    (ADR 0029) 지금까지 아무도 그 변화를 못 봤다.

    형식만 뜨면 경계를 넘지 않는다. 칸 이름은 일반 명사이고, 셋째 칸은
    **첫 낱말만** 본다. 값 전체에는 조직 식별자가 들어 있다.

    머리글은 이름으로 찾지 않고 **다음 줄이 구분줄인 행**으로 찾는다.
    파서가 쓰는 규칙과 같아야 지문이 파서의 전제를 대변한다. 이 파일에는
    표가 여덟 개 있고 전부 같은 머리글이다.

    행 수는 세지 않는다. 사전에 항목 하나만 늘어도 지문이 바뀌면 내용
    변화와 형식 변화를 구분할 수 없게 된다.
    """
    r = git(upstream, "cat-file", "-e", f"{head}:meta/{EXCLUDED}", check=False)
    if r.returncode != 0:
        return None
    text = git(upstream, "show", f"{head}:meta/{EXCLUDED}").stdout
    rows = [l.strip() for l in text.splitlines() if l.strip().startswith("|")]
    headers, firsts = set(), set()
    for i, line in enumerate(rows):
        if is_separator_row(line):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if i + 1 < len(rows) and is_separator_row(rows[i + 1]):
            headers.add(" | ".join(cells))
            continue
        if len(cells) >= 3 and cells[2]:
            firsts.add(cells[2].split()[0].lower())
    if not headers:
        return None
    return (
        "<!-- upstream llm-wiki meta/" + EXCLUDED + " 의 표 형식만 뜬 지문이다.\n"
        "     사전 내용은 조직 어휘라 가져오지 않는다(ADR 0029).\n"
        "     engram 의 internal/glossary 가 이 형식에 기댄다(ADR 0094).\n"
        "     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->\n"
        f"표 머리글: {'; '.join(sorted(headers))}\n"
        f"자동 교정 칸의 첫 낱말: {', '.join(sorted(firsts))}\n"
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
    held = tuple(n[:-3] for n in HELD)  # CHANGELOG 의 파일 절 제목은 "pin-rules: ..." 처럼 확장자 없이 시작한다
    fresh = [e for e in parse_changelog(
        git(upstream, "show", f"{head}:meta/CHANGELOG.md").stdout)
        if e[2] not in old_lines and not (e[1] or "").startswith(held)]

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


# ---------------------------------------------------------------------------
# 규칙 문서 pin-rules.md: engram 이 기대는 절만 뜬다 (ADR 0107)
# ---------------------------------------------------------------------------

def pin_rules_excerpt(text):
    """pin-rules.md 에서 PIN_RULES_SECTIONS 의 절만 떼어 낸다.

    절 제목은 끝의 괄호를 뗀 뒤 글자 그대로 견준다. 기대하는 절이 없으면 ValueError 다.
    조용히 빈 사본을 만들면 upstream 이 절 이름을 바꿨을 때 알 수 없다.
    """
    title = ""
    sections = {}
    cur = None
    for line in text.splitlines():
        if line.startswith("# ") and not title:
            title = line
        elif line.startswith("## "):
            cur = re.sub(r"\s*\(.*\)\s*$", "", line[3:].strip())
            sections[cur] = [line]
        elif cur is not None:
            sections[cur].append(line)
    out = [title] if title else []
    for name, paragraphs in PIN_RULES_SECTIONS:
        if name not in sections:
            raise ValueError(f"기대하는 절 '{name}' 을 찾지 못했습니다")
        body = sections[name]
        while body and not body[-1].strip():
            body = body[:-1]
        if paragraphs is not None:
            # 제목 다음의 첫 문단(빈 줄 전까지)만 둔다.
            kept, seen = [body[0], ""], False
            for line in body[1:]:
                if not line.strip():
                    if seen:
                        break
                    continue
                seen = True
                kept.append(line)
            body = kept
        out.append("")
        out.extend(body)
    return "\n".join(out).rstrip("\n") + "\n"


def pin_rules_header(count):
    line2 = f"익명화 치환 {count}건을 적용했다" if count else "치환 없음"
    return (
        f"<!-- upstream llm-wiki meta/{PIN_RULES} 에서 engram 이 기대는 절만 떼어 왔다. {line2}.\n"
        f"     떼어 온 절: " + ", ".join(n for n, _ in PIN_RULES_SECTIONS) + " (삭제는 첫 문단만)\n"
        f"     출처 커밋은 harness/upstream.lock 에 있다.\n"
        f"     손으로 고치지 않는다. scripts/upstream-sync.py 가 다시 만든다. -->\n"
    )


# ---------------------------------------------------------------------------
# Pin 표면 판정과 화면 가져오기 (ADR 0107)
# ---------------------------------------------------------------------------

def sha12(data):
    return hashlib.sha256(data).hexdigest()[:12]


def pin_archive(upstream, commit):
    """커밋의 scripts/pin 을 임시 디렉터리에 꺼낸다. 없으면 None.

    scripts/wiki_link.py 는 derive.py 가 읽는 공용 파일이라 있으면 함께 꺼낸다. 없다고 실패시키지 않는다.
    없어서 import 가 깨지면 견본에 돌지 못한 것이고 그 경로가 PR 로 보낸다.
    """
    dest = tempfile.mkdtemp(prefix="pin-up-")
    for path, required in (("scripts/pin", True), ("scripts/wiki_link.py", False)):
        r = subprocess.run(["git", "-C", upstream, "archive", commit, path], capture_output=True, check=False)
        if r.returncode != 0:
            if required:
                shutil.rmtree(dest, ignore_errors=True)
                return None
            continue
        with tarfile.open(fileobj=io.BytesIO(r.stdout)) as tar:
            for m in tar.getmembers():
                # 심볼릭 링크, 절대 경로, 상위 경로는 꺼내지 않는다.
                if (m.isfile() or m.isdir()) and not os.path.isabs(m.name) and ".." not in m.name.split("/"):
                    tar.extract(m, dest)
    return dest


def pin_page_requests(html):
    """화면이 보내는 요청: 연산 이름 집합과 API 경로 집합. 설정 구역의 값(API 접두)은 뺀다."""
    ops = set(re.findall(r"\bop:'(\w+)'", html)) | set(re.findall(r"sendAsk\(\w+,'(\w+)'", html))
    paths = set(re.findall(r"getJson\('(\w+)'", html)) | set(re.findall(r"API\+'(\w+)'", html))
    return {"ops": sorted(ops), "paths": sorted(paths)}


def pin_surface(code_dir):
    """코드 한 벌에서 표면을 뜬다. 견본 위키에 upstream 파이썬을 돌려 낸 JSON 이다."""
    pin_dir = os.path.join(code_dir, "scripts", "pin")
    with open(os.path.join(pin_dir, "template.html"), "rb") as f:
        page_bytes = f.read()
    env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1")
    r = subprocess.run(
        [sys.executable, PIN_DRIVER, "surface", pin_dir, PIN_FIXTURES],
        capture_output=True, text=True, env=env, check=False,
    )
    if r.returncode != 0:
        tail = (r.stderr.strip().splitlines() or ["(출력 없음)"])[-1]
        raise RuntimeError(f"견본에 돌지 못했습니다: {tail}")
    surf = json.loads(r.stdout)
    html = page_bytes.decode("utf-8")
    return {
        "화면": sha12(page_bytes),
        "상태 형태": sha12("\n".join(surf["state_shape"]).encode()),
        "상태 값": sha12(json.dumps(surf["state_value"], sort_keys=True).encode()),
        "게이트": sha12(json.dumps(surf["gate"], sort_keys=True, ensure_ascii=False).encode()),
        "상수": sha12(json.dumps(surf["consts"], sort_keys=True).encode()),
        "화면 요청": sha12(json.dumps(pin_page_requests(html), sort_keys=True).encode()),
    }, surf, html


def pin_rules_at(upstream, commit):
    r = git(upstream, "show", f"{commit}:meta/{PIN_RULES}", check=False)
    if r.returncode != 0:
        return None
    try:
        return pin_rules_excerpt(r.stdout)
    except ValueError as e:
        return str(e)


def transform_page(src):
    """upstream 화면을 engram 화면으로 바꾼다. 변환은 둘이다.

    1. 설정 구역(`pin:config` 사이)의 값 한 줄을 engram 값으로 바꾼다.
    2. 사이트 구역(`pin:site` 사이)을 지운다.
    표시 네 개가 파일에 정확히 한 번씩 있어야 하고 값 한 줄도 정확히 한 번 맞아야 한다.
    아니면 ValueError 다. 입력 예시 문구 같은 그 밖의 글은 건드리지 않는다. 경계 검사가 잡는다.
    """
    for m in PAGE_MARKS:
        n = src.count(m)
        if n != 1:
            raise ValueError(f"표시 '{m}' 가 {n}번 있습니다(정확히 한 번이어야 합니다)")
    b, e = src.index(PAGE_MARKS[0]), src.index(PAGE_MARKS[1])
    if b > e:
        raise ValueError("설정 구역의 시작과 끝 순서가 거꾸로입니다")
    block = src[b:e]
    if len(PAGE_CONFIG_RE.findall(block)) != 1:
        raise ValueError("설정 구역에서 값 한 줄(API, SOURCES, AUTH)을 정확히 한 번 찾지 못했습니다")
    out = src[:b] + PAGE_CONFIG_RE.sub(PAGE_CONFIG_ENGRAM, block) + src[e:]
    b, e = out.index(PAGE_MARKS[2]), out.index(PAGE_MARKS[3])
    if b > e:
        raise ValueError("사이트 구역의 시작과 끝 순서가 거꾸로입니다")
    return out[:b] + out[e + len(PAGE_MARKS[3]):]


def pull_page(html):
    """변환한 화면을 내장 자리에 쓰고 경계 검사를 돈다. 걸리면 이전 파일로 되돌리고 False 를 낸다."""
    new = transform_page(html)
    old = None
    if os.path.exists(PAGE_TARGET):
        with open(PAGE_TARGET, "rb") as f:
            old = f.read()
    if old == new.encode("utf-8"):
        return "같음", True
    os.makedirs(os.path.dirname(PAGE_TARGET), exist_ok=True)
    with open(PAGE_TARGET, "w", encoding="utf-8", newline="") as f:
        f.write(new)
    code, out = run_boundary()
    if code != 0 and any(l.lstrip().startswith(PAGE_REL) for l in out.splitlines()):
        print(out)
        if old is None:
            os.remove(PAGE_TARGET)
        else:
            with open(PAGE_TARGET, "wb") as f:
                f.write(old)
        return "경계 검사 적중", False
    return "바뀜", True


def judge_pin(upstream, base_commit, head, check, force_pull):
    """Pin 표면을 lock 커밋과 HEAD 에서 떠 견주고 판정을 낸다. 판정은 유지, 화면만, PR 셋이다.

    출력은 한 줄에 하나다. 마지막 줄 `pin 판정: <값>` 이 잡이 읽는 고정 문자열이다.
    돌지 못한 표면이 하나라도 있으면 PR 이다. 화면 가져오기에 실패하면 exit 1 이다.
    """
    verdict, reasons = None, []
    base_dir = head_dir = None
    try:
        if base_commit is None:
            reasons.append("lock 이 없어 견줄 기준이 없습니다")
        else:
            base_dir = pin_archive(upstream, base_commit)
            if base_dir is None:
                reasons.append(f"lock 커밋 {base_commit[:7]} 에 scripts/pin 이 없습니다")
        head_dir = pin_archive(upstream, head)
        if head_dir is None:
            reasons.append(f"HEAD {head[:7]} 에 scripts/pin 이 없습니다")
        if reasons:
            for r in reasons:
                print(f"  돌지 못함 pin: {r}")
            print("pin 판정: PR")
            return 0

        try:
            base, _, base_html = pin_surface(base_dir)
            cur, _, head_html = pin_surface(head_dir)
        except (RuntimeError, ValueError, OSError) as e:
            print(f"  돌지 못함 pin 표면: {e}")
            print("pin 판정: PR")
            return 0

        base_rules, head_rules = pin_rules_at(upstream, base_commit), pin_rules_at(upstream, head)
        changed = {}
        for name in ("화면", *PIN_SURFACE_ORDER):
            changed[name] = base[name] != cur[name]
            if changed[name]:
                print(f"  갱신 pin {name} ({base[name]} 에서 {cur[name]})")
            else:
                print(f"  유지 pin {name} ({cur[name]})")
        changed["규칙 문서"] = base_rules != head_rules
        print(f"  {'갱신' if changed['규칙 문서'] else '유지'} pin 규칙 문서 ({PIN_RULES} 의 기대하는 절)")
        others = [n for n in (*PIN_SURFACE_ORDER, "규칙 문서") if changed[n]]

        if others:
            verdict = "PR"
            print(f"  화면은 가져오지 않습니다. 바뀐 표면: {', '.join(others)}")
        elif changed["화면"]:
            verdict = "화면만"
        else:
            verdict = "유지"

        if verdict == "화면만" or force_pull:
            try:
                transformed = transform_page(head_html)
            except ValueError as e:
                print(f"실패: 화면을 가져오지 않습니다. {e}")
                return 1
            have = open(PAGE_TARGET, "rb").read() if os.path.exists(PAGE_TARGET) else None
            same = have == transformed.encode("utf-8")
            if check:
                print(f"  화면 가져오기 예정 {PAGE_REL} (내장본과 {'같음' if same else '다름'})")
            else:
                try:
                    state, ok = pull_page(head_html)
                except ValueError as e:
                    print(f"실패: 화면을 가져오지 않습니다. {e}")
                    return 1
                if not ok:
                    print(f"실패: 가져온 화면이 {state} 로 거부됐습니다. 이전 화면을 유지합니다")
                    return 1
                print(f"  화면 가져옴 {PAGE_REL} ({state})")
        print(f"pin 판정: {verdict}")
        return 0
    finally:
        for d in (base_dir, head_dir):
            if d:
                shutil.rmtree(d, ignore_errors=True)


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
    ap.add_argument("--pull-page", action="store_true",
                    help="Pin 판정과 무관하게 HEAD 의 화면을 가져온다(PR 브랜치에서 Go 구현을 고친 뒤 쓴다)")
    args = ap.parse_args()

    upstream = os.path.abspath(os.path.expanduser(args.upstream))
    r = git(upstream, "rev-parse", "--verify", "HEAD")
    head = r.stdout.strip()
    short = head[:7]

    names = read_contract_list(upstream, head)
    if EXCLUDED in names:
        names.remove(EXCLUDED)
        print(f"{EXCLUDED} 는 사전 전체가 조직 어휘 목록이라 제외합니다 (ADR 0029)")
    for name in sorted(HELD):
        if name in names:
            names.remove(name)
            print(f"{name} 는 따라가지 않기로 보류한 명세라 제외합니다 ({HELD[name]})")
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
        if name == PIN_RULES:
            # 파일 전체가 아니라 engram 이 기대는 절만 가져온다(ADR 0107).
            try:
                src = pin_rules_excerpt(src)
            except ValueError as e:
                die(f"{name}: {e}. upstream 이 절 이름을 바꿨는지 확인하십시오")
        body, n = apply_replacements(src, entries)
        body = annotate_asset_links(body)
        total_subs += n
        header = pin_rules_header(n) if name == PIN_RULES else vendor_header(n).replace("NAME", name)
        text = header + body
        target = os.path.join(UPSTREAM_DIR, name)
        if args.check:
            exists = os.path.exists(target)
            same = exists and open(target, encoding="utf-8").read() == text
            state = "유지" if same else ("갱신" if exists else "신규")
            print(f"  {state} {name} (치환 {n}건)")
        else:
            if write_if_changed(target, text):
                changed_files.append(name)
    fp = terminology_fingerprint(upstream, head)
    fp_target = os.path.join(UPSTREAM_DIR, FORMAT_NAME)
    if fp is None:
        fp_state = "없음"
    else:
        fp_exists = os.path.exists(fp_target)
        fp_same = fp_exists and open(fp_target, encoding="utf-8").read() == fp
        fp_state = "유지" if fp_same else ("갱신" if fp_exists else "신규")

    if args.check:
        print(f"  {fp_state} {FORMAT_NAME} (사전 표 형식)")
        print(f"총 치환 {total_subs}건 (사전 {dict_size}항목)")
        if old_lock is None:
            print(f"lock: 신규 (커밋 {short})")
        elif old_lock.get("commit") == head:
            print(f"lock: 유지 (커밋 {short})")
        else:
            print(f"lock: 갱신 ({old_lock['commit'][:7]} 에서 {short})")
            _, b, w, _ = build_delta(upstream, old_lock["commit"], head, entries)
            print(f"delta: private/deltas/{short}.md 생성 (binary-affecting {b}건, wiki-only {w}건)")
        return judge_pin(upstream, old_lock["commit"] if old_lock else None, head, True, args.pull_page)

    if fp is not None and write_if_changed(fp_target, fp):
        changed_files.append(FORMAT_NAME)
    written = [n for n in names]
    print(f"harness/upstream/ 에 {len(written)}개를 맞췄습니다 (치환 {total_subs}건)")
    if changed_files:
        print("  갱신: " + ", ".join(changed_files))
    else:
        print("  본문 변화 없음. 규칙 명세가 그대로다")

    # 목록에서 빠진 파일은 계약이 아니게 된 것이므로 남기지 않는다.
    stale = []
    if os.path.isdir(UPSTREAM_DIR):
        for name in os.listdir(UPSTREAM_DIR):
            if name not in names and name != FORMAT_NAME:
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

    code = judge_pin(upstream, old_lock["commit"] if old_lock else None, head, False, args.pull_page)
    if code != 0:
        print("lock 을 쓰지 않았습니다")
        return code

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
