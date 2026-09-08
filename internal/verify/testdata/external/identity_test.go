package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	gincompiler "github.com/go-devtools/gin-swagger/compiler"
	"github.com/go-devtools/openapi"
	core "github.com/go-devtools/openapi/compiler"
	"github.com/go-devtools/openapi/contracttest"
)

// Keep source identity tied to the independent consumer's actual module path.
const identityPackage = "example.test/gin-consumer"

// Nominate a known constructor through the public SDK and preserve its declared response provenance.
func identityBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	source, err := os.ReadFile("identity.go")
	if err != nil {
		t.Fatal(err)
	}
	frontend := gincompiler.Frontend()
	match := frontend.Match
	entry := frontend.Entry
	frontend.Name += "+declared-identity-factory-v1"
	frontend.Match = func(function core.Function) bool {
		return match(function) || function.Symbol == identityPackage+".IdentityFactory"
	}
	// A constructor returns a handler; it does not itself commit Gin's implicit empty response.
	frontend.Entry = func(function core.Function) []core.Effect {
		if function.Symbol == identityPackage+".IdentityFactory" {
			return nil
		}
		return entry(function)
	}
	result, err := core.Compile(context.Background(), core.Options{
		Load: core.LoadOptions{Dir: "."}, Frontends: []core.Frontend{frontend},
		Configuration: map[string]json.RawMessage{"factory": json.RawMessage("\"declared IdentityReply response; no constructor execution\"")},
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("identity.go")
	if err != nil || !bytes.Equal(source, after) {
		t.Fatal("generation changed source")
	}
	return result.Bundle
}

// Read the actual operation identifier without inferring it from source names.
func identityOperationID(t *testing.T, document *openapi.Document, path string) string {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(document.JSON(), &data); err != nil {
		t.Fatal(err)
	}
	return data["paths"].(map[string]any)[path].(map[string]any)["get"].(map[string]any)["operationId"].(string)
}

// Exercise actual shared code addresses, different captures, receivers, aliases and generic wrappers.
func TestHandlerIdentityEvidence(t *testing.T) {
	bundle := identityBundle(t)
	cfg := ginswagger.Config{OpenAPI: openapi.Config{Title: "Identity", Version: "1"}}
	prefix := "/identity"
	original, mounted := identityRouter(prefix), identityRouter(prefix)
	byPath := map[string]gin.RouteInfo{}
	for _, route := range original.Routes() {
		byPath[strings.TrimPrefix(route.Path, prefix)] = route
	}
	factoryA, factoryB := byPath["/factory/alpha"], byPath["/factory/beta"]
	if reflect.ValueOf(factoryA.HandlerFunc).Pointer() != reflect.ValueOf(factoryB.HandlerFunc).Pointer() || factoryA.Handler != factoryB.Handler {
		t.Fatal("fixture did not exercise different captures sharing runtime code")
	}
	if byPath["/value/alpha"].Handler != byPath["/value/beta"].Handler {
		t.Fatal("receiver fixture did not share a method wrapper")
	}
	for _, route := range original.Routes() {
		path := strings.TrimPrefix(route.Path, prefix)
		cfg.Include = func(method, actual string) bool { return actual == route.Path }
		_, err := ginswagger.Build(original, bundle, cfg)
		automatic := path == "/direct" || path == "/alias" || path == "/private"
		if automatic && err != nil {
			t.Fatalf("ordinary identity %s: %v", path, err)
		}
		if !automatic && (err == nil || !strings.Contains(err.Error(), "gin-swagger.handler.ambiguous")) {
			t.Fatalf("unsupported identity %s was guessed: %v", path, err)
		}
	}
	cfg.Bindings = map[string]openapi.OperationKey{}
	for path, symbol := range map[string]string{
		"/factory/alpha": identityPackage + ".IdentityFactory", "/factory/beta": identityPackage + ".IdentityFactory",
		"/value/alpha": identityPackage + ".IdentityReceiver.Value", "/value/beta": identityPackage + ".IdentityReceiver.Value",
		"/pointer/alpha": "*" + identityPackage + ".IdentityReceiver.Pointer", "/pointer/beta": "*" + identityPackage + ".IdentityReceiver.Pointer",
		"/generic/int": identityPackage + ".IdentityGeneric", "/generic/string": identityPackage + ".IdentityGeneric",
	} {
		found := false
		for _, template := range bundle.Index() {
			if template.Symbol == symbol {
				cfg.Bindings["GET "+prefix+path] = template.Key
				found = true
			}
		}
		if !found {
			t.Fatalf("missing actual template %s", symbol)
		}
	}
	cfg.Include = func(method, path string) bool { return !strings.Contains(path, "/unknown/") }
	document, err := ginswagger.Mount(mounted, bundle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(document.JSON(), []byte("UnusedPrivate")) {
		t.Fatal("unregistered model leaked into document")
	}
	for path, expected := range map[string]string{
		"/direct": "{\"Label\":\"plain\"}", "/alias": "{\"Label\":\"plain\"}", "/private": "{\"Label\":\"private\"}",
		"/factory/alpha": "{\"Label\":\"alpha\"}", "/factory/beta": "{\"Label\":\"beta\"}",
		"/value/alpha": "{\"Label\":\"value-alpha\"}", "/value/beta": "{\"Label\":\"value-beta\"}",
		"/pointer/alpha": "{\"Label\":\"pointer-alpha\"}", "/pointer/beta": "{\"Label\":\"pointer-beta\"}",
		"/generic/int": "int", "/generic/string": "string",
	} {
		t.Run(path, func(t *testing.T) {
			before, after := httptest.NewRecorder(), httptest.NewRecorder()
			original.ServeHTTP(before, httptest.NewRequest("GET", prefix+path, nil))
			mounted.ServeHTTP(after, httptest.NewRequest("GET", prefix+path, nil))
			if before.Code != 200 || after.Code != 200 || before.Body.String() != expected || after.Body.String() != expected || !reflect.DeepEqual(before.Header(), after.Header()) {
				t.Fatalf("identity or mounting changed response: %d %q vs %d %q", before.Code, before.Body.String(), after.Code, after.Body.String())
			}
			media := "application/json"
			if strings.HasPrefix(path, "/generic/") {
				media = "text/plain"
			}
			validateIdentityResponse(t, document.JSON(), prefix+path, media, 200, after.Body.Bytes())
		})
	}
	declared := false
	for _, fact := range document.Report().Facts {
		if fact.Symbol == identityPackage+".IdentityFactory" && fact.Kind == "declared" && fact.Rule == "openapi.comment.response" {
			declared = true
		}
	}
	if !declared {
		t.Fatal("factory declaration was misrepresented as derived execution")
	}
	for path, symbol := range map[string]string{"/unknown/status": identityPackage + ".IdentityReceiver.UnknownStatus", "/unknown/generic": identityPackage + ".IdentityGenericJSON", "/unknown/converted": identityPackage + ".IdentityGenericConverted"} {
		cfg.Include = func(method, actual string) bool { return actual == prefix+path }
		for _, template := range bundle.Index() {
			if template.Symbol == symbol {
				cfg.Bindings["GET "+prefix+path] = template.Key
			}
		}
		// Observe the real instance as well as rejecting its unresolved source template.
		actual := httptest.NewRecorder()
		original.ServeHTTP(actual, httptest.NewRequest("GET", prefix+path, nil))
		if path == "/unknown/generic" && (actual.Code != 200 || actual.Body.String() != "0") {
			t.Fatal("generic integer fixture must actually serialize zero, not null")
		}
		if path == "/unknown/converted" && (actual.Code != 200 || actual.Body.String() != "1") {
			t.Fatal("generic conversion fixture must actually serialize one")
		}
		if path == "/unknown/status" && actual.Code != 201 {
			t.Fatal("receiver fixture lost its captured status")
		}
		if _, err := ginswagger.Build(original, bundle, cfg); err == nil || strings.Contains(err.Error(), "gin-swagger.handler.ambiguous") {
			t.Fatalf("binding must retain unresolved behavior for %s: %v", path, err)
		}
	}
	ids := []string{}
	for _, handler := range []gin.HandlerFunc{IdentityPlain, IdentityRenamed} {
		engine := gin.New()
		engine.GET("/same", handler)
		doc, err := ginswagger.Build(engine, bundle, ginswagger.Config{OpenAPI: cfg.OpenAPI})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, identityOperationID(t, doc, "/same"))
	}
	if ids[0] != ids[1] {
		t.Fatal("operationId depends on the source function name")
	}
	// A child test process exports the real linked document for cross-build comparison.
	if output := os.Getenv("GIN_SWAGGER_IDENTITY_DOCUMENT"); output != "" {
		if err := os.WriteFile(output, document.JSON(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Shared factory code: %s; receiver wrapper: %s; generic evidence: %s / %s", runtime.FuncForPC(reflect.ValueOf(factoryA.HandlerFunc).Pointer()).Name(), byPath["/value/alpha"].Handler, byPath["/generic/int"].Handler, byPath["/generic/string"].Handler)
}

// Repeat the real router and HTTP contract matrix in independently linked binaries with changed symbol metadata.
func TestHandlerIdentityBuildModes(t *testing.T) {
	dir := t.TempDir()
	var baseline []byte
	for _, mode := range []struct {
		name  string
		flags []string
	}{
		{name: "ordinary"},
		{name: "trimpath", flags: []string{"-trimpath"}},
		{name: "stripped", flags: []string{"-ldflags=-s -w"}},
		{name: "trimpath-stripped", flags: []string{"-trimpath", "-ldflags=-s -w"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			binary := filepath.Join(dir, mode.name)
			output := binary + ".json"
			// Subprocesses use the same source and fixed modules, without workspace or inherited build flags.
			run := func(program string, args ...string) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()
				command := exec.CommandContext(ctx, program, args...)
				command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "GIN_SWAGGER_IDENTITY_DOCUMENT="+output)
				result, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("%s %v: %v\n%s", program, args, err, result)
				}
				t.Logf("%s", result)
			}
			args := append([]string{"test", "-c", "-o", binary}, mode.flags...)
			run("go", append(args, ".")...)
			run(binary, "-test.run=^TestHandlerIdentityEvidence$", "-test.v", "-test.count=1")
			actual, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if baseline == nil {
				baseline = actual
			} else if !bytes.Equal(actual, baseline) {
				t.Fatal("build metadata changed the document, component identities or operation IDs")
			}
		})
	}
}

// Validate actual JSON or text bytes with the independent engine and reject an incompatible payload.
func validateIdentityResponse(t *testing.T, document []byte, path, media string, status int, body []byte) {
	t.Helper()
	escape := func(value string) string { return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1") }
	pointer := fmt.Sprintf("/paths/%s/get/responses/%d/content/%s/schema", escape(path), status, escape(media))
	validator, err := contracttest.Compile(document, pointer, contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if media == "application/json" {
		err = validator.JSON(body)
	} else {
		err = validator.Value(string(body))
	}
	if err != nil {
		t.Fatal(err)
	}
	if validator.Value(7) == nil {
		t.Fatal("wire schema lost its type constraint")
	}
}
