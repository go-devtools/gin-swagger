// 展示零 tag 业务代码的一次生成与启动层文档挂载。

// Demonstrates generation from tag-free business code and documentation mounting at startup.
package main

import (
	"fmt"
	"net/http"
	"os"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/gin-swagger/examples/basic/internal/apidoc"
	"github.com/openapi-golang/openapi"
)

// 创建用户时提交的信息。

// Information submitted when creating a user.
type CreateUserRequest struct {
	// 用户名。

	// User name.
	// @openapi required nonnull minLength=3 maxLength=32 examples=["alice"]
	Name string
}

// 返回给客户端的用户信息。

// User information returned to the client.
type User struct {
	// 用户编号。

	// User identifier.
	// @openapi examples=[1024]
	ID int64
	// 用户名。

	// User name.
	Name string
}

// 可公开的请求错误。

// A request error that can be returned to clients.
type APIError struct {
	// 稳定错误代码。

	// Stable error code.
	Code string
	// 错误说明。

	// Error description.
	Message string
}

// 创建用户
// 创建成功后返回用户信息。

// Create a user
//
// Returns the user information after successful creation.
// @openapi tags=["Users"]
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIError{Code: "INVALID_JSON", Message: "请求体格式错误"})
		return
	}
	n := utf8.RuneCountInString(req.Name)
	if n < 3 || n > 32 {
		c.JSON(http.StatusBadRequest, APIError{Code: "INVALID_NAME", Message: "用户名长度必须为 3～32 个字符"})
		return
	}
	c.JSON(http.StatusCreated, User{ID: 1024, Name: req.Name})
}

// 构造与原业务相同的路由，再在启动层挂载一次文档。

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

// 允许导出规范供离线验收，默认只监听本地地址。

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
