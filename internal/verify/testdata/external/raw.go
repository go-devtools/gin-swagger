// Verify raw form and file reads without adding business-input tags.
package main

import "github.com/gin-gonic/gin"

// Preserve actual read values without confusing missing fields with empty strings.
type FormResult struct {

	// Title.
	Title string

	// Default display name.
	Alias string

	// Email address.
	Email string

	// Whether the email field exists.
	Present bool

	// Repeated labels.
	Labels []string

	// Repeated values.
	Values []string

	// Whether repeated values exist.
	HasValues bool
}

// Read body form fields without treating query parameters as fallback values.
func Form(c *gin.Context) {
	const title = "title"
	email, present := c.GetPostForm("email")
	values, hasValues := c.GetPostFormArray("values")
	c.JSON(200, FormResult{Title: c.PostForm(title), Alias: c.DefaultPostForm("alias", "guest"), Email: email, Present: present, Labels: c.PostFormArray("labels"), Values: values, HasValues: hasValues})
}

// Return upload results without using Go file metadata as the upload request format.
type FileResult struct {

	// Original filename.
	Filename string

	// Uploaded byte count.
	Size int64

	// Form caption.
	Caption string
}

// Read the first file with the given name and handle errors in ordinary business branches.
func Upload(c *gin.Context) {
	file, err := c.FormFile("asset")
	if err != nil {
		c.String(400, "missing file")
		return
	}
	c.JSON(201, FileResult{Filename: file.Filename, Size: file.Size, Caption: c.PostForm("caption")})
}

// Read repeated query values and preserve the presence flag returned by GetQueryArray.
func Query(c *gin.Context) {
	values, ok := c.GetQueryArray("values")
	c.JSON(200, FormResult{Values: values, HasValues: ok})
}

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

// Preserve dictionary fields and presence flags for their respective sources.
type MapResult struct {

	// Form dictionary.
	Form map[string]string

	// Plain form dictionary.
	Bare map[string]string

	// Query dictionary.
	Query map[string]string

	// Plain query dictionary.
	BareQuery map[string]string

	// Whether the form dictionary exists.
	FormPresent bool

	// Whether the query dictionary exists.
	QueryPresent bool
}

// Read Gin bracket dictionaries while keeping body and query sources separate.
func Maps(c *gin.Context) {
	form, fp := c.GetPostFormMap("filter")
	query, qp := c.GetQueryMap("query")
	c.JSON(200, MapResult{Form: form, Bare: c.PostFormMap("bare"), Query: query, BareQuery: c.QueryMap("bareq"), FormPresent: fp, QueryPresent: qp})
}

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

// Dynamic field names lack complete contract evidence and should be diagnosed only for selected routes.
func Dynamic(c *gin.Context) { c.String(200, c.PostForm(c.Query("field"))) }

// Ignoring file errors can pass nil to saving and must not fabricate a normal upload contract.
func IgnoreFileError(c *gin.Context) {
	file, _ := c.FormFile("asset")
	if err := c.SaveUploadedFile(file, c.GetString("destination")); err != nil {
		c.String(500, "save failed")
		return
	}
	c.Status(204)
}
