package main

import "github.com/gin-gonic/gin"

// The complete resource returned by read, create, replace, and partial update examples.
type Item struct {
	// Resource identifier.
	// @openapi examples=["item-1"]
	ID string
	// Resource name.
	// @openapi examples=["Sample resource"]
	Name string
	// Whether the resource is enabled.
	// @openapi examples=[true,false]
	Enabled bool
}

// Fields submitted when creating or fully replacing a resource.
type ItemInput struct {
	// Resource name, containing at least one character.
	// @openapi required nonnull minLength=1 examples=["New resource"]
	Name string
	// Whether the resource is enabled; defaults to false when omitted.
	// @openapi default=false examples=[true,false]
	Enabled bool
}

// Only non-null fields overwrite the sample resource during a partial update.
type ItemPatch struct {
	// Optional name; omission or null preserves the current value.
	// @openapi examples=["Updated name",null]
	Name *string
	// Optional enabled flag; false explicitly updates the value, while null preserves it.
	// @openapi examples=[false,null]
	Enabled *bool
}

// Get a resource
//
// Demonstrates a GET path parameter and a complete response. The identifier missing returns 404; other identifiers return an in-memory sample.
// @openapi tags=["HTTP methods"]
func GetItem(c *gin.Context) {
	id := c.Param("id")
	if id == "missing" {
		c.JSON(404, APIError{Code: "NOT_FOUND", Message: "示例资源不存在"})
		return
	}
	c.JSON(200, Item{ID: id, Name: "示例资源", Enabled: true})
}

// Create a resource
//
// Demonstrates a POST JSON request with a 201 response. The example returns the created resource without persisting it.
// @openapi tags=["HTTP methods"]
func CreateItem(c *gin.Context) {
	var req ItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "请求体格式错误"})
		return
	}
	if req.Name == "" {
		c.JSON(400, APIError{Code: "INVALID_NAME", Message: "名称不能为空"})
		return
	}
	c.JSON(201, Item{ID: "item-1", Name: req.Name, Enabled: req.Enabled})
}

// Replace a resource
//
// Demonstrates a PUT path parameter and complete JSON input. Returns the replaced sample without persisting it.
// @openapi tags=["HTTP methods"]
func ReplaceItem(c *gin.Context) {
	var req ItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "请求体格式错误"})
		return
	}
	if req.Name == "" {
		c.JSON(400, APIError{Code: "INVALID_NAME", Message: "名称不能为空"})
		return
	}
	c.JSON(200, Item{ID: c.Param("id"), Name: req.Name, Enabled: req.Enabled})
}

// Partially update a resource
//
// Demonstrates optional pointer fields with PATCH. Only submitted non-null fields are updated; false is treated as an explicit value.
// @openapi tags=["HTTP methods"]
func PatchItem(c *gin.Context) {
	var req ItemPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, APIError{Code: "INVALID_JSON", Message: "请求体格式错误"})
		return
	}
	item := Item{ID: c.Param("id"), Name: "示例资源", Enabled: true}
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	c.JSON(200, item)
}

// Delete a resource
//
// Demonstrates DELETE with a 204 No Content response and no response body. No persistent data is deleted.
// @openapi tags=["HTTP methods"]
func DeleteItem(c *gin.Context) { c.Status(204) }

// Get a resource (deprecated)
//
// This endpoint is deprecated. Use GET /examples/items/{id} instead. The legacy route demonstrates the Deprecated label and compatibility documentation.
// @openapi tags=["Legacy"] deprecated
func LegacyGetItem(c *gin.Context) {
	c.JSON(200, Item{ID: c.Param("id"), Name: "旧版资源", Enabled: true})
}
