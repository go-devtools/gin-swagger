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

// 五种方法的真实网络字节与生成契约一致，局部更新明确保留 false 和 null 的差别。
// Validate actual wire bytes for all five methods and preserve the distinction between false and null in partial updates.
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
		t.Fatal("五种 HTTP 方法没有完整生成")
	}
	if len(itemPath.Delete.Responses["204"].Value.Content) != 0 {
		t.Fatal("204 不应声明响应体")
	}
	legacy := schema.Paths["/examples/legacy/items/{id}"].Get
	if !legacy.Deprecated || !strings.Contains(legacy.Description, "GET /examples/items/{id}") {
		t.Fatal("弃用接口必须标记 Deprecated 并说明替代接口")
	}
	for _, sample := range []struct {
		method, route, path, body string
		status                    int
		name                      string
		enabled                   bool
	}{
		{"GET", "/examples/items/item-1", "/examples/items/{id}", "", 200, "示例资源", true},
		{"GET", "/examples/items/missing", "/examples/items/{id}", "", 404, "", false},
		{"POST", "/examples/items", "/examples/items", `{"Name":"创建资源","Enabled":true}`, 201, "创建资源", true},
		{"POST", "/examples/items", "/examples/items", `{"Name":"默认关闭"}`, 201, "默认关闭", false},
		{"POST", "/examples/items", "/examples/items", `{}`, 400, "", false},
		{"PUT", "/examples/items/item-1", "/examples/items/{id}", `{"Name":"替换资源","Enabled":false}`, 200, "替换资源", false},
		{"PUT", "/examples/items/item-1", "/examples/items/{id}", `{"Name":null}`, 400, "", false},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Enabled":false}`, 200, "示例资源", false},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Name":"改名","Enabled":null}`, 200, "改名", true},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{}`, 200, "示例资源", true},
		{"PATCH", "/examples/items/item-1", "/examples/items/{id}", `{"Enabled":"false"}`, 400, "", false},
		{"DELETE", "/examples/items/item-1", "/examples/items/{id}", "", 204, "", false},
		{"GET", "/examples/legacy/items/item-1", "/examples/legacy/items/{id}", "", 200, "旧版资源", true},
	} {
		t.Run(sample.method+sample.route+sample.body, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(sample.method, sample.route, bytes.NewBufferString(sample.body))
			if sample.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			engine.ServeHTTP(rr, req)
			if rr.Code != sample.status {
				t.Fatalf("状态 %d，预期 %d：%s", rr.Code, sample.status, rr.Body.String())
			}
			prefix := "/paths/" + escapePointer(sample.path) + "/" + strings.ToLower(sample.method)
			if sample.body != "" {
				request, err := contracttest.Compile(doc.JSON(), prefix+"/requestBody/content/application~1json/schema", contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if (request.JSON([]byte(sample.body)) == nil) != (sample.status < 400) {
					t.Fatal("请求与生成 Schema 不一致")
				}
			}
			if sample.status == 204 {
				if rr.Body.Len() != 0 || rr.Header().Get("Content-Type") != "" {
					t.Fatal("DELETE 204 意外携带响应体或媒体类型")
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
					t.Fatalf("资源状态错误：%+v", item)
				}
			}
		})
	}
}

// 顶部分类加载独立规范，分类内部仅展示有接口的标签。
// Load independent specifications from the top selector and show only tags containing operations.
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
			t.Fatalf("分类 %s 不可加载：%d", group.id, rr.Code)
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
			t.Fatalf("%s 分类范围错误：paths=%d operations=%d tags=%d", group.id, len(doc.Paths), operations, len(doc.Tags))
		}
	}
}
