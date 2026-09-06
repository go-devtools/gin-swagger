// Expose Gin generation rules through the public compiler SDK.
// 通过核心公开 SDK 提供 Gin 生成前端；普通业务程序不导入本包。
package compiler

import (
	"fmt"
	"go/constant"
	"go/types"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Match the complete Gin package identity rather than short method names.
// 明确限定 Gin 的真实包身份，禁止仅按方法短名匹配。
const ginPackage = "github.com/gin-gonic/gin"

// Provide the same frontend to the CLI and custom generation entry points.
// 注册静态 Gin 规则，同一个值供 CLI 与项目自定义生成器组合。
func Frontend() core.Frontend {
	return core.Frontend{Name: "gin-v1.12-front-v9", Match: func(f core.Function) bool {
		return f.Signature.Params().Len() == 1 && isContext(f.Signature.Params().At(0).Type())
	}, Entry: func(f core.Function) []core.Effect {
		source := f.Source
		source.Kind, source.Rule = "derived", "gin.default.status"
		return []core.Effect{{Kind: core.ResponseStatus, Status: "200", Source: source}}
	}, Call: analyzeCall, Callback: streamCallback, CallOutcomes: requestOutcomes, CarriesEffects: carriesResponseEffects}
}

// Recognize Gin Context by full type identity without accessing private state.
// 使用完整类型身份识别 Gin Context，不访问框架私有状态。
func isContext(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == ginPackage && named.Obj().Name() == "Context"
}

// Extract an exact constant integer status code.
// 提取无损的可求值整数状态码。
func integer(v core.Value) string {
	if v.Constant != nil && v.Constant.Kind() == constant.Int {
		return v.Constant.ExactString()
	}
	return ""
}

// Extract an actual constant string.
// 提取代码中真实的字符串常量。
func literal(v core.Value) string {
	if v.Constant != nil && v.Constant.Kind() == constant.String {
		return constant.StringVal(v.Constant)
	}
	return ""
}

// Report unsupported Gin semantics instead of hiding unknowns behind default or empty Schemas.
// 输出明确的 Gin 能力边界，不能用 default 或空 Schema 隐藏未知。
func unresolved(c core.CallContext, message string) []core.Effect {
	return []core.Effect{{Kind: core.Unresolved, Source: c.Source, Message: "gin-swagger.analysis: " + message, Fix: "Register a centralized rule through the project generation entry point"}}
}

// Translate Gin calls into neutral effects while the core manages control flow.
// 将真实 Gin 调用转换成框架中立效果；控制流由核心调度。
func analyzeCall(c core.CallContext) ([]core.Effect, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != ginPackage {
		return nil, nil
	}
	signature, _ := c.Object.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil || !isContext(signature.Recv().Type()) {
		return nil, nil
	}
	args := c.Arguments
	name := c.Object.Name()
	source := c.Source
	source.Rule = "gin." + name
	source.Kind = "derived"
	arg := func(i int) core.Value {
		if i >= len(args) {
			return core.Value{Unknown: true}
		}
		return args[i]
	}
	response := func(media string, payload core.Value) []core.Effect {
		return renderResponse(c, normalizedGinStatus(integer(arg(0))), media, payload, nil)
	}
	switch name {
	case "ShouldBindJSON", "ShouldBindBodyWithJSON":
		return bindRequest(c, "json", arg(0)), nil
	case "ShouldBindQuery":
		return bindRequest(c, "query", arg(0)), nil
	case "ShouldBindUri":
		return bindRequest(c, "uri", arg(0)), nil
	case "ShouldBindHeader":
		return bindRequest(c, "header", arg(0)), nil
	case "ShouldBindWith", "ShouldBindBodyWith":
		return explicitBinder(c, arg(1), arg(0), name == "ShouldBindBodyWith"), nil
	case "JSON", "IndentedJSON", "AsciiJSON", "PureJSON":
		return response("application/json", arg(1)), nil
	case "AbortWithStatusJSON", "AbortWithStatusPureJSON":
		return append([]core.Effect{{Kind: core.Abort, Source: source}}, response("application/json", arg(1))...), nil
	case "Status":
		if interimStatus(normalizedGinStatus(integer(arg(0)))) {
			return unresolved(c, "Final commit after an interim status requires an explicit sequence rule"), nil
		}
		return []core.Effect{{Kind: core.ResponseStatus, Status: normalizedGinStatus(integer(arg(0))), Source: source}}, nil
	case "Abort":
		return []core.Effect{{Kind: core.Abort, Source: source}}, nil
	case "ShouldBind", "Bind":
		return unresolved(c, "automatic binder depends on the actual request method and media type; conditional facts are required"), nil
	case "AbortWithStatus", "AbortWithError":
		if interimStatus(normalizedGinStatus(integer(arg(0)))) {
			return unresolved(c, "Final commit after an interim status requires an explicit sequence rule"), nil
		}
		return []core.Effect{{Kind: core.ResponseCommit, Status: normalizedGinStatus(integer(arg(0))), Source: source}, {Kind: core.Abort, Source: source}}, nil
	case "SecureJSON", "JSONP":
		return unresolved(c, "prefixed or callback output cannot be treated as ordinary JSON"), nil
	case "String":
		return renderResponse(c, normalizedGinStatus(integer(arg(0))), "text/plain", core.Value{}, spec.Typed("string")), nil
	case "Data":
		return rawResponse(c, normalizedGinStatus(integer(arg(0))), literal(arg(1))), nil
	case "DataFromReader":
		return readerResponse(c, normalizedGinStatus(integer(arg(0))), literal(arg(2)), arg(1), arg(3), arg(4)), nil
	case "Render":
		return explicitRenderer(c, normalizedGinStatus(integer(arg(0))), arg(1)), nil
	case "SSEvent":
		return sseResponse(c, "-1", arg(0), core.Value{Type: types.Typ[types.String], Constant: constant.MakeString("")}, core.Value{Type: types.Typ[types.Uint], Constant: constant.MakeInt64(0)}, arg(1)), nil
	case "File", "FileAttachment", "FileFromFS", "Stream":
		return unresolved(c, fmt.Sprintf("%s requires a media type or stream output projection", name)), nil
	case "Next":
		return unresolved(c, "route snapshot only exposes the final handler; middleware chains require a centralized declaration"), nil
	case "Header":
		return []core.Effect{{Kind: core.ResponseHeader, Name: literal(arg(0)), Payload: arg(1), DeleteHeader: arg(1).Constant != nil && literal(arg(1)) == "", Source: source}}, nil
	case "Set", "Get", "MustGet", "GetString", "GetBool", "GetInt", "GetInt64", "GetFloat64", "GetTime", "GetDuration", "GetStringSlice", "GetStringMap", "GetStringMapString", "GetStringMapStringSlice", "Error", "ContentType", "IsAborted", "FullPath", "ClientIP", "RemoteIP":
		return nil, nil
	default:
		return unresolved(c, "Unrecognized Context call: "+name), nil
	}
}

// Unrecognized response-writer calls still carry wire effects and cannot be treated as pure calls.
// 未识别的响应 Writer 调用仍携带网络效果，不能当作普通纯函数忽略。
func carriesResponseEffects(t types.Type) bool {
	if isContext(t) {
		return true
	}
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	path, name := named.Obj().Pkg().Path(), named.Obj().Name()
	return name == "ResponseWriter" && (path == ginPackage || path == "net/http")
}
