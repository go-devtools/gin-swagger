package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

// Build equivalent instances so requests before and after mounting cannot share business state.
func importedRouter() *gin.Engine {
	r := gin.New()
	r.POST("/metadata/json", ImportedJSON)
	r.GET("/metadata/query", ImportedQuery)
	r.POST("/metadata/form", ImportedForm)
	return r
}

// Preserve imported constraints, enums, and generic field comments through actual generation and mounting.
func TestImportedDTOContracts(t *testing.T) {
	before, after := importedRouter(), importedRouter()
	doc, err := ginswagger.Mount(after, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Imported DTO", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Display name from the imported DTO.", "The actual payload from the imported envelope.", "Administrator.", "Editor."} {
		if !bytes.Contains(doc.JSON(), []byte(text)) {
			t.Errorf("lost imported metadata: %s", text)
		}
	}
	var metadata struct {
		Components struct {
			Schemas map[string]struct {
				Enum         []string
				Descriptions []string `json:"x-enum-descriptions"`
			}
		}
	}
	if err := json.Unmarshal(doc.JSON(), &metadata); err != nil {
		t.Fatal(err)
	}
	enums := 0
	for _, schema := range metadata.Components.Schemas {
		if reflect.DeepEqual(schema.Enum, []string{"admin", "editor"}) {
			enums++
			if !reflect.DeepEqual(schema.Descriptions, []string{"Administrator.", "Editor."}) {
				t.Fatalf("enum labels do not match their values: %+v", schema)
			}
		}
	}
	if enums == 0 {
		t.Fatal("imported enum components were lost")
	}
	responseSchema, err := contracttest.Compile(doc.JSON(), "/paths/~1metadata~1json/post/responses/200/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{`{"Data":{"Name":"Al","Role":"admin","State":"custom"}}`, `{"Data":{"Name":"Alice","Role":"unknown","State":"custom"}}`} {
		if responseSchema.JSON([]byte(invalid)) == nil {
			t.Fatalf("imported generic response constraint was lost: %s", invalid)
		}
	}
	for _, entry := range apidoc.Bundle().Index() {
		if strings.Contains(entry.Symbol, "/testdata/contracts.") {
			t.Fatalf("dependency handler entered source-root discovery: %s", entry.Symbol)
		}
	}
	for _, sample := range []struct {
		method, path, target, media, body string
		status                            int
	}{
		{"POST", "/metadata/json", "/metadata/json", "application/json", `{"Name":"Alice","Role":"admin","State":"custom"}`, 200},
		{"POST", "/metadata/json", "/metadata/json", "application/json", `{`, 400},
		{"GET", "/metadata/query", "/metadata/query?Name=Alice&Role=editor&State=custom", "", "", 200},
		{"POST", "/metadata/form", "/metadata/form", "application/x-www-form-urlencoded", "Name=Alice&Role=admin&State=custom", 200},
	} {
		t.Run(sample.method+sample.target+sample.body, func(t *testing.T) {
			left, right := httptest.NewRecorder(), httptest.NewRecorder()
			for i, engine := range []*gin.Engine{before, after} {
				request := httptest.NewRequest(sample.method, sample.target, strings.NewReader(sample.body))
				if sample.media != "" {
					request.Header.Set("Content-Type", sample.media)
				}
				engine.ServeHTTP([]*httptest.ResponseRecorder{left, right}[i], request)
			}
			if right.Code != sample.status || left.Code != right.Code || !bytes.Equal(left.Body.Bytes(), right.Body.Bytes()) || !reflect.DeepEqual(left.Header(), right.Header()) {
				t.Fatalf("mount changed the wire response: %d/%d %s/%s", left.Code, right.Code, left.Body, right.Body)
			}
			pointer := fmt.Sprintf("/paths/%s/%s/responses/%d/content/application~1json/schema", strings.ReplaceAll(sample.path, "/", "~1"), strings.ToLower(sample.method), sample.status)
			validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err := validator.JSON(right.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, location := range []string{"/paths/~1metadata~1json/post/requestBody/content/application~1json/schema", "/paths/~1metadata~1form/post/requestBody/content/application~1x-www-form-urlencoded/schema"} {
		validator, err := contracttest.Compile(doc.JSON(), location, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := validator.JSON([]byte(`{"Name":"Alice","Role":"admin","State":"custom"}`)); err != nil {
			t.Fatal(err)
		}
		for _, invalid := range []string{`{"Name":"Al","Role":"admin"}`, `{"Role":"admin"}`, `{"Name":"Alice","Role":"unknown"}`} {
			if validator.JSON([]byte(invalid)) == nil {
				t.Errorf("imported body contract accepted %s at %s", invalid, location)
			}
		}
	}
	var document struct {
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name, In, Description string
				Required              bool
			}
		}
	}
	if err := json.Unmarshal(doc.JSON(), &document); err != nil {
		t.Fatal(err)
	}
	parameters := document.Paths["/metadata/query"]["get"].Parameters
	if len(parameters) != 3 {
		t.Fatalf("expected three imported query fields, got %+v", parameters)
	}
	for i, parameter := range parameters {
		if parameter.In != "query" {
			t.Fatalf("wrong parameter location: %+v", parameter)
		}
		validator, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1metadata~1query/get/parameters/%d/schema", i), contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		switch parameter.Name {
		case "Name":
			if !parameter.Required || parameter.Description != "Display name from the imported DTO." || validator.Value("Al") == nil || validator.Value("Alice") != nil {
				t.Fatalf("imported query constraint was lost: %+v", parameter)
			}
		case "Role":
			if validator.Value("unknown") == nil || validator.Value("editor") != nil {
				t.Fatal("imported query enum was lost")
			}
		case "State":
			if err := validator.Value("custom"); err != nil {
				t.Fatal("ordinary constants incorrectly closed the imported string")
			}
		default:
			t.Fatalf("unexpected imported field: %s", parameter.Name)
		}
	}
}

// Contract annotations must not inject validation into ordinary business requests.
func TestImportedDeclarationsDoNotEnforceRuntimeValidation(t *testing.T) {
	engine := importedRouter()
	doc, err := ginswagger.Mount(engine, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Declared constraints", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"Name":"A","Role":"unknown","State":"custom"}`)
	validator, err := contracttest.Compile(doc.JSON(), "/paths/~1metadata~1json/post/requestBody/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if validator.JSON(raw) == nil {
		t.Fatal("declared client constraints were lost")
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/metadata/json", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(response, request)
	if response.Code != 200 || response.Body.String() != `{"Data":{"Name":"A","Role":"unknown","State":"custom"}}` {
		t.Fatalf("annotations changed the original handler: %d %s", response.Code, response.Body)
	}
}

// Retain dependency field evidence per selected route without mutating routes or the immutable Bundle.
func TestImportedRouteDiagnosticSources(t *testing.T) {
	bundle := apidoc.Bundle()
	original := bundle.JSON()
	for _, sample := range []struct {
		handler              gin.HandlerFunc
		method, code, symbol string
	}{
		{ImportedMalformed, "POST", "openapi.comment.invalid", "Malformed.Value"},
		{ImportedIncompatible, "GET", "openapi.comment.type", "Incompatible.Value"},
	} {
		for _, path := range []string{"/first", "/second"} {
			engine := gin.New()
			engine.Handle(sample.method, path, sample.handler)
			_, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Failure", Version: "1"}})
			var report openapi.Report
			if !errors.As(err, &report) {
				t.Fatalf("missing structured report: %v", err)
			}
			found := false
			for _, issue := range report.Diagnostics {
				if issue.Code != sample.code {
					continue
				}
				found = true
				if issue.Route != sample.method+" "+path || issue.Source.File != "example.test/gin-consumer/testdata/contracts/dto.go" || issue.Source.Symbol != "example.test/gin-consumer/testdata/contracts."+sample.symbol || issue.Source.Line <= 0 || issue.Source.Column <= 0 || len(issue.Facts) == 0 || issue.Fix == "" {
					t.Fatalf("incomplete imported diagnostic: %+v", issue)
				}
			}
			if !found || len(engine.Routes()) != 1 || !bytes.Equal(original, bundle.JSON()) {
				t.Fatalf("missing %s or failed mount changed shared state: %+v", sample.code, report)
			}
		}
	}
}
