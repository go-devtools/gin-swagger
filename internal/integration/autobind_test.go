package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/autobind"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
)

// 编译真实自动绑定源码，并验证编译过程没有改写业务文件。
// Compile actual automatic-binding source and verify compilation leaves business files unchanged.
func automaticBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	before, err := os.ReadFile("testdata/autobind/app.go")
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/autobind"}, Frontends: []core.Frontend{front.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("testdata/autobind/app.go")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("compiler changed business source")
	}
	return result.Bundle
}

// 设置文档媒体范围，不修改实际请求或 Gin 路由。
// Configure documentation media scope without modifying actual requests or Gin routes.
func automaticConfig(t *testing.T, method string, media []string) ginswagger.Config {
	t.Helper()
	cfg := ginswagger.Config{OpenAPI: openapi.Config{Title: "Automatic binding", Version: "1"}, RequestMediaTypes: map[string][]string{method + " /value": media}}
	return cfg
}

// 独立构造请求体，使实际 Gin 选择和生成的条件契约可以互相校验。
// Construct bodies independently so real Gin selection and generated conditional contracts can cross-check each other.
func automaticBody(t *testing.T, media string) (string, string) {
	t.Helper()
	switch media {
	case "application/json":
		return `{"Name":"body","Count":7,"IDs":[1,2]}`, media + "; charset=utf-8"
	case "application/x-www-form-urlencoded":
		return "Name=body&Count=7&IDs=1&IDs=2", media
	case "multipart/form-data":
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for _, field := range [][2]string{{"Name", "body"}, {"Count", "7"}, {"IDs", "1"}, {"IDs", "2"}} {
			if err := writer.WriteField(field[0], field[1]); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return buffer.String(), writer.FormDataContentType()
	}
	return "ignored body", media
}

// 验证自动和显式 Form 绑定的真实来源、优先顺序及挂载前后等价性。
// Verify actual sources, precedence, and mount equivalence for automatic and explicit Form binding.
func TestAutomaticBinderConditions(t *testing.T) {
	bundle := automaticBundle(t)
	for _, sample := range []struct {
		method, media string
		handler       gin.HandlerFunc
		name          string
		count         float64
		ids           []any
		query, body   bool
	}{
		{"GET", "application/json", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"GET", "application/xml", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"GET", "application/x-www-form-urlencoded", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"GET", "multipart/form-data", autobind.Optional, "query", 8, []any{float64(9), float64(1), float64(2)}, true, true},
		{"POST", "application/json", autobind.Optional, "body", 7, []any{float64(1), float64(2)}, false, true},
		{"PUT", "application/json", autobind.Optional, "body", 7, []any{float64(1), float64(2)}, false, true},
		{"POST", "application/x-www-form-urlencoded", autobind.Optional, "body", 7, []any{float64(1), float64(2), float64(9)}, true, true},
		{"PUT", "application/x-www-form-urlencoded", autobind.Optional, "body", 7, []any{float64(1), float64(2), float64(9)}, true, true},
		{"PATCH", "application/x-www-form-urlencoded", autobind.Optional, "body", 7, []any{float64(1), float64(2), float64(9)}, true, true},
		{"DELETE", "application/x-www-form-urlencoded", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"QUERY", "application/x-www-form-urlencoded", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"POST", "multipart/form-data", autobind.Optional, "body", 7, []any{float64(1), float64(2)}, false, true},
		{"POST", "text/plain", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"POST", "", autobind.Optional, "query", 8, []any{float64(9)}, true, false},
		{"POST", "application/json", autobind.ExplicitForm, "query", 8, []any{float64(9)}, true, false},
		{"POST", "multipart/form-data", autobind.ExplicitForm, "query", 8, []any{float64(9), float64(1), float64(2)}, true, true},
		{"POST", "application/json", autobind.Mandatory, "body", 7, []any{float64(1), float64(2)}, false, true},
	} {
		t.Run(fmt.Sprintf("%s/%s/%p", sample.method, sample.media, sample.handler), func(t *testing.T) {
			before, after := gin.New(), gin.New()
			before.Handle(sample.method, "/value", sample.handler)
			after.Handle(sample.method, "/value", sample.handler)
			doc, err := ginswagger.Mount(after, bundle, automaticConfig(t, sample.method, []string{sample.media}))
			if err != nil {
				t.Fatal(err)
			}
			body, header := automaticBody(t, sample.media)
			var records []*httptest.ResponseRecorder
			for _, engine := range []*gin.Engine{before, after} {
				record := httptest.NewRecorder()
				request := httptest.NewRequest(sample.method, "/value?Name=query&Count=8&IDs=9", strings.NewReader(body))
				if header != "" {
					request.Header.Set("Content-Type", header)
				}
				engine.ServeHTTP(record, request)
				records = append(records, record)
			}
			actual := records[1]
			if actual.Code != 201 || records[0].Code != actual.Code || !reflect.DeepEqual(records[0].Result().Header, actual.Result().Header) || !bytes.Equal(records[0].Body.Bytes(), actual.Body.Bytes()) {
				t.Fatalf("wrong or changed response %d %s", actual.Code, actual.Body)
			}
			var result map[string]any
			if err = json.Unmarshal(actual.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result["Count"] != sample.count || result["Name"] != sample.name || !reflect.DeepEqual(result["IDs"], sample.ids) {
				t.Fatalf("incorrect binding precedence: %v", result)
			}
			method := strings.ToLower(sample.method)
			pointer := "/paths/~1value/" + method + "/responses/201/content/application~1json/schema"
			validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err = validator.JSON(actual.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = json.Unmarshal(doc.JSON(), &document); err != nil {
				t.Fatal(err)
			}
			operation := document["paths"].(map[string]any)["/value"].(map[string]any)[method].(map[string]any)
			_, hasQuery := operation["parameters"]
			_, hasBody := operation["requestBody"]
			if hasQuery != sample.query || hasBody != sample.body {
				t.Fatalf("incorrect source projection: %s", doc.JSON())
			}
			if sample.body {
				inputPointer := "/paths/~1value/" + method + "/requestBody/content/" + strings.ReplaceAll(sample.media, "/", "~1") + "/schema"
				input, err := contracttest.Compile(doc.JSON(), inputPointer, contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				values := map[string]any{"Name": "body", "Count": 7, "IDs": []any{1, 2}}
				if err := input.Value(values); err != nil {
					t.Fatal(err)
				}
				values["Count"] = "invalid"
				if input.Value(values) == nil {
					t.Fatal("conditional request lost numeric representation")
				}
				values["Count"] = 7
				values["IDs"] = nil
				if sample.media != "application/json" && input.Value(values) == nil {
					t.Fatal("form arrays acquired JSON null semantics")
				}
			}
			if sample.query {
				for i, rawParameter := range operation["parameters"].([]any) {
					parameter := rawParameter.(map[string]any)
					input, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1value/%s/parameters/%d/schema", method, i), contracttest.Options{})
					if err != nil {
						t.Fatal(err)
					}
					if parameter["name"] == "Count" && input.Value("invalid") == nil {
						t.Fatal("query count lost integer semantics")
					}
					if parameter["name"] == "IDs" && (parameter["style"] != "form" || parameter["explode"] != true || input.Value(nil) == nil) {
						t.Fatal("query collection lost repeated-value semantics")
					}
				}
			}
		})
	}
}

// 未确定媒体、未知 codec 与跨位置必填歧义必须在所选路由上明确失败。
// Unresolved media, unknown codecs, and ambiguous cross-location requirements must fail on selected routes.
func TestAutomaticBinderBoundaries(t *testing.T) {
	bundle := automaticBundle(t)
	for _, sample := range []struct {
		method  string
		media   []string
		handler gin.HandlerFunc
		success bool
	}{
		{"POST", nil, autobind.Optional, false},
		{"POST", []string{"application/xml"}, autobind.Optional, false},
		{"GET", []string{"application/xml"}, autobind.Optional, true},
		{"POST", []string{"application/json", "multipart/form-data"}, autobind.Optional, true},
		{"POST", []string{"application/json"}, autobind.Required, true},
		{"POST", []string{"application/x-www-form-urlencoded"}, autobind.Required, false},
	} {
		engine := gin.New()
		engine.Handle(sample.method, "/value", sample.handler)
		_, err := ginswagger.Build(engine, bundle, automaticConfig(t, sample.method, sample.media))
		if (err == nil) != sample.success {
			t.Fatalf("%s %v: %v", sample.method, sample.media, err)
		}
	}
}
