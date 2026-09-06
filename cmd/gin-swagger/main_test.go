package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
	// Keep this fixture separate from the later fixed-remote independent-module acceptance test.
	// Omit the runtime package deliberately to verify failed bootstrap cleanup.
	var out, errs bytes.Buffer
	code := run(context.Background(), []string{"generate", "--dir", app, "--output", "./internal/apidoc"}, &out, &errs)
	if code == 0 {
		t.Fatal("fixture without a root package was accepted")
	}
	if _, err = os.Stat(filepath.Join(app, "internal/apidoc/zz_openapi.gen.go")); !os.IsNotExist(err) {
		t.Fatal("bootstrap Bundle remained after failure")
	}
}
