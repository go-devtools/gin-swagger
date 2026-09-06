package httpresponses

import "github.com/gin-gonic/gin"

// 使用根相对路径重定向，验证路径清理和查询串。

// Redirect to a root-relative path to exercise path cleaning and query preservation.
func RedirectRoot(c *gin.Context) { c.Redirect(302, "/a/../next?from=old") }

// 使用 Gin 明确接受的 201 重定向。

// Use the 201 redirect explicitly accepted by Gin.
func RedirectCreated(c *gin.Context) { c.Redirect(201, "/created") }

// 非 ASCII 目标在 Location 中按 UTF-8 字节转义。

// Escape UTF-8 bytes in non-ASCII Location targets.
func RedirectUnicode(c *gin.Context) { c.Redirect(307, "/\u76ee\u6807") }

// 相对目标依赖本次请求路径，不能伪造固定 Location。

// Relative destinations depend on the request path and must not invent a fixed Location.
func RedirectRelative(c *gin.Context) { c.Redirect(303, "next") }

// 既有媒体头禁止标准库自动写入 HTML 正文。

// An existing media header suppresses the standard library's automatic HTML body.
func RedirectCustom(c *gin.Context) { c.Header("Content-Type", "text/plain"); c.Redirect(302, "/next") }

// 重定向后继续运行，最终状态取决于此前是否真的提交过正文。

// Continue after redirect; the final status depends on whether a body was actually committed.
func RedirectContinue(c *gin.Context) { c.Redirect(302, "/next"); c.String(201, "done") }

// 已提交状态和头不因后续重定向被替换。

// A later redirect cannot replace an already committed status or headers.
func RedirectCommitted(c *gin.Context) { c.AbortWithStatus(409); c.Redirect(302, "/next") }

// 无效状态必须产生诊断而非假造默认响应。

// Invalid statuses must produce diagnostics rather than fabricated default responses.
func RedirectInvalid(c *gin.Context) { c.Redirect(299, "/next") }

// 动态目标仍具有可知的响应状态与 Location 字符串表示。

// A dynamic target still has a known status and a string Location representation.
func RedirectDynamic(c *gin.Context) { c.Redirect(302, c.Query("next")) }

// 同一业务函数在 GET 和 HEAD 路由中复用。

// Reuse one business function for GET and HEAD routes.
func HeadJSON(c *gin.Context) { c.Header("X-Revision", "1"); c.JSON(200, gin.H{"name": "Ada"}) }

// 绝对 URI 的现有百分号转义与路径由标准库原样保留。

// The standard library preserves existing escapes and paths in absolute URIs.
func RedirectAbsolute(c *gin.Context) { c.Redirect(308, "https://example.test/a/../next?q=%2F") }

// 304 的 Location 仍存在，但网络响应没有正文。

// A 304 keeps its Location while transmitting no response body.
func RedirectNotModified(c *gin.Context) { c.Redirect(304, "/cached") }

// 删除媒体头后，重定向重新采用标准 HTML 行为。

// Removing the media header restores the standard redirect HTML behavior.
func RedirectDeletedMedia(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.Header("Content-Type", "")
	c.Redirect(301, "/next/")
}

// 提交后的媒体头不会进入已提交头，却仍能禁止后续重定向正文。

// A post-commit media header is not transmitted but still suppresses a later redirect body.
func RedirectLateMedia(c *gin.Context) {
	c.AbortWithStatus(409)
	c.Header("Content-Type", "text/plain")
	c.Redirect(302, "/next")
}

// HTTP 序列化清除头值两端的空白，文档必须使用网络上的实际值。

// HTTP serialization trims surrounding header whitespace; document the actual wire value.
func RedirectWhitespace(c *gin.Context) { c.Redirect(302, " https://example.test/next ") }
