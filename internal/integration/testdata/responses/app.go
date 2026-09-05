// 提供真实响应样本，文档工具只读取这些既有 handler。
// Provide actual response samples whose existing handlers are only read by documentation tools.
package responses

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

// 返回响应中的业务值。
// Carry a business value in the response.
type Reply struct {
	// 展示名称。
	// Display name.
	Name string
}

// 格式化普通文本并设置响应头。
// Format plain text and set a response header.
func Plain(c *gin.Context) { c.Header("X-Ready", "yes"); c.String(201, "hello %s", c.Param("name")) }

// 直接传输未经 Base64 编码的字节。
// Send raw bytes without Base64 encoding.
func Raw(c *gin.Context) { c.Data(202, "application/octet-stream", []byte{0, 255, 10}) }

// 使用明确的标准 JSON Renderer。
// Use an explicit standard JSON renderer.
func RenderJSON(c *gin.Context) { c.Render(203, render.JSON{Data: Reply{Name: "Ada"}}) }

// 使用明确的标准文本 Renderer。
// Use an explicit standard text renderer.
func RenderText(c *gin.Context) { c.Render(200, render.String{Format: "rendered"}) }

// 使用明确的原始字节 Renderer。
// Use an explicit raw-byte renderer.
func RenderData(c *gin.Context) {
	c.Render(200, render.Data{ContentType: "application/octet-stream", Data: []byte{1, 2}})
}

// 无内容状态不会调用 JSON 序列化。
// A no-content status does not serialize the JSON payload.
func NoContent(c *gin.Context) { c.JSON(204, Reply{Name: "omitted"}) }

// 未修改状态不发送文本响应体。
// A not-modified status does not send a text body.
func NotModified(c *gin.Context) { c.String(304, "omitted") }

// 立即提交禁止访问状态。
// Immediately commit a forbidden status.
func AbortOnly(c *gin.Context) { c.AbortWithStatus(403) }

// 终止链不会终止当前函数，后续 JSON 保留已提交状态。
// Aborting the chain does not return from the function; later JSON keeps the committed status.
func AbortThenJSON(c *gin.Context) { c.AbortWithStatus(401); c.JSON(200, Reply{Name: "after abort"}) }

// 保存错误本身不会自动生成错误响应体。
// Recording an error does not automatically generate an error body.
func AbortError(c *gin.Context) { _ = c.AbortWithError(409, errors.New("conflict")) }

// 只保留提交前的最终响应头，空值删除已有头。
// Keep final pre-commit headers; an empty value removes a prior header.
func Headers(c *gin.Context) {
	c.Header("X-Mode", "old")
	c.Header("x-mode", "stable")
	c.Header("X-Removed", "value")
	c.Header("X-Removed", "")
	c.JSON(200, Reply{Name: "headers"})
	c.Header("X-Late", "ignored")
}

// 不同分支的头与状态不得互相污染。
// Keep headers and statuses isolated between branches.
func HeaderBranches(c *gin.Context) {
	c.Header("X-Branch", "initial")
	if c.Query("side") == "left" {
		c.Header("X-Branch", "left")
		c.JSON(200, Reply{Name: "left"})
		return
	}
	c.Header("X-Branch", "right")
	c.JSON(201, Reply{Name: "right"})
}

// 从读取器传输字节并声明明确长度及附加头。
// Stream bytes from a reader with a known length and extra headers.
func Reader(c *gin.Context) {
	c.Header("X-Reader", "original")
	c.DataFromReader(200, 3, "application/octet-stream", strings.NewReader("abc"), map[string]string{"X-Reader": "fallback", "X-Extra": "yes"})
}

// 多次完整 JSON 写入不能被误认为合法备选。
// Multiple complete JSON writes cannot be mistaken for valid alternatives.
func Multiple(c *gin.Context) { c.JSON(200, Reply{}); c.JSON(200, Reply{}) }

// 代表没有集中规则的自定义渲染器。
// Represent a custom renderer with no registered centralized rule.
type customRenderer struct{}

// 输出自定义格式的字节。
// Write bytes in a custom representation.
func (customRenderer) Render(w http.ResponseWriter) error {
	_, err := w.Write([]byte("custom"))
	return err
}

// 声明自定义媒体类型。
// Declare the custom media type.
func (customRenderer) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/x-custom")
}

// 无法识别的渲染器必须保留诊断。
// Preserve a diagnostic for an unrecognized renderer.
func UnknownRender(c *gin.Context) { c.Render(200, customRenderer{}) }

// 直接 Writer 输出尚需独立规则，不能被静默忽略。
// Direct writer output needs its own rule and must not be silently ignored.
func UnknownWriter(c *gin.Context) { _, _ = c.Writer.Write([]byte("raw")) }

// 使用标准缩进 JSON 渲染器。
// Use the standard indented JSON renderer.
func RenderIndented(c *gin.Context) { c.Render(200, render.IndentedJSON{Data: Reply{Name: "Ada"}}) }

// 使用标准 ASCII JSON 渲染器。
// Use the standard ASCII JSON renderer.
func RenderASCII(c *gin.Context) { c.Render(200, render.AsciiJSON{Data: Reply{Name: "Ada"}}) }

// 使用标准纯 JSON 渲染器并保留其换行。
// Use the standard pure JSON renderer and retain its newline.
func RenderPure(c *gin.Context) { c.Render(200, render.PureJSON{Data: Reply{Name: "Ada"}}) }

// 使用明确读取器 Renderer。
// Use an explicit reader renderer.
func RenderReader(c *gin.Context) {
	c.Render(200, render.Reader{ContentLength: 3, ContentType: "application/octet-stream", Reader: strings.NewReader("abc")})
}

// Renderer 指针仍指向同一个明确标准类型。
// A renderer pointer still identifies the same explicit standard type.
func RenderPointer(c *gin.Context) { c.Render(200, &render.JSON{Data: Reply{Name: "Ada"}}) }

// 未指定 JSON 数据的字面 Renderer 真实输出 null。
// A literal JSON renderer with no data actually emits null.
func RenderNull(c *gin.Context) { c.Render(200, render.JSON{}) }

// 临时响应后继续写入，最终网络状态需要专门的序列模型。
// Continue after an interim response; the final wire status requires a dedicated sequence model.
func Interim(c *gin.Context) { c.JSON(103, Reply{}); c.JSON(201, Reply{Name: "final"}) }

// 无响应体状态跳过 Reader 的长度和附加头写入。
// Bodyless statuses skip the reader's length and extra-header writes.
func ReaderNoContent(c *gin.Context) {
	c.DataFromReader(204, 3, "application/octet-stream", strings.NewReader("abc"), map[string]string{"X-Reader-Extra": "unused"})
}

// 显式 Reader Renderer 在 304 状态也不执行附加头逻辑。
// An explicit reader renderer also skips its extra-header logic for status 304.
func ReaderNotModified(c *gin.Context) {
	c.Render(304, render.Reader{ContentType: "application/octet-stream", ContentLength: 3, Reader: strings.NewReader("abc"), Headers: map[string]string{"X-Reader-Extra": "unused"}})
}

// 保留现有状态的 Reader 调用需要先解决状态条件。
// A reader call that preserves the current status needs its status condition resolved first.
func ReaderPending(c *gin.Context) {
	c.Status(204)
	c.DataFromReader(-1, 3, "application/octet-stream", strings.NewReader("abc"), nil)
}

// 注册真实业务路由，不包装 handler 或 Gin 注册方法。
// Register real business routes without wrapping handlers or Gin registration methods.
func Router() *gin.Engine {
	r := gin.New()
	r.GET("/plain/:name", Plain)
	r.GET("/raw", Raw)
	r.GET("/render-json", RenderJSON)
	r.GET("/render-text", RenderText)
	r.GET("/render-data", RenderData)
	r.GET("/no-content", NoContent)
	r.GET("/not-modified", NotModified)
	r.GET("/abort", AbortOnly)
	r.GET("/abort-json", AbortThenJSON)
	r.GET("/abort-error", AbortError)
	r.GET("/headers", Headers)
	r.GET("/header-branches", HeaderBranches)
	r.GET("/reader", Reader)
	r.GET("/multiple", Multiple)
	r.GET("/unknown-render", UnknownRender)
	r.GET("/unknown-writer", UnknownWriter)
	r.GET("/interim", Interim)
	r.GET("/render-indented", RenderIndented)
	r.GET("/render-ascii", RenderASCII)
	r.GET("/render-pure", RenderPure)
	r.GET("/render-reader", RenderReader)
	r.GET("/render-pointer", RenderPointer)
	r.GET("/render-null", RenderNull)
	r.GET("/reader-no-content", ReaderNoContent)
	r.GET("/reader-not-modified", ReaderNotModified)
	r.GET("/reader-pending", ReaderPending)
	return r
}
