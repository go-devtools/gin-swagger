package compiler

import (
	"fmt"
	"go/constant"
	"go/types"
	"mime"

	core "github.com/openapi-golang/openapi/compiler"
)

// 从 Context 的实际公开字段类型构造 Writer 实参，不引入核心私有状态。
// Construct the writer argument from Context's actual public field type without core-private state.
func contextWriter(context core.Value) (core.Value, bool) {
	if value, ok := context.Fields["Writer"]; ok {
		return value, !value.Unknown
	}
	if context.Type == nil {
		return core.Value{}, false
	}
	typ := types.Unalias(context.Type)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	structure, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return core.Value{}, false
	}
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if field.Name() == "Writer" {
			return core.Value{Type: field.Type(), Object: field}, true
		}
	}
	return core.Value{}, false
}

// Gin Stream 先检查断连，再调用 step、Flush，最后按布尔结果重复。
// Gin Stream checks disconnection, invokes step, flushes, and then repeats according to the boolean result.
func streamCallback(c core.CallContext) (*core.CallbackPlan, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != ginPackage || c.Object.Name() != "Stream" || !isContext(c.Receiver.Type) {
		return nil, nil
	}
	writer, ok := contextWriter(c.Receiver)
	if !ok || len(c.Arguments) != 1 {
		return nil, fmt.Errorf("gin-swagger.analysis: Stream Writer 或回调未解决")
	}
	source := c.Source
	source.Kind, source.Rule = "derived", "gin.Stream.flush"
	falseValue := core.Value{Type: types.Typ[types.Bool], Constant: constant.MakeBool(false)}
	trueValue := core.Value{Type: types.Typ[types.Bool], Constant: constant.MakeBool(true)}
	return &core.CallbackPlan{Function: c.Arguments[0], Arguments: []core.Value{writer}, After: []core.Effect{{Kind: core.ResponseCommit, Status: "-1", Source: source}}, Results: []core.Value{falseValue}, Repeat: &core.CallbackRepeat{ContinueValue: true}, MayInterrupt: true, InterruptResults: []core.Value{trueValue}}, nil
}

// 使用完整类型身份识别 Gin Writer，普通文件或日志 Writer 不作为响应。
// Recognize Gin writers by full type identity without treating ordinary file or log writers as responses.
func isGinWriter(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == ginPackage && named.Obj().Name() == "ResponseWriter"
}

// 将标准 JSON 编码器的 Writer 身份随返回值传播，并保留 Encode 的错误分支。
// Propagate a standard JSON encoder's writer identity through its result and preserve Encode error branches.
func streamWriterOutcomes(c core.CallContext) ([]core.CallOutcome, error) {
	if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != "encoding/json" {
		return nil, nil
	}
	if c.Object.Name() == "NewEncoder" && len(c.Arguments) == 1 {
		result := core.Value{Type: c.Function.Package.Info.TypeOf(c.Call), NonNil: true, Fields: map[string]core.Value{"openapi.gin.writer": c.Arguments[0]}}
		return []core.CallOutcome{{Results: []core.Value{result}, Effects: []core.Effect{{Kind: core.Handled, Source: c.Source}}}}, nil
	}
	signature, ok := c.Object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, nil
	}
	typ := types.Unalias(signature.Recv().Type())
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Name() != "Encoder" {
		return nil, nil
	}
	writer, known := c.Receiver.Fields["openapi.gin.writer"]
	if !known {
		return nil, fmt.Errorf("gin-swagger.analysis: JSON Encoder 的输出 Writer 身份未解决")
	}
	var returned []core.Value
	for i := 0; i < signature.Results().Len(); i++ {
		returned = append(returned, core.Value{Type: signature.Results().At(i).Type()})
	}
	handled := []core.Effect{{Kind: core.Handled, Source: c.Source}}
	if !isGinWriter(writer.Type) {
		if writer.Type == nil || writer.Unknown {
			return nil, fmt.Errorf("gin-swagger.analysis: JSON Encoder 的 Writer 类型未解决")
		}
		if _, dynamic := writer.Type.Underlying().(*types.Interface); dynamic {
			return nil, fmt.Errorf("gin-swagger.analysis: JSON Encoder 的接口 Writer 可能指向响应，需要集中规则")
		}
		return []core.CallOutcome{{Results: returned, Effects: handled}}, nil
	}
	switch c.Object.Name() {
	case "SetEscapeHTML":
		return []core.CallOutcome{{Effects: handled}}, nil
	case "SetIndent":
		if len(c.Arguments) != 2 || c.Arguments[0].Constant == nil || c.Arguments[1].Constant == nil || literal(c.Arguments[0]) != "" || literal(c.Arguments[1]) != "" {
			return nil, fmt.Errorf("gin-swagger.analysis: NDJSON 不接受可能跨行的 JSON 缩进")
		}
		return []core.CallOutcome{{Effects: handled}}, nil
	case "Encode":
		if len(c.Arguments) != 1 || len(returned) != 1 {
			return nil, fmt.Errorf("gin-swagger.analysis: JSON Encode 签名未解决")
		}
		headers := c.Response.Headers
		if c.Response.Committed {
			headers = c.Response.CommittedHeaders
		}
		header, exists := headers["Content-Type"]
		media, _, err := mime.ParseMediaType(header.Value)
		if !exists || !header.Known || err != nil || (media != "application/x-ndjson" && media != "application/ndjson") {
			return nil, fmt.Errorf("gin-swagger.analysis: 逐项 JSON 输出需要明确的 NDJSON 媒体类型")
		}
		source := c.Source
		source.Kind, source.Rule = "derived", "gin.writer.json.Encode"
		effect := core.Effect{Kind: core.ResponseItem, Status: "-1", MediaType: media, PayloadMediaType: "application/json", Payload: c.Arguments[0], Source: source}
		success, failure := returned[0], returned[0]
		success.Nil = true
		failure.NonNil = true
		return []core.CallOutcome{{Results: []core.Value{success}, Effects: []core.Effect{effect}}, {Results: []core.Value{failure}, Effects: []core.Effect{effect}}}, nil
	}
	return nil, fmt.Errorf("gin-swagger.analysis: JSON Encoder 的响应调用未识别")
}
