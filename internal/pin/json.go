package pin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// obj는 원장 JSON 객체다. 원장 형식이 upstream 과 같은 파일이라 구조체로
// 좁히지 않고 JSON 값 그대로 다룬다. 칸이 늘어도 읽어서 되쓰는 쪽이
// 모르는 칸을 잃지 않게 하려는 선택이다.
type obj = map[string]any

// decodeJSON은 숫자를 json.Number로 읽는다. 원장에 적힌 숫자 표기를
// 되쓸 때 그대로 남기기 위해서다.
func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("JSON 뒤에 글자가 더 있습니다")
	}
	return v, nil
}

// dump는 JSON 값을 upstream(파이썬 json.dumps, sort_keys, ensure_ascii=False)과
// 같은 글자로 쓴다. indent가 참이면 들여쓰기 1칸이고 끝에 줄바꿈이 붙는다.
// 거짓이면 한 줄이고 구분자는 ", "와 ": "다. 원장 파일을 바이트 단위로
// 견줄 수 있게 직접 쓴다.
func dump(v any, indent bool) string {
	var b strings.Builder
	writeValue(&b, v, indent, 0)
	if indent {
		b.WriteByte('\n')
	}
	return b.String()
}

func writeValue(b *strings.Builder, v any, indent bool, depth int) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		writeString(b, x)
	case int:
		b.WriteString(strconv.Itoa(x))
	case int64:
		b.WriteString(strconv.FormatInt(x, 10))
	case float64:
		b.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
	case json.Number:
		b.WriteString(x.String())
	case []string:
		list := make([]any, len(x))
		for i, s := range x {
			list[i] = s
		}
		writeValue(b, list, indent, depth)
	case []any:
		if len(x) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
				if !indent {
					b.WriteByte(' ')
				}
			}
			if indent {
				b.WriteByte('\n')
				b.WriteString(strings.Repeat(" ", depth+1))
			}
			writeValue(b, e, indent, depth+1)
		}
		if indent {
			b.WriteByte('\n')
			b.WriteString(strings.Repeat(" ", depth))
		}
		b.WriteByte(']')
	case map[string]any:
		if len(x) == 0 {
			b.WriteString("{}")
			return
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
				if !indent {
					b.WriteByte(' ')
				}
			}
			if indent {
				b.WriteByte('\n')
				b.WriteString(strings.Repeat(" ", depth+1))
			}
			writeString(b, k)
			b.WriteString(": ")
			writeValue(b, x[k], indent, depth+1)
		}
		if indent {
			b.WriteByte('\n')
			b.WriteString(strings.Repeat(" ", depth))
		}
		b.WriteByte('}')
	default:
		panic(fmt.Sprintf("pin: JSON으로 쓸 수 없는 값: %T", v))
	}
}

// writeString은 문자열을 ensure_ascii=False 규칙으로 쓴다. 따옴표, 역슬래시,
// 제어 문자만 이스케이프하고 나머지는 그대로 둔다.
func writeString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// 아래는 JSON 값을 느슨하게 읽는 도우미다. 맞지 않는 형이면 영값을 낸다.
// 형이 틀린 요청은 검사 단계에서 거절되므로 여기서 패닉을 내지 않는다.

func str(v any) string {
	s, _ := v.(string)
	return s
}

func isStr(v any) bool {
	_, ok := v.(string)
	return ok
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case float64:
		if x == float64(int(x)) {
			return int(x), true
		}
	}
	return 0, false
}

func isInt(v any) bool {
	_, ok := toInt(v)
	return ok
}

func asObj(v any) obj {
	o, _ := v.(map[string]any)
	return o
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func strList(v any) []string {
	var out []string
	for _, e := range asList(v) {
		out = append(out, str(e))
	}
	return out
}

func toAnyList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// clone은 JSON 값을 깊은 복사한다. 요청 하나를 되돌릴 때 쓴다.
func clone(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, e := range x {
			m[k] = clone(e)
		}
		return m
	case []any:
		l := make([]any, len(x))
		for i, e := range x {
			l[i] = clone(e)
		}
		return l
	}
	return v
}

func cloneObj(o obj) obj {
	if o == nil {
		return nil
	}
	return clone(o).(obj)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

func setOf(list ...string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, e := range list {
		m[e] = true
	}
	return m
}

// jsonEqual은 두 JSON 값이 같은지 본다. 숫자 표기가 json.Number와 int로
// 갈려도 값으로 견준다.
func jsonEqual(a, b any) bool {
	return dump(normalizeNumbers(a), false) == dump(normalizeNumbers(b), false)
}

func normalizeNumbers(v any) any {
	switch x := v.(type) {
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return int(n)
		}
		f, _ := x.Float64()
		return f
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, e := range x {
			m[k] = normalizeNumbers(e)
		}
		return m
	case []any:
		l := make([]any, len(x))
		for i, e := range x {
			l[i] = normalizeNumbers(e)
		}
		return l
	}
	return v
}
