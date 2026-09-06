// Verify public generation and mounting boundaries against real Gin responses.
// 用真实 Gin 响应验证生成与挂载的公开边界。
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	ginswagger "github.com/openapi-golang/gin-swagger"
	gincompiler "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/responses"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
)

// Compile a real fixture and prove generation leaves its source bytes unchanged.
// 编译真实 fixture，并证明源码在生成过程中保持逐字节不变。
func responseBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	before, err := os.ReadFile("testdata/responses/app.go")
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/responses"}, Frontends: []core.Frontend{gincompiler.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("testdata/responses/app.go")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("generation modified handler source")
	}
	return result.Bundle
}

// Validate text or JSON independently and ensure raw binary is not described as Base64.
// 独立验证字符串或 JSON 样本，并检查原始二进制没有被描述为 Base64。
func validateResponse(t *testing.T, document []byte, path, media string, status int, body []byte) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(document, &raw); err != nil {
		t.Fatal(err)
	}
	op := raw["paths"].(map[string]any)[path].(map[string]any)["get"].(map[string]any)
	response := op["responses"].(map[string]any)[strconv.Itoa(status)].(map[string]any)
	if media == "" {
		if content, ok := response["content"].(map[string]any); ok && len(content) != 0 {
			t.Fatal("bodyless response has content")
		}
		if len(body) != 0 {
			t.Fatalf("unexpected body: %q", body)
		}
		return
	}
	content, ok := response["content"].(map[string]any)[media].(map[string]any)
	if !ok {
		t.Fatalf("missing response media %s", media)
	}
	schema, ok := content["schema"].(map[string]any)
	if !ok {
		t.Fatal("missing explicit wire schema")
	}
	if media == "application/octet-stream" {
		if schema["contentMediaType"] != media || schema["type"] != nil || schema["contentEncoding"] != nil {
			t.Fatalf("raw bytes misrepresented: %v", schema)
		}
		return
	}
	escape := func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1") }
	pointer := "/paths/" + escape(path) + "/get/responses/" + strconv.Itoa(status) + "/content/" + escape(media) + "/schema"
	validator, err := contracttest.Compile(document, pointer, contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if media == "application/json" {
		err = validator.JSON(body)
	} else {
		err = validator.Value(string(body))
	}
	if err != nil {
		t.Fatal(err)
	}
	if validator.Value(7) == nil {
		t.Fatal("wire schema lost its type constraint")
	}
}

// Compare real status, headers, bytes, and contracts on equivalent engines before and after mounting.
// 挂载前后使用等价 Engine，核对真实状态、头、字节与文档契约。
func TestResponseRenderingAndCommit(t *testing.T) {
	bundle := responseBundle(t)
	for _, sample := range []struct {
		route, url, path, media, body string
		status                        int
	}{
		{"/plain/:name", "/plain/Ada", "/plain/{name}", "text/plain", "hello Ada", 201},
		{"/raw", "/raw", "/raw", "application/octet-stream", "\x00\xff\n", 202},
		{"/render-json", "/render-json", "/render-json", "application/json", `{"Name":"Ada"}`, 203},
		{"/render-text", "/render-text", "/render-text", "text/plain", "rendered", 200},
		{"/render-indented", "/render-indented", "/render-indented", "application/json", "{\n    \"Name\": \"Ada\"\n}", 200},
		{"/render-ascii", "/render-ascii", "/render-ascii", "application/json", `{"Name":"Ada"}`, 200},
		{"/render-pure", "/render-pure", "/render-pure", "application/json", "{\"Name\":\"Ada\"}\n", 200},
		{"/render-reader", "/render-reader", "/render-reader", "application/octet-stream", "abc", 200},
		{"/render-pointer", "/render-pointer", "/render-pointer", "application/json", `{"Name":"Ada"}`, 200},
		{"/render-null", "/render-null", "/render-null", "application/json", "null", 200},
		{"/render-data", "/render-data", "/render-data", "application/octet-stream", "\x01\x02", 200},
		{"/no-content", "/no-content", "/no-content", "", "", 204},
		{"/not-modified", "/not-modified", "/not-modified", "", "", 304},
		{"/abort", "/abort", "/abort", "", "", 403},
		{"/abort-json", "/abort-json", "/abort-json", "application/json", `{"Name":"after abort"}`, 401},
		{"/abort-error", "/abort-error", "/abort-error", "", "", 409},
		{"/headers", "/headers", "/headers", "application/json", `{"Name":"headers"}`, 200},
		{"/header-branches", "/header-branches?side=left", "/header-branches", "application/json", `{"Name":"left"}`, 200},
		{"/header-branches", "/header-branches?side=right", "/header-branches", "application/json", `{"Name":"right"}`, 201},
		{"/reader", "/reader", "/reader", "application/octet-stream", "abc", 200},
	} {
		t.Run(sample.url, func(t *testing.T) {
			before, after := responses.Router(), responses.Router()
			config := ginswagger.Config{OpenAPI: openapi.Config{Title: "Response matrix", Version: "1"}, Include: func(method, path string) bool { return method == "GET" && path == sample.route }}
			document, err := ginswagger.Mount(after, bundle, config)
			if err != nil {
				t.Fatal(err)
			}
			first, second := httptest.NewRecorder(), httptest.NewRecorder()
			before.ServeHTTP(first, httptest.NewRequest("GET", sample.url, nil))
			after.ServeHTTP(second, httptest.NewRequest("GET", sample.url, nil))
			a, b := first.Result(), second.Result()
			defer a.Body.Close()
			defer b.Body.Close()
			original, _ := io.ReadAll(a.Body)
			actual, _ := io.ReadAll(b.Body)
			if a.StatusCode != b.StatusCode || !reflect.DeepEqual(a.Header, b.Header) || !bytes.Equal(original, actual) {
				t.Fatal("mount changed business response")
			}
			if b.StatusCode != sample.status || string(actual) != sample.body {
				t.Fatalf("status=%d body=%q", b.StatusCode, actual)
			}
			validateResponse(t, document.JSON(), sample.path, sample.media, b.StatusCode, actual)
			var raw map[string]any
			if err := json.Unmarshal(document.JSON(), &raw); err != nil {
				t.Fatal(err)
			}
			op := raw["paths"].(map[string]any)[sample.path].(map[string]any)["get"].(map[string]any)
			entry := op["responses"].(map[string]any)[strconv.Itoa(sample.status)].(map[string]any)
			headers, _ := entry["headers"].(map[string]any)
			for _, name := range []string{"X-Ready", "X-Mode", "X-Branch", "X-Reader", "X-Extra", "Content-Length"} {
				if value := b.Header.Get(name); value != "" && (name != "Content-Length" || sample.route == "/reader") {
					header, ok := headers[name].(map[string]any)
					if !ok {
						t.Fatalf("missing response header %s", name)
					}
					if header["schema"].(map[string]any)["const"] != value {
						t.Fatalf("wrong header %s: %v", name, header)
					}
				}
			}
			if headers["X-Removed"] != nil || headers["X-Late"] != nil {
				t.Fatal("removed or late header entered contract")
			}
		})
	}
}

// Invalid consecutive output and unknown renderers must still block selected-route construction.
// 非法连续输出及未知 Renderer 仍必须阻断所选路由的构建。
func TestUnresolvedRenderersRemainDiagnostics(t *testing.T) {
	bundle := responseBundle(t)
	for _, path := range []string{"/multiple", "/unknown-render", "/unknown-writer", "/reader-pending"} {
		_, err := ginswagger.Build(responses.Router(), bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Unsupported", Version: "1"}, Include: func(_, actual string) bool { return actual == path }})
		if err == nil {
			t.Errorf("unresolved route %s was accepted", path)
		}
	}
}

// Interim and final statuses are not one ordinary commit; reject the contract until that sequence is modeled.
// 临时响应与最终状态不是一次普通提交，未建模前必须阻断错误契约。
func TestInterimResponseNeedsSequenceModel(t *testing.T) {
	server := httptest.NewServer(responses.Router())
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/interim")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || string(body) != `{"Name":"final"}` {
		t.Fatalf("unexpected actual sequence: %d %q", response.StatusCode, body)
	}
	_, err = ginswagger.Build(responses.Router(), responseBundle(t), ginswagger.Config{OpenAPI: openapi.Config{Title: "Interim", Version: "1"}, Include: func(_, path string) bool { return path == "/interim" }})
	if err == nil {
		t.Fatal("interim status was mistaken for the final wire response")
	}
}

// Use actual committed headers to ensure a bodyless reader does not declare skipped header writes.
// 用真实提交头确认无内容 Reader 不会声明尚未执行的头设置。
func TestBodylessReaderHeaders(t *testing.T) {
	bundle := responseBundle(t)
	for _, sample := range []struct {
		path   string
		status int
	}{{"/reader-no-content", 204}, {"/reader-not-modified", 304}} {
		t.Run(sample.path, func(t *testing.T) {
			engine := responses.Router()
			document, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Bodyless reader", Version: "1"}, Include: func(_, path string) bool { return path == sample.path }})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest("GET", sample.path, nil))
			response := recorder.Result()
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != sample.status || len(body) != 0 {
				t.Fatalf("unexpected response: %d %q", response.StatusCode, body)
			}
			var raw map[string]any
			if err := json.Unmarshal(document.JSON(), &raw); err != nil {
				t.Fatal(err)
			}
			op := raw["paths"].(map[string]any)[sample.path].(map[string]any)["get"].(map[string]any)
			entry := op["responses"].(map[string]any)[strconv.Itoa(sample.status)].(map[string]any)
			headers, _ := entry["headers"].(map[string]any)
			for _, name := range []string{"Content-Length", "X-Reader-Extra"} {
				if response.Header.Get(name) != "" {
					t.Fatalf("unexpected actual header %s", name)
				}
				if headers[name] != nil {
					t.Errorf("contract declares skipped header %s", name)
				}
			}
			validateResponse(t, document.JSON(), sample.path, "", sample.status, body)
		})
	}
}
