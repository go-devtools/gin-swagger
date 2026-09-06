// Provide real behavior samples for mandatory binders.
package mustbind

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Exercise mandatory binding with tag-free fields.
type Input struct {

	// Request count.
	Count int
}

// Return immediately after binding fails.
func Checked(c *gin.Context) {
	var input Input
	if c.BindJSON(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// Continue the current handler even when the binding error is ignored.
func Ignored(c *gin.Context) {
	var input Input
	c.BindJSON(&input)
	c.JSON(201, input)
}

// A subsequent response cannot replace the binder's committed error status.
func Rewrite(c *gin.Context) {
	var input Input
	if err := c.BindJSON(&input); err != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// Use an explicit JSON binder.
func Explicit(c *gin.Context) {
	var input Input
	binder := binding.JSON
	if c.MustBindWith(&input, binder) != nil {
		return
	}
	c.JSON(201, input)
}

// Bind query parameters with automatic error commits.
func Query(c *gin.Context) {
	var input Input
	if c.BindQuery(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// Bind headers with automatic error commits.
func Header(c *gin.Context) {
	var input Input
	if c.BindHeader(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// Bind path parameters with automatic error commits.
func URI(c *gin.Context) {
	var input Input
	if c.BindUri(&input) != nil {
		return
	}
	c.JSON(201, input)
}
