package ginswagger

import (
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-devtools/openapi"
	"github.com/go-devtools/openapi/spec"
)

// Preserve the original handler; documentation mounting does not participate in its response.
func userHandler(c *gin.Context) { c.JSON(200, struct{ Name string }{Name: "alice"}) }

// Build a neutral Bundle with actual runtime symbol evidence.
func runtimeBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	symbol := runtime.FuncForPC(reflect.ValueOf(userHandler).Pointer()).Name()
	b, err := openapi.NewBundle(openapi.BundleData{FormatVersion: 1, SpecVersion: "3.2.0", Templates: []openapi.Template{{Key: "user", Symbol: symbol, RuntimeSymbols: []string{symbol}, Operation: spec.Operation{Summary: "Read user", Responses: map[string]spec.RefOr[spec.Response]{"200": spec.Inline(spec.Response{Description: "Success"})}}}}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Verify Build has no routing side effects and Mount preserves business responses.
func TestBuildMountAndNonInvasive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", userHandler)
	before := httptest.NewRecorder()
	r.ServeHTTP(before, httptest.NewRequest("GET", "/users/1", nil))
	bundle := runtimeBundle(t)
	cfg := Config{OpenAPI: openapi.Config{Title: "User service", Version: "1"}}
	doc, err := Build(r, bundle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Routes()) != 1 {
		t.Fatal("Build changed the routes")
	}
	if !strings.Contains(string(doc.JSON()), "/users/{id}") {
		t.Fatal("path was not converted")
	}
	if _, err = Mount(r, bundle, cfg); err != nil {
		t.Fatal(err)
	}
	after := httptest.NewRecorder()
	r.ServeHTTP(after, httptest.NewRequest("GET", "/users/1", nil))
	if before.Code != after.Code || before.Body.String() != after.Body.String() || !reflect.DeepEqual(before.Header(), after.Header()) {
		t.Fatal("mounting changed business behavior")
	}
	for _, path := range []string{"/docs/", "/docs/openapi.json", "/docs/swagger-ui-bundle.js"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 {
			t.Fatalf("documentation route %s returned %d", path, rec.Code)
		}
	}
}

// Detect predictable conflicts before mutating the target Engine.
func TestMountConflictLeavesRoutesUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/docs/existing", userHandler)
	before := r.Routes()
	_, err := Mount(r, runtimeBundle(t), Config{OpenAPI: openapi.Config{Title: "Users", Version: "1"}})
	if err == nil {
		t.Fatal("mount conflict was not rejected")
	}
	after := r.Routes()
	if len(before) != len(after) {
		t.Fatal("preflight conflict still changed the target routes")
	}
}

// Do not infer captured closure state from function code addresses.
func TestAmbiguousClosureRequiresCentralBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/closure", func(c *gin.Context) { c.JSON(200, "x") })
	cfg := Config{OpenAPI: openapi.Config{Title: "Users", Version: "1"}}
	if _, err := Build(r, runtimeBundle(t), cfg); err == nil {
		t.Fatal("closure matched without sufficient evidence")
	}
	cfg.Bindings = map[string]openapi.OperationKey{"GET /closure": "user"}
	if _, err := Build(r, runtimeBundle(t), cfg); err != nil {
		t.Fatal(err)
	}
}
