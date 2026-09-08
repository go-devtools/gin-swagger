package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	"github.com/go-devtools/openapi"
	"github.com/go-devtools/openapi/contracttest"
)

// Verify shared declarations through the installed CLI, actual Gin mounting, and independent validation.
func TestDeclaredRequestResponseContract(t *testing.T) {
	_, document, err := router(true)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Paths map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(document.JSON(), &raw); err != nil {
		t.Fatal(err)
	}
	var operation struct {
		RequestBody struct{ Required bool }
		Responses   map[string]json.RawMessage
	}
	if err := json.Unmarshal(raw.Paths["/users"]["post"], &operation); err != nil {
		t.Fatal(err)
	}
	if !operation.RequestBody.Required || len(operation.Responses) != 3 || operation.Responses["201"] == nil || operation.Responses["400"] == nil || operation.Responses["default"] == nil {
		t.Fatalf("declarations replaced derived contracts: %+v", operation)
	}
	validator, err := contracttest.Compile(document.JSON(), "/paths/~1users/post/responses/default/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := validator.JSON([]byte(`{"Name":"fallback"}`)); err != nil {
		t.Fatal(err)
	}
	if validator.JSON([]byte(`{"Name":42}`)) == nil {
		t.Fatal("declared default lost its real type")
	}
	kinds := map[string]bool{}
	for _, fact := range document.Report().Facts {
		if fact.Kind == "declared" && strings.HasSuffix(fact.Symbol, ".Create") {
			kinds[fact.Rule] = true
		}
	}
	if !kinds["openapi.comment.request"] || !kinds["openapi.comment.response"] {
		t.Fatal("declaration provenance was lost")
	}
	before, after := gin.New(), gin.New()
	before.DELETE("/empty", DeclaredBodyless)
	after.DELETE("/empty", DeclaredBodyless)
	bodyless, err := ginswagger.Mount(after, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Empty", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	left, right := httptest.NewRecorder(), httptest.NewRecorder()
	before.ServeHTTP(left, httptest.NewRequest("DELETE", "/empty", nil))
	after.ServeHTTP(right, httptest.NewRequest("DELETE", "/empty", nil))
	if right.Code != 204 || left.Code != right.Code || left.Body.String() != right.Body.String() || right.Body.Len() != 0 || !reflect.DeepEqual(left.Header(), right.Header()) {
		t.Fatal("mounting changed bodyless wire behavior")
	}
	if !strings.Contains(string(bodyless.JSON()), `"204"`) {
		t.Fatal("bodyless declaration disappeared")
	}
}

// Invalid declarations remain isolated until their actual Gin routes are selected.
func TestDeclaredRouteFailures(t *testing.T) {
	for _, sample := range []struct {
		name    string
		handler gin.HandlerFunc
		code    string
	}{
		{"unknown", DeclaredUnknown, "openapi.analysis.unresolved"},
		{"conflict", DeclaredConflict, "openapi.declaration.conflict"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/test", sample.handler)
			_, err := ginswagger.Build(engine, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Failure", Version: "1"}})
			if err == nil || !strings.Contains(err.Error(), sample.code) {
				t.Fatalf("expected %s, got %v", sample.code, err)
			}
		})
	}
}
