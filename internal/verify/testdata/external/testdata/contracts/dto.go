// 保留依赖包的真实 DTO 语义，不能将依赖函数加入应用候选。

// Keep DTO metadata in an imported package excluded from source-root patterns.
package contracts

import "github.com/gin-gonic/gin"

// 明确闭合的角色取值。

// A role with an explicitly closed value set.
// @openapi enum
type Role string

// 可用角色及其独立说明。

// Declare the known role values and their labels.
const (
	// 管理员。

	// Administrator.
	Admin Role = "admin"
	// 编辑者。

	// Editor.
	Editor Role = "editor"
)

// 已知常量不应使普通命名类型自动闭合。

// Known constants do not close this ordinary named string.
type State string

// 已知状态仅是普通常量。

// A known state remains an ordinary constant.
const Ready State = "ready"

// 多种真实绑定器共用的零 tag 请求。

// A tag-free request shared by actual binders.
type Request struct {
	// 导入 DTO 的展示名称。

	// Display name from the imported DTO.
	// @openapi required minLength=3 examples=["Alice"]
	Name string
	// 所选角色。

	// Selected role.
	Role Role
	// 开放状态。

	// An open state.
	State State
}

// 真实泛型字段应保留来源注释。

// A generic envelope whose fields retain their declaration metadata.
type Envelope[T any] struct {
	// 实际响应载荷。

	// The actual payload from the imported envelope.
	// @openapi required
	Data T
}

// 未引用时不影响有效路由的无效元数据。

// Malformed metadata is reported only when this dependency type is projected.
type Malformed struct {
	// @openapi examples=[
	Value string
}

// 依赖中的类型矛盾需保留结构化字段来源。

// An incompatible source constraint retains a structured field location.
type Incompatible struct {
	// @openapi minimum=1
	Value string
}

// 依赖函数不是源码根中的业务候选。

// This dependency handler must not enter application candidate discovery.
func LibraryHandler(c *gin.Context) { c.Status(204) }
