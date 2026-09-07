// Verify independent modules, runtime dependency boundaries, and real first-generation CLI flows.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openapi-golang/openapi"
)

// Locate this adapter checkout without using neighboring product repositories.
func checkoutRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

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

// Ordinary Mount consumers must not link analyzers or the independent test engine.
func TestRuntimeDependencyBoundary(t *testing.T) {
	output := execute(t, checkoutRoot(t), "go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	for _, forbidden := range []string{"github.com/openapi-golang/gin-swagger/compiler", "github.com/openapi-golang/openapi/compiler", "github.com/openapi-golang/openapi/contracttest", "golang.org/x/tools/", "github.com/santhosh-tekuri/jsonschema/"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("runtime imports %s", forbidden)
		}
	}
}

// Reject module replacements and verify the core selected by Go matches the declared fixed dependency.
func TestNoLocalReplace(t *testing.T) {
	root := checkoutRoot(t)
	var manifest struct {
		Require []struct{ Path, Version string }
	}
	if err := json.Unmarshal([]byte(execute(t, root, "go", "mod", "edit", "-json")), &manifest); err != nil {
		t.Fatal(err)
	}
	pinned := ""
	for _, dependency := range manifest.Require {
		if dependency.Path == "github.com/openapi-golang/openapi" {
			pinned = dependency.Version
		}
	}
	if pinned == "" {
		t.Fatal("the adapter has no fixed core dependency")
	}
	decoder := json.NewDecoder(strings.NewReader(execute(t, root, "go", "list", "-m", "-json", "all")))
	found := false
	for {
		var module struct {
			Path, Version string
			Replace       *json.RawMessage
		}
		if err := decoder.Decode(&module); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if module.Replace != nil {
			t.Fatalf("module replacement bypasses independent consumption: %s", module.Path)
		}
		if module.Path == "github.com/openapi-golang/openapi" {
			found = true
			if module.Version != pinned {
				t.Fatalf("selected core %s differs from pinned %s", module.Version, pinned)
			}
		}
	}
	if !found {
		t.Fatal("the core is missing from the selected module graph")
	}
}

// Generate from real source, build a stripped application, and verify its public runtime and contracts.
func TestGeneratorBootstrap(t *testing.T) {
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
	sourceFiles := map[string][]byte{}
	// Copy the complete owned fixture tree so new consumer cases cannot silently disappear locally.
	copySource := func(name, source string) {
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".go") {
			sourceFiles[name] = append([]byte(nil), raw...)
		}
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	copySource("go.sum", filepath.Join(root, "go.sum"))
	fixture := filepath.Join("testdata", "external")
	if err := filepath.WalkDir(fixture, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("consumer fixture must contain regular files: %s", path)
		}
		name, err := filepath.Rel(fixture, path)
		if err != nil {
			return err
		}
		copySource(name, path)
		return nil
	}); err != nil {
		t.Fatal(err)
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
	for name, before := range sourceFiles {
		after, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("generation changed business source: %s", name)
		}
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
	// Build the real application with different tags and reject its stale Bundle before writing a document.
	mismatched := filepath.Join(t.TempDir(), "consumer-mismatched")
	execute(t, dir, "go", "build", "-tags=openapi_runtime_mismatch", "-trimpath", "-ldflags=-s -w", "-o", mismatched, ".")
	rejectedPath := filepath.Join(dir, "rejected.json")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, mismatched, rejectedPath)
	command.Dir = dir
	output, runError := command.CombinedOutput()
	if runError == nil || !strings.Contains(string(output), "openapi.build.mismatch") {
		t.Fatalf("runtime build mismatch was not rejected: %v\n%s", runError, output)
	}
	if _, err := os.Stat(rejectedPath); !os.IsNotExist(err) {
		t.Fatal("rejected build wrote an output document")
	}

}
