package main

import (
	"example.test/gin-consumer/testdata/contracts"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 按真实 JSON 绑定方式读取导入 DTO。

// Bind an imported DTO through the actual JSON codec.
func ImportedJSON(c *gin.Context) {
	var input contracts.Request
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.Header("X-Source", "imported")
	c.JSON(200, contracts.Envelope[contracts.Request]{Data: input})
}

// Query 使用其真实文本绑定规则。

// Bind imported fields through the actual query codec.
func ImportedQuery(c *gin.Context) {
	var input contracts.Request
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, contracts.Envelope[contracts.Request]{Data: input})
}

// 表单使用显式 FormPost 绑定器。

// Bind imported fields through the explicit FormPost codec.
func ImportedForm(c *gin.Context) {
	var input contracts.Request
	if err := c.ShouldBindWith(&input, binding.FormPost); err != nil {
		c.JSON(400, Request{Name: "invalid"})
		return
	}
	c.JSON(200, contracts.Envelope[contracts.Request]{Data: input})
}

// 仅选中对应路由时报告导入元数据语法错误。

// Selecting this route exposes the imported metadata syntax error.
func ImportedMalformed(c *gin.Context) {
	var input contracts.Malformed
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Status(400)
		return
	}
	c.Status(204)
}

// 参数对象投影也需要保留原始字段诊断。

// Parameter projection must retain the original field's semantic diagnostic.
func ImportedIncompatible(c *gin.Context) {
	var input contracts.Incompatible
	if err := c.ShouldBindQuery(&input); err != nil {
		c.Status(400)
		return
	}
	c.Status(204)
}
