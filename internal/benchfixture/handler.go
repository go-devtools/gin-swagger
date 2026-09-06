// Provide actual compiled handlers for route-linking benchmarks.
// 为路由链接基准提供真实编译的 handler。
package benchfixture

import "github.com/gin-gonic/gin"

// 创建请求数据。

// Creation input.
type Request struct {
	// 展示名称。

	// Display name.
	// @openapi required minLength=3
	Name string
}

// 通过既有 Gin 调用读写 JSON。

// Read and write JSON through ordinary Gin calls.
func Handle(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, request)
}
