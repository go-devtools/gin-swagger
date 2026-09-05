// 提供不需要 DTO tag 或 handler 包装的自动绑定样本。
// Provide automatic-binding samples without DTO tags or handler wrappers.
package autobind

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 从实际选中的 JSON、查询或表单位置读取输入。
// Read input from the actually selected JSON, query, or form location.
type Input struct {
	// 显示名称。 Display name.
	Name string
	// 请求数量。 Request count.
	Count int
	// 编号列表。 Identifier list.
	IDs []int
}

// 自动选择绑定器，保留业务自己的错误状态。
// Select a binder automatically and retain the business-defined error status.
func Optional(c *gin.Context) {
	var input Input
	if c.ShouldBind(&input) != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// 自动绑定失败时立即返回已提交的框架错误。
// Return the committed framework error immediately when automatic binding fails.
func Mandatory(c *gin.Context) {
	var input Input
	if c.Bind(&input) != nil {
		return
	}
	c.JSON(201, input)
}

// 显式 Form 绑定器仍保留方法和媒体类型条件。
// An explicit Form binder still preserves method and media conditions.
func ExplicitForm(c *gin.Context) {
	var input Input
	if c.ShouldBindWith(&input, binding.Form) != nil {
		c.JSON(422, gin.H{"error": "invalid"})
		return
	}
	c.JSON(201, input)
}

// 声明逻辑字段必须存在，跨位置来源需要额外契约处理。
// Declare logical field presence, which needs additional handling across alternative locations.
type RequiredInput struct {
	// 请求数量。 Request count.
	// @openapi required
	Count int
}

// 用同一个 handler 验证必填声明只影响适用条件。
// Verify one handler's required declaration affects only the applicable conditions.
func Required(c *gin.Context) {
	var input RequiredInput
	if c.ShouldBind(&input) != nil {
		return
	}
	c.JSON(201, input)
}
