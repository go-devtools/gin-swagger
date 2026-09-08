package ginswagger

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-devtools/openapi"
	"github.com/go-devtools/openapi/spec"
)

// Build independent grouped documents while preserving caching and HEAD behavior.
func TestDocumentGroups(t *testing.T) {
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/admin/:id", userHandler)
	cfg := Config{OpenAPI: openapi.Config{Title: "Service", Version: "1"}, Groups: []DocumentGroup{
		{ID: "all", Name: "All endpoints"},
		{ID: "users", Name: "User endpoints", Include: func(_, path string) bool { return strings.HasPrefix(path, "/users/") }},
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
		t.Fatal("group specification could not be read")
	}
	var a, b spec.OpenAPI
	if err := json.Unmarshal(all.Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(users.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if len(a.Paths) != 2 || len(b.Paths) != 1 || b.Paths["/users/{id}"] == nil {
		t.Fatal("group was not built from the actual route scope")
	}
	if all.Header().Get("ETag") == users.Header().Get("ETag") {
		t.Fatal("group cache is incorrectly shared")
	}
	if got := request("HEAD", "/docs/groups/users.json"); got.Code != 200 || got.Body.Len() != 0 {
		t.Fatal("group HEAD response is incorrect")
	}
	if got := request("GET", "/docs/groups/unknown.json"); got.Code != 404 {
		t.Fatal("unknown group must return 404")
	}
	if got := request("GET", "/docs/config.js"); !strings.Contains(got.Body.String(), `"urls":[`) {
		t.Fatal("UI did not use the definition selector")
	}
}

// Reject invalid group options before registering documentation routes.
func TestInvalidGroupsLeaveRoutesUnchanged(t *testing.T) {
	for _, groups := range [][]DocumentGroup{{{ID: "../x", Name: "Invalid"}}, {{ID: "same", Name: "One"}, {ID: "same", Name: "Two"}}, {{ID: "a", Name: "Same"}, {ID: "b", Name: "Same"}}} {
		r := gin.New()
		r.GET("/users/:id", userHandler)
		before := r.Routes()
		_, err := Mount(r, runtimeBundle(t), Config{OpenAPI: openapi.Config{Title: "Service", Version: "1"}, Groups: groups})
		if err == nil {
			t.Fatal("invalid group configuration was accepted")
		}
		after := r.Routes()
		if len(before) != len(after) {
			t.Fatal("failure changed the route count")
		}
		for i := range before {
			if before[i].Method != after[i].Method || before[i].Path != after[i].Path || before[i].Handler != after[i].Handler {
				t.Fatal("failure changed route identity")
			}
		}
	}
}

// Prevent groups from restoring globally excluded routes and retain per-document validators for conditional requests.
func TestGroupScopeIntersectionAndCache(t *testing.T) {
	r := gin.New()
	r.GET("/users/:id", userHandler)
	r.GET("/admin/:id", userHandler)
	cfg := Config{
		OpenAPI:      openapi.Config{Title: "Service", Version: "1"},
		Include:      func(_, path string) bool { return strings.HasPrefix(path, "/users/") },
		Groups:       []DocumentGroup{{ID: "all", Name: "Allowed scope", Include: func(_, _ string) bool { return true }}},
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
		t.Fatal("group expanded the global document scope")
	}
	for _, method := range []string{"GET", "HEAD"} {
		req := httptest.NewRequest(method, "/docs/groups/all.json", nil)
		req.Header.Set("If-None-Match", first.Header().Get("ETag"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != 304 || rr.Body.Len() != 0 {
			t.Fatal("group conditional request did not return a bodyless 304")
		}
	}
}
