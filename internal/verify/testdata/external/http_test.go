package main

import (
	"bytes"
	"encoding/json"
	"example.test/gin-consumer/internal/apidoc"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	"github.com/openapi-golang/openapi"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Reuse the Bundle generated from this consumer by the installed external CLI.
func httpResponseBundle(t *testing.T) openapi.Bundle { t.Helper(); return apidoc.Bundle() }

// Compare real HTTP responses before/after mounting with automatic redirect following disabled.
func TestHTTPRedirects(t *testing.T) {
	bundle := httpResponseBundle(t)
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		for _, sample := range []struct {
			name     string
			handler  gin.HandlerFunc
			status   int
			location string
			media    string
			body     bool
			constant bool
		}{
			{"absolute", RedirectAbsolute, 308, "https://example.test/a/../next?q=%2F", "text/html", true, true},
			{"not-modified", RedirectNotModified, 304, "/cached", "", false, true},
			{"deleted-media", RedirectDeletedMedia, 301, "/next/", "text/html", true, true},
			{"late-media", RedirectLateMedia, 409, "", "", false, false},
			{"whitespace", RedirectWhitespace, 302, "https://example.test/next", "text/html", true, true},
			{"root", RedirectRoot, 302, "/next?from=old", "text/html", true, true},
			{"created", RedirectCreated, 201, "/created", "text/html", true, true},
			{"unicode", RedirectUnicode, 307, "/%e7%9b%ae%e6%a0%87", "text/html", true, true},
			{"relative", RedirectRelative, 303, "/parent/next", "text/html", true, false},
			{"custom", RedirectCustom, 302, "/next", "text/plain", false, true},
			{"committed", RedirectCommitted, 409, "", "text/html", true, false},
			{"dynamic", RedirectDynamic, 302, "/dynamic", "text/html", true, false},
		} {
			t.Run(method+"/"+sample.name, func(t *testing.T) {
				path := "/parent/" + sample.name
				before, after := gin.New(), gin.New()
				before.Handle(method, path, sample.handler)
				after.Handle(method, path, sample.handler)
				doc, err := ginswagger.Mount(after, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "HTTP", Version: "1"}})
				if err != nil {
					t.Fatal(err)
				}
				var bodies [][]byte
				var statuses []int
				var headers []http.Header
				for _, engine := range []*gin.Engine{before, after} {
					server := httptest.NewServer(engine)
					defer server.Close()
					client := server.Client()
					client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
					req, err := http.NewRequest(method, server.URL+path+"?next=%2Fdynamic", nil)
					if err != nil {
						t.Fatal(err)
					}
					response, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					body, err := io.ReadAll(response.Body)
					response.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					response.Header.Del("Date")
					bodies = append(bodies, body)
					statuses = append(statuses, response.StatusCode)
					headers = append(headers, response.Header)
				}
				if !bytes.Equal(bodies[0], bodies[1]) || statuses[0] != statuses[1] || !reflect.DeepEqual(headers[0], headers[1]) {
					t.Fatal("Mount changed HTTP response")
				}
				if statuses[1] != sample.status || headers[1].Get("Location") != sample.location {
					t.Fatalf("actual status/location %d %v", statuses[1], headers[1])
				}
				hasBody := sample.body && method == "GET"
				if (len(bodies[1]) > 0) != hasBody {
					t.Fatalf("unexpected body: %q", bodies[1])
				}
				var document spec.OpenAPI
				if err = json.Unmarshal(doc.JSON(), &document); err != nil {
					t.Fatal(err)
				}
				op := httpOperation(document.Paths[path], method)
				response := op.Responses[httpStatus(sample.status)].Value
				if response == nil {
					t.Fatalf("wrong response statuses: %v", op.Responses)
				}
				if len(op.Responses) != 1 {
					t.Fatalf("invented response status: %v", op.Responses)
				}
				if (len(response.Content) > 0) != hasBody {
					t.Fatalf("wrong body contract: %#v", response.Content)
				}
				if sample.location != "" {
					header := response.Headers["Location"].Value
					if header == nil || header.Schema == nil {
						t.Fatal("missing Location schema")
					}
					raw, _ := json.Marshal(header.Schema)
					validator, err := contracttest.Compile(raw, "", contracttest.Options{})
					if err != nil {
						t.Fatal(err)
					}
					if err = validator.Value(sample.location); err != nil {
						t.Fatal(err)
					}
					if validator.Value(7) == nil {
						t.Fatal("Location lost string type")
					}
					if sample.constant && validator.Value("/wrong") == nil {
						t.Fatal("known Location lost exact value")
					}
				} else if len(response.Headers) != 0 {
					t.Fatal("late redirect changed committed headers")
				}
				if hasBody {
					if !strings.HasPrefix(headers[1].Get("Content-Type"), sample.media) {
						t.Fatalf("wrong media: %v", headers[1])
					}
					schema := response.Content[sample.media].Value
					if schema == nil {
						t.Fatal("missing HTML media")
					}
					raw, _ := json.Marshal(schema.Schema)
					validator, err := contracttest.Compile(raw, "", contracttest.Options{})
					if err != nil {
						t.Fatal(err)
					}
					if err = validator.Value(string(bodies[1])); err != nil {
						t.Fatal(err)
					}
					if validator.Value(7) == nil {
						t.Fatal("HTML lost type")
					}
				}
			})
		}
	}
}

// Read the operation for the actual HTTP method rather than substituting one GET result for the matrix.
func httpOperation(path *spec.PathItem, method string) *spec.Operation {
	switch method {
	case "GET":
		return path.Get
	case "HEAD":
		return path.Head
	case "POST":
		return path.Post
	case "PUT":
		return path.Put
	case "PATCH":
		return path.Patch
	case "DELETE":
		return path.Delete
	}
	return nil
}

// Convert known matrix statuses to document keys.
func httpStatus(status int) string {
	switch status {
	case 200:
		return "200"
	case 201:
		return "201"
	case 301:
		return "301"
	case 304:
		return "304"
	case 308:
		return "308"
	case 302:
		return "302"
	case 303:
		return "303"
	case 307:
		return "307"
	case 409:
		return "409"
	}
	return "invalid"
}

// Distinguish continued redirect writes from two complete bodies and isolate unselected invalid routes.
func TestRedirectContinuation(t *testing.T) {
	bundle := httpResponseBundle(t)
	for _, method := range []string{"POST", "GET"} {
		engine := gin.New()
		engine.Handle(method, "/continue", RedirectContinue)
		doc, err := ginswagger.Build(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Continue", Version: "1"}})
		if method == "GET" {
			if err == nil {
				t.Fatalf("mixed redirect body not diagnosed: %v", err)
			}
			record := httptest.NewRecorder()
			engine.ServeHTTP(record, httptest.NewRequest(method, "/continue", nil))
			if record.Code != 302 || !strings.Contains(record.Body.String(), "</a>.") || !strings.HasSuffix(record.Body.String(), "done") {
				t.Fatal("missing actual mixed redirect output")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		record := httptest.NewRecorder()
		engine.ServeHTTP(record, httptest.NewRequest(method, "/continue", nil))
		if record.Code != 201 || record.Body.String() != "done" {
			t.Fatal("redirect incorrectly committed POST")
		}
		var document spec.OpenAPI
		json.Unmarshal(doc.JSON(), &document)
		response := document.Paths["/continue"].Post.Responses["201"].Value
		if response == nil || response.Content["text/plain"].Value == nil {
			t.Fatal("lost post-redirect response")
		}
	}
	engine := gin.New()
	engine.GET("/invalid", RedirectInvalid)
	if _, err := ginswagger.Build(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Invalid", Version: "1"}}); err == nil || !strings.Contains(err.Error(), "redirect status") {
		t.Fatalf("invalid redirect status: %v", err)
	}
}

// A real HTTP server suppresses HEAD bodies while retaining GET representation for the same handler.
func TestHEADWireResponse(t *testing.T) {
	bundle := httpResponseBundle(t)
	engine := gin.New()
	engine.GET("/head", HeadJSON)
	engine.HEAD("/head", HeadJSON)
	doc, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "HEAD", Version: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(engine)
	defer server.Close()
	response, err := server.Client().Head(server.URL + "/head")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || len(raw) != 0 || response.StatusCode != 200 || response.Header.Get("X-Revision") != "1" {
		t.Fatal("wrong HEAD wire response")
	}
	var document spec.OpenAPI
	json.Unmarshal(doc.JSON(), &document)
	path := document.Paths["/head"]
	if len(path.Head.Responses["200"].Value.Content) != 0 || len(path.Get.Responses["200"].Value.Content) == 0 {
		t.Fatal("HEAD/GET representations were conflated")
	}
}
