package compiler

import (
	"encoding/json"
	"go/constant"
	"go/types"
	"mime"
	"strings"
	"unicode/utf8"

	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/spec"
)

// Recognize only the actual dependency's event type, rejecting custom renderers with the same short name.
// 只识别实际依赖的标准事件类型，拒绝同名的自定义 Renderer。
func isSSEEvent(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "github.com/gin-contrib/sse" && named.Obj().Name() == "Event"
}

// Read propagated standard-event fields, using actual Go zero values for omitted fields.
// 读取标准事件已传播的字段；未提供的字段采用真实 Go 零值。
func explicitSSE(c core.CallContext, status string, event core.Value) []core.Effect {
	if _, ok := types.Unalias(event.Type).(*types.Pointer); ok {
		if event.Nil || event.DynamicNil || !concreteNonNil(event) {
			return unresolved(c, "SSE Renderer pointer may be nil; successful rendering cannot be guaranteed")
		}
	}
	if event.Fields == nil {
		return unresolved(c, "SSE Renderer field value is unresolved")
	}
	field := func(name string, zero core.Value) core.Value {
		if value, ok := event.Fields[name]; ok {
			return value
		}
		return zero
	}
	text := core.Value{Type: types.Typ[types.String], Constant: constant.MakeString("")}
	return sseResponse(c, status, field("Event", text), field("Id", text), field("Retry", core.Value{Type: types.Typ[types.Uint], Constant: constant.MakeInt64(0)}), field("Data", core.Value{Type: types.Typ[types.UntypedNil], Nil: true}))
}

// Translate SSE headers, protocol items, and inner JSON payloads into neutral effects.
// 将 SSE 的响应头、逐项协议对象及 JSON 内层载荷转换为中立效果。
func sseResponse(c core.CallContext, status string, name, id, retry, payload core.Value) []core.Effect {
	status = normalizedGinStatus(status)
	if interimStatus(status) {
		return unresolved(c, "Interim SSE response status requires an explicit commit sequence rule")
	}
	source := c.Source
	source.Kind, source.Rule = "derived", "gin."+c.Object.Name()+".sse"
	header := func(name, value string) core.Effect {
		return core.Effect{Kind: core.ResponseHeader, Name: name, Payload: core.Value{Type: types.Typ[types.String], Constant: constant.MakeString(value)}, Source: source}
	}
	effects := []core.Effect{header("Content-Type", "text/event-stream;charset=utf-8")}
	if _, exists := c.Response.Headers["Cache-Control"]; !exists {
		effects = append(effects, header("Cache-Control", "no-cache"))
	}
	// An explicit bodyless Render writes only headers without invoking the event encoder.
	// 显式无正文 Render 只写响应头，不调用事件编码器。
	if status == "204" || status == "304" {
		return append(effects, core.Effect{Kind: core.ResponseCommit, Status: status, Source: source})
	}
	effectiveStatus := status
	if c.Response.Committed || status == "-1" {
		effectiveStatus = c.Response.Status
	}
	if effectiveStatus == "204" || effectiveStatus == "304" {
		return append(effects, core.Effect{Kind: core.ResponseCommit, Status: status, Source: source})
	}
	if c.Response.Committed {
		header, exists := c.Response.CommittedHeaders["Content-Type"]
		media, _, err := mime.ParseMediaType(header.Value)
		if !exists || !header.Known || err != nil || media != "text/event-stream" {
			return unresolved(c, "committed response has no explicit SSE media type; headers written later cannot change the wire representation")
		}
	}
	item := core.Effect{Kind: core.ResponseItem, Status: status, MediaType: "text/event-stream", Source: source}
	data, project, nonNull, maybeNil, err := ssePayload(payload)
	if err != "" {
		return unresolved(c, err)
	}
	if project {
		item.Payload = payload
		item.PayloadMediaType = "application/json"
		// Only known nil collections use null on the JSON path; interface non-nil identity cannot replace payload identity.
		// JSON 分支只有已知具体 nil 集合使用 null；接口自身的非空标志不能改写动态载荷。
		if payload.DynamicNil {
			item.Payload.Nil = true
		}
	} else {
		item.WireSchema = data
	}
	item.TransformSchema = func(schema *spec.Schema) (*spec.Schema, error) {
		if project {
			if nonNull {
				schema = &spec.Schema{SchemaObject: &spec.SchemaObject{AllOf: []*spec.Schema{schema}, Not: spec.Typed("null")}}
			}
			text := spec.Typed("string")
			text.ContentMediaType, text.ContentSchema = "application/json", schema
			schema = text
			if maybeNil {
				schema = &spec.Schema{SchemaObject: &spec.SchemaObject{AnyOf: []*spec.Schema{schema, sseTextSchema(core.Value{Type: payload.Type, Nil: true})}}}
			}
		}
		event := spec.Typed("object")
		event.Properties = map[string]*spec.Schema{"data": schema}
		required := []string{"data"}
		for _, field := range []struct {
			name  string
			value core.Value
		}{{"event", name}, {"id", id}} {
			value, known := field.value.Constant, false
			if value != nil && value.Kind() == constant.String {
				known = true
			}
			if known && (constant.StringVal(value) == "" || field.name == "id" && strings.ContainsRune(constant.StringVal(value), 0)) {
				continue
			}
			property := spec.Typed("string")
			if known {
				text := constant.StringVal(value)
				if utf8.ValidString(text) {
					property.Const = spec.Set[any](strings.TrimPrefix(strings.NewReplacer("\n", "\\n", "\r", "\\r").Replace(text), " "))
				}
				required = append(required, field.name)
			}
			event.Properties[field.name] = property
		}
		if retry.Constant == nil || retry.Constant.Kind() != constant.Int || constant.Sign(retry.Constant) > 0 {
			property := spec.Typed("integer")
			property.Minimum = spec.Set(json.Number("1"))
			if retry.Constant != nil && retry.Constant.Kind() == constant.Int {
				property.Const = spec.Set[any](json.Number(retry.Constant.ExactString()))
				required = append(required, "retry")
			}
			event.Properties["retry"] = property
		}
		event.Required = spec.Set(required)
		return event, nil
	}
	return append(effects, item)
}

// Distinguish interface identity from the concrete payload so boxing cannot make an unknown pointer definitely non-nil.
// 区分接口身份和其内部具体值，避免装箱把未知指针变成确定非 nil。
func concreteNonNil(value core.Value) bool {
	if value.Boxed {
		return value.DynamicNonNil
	}
	return value.NonNil
}

// Select encoding using gin-contrib/sse's exact byte assertion and single pointer dereference.
// 按 gin-contrib/sse 的精确字节断言及一次指针解引用选择编码路径。
func ssePayload(value core.Value) (wire *spec.Schema, project, nonNull, maybeNil bool, message string) {
	if value.Unknown || value.Type == nil {
		return nil, false, false, false, "The actual type of SSE data is unresolved"
	}
	t := types.Unalias(value.Type)
	if types.Identical(t, types.NewSlice(types.Typ[types.Byte])) {
		schema := sseTextSchema(value)
		if value.Nil || value.DynamicNil {
			schema.Const = spec.Set[any]("")
		}
		return schema, false, false, false, ""
	}
	pointer, ptr := t.Underlying().(*types.Pointer)
	if ptr {
		if value.Nil || value.DynamicNil {
			return sseTextSchema(value), false, false, false, ""
		}
		t = types.Unalias(pointer.Elem())
		maybeNil = !concreteNonNil(value)
	}
	switch t.Underlying().(type) {
	case *types.Struct:
		return nil, true, !hasSSEMethods(value.Type, "MarshalJSON", "MarshalText"), maybeNil, ""
	case *types.Map, *types.Slice:
		return nil, true, false, maybeNil, ""
	case *types.Interface:
		if value.Nil || value.DynamicNil {
			return sseTextSchema(value), false, false, false, ""
		}
		if !ptr {
			return nil, false, false, false, "The dynamic type of SSE data determines text or JSON encoding; register a centralized rule"
		}
	}
	return sseTextSchema(value), false, false, false, ""
}

// Constrain known text only when custom formatters cannot intervene; arrays and other formats retain string wire types.
// 只在真实格式化方法不参与时约束已知文本；数组和其他格式保持字符串网络表示。
func sseTextSchema(value core.Value) *spec.Schema {
	schema := spec.Typed("string")
	if hasSSEMethods(value.Type, "String", "Error", "Format") {
		return schema
	}
	text, known := "", false
	if value.Nil || value.DynamicNil {
		text, known = "<nil>", true
	} else if value.Constant != nil {
		switch value.Constant.Kind() {
		case constant.String:
			text, known = constant.StringVal(value.Constant), true
		case constant.Int, constant.Bool:
			text, known = value.Constant.ExactString(), true
		}
	}
	if known && utf8.ValidString(text) {
		parts := strings.Split(strings.ReplaceAll(text, "\r", "\\r"), "\n")
		for i := range parts {
			parts[i] = strings.TrimPrefix(parts[i], " ")
		}
		schema.Const = spec.Set[any](strings.Join(parts, "\n"))
	}
	return schema
}

// Inspect the actual method set so default constant rules cannot override custom formatting.
// 检查实际方法集，已知自定义格式不被默认编码的常量规则覆盖。
func hasSSEMethods(t types.Type, names ...string) bool {
	if t == nil {
		return false
	}
	set := types.NewMethodSet(t)
	for i := 0; i < set.Len(); i++ {
		for _, name := range names {
			if set.At(i).Obj().Name() == name {
				return true
			}
		}
	}
	return false
}
