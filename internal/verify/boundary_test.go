// 验证独立 module、运行时依赖边界和真实 CLI 首次生成链路。
// Verify independent modules, runtime dependency boundaries, and real first-generation CLI flows.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openapi-golang/openapi"
)

// 定位当前适配器 checkout，不借用相邻产品仓库。
// Locate this adapter checkout without using neighboring product repositories.
func checkoutRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// 每个子进程有时间预算并关闭 workspace；失败保留完整输出。
// Bound each subprocess, disable workspaces, and preserve full output on failure.
func execute(t *testing.T, dir, program string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=", "GIN_MODE=release")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", program, args, err, output)
	}
	return string(output)
}

// 普通 Mount 用户不能链接分析器或独立测试引擎。
// Ordinary Mount consumers must not link analyzers or the independent test engine.
func TestRuntimeDependencyBoundary(t *testing.T) {
	output := execute(t, checkoutRoot(t), "go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	for _, forbidden := range []string{"github.com/openapi-golang/gin-swagger/compiler", "github.com/openapi-golang/openapi/compiler", "github.com/openapi-golang/openapi/contracttest", "golang.org/x/tools/", "github.com/santhosh-tekuri/jsonschema/"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("runtime imports %s", forbidden)
		}
	}
}

// 从真实源码首次生成、构建裁剪符号的程序并验证公开运行时与契约。
// Generate from real source, build a stripped application, and verify its public runtime and contracts.
func TestExternalModule(t *testing.T) {
	root, dir := checkoutRoot(t), t.TempDir()
	version := os.Getenv("GIN_SWAGGER_TEST_VERSION")
	remote := version != ""
	if !remote {
		version = "v0.0.0"
	}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	module = bytes.Replace(module, []byte("module github.com/openapi-golang/gin-swagger"), []byte("module example.test/gin-consumer"), 1)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.sum", "main.go", "main_test.go"} {
		source := filepath.Join(root, name)
		if name != "go.sum" {
			source = filepath.Join("testdata", "external", name)
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	execute(t, dir, "go", "mod", "edit", "-require=github.com/openapi-golang/gin-swagger@"+version)
	if !remote {
		execute(t, dir, "go", "mod", "edit", "-replace=github.com/openapi-golang/gin-swagger="+root)
		t.Log("development consumer uses this adapter checkout; core remains a fixed remote dependency")
	} else {
		execute(t, dir, "go", "mod", "download", "github.com/openapi-golang/gin-swagger@"+version)
	}
	var dependency struct {
		Version string
		Replace *struct{}
	}
	if err := json.Unmarshal([]byte(execute(t, dir, "go", "list", "-m", "-json", "github.com/openapi-golang/gin-swagger")), &dependency); err != nil {
		t.Fatal(err)
	}
	if remote && (dependency.Version != version || dependency.Replace != nil) {
		t.Fatal("remote consumer contains replacement or wrong version")
	}
	dependency = struct {
		Version string
		Replace *struct{}
	}{}
	if err := json.Unmarshal([]byte(execute(t, dir, "go", "list", "-m", "-json", "github.com/openapi-golang/openapi")), &dependency); err != nil {
		t.Fatal(err)
	}
	if dependency.Replace != nil {
		t.Fatal("core dependency uses a local replacement")
	}
	cli := filepath.Join(t.TempDir(), "gin-swagger")
	execute(t, root, "go", "build", "-o", cli, "./cmd/gin-swagger")
	generated := filepath.Join(dir, "internal", "apidoc", "zz_openapi.gen.go")
	if _, err := os.Stat(generated); !os.IsNotExist(err) {
		t.Fatal("first-generation fixture already contains generated output")
	}
	before, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	execute(t, dir, cli, "generate", "--dir", dir, "--timeout=2m")
	first, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	execute(t, dir, cli, "check", "--dir", dir, "--timeout=2m")
	execute(t, dir, cli, "generate", "--dir", dir, "--timeout=2m")
	second, err := os.ReadFile(generated)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("repeated generation changed its output")
	}
	after, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("generation changed business source")
	}
	t.Log(execute(t, dir, "go", "test", "-count=1", "-v", "./..."))
	application := filepath.Join(t.TempDir(), "consumer")
	execute(t, dir, "go", "build", "-trimpath", "-ldflags=-s -w", "-o", application, ".")
	specPath := filepath.Join(dir, "openapi.json")
	execute(t, dir, application, specPath)
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if report := openapi.Check(raw); report.HasErrors() {
		t.Fatal(report)
	}
	execute(t, dir, cli, "check", "--spec", specPath)
}
