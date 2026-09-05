package main

import "github.com/gin-gonic/gin"

// 无 tag 的自动绑定输入保留普通整数字段与集合。
// Keep ordinary integer fields and collections in tag-free automatic-binding input.
type AutomaticInput struct {
	// 数量。 Item count.
	Count int
	// 编号集合。 Identifier collection.
	IDs []int
}

// 同一函数通过请求方法和媒体类型选择绑定器。
// Select a binder by request method and media type within the same function.
func Automatic(c *gin.Context) {
	var input AutomaticInput
	if c.ShouldBind(&input) != nil {
		c.JSON(422, AutomaticInput{Count: -1})
		return
	}
	c.JSON(201, input)
}

// 自动强制绑定失败后保留 Gin 的隐含错误提交。
// Preserve Gin's implicit error commit when automatic mandatory binding fails.
func AutomaticMandatory(c *gin.Context) {
	var input AutomaticInput
	if c.Bind(&input) != nil {
		return
	}
	c.JSON(201, input)
}
