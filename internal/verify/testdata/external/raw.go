// 验证原始表单和文件读取，业务输入不添加 tag。
// Verify raw form and file reads without adding business-input tags.
package main

import "github.com/gin-gonic/gin"

// 保存真实读取值，不把字段缺失与空字符串混淆。
// Preserve actual read values without confusing missing fields with empty strings.
type FormResult struct {
	// 标题。 Title.
	Title string
	// 默认显示名。 Default display name.
	Alias string
	// 邮件地址。 Email address.
	Email string
	// 邮件字段是否存在。 Whether the email field exists.
	Present bool
	// 重复标签。 Repeated labels.
	Labels []string
	// 重复值。 Repeated values.
	Values []string
	// 重复值是否存在。 Whether repeated values exist.
	HasValues bool
}

// 按正文读取表单；查询参数不作为表单字段的兜底。
// Read body form fields without treating query parameters as fallback values.
func Form(c *gin.Context) {
	const title = "title"
	email, present := c.GetPostForm("email")
	values, hasValues := c.GetPostFormArray("values")
	c.JSON(200, FormResult{Title: c.PostForm(title), Alias: c.DefaultPostForm("alias", "guest"), Email: email, Present: present, Labels: c.PostFormArray("labels"), Values: values, HasValues: hasValues})
}

// 保存上传结果，不把 Go 文件元数据当作上传请求格式。
// Return upload results without using Go file metadata as the upload request format.
type FileResult struct {
	// 原始文件名。 Original filename.
	Filename string
	// 上传字节数。 Uploaded byte count.
	Size int64
	// 表单说明。 Form caption.
	Caption string
}

// 读取第一个同名文件，错误交给原有分支处理。
// Read the first file with the given name and handle errors in ordinary business branches.
func Upload(c *gin.Context) {
	file, err := c.FormFile("asset")
	if err != nil {
		c.String(400, "missing file")
		return
	}
	c.JSON(201, FileResult{Filename: file.Filename, Size: file.Size, Caption: c.PostForm("caption")})
}

// 读取重复查询值，保留 GetQueryArray 的存在标志。
// Read repeated query values and preserve the presence flag returned by GetQueryArray.
func Query(c *gin.Context) {
	values, ok := c.GetQueryArray("values")
	c.JSON(200, FormResult{Values: values, HasValues: ok})
}

// 注册独立验收路由，不修改基本示例或现有业务路由。
// Register independent acceptance routes without changing the basic example or existing business routes.
func Router() *gin.Engine {
	r := gin.New()
	r.POST("/form", Form)
	r.GET("/form", Form)
	r.PUT("/form", Form)
	r.PATCH("/form", Form)
	r.DELETE("/form", Form)
	r.POST("/upload", Upload)
	r.GET("/query", Query)
	return r
}

// 保留字典字段及各来源的存在标志。
// Preserve dictionary fields and presence flags for their respective sources.
type MapResult struct {
	// 表单字典。 Form dictionary.
	Form map[string]string
	// 普通表单字典。 Plain form dictionary.
	Bare map[string]string
	// 查询字典。 Query dictionary.
	Query map[string]string
	// 普通查询字典。 Plain query dictionary.
	BareQuery map[string]string
	// 表单字典是否存在。 Whether the form dictionary exists.
	FormPresent bool
	// 查询字典是否存在。 Whether the query dictionary exists.
	QueryPresent bool
}

// 读取 Gin 方括号字典，表单与查询来源保持分离。
// Read Gin bracket dictionaries while keeping body and query sources separate.
func Maps(c *gin.Context) {
	form, fp := c.GetPostFormMap("filter")
	query, qp := c.GetQueryMap("query")
	c.JSON(200, MapResult{Form: form, Bare: c.PostFormMap("bare"), Query: query, BareQuery: c.QueryMap("bareq"), FormPresent: fp, QueryPresent: qp})
}

// 保存上传文件；目标由现有应用上下文提供，不从请求参数读取路径。
// Save an upload to a destination provided by application context rather than request parameters.
func Save(c *gin.Context) {
	file, err := c.FormFile("asset")
	if err != nil {
		c.String(400, "missing file")
		return
	}
	if err = c.SaveUploadedFile(file, c.GetString("destination")); err != nil {
		c.String(500, "save failed")
		return
	}
	c.Status(204)
}

// 动态字段名没有完整契约证据，应仅在该路由被选中时诊断。
// Dynamic field names lack complete contract evidence and should be diagnosed only for selected routes.
func Dynamic(c *gin.Context) { c.String(200, c.PostForm(c.Query("field"))) }

// 忽略文件错误后可能向保存函数传递 nil，不能伪造正常上传契约。
// Ignoring file errors can pass nil to saving and must not fabricate a normal upload contract.
func IgnoreFileError(c *gin.Context) {
	file, _ := c.FormFile("asset")
	if err := c.SaveUploadedFile(file, c.GetString("destination")); err != nil {
		c.String(500, "save failed")
		return
	}
	c.Status(204)
}
