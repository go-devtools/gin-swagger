package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi/contracttest"
)

// Compare real handler responses before and after mounting, then validate their wire bytes independently.
// 使用真实零 tag handler 比较挂载前后行为，再以独立 Schema 引擎检查网络字节。
func TestRequestContracts(t *testing.T) {
	if reflect.TypeFor[CreateUserRequest]().Field(0).Tag != "" {
		t.Fatal("example unexpectedly depends on DTO tags")
	}
	original := gin.New()
	original.POST("/users", CreateUser)
	mounted, doc, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	request, err := contracttest.Compile(doc.JSON(), "/paths/~1users/post/requestBody/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		name, body string
		status     int
	}{
		{"Valid ASCII", `{"Name":"alice"}`, 201},
		{"Valid Unicode", "{\"Name\":\"\u5f20\u5c0f\u660e\"}", 201},
		{"Missing field", `{}`, 400},
		{"Explicit null", `{"Name":null}`, 400},
		{"Too short", "{\"Name\":\"\u5c0f\u660e\"}", 400},
		{"Too long", `{"Name":"` + strings.Repeat("\u7532", 33) + `"}`, 400},
		{"Invalid JSON", `{`, 400},
		{"Empty body", ``, 400},
	} {
		t.Run(sample.name, func(t *testing.T) {
			call := func(engine *gin.Engine) *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(sample.body))
				req.Header.Set("Content-Type", "application/json")
				record := httptest.NewRecorder()
				engine.ServeHTTP(record, req)
				return record
			}
			before, after := call(original), call(mounted)
			if after.Code != sample.status {
				t.Fatalf("status %d, response %s", after.Code, after.Body.String())
			}
			if before.Code != after.Code || !reflect.DeepEqual(before.Header(), after.Header()) || before.Body.String() != after.Body.String() {
				t.Fatal("mounting documentation changed the business response")
			}
			if accepted := request.JSON([]byte(sample.body)) == nil; accepted != (sample.status == 201) {
				t.Fatalf("sample differs from its declaration: %s", sample.body)
			}
			pointer := fmt.Sprintf("/paths/~1users/post/responses/%d/content/application~1json/schema", sample.status)
			response, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err = response.JSON(after.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Validate the complete type sample and reject invalid string and numeric enum values.
// 用真实 HTTP 输出验证完整类型示例及字符串、数字枚举的反例。
func TestAllFieldTypesContract(t *testing.T) {
	engine, doc, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	validator, err := contracttest.Compile(doc.JSON(), "/paths/~1examples~1types/get/responses/200/content/application~1json/schema", contracttest.Options{AssertFormat: true})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/examples/types", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatalf("response %d: %s", response.Code, response.Body.String())
	}
	if err = validator.JSON(response.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(response.Body.Bytes()))
	decoder.UseNumber()
	if err = decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value["Enabled"] != false || value["State"] != json.Number("0") || value["MissingNote"] != nil {
		t.Fatal("zero value or null was incorrectly omitted")
	}
	if value["Bytes"] != "aGVsbG8=" {
		t.Fatal("byte array does not match the actual JSON codec encoding")
	}
	for _, field := range []string{"CreatedAt", "Text", "Label", "Role", "State", "Numbers", "Address", "Coordinates", "Tags", "EmptyTags", "NilTags", "Counts", "Indexed", "EmptyMap", "NilMap", "Note", "Raw", "Dynamic", "Elapsed", "Users"} {
		if _, ok := value[field]; !ok {
			t.Errorf("missing field %s", field)
		}
	}
	value["Role"] = "unknown"
	if validator.Value(value) == nil {
		t.Fatal("value outside the string enum was not rejected")
	}
	value["Role"] = "admin"
	value["State"] = json.Number("3")
	if validator.Value(value) == nil {
		t.Fatal("value outside the numeric enum was not rejected")
	}
}
