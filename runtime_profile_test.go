package ginswagger

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi"
)

// 构建条件不匹配必须在挂载任何文档路由前失败。
// Reject mismatched build conditions before mounting any documentation routes.
func TestRuntimeProfileMismatchLeavesRoutesUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/users/:id", userHandler)
	before := engine.Routes()
	data := runtimeBundle(t).Snapshot()
	data.Profile.GOOS = "linux"
	if runtime.GOOS == "linux" {
		data.Profile.GOOS = "darwin"
	}
	bundle, err := openapi.NewBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Mount(engine, bundle, Config{OpenAPI: openapi.Config{Title: "Profile", Version: "1"}})
	if err == nil || !strings.Contains(err.Error(), "openapi.build.mismatch") {
		t.Fatalf("mismatched runtime profile was accepted: %v", err)
	}
	after := engine.Routes()
	if len(before) != len(after) {
		t.Fatal("failed mount changed route count")
	}
	for i := range before {
		if reflect.ValueOf(before[i].HandlerFunc).Pointer() != reflect.ValueOf(after[i].HandlerFunc).Pointer() {
			t.Fatal("failed mount changed handler")
		}
		before[i].HandlerFunc, after[i].HandlerFunc = nil, nil
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed mount changed business routes")
	}
}
