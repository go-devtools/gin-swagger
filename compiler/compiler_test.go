package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
)

// 验证 Gin 规则经公开 SDK 编译真实零 tag handler。
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
	module = []byte(strings.Replace(string(module), "module github.com/openapi-golang/gin-swagger", "module example.com/gin-fixture", 1))
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0600); err != nil {
		t.Fatal(err)
	}
	source := `package sample
import "github.com/gin-gonic/gin"
// 创建信息。
type Request struct {
 // 用户名。
 // @openapi required minLength=3
 Name string
}
// 响应信息。
type User struct { ID int64; Name string }
// 错误信息。
type APIError struct { Message string }
// 创建用户
func Create(c *gin.Context) {
 var req Request
 if err:=c.ShouldBindJSON(&req);err!=nil { c.JSON(400,APIError{Message:"无效请求"});return }
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
		t.Fatalf("候选识别错误：%+v", index)
	}
	doc, err := openapi.Build(result.Bundle, []openapi.Route{{Method: "POST", Path: "/users", OperationKey: index[0].Key}}, openapi.Config{Title: "用户服务", Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"400"`, `"201"`, `"Name"`, `"minLength": 3`, "创建用户"} {
		if !strings.Contains(string(doc.JSON()), want) {
			t.Fatalf("缺少 %s：%s", want, doc.JSON())
		}
	}
}
