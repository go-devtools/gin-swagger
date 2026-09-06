package compiler

import (
	"go/constant"
	"go/types"
	"net/textproto"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Model standard redirect bodies and pending status using the actual method and existing response headers.
func redirectOutcomes(c core.CallContext) ([]core.CallOutcome, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != ginPackage || c.Object.Name() != "Redirect" {
		return nil, nil
	}
	signature, ok := c.Object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || !isContext(signature.Recv().Type()) {
		return nil, nil
	}
	if len(c.Arguments) < 2 {
		return []core.CallOutcome{{Effects: unresolved(c, "redirect arguments are unresolved")}}, nil
	}
	status := integer(c.Arguments[0])
	code, err := strconv.Atoi(status)
	if err != nil || (code != 201 && (code < 300 || code > 308)) {
		return []core.CallOutcome{{Effects: unresolved(c, "redirect status must be an explicit 201 or 300 through 308 accepted by Gin")}}, nil
	}
	source := c.Source
	source.Kind, source.Rule = "derived", "gin.Redirect"
	value := core.Value{Type: types.Typ[types.String]}
	if c.Arguments[1].Constant != nil && c.Arguments[1].Constant.Kind() == constant.String {
		if location, known := redirectLocation(literal(c.Arguments[1])); known {
			value.Constant = constant.MakeString(location)
		}
	}
	location := core.Effect{Kind: core.ResponseHeader, Name: "Location", Payload: value, Source: source}
	pending := core.Effect{Kind: core.ResponseStatus, Status: status, Source: source}
	if _, exists := c.Response.Headers["Content-Type"]; exists {
		return []core.CallOutcome{{Effects: []core.Effect{location, pending}}}, nil
	}
	contentType := core.Effect{Kind: core.ResponseHeader, Name: "Content-Type", Payload: core.Value{Type: types.Typ[types.String], Constant: constant.MakeString("text/html; charset=utf-8")}, Source: source}
	body := renderResponse(c, status, "text/html", core.Value{}, spec.Typed("string"))
	return []core.CallOutcome{
		{When: openapi.RequestCondition{Methods: []string{"GET"}}, Effects: append([]core.Effect{location, contentType}, body...)},
		{When: openapi.RequestCondition{Methods: []string{"HEAD"}}, Effects: []core.Effect{location, contentType, pending}},
		{When: openapi.RequestCondition{ExceptMethods: []string{"GET", "HEAD"}}, Effects: []core.Effect{location, pending}},
	}, nil
}

// Normalize only targets independent of the request path; relative targets retain their known string wire type.
func redirectLocation(target string) (string, bool) {
	parsed, err := url.Parse(target)
	if err == nil && parsed.Scheme == "" && parsed.Host == "" {
		if !strings.HasPrefix(target, "/") {
			return "", false
		}
		suffix := ""
		if i := strings.IndexByte(target, '?'); i >= 0 {
			target, suffix = target[:i], target[i:]
		}
		trailing := strings.HasSuffix(target, "/")
		target = path.Clean(target)
		if trailing && !strings.HasSuffix(target, "/") {
			target += "/"
		}
		target += suffix
	}
	const hex = "0123456789abcdef"
	var result strings.Builder
	for i := 0; i < len(target); i++ {
		b := target[i]
		if b >= 128 {
			result.WriteByte('%')
			result.WriteByte(hex[b>>4])
			result.WriteByte(hex[b&15])
		} else if b == '\r' || b == '\n' {
			result.WriteByte(' ')
		} else {
			result.WriteByte(b)
		}
	}
	return textproto.TrimString(result.String()), true
}
