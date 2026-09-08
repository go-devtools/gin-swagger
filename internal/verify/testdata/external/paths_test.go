package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	"github.com/go-devtools/openapi"
)

// Verify the public path configuration against generated real-source templates in an independent module.
func TestPublishedPathProfilesAndSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name, route, documented, wire string
		raw, escaped                  bool
	}{
		{"decoded", "/percent/%2F", "/percent/%252F", "/percent/%252F", false, false},
		{"Unicode", "/users/张三", "/users/%E5%BC%A0%E4%B8%89", "/users/%E5%BC%A0%E4%B8%89", false, false},
		{"raw", "/percent/%2F", "/percent/%2F", "/percent/%2F", true, false},
		{"escaped", "/percent/%20", "/percent/%20", "/percent/%20", false, true},
		{"colon", "/action/run\\:now", "/action/run:now", "/action/run:now", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			engine.UseRawPath, engine.UseEscapedPath = tc.raw, tc.escaped
			engine.GET(tc.route, Text)
			saved := engine.Routes()
			cfg := ginswagger.Config{OpenAPI: openapi.Config{Title: "Independent paths", Version: "1"}}
			before, err := ginswagger.Build(engine, apidoc.Bundle(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest("GET", tc.wire, nil))
			if recorder.Code != 200 || recorder.Body.String() != "ready" {
				t.Fatalf("documented wire path failed: %d %q", recorder.Code, recorder.Body.String())
			}
			cfg.RegisteredRoutes = saved
			after, err := ginswagger.Build(engine, apidoc.Bundle(), cfg)
			if err != nil || !bytes.Equal(before.JSON(), after.JSON()) {
				t.Fatalf("initialized path changed: %v", err)
			}
			var document map[string]json.RawMessage
			var paths map[string]json.RawMessage
			if err := json.Unmarshal(after.JSON(), &document); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(document["paths"], &paths); err != nil {
				t.Fatal(err)
			}
			if len(paths) != 1 || paths[tc.documented] == nil {
				t.Fatalf("unexpected public path: %s", after.JSON())
			}
		})
	}
}
