// 将 Gin 原始路径转换为框架中立 OpenAPI 路径。
package routes

import (
	"fmt"
	"strings"
)

// 保留 catch-all 不可无损表达的跨斜杠语义。
type Path struct {
	Path       string
	Parameters []string
	CatchAll   bool
}

// 按 Gin 一点十二路径词法扫描，转义冒号仍是静态字符。
func Parse(raw string) (Path, error) {
	out := Path{}
	if len(raw) == 0 || raw[0] != '/' || len(raw) > 16384 {
		return out, fmt.Errorf("gin-swagger.path.invalid: 路径为空、非绝对或超过预算")
	}
	var b strings.Builder
	names := map[string]bool{}
	for i := 0; i < len(raw); {
		c := raw[i]
		if c == '\\' {
			if i+1 >= len(raw) || raw[i+1] != ':' {
				return out, fmt.Errorf("gin-swagger.path.escape: 仅允许转义冒号")
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
					return out, fmt.Errorf("gin-swagger.path.parameter: 同一段存在多个通配符或非法名称")
				}
				i++
			}
			name := raw[start+1 : i]
			if name == "" || names[name] {
				return out, fmt.Errorf("gin-swagger.path.parameter: 参数为空或重复")
			}
			names[name] = true
			out.Parameters = append(out.Parameters, name)
			if c == '*' {
				if start == 0 || raw[start-1] != '/' || i != len(raw) {
					return out, fmt.Errorf("gin-swagger.path.catchall: catch-all 必须单独位于最后一段")
				}
				out.CatchAll = true
			}
			b.WriteByte('{')
			b.WriteString(name)
			b.WriteByte('}')
			continue
		}
		if c == '{' || c == '}' || c == '?' || c == '#' || c < 32 {
			return out, fmt.Errorf("gin-swagger.path.static: 静态字符无法直接表示为 OpenAPI 路径")
		}
		b.WriteByte(c)
		i++
	}
	out.Path = b.String()
	return out, nil
}
