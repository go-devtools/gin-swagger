package ginswagger

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/spec"
)

// 整体分类分别构建独立文档，并沿用缓存与 HEAD 行为。
func TestDocumentGroups(t *testing.T) {
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/admin/:id", userHandler)
	cfg := Config{OpenAPI: openapi.Config{Title: "服务", Version: "1"}, Groups: []DocumentGroup{
		{ID: "all", Name: "全部接口"},
		{ID: "users", Name: "用户接口", Include: func(_, path string) bool { return strings.HasPrefix(path, "/users/") }},
	}}
	if _, err := Mount(r, runtimeBundle(t), cfg); err != nil {
		t.Fatal(err)
	}
	request := func(method, path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, httptest.NewRequest(method, path, nil))
		return rr
	}
	all := request("GET", "/docs/groups/all.json")
	users := request("GET", "/docs/groups/users.json")
	if all.Code != 200 || users.Code != 200 {
		t.Fatal("分类规范不可读取")
	}
	var a, b spec.OpenAPI
	if err := json.Unmarshal(all.Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(users.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if len(a.Paths) != 2 || len(b.Paths) != 1 || b.Paths["/users/{id}"] == nil {
		t.Fatal("分类未按实际路由范围构建")
	}
	if all.Header().Get("ETag") == users.Header().Get("ETag") {
		t.Fatal("分类缓存错误共享")
	}
	if got := request("HEAD", "/docs/groups/users.json"); got.Code != 200 || got.Body.Len() != 0 {
		t.Fatal("分类 HEAD 错误")
	}
	if got := request("GET", "/docs/groups/unknown.json"); got.Code != 404 {
		t.Fatal("未知分类应返回 404")
	}
	if got := request("GET", "/docs/config.js"); !strings.Contains(got.Body.String(), `"urls":[`) {
		t.Fatal("UI 未使用分类选择器")
	}
}

// 错误分类配置必须在注册文档路由前失败。
func TestInvalidGroupsLeaveRoutesUnchanged(t *testing.T) {
	for _, groups := range [][]DocumentGroup{{{ID: "../x", Name: "非法"}}, {{ID: "same", Name: "一"}, {ID: "same", Name: "二"}}, {{ID: "a", Name: "相同"}, {ID: "b", Name: "相同"}}} {
		r := gin.New()
		r.GET("/users/:id", userHandler)
		before := r.Routes()
		_, err := Mount(r, runtimeBundle(t), Config{OpenAPI: openapi.Config{Title: "服务", Version: "1"}, Groups: groups})
		if err == nil {
			t.Fatal("错误接受分类配置")
		}
		after := r.Routes()
		if len(before) != len(after) {
			t.Fatal("失败改变了路由数量")
		}
		for i := range before {
			if before[i].Method != after[i].Method || before[i].Path != after[i].Path || before[i].Handler != after[i].Handler {
				t.Fatal("失败改变了路由身份")
			}
		}
	}
}

// 分类不能重新包含被全局范围排除的路由，条件请求沿用各自的内容标识。
func TestGroupScopeIntersectionAndCache(t *testing.T) {
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/admin/:id", userHandler)
	cfg := Config{
		OpenAPI:      openapi.Config{Title: "服务", Version: "1"},
		Include:      func(_, path string) bool { return strings.HasPrefix(path, "/users/") },
		Groups:       []DocumentGroup{{ID: "all", Name: "允许范围", Include: func(_, _ string) bool { return true }}},
		DefaultGroup: "all",
	}
	if _, err := Mount(r, runtimeBundle(t), cfg); err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest("GET", "/docs/groups/all.json", nil))
	var doc spec.OpenAPI
	if err := json.Unmarshal(first.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Paths) != 1 || doc.Paths["/users/{id}"] == nil {
		t.Fatal("分类扩大了全局文档范围")
	}
	for _, method := range []string{"GET", "HEAD"} {
		req := httptest.NewRequest(method, "/docs/groups/all.json", nil)
		req.Header.Set("If-None-Match", first.Header().Get("ETag"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != 304 || rr.Body.Len() != 0 {
			t.Fatal("分类条件请求未返回无响应体的 304")
		}
	}
}
