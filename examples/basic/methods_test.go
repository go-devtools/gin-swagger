package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Validate actual wire bytes for all five methods and preserve the distinction between false and null in partial updates.
// 五种方法的真实网络字节与生成契约一致，局部更新明确保留 false 和 null 的差别。
func TestHTTPMethodExamples(t *testing.T) {
	engine, doc, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	var schema spec.OpenAPI
	if err = json.Unmarshal(doc.JSON(), &schema); err != nil {
		t.Fatal(err)
	}
	itemPath := schema.Paths["/examples/items/{id}"]
	if itemPath == nil || itemPath.Get == nil || itemPath.Put == nil || itemPath.Patch == nil || itemPath.Delete == nil || schema.Paths["/examples/items"].Post == nil {
		t.Fatal("not all five HTTP methods were generated")
	}
	if len(itemPath.Delete.Responses["204"].Value.Content) != 0 {
		t.Fatal("204 must not declare a response body")
	}
	legacy := schema.Paths["/examples/legacy/items/{id}"].Get
	if !legacy.Deprecated || !strings.Contains(legacy.Description, "GET /examples/items/{id}") {
		t.Fatal("deprecated operation must be marked Deprecated and describe its replacement")
	}
	for _, sample := range []struct {
		method, route, path, body string
		status                    int
		name                      string
		enabled                   bool
	}{
		{"GET", "/examples/items/item-1", "/examples/items/{id}", "", 200, "Example resource", true},
		{"GET", "/examples/items/missing", "/examples/items/{id}", "", 404, "", false},
		{"POST", "/examples/items", "/examples/items", `{"Name":"Created resource","Enabled":true}`, 201, "Created resource", true},
		{"POST", "/examples/items", "/examples/items", `{"Name":"Disabled by default"}`, 201, "Disabled by default", false},
		{"POST", "/examples/items", "/examples/items", `{}`, 400, "", false},
		{"PUT", "/examples/items/item-1", "/examples/items/{id}", `{"Name":"Replacement resource","Enabled":false}`, 200, "Replacement resource", false},
		{"PUT", "/examples/items/item-1", "/examples/items/{id}", `{"Name":null}`, 400, "", false},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Enabled":false}`, 200, "Example resource", false},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Name":"Renamed","Enabled":null}`, 200, "Renamed", true},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{}`, 200, "Example resource", true},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Enabled":"false"}`, 400, "", false},
		{"DELETE", "/examples/items/item-1", "/examples/items/{id}", "", 204, "", false},
		{"GET", "/examples/legacy/items/item-1", "/examples/legacy/items/{id}", "", 200, "Legacy resource", true},
	} {
		t.Run(sample.method+sample.route+sample.body, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(sample.method, sample.route, bytes.NewBufferString(sample.body))
			if sample.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			engine.ServeHTTP(rr, req)
			if rr.Code != sample.status {
				t.Fatalf("status %d, expected %d: %s", rr.Code, sample.status, rr.Body.String())
			}
			prefix := "/paths/" + escapePointer(sample.path) + "/" + strings.ToLower(sample.method)
			if sample.body != "" {
				request, err := contracttest.Compile(doc.JSON(), prefix+"/requestBody/content/application~1json/schema", contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if (request.JSON([]byte(sample.body)) == nil) != (sample.status < 400) {
					t.Fatal("request differs from the generated Schema")
				}
			}
			if sample.status == 204 {
				if rr.Body.Len() != 0 || rr.Header().Get("Content-Type") != "" {
					t.Fatal("DELETE 204 unexpectedly included a response body or media type")
				}
				return
			}
			response, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("%s/responses/%d/content/application~1json/schema", prefix, sample.status), contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err = response.JSON(rr.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
			if sample.status < 400 {
				var item Item
				if err = json.Unmarshal(rr.Body.Bytes(), &item); err != nil {
					t.Fatal(err)
				}
				if item.Name != sample.name || item.Enabled != sample.enabled {
					t.Fatalf("resource state is incorrect: %+v", item)
				}
			}
		})
	}
}

// Load independent specifications from the top selector and show only tags containing operations.
// 顶部分类加载独立规范，分类内部仅展示有接口的标签。
func TestExampleDocumentDefinitions(t *testing.T) {
	engine, _, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range []struct {
		id                      string
		paths, operations, tags int
	}{
		{"all", 7, 10, 6}, {"resources", 3, 6, 2}, {"types", 2, 2, 2}, {"auth", 1, 1, 1}, {"legacy", 1, 1, 1},
	} {
		rr := httptest.NewRecorder()
		engine.ServeHTTP(rr, httptest.NewRequest("GET", "/docs/groups/"+group.id+".json", nil))
		if rr.Code != 200 {
			t.Fatalf("group %s could not be loaded: %d", group.id, rr.Code)
		}
		var doc spec.OpenAPI
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		operations := 0
		for _, path := range doc.Paths {
			for _, op := range []*spec.Operation{path.Get, path.Post, path.Put, path.Delete, path.Patch} {
				if op != nil {
					operations++
				}
			}
		}
		if len(doc.Paths) != group.paths || operations != group.operations || len(doc.Tags) != group.tags {
			t.Fatalf("%s group scope mismatch: paths=%d operations=%d tags=%d", group.id, len(doc.Paths), operations, len(doc.Tags))
		}
	}
}
