package main

import "github.com/gin-gonic/gin"

// Keep ordinary integer fields and collections in tag-free automatic-binding input.
type AutomaticInput struct {
	// Item count.
	Count int
	// Identifier collection.
	IDs []int
}

// Select a binder by request method and media type within the same function.
func Automatic(c *gin.Context) {
	var input AutomaticInput
	if c.ShouldBind(&input) != nil {
		c.JSON(422, AutomaticInput{Count: -1})
		return
	}
	c.JSON(201, input)
}

// Preserve Gin's implicit error commit when automatic mandatory binding fails.
func AutomaticMandatory(c *gin.Context) {
	var input AutomaticInput
	if c.Bind(&input) != nil {
		return
	}
	c.JSON(201, input)
}
