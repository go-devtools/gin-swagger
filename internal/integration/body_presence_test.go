package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/bodypresence"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Compare inferred body presence with real empty and nonempty requests without changing business source.
func TestBodyPresenceFromBindingOutcomes(t *testing.T) {
	path := "testdata/bodypresence/app.go"
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/bodypresence"}, Frontends: []core.Frontend{front.Frontend()}, Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("analysis modified business source")
	}
	for _, sample := range []struct {
		name            string
		handler         gin.HandlerFunc
		media           string
		required        bool
		empty, nonempty int
	}{
		{"checked", bodypresence.Checked, "application/json", true, 400, 201},
		{"ignored", bodypresence.Ignored, "application/json", false, 200, 200},
		{"overridden", bodypresence.Overridden, "application/json", false, 200, 200},
		{"empty-success", bodypresence.EmptySuccess, "application/json", false, 204, 201},
		{"aborted", bodypresence.Aborted, "application/json", false, 200, 200},
		{"mandatory", bodypresence.Mandatory, "application/json", true, 400, 201},
		{"cached", bodypresence.Cached, "application/json", true, 400, 201},
		{"explicit", bodypresence.Explicit, "application/json", true, 400, 201},
		{"form", bodypresence.Form, "application/x-www-form-urlencoded", false, 201, 201},
		{"multipart", bodypresence.Multipart, "multipart/form-data", true, 400, 201},
		{"ignored-multipart", bodypresence.IgnoredMultipart, "multipart/form-data", false, 200, 200},
		{"upload", bodypresence.Upload, "multipart/form-data", true, 400, 201},
		{"helper", bodypresence.Helper, "application/json", true, 400, 201},
		{"automatic-json", bodypresence.Automatic, "application/json", true, 400, 201},
		{"automatic-form", bodypresence.Automatic, "application/x-www-form-urlencoded", false, 201, 201},
		{"automatic-multipart", bodypresence.Automatic, "multipart/form-data", true, 400, 201},
	} {
		t.Run(sample.name, func(t *testing.T) {
			engine := gin.New()
			engine.POST("/value", sample.handler)
			doc, err := ginswagger.Build(engine, result.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Body presence", Version: "1"}, DefaultRequestMediaTypes: []string{sample.media}})
			if err != nil {
				t.Fatal(err)
			}
			var parsed spec.OpenAPI
			if err = json.Unmarshal(doc.JSON(), &parsed); err != nil {
				t.Fatal(err)
			}
			if sample.name == "checked" {
				explanation, err := result.Explain(core.ExplainQuery{Symbol: "github.com/openapi-golang/gin-swagger/internal/integration/testdata/bodypresence.Checked"})
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, use := range explanation.Uses {
					if use.NonEmptyBody && use.Source.Rule == "gin.ShouldBindJSON" && use.Source.Line > 0 {
						found = true
					}
				}
				if !found {
					t.Fatal("nonempty body proof lost source evidence")
				}
			}
			op := parsed.Paths["/value"].Post
			if op.RequestBody == nil || !op.RequestBody.Value.Required.Present || op.RequestBody.Value.Required.Value != sample.required {
				t.Fatalf("incorrect body presence: %s", doc.JSON())
			}
			for _, empty := range []bool{true, false} {
				body, media := bodyPresenceInput(t, sample.media, empty)
				req := httptest.NewRequest("POST", "/value", strings.NewReader(body))
				req.Header.Set("Content-Type", media)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, req)
				want := sample.nonempty
				if empty {
					want = sample.empty
				}
				if recorder.Code != want {
					t.Fatalf("empty=%t status=%d want=%d body=%s", empty, recorder.Code, want, recorder.Body)
				}
				response := op.Responses[fmt.Sprint(want)].Value
				if response == nil {
					t.Fatal("actual status is absent from the document")
				}
				if recorder.Body.Len() == 0 {
					if len(response.Content) != 0 {
						t.Fatal("bodyless error acquired a response body")
					}
					continue
				}
				if strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
					validator, err := contracttest.Compile(doc.JSON(), fmt.Sprintf("/paths/~1value/post/responses/%d/content/application~1json/schema", want), contracttest.Options{})
					if err != nil {
						t.Fatal(err)
					}
					if err = validator.JSON(recorder.Body.Bytes()); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
	// A single body-required boolean cannot represent contradictory media-dependent presence.
	for _, media := range [][]string{{"application/json", "application/x-www-form-urlencoded"}, {"application/json", "multipart/form-data"}} {
		engine := gin.New()
		engine.POST("/value", bodypresence.Automatic)
		_, err := ginswagger.Build(engine, result.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Body presence", Version: "1"}, DefaultRequestMediaTypes: media})
		if media[1] == "application/x-www-form-urlencoded" {
			if err == nil || !strings.Contains(err.Error(), "request body required differs") {
				t.Fatalf("lost presence ambiguity: %v", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

// Produce real JSON, URL-encoded, and multipart bodies, including genuinely empty streams.
func bodyPresenceInput(t *testing.T, media string, empty bool) (string, string) {
	t.Helper()
	if empty {
		if media == "multipart/form-data" {
			media += "; boundary=presence"
		}
		return "", media
	}
	switch media {
	case "application/json":
		return `{"Name":"Ada"}`, media
	case "application/x-www-form-urlencoded":
		return "Name=Ada", media
	default:
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("Name", "Ada"); err != nil {
			t.Fatal(err)
		}
		file, err := writer.CreateFormFile("asset", "input.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte("sample")); err != nil {
			t.Fatal(err)
		}
		if err = writer.Close(); err != nil {
			t.Fatal(err)
		}
		return body.String(), writer.FormDataContentType()
	}
}
