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

	"example.test/gin-consumer/internal/apidoc"
	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/contracttest"
)

// 独立生成和挂载必须区分同一 handler 的 GET 表单与 POST JSON 契约。

// Independent generation and mounting must distinguish GET form and POST JSON contracts for one handler.
func TestAutomaticRemoteContract(t *testing.T) {
	before, after := gin.New(), gin.New()
	for _, engine := range []*gin.Engine{before, after} {
		engine.GET("/automatic", Automatic)
		engine.POST("/automatic", Automatic)
		engine.POST("/automatic-mandatory", AutomaticMandatory)
	}
	doc, err := ginswagger.Mount(after, apidoc.Bundle(), ginswagger.Config{
		OpenAPI:                  openapi.Config{Title: "Automatic consumer", Version: "1"},
		DefaultRequestMediaTypes: []string{"application/xml"},
		RequestMediaTypes: map[string][]string{
			"GET /automatic":            {"application/json"},
			"POST /automatic":           {"application/json"},
			"POST /automatic-mandatory": {"application/json"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		method, path, body string
		limit              int64
		status, count      int
	}{
		{"GET", "/automatic?Count=8&IDs=1&IDs=2", `{"Count":7}`, 0, 201, 8},
		{"POST", "/automatic?Count=8", `{"Count":7,"IDs":[1,2]}`, 0, 201, 7},
		{"POST", "/automatic", `{"Count":"invalid"}`, 0, 422, -1},
		{"POST", "/automatic-mandatory", `{"Count":7}`, 0, 201, 7},
		{"POST", "/automatic-mandatory", `{`, 0, 400, 0},
		{"POST", "/automatic-mandatory", `{"Count":7}`, 2, 413, 0},
	} {
		var records []*httptest.ResponseRecorder
		for _, engine := range []*gin.Engine{before, after} {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(sample.method, sample.path, strings.NewReader(sample.body))
			request.Header.Set("Content-Type", "application/json; charset=utf-8")
			if sample.limit > 0 {
				request.Body = http.MaxBytesReader(recorder, request.Body, sample.limit)
			}
			engine.ServeHTTP(recorder, request)
			records = append(records, recorder)
		}
		left, right := records[0], records[1]
		if right.Code != sample.status || left.Code != right.Code || !reflect.DeepEqual(left.Result().Header, right.Result().Header) || !bytes.Equal(left.Body.Bytes(), right.Body.Bytes()) {
			t.Fatalf("wrong or changed response: %s %s: %d %s", sample.method, sample.path, right.Code, right.Body)
		}
		if sample.status == 400 || sample.status == 413 {
			if right.Body.Len() != 0 {
				t.Fatal("mandatory failure must be bodyless")
			}
			continue
		}
		var input AutomaticInput
		if err := json.Unmarshal(right.Body.Bytes(), &input); err != nil {
			t.Fatal(err)
		}
		if input.Count != sample.count {
			t.Fatalf("wrong binder selected: %+v", input)
		}
		path := strings.SplitN(sample.path, "?", 2)[0]
		pointer := fmt.Sprintf("/paths/%s/%s/responses/%d/content/application~1json/schema", strings.ReplaceAll(path, "/", "~1"), strings.ToLower(sample.method), sample.status)
		validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err := validator.JSON(right.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
		if validator.JSON([]byte(`{"Count":"invalid","IDs":[1]}`)) == nil {
			t.Fatal("integer constraint was lost")
		}
	}
	var object struct {
		Paths map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(doc.JSON(), &object); err != nil {
		t.Fatal(err)
	}
	get, post := object.Paths["/automatic"]["get"], object.Paths["/automatic"]["post"]
	if bytes.Contains(get, []byte(`"requestBody"`)) || !bytes.Contains(get, []byte(`"parameters"`)) || !bytes.Contains(post, []byte(`"requestBody"`)) || bytes.Contains(post, []byte(`"parameters"`)) {
		t.Fatalf("conditional inputs were mixed: GET %s POST %s", get, post)
	}
	var mandatory struct {
		Responses map[string]struct{ Content map[string]json.RawMessage }
	}
	if err := json.Unmarshal(object.Paths["/automatic-mandatory"]["post"], &mandatory); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"400", "413"} {
		response, ok := mandatory.Responses[status]
		if !ok || len(response.Content) != 0 {
			t.Fatalf("wrong bodyless contract for %s", status)
		}
	}
}
