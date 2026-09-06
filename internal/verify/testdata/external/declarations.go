package main

import "github.com/gin-gonic/gin"

// Return an explicitly documented bodyless response.
// @openapi response status=204
func DeclaredBodyless(c *gin.Context) { c.Status(204) }

// Preserve unknown status diagnostics even with an explicit fallback contract.
// @openapi response status="default" mediaType="application/json" type="Request"
func DeclaredUnknown(c *gin.Context) { c.JSON(c.GetInt("status"), Request{Name: "unknown"}) }

// Reject a declaration that contradicts the actual emitted Go type.
// @openapi response status=201 mediaType="application/json" type="string"
func DeclaredConflict(c *gin.Context) { c.JSON(201, Request{Name: "actual"}) }
