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

// 使用真实零 tag handler 比较挂载前后行为，再以独立 Schema 引擎检查网络字节。
// Compare real handler responses before and after mounting, then validate their wire bytes independently.
func TestRequestContracts(t *testing.T) {
	if reflect.TypeFor[CreateUserRequest]().Field(0).Tag != "" {
		t.Fatal("示例意外依赖 DTO tag")
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
		{"有效 ASCII", `{"Name":"alice"}`, 201},
		{"有效中文", `{"Name":"张小明"}`, 201},
		{"缺失字段", `{}`, 400},
		{"显式 null", `{"Name":null}`, 400},
		{"太短", `{"Name":"小明"}`, 400},
		{"太长", `{"Name":"` + strings.Repeat("甲", 33) + `"}`, 400},
		{"错误 JSON", `{`, 400},
		{"空体", ``, 400},
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
				t.Fatalf("状态 %d，响应 %s", after.Code, after.Body.String())
			}
			if before.Code != after.Code || !reflect.DeepEqual(before.Header(), after.Header()) || before.Body.String() != after.Body.String() {
				t.Fatal("文档挂载改变了业务响应")
			}
			if accepted := request.JSON([]byte(sample.body)) == nil; accepted != (sample.status == 201) {
				t.Fatalf("样本与声明不一致：%s", sample.body)
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

// 用真实 HTTP 输出验证完整类型示例及字符串、数字枚举的反例。
// Validate the complete type sample and reject invalid string and numeric enum values.
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
		t.Fatalf("响应 %d：%s", response.Code, response.Body.String())
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
		t.Fatal("零值或 null 被错误省略")
	}
	if value["Bytes"] != "aGVsbG8=" {
		t.Fatal("字节数组没有按真实 JSON codec 编码")
	}
	for _, field := range []string{"CreatedAt", "Text", "Label", "Role", "State", "Numbers", "Address", "Coordinates", "Tags", "EmptyTags", "NilTags", "Counts", "Indexed", "EmptyMap", "NilMap", "Note", "Raw", "Dynamic", "Elapsed", "Users"} {
		if _, ok := value[field]; !ok {
			t.Errorf("缺少字段 %s", field)
		}
	}
	value["Role"] = "unknown"
	if validator.Value(value) == nil {
		t.Fatal("未拒绝字符串枚举之外的值")
	}
	value["Role"] = "admin"
	value["State"] = json.Number("3")
	if validator.Value(value) == nil {
		t.Fatal("未拒绝数字枚举之外的值")
	}
}
