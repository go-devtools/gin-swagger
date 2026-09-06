package main

import (
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

// The business object carried by events uses no additional tags.
type Message struct {
	Name  string
	Count int
}

// A named byte collection does not satisfy the encoder's exact []byte type assertion.
type NamedBytes []byte

// Consecutive events preserve their names and JSON structures.
func SSEObjects(c *gin.Context) {
	c.Status(202)
	c.SSEvent("message", Message{Name: "Ada", Count: 2})
	c.SSEvent("count", map[string]int{"count": 3})
}

// Encode plain text with the actual multiline SSE rules.
func SSEText(c *gin.Context) { c.SSEvent("text", "first\nsecond\rline") }

// The SSE protocol removes one leading space after each field colon.
func SSEWhitespace(c *gin.Context) { c.SSEvent(" name", " first\n  second") }

// Require only metadata actually emitted; the encoder escapes line endings.
func SSEMetadata(c *gin.Context) {
	c.Render(201, sse.Event{Event: "notice\nnext", Id: " id\r", Retry: 1500, Data: Message{Name: "Ada", Count: 2}})
}

// Unset metadata must not automatically become required.
func SSEDefault(c *gin.Context) { c.Render(200, sse.Event{Data: "hello"}) }

// Pointer forms of standard events preserve the complete renderer type identity.
func SSEEventPointer(c *gin.Context) { c.Render(200, &sse.Event{Data: "pointer"}) }

// The protocol parser ignores ids containing NUL.
func SSEInvalidID(c *gin.Context) { c.Render(200, sse.Event{Id: "bad\x00id", Data: "hello"}) }

// A nil interface is emitted through the text path.
func SSENil(c *gin.Context) { c.SSEvent("nil", nil) }

// A nil pointer is emitted as text rather than JSON null.
func SSENilPointer(c *gin.Context) { var value *Message; c.SSEvent("nil", value) }

// Interface boxing does not change the reflected kind of its nil pointer payload.
func SSEBoxedNil(c *gin.Context) { var value *Message; var boxed any = value; c.SSEvent("nil", boxed) }

// A nil map uses JSON encoding and produces the string null.
func SSENilMap(c *gin.Context) { var value map[string]int; c.SSEvent("map", value) }

// A nil slice uses the JSON path.
func SSENilSlice(c *gin.Context) { var value []string; c.SSEvent("slice", value) }

// Non-nil object pointers retain the actual JSON object constraints.
func SSEPointer(c *gin.Context) { c.SSEvent("pointer", &Message{Name: "Ada", Count: 2}) }

// Arrays take the formatter's text path in this encoder.
func SSEArray(c *gin.Context) { c.SSEvent("array", [2]int{1, 2}) }

// An exact []byte is raw text rather than Base64.
func SSEBytes(c *gin.Context) { c.SSEvent("bytes", []byte("first\nsecond")) }

// Named byte slices use JSON encoding, so data contains quoted Base64.
func SSENamedBytes(c *gin.Context) { c.SSEvent("bytes", NamedBytes{65, 66}) }

// A nil byte slice emits an event whose data is an empty string.
func SSENilBytes(c *gin.Context) { var value []byte; c.SSEvent("bytes", value) }

// Dynamic event names may be empty, so their string representation is optional.
func SSEDynamicName(c *gin.Context) { c.SSEvent(c.Query("name"), "hello") }

// An explicit cache header is not replaced by the SSE default.
func SSECache(c *gin.Context) {
	c.Header("Cache-Control", "private")
	c.Header("Content-Type", "text/plain")
	c.SSEvent("cache", "hello")
}

// Subsequent events cannot overwrite a committed correct media type or status.
func SSECommitted(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.AbortWithStatus(409)
	c.SSEvent("committed", "hello")
}

// Status and header changes after an event do not enter the wire snapshot.
func SSELateHeaders(c *gin.Context) {
	c.SSEvent("first", "one")
	c.Status(201)
	c.Header("Cache-Control", "private")
	c.Header("Content-Type", "text/plain")
	c.SSEvent("second", "two")
}

// A 204 skips data encoding, so the unsent channel must not be projected.
func SSENoContent(c *gin.Context) { c.Render(204, sse.Event{Data: struct{ C chan int }{}}) }

// A 304 retains SSE headers without a response body.
func SSENotModified(c *gin.Context) { c.Render(304, sse.Event{Data: struct{ C chan int }{}}) }

// Mixing a complete body and an event is not an alternative response branch.
func SSEMixed(c *gin.Context) { c.SSEvent("one", "hello"); c.String(200, "invalid frame") }

// JSON encoding failures cannot be disguised as complete events.
func SSEUnsupported(c *gin.Context) { c.SSEvent("bad", struct{ C chan int }{}) }

// A late SSE header cannot correct another media type already committed.
func SSEWrongCommittedMedia(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.AbortWithStatus(200)
	c.SSEvent("wrong", "hello")
}

// A nil renderer pointer has no normal rendering contract.
func SSENilRenderer(c *gin.Context) { var event *sse.Event; c.Render(200, event) }

// A zero-valued event variable must behave like an empty event literal.
func SSEZeroEvent(c *gin.Context) { var event sse.Event; c.Render(200, event) }

// An explicit any conversion preserves the nil pointer's actual dynamic type.
func SSEConvertedNil(c *gin.Context) { c.SSEvent("nil", any((*Message)(nil))) }

// Local data flow does not assume the actual value of this global object.
var DynamicPointer = &Message{Name: "Global", Count: 4}

// A global pointer's nil case must also be a valid text alternative.
var DynamicNilPointer *Message

// A non-nil sample of an unknown pointer verifies the JSON branch.
func SSEUnknownPointer(c *gin.Context) { var value any = DynamicPointer; c.SSEvent("pointer", value) }

// A nil sample of an unknown pointer verifies the text branch.
func SSEUnknownNilPointer(c *gin.Context) {
	var value any = DynamicNilPointer
	c.SSEvent("pointer", value)
}

// A non-nil pointer to a nil slice still encodes as JSON null.
func SSEPointerSlice(c *gin.Context) { var value []string; c.SSEvent("slice", &value) }

// A text formatter can override a named basic type's literal value.
type Formatted string

// Expose fmt method-set behavior with fixed text; the generator never executes this method.
func (Formatted) String() string { return "formatted text" }

// A named string's output is controlled by its String method.
func SSEFormatted(c *gin.Context) { c.SSEvent("formatted", Formatted("raw")) }

// A committed response without an SSE media header is detected as another type by the HTTP server.
func SSECommittedWithoutMedia(c *gin.Context) { c.AbortWithStatus(200); c.SSEvent("wrong", "hello") }

// Invalid UTF-8 bytes are replaced independently during protocol decoding.
func SSEInvalidUTF8(c *gin.Context) { c.SSEvent("utf8", []byte{0xff, 0xfe}) }

// Non-empty raw metadata remains an emitted field when parsing yields an empty string.
func SSEEmptyMetadata(c *gin.Context) { c.Render(200, sse.Event{Event: " ", Id: " ", Data: ""}) }

// A mutable global retry value is not inferred as a source constant.
var DynamicRetry uint = 1500

// A dynamic retry value may be zero, so retry remains optional.
func SSEDynamicRetry(c *gin.Context) { c.Render(200, sse.Event{Retry: DynamicRetry, Data: "retry"}) }

// Preserve unsigned retry values as exact decimal integers.
func SSELargeRetry(c *gin.Context) { c.Render(200, sse.Event{Retry: 4294967295, Data: "retry"}) }

// A late SSE header cannot conceal the missing media type at commit time.
func SSELateCommittedMedia(c *gin.Context) {
	c.AbortWithStatus(200)
	c.Header("Content-Type", "text/event-stream")
	c.SSEvent("wrong", "hello")
}

// Non-positive status codes preserve Gin's pending status.
func SSEIgnoredStatus(c *gin.Context) {
	c.Status(202)
	c.Status(0)
	c.Render(-2, sse.Event{Data: "pending"})
}

// A renderer status of zero preserves the default successful status.
func SSEZeroStatus(c *gin.Context) { c.Render(0, sse.Event{Data: "default"}) }

// A pending 204 likewise omits event data that is never sent.
func SSEPendingNoContent(c *gin.Context) { c.Status(204); c.SSEvent("hidden", struct{ C chan int }{}) }
