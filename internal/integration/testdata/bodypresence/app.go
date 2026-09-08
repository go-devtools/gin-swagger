package bodypresence

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Describe a field requirement independently of the existence of an HTTP body.
type Input struct {
	// Name is required when a document payload is supplied; comments add no runtime validation.
	// @openapi required
	Name string
}

// Reject decoding failures before returning a successful JSON response.
func Checked(c *gin.Context) {
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// Ignore decoding failures and continue with zero-valued fields.
func Ignored(c *gin.Context) {
	var input Input
	c.ShouldBindJSON(&input)
	c.JSON(200, input)
}

// Replace a pending error status with a success after a decoding failure.
func Overridden(c *gin.Context) {
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Status(400)
	}
	c.JSON(200, input)
}

// Accept missing input through a deliberate bodyless success branch.
func EmptySuccess(c *gin.Context) {
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Status(204)
		return
	}
	c.JSON(201, input)
}

// Aborting middleware traversal does not prevent the current handler from succeeding.
func Aborted(c *gin.Context) {
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Abort()
	}
	c.JSON(200, input)
}

// The mandatory binder commits an error that later rendering cannot replace.
func Mandatory(c *gin.Context) {
	var input Input
	c.BindJSON(&input)
	c.JSON(201, input)
}

// Cache and decode the JSON body while retaining the binding error branch.
func Cached(c *gin.Context) {
	var input Input
	if err := c.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// Decode through a propagated explicit JSON binder.
func Explicit(c *gin.Context) {
	var input Input
	decoder := binding.JSON
	if err := c.ShouldBindWith(&input, decoder); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// Empty URL-encoded input binds successfully without runtime field validation.
func Form(c *gin.Context) {
	var input Input
	if err := c.ShouldBindWith(&input, binding.FormPost); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// Multipart framing must exist before a successful bind can return.
func Multipart(c *gin.Context) {
	var input Input
	if err := c.ShouldBindWith(&input, binding.FormMultipart); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// An ignored multipart error permits a successful response without any input bytes.
func IgnoredMultipart(c *gin.Context) {
	var input Input
	c.ShouldBindWith(&input, binding.FormMultipart)
	c.JSON(200, input)
}

// A successfully retrieved upload requires multipart request bytes.
func Upload(c *gin.Context) {
	file, err := c.FormFile("asset")
	if err != nil {
		c.Status(400)
		return
	}
	c.String(201, "%s", file.Filename)
}

// Select the real binder from the request method and media type.
func Automatic(c *gin.Context) {
	var input struct{ Name string }
	if err := c.ShouldBind(&input); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}

// Carry the decoding result through a normal helper.
func decode(c *gin.Context, input *Input) error { return c.ShouldBindJSON(input) }

// Preserve the helper's correlated binding result.
func Helper(c *gin.Context) {
	var input Input
	if err := decode(c, &input); err != nil {
		c.Status(400)
		return
	}
	c.JSON(201, input)
}
