// Exercise generated Gin contracts through the shared offline browser resources.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/gin-swagger/internal/verify/testdata/browser/internal/apidoc"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/spec"
	"github.com/openapi-golang/openapi/swaggerui"
)

// Supported roles.
// @openapi enum
type Role string

// Declare allowed roles and their descriptions.
const (

	// Administrator
	RoleAdmin Role = "admin"

	// Editor
	RoleEditor Role = "editor"
)

// Submitted role information.
type Request struct {

	// Selected role.
	// @openapi required examples=["admin"]
	Role Role
}

// A public request error.
type APIError struct{ Message string }

// Submit a role.
// @openapi tags=["Browser"]
func Submit(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, APIError{Message: "Invalid JSON"})
		return
	}
	c.JSON(200, request)
}

// Start only the test-owned service, preserving the ordinary registered handler.
func main() {
	gin.SetMode(gin.ReleaseMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	engine := gin.New()
	engine.POST("/submit", Submit)
	for _, mode := range []string{"safe", "enabled", "metadata"} {
		cfg := ginswagger.Config{Path: "/" + mode + "/docs", Include: func(method, path string) bool { return path == "/submit" }, OpenAPI: openapi.Config{Title: "Browser API", Version: "1", Configure: func(doc *spec.OpenAPI) error {
			doc.Components.SecuritySchemes = map[string]spec.RefOr[spec.SecurityScheme]{"BearerAuth": spec.Inline(spec.SecurityScheme{Type: "http", Scheme: "bearer"})}
			// Declare native tag metadata centrally without changing the registered handler or its source contract.
			if mode == "metadata" {
				doc.Tags = []spec.Tag{{Name: "Platform", Summary: "Platform services", Kind: "nav"}, {Name: "Browser", Summary: "Browser examples", Parent: spec.Set("Platform"), Kind: "nav"}}
			}
			doc.Security = spec.Set([]spec.SecurityRequirement{{"BearerAuth": {}}})
			return nil
		}}, UI: swaggerui.Config{Title: "Browser contract", DocExpansion: "full"}, Groups: []ginswagger.DocumentGroup{{ID: "all", Name: "All endpoints"}, {ID: "reference", Name: "Reference"}}, DefaultGroup: "all"}
		if mode == "enabled" {
			cfg.UI.SubmitMethods = []string{"post"}
		}
		if _, err := ginswagger.Mount(engine, apidoc.Bundle(), cfg); err != nil {
			panic(err)
		}
	}
	var guard sync.Mutex
	state := map[string]any{"requests": 0, "authorization": ""}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/state" {
			guard.Lock()
			defer guard.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(state)
			return
		}
		if r.URL.Path == "/submit" && r.Method == "POST" {
			guard.Lock()
			state["requests"] = state["requests"].(int) + 1
			state["authorization"] = r.Header.Get("Authorization")
			guard.Unlock()
		}
		engine.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Printf("{\"url\":\"http://%s\"}\n", listener.Addr())
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
