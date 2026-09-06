package compiler

import (
	"go/types"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Emit facts for explicit binders while the core analyzes business branches after binding failures.
// 为显式绑定器生成请求事实，绑定失败的业务分支仍由核心控制流分析。
func bindRequest(c core.CallContext, mode string, payload core.Value) []core.Effect {
	if payload.Type == nil {
		return unresolved(c, "binding target type is unresolved")
	}
	typ := types.Unalias(payload.Type)
	pointer, ok := typ.(*types.Pointer)
	if !ok || payload.Nil {
		return unresolved(c, "binding target must be an explicit non-nil pointer")
	}
	payload.Type = pointer.Elem()
	payload.Fields = nil
	source := c.Source
	source.Kind, source.Rule = "derived", "gin."+c.Object.Name()
	effect := core.Effect{Kind: core.RequestBody, Payload: payload, Source: source}
	switch mode {
	case "form-query", "form-urlencoded", "form-multipart":
		effect.Kind = core.ParameterObject
		effect.In = "query"
		effect.MediaType = "application/x-www-form-urlencoded"
		effect.Codec = BindingCodec{Mode: "form"}
		effect.Style = "form"
		effect.Explode = spec.Set(true)
		effect.Source.Rule += "." + mode
		if mode == "form-query" {
			return []core.Effect{effect}
		}
		effect.AlternativeLocations = true
		body := effect
		body.Kind = core.RequestBody
		body.In = ""
		body.Style = ""
		body.Explode = spec.Optional[bool]{}
		if mode == "form-multipart" {
			body.MediaType = "multipart/form-data"
			body.Source.Rule += ".query-before-body"
		} else {
			body.Source.Rule += ".body-before-query"
		}
		return []core.Effect{effect, body}

	case "json":
		effect.MediaType = "application/json"
	case "query", "uri", "header":
		effect.Kind = core.ParameterObject
		effect.In = mode
		if mode == "uri" {
			effect.In = "path"
		}
		effect.MediaType = "application/x-www-form-urlencoded"
		effect.Codec = BindingCodec{Mode: mode}
		effect.Style = "simple"
		effect.Explode = spec.Set(false)
		if mode == "query" {
			effect.Style = "form"
			effect.Explode = spec.Set(true)
		}
	case "form-post":
		effect.MediaType = "application/x-www-form-urlencoded"
		effect.Codec = BindingCodec{Mode: mode}
	case "multipart":
		effect.MediaType = "multipart/form-data"
		effect.Codec = BindingCodec{Mode: mode}
	default:
		return unresolved(c, "This binder requires an explicit method, media type, or codec rule: "+mode)
	}
	return []core.Effect{effect}
}

// Identify exported Gin binders by propagated full variable identity instead of short names.
// 使用已传播的完整变量身份识别 Gin 导出的绑定器，不按短名称匹配。
func explicitBinder(c core.CallContext, value core.Value, payload core.Value, bodyOnly bool) []core.Effect {
	object, ok := value.Object.(*types.Var)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != ginPackage+"/binding" || object.Parent() != object.Pkg().Scope() {
		return unresolved(c, "explicit binder identity cannot be determined")
	}
	mode := ""
	switch object.Name() {
	case "JSON":
		mode = "json"
	case "Query":
		mode = "query"
	case "Header":
		mode = "header"
	case "FormPost":
		mode = "form-post"
	case "FormMultipart":
		mode = "multipart"
	default:
		return unresolved(c, "Explicit binder requires a centralized rule: "+object.Name())
	}
	if bodyOnly && mode != "json" {
		return unresolved(c, "cached request-body binder requires a dedicated codec rule")
	}
	return bindRequest(c, mode, payload)
}
