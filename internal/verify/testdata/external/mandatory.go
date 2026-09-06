package main

import "github.com/gin-gonic/gin"

// A mandatory binding failure still allows the current handler to write after independent installation.
func Mandatory(c *gin.Context) {
	var request Request
	if c.BindJSON(&request) != nil {
		c.JSON(422, Request{Name: "invalid"})
		return
	}
	c.JSON(201, request)
}
