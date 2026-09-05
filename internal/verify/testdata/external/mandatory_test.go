package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/contracttest"
)

// 固定远端 CLI 的产物必须保留真实 400/413 提交和后续 JSON 响应。
// Output from the fixed remote CLI must preserve actual 400/413 commits and subsequent JSON responses.
func TestMandatoryRemoteContract(t *testing.T) {
	before, after := gin.New(), gin.New()
	before.POST("/mandatory", Mandatory)
	after.POST("/mandatory", Mandatory)
	doc, err := ginswagger.Mount(after, apidoc.Bundle(), ginswagger.Config{OpenAPI: openapi.Config{Title: "Mandatory consumer", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(doc.JSON()), `"422"`) {
		t.Fatal("unreachable override status leaked into the contract")
	}
	for _, sample := range []struct {
		body   string
		limit  int64
		status int
	}{{`{"Name":"Alice"}`, 0, 201}, {`{`, 0, 400}, {`{"Name":"Alice"}`, 2, 413}} {
		var records []*httptest.ResponseRecorder
		for _, engine := range []*gin.Engine{before, after} {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/mandatory", strings.NewReader(sample.body))
			request.Header.Set("Content-Type", "application/json")
			if sample.limit > 0 {
				request.Body = http.MaxBytesReader(recorder, request.Body, sample.limit)
			}
			engine.ServeHTTP(recorder, request)
			records = append(records, recorder)
		}
		actual := records[1]
		if actual.Code != sample.status || records[0].Code != actual.Code || !reflect.DeepEqual(records[0].Result().Header, actual.Result().Header) || !bytes.Equal(records[0].Body.Bytes(), actual.Body.Bytes()) {
			t.Fatalf("wrong or changed response: %d %s", actual.Code, actual.Body)
		}
		pointer := fmt.Sprintf("/paths/~1mandatory/post/responses/%d/content/application~1json/schema", sample.status)
		validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err = validator.JSON(actual.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
}
