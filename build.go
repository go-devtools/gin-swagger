// Build and mount documentation while preserving existing Gin handlers and routes.
// 在保持既有 Gin handler 与路由注册不变的前提下构建和挂载文档。
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

// Separate core document settings from Gin scope and mounting options.
// 将核心文档设置与 Gin 挂载和作用域设置分开。
type Config struct {
	// Declare default request media and overrides keyed by original METHOD /Gin/path for documentation linking only.
	// 默认请求媒体范围及按原始 METHOD /Gin/path 覆盖的声明，仅用于文档链接。
	DefaultRequestMediaTypes []string
	RequestMediaTypes        map[string][]string
	OpenAPI                  openapi.Config
	Path                     string
	UI                       swaggerui.Config
	Middlewares              []gin.HandlerFunc
	Bindings                 map[string]openapi.OperationKey
	Include                  func(method, path string) bool
	// Optionally group complete documents, always intersecting their scope with global Include.
	// 可选的整份文档分类，范围始终与全局 Include 求交集。
	Groups       []DocumentGroup
	DefaultGroup string
}

// Describe one top-right document choice without changing Gin business route registration.
// 描述右上角文档选择器的一个分类，不改变 Gin 的业务路由注册。
type DocumentGroup struct {
	ID      string
	Name    string
	Include func(method, path string) bool
}

// Match a snapshot of registered routes against public template evidence without registering routes.
// 对真实已注册路由快照并匹配公开模板证据；不注册任何路由。
func Build(r *gin.Engine, bundle openapi.Bundle, cfg Config) (*openapi.Document, error) {
	if r == nil {
		return nil, fmt.Errorf("gin-swagger.engine.nil: Engine is required")
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
				return nil, openapi.Report{Diagnostics: []openapi.Diagnostic{{Code: "gin-swagger.handler.ambiguous", Severity: openapi.Error, Message: "cannot uniquely match the final handler: " + route.Handler, Route: routeName, Fix: "Add a centralized Config.Bindings entry for a closure or receiver whose identity cannot be resolved statically"}}}
			}
			key = matches[0]
		}
		neutral := openapi.Route{Method: route.Method, Path: normalized.Path, OperationKey: key, Source: openapi.Source{Kind: "derived", Rule: "gin.Engine.Routes", Symbol: route.Handler}}
		neutral.RequestMediaTypes = append([]string(nil), cfg.DefaultRequestMediaTypes...)
		if media, ok := cfg.RequestMediaTypes[routeName]; ok {
			neutral.RequestMediaTypes = append([]string(nil), media...)
		}
		if normalized.CatchAll {
			neutral.Extensions = spec.Extensions{"x-gin-catch-all": []byte(`true`), "x-gin-catch-all-note": []byte(`"The Gin catch-all may span slashes; OpenAPI path parameters and generated clients may not preserve this behavior."`)}
		}
		selected = append(selected, neutral)
	}
	// Gin documents bind the current Engine, so always verify observable executable build conditions.
	// Gin 文档绑定当前 Engine，始终验证可读取的程序构建条件。
	cfg.OpenAPI.VerifyRuntimeBuild = true
	return openapi.Build(bundle, selected, cfg.OpenAPI)
}

// Validate the documentation prefix without changing business route prefixes.
// 检查用于单次文档挂载的前缀，不修改业务路由前缀。
func mountPath(value string) (string, error) {
	if value == "" {
		value = "/docs"
	}
	value = strings.TrimSuffix(value, "/")
	if value == "" || value[0] != '/' || strings.ContainsAny(value, "\\:*{}?#%") || strings.Contains(value, "//") {
		return "", fmt.Errorf("gin-swagger.mount.path: invalid documentation prefix")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("gin-swagger.mount.path: directory traversal is not allowed")
		}
	}
	return value, nil
}
