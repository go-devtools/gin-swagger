package compiler

import (
	"go/constant"
	"go/types"
	"mime"
	"net/textproto"
	"sort"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// 用 Gin 实际采用的渲染格式构造响应效果，JSON 类型仍由核心投影。
// Construct response effects from Gin's actual renderer while leaving JSON type projection to the core.
func renderResponse(c core.CallContext, status, media string, payload core.Value, schema *spec.Schema) []core.Effect {
	if interimStatus(status) {
		return unresolved(c, "临时响应与最终状态需要完整的 Writer 提交序列规则")
	}
	if _, _, err := mime.ParseMediaType(media); err != nil {
		return unresolved(c, "响应媒体类型不是有效的明确值")
	}
	source := c.Source
	source.Kind = "derived"
	source.Rule = "gin." + c.Object.Name()
	return []core.Effect{{Kind: core.ResponseBody, Status: status, MediaType: media, Payload: payload, WireSchema: schema, Source: source}}
}

// 原始 HTTP 字节使用内容媒体类型注解，不添加 JSON 字符串或 Base64 约束。
// Describe raw HTTP bytes with a content media type instead of JSON string or Base64 constraints.
func rawResponse(c core.CallContext, status, media string) []core.Effect {
	return renderResponse(c, status, media, core.Value{}, &spec.Schema{SchemaObject: &spec.SchemaObject{ContentMediaType: media}})
}

// 读取器附加头只在原响应头为空时应用，明确长度覆盖同名附加字段。
// Apply reader headers only when the response header is empty; a known length overrides its extra-header entry.
func readerResponse(c core.CallContext, status, media string, length, reader, headers core.Value) []core.Effect {
	if status == "-1" {
		return unresolved(c, "保留现有状态的读取器需要状态相关的头部规则")
	}
	if status == "204" || status == "304" {
		// Gin 此时只调用 WriteContentType，跳过 Reader.Render 的所有附加头。
		// Gin only calls WriteContentType here, skipping all extra headers from Reader.Render.
		return rawResponse(c, status, media)
	}
	if reader.Nil {
		return unresolved(c, "读取器为 nil，不能产生正常响应")
	}
	if !headers.Nil && headers.Fields == nil {
		return unresolved(c, "读取器附加头需要可分析的字面映射或集中规则")
	}
	knownLength := length.Constant != nil && length.Constant.Kind() == constant.Int
	withLength := knownLength && constant.Sign(length.Constant) >= 0
	if !knownLength {
		return unresolved(c, "读取器长度控制响应头，必须可求值或提供集中规则")
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
			return unresolved(c, "附加头含大小写不同的重复名称，写入顺序无法消歧")
		}
		seen[canonical] = true
		if withLength && canonical == "Content-Length" {
			if name != "Content-Length" {
				return unresolved(c, "长度头与自动生成字段的大小写冲突")
			}
			continue
		}
		if canonical == "Content-Type" {
			// Renderer 已先写入明确媒体类型，附加头无法覆盖它。
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

// 通过完整包和类型身份识别 Renderer，不按短名称猜测自定义实现。
// Match renderers by full package and type identity rather than custom implementations with matching short names.
func explicitRenderer(c core.CallContext, status string, renderer core.Value) []core.Effect {
	typ := renderer.Type
	if typ == nil {
		return unresolved(c, "Renderer 类型未解决")
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != ginPackage+"/render" {
		return unresolved(c, "自定义 Renderer 需要集中编解码规则")
	}
	if renderer.Fields == nil {
		return unresolved(c, "Renderer 字段值无法静态确定")
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
		return unresolved(c, "尚未识别的 Renderer 编码："+named.Obj().Name())
	}
}

// Gin 包装 Writer 的临时状态不能按普通最终状态推导。
// Interim statuses in Gin's wrapped writer cannot be inferred as ordinary final statuses.
func interimStatus(status string) bool {
	return len(status) == 3 && status[0] == '1'
}
