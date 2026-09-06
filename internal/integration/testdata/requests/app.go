// Provide tag-free request-binding samples whose business source cannot be rewritten by the generator.
package requests

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"mime/multipart"
	"time"
)

// Declare canonical role values; ordinary string decoding does not enforce the enum.
// @openapi enum
type Role string

// Unspecified role.
const RoleUnset Role = ""

// Administrator.
const RoleAdmin Role = "admin"

// Viewer.
const RoleViewer Role = "viewer"

// Bind primitives, arrays, pointers, and time values from text fields.
type Input struct {

	// User role.
	Role Role

	// Display name.
	// @openapi required minLength=2
	Name string

	// Count.
	Count int

	// Byte identifiers rather than Base64.
	IDs []byte

	// Optional limit.
	Limit *int

	// Exactly two values.
	Pair [2]int

	// Whether enabled.
	Active bool

	// RFC3339 timestamp.
	At time.Time

	// Go duration text.
	Delay time.Duration
}

// Path names come from actual Go fields.
type PathInput struct {

	// Resource identifier.
	ID int
}

// Match header fields using HTTP case rules.
type HeaderInput struct {

	// Request credential.
	Token string

	// Request count.
	Count int
}

// Multipart fields describe uploads instead of exposing JSON FileHeader objects.
type UploadInput struct {

	// Upload title.
	Title string

	// One file.
	// @openapi required
	File *multipart.FileHeader

	// Multiple files.
	Files []*multipart.FileHeader
}

// Bind query parameters and retain actual error branches.
func Query(c *gin.Context) {
	var input Input
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// An explicitly aliased binder retains the same rules.
func ExplicitQuery(c *gin.Context) {
	var input Input
	binder := binding.Query
	if err := c.ShouldBindWith(&input, binder); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// Bind the resource identifier from the registered path.
func URI(c *gin.Context) {
	var input PathInput
	if err := c.ShouldBindUri(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// Bind text values from request headers.
func Header(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindHeader(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// Explicitly select the URL-encoded form binder that reads only the request body.
func FormPost(c *gin.Context) {
	var input Input
	if err := c.ShouldBindWith(&input, binding.FormPost); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// Use the multipart binder and handle missing files through ordinary control flow.
func Multipart(c *gin.Context) {
	var input UploadInput
	if err := c.ShouldBindWith(&input, binding.FormMultipart); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	if input.File == nil {
		c.JSON(400, gin.H{"error": "file required"})
		return
	}
	c.JSON(200, gin.H{"Title": input.Title, "Filename": input.File.Filename, "Count": len(input.Files)})
}

// Explicit JSON binders share JSON projection with convenience methods.
func JSON(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindWith(&input, binding.JSON); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// An explicit cached-body JSON binder retains its real media type.
func BodyJSON(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// Explicit non-JSON binders must not inherit JSON encoding rules.
func XML(c *gin.Context) {
	var input HeaderInput
	_ = c.ShouldBindWith(&input, binding.XML)
	c.JSON(200, input)
}

// Register ordinary Gin routes without replacing binding logic through documentation mounting.
func Router() *gin.Engine {
	r := gin.New()
	r.GET("/query", Query)
	r.GET("/query-explicit", ExplicitQuery)
	r.GET("/uri/:ID", URI)
	r.GET("/header", Header)
	r.POST("/form", FormPost)
	r.POST("/multipart", Multipart)
	r.POST("/json", JSON)
	r.POST("/body-json", BodyJSON)
	r.POST("/xml", XML)
	r.GET("/embedded", Embedded)
	r.GET("/custom", Custom)
	r.GET("/repeated-header", RepeatedHeader)
	return r
}

// Ordinary anonymous embedding follows Gin's recursive field traversal.
type EmbeddedFields struct {

	// Shared name.
	Shared string
}

// Embedded time has no bindable exported children and must not invent a Time parameter.
type EmbeddedInput struct {
	EmbeddedFields
	time.Time
}

// Bind actual embedded fields instead of JSON promotion precedence.
func Embedded(c *gin.Context) {
	var input EmbeddedInput
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, gin.H{"Shared": input.Shared})
}

// Custom parameter decoding must be described by a centralized mapping.
type CustomParam string

// Convert input into a custom business value.
func (value *CustomParam) UnmarshalParam(raw string) error {
	*value = CustomParam("custom:" + raw)
	return nil
}

// A request containing a custom parameter decoder.
type CustomInput struct{ Value CustomParam }

// Do not claim a complete input contract without a registered custom codec.
func Custom(c *gin.Context) { var input CustomInput; _ = c.ShouldBindQuery(&input); c.JSON(200, input) }

// Repeated headers are not comma-separated arrays.
type RepeatedHeaders struct{ Values []string }

// Preserve diagnostics for default repeated-header serialization that cannot be represented.
func RepeatedHeader(c *gin.Context) {
	var input RepeatedHeaders
	_ = c.ShouldBindHeader(&input)
	c.JSON(200, input)
}
