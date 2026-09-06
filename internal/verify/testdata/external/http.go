package main

import "github.com/gin-gonic/gin"

// Redirect to a root-relative path to exercise path cleaning and query preservation.
func RedirectRoot(c *gin.Context) { c.Redirect(302, "/a/../next?from=old") }

// Use the 201 redirect explicitly accepted by Gin.
func RedirectCreated(c *gin.Context) { c.Redirect(201, "/created") }

// Escape UTF-8 bytes in non-ASCII Location targets.
func RedirectUnicode(c *gin.Context) { c.Redirect(307, "/\u76ee\u6807") }

// Relative destinations depend on the request path and must not invent a fixed Location.
func RedirectRelative(c *gin.Context) { c.Redirect(303, "next") }

// An existing media header suppresses the standard library's automatic HTML body.
func RedirectCustom(c *gin.Context) { c.Header("Content-Type", "text/plain"); c.Redirect(302, "/next") }

// Continue after redirect; the final status depends on whether a body was actually committed.
func RedirectContinue(c *gin.Context) { c.Redirect(302, "/next"); c.String(201, "done") }

// A later redirect cannot replace an already committed status or headers.
func RedirectCommitted(c *gin.Context) { c.AbortWithStatus(409); c.Redirect(302, "/next") }

// Invalid statuses must produce diagnostics rather than fabricated default responses.
func RedirectInvalid(c *gin.Context) { c.Redirect(299, "/next") }

// A dynamic target still has a known status and a string Location representation.
func RedirectDynamic(c *gin.Context) { c.Redirect(302, c.Query("next")) }

// Reuse one business function for GET and HEAD routes.
func HeadJSON(c *gin.Context) { c.Header("X-Revision", "1"); c.JSON(200, gin.H{"name": "Ada"}) }

// The standard library preserves existing escapes and paths in absolute URIs.
func RedirectAbsolute(c *gin.Context) { c.Redirect(308, "https://example.test/a/../next?q=%2F") }

// A 304 keeps its Location while transmitting no response body.
func RedirectNotModified(c *gin.Context) { c.Redirect(304, "/cached") }

// Removing the media header restores the standard redirect HTML behavior.
func RedirectDeletedMedia(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Type", "")
	c.Redirect(301, "/next/")
}

// A post-commit media header is not transmitted but still suppresses a later redirect body.
func RedirectLateMedia(c *gin.Context) {
	c.AbortWithStatus(409)
	c.Header("Content-Type", "text/plain")
	c.Redirect(302, "/next")
}

// HTTP serialization trims surrounding header whitespace; document the actual wire value.
func RedirectWhitespace(c *gin.Context) { c.Redirect(302, " https://example.test/next ") }
