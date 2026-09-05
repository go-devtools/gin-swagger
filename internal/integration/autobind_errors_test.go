package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/autobind"
	"github.com/openapi-golang/openapi/contracttest"
)

// 错误和体积限制也必须服从自动绑定条件，不能沿用不适用的错误状态。
// Binding failures and body limits must follow automatic conditions without importing inapplicable error statuses.
func TestAutomaticBindingErrorPaths(t *testing.T) {
	bundle := automaticBundle(t)
	for _, sample := range []struct {
		method, media, body, url string
		handler                  gin.HandlerFunc
		limit                    int64
		status                   int
		responseBody             bool
	}{
		{"GET", "application/json", `{"Count":7}`, "/value?Count=invalid", autobind.Optional, 0, 422, true},
		{"GET", "application/json", `{"Count":7}`, "/value?Count=invalid", autobind.Mandatory, 0, 400, false},
		{"POST", "application/json", `{"Count":"invalid"}`, "/value", autobind.Optional, 0, 422, true},
		{"POST", "application/json", `{"Count":7}`, "/value", autobind.Optional, 2, 422, true},
		{"POST", "application/json", `{"Count":7}`, "/value", autobind.Mandatory, 2, 413, false},
		{"POST", "application/x-www-form-urlencoded", "Count=invalid", "/value", autobind.Mandatory, 0, 400, false},
		{"POST", "application/x-www-form-urlencoded", "Count=7", "/value", autobind.Mandatory, 2, 413, false},
		{"GET", "application/json", `{"Count":7}`, "/value?Count=8", autobind.Mandatory, 2, 201, true},
	} {
		t.Run(fmt.Sprint(sample.method, "/", sample.media, "/", sample.status, "/", sample.limit), func(t *testing.T) {
			engine := gin.New()
			engine.Handle(sample.method, "/value", sample.handler)
			document, err := ginswagger.Mount(engine, bundle, automaticConfig(t, sample.method, []string{sample.media}))
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(sample.method, sample.url, strings.NewReader(sample.body))
			request.Header.Set("Content-Type", sample.media)
			if sample.limit > 0 {
				request.Body = http.MaxBytesReader(recorder, request.Body, sample.limit)
			}
			engine.ServeHTTP(recorder, request)
			if recorder.Code != sample.status || (recorder.Body.Len() > 0) != sample.responseBody {
				t.Fatalf("wrong response: %d %s", recorder.Code, recorder.Body)
			}
			var raw map[string]any
			if err = json.Unmarshal(document.JSON(), &raw); err != nil {
				t.Fatal(err)
			}
			operation := raw["paths"].(map[string]any)["/value"].(map[string]any)[strings.ToLower(sample.method)].(map[string]any)
			responses := operation["responses"].(map[string]any)
			response, ok := responses[fmt.Sprint(sample.status)].(map[string]any)
			if !ok {
				t.Fatal("missing actual error status")
			}
			_, hasBody := response["content"]
			if hasBody != sample.responseBody {
				t.Fatal("incorrect error-body contract")
			}
			if sample.method == "GET" && responses["413"] != nil {
				t.Fatal("unread request body produced an invented 413")
			}
			if sample.responseBody {
				pointer := fmt.Sprintf("/paths/~1value/%s/responses/%d/content/application~1json/schema", strings.ToLower(sample.method), sample.status)
				validator, err := contracttest.Compile(document.JSON(), pointer, contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err = validator.JSON(recorder.Body.Bytes()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// 默认媒体设置与路由覆盖只决定文档条件，声明会被记录在来源报告中。
// Default media and route overrides select documentation conditions and appear as declared provenance.
func TestAutomaticMediaScopeConfiguration(t *testing.T) {
	bundle := automaticBundle(t)
	engine := gin.New()
	engine.POST("/value", autobind.Optional)
	cfg := automaticConfig(t, "POST", []string{"application/json"})
	cfg.DefaultRequestMediaTypes = []string{"application/xml"}
	doc, err := ginswagger.Build(engine, bundle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(doc.JSON(), []byte(`"application/json"`)) {
		t.Fatal("route override was ignored")
	}
	found := false
	for _, fact := range doc.Report().Facts {
		if fact.Kind == "declared" && fact.When != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("missing media declaration provenance")
	}
	cfg.DefaultRequestMediaTypes = []string{"application/json"}
	cfg.RequestMediaTypes = nil
	if _, err := ginswagger.Build(engine, bundle, cfg); err != nil {
		t.Fatal(err)
	}
}
