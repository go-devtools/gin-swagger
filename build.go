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

// Separate core document settings from Gin scope and mounting options.
type Config struct {
	// Declare default request media and overrides keyed by original METHOD /Gin/path for documentation linking only.
	DefaultRequestMediaTypes []string
	RequestMediaTypes        map[string][]string
	OpenAPI                  openapi.Config
	Path                     string
	UI                       swaggerui.Config
	Middlewares              []gin.HandlerFunc
	Bindings                 map[string]openapi.OperationKey
	Include                  func(method, path string) bool
	// Optionally group complete documents, always intersecting their scope with global Include.
	Groups       []DocumentGroup
	DefaultGroup string
}

// Describe one top-right document choice without changing Gin business route registration.
type DocumentGroup struct {
	ID      string
	Name    string
	Include func(method, path string) bool
}

// Match a snapshot of registered routes against public template evidence without registering routes.
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
	cfg.OpenAPI.VerifyRuntimeBuild = true
	return openapi.Build(bundle, selected, cfg.OpenAPI)
}

// Validate the documentation prefix without changing business route prefixes.
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
