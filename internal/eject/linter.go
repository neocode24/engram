package eject

import (
	"fmt"
	"strings"

	"github.com/neocode24/engram/internal/config"
	"github.com/neocode24/engram/internal/i18n"
)

// linterScript는 문서 단위 규칙을 판정하는 Python 린터를 낸다.
// 판정에 쓰는 속성, 허용값, 임계값, 디렉토리는 engram.yaml 을 실행 시점에
// 읽는다. 설정을 바꾸면 스크립트를 다시 만들지 않아도 판정이 따라간다.
// engram.yaml 이 고칠 수 없는 값(고정 허용 집합, 단계와 디렉토리의
// 대응)만 상수로 싣는다. 프리셋 이름은 주석에 남겨 어느 위키에서
// 만들어졌는지 드러낸다.
//
// 산출물의 모든 한국어는 카탈로그에서 얻는다. 내보내는 시점 언어로
// 굳고 그 뒤로는 engram 과 무관하게 산다(ADR 0049). Python 안에서
// 채우는 %%s, %%d 포맷은 Go 에서 서식으로 다루지 않도록 인자 없는
// i18n.T 로 꺼낸 뒤 pyString 으로 리터럴에 박는다.
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

	var b strings.Builder
	b.WriteString("#!/usr/bin/env python3\n")
	b.WriteString(`"""` + i18n.T("eject.linter.docstring", string(cfg.Preset)) + `"""
`)
	b.WriteString(`import os
import re
import sys

`)
	b.WriteString(i18n.T("eject.linter.console_comment"))
	b.WriteString(i18n.T("eject.linter.newline_comment"))
	b.WriteString(`for _stream in (sys.stdout, sys.stderr):
    if hasattr(_stream, "reconfigure"):
        _stream.reconfigure(encoding="utf-8", newline="\n")

`)
	b.WriteString(i18n.T("eject.linter.constants_comment"))
	b.WriteString(fmt.Sprintf("ARTIFACT_STAGES = %s\n", pyList(cfg.Schema.ArtifactStages.Values)))
	b.WriteString(fmt.Sprintf("STATUSES = %s\n", pyList(cfg.Schema.Statuses.Values)))
	b.WriteString(fmt.Sprintf("SCOPES = %s\n", pyList(cfg.Schema.Scopes.Values)))
	b.WriteString(fmt.Sprintf("SENSITIVITIES = %s\n", pyList(cfg.Schema.Sensitivities.Values)))
	b.WriteString(fmt.Sprintf("TRIGGER_MODES = %s\n", pyList(cfg.Schema.TriggerModes.Values)))
	b.WriteString("\n")
	b.WriteString(i18n.T("eject.linter.stage_dirs_comment"))
	b.WriteString(fmt.Sprintf("STAGE_DIRS = {%s}\n", strings.Join(stageDirEntries, ",\n    ")))
	b.WriteString("\n")
	b.WriteString(i18n.T("eject.linter.preset_axes_comment"))
	b.WriteString(fmt.Sprintf("BASE_AXES = %s\n", pyList(baseAxes())))
	b.WriteString(fmt.Sprintf("PERSONAL_AXES = BASE_AXES + %s\n", pyList(personalExtra())))
	b.WriteString(fmt.Sprintf("TEAM_AXES = PERSONAL_AXES + %s\n", pyList(teamExtra())))
	b.WriteString(`PRESET_AXES = {
    "minimal": set(BASE_AXES),
    "personal": set(PERSONAL_AXES),
    "team": set(TEAM_AXES),
}

`)
	b.WriteString(fmt.Sprintf("AXIS_NAMES = BASE_AXES + %s\n", pyList(extraAxes())))
	b.WriteString("\n")
	b.WriteString(i18n.T("eject.linter.defaults_comment"))
	b.WriteString(fmt.Sprintf("DEFAULTS = {\n%s\n}\n", defaults))
	b.WriteString(`
WIKI_LINK_RE = re.compile(r"\[\[([^[\]]+)\]\]")
DAY_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-$")
MONTH_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-$")

`)
	b.WriteString(i18n.T("eject.linter.backtick_comment"))
	b.WriteString(`BACKTICK = chr(96)
FENCE = BACKTICK * 3


`)
	b.WriteString(i18n.T("eject.linter.config_section"))
	b.WriteString(`
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
    ` + `"""` + i18n.T("eject.linter.yaml_docstring") + `"""` + `
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
`)
	b.WriteString(i18n.T("eject.linter.empty_key_comment"))
	b.WriteString(`                out[key] = ""
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


`)
	b.WriteString(i18n.T("eject.linter.parse_section"))
	b.WriteString(`
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
    return [p for i, p in enumerate(parts) if i % 2 == 0]


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


`)
	b.WriteString(i18n.T("eject.linter.walk_section"))
	b.WriteString(`
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


`)
	b.WriteString(i18n.T("eject.linter.judge_section"))
	b.WriteString(`
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
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.unclosed.message", "",
		"eject.linter.unclosed.fix", "", ")\n"))
	b.WriteString(`                continue
            fm_text = "\n".join(lines[1:close])
            doc["fields"] = parse_simple_yaml(fm_text)
            doc["fm_lines"] = lines[1:close]
            body = "\n".join(lines[close + 1:])
            body_line = close + 2
            doc["links"] = body_links(body, body_line)
        else:
            add(rel, 1, "frontmatter.missing", "error",
`)
	b.WriteString(pyViolation("                ",
		"eject.linter.missing.message", "",
		"eject.linter.missing.fix", "", ")\n"))
	b.WriteString(`            continue

        fields = doc["fields"]
        stage = fields.get("artifact_stage")
`)
	b.WriteString(i18n.T("eject.linter.int_stage_comment"))
	b.WriteString(`        if isinstance(stage, int) and not isinstance(stage, bool):
            stage = str(stage)
        if isinstance(stage, str):
            doc["stage"] = stage

        if "artifact_stage" not in fields:
`)
	b.WriteString(i18n.T("eject.linter.stage_input_comment"))
	b.WriteString(`            if "artifact_stage" in cfg["axes"]:
                add(rel, line_of_key(text, "artifact_stage"), "frontmatter.missing-field", "error",
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.stage_missing.message", "",
		"eject.linter.stage_missing.fix", "", ")\n"))
	b.WriteString(`        elif stage:
            for f in required_fields(stage, cfg["axes"]):
                if f not in fields:
                    add(rel, line_of_key(text, "artifact_stage"), "frontmatter.missing-field", "error",
`)
	b.WriteString(pyViolation("                        ",
		"eject.linter.required_missing.message", " % (stage, f)",
		"eject.linter.required_missing.fix", " % f", ")\n"))
	b.WriteString(`
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
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.allowed_value.message", ` % (key, val, ", ".join(values))`,
		"eject.linter.allowed_value.fix", " % key", ")\n"))
	b.WriteString(`
        if stage and not doc["root"]:
            top = rel.split("/")[0]
            expected = stage_for_dir(top)
            if expected and stage != expected:
                severity = "error" if stage == "context" else "warn"
                add(rel, line_of_key(text, "artifact_stage"), "location.stage-agreement", severity,
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.stage_agreement.message", " % (top, stage)",
		"eject.linter.stage_agreement.fix", " % expected", ")\n"))
	b.WriteString(`
        for axis in AXIS_NAMES:
            if axis not in cfg["axes"] and axis in fields:
                add(rel, line_of_key(text, axis), "schema.axis-off", "error",
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.axis_off.message", " % (axis, cfg[\"preset\"])",
		"eject.linter.axis_off.fix", " % (axis, axis)", ")\n"))
	b.WriteString(`
        form = fields.get("form")
        if isinstance(form, str) and form != "":
            forms = cfg["forms"]
            if forms and form not in forms:
                add(rel, line_of_key(text, "form"), "taxonomy.forms", "error",
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.forms.message", ` % (form, ", ".join(forms))`,
		"eject.linter.forms.fix", "", ")\n"))
	b.WriteString(`        topics = fields.get("topics")
        if isinstance(topics, list):
            for v in topics:
                if v not in cfg["topics"]:
                    add(rel, line_of_key(text, "topics"), "taxonomy.topics", "warn",
`)
	b.WriteString(pyViolation("                        ",
		"eject.linter.topics.message", " % v",
		"eject.linter.topics.fix", " % v", ")\n"))
	b.WriteString(`
        sources_dir = STAGE_DIRS.get("source", "sources")
        if "updated" in fields and (rel == sources_dir or rel.startswith(sources_dir + "/")):
            add(rel, line_of_key(text, "updated"), "sources.updated", "warn",
`)
	b.WriteString(pyViolation("                ",
		"eject.linter.sources_updated.message", "",
		"eject.linter.sources_updated.fix", "", ")\n"))
	b.WriteString(`
        if line_count(text) > cfg["max_lines"]:
            add(rel, line_of_key(text, "artifact_stage"), "body.max-lines", "warn",
`)
	b.WriteString(pyViolation("                ",
		"eject.linter.max_lines.message", " % (line_count(text), cfg[\"max_lines\"])",
		"eject.linter.max_lines.fix", "", ")\n"))
	b.WriteString("\n")
	b.WriteString(i18n.T("eject.linter.related_comment"))
	b.WriteString(`        related = fields.get("related")
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
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.link_broken.message", " % slug",
		"eject.linter.link_broken.fix", " % slug", "))\n"))
	b.WriteString(`        for v in doc["relation_values"]:
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
`)
	b.WriteString(pyViolation("                ",
		"eject.linter.orphan.message", "",
		"eject.linter.orphan.fix", " % slug", "))\n"))
	b.WriteString(i18n.T("eject.linter.gate_dir_comment"))
	b.WriteString(`        if context_dir and doc["rel"].startswith(context_dir + "/") and min_wikilinks > 0:
            n = len(outgoing)
            targets = sum(1 for d in docs if d["rel"] != doc["rel"] and linkable(d))
            passed, deferred = evaluate_gate(n, targets, min_wikilinks)
`)
	b.WriteString(i18n.T("eject.linter.gate_line_comment"))
	b.WriteString(`            gate_line = line_of_key(doc["text"], "related")
            if deferred and n < min_wikilinks:
                violations.append(Violation(
                    doc["rel"], gate_line, "gate.deferred", "warn",
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.gate_deferred.message", " % (targets, min_wikilinks, min_wikilinks)",
		"eject.linter.gate_deferred.fix", "", "))\n"))
	b.WriteString(`                continue
            if not passed:
                violations.append(Violation(
                    doc["rel"], gate_line, "gate.min-wikilinks", "reject",
`)
	b.WriteString(pyViolation("                    ",
		"eject.linter.gate_reject.message", " % (n, min_wikilinks)",
		"eject.linter.gate_reject.fix", " % (min_wikilinks - n)", "))\n"))
	b.WriteString(`    return violations


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
        print(` + `"  [%s] %d %s"` + ` % (v.severity, v.line, v.rule))
        print(` + `"    %s"` + ` % v.message)
        print(` + pyString(i18n.T("eject.linter.print_fix")) + ` % v.fix)
        counts[v.severity] += 1
    files = len(walk_docs(root, cfg))
    print(` + pyString(i18n.T("eject.linter.summary")) + ` % (files, counts["error"], counts["warn"], counts["reject"]))
`)
	b.WriteString(i18n.T("eject.linter.exit_code_comment"))
	b.WriteString(`    if counts["error"] > 0 or counts["reject"] > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
`)
	return b.String()
}

// pyViolation은 Python 판정 코드의 message, fix 두 줄을 만든다.
// indent 는 줄 앞 들여쓰기, msgTail 과 fixTail 은 Python 이 % 포맷을
// 채우는 표현식으로 리터럴 뒤에 붙고, close 는 add 호출 또는
// Violation(...) 생성을 닫는 괄호다. 문장은 인자 없는 i18n.T 로
// 꺼낸다. %s 가 Go 서식으로 다루어지면 안 되기 때문이다.
func pyViolation(indent, msgID, msgTail, fixID, fixTail, close string) string {
	return indent + pyString(i18n.T(msgID)) + msgTail + ",\n" +
		indent + pyString(i18n.T(fixID)) + fixTail + close
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
