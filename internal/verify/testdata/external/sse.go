package main

import (
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

// 事件中的业务对象不使用额外标签。

// The business object carried by events uses no additional tags.
type Message struct {
	Name  string
	Count int
}

// 命名字节集合不满足编码器的精确 []byte 类型断言。

// A named byte collection does not satisfy the encoder's exact []byte type assertion.
type NamedBytes []byte

// 连续事件保留各自名称与 JSON 结构。

// Consecutive events preserve their names and JSON structures.
func SSEObjects(c *gin.Context) {
	c.Status(202)
	c.SSEvent("message", Message{Name: "Ada", Count: 2})
	c.SSEvent("count", map[string]int{"count": 3})
}

// 普通文本按真实 SSE 多行规则编码。

// Encode plain text with the actual multiline SSE rules.
func SSEText(c *gin.Context) { c.SSEvent("text", "first\nsecond\rline") }

// 每个字段冒号后的首个空格由 SSE 协议移除。

// The SSE protocol removes one leading space after each field colon.
func SSEWhitespace(c *gin.Context) { c.SSEvent(" name", " first\n  second") }

// 元数据只要求实际发出的字段，换行在编码器中转义。

// Require only metadata actually emitted; the encoder escapes line endings.
func SSEMetadata(c *gin.Context) {
	c.Render(201, sse.Event{Event: "notice\nnext", Id: " id\r", Retry: 1500, Data: Message{Name: "Ada", Count: 2}})
}

// 未设置的元数据不应被自动声明为必填。

// Unset metadata must not automatically become required.
func SSEDefault(c *gin.Context) { c.Render(200, sse.Event{Data: "hello"}) }

// 指针形式的标准事件仍保留完整类型身份。

// Pointer forms of standard events preserve the complete renderer type identity.
func SSEEventPointer(c *gin.Context) { c.Render(200, &sse.Event{Data: "pointer"}) }

// NUL id 被协议解析器忽略。

// The protocol parser ignores ids containing NUL.
func SSEInvalidID(c *gin.Context) { c.Render(200, sse.Event{Id: "bad\x00id", Data: "hello"}) }

// nil 接口通过文本路径写出。

// A nil interface is emitted through the text path.
func SSENil(c *gin.Context) { c.SSEvent("nil", nil) }

// nil 指针通过文本路径写出而不是 JSON null。

// A nil pointer is emitted as text rather than JSON null.
func SSENilPointer(c *gin.Context) { var value *Message; c.SSEvent("nil", value) }

// 接口装箱不改变其内部 nil 指针的反射种类。

// Interface boxing does not change the reflected kind of its nil pointer payload.
func SSEBoxedNil(c *gin.Context) { var value *Message; var boxed any = value; c.SSEvent("nil", boxed) }

// nil map 使用 JSON 路径编码为字符串形式的 null。

// A nil map uses JSON encoding and produces the string null.
func SSENilMap(c *gin.Context) { var value map[string]int; c.SSEvent("map", value) }

// nil slice 使用 JSON 路径。

// A nil slice uses the JSON path.
func SSENilSlice(c *gin.Context) { var value []string; c.SSEvent("slice", value) }

// 非 nil 对象指针保留实际 JSON 对象约束。

// Non-nil object pointers retain the actual JSON object constraints.
func SSEPointer(c *gin.Context) { c.SSEvent("pointer", &Message{Name: "Ada", Count: 2}) }

// 数组在此编码器中走 fmt 文本路径。

// Arrays take the formatter's text path in this encoder.
func SSEArray(c *gin.Context) { c.SSEvent("array", [2]int{1, 2}) }

// 精确 []byte 作为原始文本而不是 Base64。

// An exact []byte is raw text rather than Base64.
func SSEBytes(c *gin.Context) { c.SSEvent("bytes", []byte("first\nsecond")) }

// 命名字节切片经过 JSON 编码，data 包含带引号的 Base64。

// Named byte slices use JSON encoding, so data contains quoted Base64.
func SSENamedBytes(c *gin.Context) { c.SSEvent("bytes", NamedBytes{65, 66}) }

// 空字节切片发送一个具有空字符串 data 的事件。

// A nil byte slice emits an event whose data is an empty string.
func SSENilBytes(c *gin.Context) { var value []byte; c.SSEvent("bytes", value) }

// 动态事件名称可能为空，因此只保留其可选字符串表示。

// Dynamic event names may be empty, so their string representation is optional.
func SSEDynamicName(c *gin.Context) { c.SSEvent(c.Query("name"), "hello") }

// 显式缓存头不会被 SSE 默认值替换。

// An explicit cache header is not replaced by the SSE default.
func SSECache(c *gin.Context) {
	c.Header("Cache-Control", "private")
	c.Header("Content-Type", "text/plain")
	c.SSEvent("cache", "hello")
}

// 已提交的正确媒体头和状态不会被后续事件改写。

// Subsequent events cannot overwrite a committed correct media type or status.
func SSECommitted(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.AbortWithStatus(409)
	c.SSEvent("committed", "hello")
}

// 事件之后的状态及头修改不会进入网络快照。

// Status and header changes after an event do not enter the wire snapshot.
func SSELateHeaders(c *gin.Context) {
	c.SSEvent("first", "one")
	c.Status(201)
	c.Header("Cache-Control", "private")
	c.Header("Content-Type", "text/plain")
	c.SSEvent("second", "two")
}

// 204 跳过实际数据编码，因此不应投影未发送的 channel。

// A 204 skips data encoding, so the unsent channel must not be projected.
func SSENoContent(c *gin.Context) { c.Render(204, sse.Event{Data: struct{ C chan int }{}}) }

// 304 保留 SSE 头但没有响应体。

// A 304 retains SSE headers without a response body.
func SSENotModified(c *gin.Context) { c.Render(304, sse.Event{Data: struct{ C chan int }{}}) }

// 普通正文和事件混写不是可替换的响应分支。

// Mixing a complete body and an event is not an alternative response branch.
func SSEMixed(c *gin.Context) { c.SSEvent("one", "hello"); c.String(200, "invalid frame") }

// JSON 编码失败不能伪造为完整事件。

// JSON encoding failures cannot be disguised as complete events.
func SSEUnsupported(c *gin.Context) { c.SSEvent("bad", struct{ C chan int }{}) }

// 已提交的其他媒体类型不能被晚写的 SSE 头纠正。

// A late SSE header cannot correct another media type already committed.
func SSEWrongCommittedMedia(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.AbortWithStatus(200)
	c.SSEvent("wrong", "hello")
}

// nil Renderer 指针没有正常渲染契约。

// A nil renderer pointer has no normal rendering contract.
func SSENilRenderer(c *gin.Context) { var event *sse.Event; c.Render(200, event) }

// 普通变量的零值事件应等同空字面事件。

// A zero-valued event variable must behave like an empty event literal.
func SSEZeroEvent(c *gin.Context) { var event sse.Event; c.Render(200, event) }

// 显式 any 转换保留 nil 指针的真实动态类型。

// An explicit any conversion preserves the nil pointer's actual dynamic type.
func SSEConvertedNil(c *gin.Context) { c.SSEvent("nil", any((*Message)(nil))) }

// 此全局对象的实际值不由局部数据流假定。

// Local data flow does not assume the actual value of this global object.
var DynamicPointer = &Message{Name: "Global", Count: 4}

// 全局指针的 nil 情况也必须是可验证的文本备选。

// A global pointer's nil case must also be a valid text alternative.
var DynamicNilPointer *Message

// 未知指针的非空样本验证 JSON 分支。

// A non-nil sample of an unknown pointer verifies the JSON branch.
func SSEUnknownPointer(c *gin.Context) { var value any = DynamicPointer; c.SSEvent("pointer", value) }

// 未知指针的空样本验证文本分支。

// A nil sample of an unknown pointer verifies the text branch.
func SSEUnknownNilPointer(c *gin.Context) {
	var value any = DynamicNilPointer
	c.SSEvent("pointer", value)
}

// 指向 nil slice 的非空指针依旧通过 JSON 编码为 null。

// A non-nil pointer to a nil slice still encodes as JSON null.
func SSEPointerSlice(c *gin.Context) { var value []string; c.SSEvent("slice", &value) }

// 文本格式化方法可以覆盖命名基础类型的字面值。

// A text formatter can override a named basic type's literal value.
type Formatted string

// 使用固定文本暴露 fmt 方法集，生成器不执行此方法。

// Expose fmt method-set behavior with fixed text; the generator never executes this method.
func (Formatted) String() string { return "formatted text" }

// 命名字符串的输出受其 String 方法控制。

// A named string's output is controlled by its String method.
func SSEFormatted(c *gin.Context) { c.SSEvent("formatted", Formatted("raw")) }

// 已提交且没有 SSE 媒体头的正文会被真实 HTTP 服务识别为其他类型。

// A committed response without an SSE media header is detected as another type by the HTTP server.
func SSECommittedWithoutMedia(c *gin.Context) { c.AbortWithStatus(200); c.SSEvent("wrong", "hello") }

// 非法 UTF-8 字节在协议解码时逐个替换。

// Invalid UTF-8 bytes are replaced independently during protocol decoding.
func SSEInvalidUTF8(c *gin.Context) { c.SSEvent("utf8", []byte{0xff, 0xfe}) }

// 原始非空元数据解析为空字符串时仍是已发送字段。

// Non-empty raw metadata remains an emitted field when parsing yields an empty string.
func SSEEmptyMetadata(c *gin.Context) { c.Render(200, sse.Event{Event: " ", Id: " ", Data: ""}) }

// 可变全局重试值不作为源码常量推导。

// A mutable global retry value is not inferred as a source constant.
var DynamicRetry uint = 1500

// 动态重试值可能为零，所以 retry 保持可选。

// A dynamic retry value may be zero, so retry remains optional.
func SSEDynamicRetry(c *gin.Context) { c.Render(200, sse.Event{Retry: DynamicRetry, Data: "retry"}) }

// 无符号重试值按精确十进制整数保留。

// Preserve unsigned retry values as exact decimal integers.
func SSELargeRetry(c *gin.Context) { c.Render(200, sse.Event{Retry: 4294967295, Data: "retry"}) }

// 晚写的 SSE 头不能掩盖提交时缺少媒体类型的事实。

// A late SSE header cannot conceal the missing media type at commit time.
func SSELateCommittedMedia(c *gin.Context) {
	c.AbortWithStatus(200)
	c.Header("Content-Type", "text/event-stream")
	c.SSEvent("wrong", "hello")
}

// 非正数状态在 Gin 中保留待提交状态。

// Non-positive status codes preserve Gin's pending status.
func SSEIgnoredStatus(c *gin.Context) {
	c.Status(202)
	c.Status(0)
	c.Render(-2, sse.Event{Data: "pending"})
}

// 零状态的 Renderer 保留默认成功状态。

// A renderer status of zero preserves the default successful status.
func SSEZeroStatus(c *gin.Context) { c.Render(0, sse.Event{Data: "default"}) }

// 待提交的 204 同样不投影没有发送的事件数据。

// A pending 204 likewise omits event data that is never sent.
func SSEPendingNoContent(c *gin.Context) { c.Status(204); c.SSEvent("hidden", struct{ C chan int }{}) }
