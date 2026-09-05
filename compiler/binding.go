package compiler

import (
	"go/types"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// 为显式绑定器生成请求事实，绑定失败的业务分支仍由核心控制流分析。
// Emit facts for explicit binders while the core analyzes business branches after binding failures.
func bindRequest(c core.CallContext, mode string, payload core.Value) []core.Effect {
	if payload.Type == nil {
		return unresolved(c, "绑定目标类型未解决")
	}
	typ := types.Unalias(payload.Type)
	pointer, ok := typ.(*types.Pointer)
	if !ok || payload.Nil {
		return unresolved(c, "绑定目标必须是明确的非 nil 指针")
	}
	payload.Type = pointer.Elem()
	payload.Fields = nil
	source := c.Source
	source.Kind, source.Rule = "derived", "gin."+c.Object.Name()
	effect := core.Effect{Kind: core.RequestBody, Payload: payload, Source: source}
	switch mode {
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
		return unresolved(c, "此绑定器需要明确的方法、媒体类型或编解码规则："+mode)
	}
	return []core.Effect{effect}
}

// 使用已传播的完整变量身份识别 Gin 导出的绑定器，不按短名称匹配。
// Identify exported Gin binders by propagated full variable identity instead of short names.
func explicitBinder(c core.CallContext, value core.Value, payload core.Value, bodyOnly bool) []core.Effect {
	object, ok := value.Object.(*types.Var)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != ginPackage+"/binding" || object.Parent() != object.Pkg().Scope() {
		return unresolved(c, "显式绑定器身份无法确定")
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
		return unresolved(c, "显式绑定器尚需集中规则："+object.Name())
	}
	if bodyOnly && mode != "json" {
		return unresolved(c, "缓存请求体绑定器尚需专用编解码规则")
	}
	return bindRequest(c, mode, payload)
}
