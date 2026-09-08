package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/scalarreads"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// scalarReadBundle compiles unchanged business source through the public SDK.
func scalarReadBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	path := "testdata/scalarreads/app.go"
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/scalarreads"}, Frontends: []core.Frontend{front.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("generation changed business source")
	}
	return result.Bundle
}

// TestScalarRequestSources checks actual HTTP values and independently validates generated contracts before and after mounting.
func TestScalarRequestSources(t *testing.T) {
	bundle := scalarReadBundle(t)
	engines := []*gin.Engine{gin.New(), gin.New()}
	for _, engine := range engines {
		engine.GET("/scalar/:id", scalarreads.ScalarReads)
		engine.GET("/cookie", scalarreads.CheckedCookie)
	}
	doc, err := ginswagger.Mount(engines[1], bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Scalar request sources", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	var parsed spec.OpenAPI
	if err := json.Unmarshal(doc.JSON(), &parsed); err != nil {
		t.Fatal(err)
	}
	op := parsed.Paths["/scalar/{id}"].Get
	if op == nil || op.RequestBody != nil || len(op.Parameters) != 6 || len(op.Responses) != 1 {
		t.Fatal("unexpected scalar request contract")
	}
	locations := map[string]string{"id": "path", "q": "query", "optional": "query", "alias": "query", "x-probe": "header", "session": "cookie"}
	for i, reference := range op.Parameters {
		p := reference.Value
		if p == nil || locations[p.Name] != p.In {
			t.Fatalf("unexpected parameter: %+v", p)
		}
		delete(locations, p.Name)
		style, explode := "form", true
		if p.In == "path" || p.In == "header" {
			style, explode = "simple", false
		}
		if p.Style != style || p.Explode.Value != explode || p.Required.Value != (p.In == "path") {
			t.Fatalf("incorrect scalar serialization: %+v", p)
		}
		validator, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1scalar~1{id}/get/parameters/%d/schema", i), contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{`""`, `"0"`, `"a b"`, `"a+b"`} {
			if err := validator.JSON([]byte(value)); err != nil {
				t.Fatal(err)
			}
		}
		for _, value := range []string{`null`, `0`, `false`, `["a"]`} {
			if validator.JSON([]byte(value)) == nil {
				t.Fatalf("non-string input accepted: %s", value)
			}
		}
		if p.Name == "alias" && (!p.Schema.Default.Present || p.Schema.Default.Value != "guest") {
			t.Fatal("query fallback was lost")
		}
	}
	if len(locations) != 0 {
		t.Fatal("missing scalar sources", locations)
	}
	cookieOp := parsed.Paths["/cookie"].Get
	if cookieOp == nil || len(cookieOp.Parameters) != 1 || cookieOp.Parameters[0].Value.In != "cookie" || len(cookieOp.Responses) != 2 || cookieOp.Responses["401"].Value == nil || cookieOp.Responses["200"].Value == nil {
		t.Fatal("cookie error branch was lost or an HTTP error was invented")
	}
	scalarValidator, err := contracttest.Compile(doc.JSON(), "/paths/~1scalar~1{id}/get/responses/200/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cookieValidator, err := contracttest.Compile(doc.JSON(), "/paths/~1cookie/get/responses/200/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	servers := []*httptest.Server{httptest.NewServer(engines[0]), httptest.NewServer(engines[1])}
	for _, server := range servers {
		defer server.Close()
	}
	for _, sample := range []struct {
		name, query, cookie, value string
		missing, present           bool
		q, alias, optional         string
	}{
		{name: "missing", missing: true, alias: "guest"},
		{name: "empty", cookie: "session=", query: "q=&optional=&alias=", present: true},
		{name: "zero", cookie: "session=0", value: "0", query: "q=0&optional=0&alias=0", present: true, q: "0", optional: "0", alias: "0"},
		{name: "encoded", cookie: "session=a%20b", value: "a b", query: "q=a%20b&optional=a%2Bb", present: true, q: "a b", optional: "a+b", alias: "guest"},
		{name: "plus", cookie: "session=a+b", value: "a b", alias: "guest"},
		{name: "literal-plus", cookie: "session=a%2Bb", value: "a+b", alias: "guest"},
		{name: "repeated", cookie: "session=first; session=second", value: "first", query: "q=first&q=second&optional=one&optional=two", present: true, q: "first", optional: "one", alias: "guest"},
		{name: "quoted", cookie: `session="a%20b"`, value: "a b", alias: "guest"},
		{name: "malformed-escape", cookie: "session=%zz", alias: "guest"},
		{name: "case-sensitive", cookie: "Session=other", missing: true, alias: "guest"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			var previous http.Header
			var previousBody []byte
			for i, server := range servers {
				for _, path := range []string{"/scalar/42?" + sample.query, "/cookie"} {
					request, err := http.NewRequest("GET", server.URL+path, nil)
					if err != nil {
						t.Fatal(err)
					}
					request.Header.Add("X-Probe", "first")
					request.Header.Add("X-Probe", "second")
					if sample.cookie != "" {
						request.Header.Set("Cookie", sample.cookie)
					}
					response, err := server.Client().Do(request)
					if err != nil {
						t.Fatal(err)
					}
					body, err := io.ReadAll(response.Body)
					response.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					if path == "/cookie" {
						if sample.missing {
							if response.StatusCode != 401 || string(body) != "missing cookie" {
								t.Fatal(response.StatusCode, string(body))
							}
						} else {
							var value string
							if response.StatusCode != 200 || json.Unmarshal(body, &value) != nil || value != sample.value {
								t.Fatal(response.StatusCode, string(body))
							}
							if err := cookieValidator.JSON(body); err != nil {
								t.Fatal(err)
							}
						}
						continue
					}
					var actual scalarreads.ScalarResult
					if response.StatusCode != 200 || json.Unmarshal(body, &actual) != nil {
						t.Fatal(response.StatusCode, string(body))
					}
					want := scalarreads.ScalarResult{ID: "42", Query: sample.q, Optional: sample.optional, Present: sample.present, Default: sample.alias, Header: "first", Cookie: sample.value, CookieError: sample.missing}
					if actual != want {
						t.Fatalf("actual %+v; want %+v", actual, want)
					}
					if err := scalarValidator.JSON(body); err != nil {
						t.Fatal(err)
					}
					// Date is transport-generated and can legitimately cross a second boundary.
					response.Header.Del("Date")
					if i == 0 {
						previous, previousBody = response.Header.Clone(), body
					} else if !reflect.DeepEqual(previous, response.Header) || !bytes.Equal(previousBody, body) {
						t.Fatal("documentation mounting changed scalar request behavior")
					}
				}
			}
		})
	}
}
