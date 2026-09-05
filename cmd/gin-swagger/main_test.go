package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 验证首次不存在生成目录时可以引导，失败不会留下空 Bundle。
// Test bootstrap failure cleanup when the generated directory is initially absent.
func TestBootstrapFailureRestoresOutput(t *testing.T) {
	root := t.TempDir()
	module, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../examples/basic/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "go.sum"), sum, 0600); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(root, "examples", "basic")
	if err = os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(app, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	// 临时消费模块无需复制运行时源码，使用真实固定远端模块仍属于后续独立验收。
	// Keep this fixture separate from the later fixed-remote independent-module acceptance test.
	// 此处只验证失败恢复：故意缺少同模块根运行时包，首次加载必须失败并移除引导文件。
	// Omit the runtime package deliberately to verify failed bootstrap cleanup.
	var out, errs bytes.Buffer
	code := run(context.Background(), []string{"generate", "--dir", app, "--output", "./internal/apidoc"}, &out, &errs)
	if code == 0 {
		t.Fatal("错误接受缺失根包的 fixture")
	}
	if _, err = os.Stat(filepath.Join(app, "internal/apidoc/zz_openapi.gen.go")); !os.IsNotExist(err) {
		t.Fatal("失败后留下引导 Bundle")
	}
}
