// 在保持既有 Gin handler 与路由注册不变的前提下构建和挂载文档。
// Build and mount documentation while preserving existing Gin handlers and routes.
package ginswagger

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/gin-swagger/internal/routes"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/spec"
	"github.com/openapi-golang/openapi/swaggerui"
)

// 将核心文档设置与 Gin 挂载和作用域设置分开。
// Separate core document settings from Gin scope and mounting options.
type Config struct {
	OpenAPI     openapi.Config
	Path        string
	UI          swaggerui.Config
	Middlewares []gin.HandlerFunc
	Bindings    map[string]openapi.OperationKey
	Include     func(method, path string) bool
	// 可选的整份文档分类，范围始终与全局 Include 求交集。
	// Optionally group complete documents, always intersecting their scope with global Include.
	Groups       []DocumentGroup
	DefaultGroup string
}

// 描述右上角文档选择器的一个分类，不改变 Gin 的业务路由注册。
// Describe one top-right document choice without changing Gin business route registration.
type DocumentGroup struct {
	ID      string
	Name    string
	Include func(method, path string) bool
}

// 对真实已注册路由快照并匹配公开模板证据；不注册任何路由。
// Match a snapshot of registered routes against public template evidence without registering routes.
func Build(r *gin.Engine, bundle openapi.Bundle, cfg Config) (*openapi.Document, error) {
	if r == nil {
		return nil, fmt.Errorf("gin-swagger.engine.nil: 缺少 Engine")
	}
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	index := bundle.Index()
	symbols := map[string][]openapi.OperationKey{}
	for _, template := range index {
		for _, symbol := range template.RuntimeSymbols {
			symbols[symbol] = append(symbols[symbol], template.Key)
		}
	}
	selected := []openapi.Route{}
	for _, route := range r.Routes() {
		if cfg.Include != nil && !cfg.Include(route.Method, route.Path) {
			continue
		}
		normalized, err := routes.Parse(route.Path)
		if err != nil {
			return nil, err
		}
		routeName := route.Method + " " + route.Path
		key, bound := cfg.Bindings[routeName]
		if !bound {
			matches := symbols[route.Handler]
			if len(matches) != 1 {
				return nil, openapi.Report{Diagnostics: []openapi.Diagnostic{{Code: "gin-swagger.handler.ambiguous", Severity: openapi.Error, Message: "无法唯一匹配末位 handler：" + route.Handler, Route: routeName, Fix: "为无法静态消歧的闭包或 receiver 添加一次集中 Config.Bindings"}}}
			}
			key = matches[0]
		}
		neutral := openapi.Route{Method: route.Method, Path: normalized.Path, OperationKey: key, Source: openapi.Source{Kind: "derived", Rule: "gin.Engine.Routes", Symbol: route.Handler}}
		if normalized.CatchAll {
			neutral.Extensions = spec.Extensions{"x-gin-catch-all": []byte(`true`), "x-gin-catch-all-note": []byte(`"The Gin catch-all may span slashes; OpenAPI path parameters and generated clients may not preserve this behavior."`)}
		}
		selected = append(selected, neutral)
	}
	return openapi.Build(bundle, selected, cfg.OpenAPI)
}

// 检查用于单次文档挂载的前缀，不修改业务路由前缀。
// Validate the documentation prefix without changing business route prefixes.
func mountPath(value string) (string, error) {
	if value == "" {
		value = "/docs"
	}
	value = strings.TrimSuffix(value, "/")
	if value == "" || value[0] != '/' || strings.ContainsAny(value, "\\:*{}?#%") || strings.Contains(value, "//") {
		return "", fmt.Errorf("gin-swagger.mount.path: 非法文档前缀")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("gin-swagger.mount.path: 不允许目录跳转")
		}
	}
	return value, nil
}
