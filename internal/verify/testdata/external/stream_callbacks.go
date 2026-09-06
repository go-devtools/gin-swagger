package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
)

// 带有真实 JSON 字段的流条目。

// A stream item with real JSON fields.
type StreamItem struct {
	Name  string
	Count int
}

// 单次回调在返回 false 后结束。

// A single callback stops after returning false.
func StreamOnce(c *gin.Context) {
	c.Status(202)
	c.Stream(func(io.Writer) bool { c.SSEvent("one", "hello"); return false })
}

// 多次回调通过同一个捕获单元计数。

// Repeated callbacks count through the same captured cell.
func StreamRepeated(c *gin.Context) {
	n := 0
	c.Stream(func(io.Writer) bool { n++; c.SSEvent("count", n); return n < 3 })
}

// 返回闭包保留本次工厂调用的 Context 与值。

// A returned closure preserves this factory invocation's context and value.
func stepFactory(c *gin.Context, value string) func(io.Writer) bool {
	return func(io.Writer) bool { c.SSEvent("factory", value); return false }
}

// 工厂回调使用真实捕获值。

// A factory callback uses its actual captured value.
func StreamFactory(c *gin.Context) {
	step := stepFactory(c, "captured")
	alias := step
	c.Stream(alias)
}

// 每次 Flush 提交状态，即使回调没有写正文。

// Each flush commits status even when the callback writes no body.
func StreamEmpty(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Status(202)
	c.Stream(func(io.Writer) bool { return false })
	c.Status(201)
}

// 通过回调参数创建 JSON 编码器，逐行发送真实对象。

// Create a JSON encoder from the callback argument and send real objects line by line.
func StreamNDJSON(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	n := 0
	c.Stream(func(w io.Writer) bool {
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(StreamItem{Name: "Ada", Count: n}); err != nil {
			return false
		}
		n++
		return n < 3
	})
}

// 捕获在回调外创建的编码器也保留真实 Writer 身份。

// An encoder created outside the callback retains its actual writer identity when captured.
func StreamCapturedEncoder(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	encoder := json.NewEncoder(c.Writer)
	c.Stream(func(io.Writer) bool { _ = encoder.Encode(StreamItem{Name: "outside", Count: 4}); return false })
}

// 未设置逐行媒体类型不能猜测为 NDJSON。

// Missing line-oriented media must not be guessed as NDJSON.
func StreamMissingMedia(c *gin.Context) {
	c.Stream(func(w io.Writer) bool { _ = json.NewEncoder(w).Encode(StreamItem{}); return false })
}

// 缩进使单条 JSON 跨多行，不满足 NDJSON 分帧。

// Indentation splits one JSON value across lines and violates NDJSON framing.
func StreamIndented(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Stream(func(w io.Writer) bool {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		_ = e.Encode(StreamItem{})
		return false
	})
}

// nil 回调不会被当作空流。

// A nil callback is not treated as an empty stream.
func StreamNil(c *gin.Context) { c.Stream(nil) }

// 首次空回调的 Flush 已提交缺少媒体类型的头，后续 SSE 不能补救。

// A first empty callback flush commits missing media headers that later SSE cannot repair.
func StreamLateMedia(c *gin.Context) {
	n := 0
	c.Stream(func(io.Writer) bool {
		n++
		if n == 2 {
			c.SSEvent("late", "hello")
		}
		return n < 2
	})
}

// 生产式长连接在对端关闭后结束，不能要求业务添加有限循环。

// A production-style long-lived stream ends when its peer closes without requiring a finite business loop.
func StreamUntilClosed(c *gin.Context) {
	c.Stream(func(io.Writer) bool { c.SSEvent("tick", "ready"); return true })
}

// 接口 Writer 的动态实现不由局部源码确定。

// Local source does not establish this interface writer's dynamic implementation.
var UnknownStreamWriter io.Writer

// 不能忽略可能指向响应的未知 Writer。

// An unknown writer that could target the response must not be ignored.
func StreamUnknownWriter(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Stream(func(io.Writer) bool { _ = json.NewEncoder(UnknownStreamWriter).Encode(StreamItem{}); return false })
}
