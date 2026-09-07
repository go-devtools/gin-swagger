package ginswagger

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/spec"
)

// Expose the value selected by the actual Gin router for independent wire assertions.
func pathHandler(c *gin.Context) { c.String(200, "value="+c.Param("id")) }

// Describe only a response; Build must derive every path and parameter from public routes.
func pathBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	symbol := runtime.FuncForPC(reflect.ValueOf(pathHandler).Pointer()).Name()
	bundle, err := openapi.NewBundle(openapi.BundleData{FormatVersion: 1, SpecVersion: "3.2.0", Templates: []openapi.Template{{Key: "path", RuntimeSymbols: []string{symbol}, Operation: spec.Operation{Responses: map[string]spec.RefOr[spec.Response]{"200": spec.Inline(spec.Response{Description: "Success"})}}}}})
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

// Compare generated URLs with actual requests under each public Gin path configuration.
func TestPathEncodingMatchesGinRequests(t *testing.T) {
	bundle := pathBundle(t)
	for _, tc := range []struct {
		name, route, path, wire, value, rejected string
		raw, escaped, conditional, invalid       bool
	}{
		{name: "decoded Unicode", route: "/users/张三", path: "/users/%E5%BC%A0%E4%B8%89", wire: "/users/%E5%BC%A0%E4%B8%89"},
		{name: "decoded space", route: "/a b", path: "/a%20b", wire: "/a%20b"},
		{name: "decoded percent", route: "/percent/%2F", path: "/percent/%252F", wire: "/percent/%252F", rejected: "/percent/%2F"},
		{name: "decoded malformed percent", route: "/percent/%xz", path: "/percent/%25xz", wire: "/percent/%25xz"},
		{name: "decoded braces", route: "/brace/{value}", path: "/brace/%7Bvalue%7D", wire: "/brace/%7Bvalue%7D"},
		{name: "decoded query and fragment", route: "/mark/?#end", path: "/mark/%3F%23end", wire: "/mark/%3F%23end"},
		{name: "decoded static colon", route: "/a\\:value", path: "/a:value", wire: "/a:value"},
		{name: "decoded prefix and parameter", route: "/张三/:id", path: "/%E5%BC%A0%E4%B8%89/{id}", wire: "/%E5%BC%A0%E4%B8%89/42", value: "42"},
		{name: "raw canonical fallback", raw: true, route: "/percent/%20", path: "/percent/%2520", wire: "/percent/%2520", rejected: "/percent/%20"},
		{name: "raw noncanonical literal", raw: true, route: "/percent/%2F", path: "/percent/%2F", wire: "/percent/%2F"},
		{name: "raw lowercase literal", raw: true, route: "/percent/%2f/:id", path: "/percent/%2f/{id}", wire: "/percent/%2f/a%20b", value: "a b"},
		{name: "raw unreserved literal", raw: true, route: "/percent/%41/:id", path: "/percent/%41/{id}", wire: "/percent/%41/a%2Fb", value: "a/b"},
		{name: "raw conditional prefix", raw: true, conditional: true, route: "/percent/%20/:id", path: "/percent/%2520/{id}", wire: "/percent/%2520/a%20b", value: "a b", rejected: "/percent/%2520/a%2Fb"},
		{name: "raw Unicode prefix", raw: true, conditional: true, route: "/张三/:id", path: "/%E5%BC%A0%E4%B8%89/{id}", wire: "/%E5%BC%A0%E4%B8%89/42", value: "42"},
		{name: "escaped canonical percent", escaped: true, route: "/percent/%20", path: "/percent/%20", wire: "/percent/%20", rejected: "/percent/%2520"},
		{name: "escaped overrides raw", raw: true, escaped: true, route: "/percent/%20", path: "/percent/%20", wire: "/percent/%20"},
		{name: "escaped parameter", escaped: true, route: "/percent/%2F/:id", path: "/percent/%2F/{id}", wire: "/percent/%2F/a%2Fb", value: "a/b"},
		{name: "escaped Unicode rejected", escaped: true, route: "/users/张三", invalid: true},
		{name: "escaped space rejected", escaped: true, route: "/a b", invalid: true},
		{name: "escaped malformed percent rejected", escaped: true, route: "/percent/%xz", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			engine.UseRawPath, engine.UseEscapedPath = tc.raw, tc.escaped
			calls := 0
			engine.Use(func(c *gin.Context) { calls++; c.Next() })
			engine.GET(tc.route, pathHandler)
			saved := engine.Routes()
			cfg := Config{OpenAPI: openapi.Config{Title: "Path encodings", Version: "1"}}
			doc, err := Build(engine, bundle, cfg)
			if calls != 0 {
				t.Fatal("Build executed a request")
			}
			if tc.invalid {
				if err == nil || !strings.Contains(err.Error(), "gin-swagger.path.escaped") {
					t.Fatalf("expected escaped path diagnostic, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]json.RawMessage
			var paths map[string]map[string]map[string]any
			if err := json.Unmarshal(doc.JSON(), &document); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(document["paths"], &paths); err != nil {
				t.Fatal(err)
			}
			operation := paths[tc.path]["get"]
			if len(paths) != 1 || operation == nil {
				t.Fatalf("unexpected path: %s", doc.JSON())
			}
			_, conditional := operation["x-gin-raw-path-note"]
			if conditional != tc.conditional {
				t.Fatalf("conditional routing note = %t", conditional)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest("GET", tc.wire, nil))
			if recorder.Code != 200 || recorder.Body.String() != "value="+tc.value {
				t.Fatalf("wire %s returned %d %q", tc.wire, recorder.Code, recorder.Body.String())
			}
			if tc.rejected != "" {
				negative := httptest.NewRecorder()
				engine.ServeHTTP(negative, httptest.NewRequest("GET", tc.rejected, nil))
				if negative.Code != 404 {
					t.Fatalf("incorrect wire %s returned %d", tc.rejected, negative.Code)
				}
			}
			beforeCalls := calls
			cfg.RegisteredRoutes = saved
			after, err := Build(engine, bundle, cfg)
			if err != nil || !bytes.Equal(doc.JSON(), after.JSON()) {
				t.Fatalf("initialization changed the document: %v", err)
			}
			if calls != beforeCalls {
				t.Fatal("Build executed business middleware")
			}
		})
	}
}

// Restore colliding static and dynamic colon paths from an unmodified public snapshot.
func TestRegisteredRoutesPreserveInitializedColons(t *testing.T) {
	engine := gin.New()
	engine.GET("/r/r\\:id", pathHandler)
	engine.GET("/r/r:id", pathHandler)
	saved := engine.Routes()
	cfg := Config{OpenAPI: openapi.Config{Title: "Colon paths", Version: "1"}, RegisteredRoutes: saved}
	bundle := pathBundle(t)
	before, err := Build(engine, bundle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, wire := range []string{"/r/r:id", "/r/r42"} {
		engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", wire, nil))
	}
	current := engine.Routes()
	if current[0].Path != current[1].Path {
		t.Fatal("fixture did not expose Gin's collapsed paths")
	}
	if _, err := Build(engine, bundle, Config{OpenAPI: cfg.OpenAPI}); err == nil || !strings.Contains(err.Error(), "gin-swagger.routes.ambiguous") {
		t.Fatalf("missing ambiguity diagnostic: %v", err)
	}
	after, err := Build(engine, bundle, cfg)
	if err != nil || !bytes.Equal(before.JSON(), after.JSON()) {
		t.Fatalf("saved paths were not preserved: %v", err)
	}
	if _, err := Mount(engine, bundle, cfg); err != nil {
		t.Fatal(err)
	}
	for wire, body := range map[string]string{"/r/r:id": "value=", "/r/r42": "value=42"} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest("GET", wire, nil))
		if recorder.Code != 200 || recorder.Body.String() != body {
			t.Fatalf("Mount changed route %s: %d %q", wire, recorder.Code, recorder.Body.String())
		}
	}
	if saved[0].Path != "/r/r\\:id" && saved[1].Path != "/r/r\\:id" {
		t.Fatal("Build modified the caller's snapshot")
	}
}

// Reject changed route evidence before any documentation endpoints are registered.
func TestRegisteredRoutesRejectStaleEvidence(t *testing.T) {
	for _, mutation := range []string{"added", "method", "path", "handler", "function", "duplicate"} {
		t.Run(mutation, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/users/:id", pathHandler)
			engine.GET("/items/:id", pathHandler)
			saved := engine.Routes()
			switch mutation {
			case "added":
				engine.GET("/later", pathHandler)
			case "method":
				saved[0].Method = "POST"
			case "path":
				saved[0].Path = "/else/:id"
			case "handler":
				saved[0].Handler = "another.function"
			case "function":
				saved[0].HandlerFunc = userHandler
			case "duplicate":
				saved[0] = saved[1]
			}
			count := len(engine.Routes())
			_, err := Mount(engine, pathBundle(t), Config{OpenAPI: openapi.Config{Title: "Stale routes", Version: "1"}, RegisteredRoutes: saved})
			if err == nil || !strings.Contains(err.Error(), "gin-swagger.routes.stale") {
				t.Fatalf("expected stale snapshot diagnostic, got %v", err)
			}
			if len(engine.Routes()) != count {
				t.Fatal("failed validation registered documentation routes")
			}
		})
	}
}

// Keep the selected scope independent from excluded ambiguities and retain dynamic prefixes.
func TestPathScopeUsesOriginalSnapshot(t *testing.T) {
	for _, prefix := range []string{"/v1", "/tenant/v2"} {
		engine := gin.New()
		group := engine.Group(prefix)
		group.GET("/r/r\\:id", pathHandler)
		group.GET("/r/r:id", pathHandler)
		engine.GET("/health", pathHandler)
		cfg := Config{OpenAPI: openapi.Config{Title: "Scope", Version: "1"}, RegisteredRoutes: engine.Routes()}
		cfg.Include = func(method, path string) bool { return method == "GET" && strings.Contains(path, "\\:") }
		before, err := Build(engine, pathBundle(t), cfg)
		if err != nil {
			t.Fatal(err)
		}
		engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", prefix+"/r/r:id", nil))
		after, err := Build(engine, pathBundle(t), cfg)
		if err != nil || !bytes.Equal(before.JSON(), after.JSON()) {
			t.Fatalf("scope changed after initialization: %v", err)
		}
		if !bytes.Contains(after.JSON(), []byte(prefix+"/r/r:id")) || bytes.Contains(after.JSON(), []byte("/health")) {
			t.Fatal("scope did not use original full paths")
		}
		cfg.RegisteredRoutes = nil
		cfg.Include = func(method, path string) bool { return path == "/health" }
		if _, err := Build(engine, pathBundle(t), cfg); err != nil {
			t.Fatalf("excluded paths blocked the selected scope: %v", err)
		}
	}
}

// Keep Gin parameter decoding visible when raw or escaped path values are deliberately retained.
func TestPathValueUnescapingMatchesGin(t *testing.T) {
	for _, escaped := range []bool{false, true} {
		for _, unescape := range []bool{false, true} {
			engine := gin.New()
			engine.UseRawPath, engine.UseEscapedPath = true, escaped
			engine.UnescapePathValues = unescape
			engine.GET("/encoded/%2F/:id", pathHandler)
			doc, err := Build(engine, pathBundle(t), Config{OpenAPI: openapi.Config{Title: "Path values", Version: "1"}})
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(doc.JSON(), &document); err != nil {
				t.Fatal(err)
			}
			operation := document["paths"].(map[string]any)["/encoded/%2F/{id}"].(map[string]any)["get"].(map[string]any)
			if operation["x-gin-unescape-path-values"] != unescape {
				t.Fatal("decoding setting was not retained")
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/encoded/%2F/a+b%2Fc", nil))
			expected := "value=a+b%2Fc"
			if unescape {
				expected = "value=a b/c"
			}
			if recorder.Code != 200 || recorder.Body.String() != expected {
				t.Fatalf("unexpected Gin parameter value: %d %q", recorder.Code, recorder.Body.String())
			}
		}
	}
}
