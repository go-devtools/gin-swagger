package main

import (
	"bytes"
	"io"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/openapi-golang/openapi/contracttest"
)

// 生成契约与真实请求相符，文档挂载不改变业务响应。
// Match generated contracts to actual requests without changing business responses through mounting.
func TestGeneratedContract(t *testing.T) {
	before, _, err := router(false)
	if err != nil {
		t.Fatal(err)
	}
	after, document, err := router(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		method, path, input string
		status              int
	}{
		{"POST", "/users", `{"Name":"Alice"}`, 201},
		{"POST", "/users", `{`, 400},
		{"GET", "/text", "", 200},
	} {
		left, right := httptest.NewRecorder(), httptest.NewRecorder()
		before.ServeHTTP(left, httptest.NewRequest(sample.method, sample.path, bytes.NewBufferString(sample.input)))
		after.ServeHTTP(right, httptest.NewRequest(sample.method, sample.path, bytes.NewBufferString(sample.input)))
		a, b := left.Result(), right.Result()
		original, _ := io.ReadAll(a.Body)
		actual, _ := io.ReadAll(b.Body)
		a.Body.Close()
		b.Body.Close()
		if b.StatusCode != sample.status || a.StatusCode != b.StatusCode || !reflect.DeepEqual(a.Header, b.Header) || !bytes.Equal(original, actual) {
			t.Fatal("documentation changed business response")
		}
		pointer := "/paths/~1users/post/responses/201/content/application~1json/schema"
		if sample.status == 400 {
			pointer = "/paths/~1users/post/responses/400/content/application~1json/schema"
		}
		if sample.method == "GET" {
			pointer = "/paths/~1text/get/responses/200/content/text~1plain/schema"
		}
		validator, err := contracttest.Compile(document.JSON(), pointer, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if sample.method == "GET" {
			err = validator.Value(string(actual))
		} else {
			err = validator.JSON(actual)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	request, err := contracttest.Compile(document.JSON(), "/paths/~1users/post/requestBody/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if request.JSON([]byte(`{"Name":"A"}`)) == nil {
		t.Fatal("source constraint was lost")
	}
}
