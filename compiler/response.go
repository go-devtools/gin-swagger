package compiler

import (
	"go/constant"
	"go/types"
	"mime"
	"net/textproto"
	"sort"
	"strconv"

	core "github.com/go-devtools/openapi/compiler"
	"github.com/go-devtools/openapi/spec"
)

// Construct response effects from Gin's actual renderer while leaving JSON type projection to the core.
func renderResponse(c core.CallContext, status, media string, payload core.Value, schema *spec.Schema) []core.Effect {
	status = normalizedGinStatus(status)
	if interimStatus(status) {
		return unresolved(c, "Interim responses and final status require a complete Writer commit sequence rule")
	}
	if _, _, err := mime.ParseMediaType(media); err != nil {
		return unresolved(c, "response media type is not a valid explicit value")
	}
	source := c.Source
	source.Kind = "derived"
	source.Rule = "gin." + c.Object.Name()
	return []core.Effect{{Kind: core.ResponseBody, Status: status, MediaType: media, Payload: payload, WireSchema: schema, Source: source}}
}

// Describe raw HTTP bytes with a content media type instead of JSON string or Base64 constraints.
func rawResponse(c core.CallContext, status, media string) []core.Effect {
	return renderResponse(c, status, media, core.Value{}, &spec.Schema{SchemaObject: &spec.SchemaObject{ContentMediaType: media}})
}

// Apply reader headers only when the response header is empty; a known length overrides its extra-header entry.
func readerResponse(c core.CallContext, status, media string, length, reader, headers core.Value) []core.Effect {
	status = normalizedGinStatus(status)
	if status == "-1" {
		return unresolved(c, "A reader that preserves the existing status requires status-dependent header rules")
	}
	if status == "204" || status == "304" {
		// Gin only calls WriteContentType here, skipping all extra headers from Reader.Render.
		return rawResponse(c, status, media)
	}
	if reader.Nil {
		return unresolved(c, "reader is nil and cannot produce a normal response")
	}
	if !headers.Nil && headers.Fields == nil {
		return unresolved(c, "reader extra headers require an analyzable map literal or centralized rule")
	}
	knownLength := length.Constant != nil && length.Constant.Kind() == constant.Int
	withLength := knownLength && constant.Sign(length.Constant) >= 0
	if !knownLength {
		return unresolved(c, "reader length controls response headers and must be evaluable or have a centralized rule")
	}
	source := c.Source
	source.Kind, source.Rule = "derived", "gin."+c.Object.Name()
	var effects []core.Effect
	keys := make([]string, 0, len(headers.Fields))
	for key := range headers.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seen := map[string]bool{}
	for _, name := range keys {
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		if seen[canonical] {
			return unresolved(c, "extra headers have case-insensitive duplicate names; write order is ambiguous")
		}
		seen[canonical] = true
		if withLength && canonical == "Content-Length" {
			if name != "Content-Length" {
				return unresolved(c, "length header conflicts with an automatically generated field's case")
			}
			continue
		}
		if canonical == "Content-Type" {
			// The renderer writes its explicit media type before extra headers can override it.
			continue
		}
		effects = append(effects, core.Effect{Kind: core.ResponseHeader, Name: name, Payload: headers.Fields[name], HeaderIfEmpty: true, Source: source})
	}
	if withLength {
		effects = append(effects, core.Effect{Kind: core.ResponseHeader, Name: "Content-Length", Payload: core.Value{Type: types.Typ[types.String], Constant: constant.MakeString(length.Constant.ExactString())}, HeaderIfEmpty: true, Source: source})
	}
	return append(effects, rawResponse(c, status, media)...)
}

// Match renderers by full package and type identity rather than custom implementations with matching short names.
func explicitRenderer(c core.CallContext, status string, renderer core.Value) []core.Effect {
	status = normalizedGinStatus(status)
	typ := renderer.Type
	if isSSEEvent(typ) {
		return explicitSSE(c, status, renderer)
	}
	if typ == nil {
		return unresolved(c, "Renderer type is unresolved")
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != ginPackage+"/render" {
		return unresolved(c, "custom Renderer requires a centralized codec rule")
	}
	if renderer.Fields == nil {
		return unresolved(c, "Renderer field value cannot be determined statically")
	}
	field := func(name string) core.Value {
		if value, ok := renderer.Fields[name]; ok {
			return value
		}
		return core.Value{Type: types.Typ[types.UntypedNil], Nil: true}
	}
	switch named.Obj().Name() {
	case "JSON", "IndentedJSON", "AsciiJSON", "PureJSON":
		return renderResponse(c, status, "application/json", field("Data"), nil)
	case "String":
		return renderResponse(c, status, "text/plain", core.Value{}, spec.Typed("string"))
	case "Data":
		return rawResponse(c, status, literal(field("ContentType")))
	case "Reader":
		length := field("ContentLength")
		if length.Nil {
			length = core.Value{Type: types.Typ[types.Int64], Constant: constant.MakeInt64(0)}
		}
		return readerResponse(c, status, literal(field("ContentType")), length, field("Reader"), field("Headers"))
	default:
		return unresolved(c, "Unrecognized Renderer encoding: "+named.Obj().Name())
	}
}

// Interim statuses in Gin's wrapped writer cannot be inferred as ordinary final statuses.
func interimStatus(status string) bool {
	return len(status) == 3 && status[0] == '1'
}

// Gin ignores non-positive status codes; use the neutral preservation marker for the current pending status.
func normalizedGinStatus(status string) string {
	if code, err := strconv.Atoi(status); err == nil && code <= 0 {
		return "-1"
	}
	return status
}
