package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-devtools/openapi/contracttest"
)

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
		{"GET", "/search?IDs=1&IDs=2", "", 200},
		{"GET", "/search?IDs=invalid", "", 400},
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
		if sample.path == "/text" {
			pointer = "/paths/~1text/get/responses/200/content/text~1plain/schema"
		}
		if strings.HasPrefix(sample.path, "/search") {
			pointer = fmt.Sprintf("/paths/~1search/get/responses/%d/content/application~1json/schema", sample.status)
		}
		validator, err := contracttest.Compile(document.JSON(), pointer, contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if sample.path == "/text" {
			err = validator.Value(string(actual))
		} else {
			err = validator.JSON(actual)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	parameter, err := contracttest.Compile(document.JSON(), "/paths/~1search/get/parameters/0/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := parameter.Value([]any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if parameter.Value("AQI=") == nil {
		t.Fatal("independent query binding used JSON byte semantics")
	}
	request, err := contracttest.Compile(document.JSON(), "/paths/~1users/post/requestBody/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if request.JSON([]byte(`{"Name":"A"}`)) == nil {
		t.Fatal("source constraint was lost")
	}
}
