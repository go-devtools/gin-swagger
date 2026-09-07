// Convert raw Gin paths into neutral OpenAPI paths.
package routes

import (
	"fmt"
	"net/url"
	"strings"
)

// Preserve template names and describe lossy catch-all or conditional raw-path semantics.
type Path struct {
	Path               string
	Parameters         []string
	CatchAll           bool
	RawPathConditional bool
}

// Select the public Gin request-path configuration; escaped paths take precedence.
type Profile struct {
	UseRawPath     bool
	UseEscapedPath bool
}

// Parse decoded Gin path syntax while preserving escaped literal colons.
func Parse(raw string) (Path, error) {
	return ParseWithProfile(raw, Profile{})
}

// Encode static route bytes according to the path string selected by Gin 1.12.
func ParseWithProfile(raw string, profile Profile) (Path, error) {
	out := Path{}
	if len(raw) == 0 || raw[0] != '/' || len(raw) > 16384 {
		return out, fmt.Errorf("gin-swagger.path.invalid: path is empty, not absolute, or exceeds the budget")
	}
	var decoded, escaped, sample strings.Builder
	var literal strings.Builder
	validEscaped, changesLiteral := true, false
	// Flush only literal bytes: parameter names belong to the template, not the URL.
	flush := func() {
		value := literal.String()
		wire := (&url.URL{Path: value}).EscapedPath()
		decoded.WriteString(wire)
		escaped.WriteString(value)
		sample.WriteString(value)
		validEscaped = validEscaped && validLiteral(value)
		changesLiteral = changesLiteral || value != wire
		literal.Reset()
	}
	names := map[string]bool{}
	for i := 0; i < len(raw); {
		c := raw[i]
		if c == '\\' {
			if i+1 >= len(raw) || raw[i+1] != ':' {
				return out, fmt.Errorf("gin-swagger.path.escape: only colons may be escaped")
			}
			literal.WriteByte(':')
			i += 2
			continue
		}
		if c == ':' || c == '*' {
			flush()
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
			decoded.WriteString("{" + name + "}")
			escaped.WriteString("{" + name + "}")
			sample.WriteString("sample")
			continue
		}
		literal.WriteByte(c)
		i++
	}
	flush()
	out.Path = decoded.String()
	if profile.UseEscapedPath {
		if !validEscaped {
			return Path{}, fmt.Errorf("gin-swagger.path.escaped: UseEscapedPath requires static route literals to be RFC 3986 pchar bytes or valid percent escapes")
		}
		out.Path = escaped.String()
	} else if profile.UseRawPath {
		// A noncanonical static escape keeps RawPath populated independently of parameter values.
		wire, err := url.ParseRequestURI(sample.String())
		if validEscaped && err == nil && wire.RawPath != "" {
			out.Path = escaped.String()
		} else {
			out.RawPathConditional = changesLiteral && len(out.Parameters) > 0
		}
	}
	return out, nil
}

// Validate the literal subset that OpenAPI can preserve in an escaped Gin routing tree.
func validLiteral(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '%' {
			if i+2 >= len(value) || !hexByte(value[i+1]) || !hexByte(value[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("/-._~!$&'()*+,;=:@", rune(c)) {
			continue
		}
		return false
	}
	return true
}

// Recognize one hexadecimal digit without normalizing the registered escape's case.
func hexByte(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'F' || value >= 'a' && value <= 'f'
}
