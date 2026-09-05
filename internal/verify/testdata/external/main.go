// 展示独立业务 module 的首次生成与启动层挂载。
// Demonstrate first generation and startup mounting in an independent business module.
package main

import (
	"os"

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/openapi"
)

// 创建请求，不使用 struct tag。
// Carry a creation request without struct tags.
type Request struct {
	// 显示名称。
	// Display name.
	// @openapi required minLength=2
	Name string
}

// 提交请求并按原有业务方式返回结果。
// Submit a request and return the result through ordinary business code.
func Create(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.Header("X-Source", "consumer")
	c.JSON(201, request)
}

// 返回普通文本，验证独立消费的新渲染规则。
// Return plain text to verify independently consumed rendering rules.
func Text(c *gin.Context) { c.String(200, "ready") }

// 独立消费者的文本字节集合不是 Base64 请求参数。
// The independent consumer's text byte collection is not a Base64 request parameter.
type QueryInput struct {
	// 字节编号。 Byte identifiers.
	IDs []byte
}

// 通过真实查询绑定验证独立安装后的 codec 扩展。
// Verify independently installed codec extensions through real query binding.
func Search(c *gin.Context) {
	var input QueryInput
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, input)
}

// 保持 Gin 路由注册方式，按需在启动层挂载文档。
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

// 导出真实运行时链接的文档，不启动公网服务器。
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
