package main

import "github.com/gin-gonic/gin"

// 返回显式声明的无响应体响应。

// Return an explicitly documented bodyless response.
// @openapi response status=204
func DeclaredBodyless(c *gin.Context) { c.Status(204) }

// 即使存在显式回退契约，也保留未知状态诊断。

// Preserve unknown status diagnostics even with an explicit fallback contract.
// @openapi response status="default" mediaType="application/json" type="Request"
func DeclaredUnknown(c *gin.Context) { c.JSON(c.GetInt("status"), Request{Name: "unknown"}) }

// 拒绝与实际输出 Go 类型冲突的声明。

// Reject a declaration that contradicts the actual emitted Go type.
// @openapi response status=201 mediaType="application/json" type="string"
func DeclaredConflict(c *gin.Context) { c.JSON(201, Request{Name: "actual"}) }
