// 提供强制绑定器的真实行为样本。

// Provide real behavior samples for mandatory binders.
package mustbind

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 通过无 tag 字段测试强制绑定。

// Exercise mandatory binding with tag-free fields.
type Input struct {
	// 请求数量。

	// Request count.
	Count int
}

// 在绑定失败后立即返回。

// Return immediately after binding fails.
func Checked(c *gin.Context) {
	var input Input
	if c.BindJSON(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// 忽略绑定错误仍继续执行当前 handler。

// Continue the current handler even when the binding error is ignored.
func Ignored(c *gin.Context) {
	var input Input
	c.BindJSON(&input)
	c.JSON(201, input)
}

// 后续响应不能覆盖绑定器已提交的错误状态。

// A subsequent response cannot replace the binder's committed error status.
func Rewrite(c *gin.Context) {
	var input Input
	if err := c.BindJSON(&input); err != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// 使用显式 JSON 绑定器。

// Use an explicit JSON binder.
func Explicit(c *gin.Context) {
	var input Input
	binder := binding.JSON
	if c.MustBindWith(&input, binder) != nil {
		return
	}
	c.JSON(201, input)
}

// 从查询参数强制绑定。

// Bind query parameters with automatic error commits.
func Query(c *gin.Context) {
	var input Input
	if c.BindQuery(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// 从请求头强制绑定。

// Bind headers with automatic error commits.
func Header(c *gin.Context) {
	var input Input
	if c.BindHeader(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// 从路径参数强制绑定。

// Bind path parameters with automatic error commits.
func URI(c *gin.Context) {
	var input Input
	if c.BindUri(&input) != nil {
		return
	}
	c.JSON(201, input)
}
