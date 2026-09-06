// Provide actual compiled handlers for route-linking benchmarks.
package benchfixture

import "github.com/gin-gonic/gin"

// Creation input.
type Request struct {

	// Display name.
	// @openapi required minLength=3
	Name string
}

// Read and write JSON through ordinary Gin calls.
func Handle(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, request)
}
