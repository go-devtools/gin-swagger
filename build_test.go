package ginswagger

import (
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/spec"
)

// 保留原始业务 handler；文档挂载不参与业务响应。
func userHandler(c *gin.Context) { c.JSON(200, struct{ Name string }{Name: "alice"}) }

// 构造具备真实运行时符号证据的中立 Bundle。
func runtimeBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	symbol := runtime.FuncForPC(reflect.ValueOf(userHandler).Pointer()).Name()
	b, err := openapi.NewBundle(openapi.BundleData{FormatVersion: 1, SpecVersion: "3.2.0", Templates: []openapi.Template{{Key: "user", Symbol: symbol, RuntimeSymbols: []string{symbol}, Operation: spec.Operation{Summary: "读取用户", Responses: map[string]spec.RefOr[spec.Response]{"200": spec.Inline(spec.Response{Description: "成功"})}}}}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// 验证 Build 不修改路由，Mount 后业务状态头与 body 不变。
func TestBuildMountAndNonInvasive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", userHandler)
	before := httptest.NewRecorder()
	r.ServeHTTP(before, httptest.NewRequest("GET", "/users/1", nil))
	bundle := runtimeBundle(t)
	cfg := Config{OpenAPI: openapi.Config{Title: "用户服务", Version: "1"}}
	doc, err := Build(r, bundle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Routes()) != 1 {
		t.Fatal("Build 修改路由")
	}
	if !strings.Contains(string(doc.JSON()), "/users/{id}") {
		t.Fatal("路径未转换")
	}
	if _, err = Mount(r, bundle, cfg); err != nil {
		t.Fatal(err)
	}
	after := httptest.NewRecorder()
	r.ServeHTTP(after, httptest.NewRequest("GET", "/users/1", nil))
	if before.Code != after.Code || before.Body.String() != after.Body.String() || !reflect.DeepEqual(before.Header(), after.Header()) {
		t.Fatal("挂载改变业务行为")
	}
	for _, path := range []string{"/docs/", "/docs/openapi.json", "/docs/swagger-ui-bundle.js"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 {
			t.Fatalf("文档路由 %s 返回 %d", path, rec.Code)
		}
	}
}

// 预检查在目标 Engine 发生任何文档注册前发现冲突。
func TestMountConflictLeavesRoutesUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/docs/existing", userHandler)
	before := r.Routes()
	_, err := Mount(r, runtimeBundle(t), Config{OpenAPI: openapi.Config{Title: "用户", Version: "1"}})
	if err == nil {
		t.Fatal("未拒绝挂载冲突")
	}
	after := r.Routes()
	if len(before) != len(after) {
		t.Fatal("预判冲突仍改变目标路由")
	}
}

// 不使用函数代码地址猜测不同捕获状态的闭包身份。
func TestAmbiguousClosureRequiresCentralBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/closure", func(c *gin.Context) { c.JSON(200, "x") })
	cfg := Config{OpenAPI: openapi.Config{Title: "用户", Version: "1"}}
	if _, err := Build(r, runtimeBundle(t), cfg); err == nil {
		t.Fatal("无证据闭包错误匹配")
	}
	cfg.Bindings = map[string]openapi.OperationKey{"GET /closure": "user"}
	if _, err := Build(r, runtimeBundle(t), cfg); err != nil {
		t.Fatal(err)
	}
}
