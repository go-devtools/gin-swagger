package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/constant"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/protocolbounds"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Declare the verified fixed-file profile through public SDK alternatives without changing its handlers.
func fixedFileFrontend() core.Frontend {
	frontend := front.Frontend()
	frontend.Name += "+declared-fixed-text-file-v1"
	original := frontend.CallOutcomes
	frontend.CallOutcomes = func(c core.CallContext) ([]core.CallOutcome, error) {
		if c.Object == nil || c.Object.Pkg() == nil || c.Object.Pkg().Path() != "github.com/gin-gonic/gin" || c.Function.Object.Pkg().Path() != "github.com/openapi-golang/gin-swagger/internal/integration/testdata/protocolbounds" {
			return original(c)
		}
		method := c.Object.FullName()
		if method != "(*github.com/gin-gonic/gin.Context).File" && method != "(*github.com/gin-gonic/gin.Context).FileAttachment" && method != "(*github.com/gin-gonic/gin.Context).FileFromFS" {
			return original(c)
		}
		if len(c.Arguments) == 0 || c.Arguments[0].Constant == nil {
			return original(c)
		}
		// Bind this declaration to exact source handlers, paths, filesystem root, and download name.
		expected := map[string]struct{ method, path, extra string }{
			"File":       {"File", "testdata/protocolbounds/report.txt", ""},
			"Attachment": {"FileAttachment", "testdata/protocolbounds/report.txt", "report.txt"},
			"FileSystem": {"FileFromFS", "report.txt", "testdata/protocolbounds"},
			"Missing":    {"File", "testdata/protocolbounds/missing.txt", ""},
		}[c.Function.Object.Name()]
		if expected.method != c.Object.Name() || constant.StringVal(c.Arguments[0].Constant) != expected.path {
			return original(c)
		}
		if expected.extra != "" && (len(c.Arguments) < 2 || c.Arguments[1].Constant == nil || c.Arguments[1].Constant.Kind() != constant.String || constant.StringVal(c.Arguments[1].Constant) != expected.extra) {
			return original(c)
		}

		source := c.Source
		source.Kind, source.Rule = "declared", "project.fixed-text-file.v1"
		var outcomes []core.CallOutcome
		for _, entry := range []struct{ status, media string }{{"200", "text/plain"}, {"206", "text/plain"}, {"206", "multipart/byteranges"}, {"304", ""}, {"412", ""}, {"400", "text/plain"}, {"403", "text/plain"}, {"404", "text/plain"}, {"416", "text/plain"}, {"500", "text/plain"}} {
			var effects []core.Effect
			if c.Object.Name() == "FileAttachment" {
				effects = append(effects, core.Effect{Kind: core.ResponseHeader, Name: "Content-Disposition", Payload: core.Value{Constant: constant.MakeString(`attachment; filename="report.txt"`)}, Source: source})
			}
			if entry.status == "200" || entry.status == "206" {
				effects = append(effects, core.Effect{Kind: core.ResponseHeader, Name: "Accept-Ranges", Payload: core.Value{Constant: constant.MakeString("bytes")}, Source: source})
			}
			if entry.status == "206" && entry.media == "text/plain" || entry.status == "416" {
				// Header schemas describe values when present; invalid-range errors need not include this header.
				effects = append(effects, core.Effect{Kind: core.ResponseHeader, Name: "Content-Range", Payload: core.Value{}, Source: source})
			}
			effect := core.Effect{Kind: core.ResponseStatus, Status: entry.status, Source: source}
			if entry.media != "" {
				effect.Kind, effect.MediaType, effect.WireSchema = core.ResponseBody, entry.media, spec.Typed("string")
				if entry.media == "multipart/byteranges" {
					effect.WireSchema = &spec.Schema{SchemaObject: &spec.SchemaObject{ContentMediaType: entry.media}}
				}
			}
			effects = append(effects, effect)
			outcomes = append(outcomes, core.CallOutcome{When: openapi.RequestCondition{Methods: []string{"GET", "HEAD"}}, Effects: effects})
		}
		outcomes = append(outcomes, core.CallOutcome{When: openapi.RequestCondition{ExceptMethods: []string{"GET", "HEAD"}}, Effects: []core.Effect{{Kind: core.Unresolved, Source: source, Message: "fixed file profile supports only GET and HEAD", Fix: "Declare and verify the additional request methods"}}})
		return outcomes, nil
	}
	return frontend
}

// Verify default diagnostics separately from a declared, bounded file-serving contract and actual wire samples.
func TestFileServingContractsAndBoundaries(t *testing.T) {
	defaults := protocolBoundaryBundle(t, front.Frontend(), nil)
	asset, err := os.ReadFile("testdata/protocolbounds/report.txt")
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := json.Marshal(map[string]any{"profile": "fixed-regular-utf8-text-v1", "assetSHA256": fmt.Sprintf("%x", sha256.Sum256(asset)), "methods": []string{"GET", "HEAD"}})
	if err != nil {
		t.Fatal(err)
	}
	adapted := protocolBoundaryBundle(t, fixedFileFrontend(), map[string]json.RawMessage{"fixed-file": configuration})
	for _, sample := range []struct {
		name    string
		handler gin.HandlerFunc
	}{{"file", protocolbounds.File}, {"attachment", protocolbounds.Attachment}, {"filesystem", protocolbounds.FileSystem}, {"missing", protocolbounds.Missing}} {
		t.Run(sample.name, func(t *testing.T) {
			before, after := gin.New(), gin.New()
			for _, engine := range []*gin.Engine{before, after} {
				engine.GET("/value", sample.handler)
				engine.HEAD("/value", sample.handler)
			}
			cfg := ginswagger.Config{OpenAPI: openapi.Config{Title: "Fixed text file", Version: "1"}}
			_, err := ginswagger.Build(after, defaults.Bundle, cfg)
			if err == nil || !strings.Contains(err.Error(), "complete centralized file contract") {
				t.Fatalf("file semantics were guessed: %v", err)
			}
			doc, err := ginswagger.Mount(after, adapted.Bundle, cfg)
			if err != nil {
				t.Fatal(err)
			}
			declared := false
			for _, fact := range doc.Report().Facts {
				if fact.Kind == "declared" && fact.Rule == "project.fixed-text-file.v1" && fact.Line > 0 {
					declared = true
				}
			}
			if !declared {
				t.Fatal("central file declaration lost its provenance")
			}
			var parsed spec.OpenAPI
			if err := json.Unmarshal(doc.JSON(), &parsed); err != nil {
				t.Fatal(err)
			}
			for _, response := range parsed.Paths["/value"].Head.Responses {
				if response.Value == nil || len(response.Value.Content) != 0 {
					t.Fatal("HEAD contract retained a body")
				}
			}
			servers := []*httptest.Server{httptest.NewServer(before), httptest.NewServer(after)}
			defer servers[0].Close()
			defer servers[1].Close()
			for _, method := range []string{"GET", "HEAD"} {
				cases := []struct {
					name    string
					headers map[string]string
					status  int
					body    string
				}{
					{"full", nil, 200, string(asset)},
					{"single", map[string]string{"Range": "bytes=1-3"}, 206, "123"},
					{"suffix", map[string]string{"Range": "bytes=-3"}, 206, "ef\n"},
					{"multiple", map[string]string{"Range": "bytes=0-1,4-5"}, 206, ""},
					{"unsatisfiable", map[string]string{"Range": "bytes=999-1000"}, 416, "invalid range: failed to overlap\n"},
					{"invalid-range", map[string]string{"Range": "invalid"}, 416, "invalid range\n"},
					{"not-modified", map[string]string{"If-None-Match": "*"}, 304, ""},
					{"precondition", map[string]string{"If-Match": `"unknown"`}, 412, ""},
					{"if-range", map[string]string{"If-Range": `"unknown"`, "Range": "bytes=1-3"}, 200, string(asset)},
				}
				if sample.name == "missing" {
					cases = cases[:1]
					cases[0].status, cases[0].body = 404, "404 page not found\n"
				}
				for _, scenario := range cases {
					t.Run(method+"/"+scenario.name, func(t *testing.T) {
						var snapshots []fileSnapshot
						for _, server := range servers {
							snapshots = append(snapshots, fileRequest(t, server, method, scenario.headers))
						}
						if !reflect.DeepEqual(snapshots[0], snapshots[1]) {
							t.Fatalf("mount changed file response: before=%+v after=%+v", snapshots[0], snapshots[1])
						}
						got := snapshots[1]
						if got.status != scenario.status {
							t.Fatalf("status %d want %d", got.status, scenario.status)
						}
						if method == "HEAD" {
							if got.body != "" || len(got.parts) != 0 {
								t.Fatal("HEAD wrote bytes on the actual HTTP wire")
							}
							return
						}
						if scenario.name == "multiple" {
							if !reflect.DeepEqual(got.parts, []string{"bytes 0-1/17:01", "bytes 4-5/17:45"}) {
								t.Fatalf("wrong multipart ranges: %v", got.parts)
							}
							if got.media != "multipart/byteranges" {
								t.Fatal("range multipart media was lost")
							}
						} else if scenario.body != "" || got.status == 304 || got.status == 412 {
							if got.body != scenario.body {
								t.Fatalf("wrong bytes: %q want %q", got.body, scenario.body)
							}
						}
						if value := got.headers.Get("Content-Range"); value != "" {
							validator, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1value/get/responses/%d/headers/Content-Range/schema", got.status), contracttest.Options{})
							if err != nil {
								t.Fatal(err)
							}
							if err = validator.Value(value); err != nil {
								t.Fatal(err)
							}
						}
						if scenario.name == "single" && got.headers.Get("Content-Range") != "bytes 1-3/17" {
							t.Fatal("lost content range")
						}
						if scenario.name == "unsatisfiable" && got.headers.Get("Content-Range") != "bytes */17" {
							t.Fatal("lost unsatisfiable range length")
						}
						if sample.name == "attachment" && got.headers.Get("Content-Disposition") != `attachment; filename="report.txt"` {
							t.Fatal("lost attachment filename")
						}
						if got.media == "text/plain" {
							validator, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1value/get/responses/%d/content/text~1plain/schema", got.status), contracttest.Options{})
							if err != nil {
								t.Fatal(err)
							}
							if err = validator.Value(got.body); err != nil {
								t.Fatal(err)
							}
						}
					})
				}
			}
		})
	}
	unmapped := gin.New()
	unmapped.GET("/value", protocolbounds.Unmapped)
	if _, err := ginswagger.Build(unmapped, adapted.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Unmapped file", Version: "1"}}); err == nil || !strings.Contains(err.Error(), "complete centralized file contract") {
		t.Fatalf("file rule widened its source or path scope: %v", err)
	}
	engine := gin.New()
	engine.POST("/value", protocolbounds.File)
	if _, err := ginswagger.Build(engine, adapted.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Out of scope", Version: "1"}}); err == nil || !strings.Contains(err.Error(), "only GET and HEAD") {
		t.Fatalf("declared profile silently widened method scope: %v", err)
	}
}

// Keep semantic file response fields while normalizing only server dates and random multipart boundaries.
type fileSnapshot struct {
	status      int
	media, body string
	headers     http.Header
	parts       []string
}

// Read an actual HTTP response and parse byte-range multipart framing independently of the compiler.
func fileRequest(t *testing.T, server *httptest.Server, method string, headers map[string]string) fileSnapshot {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+"/value", nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	client := server.Client()
	client.Timeout = 5 * time.Second
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	media, parameters, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.Header.Get("Content-Type") != "" && err != nil {
		t.Fatal(err)
	}
	snapshot := fileSnapshot{status: response.StatusCode, media: media, body: string(raw), headers: response.Header.Clone()}
	snapshot.headers.Del("Date")
	if media == "multipart/byteranges" {
		snapshot.headers.Set("Content-Type", media)
		if method == "HEAD" {
			return snapshot
		}
		reader := multipart.NewReader(bytes.NewReader(raw), parameters["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				t.Fatal(err)
			}
			if part.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatal("range part lost its original media type")
			}
			snapshot.parts = append(snapshot.parts, part.Header.Get("Content-Range")+":"+string(data))
		}
		snapshot.body = ""
	}
	return snapshot
}
