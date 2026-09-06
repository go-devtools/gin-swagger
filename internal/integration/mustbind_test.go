package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/mustbind"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
)

// Check mandatory-binding success, failure, and size-limit commits without changing actual responses.
// 核对强制绑定的成功、错误与限流提交，文档挂载不能改变真实响应。
func TestMandatoryBindingOutcomes(t *testing.T) {
	beforeSource, err := os.ReadFile("testdata/mustbind/app.go")
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/mustbind"}, Frontends: []core.Frontend{front.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	afterSource, err := os.ReadFile("testdata/mustbind/app.go")
	if err != nil || !bytes.Equal(beforeSource, afterSource) {
		t.Fatal("compiler modified handlers")
	}
	for _, sample := range []struct {
		name, route, url, header string
		handler                  gin.HandlerFunc
		body                     bool
	}{
		{"Checked", "/checked", "/checked", "", mustbind.Checked, true},
		{"Ignored", "/ignored", "/ignored", "", mustbind.Ignored, true},
		{"Rewrite", "/rewrite", "/rewrite", "", mustbind.Rewrite, true},
		{"Explicit", "/explicit", "/explicit", "", mustbind.Explicit, true},
		{"Query", "/query", "/query?Count=7", "", mustbind.Query, false},
		{"Header", "/header", "/header", "7", mustbind.Header, false},
		{"URI", "/uri/:Count", "/uri/7", "", mustbind.URI, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			before, after := gin.New(), gin.New()
			before.POST(sample.route, sample.handler)
			after.POST(sample.route, sample.handler)
			doc, err := ginswagger.Mount(after, result.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Mandatory binding", Version: "1"}})
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err = json.Unmarshal(doc.JSON(), &data); err != nil {
				t.Fatal(err)
			}
			path := strings.ReplaceAll(sample.route, ":Count", "{Count}")
			responses := data["paths"].(map[string]any)[path].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)
			var got []string
			for status := range responses {
				got = append(got, status)
			}
			sort.Strings(got)
			want := []string{"201", "400"}
			if sample.body {
				want = append(want, "413")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("responses=%v, want %v", got, want)
			}
			variants := []string{"valid", "invalid"}
			if sample.body {
				variants = append(variants, "limited")
			}
			for _, variant := range variants {
				t.Run(variant, func(t *testing.T) {
					url, header, body := sample.url, sample.header, `{"Count":7}`
					expected := 201
					if variant == "invalid" {
						expected = 400
						body = `{"Count":`
						url = strings.ReplaceAll(url, "7", "bad")
						if header != "" {
							header = "bad"
						}
					}
					if variant == "limited" {
						expected = 413
					}
					var records []*httptest.ResponseRecorder
					for _, engine := range []*gin.Engine{before, after} {
						record := httptest.NewRecorder()
						req := httptest.NewRequest("POST", url, strings.NewReader(body))
						req.Header.Set("Content-Type", "application/json")
						if header != "" {
							req.Header.Set("Count", header)
						}
						if variant == "limited" {
							req.Body = http.MaxBytesReader(record, req.Body, 2)
						}
						engine.ServeHTTP(record, req)
						records = append(records, record)
					}
					actual := records[1]
					if actual.Code != expected || records[0].Code != actual.Code || !bytes.Equal(records[0].Body.Bytes(), actual.Body.Bytes()) || !reflect.DeepEqual(records[0].Result().Header, actual.Result().Header) {
						t.Fatalf("wrong or changed response: %d %s", actual.Code, actual.Body)
					}
					hasBody := variant == "valid" || sample.name == "Ignored" || sample.name == "Rewrite"
					response := responses[http.StatusText(0)]
					_ = response
					status := "201"
					if expected == 400 {
						status = "400"
					}
					if expected == 413 {
						status = "413"
					}
					contract := responses[status].(map[string]any)
					_, content := contract["content"]
					if content != hasBody || (actual.Body.Len() > 0) != hasBody {
						t.Fatalf("incorrect body presence: %v %s", contract, actual.Body)
					}
					if hasBody {
						prefix := "/paths/" + strings.ReplaceAll(path, "/", "~1") + "/post/responses/" + status + "/content/application~1json/schema"
						validator, err := contracttest.Compile(doc.JSON(), prefix, contracttest.Options{})
						if err != nil {
							t.Fatal(err)
						}
						if err = validator.JSON(actual.Body.Bytes()); err != nil {
							t.Fatal(err)
						}
						if validator.JSON([]byte(`{"unexpected":true}`)) == nil {
							t.Fatal("response schema lost required shape")
						}
					}
				})
			}
		})
	}
}
