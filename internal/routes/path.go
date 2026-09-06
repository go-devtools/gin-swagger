// Convert raw Gin paths into neutral OpenAPI paths.
// 将 Gin 原始路径转换为框架中立 OpenAPI 路径。
package routes

import (
	"fmt"
	"strings"
)

// Record the lossy cross-slash semantics of catch-all parameters.
// 保留 catch-all 不可无损表达的跨斜杠语义。
type Path struct {
	Path       string
	Parameters []string
	CatchAll   bool
}

// Parse Gin 1.12 path syntax while preserving escaped literal colons.
// 按 Gin 一点十二路径词法扫描，转义冒号仍是静态字符。
func Parse(raw string) (Path, error) {
	out := Path{}
	if len(raw) == 0 || raw[0] != '/' || len(raw) > 16384 {
		return out, fmt.Errorf("gin-swagger.path.invalid: path is empty, not absolute, or exceeds the budget")
	}
	var b strings.Builder
	names := map[string]bool{}
	for i := 0; i < len(raw); {
		c := raw[i]
		if c == '\\' {
			if i+1 >= len(raw) || raw[i+1] != ':' {
				return out, fmt.Errorf("gin-swagger.path.escape: only colons may be escaped")
			}
			b.WriteByte(':')
			i += 2
			continue
		}
		if c == ':' || c == '*' {
			start := i
			i++
			for i < len(raw) && raw[i] != '/' {
				if raw[i] == ':' || raw[i] == '*' || raw[i] == '{' || raw[i] == '}' {
					return out, fmt.Errorf("gin-swagger.path.parameter: segment contains multiple wildcards or an invalid name")
				}
				i++
			}
			name := raw[start+1 : i]
			if name == "" || names[name] {
				return out, fmt.Errorf("gin-swagger.path.parameter: parameter is empty or duplicated")
			}
			names[name] = true
			out.Parameters = append(out.Parameters, name)
			if c == '*' {
				if start == 0 || raw[start-1] != '/' || i != len(raw) {
					return out, fmt.Errorf("gin-swagger.path.catchall: catch-all must occupy the final segment alone")
				}
				out.CatchAll = true
			}
			b.WriteByte('{')
			b.WriteString(name)
			b.WriteByte('}')
			continue
		}
		if c == '{' || c == '}' || c == '?' || c == '#' || c < 32 {
			return out, fmt.Errorf("gin-swagger.path.static: static character cannot be represented directly in an OpenAPI path")
		}
		b.WriteByte(c)
		i++
	}
	out.Path = b.String()
	return out, nil
}
