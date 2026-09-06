// Demonstrate first generation and startup mounting in an independent business module.
package main

import (
	"os"

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/openapi"
)

// Carry a creation request without struct tags.
type Request struct {
	// Display name.
	// @openapi required minLength=2
	Name string
}

// Submit a request and return the result through ordinary business code.
// @openapi request mediaType="application/json" type="Request" required
// @openapi response status=201 mediaType="application/json" type="Request"
// @openapi response status="default" mediaType="application/json" type="Request"
func Create(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.Header("X-Source", "consumer")
	c.JSON(201, request)
}

// Return plain text to verify independently consumed rendering rules.
func Text(c *gin.Context) { c.String(200, "ready") }

// The independent consumer's text byte collection is not a Base64 request parameter.
type QueryInput struct {
	// Byte identifiers.
	IDs []byte
}

// Verify independently installed codec extensions through real query binding.
func Search(c *gin.Context) {
	var input QueryInput
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, input)
}

// Keep ordinary Gin route registration and optionally mount documentation at startup.
func router(mount bool) (*gin.Engine, *openapi.Document, error) {
	r := gin.New()
	r.POST("/users", Create)
	r.GET("/text", Text)
	r.GET("/search", Search)
	if !mount {
		return r, nil, nil
	}
	doc, err := ginswagger.Mount(r, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Independent consumer", Version: "1"}})
	return r, doc, err
}

// Export the document linked at runtime without starting a public server.
func main() {
	_, document, err := router(true)
	if err != nil {
		panic(err)
	}
	if err := document.WriteFile(os.Args[1]); err != nil {
		panic(err)
	}
}
