// Provide actual response samples whose existing handlers are only read by documentation tools.
package responses

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

// Carry a business value in the response.
type Reply struct {

	// Display name.
	Name string
}

// Format plain text and set a response header.
func Plain(c *gin.Context) { c.Header("X-Ready", "yes"); c.String(201, "hello %s", c.Param("name")) }

// Send raw bytes without Base64 encoding.
func Raw(c *gin.Context) { c.Data(202, "application/octet-stream", []byte{0, 255, 10}) }

// Use an explicit standard JSON renderer.
func RenderJSON(c *gin.Context) { c.Render(203, render.JSON{Data: Reply{Name: "Ada"}}) }

// Use an explicit standard text renderer.
func RenderText(c *gin.Context) { c.Render(200, render.String{Format: "rendered"}) }

// Use an explicit raw-byte renderer.
func RenderData(c *gin.Context) {
	c.Render(200, render.Data{ContentType: "application/octet-stream", Data: []byte{1, 2}})
}

// A no-content status does not serialize the JSON payload.
func NoContent(c *gin.Context) { c.JSON(204, Reply{Name: "omitted"}) }

// A not-modified status does not send a text body.
func NotModified(c *gin.Context) { c.String(304, "omitted") }

// Immediately commit a forbidden status.
func AbortOnly(c *gin.Context) { c.AbortWithStatus(403) }

// Aborting the chain does not return from the function; later JSON keeps the committed status.
func AbortThenJSON(c *gin.Context) { c.AbortWithStatus(401); c.JSON(200, Reply{Name: "after abort"}) }

// Recording an error does not automatically generate an error body.
func AbortError(c *gin.Context) { _ = c.AbortWithError(409, errors.New("conflict")) }

// Keep final pre-commit headers; an empty value removes a prior header.
func Headers(c *gin.Context) {
	c.Header("X-Mode", "old")
	c.Header("x-mode", "stable")
	c.Header("X-Removed", "value")
	c.Header("X-Removed", "")
	c.JSON(200, Reply{Name: "headers"})
	c.Header("X-Late", "ignored")
}

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

// Stream bytes from a reader with a known length and extra headers.
func Reader(c *gin.Context) {
	c.Header("X-Reader", "original")
	c.DataFromReader(200, 3, "application/octet-stream", strings.NewReader("abc"), map[string]string{"X-Reader": "fallback", "X-Extra": "yes"})
}

// Multiple complete JSON writes cannot be mistaken for valid alternatives.
func Multiple(c *gin.Context) { c.JSON(200, Reply{}); c.JSON(200, Reply{}) }

// Represent a custom renderer with no registered centralized rule.
type customRenderer struct{}

// Write bytes in a custom representation.
func (customRenderer) Render(w http.ResponseWriter) error {
	_, err := w.Write([]byte("custom"))
	return err
}

// Declare the custom media type.
func (customRenderer) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/x-custom")
}

// Preserve a diagnostic for an unrecognized renderer.
func UnknownRender(c *gin.Context) { c.Render(200, customRenderer{}) }

// Direct writer output needs its own rule and must not be silently ignored.
func UnknownWriter(c *gin.Context) { _, _ = c.Writer.Write([]byte("raw")) }

// Use the standard indented JSON renderer.
func RenderIndented(c *gin.Context) { c.Render(200, render.IndentedJSON{Data: Reply{Name: "Ada"}}) }

// Use the standard ASCII JSON renderer.
func RenderASCII(c *gin.Context) { c.Render(200, render.AsciiJSON{Data: Reply{Name: "Ada"}}) }

// Use the standard pure JSON renderer and retain its newline.
func RenderPure(c *gin.Context) { c.Render(200, render.PureJSON{Data: Reply{Name: "Ada"}}) }

// Use an explicit reader renderer.
func RenderReader(c *gin.Context) {
	c.Render(200, render.Reader{ContentLength: 3, ContentType: "application/octet-stream", Reader: strings.NewReader("abc")})
}

// A renderer pointer still identifies the same explicit standard type.
func RenderPointer(c *gin.Context) { c.Render(200, &render.JSON{Data: Reply{Name: "Ada"}}) }

// A literal JSON renderer with no data actually emits null.
func RenderNull(c *gin.Context) { c.Render(200, render.JSON{}) }

// Continue after an interim response; the final wire status requires a dedicated sequence model.
func Interim(c *gin.Context) { c.JSON(103, Reply{}); c.JSON(201, Reply{Name: "final"}) }

// Bodyless statuses skip the reader's length and extra-header writes.
func ReaderNoContent(c *gin.Context) {
	c.DataFromReader(204, 3, "application/octet-stream", strings.NewReader("abc"), map[string]string{"X-Reader-Extra": "unused"})
}

// An explicit reader renderer also skips its extra-header logic for status 304.
func ReaderNotModified(c *gin.Context) {
	c.Render(304, render.Reader{ContentType: "application/octet-stream", ContentLength: 3, Reader: strings.NewReader("abc"), Headers: map[string]string{"X-Reader-Extra": "unused"}})
}

// A reader call that preserves the current status needs its status condition resolved first.
func ReaderPending(c *gin.Context) {
	c.Status(204)
	c.DataFromReader(-1, 3, "application/octet-stream", strings.NewReader("abc"), nil)
}

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
