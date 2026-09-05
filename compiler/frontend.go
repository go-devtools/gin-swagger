// 通过核心公开 SDK 提供 Gin 生成前端；普通业务程序不导入本包。
// Expose Gin generation rules through the public compiler SDK.
package compiler

import (
	"fmt"
	"go/constant"
	"go/types"

	core "github.com/openapi-golang/openapi/compiler"
)

// 明确限定 Gin 的真实包身份，禁止仅按方法短名匹配。
// Match the complete Gin package identity rather than short method names.
const ginPackage = "github.com/gin-gonic/gin"

// 注册静态 Gin 规则，同一个值供 CLI 与项目自定义生成器组合。
// Provide the same frontend to the CLI and custom generation entry points.
func Frontend() core.Frontend {
	return core.Frontend{Name: "gin-v1.12-front-v1", Match: func(f core.Function) bool {
		return f.Signature.Params().Len() == 1 && isContext(f.Signature.Params().At(0).Type())
	}, Call: analyzeCall, CarriesEffects: isContext}
}

// 使用完整类型身份识别 Gin Context，不访问框架私有状态。
// Recognize Gin Context by full type identity without accessing private state.
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

// 提取无损的可求值整数状态码。
// Extract an exact constant integer status code.
func integer(v core.Value) string {
	if v.Constant != nil && v.Constant.Kind() == constant.Int {
		return v.Constant.ExactString()
	}
	return ""
}

// 提取代码中真实的字符串常量。
// Extract an actual constant string.
func literal(v core.Value) string {
	if v.Constant != nil && v.Constant.Kind() == constant.String {
		return constant.StringVal(v.Constant)
	}
	return ""
}

// 输出明确的 Gin 能力边界，不能用 default 或空 Schema 隐藏未知。
// Report unsupported Gin semantics instead of hiding unknowns behind default or empty Schemas.
func unresolved(c core.CallContext, message string) []core.Effect {
	return []core.Effect{{Kind: core.Unresolved, Source: c.Source, Message: "gin-swagger.analysis: " + message, Fix: "通过项目生成入口注册集中规则"}}
}

// 将真实 Gin 调用转换成框架中立效果；控制流由核心调度。
// Translate Gin calls into neutral effects while the core manages control flow.
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
		return []core.Effect{{Kind: core.ResponseBody, Status: integer(arg(0)), MediaType: media, Payload: payload, Source: source}}
	}
	switch name {
	case "ShouldBindJSON", "ShouldBindBodyWithJSON":
		payload := arg(0)
		if ptr, ok := payload.Type.(*types.Pointer); ok {
			payload.Type = ptr.Elem()
		}
		return []core.Effect{{Kind: core.RequestBody, MediaType: "application/json", Payload: payload, Source: source}}, nil
	case "JSON", "IndentedJSON", "AsciiJSON", "PureJSON":
		return response("application/json", arg(1)), nil
	case "AbortWithStatusJSON", "AbortWithStatusPureJSON":
		return append([]core.Effect{{Kind: core.Abort, Source: source}}, response("application/json", arg(1))...), nil
	case "Status":
		return []core.Effect{{Kind: core.ResponseStatus, Status: integer(arg(0)), Source: source}}, nil
	case "Abort":
		return []core.Effect{{Kind: core.Abort, Source: source}}, nil
	case "Param", "Query", "GetQuery", "DefaultQuery", "QueryArray", "GetHeader", "Cookie", "PostForm", "DefaultPostForm", "GetPostForm", "PostFormArray":
		location := "query"
		if name == "Param" {
			location = "path"
		}
		if name == "GetHeader" {
			location = "header"
		}
		if name == "Cookie" {
			location = "cookie"
		}
		if name == "PostForm" || name == "DefaultPostForm" || name == "GetPostForm" || name == "PostFormArray" {
			return unresolved(c, "表单字段需要 requestBody 表单投影"), nil
		}
		typ := types.Type(types.Typ[types.String])
		if name == "QueryArray" {
			typ = types.NewSlice(typ)
		}
		return []core.Effect{{Kind: core.ParameterRead, Name: literal(arg(0)), In: location, Payload: core.Value{Type: typ}, Source: source}}, nil
	case "ShouldBind", "Bind":
		return unresolved(c, "自动绑定器依赖实际请求方法和媒体类型，需要条件事实"), nil
	case "BindJSON", "BindQuery", "BindUri", "BindHeader", "MustBindWith":
		return unresolved(c, "强制绑定器隐含错误写入，需要路径相关的提交规则"), nil
	case "AbortWithStatus", "AbortWithError":
		return unresolved(c, "此调用立即提交状态，后续写入需要明确的提交顺序分析"), nil
	case "SecureJSON", "JSONP":
		return unresolved(c, "带前缀或回调的输出不能当作普通 JSON"), nil
	case "String", "Data", "DataFromReader", "Redirect", "Render", "File", "FileAttachment", "FileFromFS", "SSEvent", "Stream":
		return unresolved(c, fmt.Sprintf("%s 需要媒体类型或流式输出投影", name)), nil
	case "Next":
		return unresolved(c, "路由快照仅暴露末位 handler，中间件链需要集中声明"), nil
	case "Header":
		return []core.Effect{{Kind: core.ResponseHeader, Name: literal(arg(0)), Payload: arg(1), Source: source}}, nil
	case "Set", "Get", "MustGet", "GetString", "GetBool", "GetInt", "GetInt64", "GetFloat64", "GetTime", "GetDuration", "GetStringSlice", "GetStringMap", "GetStringMapString", "GetStringMapStringSlice", "Error", "ContentType", "IsAborted", "FullPath", "ClientIP", "RemoteIP":
		return nil, nil
	default:
		return unresolved(c, "尚未识别的 Context 调用："+name), nil
	}
}
