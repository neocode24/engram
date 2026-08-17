package eject

import (
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
)

// linterScript는 문서 단위 규칙을 판정하는 Python 린터를 낸다.
// 판정에 쓰는 속성, 허용값, 임계값, 디렉토리는 engram.yaml 을 실행 시점에
// 읽는다. 설정을 바꾸면 스크립트를 다시 만들지 않아도 판정이 따라간다.
// engram.yaml 이 고칠 수 없는 값(고정 허용 집합, 단계와 디렉토리의
// 대응)만 상수로 싣는다. 프리셋 이름은 주석에 남겨 어느 위키에서
// 만들어졌는지 드러낸다.
func linterScript(cfg config.Config, dirs map[string]string) string {
	var stageDirEntries []string
	for _, pair := range stageDirsSorted(dirs) {
		stageDirEntries = append(stageDirEntries, fmt.Sprintf("%s: %s", pyString(pair[0]), pyString(pair[1])))
	}
	defaults := fmt.Sprintf(`    "preset": %s,
    "types": %s,
    "topics": %s,
    "forms": %s,
    "min_wikilinks": %s,
    "stale_days": %s,
    "max_lines": %s,
    "broad_topic_pct": %s,
    "page_dirs": %s,
    "root_files": %s,
    "ignore_files": %s,`,
		pyString(string(cfg.Preset)), pyList(cfg.Schema.Types), pyList(cfg.Schema.Taxonomy.Topics.Values),
		pyList(cfg.Schema.Taxonomy.Forms.Values), pyInt(cfg.Thresholds.MinWikilinks), pyInt(cfg.Thresholds.StaleDays),
		pyInt(cfg.Thresholds.MaxLines), pyInt(cfg.Thresholds.BroadTopicPct), pyList(cfg.PageDirs),
		pyList(cfg.RootFiles), pyList(cfg.IgnoreFiles))

	return fmt.Sprintf(`#!/usr/bin/env python3
"""문서 단위 규칙을 판정하는 린터. engram eject 가 만들었다.

속성, 허용값, 임계값, 디렉토리는 engram.yaml 을 실행 시점에 읽는다.
설정을 바꾸면 이 스크립트를 다시 만들지 않아도 판정이 따라간다.
engram.yaml 이 고칠 수 없는 값은 아래 상수로 받았다. 고정 허용 집합과
단계와 디렉토리의 대응이 그것이다.

이 린터가 내보내지 않는 것:
- wiki.broad-topic. 위키 전체의 통계로 판정하는 진단이라 문서 하나를
  보는 훅의 자리가 아니다. engram lint 가 계속 판정한다.
- 검색 색인, 재발견, 링크 그래프 계산, 다이제스트. 연산은 파일로
  표현되지 않는다. engram search, recall, resurface, bridge, digest,
  backlinks 가 계속 수행한다.

이 위키는 eject 시점에 %s 프리셋을 썼다.

사용법: python3 scripts/lint-frontmatter.py [위키루트]
종료 코드는 engram lint 와 같다. error 나 reject 가 있으면 1, 그 밖에는
0 이다. 경고는 종료 코드에 영향을 주지 않는다. 이 스크립트를 부르는
커밋 훅이 경고만 있는 위키를 막으면 안 되기 때문이다.
"""
import os
import re
import sys

# Windows 콘솔의 기본 인코딩은 UTF-8 이 아니라 cp949 나 cp1252 다. 한글
# 메시지를 그대로 내면 UnicodeEncodeError 로 죽는다. engram 본체는 콘솔
# 코드페이지를 UTF-8 로 바꿔서 푸는데, 내보낸 이 스크립트는 그 처리를
# 받지 못하므로 여기서 스트림을 다시 연다.
# newline 도 함께 고정한다. Windows 의 텍스트 모드는 \n 을 \r\n 으로
# 바꿔 내보내는데, engram lint 는 \n 만 내므로 그대로 두면 두 린터의
# 출력이 줄바꿈에서 갈린다. 판정이 같아야 한다는 계약이 깨진다.
for _stream in (sys.stdout, sys.stderr):
    if hasattr(_stream, "reconfigure"):
        _stream.reconfigure(encoding="utf-8", newline="\n")

# --- 고정 상수. engram.yaml 이 바꿀 수 없는 값이다. ---
ARTIFACT_STAGES = %s
STATUSES = %s
SCOPES = %s
SENSITIVITIES = %s
TRIGGER_MODES = %s

# 단계와 디렉토리의 대응.
STAGE_DIRS = {%s}

# 프리셋별 속성 기본값. minimal 이 personal 에, personal 이 team 에
# 포함된다. engram.yaml 의 axes 가 개별 속성을 덮어 쓴다.
BASE_AXES = %s
PERSONAL_AXES = BASE_AXES + %s
TEAM_AXES = PERSONAL_AXES + %s
PRESET_AXES = {
    "minimal": set(BASE_AXES),
    "personal": set(PERSONAL_AXES),
    "team": set(TEAM_AXES),
}

AXIS_NAMES = BASE_AXES + %s

# engram.yaml 에 키가 없을 때의 기본값. eject 시점의 설정에서 왔다.
DEFAULTS = {
%s
}

WIKI_LINK_RE = re.compile(r"\[\[([^[\]]+)\]\]")
DAY_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-$")
MONTH_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-$")

# 백틱 문자. 코드 펜스와 인라인 코드 판정에 쓴다.
BACKTICK = chr(96)
FENCE = BACKTICK * 3


# --- 설정 읽기 ---

def unquote(s):
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] and s[0] in "\"'":
        return s[1:-1]
    return s


def strip_comment(s):
    out = []
    quote = None
    for ch in s:
        if quote:
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
        elif ch == "#":
            break
        out.append(ch)
    return "".join(out).strip()


def parse_scalar(s):
    s = s.strip()
    if s.startswith("[") and s.endswith("]"):
        inner = s[1:-1].strip()
        if not inner:
            return []
        return [unquote(x) for x in inner.split(",")]
    if s == "true":
        return True
    if s == "false":
        return False
    if re.fullmatch(r"-?\d+", s):
        return int(s)
    return unquote(s)


def parse_simple_yaml(text):
    """프론트매터와 engram.yaml 이 쓰는 YAML 부분집합을 파싱한다.

    키: 값, 흐름 목록 [a, b], 블록 목록, 한 단계 맵을 다룬다.
    PyYAML 을 쓰지 않는다. 표준 라이브러리만 쓴다는 eject 의 계약이다.
    """
    out = {}
    lines = text.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    i = 0
    while i < len(lines):
        stripped = lines[i].strip()
        i += 1
        if not stripped or stripped.startswith("#"):
            continue
        m = re.match(r"^([^\s:]+):\s*(.*)$", stripped)
        if not m:
            continue
        key, rest = m.group(1), strip_comment(m.group(2))
        if rest == "":
            items, sub = [], {}
            while i < len(lines):
                nxt = lines[i]
                if not nxt.strip() or nxt.strip().startswith("#"):
                    i += 1
                    continue
                if not nxt.startswith((" ", "\t")):
                    break
                i += 1
                t = nxt.strip()
                if t.startswith("- "):
                    items.append(parse_scalar(t[2:]))
                elif t == "-":
                    items.append("")
                else:
                    sm = re.match(r"^([^\s:]+):\s*(.*)$", t)
                    if sm:
                        sub[sm.group(1)] = parse_scalar(strip_comment(sm.group(2)))
            if sub:
                out[key] = sub
            elif items:
                out[key] = items
            else:
                # 값이 없는 키는 빈 값으로 둔다. 키의 존재 자체가 판정
                # 대상이다(source_channel: 처럼).
                out[key] = ""
            continue
        out[key] = parse_scalar(rest)
    return out


def load_config(root):
    cfg = dict(DEFAULTS)
    for k, v in cfg.items():
        if isinstance(v, list):
            cfg[k] = list(v)
    axes = set(PRESET_AXES.get(cfg["preset"], ()))
    path = os.path.join(root, "engram.yaml")
    if os.path.exists(path):
        with open(path, encoding="utf-8-sig") as f:
            raw = parse_simple_yaml(f.read())
        for key, val in raw.items():
            if key == "preset":
                cfg["preset"] = val
                axes = set(PRESET_AXES.get(val, axes))
            elif key == "axes":
                for name, on in val.items():
                    axes[name] = bool(on)
            elif key in cfg:
                cfg[key] = val
    cfg["axes"] = axes
    return cfg


# --- 문서 파싱 ---

def read_text(path):
    with open(path, "rb") as f:
        text = f.read().decode("utf-8-sig")
    return text.replace("\r\n", "\n").replace("\r", "\n")


def line_of_key(text, key):
    for i, line in enumerate(text.split("\n")):
        if line.lstrip(" \t").startswith(key + ":"):
            return i + 1
    return 1


def line_count(text):
    if text == "":
        return 0
    return text.count("\n") + (0 if text.endswith("\n") else 1)


def slug_of(content):
    for sep in ("#", "|"):
        idx = content.find(sep)
        if idx >= 0:
            content = content[:idx]
    return content.strip()


def strip_brackets(v):
    v = v.strip()
    if v.startswith("[["):
        v = v[2:]
    if v.endswith("]]"):
        v = v[:-2]
    return v


def outside_code(line):
    parts = line.split(BACKTICK)
    return [p for i, p in enumerate(parts) if i %% 2 == 0]


def body_links(body, body_line):
    links = []
    in_fence = False
    for i, line in enumerate(body.split("\n")):
        if line.lstrip(" \t").startswith(FENCE):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        for seg in outside_code(line):
            for m in WIKI_LINK_RE.finditer(seg):
                slug = slug_of(m.group(1))
                if slug:
                    links.append((slug, body_line + i))
    return links


def related_lines(fm_lines):
    key_line = 2
    item_lines = []
    in_list = False
    for i, line in enumerate(fm_lines):
        t = line.strip()
        if not in_list:
            if line.startswith("related:"):
                key_line = i + 2
                in_list = True
            continue
        if t.startswith("-"):
            item_lines.append(i + 2)
            continue
        break
    return key_line, item_lines


def relation_slug(v):
    v = strip_brackets(v)
    base = v.split("/")[-1]
    if base.endswith(".md"):
        base = base[:-3]
    if len(base) > 11 and DAY_PREFIX_RE.match(base[:11]):
        return base[11:]
    if len(base) > 8 and MONTH_PREFIX_RE.match(base[:8]):
        return base[8:]
    return base


def required_fields(stage, axes):
    req = ["type", "artifact_stage", "status", "indexable"]
    for f in ("scope", "sensitivity", "source_channel", "trigger_mode", "workflow"):
        if f in axes:
            req.append(f)
    if stage == "source":
        for f in ("source_refs", "derived_from", "derived_context"):
            if f in axes:
                req.append(f)
        req.extend(["created", "sourced_at"])
    elif stage == "context":
        for f in ("source_refs", "derived_from", "related"):
            if f in axes:
                req.append(f)
    return req


# --- 순회 ---

def walk_docs(root, cfg):
    rels = []
    for d in sorted(cfg["page_dirs"]):
        if d.startswith("."):
            continue
        base = os.path.join(root, d)
        if not os.path.isdir(base):
            continue
        for dirpath, dirnames, filenames in os.walk(base):
            dirnames[:] = [x for x in dirnames if not x.startswith(".")]
            for name in filenames:
                if name.endswith(".md") and name not in cfg["ignore_files"]:
                    full = os.path.join(dirpath, name)
                    rel = os.path.relpath(full, root).replace(os.sep, "/")
                    rels.append(rel)
    for f in cfg["root_files"]:
        if f.endswith(".md") and os.path.exists(os.path.join(root, f)):
            rels.append(f.replace(os.sep, "/"))
    return sorted(set(rels))


# --- 판정 ---

class Violation:
    def __init__(self, path, line, rule, severity, message, fix):
        self.path = path
        self.line = line
        self.rule = rule
        self.severity = severity
        self.message = message
        self.fix = fix


def stage_for_dir(dirname):
    for stage, d in STAGE_DIRS.items():
        if d == dirname:
            return stage
    return None


def check_docs(root, cfg):
    violations = []
    docs = []

    def add(path, line, rule, severity, message, fix):
        violations.append(Violation(path, line, rule, severity, message, fix))

    for rel in walk_docs(root, cfg):
        path = os.path.join(root, rel.replace("/", os.sep))
        text = read_text(path)
        lines = text.split("\n")
        doc = {"rel": rel, "root": rel in cfg["root_files"], "links": [],
               "relation_values": [], "fields": {}, "stage": None,
               "fm_lines": [], "text": text}
        if lines and lines[0].rstrip(" \t") == "---":
            close = -1
            for i in range(1, len(lines)):
                if lines[i].rstrip(" \t") == "---":
                    close = i
                    break
            if close < 0:
                add(rel, 1, "frontmatter.unclosed", "error",
                    "프론트매터가 닫는 --- 구분자 없이 끝났습니다",
                    "프론트매터 끝에 --- 줄을 추가하세요")
                continue
            fm_text = "\n".join(lines[1:close])
            doc["fields"] = parse_simple_yaml(fm_text)
            doc["fm_lines"] = lines[1:close]
            body = "\n".join(lines[close + 1:])
            body_line = close + 2
            doc["links"] = body_links(body, body_line)
        else:
            add(rel, 1, "frontmatter.missing", "error",
                "프론트매터가 없습니다",
                "문서 첫 줄에 --- 로 여는 구분자를 두고 필드를 채운 뒤 --- 로 닫으세요")
            continue

        fields = doc["fields"]
        stage = fields.get("artifact_stage")
        # YAML 이 숫자로 읽는 값도 engram 은 문자열로 다룬다.
        if isinstance(stage, int) and not isinstance(stage, bool):
            stage = str(stage)
        if isinstance(stage, str):
            doc["stage"] = stage

        if "artifact_stage" not in fields:
            # artifact_stage 는 단계 판정의 입력이다. 없으면 그 자체가
            # 오류이고 어느 단계인지 모르므로 다른 필수 필드는 보고하지
            # 않는다(ADR 0040).
            if "artifact_stage" in cfg["axes"]:
                add(rel, line_of_key(text, "artifact_stage"), "frontmatter.missing-field", "error",
                    "artifact_stage 필드가 없습니다",
                    "프론트매터에 artifact_stage 필드를 채우세요")
        elif stage:
            for f in required_fields(stage, cfg["axes"]):
                if f not in fields:
                    add(rel, line_of_key(text, "artifact_stage"), "frontmatter.missing-field", "error",
                        "단계 %%s의 필수 필드 %%s가 없습니다" %% (stage, f),
                        "프론트매터에 %%s 필드를 추가하세요" %% f)

        for key, values in (("artifact_stage", ARTIFACT_STAGES), ("status", STATUSES),
                            ("scope", SCOPES), ("sensitivity", SENSITIVITIES),
                            ("trigger_mode", TRIGGER_MODES)):
            if key not in cfg["axes"]:
                continue
            val = fields.get(key)
            if isinstance(val, int) and not isinstance(val, bool):
                val = str(val)
            if not isinstance(val, str) or val == "":
                continue
            if val not in values:
                add(rel, line_of_key(text, key), "schema.allowed-value", "error",
                    "%%s 값이 허용값 밖입니다: \"%%s\" (허용값: %%s)" %% (key, val, ", ".join(values)),
                    "%%s 값을 허용값 중 하나로 바꿉니다" %% key)

        if stage and not doc["root"]:
            top = rel.split("/")[0]
            expected = stage_for_dir(top)
            if expected and stage != expected:
                severity = "error" if stage == "context" else "warn"
                add(rel, line_of_key(text, "artifact_stage"), "location.stage-agreement", severity,
                    "문서가 %%s 디렉토리에 있지만 artifact_stage가 \"%%s\"입니다" %% (top, stage),
                    "문서를 artifact_stage에 맞는 디렉토리로 옮기거나 artifact_stage를 %%s로 고치세요. 문서를 옮길 때는 engram promote, demote, archive를 쓰세요" %% expected)

        for axis in AXIS_NAMES:
            if axis not in cfg["axes"] and axis in fields:
                add(rel, line_of_key(text, axis), "schema.axis-off", "error",
                    "설정에서 꺼진 속성이 문서에 있습니다: %%s (프리셋 %%s)" %% (axis, cfg["preset"]),
                    "engram.yaml의 axes에서 %%s를 켜거나 문서에서 %%s 필드를 지웁니다" %% (axis, axis))

        form = fields.get("form")
        if isinstance(form, str) and form != "":
            forms = cfg["forms"]
            if forms and form not in forms:
                add(rel, line_of_key(text, "form"), "taxonomy.forms", "error",
                    "form 값이 forms 폐쇄 집합에 없습니다: \"%%s\" (허용값: %%s)" %% (form, ", ".join(forms)),
                    "form 값을 허용값 중 하나로 바꿉니다")
        topics = fields.get("topics")
        if isinstance(topics, list):
            for v in topics:
                if v not in cfg["topics"]:
                    add(rel, line_of_key(text, "topics"), "taxonomy.topics", "warn",
                        "topics 값이 설정에 정의되지 않았습니다: \"%%s\" (topics는 개방 집합입니다)" %% v,
                        "engram.yaml의 topics 목록에 \"%%s\"를 추가하세요" %% v)

        sources_dir = STAGE_DIRS.get("source", "sources")
        if "updated" in fields and (rel == sources_dir or rel.startswith(sources_dir + "/")):
            add(rel, line_of_key(text, "updated"), "sources.updated", "warn",
                "sources 계층 문서에 updated 필드가 있습니다",
                "updated 필드를 지우세요. sources는 원본 보존 계층이라 갱신하지 않습니다")

        if line_count(text) > cfg["max_lines"]:
            add(rel, line_of_key(text, "artifact_stage"), "body.max-lines", "warn",
                "문서가 %%d줄로 max_lines %%d줄을 넘습니다" %% (line_count(text), cfg["max_lines"]),
                "문서를 나누세요. 상한은 engram.yaml의 max_lines로 조정하세요")

        # related 필드의 링크. 파싱된 값을 진실원으로 쓰고 줄 번호만 원문에서 잡는다.
        related = fields.get("related")
        if isinstance(related, list):
            key_line, item_lines = related_lines(doc["fm_lines"])
            for i, v in enumerate(related):
                line = key_line if i >= len(item_lines) else item_lines[i]
                doc["links"].append((strip_brackets(v), line))

        for key in ("derived_from", "derived_context", "source_refs"):
            v = fields.get(key)
            if isinstance(v, list):
                doc["relation_values"].extend(v)
            elif isinstance(v, str) and v != "":
                doc["relation_values"].append(v)
        docs.append(doc)

    violations.extend(graph_rules(docs, cfg))
    return violations


def linkable(doc):
    if doc["root"]:
        return True
    if doc["stage"] is None:
        return False
    return doc["stage"] != "inbox"


def evaluate_gate(links, targets, min_wikilinks):
    if min_wikilinks <= 0:
        return True, False
    if targets < min_wikilinks:
        return True, True
    return links >= min_wikilinks, False


def graph_rules(docs, cfg):
    violations = []
    by_slug = {}
    for doc in docs:
        slug = doc["rel"].split("/")[-1]
        if slug.endswith(".md"):
            slug = slug[:-3]
        if slug not in by_slug:
            by_slug[slug] = doc["rel"]
    incoming = {}
    for doc in docs:
        self_key = relation_slug(doc["rel"])
        for slug, _line in doc["links"]:
            key = relation_slug(slug)
            if key != self_key:
                incoming[key] = incoming.get(key, 0) + 1
            if slug not in by_slug:
                violations.append(Violation(
                    doc["rel"], _line, "link.broken", "warn",
                    "깨진 위키링크: [[%%s]]에 해당하는 문서가 없습니다" %% slug,
                    "슬러그를 고치거나 [[%%s]] 문서를 만드세요" %% slug))
        for v in doc["relation_values"]:
            key = relation_slug(v)
            if key != "" and key != self_key:
                incoming[key] = incoming.get(key, 0) + 1

    min_wikilinks = cfg["min_wikilinks"]
    context_dir = STAGE_DIRS.get("context")
    for doc in docs:
        if doc["root"]:
            continue
        slug = doc["rel"].split("/")[-1]
        if slug.endswith(".md"):
            slug = slug[:-3]
        outgoing = set(slug for slug, _ in doc["links"])
        has_relations = len(doc["relation_values"]) > 0
        if not outgoing and not has_relations and incoming.get(relation_slug(doc["rel"]), 0) == 0:
            violations.append(Violation(
                doc["rel"], 1, "graph.orphan", "warn",
                "들어오는 관계와 나가는 관계가 모두 없습니다",
                "다른 문서의 related나 본문에서 [[%%s]]로 연결하거나 관계 필드로 잇으세요" %% slug))
        # 게이트는 문서가 놓인 디렉토리로 발동한다(ADR 0040). 선언을 보면
        # 값을 비우거나 낮춰 우회할 수 있다.
        if context_dir and doc["rel"].startswith(context_dir + "/") and min_wikilinks > 0:
            n = len(outgoing)
            targets = sum(1 for d in docs if d["rel"] != doc["rel"] and linkable(d))
            passed, deferred = evaluate_gate(n, targets, min_wikilinks)
            # 줄 번호는 related 키가 있는 줄로 잡는다. engram lint 와 같다.
            gate_line = line_of_key(doc["text"], "related")
            if deferred and n < min_wikilinks:
                violations.append(Violation(
                    doc["rel"], gate_line, "gate.deferred", "warn",
                    "링크 가능한 대상 문서가 %%d개로 min_wikilinks %%d개보다 적어 게이트를 유예합니다. 대상 문서가 %%d개가 되면 게이트가 동작합니다" %% (targets, min_wikilinks, min_wikilinks),
                    "연결할 문서를 만들어 대상을 늘리세요. 기준은 engram.yaml의 min_wikilinks로 조정하세요"))
                continue
            if not passed:
                violations.append(Violation(
                    doc["rel"], gate_line, "gate.min-wikilinks", "reject",
                    "위키링크가 %%d개로 min_wikilinks %%d개에 못 미칩니다" %% (n, min_wikilinks),
                    "related 필드나 본문에 위키링크를 %%d개 더 추가하세요" %% (min_wikilinks - n)))
    return violations


def main():
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    cfg = load_config(root)
    violations = check_docs(root, cfg)
    violations.sort(key=lambda v: (v.path, v.line, v.rule))
    current = None
    counts = {"error": 0, "warn": 0, "reject": 0}
    for v in violations:
        if v.path != current:
            print(v.path)
            current = v.path
        print("  [%%s] %%d %%s" %% (v.severity, v.line, v.rule))
        print("    %%s" %% v.message)
        print("    고치는 법: %%s" %% v.fix)
        counts[v.severity] += 1
    files = len(walk_docs(root, cfg))
    print("검사한 파일 %%d개, error %%d, warn %%d, reject %%d" %% (files, counts["error"], counts["warn"], counts["reject"]))
    # 종료 코드의 규칙은 engram lint 의 HasBlocking 과 같다. error 나
    # reject 가 있어야 1 이다. 경고만으로 커밋을 막지 않는다.
    if counts["error"] > 0 or counts["reject"] > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
`,
		string(cfg.Preset),
		pyList(cfg.Schema.ArtifactStages.Values), pyList(cfg.Schema.Statuses.Values),
		pyList(cfg.Schema.Scopes.Values), pyList(cfg.Schema.Sensitivities.Values),
		pyList(cfg.Schema.TriggerModes.Values),
		strings.Join(stageDirEntries, ",\n    "),
		pyList(baseAxes()), pyList(personalExtra()), pyList(teamExtra()),
		pyList(extraAxes()),
		defaults)
}

// baseAxes는 모든 프리셋이 켜는 속성이다. config 의 프리셋 정의에서 온다.
func baseAxes() []string {
	return []string{
		string(config.AxisType), string(config.AxisArtifactStage), string(config.AxisStatus),
		string(config.AxisIndexable), string(config.AxisTags), string(config.AxisSourceRefs),
		string(config.AxisDerivedFrom), string(config.AxisRelated),
	}
}

// personalExtra는 personal 프리셋이 추가로 켜는 속성이다.
func personalExtra() []string {
	return []string{string(config.AxisSourceChannel), string(config.AxisDerivedContext)}
}

// teamExtra는 team 프리셋이 추가로 켜는 속성이다.
func teamExtra() []string {
	return []string{string(config.AxisScope), string(config.AxisSensitivity),
		string(config.AxisTriggerMode), string(config.AxisWorkflow)}
}

// extraAxes는 BASE_AXES 에 없는 나머지 속성이다. 속성 전체를 나열하는 데 쓴다.
func extraAxes() []string {
	return append(append([]string{}, personalExtra()...), teamExtra()...)
}

// pyString은 문자열을 Python 문자열 리터럴로 낸다.
func pyString(s string) string {
	return fmt.Sprintf("%q", s)
}
