package compiler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	gincompiler "github.com/go-devtools/gin-swagger/compiler"
	"github.com/go-devtools/gin-swagger/internal/benchfixture"
	"github.com/go-devtools/openapi"
	core "github.com/go-devtools/openapi/compiler"
)

// Retain read results so benchmarks measure the public defensive-copy allocation.
var scaleJSON []byte

// Prepare distinct actual Gin handlers without changing repository source or module files.
func scaleProject(b *testing.B, count int) core.Options {
	b.Helper()
	dir := b.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		raw, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			b.Fatal(err)
		}
		if name == "go.mod" {
			raw = bytes.Replace(raw, []byte("module github.com/go-devtools/gin-swagger"), []byte("module example.test/gin-scale"), 1)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			b.Fatal(err)
		}
	}
	fixture, err := os.ReadFile(filepath.Join("..", "internal", "benchfixture", "handler.go"))
	if err != nil {
		b.Fatal(err)
	}
	start := bytes.Index(fixture, []byte("// Read and write JSON through ordinary Gin calls."))
	if start < 0 {
		b.Fatal("compiled benchmark handler is missing")
	}
	var source bytes.Buffer
	source.Write(fixture[:start])
	for i := 0; i < count; i++ {
		source.Write(bytes.Replace(fixture[start:], []byte("func Handle("), []byte(fmt.Sprintf("func Handle%04d(", i)), 1))
		source.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "api.go"), source.Bytes(), 0600); err != nil {
		b.Fatal(err)
	}
	return core.Options{Load: core.LoadOptions{Dir: dir, Env: []string{"GOWORK=off", "GOPROXY=off"}}, Frontends: []core.Frontend{gincompiler.Frontend()}}
}

// Compare automatic Gin linking with core Build over the same real route snapshot and Bundle.
func BenchmarkGinScale(b *testing.B) {
	previousMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	b.Cleanup(func() { gin.SetMode(previousMode) })
	runtimeOptions := core.Options{Load: core.LoadOptions{Dir: filepath.Join("..", "internal", "benchfixture"), Env: []string{"GOWORK=off", "GOPROXY=off"}}, Frontends: []core.Frontend{gincompiler.Frontend()}}
	runtimeResult, err := core.Compile(context.Background(), runtimeOptions)
	if err != nil {
		b.Fatal(err)
	}
	index := runtimeResult.Bundle.Index()
	if len(index) != 1 {
		b.Fatal("runtime fixture must have exactly one ordinary handler")
	}
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprintf("routes=%d", count), func(b *testing.B) {
			options := scaleProject(b, count)
			generated, err := core.Compile(context.Background(), options)
			if err != nil {
				b.Fatal(err)
			}
			if len(generated.Bundle.Index()) != count {
				b.Fatal("source generation lost distinct Gin handler templates")
			}
			engine := gin.New()
			for i := 0; i < count; i++ {
				engine.POST(fmt.Sprintf("/items/%d", i), benchfixture.Handle)
			}
			var routes []openapi.Route
			for _, route := range engine.Routes() {
				routes = append(routes, openapi.Route{Method: route.Method, Path: route.Path, OperationKey: index[0].Key, Source: openapi.Source{Kind: "derived", Rule: "gin.Engine.Routes", Symbol: route.Handler}})
			}
			config := openapi.Config{Title: "Gin scale", Version: "1", VerifyRuntimeBuild: true}
			document, err := ginswagger.Build(engine, runtimeResult.Bundle, ginswagger.Config{OpenAPI: config})
			if err != nil {
				b.Fatal(err)
			}
			neutral, err := openapi.Build(runtimeResult.Bundle, routes, config)
			if err != nil || !bytes.Equal(neutral.JSON(), document.JSON()) {
				b.Fatalf("paired core/Gin documents disagree: %v", err)
			}
			var decoded struct{ Paths map[string]json.RawMessage }
			if err := json.Unmarshal(document.JSON(), &decoded); err != nil || len(decoded.Paths) != count || len(engine.Routes()) != count {
				b.Fatalf("expected %d actual routes and paths: %v", count, err)
			}
			size := len(document.JSON())
			b.Run("Generate", func(b *testing.B) {
				output := core.WriteOptions{Dir: filepath.Join(options.Load.Dir, "internal", "apidoc")}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					compiled, err := core.Compile(context.Background(), options)
					if err != nil {
						b.Fatal(err)
					}
					if err := compiled.Write(output); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("CoreBuildSharedHandler", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := openapi.Build(runtimeResult.Bundle, routes, config); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("GinBuildSharedHandler", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := ginswagger.Build(engine, runtimeResult.Bundle, ginswagger.Config{OpenAPI: config}); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("Read", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					scaleJSON = document.JSON()
				}
			})
		})
	}
}
