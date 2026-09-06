// 提供零 tag 请求绑定样本，生成器不能改写业务源码。

// Provide tag-free request-binding samples whose business source cannot be rewritten by the generator.
package requests

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"mime/multipart"
	"time"
)

// 声明规范化的角色取值，普通字符串解码并不执行枚举校验。

// Declare canonical role values; ordinary string decoding does not enforce the enum.
// @openapi enum
type Role string

// 未指定角色。

// Unspecified role.
const RoleUnset Role = ""

// 管理员。

// Administrator.
const RoleAdmin Role = "admin"

// 查看者。

// Viewer.
const RoleViewer Role = "viewer"

// 通过文本字段绑定基本类型、数组、指针与时间。

// Bind primitives, arrays, pointers, and time values from text fields.
type Input struct {
	// 用户角色。

	// User role.
	Role Role
	// 显示名称。

	// Display name.
	// @openapi required minLength=2
	Name string
	// 数量。

	// Count.
	Count int
	// 字节编号而不是 Base64。

	// Byte identifiers rather than Base64.
	IDs []byte
	// 可选限制。

	// Optional limit.
	Limit *int
	// 固定两个值。

	// Exactly two values.
	Pair [2]int
	// 是否启用。

	// Whether enabled.
	Active bool
	// RFC3339 时间。

	// RFC3339 timestamp.
	At time.Time
	// Go 持续时间文本。

	// Go duration text.
	Delay time.Duration
}

// 路径字段名称来自真实 Go 字段。

// Path names come from actual Go fields.
type PathInput struct {
	// 资源标识。

	// Resource identifier.
	ID int
}

// 请求头按 HTTP 大小写规则匹配字段。

// Match header fields using HTTP case rules.
type HeaderInput struct {
	// 请求凭据。

	// Request credential.
	Token string
	// 请求数量。

	// Request count.
	Count int
}

// 文件上传由 multipart 字段表达，不能被当成 JSON FileHeader 对象。

// Multipart fields describe uploads instead of exposing JSON FileHeader objects.
type UploadInput struct {
	// 上传标题。

	// Upload title.
	Title string
	// 单个文件。

	// One file.
	// @openapi required
	File *multipart.FileHeader
	// 多个文件。

	// Multiple files.
	Files []*multipart.FileHeader
}

// 绑定查询参数并保留实际错误分支。

// Bind query parameters and retain actual error branches.
func Query(c *gin.Context) {
	var input Input
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 显式绑定器经别名传递后仍采用同一规则。

// An explicitly aliased binder retains the same rules.
func ExplicitQuery(c *gin.Context) {
	var input Input
	binder := binding.Query
	if err := c.ShouldBindWith(&input, binder); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 仅从已注册路径绑定资源编号。

// Bind the resource identifier from the registered path.
func URI(c *gin.Context) {
	var input PathInput
	if err := c.ShouldBindUri(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 绑定请求头中的文本值。

// Bind text values from request headers.
func Header(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindHeader(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 显式选择只读取请求体的 URL 编码表单绑定器。

// Explicitly select the URL-encoded form binder that reads only the request body.
func FormPost(c *gin.Context) {
	var input Input
	if err := c.ShouldBindWith(&input, binding.FormPost); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 使用 multipart 绑定器并按原有控制流处理缺少文件。

// Use the multipart binder and handle missing files through ordinary control flow.
func Multipart(c *gin.Context) {
	var input UploadInput
	if err := c.ShouldBindWith(&input, binding.FormMultipart); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	if input.File == nil {
		c.JSON(400, gin.H{"error": "file required"})
		return
	}
	c.JSON(200, gin.H{"Title": input.Title, "Filename": input.File.Filename, "Count": len(input.Files)})
}

// 显式 JSON 绑定器与快捷方法共用 JSON 投影。

// Explicit JSON binders share JSON projection with convenience methods.
func JSON(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindWith(&input, binding.JSON); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 缓存请求体的显式 JSON 绑定器仍保留其真实媒体类型。

// An explicit cached-body JSON binder retains its real media type.
func BodyJSON(c *gin.Context) {
	var input HeaderInput
	if err := c.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, input)
}

// 非 JSON 显式绑定器不能套用 JSON 编码规则。

// Explicit non-JSON binders must not inherit JSON encoding rules.
func XML(c *gin.Context) {
	var input HeaderInput
	_ = c.ShouldBindWith(&input, binding.XML)
	c.JSON(200, input)
}

// 注册实际 Gin 路由，文档挂载不替代绑定逻辑。

// Register ordinary Gin routes without replacing binding logic through documentation mounting.
func Router() *gin.Engine {
	r := gin.New()
	r.GET("/query", Query)
	r.GET("/query-explicit", ExplicitQuery)
	r.GET("/uri/:ID", URI)
	r.GET("/header", Header)
	r.POST("/form", FormPost)
	r.POST("/multipart", Multipart)
	r.POST("/json", JSON)
	r.POST("/body-json", BodyJSON)
	r.POST("/xml", XML)
	r.GET("/embedded", Embedded)
	r.GET("/custom", Custom)
	r.GET("/repeated-header", RepeatedHeader)
	return r
}

// 普通匿名嵌入遵循 Gin 的递归字段遍历。

// Ordinary anonymous embedding follows Gin's recursive field traversal.
type EmbeddedFields struct {
	// 共享名称。

	// Shared name.
	Shared string
}

// 嵌入时间类型没有可绑定的导出子字段，不能凭空生成 Time 参数。

// Embedded time has no bindable exported children and must not invent a Time parameter.
type EmbeddedInput struct {
	EmbeddedFields
	time.Time
}

// 绑定真实嵌入字段而不是 JSON 的提升优先级。

// Bind actual embedded fields instead of JSON promotion precedence.
func Embedded(c *gin.Context) {
	var input EmbeddedInput
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}
	c.JSON(200, gin.H{"Shared": input.Shared})
}

// 自定义参数解码必须由集中映射表达。

// Custom parameter decoding must be described by a centralized mapping.
type CustomParam string

// 将输入转换成业务自定义值。

// Convert input into a custom business value.
func (value *CustomParam) UnmarshalParam(raw string) error {
	*value = CustomParam("custom:" + raw)
	return nil
}

// 包含自定义参数解码器的请求。

// A request containing a custom parameter decoder.
type CustomInput struct{ Value CustomParam }

// 未注册自定义 codec 时不能宣称完整掌握输入协议。

// Do not claim a complete input contract without a registered custom codec.
func Custom(c *gin.Context) { var input CustomInput; _ = c.ShouldBindQuery(&input); c.JSON(200, input) }

// 重复请求头不是逗号分隔数组。

// Repeated headers are not comma-separated arrays.
type RepeatedHeaders struct{ Values []string }

// 对无法表示的默认重复头序列化保留诊断。

// Preserve diagnostics for default repeated-header serialization that cannot be represented.
func RepeatedHeader(c *gin.Context) {
	var input RepeatedHeaders
	_ = c.ShouldBindHeader(&input)
	c.JSON(200, input)
}
