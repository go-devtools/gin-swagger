package protocolbounds

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Return the value actually produced by a numeric parser.
type Parsed struct {
	// Parsed value, including zero or saturation when errors are ignored.
	Value int64
}

// Reject invalid native-width decimal input.
func CheckedAtoi(c *gin.Context) {
	value, err := strconv.Atoi(c.Query("value"))
	if err != nil {
		c.Status(400)
		return
	}
	c.JSON(200, Parsed{Value: int64(value)})
}

// Continue with Atoi's actual result even when conversion fails.
func IgnoredAtoi(c *gin.Context) {
	value, _ := strconv.Atoi(c.Query("value"))
	c.JSON(200, Parsed{Value: int64(value)})
}

// Reject decimal syntax errors and values outside the signed sixteen-bit range.
func CheckedParseInt(c *gin.Context) {
	value, err := strconv.ParseInt(c.Query("value"), 10, 16)
	if err != nil {
		c.Status(400)
		return
	}
	c.JSON(200, Parsed{Value: value})
}

// Preserve zero or a saturated value when a ParseInt error is deliberately ignored.
func IgnoredParseInt(c *gin.Context) {
	value, _ := strconv.ParseInt(c.Query("value"), 10, 16)
	c.JSON(200, Parsed{Value: value})
}

// Count UTF-8 bytes rather than JSON Schema string characters.
func ByteLength(c *gin.Context) {
	value := c.Query("value")
	if len(value) < 3 {
		c.Status(400)
		return
	}
	c.JSON(200, Parsed{Value: int64(len(value))})
}

// Serve a fixed regular text asset using the standard file server.
func File(c *gin.Context) { c.File("testdata/protocolbounds/report.txt") }

// Add a download name before invoking the same file-server behavior.
func Attachment(c *gin.Context) { c.FileAttachment("testdata/protocolbounds/report.txt", "report.txt") }

// Resolve the asset through an explicitly bounded filesystem root.
func FileSystem(c *gin.Context) { c.FileFromFS("report.txt", http.Dir("testdata/protocolbounds")) }

// Observe the actual filesystem failure rather than inventing a successful response.
func Missing(c *gin.Context) { c.File("testdata/protocolbounds/missing.txt") }

// Keep a different asset outside the explicitly declared file profile.
func Unmapped(c *gin.Context) { c.File("testdata/protocolbounds/other.json") }
