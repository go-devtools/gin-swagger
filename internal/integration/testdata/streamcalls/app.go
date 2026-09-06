package streamcalls

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
)

// A stream item with real JSON fields.
type StreamItem struct {
	Name  string
	Count int
}

// A single callback stops after returning false.
func StreamOnce(c *gin.Context) {
	c.Status(202)
	c.Stream(func(io.Writer) bool { c.SSEvent("one", "hello"); return false })
}

// Repeated callbacks count through the same captured cell.
func StreamRepeated(c *gin.Context) {
	n := 0
	c.Stream(func(io.Writer) bool { n++; c.SSEvent("count", n); return n < 3 })
}

// A returned closure preserves this factory invocation's context and value.
func stepFactory(c *gin.Context, value string) func(io.Writer) bool {
	return func(io.Writer) bool { c.SSEvent("factory", value); return false }
}

// A factory callback uses its actual captured value.
func StreamFactory(c *gin.Context) {
	step := stepFactory(c, "captured")
	alias := step
	c.Stream(alias)
}

// Each flush commits status even when the callback writes no body.
func StreamEmpty(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Status(202)
	c.Stream(func(io.Writer) bool { return false })
	c.Status(201)
}

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

// An encoder created outside the callback retains its actual writer identity when captured.
func StreamCapturedEncoder(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	encoder := json.NewEncoder(c.Writer)
	c.Stream(func(io.Writer) bool { _ = encoder.Encode(StreamItem{Name: "outside", Count: 4}); return false })
}

// Missing line-oriented media must not be guessed as NDJSON.
func StreamMissingMedia(c *gin.Context) {
	c.Stream(func(w io.Writer) bool { _ = json.NewEncoder(w).Encode(StreamItem{}); return false })
}

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

// A nil callback is not treated as an empty stream.
func StreamNil(c *gin.Context) { c.Stream(nil) }

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

// A production-style long-lived stream ends when its peer closes without requiring a finite business loop.
func StreamUntilClosed(c *gin.Context) {
	c.Stream(func(io.Writer) bool { c.SSEvent("tick", "ready"); return true })
}

// Local source does not establish this interface writer's dynamic implementation.
var UnknownStreamWriter io.Writer

// An unknown writer that could target the response must not be ignored.
func StreamUnknownWriter(c *gin.Context) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Stream(func(io.Writer) bool { _ = json.NewEncoder(UnknownStreamWriter).Encode(StreamItem{}); return false })
}
