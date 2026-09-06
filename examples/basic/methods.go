package main

import "github.com/gin-gonic/gin"

// 用于展示资源读取、创建、替换与局部更新后的完整响应。

// The complete resource returned by read, create, replace, and partial update examples.
type Item struct {

	// 资源标识。

	// Resource identifier.
	// @openapi examples=["item-1"]
	ID string

	// 资源名称。

	// Resource name.
	// @openapi examples=["Sample resource"]
	Name string

	// 是否启用。

	// Whether the resource is enabled.
	// @openapi examples=[true,false]
	Enabled bool
}

// 创建和完整替换时提交的字段。

// Fields submitted when creating or fully replacing a resource.
type ItemInput struct {

	// 资源名称，至少包含一个字符。

	// Resource name, containing at least one character.
	// @openapi required nonnull minLength=1 examples=["New resource"]
	Name string

	// 是否启用，省略时采用 false。

	// Whether the resource is enabled; defaults to false when omitted.
	// @openapi default=false examples=[true,false]
	Enabled bool
}

// 局部更新中只有非 null 的字段会覆盖示例资源。

// Only non-null fields overwrite the sample resource during a partial update.
type ItemPatch struct {

	// 可选名称；缺失或 null 均保持原值。

	// Optional name; omission or null preserves the current value.
	// @openapi examples=["Updated name",null]
	Name *string

	// 可选开关；false 是明确更新，null 保持原值。

	// Optional enabled flag; false explicitly updates the value, while null preserves it.
	// @openapi examples=[false,null]
	Enabled *bool
}

// 读取资源
// GET 演示路径参数和完整响应。资源标识为 missing 时演示 404；其他标识返回即时构造的样本。

// Get a resource
//
// Demonstrates a GET path parameter and a complete response. The identifier missing returns 404; other identifiers return an in-memory sample.
// @openapi tags=["HTTP methods"]
func GetItem(c *gin.Context) {
	id := c.Param("id")
	if id == "missing" {
		c.JSON(404, APIError{Code: "NOT_FOUND", Message: "Example resource does not exist"})
		return
	}
	c.JSON(200, Item{ID: id, Name: "Example resource", Enabled: true})
}

// 创建资源
// POST 演示 JSON 请求与 201 响应。此示例仅返回创建结果，不保存数据。

// Create a resource
//
// Demonstrates a POST JSON request with a 201 response. The example returns the created resource without persisting it.
// @openapi tags=["HTTP methods"]
func CreateItem(c *gin.Context) {
	var req ItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "Invalid request body format"})
		return
	}
	if req.Name == "" {
		c.JSON(400, APIError{Code: "INVALID_NAME", Message: "Name must not be empty"})
		return
	}
	c.JSON(201, Item{ID: "item-1", Name: req.Name, Enabled: req.Enabled})
}

// 完整替换资源
// PUT 演示路径参数与完整 JSON 输入，返回替换后的样本，不保存数据。

// Replace a resource
//
// Demonstrates a PUT path parameter and complete JSON input. Returns the replaced sample without persisting it.
// @openapi tags=["HTTP methods"]
func ReplaceItem(c *gin.Context) {
	var req ItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "Invalid request body format"})
		return
	}
	if req.Name == "" {
		c.JSON(400, APIError{Code: "INVALID_NAME", Message: "Name must not be empty"})
		return
	}
	c.JSON(200, Item{ID: c.Param("id"), Name: req.Name, Enabled: req.Enabled})
}

// 局部更新资源
// PATCH 演示可选指针字段；只更新提交的非 null 字段，false 不会被当作缺失。

// Partially update a resource
//
// Demonstrates optional pointer fields with PATCH. Only submitted non-null fields are updated; false is treated as an explicit value.
// @openapi tags=["HTTP methods"]
func PatchItem(c *gin.Context) {
	var req ItemPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "Invalid request body format"})
		return
	}
	item := Item{ID: c.Param("id"), Name: "Example resource", Enabled: true}
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	c.JSON(200, item)
}

// 删除资源
// DELETE 演示 204 No Content，不返回响应体；此示例不执行持久化删除。

// Delete a resource
//
// Demonstrates DELETE with a 204 No Content response and no response body. No persistent data is deleted.
// @openapi tags=["HTTP methods"]
func DeleteItem(c *gin.Context) { c.Status(204) }

// 旧版资源读取
// 此接口已弃用，请改用 GET /examples/items/{id}。保留此路由用于展示 Deprecated 样式与兼容文档。

// Get a resource (deprecated)
//
// This endpoint is deprecated. Use GET /examples/items/{id} instead. The legacy route demonstrates the Deprecated label and compatibility documentation.
// @openapi tags=["Legacy"] deprecated
func LegacyGetItem(c *gin.Context) {
	c.JSON(200, Item{ID: c.Param("id"), Name: "Legacy resource", Enabled: true})
}
