package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/types"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/requests"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Compile real request samples and verify business source remains unchanged.
// 编译真实请求样本并核对业务源码保持不变。
func requestBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	before, err := os.ReadFile("testdata/requests/app.go")
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/requests"}, Frontends: []core.Frontend{front.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("testdata/requests/app.go")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("generation modified request source")
	}
	return result.Bundle
}

// Validate decoded parameter values independently while checking serialization locations separately.
// 逐个参数用独立引擎验证已解码值，序列化位置另外检查。
func validateParameters(t *testing.T, document []byte, path, location string, values map[string]any) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(document, &raw); err != nil {
		t.Fatal(err)
	}
	operation := raw["paths"].(map[string]any)[path].(map[string]any)["get"].(map[string]any)
	parameters, ok := operation["parameters"].([]any)
	if !ok {
		t.Fatal("missing bound parameters")
	}
	if len(parameters) != len(values) {
		t.Fatalf("parameters=%d values=%d", len(parameters), len(values))
	}
	for i, entry := range parameters {
		p := entry.(map[string]any)
		name := p["name"].(string)
		value, ok := values[name]
		if !ok {
			t.Fatalf("unexpected parameter %s", name)
		}
		if p["in"] != location {
			t.Fatalf("wrong parameter location: %v", p)
		}
		if location == "path" && p["required"] != true {
			t.Fatal("optional path parameter")
		}
		pointer := "/paths/" + strings.ReplaceAll(path, "/", "~1") + "/get/parameters/" + fmt.Sprint(i) + "/schema"
		validator, err := contracttest.Compile(document, pointer, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := validator.Value(value); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name == "Role" && validator.Value("owner") == nil {
			t.Fatal("enum constraints lost in text parameter projection")
		}
		if name == "Count" || name == "ID" {
			if validator.Value("invalid") == nil {
				t.Fatalf("lost numeric constraint for %s", name)
			}
		}
		if name == "IDs" {
			if validator.Value("AQI=") == nil {
				t.Fatal("parameter bytes became Base64")
			}
			if p["style"] != "form" || p["explode"] != true {
				t.Fatal("repeated query values lost serialization")
			}
		}
		if name == "Limit" && validator.Value(nil) == nil {
			t.Fatal("text parameter allows JSON null")
		}
		if name == "Name" {
			if p["required"] != true || validator.Value("A") == nil {
				t.Fatal("field comments lost")
			}
		}
	}
}

// Real requests, responses, and independent contracts must agree before and after mounting.
// 实际请求、响应和独立契约必须同时匹配，挂载前后保持等价。
func TestExplicitRequestBinders(t *testing.T) {
	bundle := requestBundle(t)
	query := "Role=admin&Name=Ada&Count=7&IDs=1&IDs=2&Limit=3&Pair=1&Pair=2&Active=true&At=2026-01-01T00%3A00%3A00Z&Delay=1s"
	values := map[string]any{"Role": "admin", "Name": "Ada", "Count": 7, "IDs": []any{1, 2}, "Limit": 3, "Pair": []any{1, 2}, "Active": true, "At": "2026-01-01T00:00:00Z", "Delay": "1s"}
	for _, sample := range []struct {
		route, path, url, method, location, media, body string
		headers                                         map[string]string
		values                                          map[string]any
	}{
		{"/query", "/query", "/query?" + query, "GET", "query", "", "", nil, values},
		{"/query-explicit", "/query-explicit", "/query-explicit?" + query, "GET", "query", "", "", nil, values},
		{"/uri/:ID", "/uri/{ID}", "/uri/7", "GET", "path", "", "", nil, map[string]any{"ID": 7}},
		{"/header", "/header", "/header", "GET", "header", "", "", map[string]string{"token": "secret", "COUNT": "7"}, map[string]any{"Token": "secret", "Count": 7}},
		{"/form", "/form", "/form?Name=ignored", "POST", "", "application/x-www-form-urlencoded", query, nil, values},
		{"/json", "/json", "/json", "POST", "", "application/json", `{"Token":"secret","Count":7}`, nil, map[string]any{"Token": "secret", "Count": 7}},
		{"/body-json", "/body-json", "/body-json", "POST", "", "application/json", `{"Token":"secret","Count":7}`, nil, map[string]any{"Token": "secret", "Count": 7}},
	} {
		t.Run(sample.route, func(t *testing.T) {
			before, after := requests.Router(), requests.Router()
			document, err := ginswagger.Mount(after, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Requests", Version: "1"}, Include: func(_, path string) bool { return path == sample.route }})
			if err != nil {
				t.Fatal(err)
			}
			var records []*httptest.ResponseRecorder
			for _, engine := range []*gin.Engine{before, after} {
				req := httptest.NewRequest(sample.method, sample.url, strings.NewReader(sample.body))
				if sample.media != "" {
					req.Header.Set("Content-Type", sample.media)
				}
				for name, value := range sample.headers {
					req.Header.Set(name, value)
				}
				record := httptest.NewRecorder()
				engine.ServeHTTP(record, req)
				records = append(records, record)
			}
			if records[0].Code != 200 || records[1].Code != records[0].Code || !bytes.Equal(records[0].Body.Bytes(), records[1].Body.Bytes()) || !reflect.DeepEqual(records[0].Result().Header, records[1].Result().Header) {
				t.Fatalf("unexpected or changed response: %d %s", records[1].Code, records[1].Body)
			}
			var actual map[string]any
			if err := json.Unmarshal(records[1].Body.Bytes(), &actual); err != nil {
				t.Fatal(err)
			}
			if actual["Count"] != float64(7) && actual["ID"] != float64(7) {
				t.Fatal("actual binder lost numeric value")
			}
			if sample.route == "/form" && actual["Name"] != "Ada" {
				t.Fatal("FormPost incorrectly included query fallback")
			}
			prefix := "/paths/" + strings.ReplaceAll(sample.path, "/", "~1") + "/" + strings.ToLower(sample.method)
			response, err := contracttest.Compile(document.JSON(), prefix+"/responses/200/content/application~1json/schema", contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err := response.JSON(records[1].Body.Bytes()); err != nil {
				t.Fatal(err)
			}
			if sample.location != "" {
				validateParameters(t, document.JSON(), sample.path, sample.location, sample.values)
			} else {
				input, err := contracttest.Compile(document.JSON(), prefix+"/requestBody/content/"+strings.ReplaceAll(sample.media, "/", "~1")+"/schema", contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err := input.Value(sample.values); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Multipart files retain raw-byte semantics and actual upload results satisfy the response contract.
// multipart 文件保持原始字节语义，真实上传返回值通过响应契约。
func TestMultipartRequestBinding(t *testing.T) {
	engine := requests.Router()
	doc, err := ginswagger.Mount(engine, requestBundle(t), ginswagger.Config{OpenAPI: openapi.Config{Title: "Upload", Version: "1"}, Include: func(_, path string) bool { return path == "/multipart" }})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("Title", "Report"); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct{ field, name string }{{"File", "report.bin"}, {"Files", "first.bin"}, {"Files", "second.bin"}} {
		part, err := writer.CreateFormFile(file.field, file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write([]byte{0, 255, 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/multipart", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	record := httptest.NewRecorder()
	engine.ServeHTTP(record, req)
	if record.Code != 200 || record.Body.String() != `{"Count":2,"Filename":"report.bin","Title":"Report"}` {
		t.Fatalf("unexpected upload: %d %s", record.Code, record.Body)
	}
	response, err := contracttest.Compile(doc.JSON(), "/paths/~1multipart/post/responses/200/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := response.JSON(record.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(doc.JSON(), &raw); err != nil {
		t.Fatal(err)
	}
	operation := raw["paths"].(map[string]any)["/multipart"].(map[string]any)["post"].(map[string]any)
	schema := operation["requestBody"].(map[string]any)["content"].(map[string]any)["multipart/form-data"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := schema["$ref"].(string); ok {
		schema = raw["components"].(map[string]any)["schemas"].(map[string]any)[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any)
	}
	fields := schema["properties"].(map[string]any)
	file := fields["File"].(map[string]any)
	if file["contentMediaType"] != "application/octet-stream" || file["type"] != nil || file["contentEncoding"] != nil {
		t.Fatalf("wrong file representation: %v", file)
	}
	if fields["Files"].(map[string]any)["type"] != "array" {
		t.Fatal("multiple upload shape lost")
	}
}

// Actual invalid-request paths still satisfy their contracts, and mounting must not alter status or responses.
// 非法请求的真实错误路径仍满足契约，挂载不能改变原有状态或响应。
func TestBindingErrorResponses(t *testing.T) {
	bundle := requestBundle(t)
	for _, sample := range []struct {
		route, path, url, method, media, body string
		headers                               map[string]string
	}{
		{"/query", "/query", "/query?Count=invalid", "GET", "", "", nil},
		{"/query-explicit", "/query-explicit", "/query-explicit?Pair=1", "GET", "", "", nil},
		{"/uri/:ID", "/uri/{ID}", "/uri/invalid", "GET", "", "", nil},
		{"/header", "/header", "/header", "GET", "", "", map[string]string{"Count": "invalid"}},
		{"/form", "/form", "/form", "POST", "application/x-www-form-urlencoded", "Count=invalid", nil},
		{"/json", "/json", "/json", "POST", "application/json", "{", nil},
		{"/body-json", "/body-json", "/body-json", "POST", "application/json", "{", nil},
	} {
		t.Run(sample.route, func(t *testing.T) {
			before, after := requests.Router(), requests.Router()
			doc, err := ginswagger.Mount(after, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Invalid request", Version: "1"}, Include: func(_, path string) bool { return path == sample.route }})
			if err != nil {
				t.Fatal(err)
			}
			var records []*httptest.ResponseRecorder
			for _, engine := range []*gin.Engine{before, after} {
				request := httptest.NewRequest(sample.method, sample.url, strings.NewReader(sample.body))
				if sample.media != "" {
					request.Header.Set("Content-Type", sample.media)
				}
				for name, value := range sample.headers {
					request.Header.Set(name, value)
				}
				record := httptest.NewRecorder()
				engine.ServeHTTP(record, request)
				records = append(records, record)
			}
			if records[1].Code != 400 || records[0].Code != records[1].Code || records[0].Body.String() != records[1].Body.String() || !reflect.DeepEqual(records[0].Result().Header, records[1].Result().Header) {
				t.Fatalf("invalid request changed: %d %s", records[1].Code, records[1].Body)
			}
			pointer := "/paths/" + strings.ReplaceAll(sample.path, "/", "~1") + "/" + strings.ToLower(sample.method) + "/responses/400/content/application~1json/schema"
			validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err := validator.JSON(records[1].Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Check embedded fields and unknown binding rules separately so successful samples cannot hide incorrect inference.
// 嵌入字段和未知绑定规则分别验证，避免成功样本掩盖错误自动推导。
func TestBindingBoundaries(t *testing.T) {
	bundle := requestBundle(t)
	for _, route := range []string{"/xml", "/custom", "/repeated-header"} {
		if _, err := ginswagger.Build(requests.Router(), bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Unsupported", Version: "1"}, Include: func(_, path string) bool { return path == route }}); err == nil {
			t.Errorf("unsupported binder accepted: %s", route)
		}
	}
	engine := requests.Router()
	doc, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Embedded", Version: "1"}, Include: func(_, path string) bool { return path == "/embedded" }})
	if err != nil {
		t.Fatal(err)
	}
	record := httptest.NewRecorder()
	engine.ServeHTTP(record, httptest.NewRequest("GET", "/embedded?Shared=Ada&Time=not-a-time", nil))
	if record.Code != 200 || record.Body.String() != `{"Shared":"Ada"}` {
		t.Fatalf("unexpected embedded request: %d %s", record.Code, record.Body)
	}
	validateParameters(t, doc.JSON(), "/embedded", "query", map[string]any{"Shared": "Ada"})
}

// A centralized mapper can describe custom parameter decoding without changing DTOs or handlers.
// 自定义参数解码可以通过一次集中 mapper 补足，无需更改 DTO 或 handler。
func TestCustomBindingMapper(t *testing.T) {
	mapper := func(request core.ProjectionRequest) (*spec.Schema, bool, error) {
		named, ok := types.Unalias(request.Type).(*types.Named)
		if !ok || request.Direction != core.Input || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "github.com/openapi-golang/gin-swagger/internal/integration/testdata/requests" || named.Obj().Name() != "CustomParam" {
			return nil, false, nil
		}
		schema := spec.Typed("string")
		schema.MinLength = spec.Set(uint64(2))
		return schema, true, nil
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/requests"}, Frontends: []core.Frontend{front.Frontend()}, Mappers: []core.TypeMapper{mapper}, Configuration: map[string]json.RawMessage{"custom-parameter": json.RawMessage(`{"version":1,"minLength":2}`)}})
	if err != nil {
		t.Fatal(err)
	}
	engine := requests.Router()
	doc, err := ginswagger.Mount(engine, result.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Mapped binding", Version: "1"}, Include: func(_, path string) bool { return path == "/custom" }})
	if err != nil {
		t.Fatal(err)
	}
	record := httptest.NewRecorder()
	engine.ServeHTTP(record, httptest.NewRequest("GET", "/custom?Value=hello", nil))
	if record.Code != 200 || record.Body.String() != `{"Value":"custom:hello"}` {
		t.Fatalf("unexpected custom binding: %d %s", record.Code, record.Body)
	}
	input, err := contracttest.Compile(doc.JSON(), "/paths/~1custom/get/parameters/0/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := input.Value("hello"); err != nil {
		t.Fatal(err)
	}
	if input.Value("a") == nil {
		t.Fatal("central mapper constraint lost")
	}
	response, err := contracttest.Compile(doc.JSON(), "/paths/~1custom/get/responses/200/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := response.JSON(record.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
}
