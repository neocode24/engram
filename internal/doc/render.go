package doc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// StandardKeys는 새 문서의 프론트매터 키 표준 순서다. 파싱이 키 순서를
// 보존하므로 직렬화도 순서를 그대로 쓰며, 기준 순서가 없는 새 문서는
// 이 순서를 따른다. ADR 0005가 프론트매터 정규화를 parity 비교 축으로
// 삼으므로 직렬화 순서는 언제나 결정론적이어야 한다.
func StandardKeys() []string {
	return []string{
		"type", "artifact_stage", "status", "scope", "sensitivity",
		"source_channel", "trigger_mode", "workflow", "form", "topics",
		"tags", "source_refs", "derived_from", "derived_context", "related",
		"indexable", "created", "sourced_at", "updated",
	}
}

// Render는 파싱의 역연산으로 필드와 본문을 문서 바이트로 만든다.
// 필드 순서는 입력 순서를 그대로 쓴다. 필드가 없으면 프론트매터 구분자를
// 내지 않는다. 줄바꿈은 LF이고 본문 끝에는 개행 하나를 보장한다.
func Render(fields []Field, body string) []byte {
	var b strings.Builder
	if len(fields) > 0 {
		b.WriteString("---\n")
		for _, f := range fields {
			renderField(&b, f)
		}
		b.WriteString("---\n")
	}
	if body != "" {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}

// RenderMap은 프론트매터 맵과 본문으로 문서 바이트를 만든다.
// 키 순서는 StandardKeys의 표준 순서이고 표준에 없는 키는 이름순으로 뒤에 붙는다.
// 값은 문자열, 불리언, []string, nil(빈 값 키)을 받는다.
func RenderMap(fm map[string]any, body string) []byte {
	standard := StandardKeys()
	rank := make(map[string]int, len(standard))
	for i, k := range standard {
		rank[k] = i
	}
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ri, oki := rank[keys[i]]
		rj, okj := rank[keys[j]]
		switch {
		case oki && okj:
			return ri < rj
		case oki:
			return true
		case okj:
			return false
		}
		return keys[i] < keys[j]
	})
	fields := make([]Field, 0, len(keys))
	for _, k := range keys {
		fields = append(fields, toField(k, widen(fm[k])))
	}
	return Render(fields, body)
}

// widen은 맵 값을 toField가 이해하는 형태로 바꾼다. []string 은
// YAML 파서가 만드는 []interface{} 와 같은 취급을 받아야 한다.
func widen(v any) any {
	if list, ok := v.([]string); ok {
		out := make([]interface{}, len(list))
		for i, s := range list {
			out[i] = s
		}
		return out
	}
	return v
}

// renderField는 필드 하나를 YAML 줄로 낸다. 배열은 블록 스타일(하이픈 목록)로
// 내고 빈 배열은 [] 로 낸다. 사람이 손으로 고치는 파일이기 때문이다.
func renderField(b *strings.Builder, f Field) {
	switch f.Kind {
	case KindEmpty:
		fmt.Fprintf(b, "%s:\n", f.Key)
	case KindBool:
		fmt.Fprintf(b, "%s: %t\n", f.Key, f.Bool)
	case KindStringList:
		if len(f.List) == 0 {
			fmt.Fprintf(b, "%s: []\n", f.Key)
			return
		}
		fmt.Fprintf(b, "%s:\n", f.Key)
		for _, item := range f.List {
			fmt.Fprintf(b, "  - %s\n", renderScalar(item))
		}
	default:
		fmt.Fprintf(b, "%s: %s\n", f.Key, renderScalar(f.Str))
	}
}

// renderScalar는 문자열 하나를 YAML 스칼라로 낸다. 인용이 필요한 값은
// 큰따옴표로 감싼다. "[[슬러그]]" 처럼 인용 없이는 목록으로 읽히는 값이
// 왕복을 하려면 인용이 필수다.
func renderScalar(s string) string {
	if !needsQuote(s) {
		return s
	}
	return strconv.Quote(s)
}

// needsQuote는 값이 인용 없이는 다른 YAML 자료형으로 읽히는지 검사한다.
// 콜론은 뒤에 공백이 오거나 값 끝에 붙었을 때만 문제가 된다. URL 같은
// 일반 값에 인용을 붙이지 않기 위해서다.
func needsQuote(s string) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	if strings.ContainsAny(s, ",[]{}#&*!|>'\"%@`") {
		return true
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") || strings.Contains(s, " #") {
		return true
	}
	if strings.Contains(s, "\n") || strings.Contains(s, "\t") {
		return true
	}
	switch s {
	case "true", "false", "null", "~", "yes", "no", "on", "off":
		return true
	}
	return false
}
