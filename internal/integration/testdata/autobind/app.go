// Provide automatic-binding samples without DTO tags or handler wrappers.
package autobind

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Read input from the actually selected JSON, query, or form location.
type Input struct {
	// Display name.
	Name string
	// Request count.
	Count int
	// Identifier list.
	IDs []int
}

// Select a binder automatically and retain the business-defined error status.
func Optional(c *gin.Context) {
	var input Input
	if c.ShouldBind(&input) != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// Return the committed framework error immediately when automatic binding fails.
func Mandatory(c *gin.Context) {
	var input Input
	if c.Bind(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// An explicit Form binder still preserves method and media conditions.
func ExplicitForm(c *gin.Context) {
	var input Input
	if c.ShouldBindWith(&input, binding.Form) != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// Declare logical field presence, which needs additional handling across alternative locations.
type RequiredInput struct {
	// Request count.
	// @openapi required
	Count int
}

// Verify one handler's required declaration affects only the applicable conditions.
func Required(c *gin.Context) {
	var input RequiredInput
	if c.ShouldBind(&input) != nil {
		return
	}
	c.JSON(201, input)
}
