package ginswagger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/swaggerui"
)

// Prepare and validate everything before mounting during application startup.
// 完成文档、UI 和冲突检查后才挂载；仅在全部业务路由注册后、服务启动前调用。
func Mount(r *gin.Engine, bundle openapi.Bundle, cfg Config) (*openapi.Document, error) {
	base, err := mountPath(cfg.Path)
	if err != nil {
		return nil, err
	}
	doc, err := Build(r, bundle, cfg)
	if err != nil {
		return nil, err
	}
	documents, uiConfig, err := groupDocuments(r, bundle, doc, cfg)
	if err != nil {
		return nil, err
	}
	ui, err := swaggerui.New(uiConfig)
	if err != nil {
		return nil, err
	}
	route := base + "/*asset"
	if err = preflight(r, route, cfg.Middlewares); err != nil {
		return nil, err
	}
	// Serve cached documentation without invoking analysis or business handlers.
	// 文档请求只读取已构建缓存，不执行分析器或业务 handler。
	handler := func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("asset"), "/")
		if document, ok := documents[name]; ok {
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("Cache-Control", "no-cache")
			c.Header("ETag", document.etag)
			if c.GetHeader("If-None-Match") == document.etag {
				c.Status(http.StatusNotModified)
				return
			}
			if c.Request.Method == "HEAD" {
				c.Status(http.StatusOK)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", document.raw)
			return
		}
		resource, err := ui.Resource(name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		headers := resource.Headers()
		for k, v := range headers {
			c.Header(k, v)
		}
		if c.GetHeader("If-None-Match") == headers["ETag"] {
			c.Status(http.StatusNotModified)
			return
		}
		if c.Request.Method == "HEAD" {
			c.Status(http.StatusOK)
			return
		}
		c.Data(http.StatusOK, resource.ContentType(), resource.Bytes())
	}
	chain := append(append([]gin.HandlerFunc(nil), cfg.Middlewares...), handler)
	r.GET(route, chain...)
	r.HEAD(route, chain...)
	return doc, nil
}

// Replay public routes on an isolated Engine; only the preflight instance can panic.
// 在隔离 Engine 上用公开 API 重放路径，panic 仅来自预检查实例，无需回滚目标 Engine。
func preflight(r *gin.Engine, path string, middlewares []gin.HandlerFunc) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("gin-swagger.mount.conflict: documentation mount preflight failed: %v", failure)
		}
	}()
	shadow := gin.New()
	shadow.Use(r.Handlers...)
	noop := func(*gin.Context) {}
	for _, route := range r.Routes() {
		shadow.Handle(route.Method, route.Path, noop)
	}
	chain := append(append([]gin.HandlerFunc(nil), middlewares...), noop)
	shadow.GET(path, chain...)
	shadow.HEAD(path, chain...)
	return nil
}

// Store only startup-built bytes and content identifiers for each document.
// 每份文档只保存启动时构建的字节和内容标识。
type cachedDocument struct {
	raw  []byte
	etag string
}

// Build every group before mounting routes; keep the default overview at openapi.json.
// 在任何挂载修改之前完成全部分类构建，默认总览文档仍位于 openapi.json。
func groupDocuments(r *gin.Engine, bundle openapi.Bundle, main *openapi.Document, cfg Config) (map[string]cachedDocument, swaggerui.Config, error) {
	documents := map[string]cachedDocument{}
	cache := func(name string, doc *openapi.Document) {
		raw := doc.JSON()
		sum := sha256.Sum256(raw)
		documents[name] = cachedDocument{raw: raw, etag: "\"" + hex.EncodeToString(sum[:]) + "\""}
	}
	cache("openapi.json", main)
	uiConfig := cfg.UI
	if len(cfg.Groups) == 0 {
		return documents, uiConfig, nil
	}
	if len(uiConfig.Definitions) != 0 {
		return nil, uiConfig, fmt.Errorf("gin-swagger.groups.config: Groups and UI.Definitions cannot be configured together")
	}
	ids, names := map[string]bool{}, map[string]bool{}
	for _, group := range cfg.Groups {
		valid := group.ID != "" && group.Name != "" && !ids[group.ID] && !names[group.Name]
		for _, char := range group.ID {
			valid = valid && ((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-')
		}
		if !valid {
			return nil, uiConfig, fmt.Errorf("gin-swagger.groups.invalid: group IDs must be unique lowercase letters, digits, or hyphens; display names must also be unique")
		}
		ids[group.ID], names[group.Name] = true, true
	}
	if cfg.DefaultGroup != "" && !ids[cfg.DefaultGroup] {
		return nil, uiConfig, fmt.Errorf("gin-swagger.groups.default: default group does not exist")
	}
	for _, group := range cfg.Groups {
		grouped := cfg
		grouped.Groups = nil
		grouped.OpenAPI.Title = cfg.OpenAPI.Title + " · " + group.Name
		grouped.Include = func(method, path string) bool {
			return (cfg.Include == nil || cfg.Include(method, path)) && (group.Include == nil || group.Include(method, path))
		}
		doc, err := Build(r, bundle, grouped)
		if err != nil {
			return nil, uiConfig, fmt.Errorf("gin-swagger.groups.build: %s: %w", group.ID, err)
		}
		name := "groups/" + group.ID + ".json"
		cache(name, doc)
		uiConfig.Definitions = append(uiConfig.Definitions, swaggerui.Definition{Name: group.Name, URL: "./" + name})
		if group.ID == cfg.DefaultGroup {
			uiConfig.PrimaryDefinition = group.Name
		}
	}
	return documents, uiConfig, nil
}
