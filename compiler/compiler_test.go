package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-devtools/openapi"
	core "github.com/go-devtools/openapi/compiler"
)

// Compile actual tag-free Gin handlers through the public SDK.
func TestGinFrontendFromSource(t *testing.T) {
	dir := t.TempDir()
	module, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile("../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	module = []byte(strings.Replace(string(module), "module github.com/go-devtools/gin-swagger", "module example.com/gin-fixture", 1))
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0600); err != nil {
		t.Fatal(err)
	}
	source := `package sample
import "github.com/gin-gonic/gin"
// Describe creation input.
type Request struct {
 // User name.
 // @openapi required minLength=3
 Name string
}
// Describe response data.
type User struct { ID int64; Name string }
// Error information.
type APIError struct { Message string }
// Create a user
func Create(c *gin.Context) {
 var req Request
 if err:=c.ShouldBindJSON(&req);err!=nil { c.JSON(400,APIError{Message:"Invalid request"});return }
 c.JSON(201,User{ID:1,Name:req.Name})
}
`
	if err = os.WriteFile(filepath.Join(dir, "api.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: dir, Env: []string{"GOWORK=off"}}, Frontends: []core.Frontend{Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	index := result.Bundle.Index()
	if len(index) != 1 {
		t.Fatalf("candidate recognition failed: %+v", index)
	}
	doc, err := openapi.Build(result.Bundle, []openapi.Route{{Method: "POST", Path: "/users", OperationKey: index[0].Key}}, openapi.Config{Title: "User service", Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"400"`, `"201"`, `"Name"`, `"minLength": 3`, "Create a user"} {
		if !strings.Contains(string(doc.JSON()), want) {
			t.Fatalf("missing %s: %s", want, doc.JSON())
		}
	}
}
