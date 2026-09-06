package main

import "github.com/gin-gonic/gin"

// 独立安装后的强制绑定失败仍继续写入当前 handler 的响应。

// A mandatory binding failure still allows the current handler to write after independent installation.
func Mandatory(c *gin.Context) {
	var request Request
	if c.BindJSON(&request) != nil {
		c.JSON(422, Request{Name: "invalid"})
		return
	}
	c.JSON(201, request)
}
