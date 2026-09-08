// Demonstrates generation from tag-free business code and documentation mounting at startup.
package main

import (
	"fmt"
	"net/http"
	"os"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	"github.com/go-devtools/gin-swagger/examples/basic/internal/apidoc"
	"github.com/go-devtools/openapi"
)

// Information submitted when creating a user.
type CreateUserRequest struct {

	// User name.
	// @openapi required nonnull minLength=3 maxLength=32 examples=["alice"]
	Name string
}

// User information returned to the client.
type User struct {

	// User identifier.
	// @openapi examples=[1024]
	ID int64

	// User name.
	Name string
}

// A request error that can be returned to clients.
type APIError struct {

	// Stable error code.
	Code string

	// Error description.
	Message string
}

// Create a user
//
// Returns the user information after successful creation.
// @openapi tags=["Users"]
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIError{Code: "INVALID_JSON", Message: "Invalid request body format"})
		return
	}
	n := utf8.RuneCountInString(req.Name)
	if n < 3 || n > 32 {
		c.JSON(http.StatusBadRequest, APIError{Code: "INVALID_NAME", Message: "Username must contain 3 to 32 characters"})
		return
	}
	c.JSON(http.StatusCreated, User{ID: 1024, Name: req.Name})
}

// Builds the application routes and mounts documentation once at startup.
func Router() (*gin.Engine, *openapi.Document, error) {
	r := gin.New()
	r.POST("/users", CreateUser)
	r.GET("/examples/types", GetTypeExamples)
	r.POST("/examples/enums", SelectEnums)
	items := r.Group("/examples/items")
	items.GET("/:id", GetItem)
	items.POST("", CreateItem)
	items.PUT("/:id", ReplaceItem)
	items.PATCH("/:id", PatchItem)
	items.DELETE("/:id", DeleteItem)
	r.GET("/examples/legacy/items/:id", LegacyGetItem)
	bearer := r.Group("/examples/auth", requireDemoBearer)
	bearer.GET("/bearer", GetAuthorizedExample)
	doc, err := ginswagger.Mount(r, apidoc.Bundle(), documentationConfig())
	return r, doc, err
}

// Supports exporting the specification for offline verification and listens locally by default.
func main() {
	r, doc, err := Router()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) == 3 && os.Args[1] == "--export" {
		if err = doc.WriteFile(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	if err = r.Run(address); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
