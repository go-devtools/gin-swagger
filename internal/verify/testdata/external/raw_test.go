package main

import (
	"bytes"
	"encoding/json"
	"example.test/gin-consumer/internal/apidoc"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	"github.com/go-devtools/openapi"
	"github.com/go-devtools/openapi/contracttest"
	"github.com/go-devtools/openapi/spec"
)

// Use the Bundle generated from this consumer's actual source by the external CLI.
func rawRequestBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	return apidoc.Bundle()
}

// Assert real URL-encoded and multipart reads and independently validate the body representation.
func TestRawFormGetters(t *testing.T) {
	bundle := rawRequestBundle(t)
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		for _, multipartBody := range []bool{false, true} {
			t.Run(method+"/"+map[bool]string{false: "urlencoded", true: "multipart"}[multipartBody], func(t *testing.T) {
				var body bytes.Buffer
				media := "application/x-www-form-urlencoded"
				fields := [][2]string{{"title", "body"}, {"email", ""}, {"labels", "first"}, {"labels", "second"}, {"values", "one"}, {"values", "two"}}
				if multipartBody {
					writer := multipart.NewWriter(&body)
					for _, field := range fields {
						if err := writer.WriteField(field[0], field[1]); err != nil {
							t.Fatal(err)
						}
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					media = writer.FormDataContentType()
				} else {
					body.WriteString("title=body&email=&labels=first&labels=second&values=one&values=two")
				}
				before, after := Router(), Router()
				doc, err := ginswagger.Mount(after, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Raw forms", Version: "1"}, Include: func(m, p string) bool { return m == method && p == "/form" }})
				if err != nil {
					t.Fatal(err)
				}
				var records []*httptest.ResponseRecorder
				for _, engine := range []*gin.Engine{before, after} {
					req := httptest.NewRequest(method, "/form?title=query&alias=query", bytes.NewReader(body.Bytes()))
					req.Header.Set("Content-Type", media)
					record := httptest.NewRecorder()
					engine.ServeHTTP(record, req)
					records = append(records, record)
				}
				if records[0].Code != 200 || records[1].Code != 200 || !bytes.Equal(records[0].Body.Bytes(), records[1].Body.Bytes()) || !reflect.DeepEqual(records[0].Header(), records[1].Header()) {
					t.Fatal("mount changed raw-form behavior")
				}
				var actual FormResult
				if err = json.Unmarshal(records[1].Body.Bytes(), &actual); err != nil {
					t.Fatal(err)
				}
				reads := multipartBody || method == "POST" || method == "PUT" || method == "PATCH"
				if actual.Alias != "guest" {
					t.Fatal("query fallback was invented", actual)
				}
				if reads {
					if actual.Title != "body" || actual.Email != "" || !actual.Present || !actual.HasValues || !reflect.DeepEqual(actual.Labels, []string{"first", "second"}) || !reflect.DeepEqual(actual.Values, []string{"one", "two"}) {
						t.Fatalf("wrong actual read: %+v", actual)
					}
				} else if actual.Title != "" || actual.Present || actual.HasValues || len(actual.Labels) != 0 {
					t.Fatalf("unexpected URL-encoded body read: %+v", actual)
				}
				var parsed spec.OpenAPI
				if err = json.Unmarshal(doc.JSON(), &parsed); err != nil {
					t.Fatal(err)
				}
				var operation *spec.Operation
				item := parsed.Paths["/form"]
				switch method {
				case "GET":
					operation = item.Get
				case "POST":
					operation = item.Post
				case "PUT":
					operation = item.Put
				case "PATCH":
					operation = item.Patch
				case "DELETE":
					operation = item.Delete
				}
				if len(operation.Parameters) != 0 {
					t.Fatal("raw body reads became query parameters")
				}
				content := operation.RequestBody.Value.Content
				if _, hasURL := content["application/x-www-form-urlencoded"]; hasURL != (method == "POST" || method == "PUT" || method == "PATCH") {
					t.Fatal("wrong method-specific body media")
				}
				response, err := contracttest.Compile(doc.JSON(), "/paths/~1form/"+strings.ToLower(method)+"/responses/200/content/application~1json/schema", contracttest.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err = response.JSON(records[1].Body.Bytes()); err != nil {
					t.Fatal(err)
				}
				for _, kind := range []string{"multipart/form-data", "application/x-www-form-urlencoded"} {
					entry, ok := content[kind]
					if !ok {
						continue
					}
					if entry.Value.Schema.SchemaObject == nil || len(entry.Value.Schema.Properties) != 5 {
						t.Fatalf("missing combined fields: %+v", entry.Value.Schema)
					}
					input, err := contracttest.Compile(doc.JSON(), "/paths/~1form/"+strings.ToLower(method)+"/requestBody/content/"+strings.ReplaceAll(kind, "/", "~1")+"/schema", contracttest.Options{})
					if err != nil {
						t.Fatal(err)
					}
					if err = input.JSON([]byte(`{"title":"body","email":"","labels":["first","second"],"values":["one","two"]}`)); err != nil {
						t.Fatal(err)
					}
					if input.JSON([]byte(`{"labels":null}`)) == nil || input.JSON([]byte(`{"title":12}`)) == nil {
						t.Fatal("raw form wire types were lost")
					}
					if entry.Value.Encoding["labels"].Style != "form" || !entry.Value.Encoding["labels"].Explode.Value {
						t.Fatal("repeated form values lost encoding")
					}
				}
			})
		}
	}
}

// Upload raw binary data and missing files to verify error branches and FileHeader result correlation.
func TestRawFormFile(t *testing.T) {
	bundle := rawRequestBundle(t)
	engine := Router()
	doc, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Upload", Version: "1"}, Include: func(_, p string) bool { return p == "/upload" }})
	if err != nil {
		t.Fatal(err)
	}
	for _, withFile := range []bool{true, false} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err = writer.WriteField("caption", "report"); err != nil {
			t.Fatal(err)
		}
		if withFile {
			for _, name := range []string{"first.bin", "second.bin"} {
				part, e := writer.CreateFormFile("asset", name)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = part.Write([]byte{0, 255, 1}); e != nil {
					t.Fatal(e)
				}
			}
		}
		if err = writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/upload?caption=ignored", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		record := httptest.NewRecorder()
		engine.ServeHTTP(record, req)
		if withFile {
			if record.Code != 201 || record.Body.String() != `{"Filename":"first.bin","Size":3,"Caption":"report"}` {
				t.Fatal(record.Code, record.Body)
			}
			validator, e := contracttest.Compile(doc.JSON(), "/paths/~1upload/post/responses/201/content/application~1json/schema", contracttest.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if e = validator.JSON(record.Body.Bytes()); e != nil {
				t.Fatal(e)
			}
		} else if record.Code != 400 || record.Body.String() != "missing file" {
			t.Fatal(record.Code, record.Body)
		}
	}
	var parsed spec.OpenAPI
	if err = json.Unmarshal(doc.JSON(), &parsed); err != nil {
		t.Fatal(err)
	}
	operation := parsed.Paths["/upload"].Post
	if len(operation.Responses) != 2 || operation.Responses["200"].Value != nil {
		t.Fatal("invented upload response")
	}
	entry := operation.RequestBody.Value.Content["multipart/form-data"].Value
	raw, _ := json.Marshal(entry.Schema)
	if bytes.Contains(raw, []byte("FileHeader")) || bytes.Contains(raw, []byte("base64")) || !bytes.Contains(raw, []byte("application/octet-stream")) {
		t.Fatalf("wrong uploaded-file schema: %s", raw)
	}
}

// Repeated query values use form/explode with non-null input arrays while absent results retain actual JSON behavior.
func TestRawQueryArray(t *testing.T) {
	engine := Router()
	doc, err := ginswagger.Mount(engine, rawRequestBundle(t), ginswagger.Config{OpenAPI: openapi.Config{Title: "Query", Version: "1"}, Include: func(_, p string) bool { return p == "/query" }})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/query?values=first&values=second", "/query"} {
		record := httptest.NewRecorder()
		engine.ServeHTTP(record, httptest.NewRequest("GET", target, nil))
		var result FormResult
		if err = json.Unmarshal(record.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if target == "/query" {
			if result.HasValues || result.Values != nil {
				t.Fatal(result)
			}
		} else if !result.HasValues || !reflect.DeepEqual(result.Values, []string{"first", "second"}) {
			t.Fatal(result)
		}
	}
	var parsed spec.OpenAPI
	if err = json.Unmarshal(doc.JSON(), &parsed); err != nil {
		t.Fatal(err)
	}
	parameter := parsed.Paths["/query"].Get.Parameters[0].Value
	if parameter.Name != "values" || parameter.In != "query" || parameter.Style != "form" || !parameter.Explode.Value {
		t.Fatalf("wrong repeated query contract: %+v", parameter)
	}
	input, err := contracttest.Compile(doc.JSON(), "/paths/~1query/get/parameters/0/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err = input.JSON([]byte(`["first","second"]`)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"null", `"first,second"`, `[1]`} {
		if input.JSON([]byte(bad)) == nil {
			t.Fatal("invalid repeated query contract", bad)
		}
	}
}

// Read bracket-key dictionaries using the first repeated value without mixing body and query sources.
func TestRawDictionaryGetters(t *testing.T) {
	bundle := rawRequestBundle(t)
	for _, multipartBody := range []bool{false, true} {
		engine := gin.New()
		engine.POST("/maps", Maps)
		doc, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Maps", Version: "1"}})
		if err != nil {
			t.Fatal(err)
		}
		var body bytes.Buffer
		media := "application/x-www-form-urlencoded"
		if multipartBody {
			writer := multipart.NewWriter(&body)
			for _, field := range [][2]string{{"filter[role]", "editor"}, {"filter[role]", "admin"}, {"bare[name]", "body"}} {
				if err = writer.WriteField(field[0], field[1]); err != nil {
					t.Fatal(err)
				}
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			media = writer.FormDataContentType()
		} else {
			body.WriteString("filter%5Brole%5D=editor&filter%5Brole%5D=admin&bare%5Bname%5D=body")
		}
		request := httptest.NewRequest("POST", "/maps?query%5Bname%5D=Ada&bareq%5Brole%5D=reader&filter%5Brole%5D=ignored", bytes.NewReader(body.Bytes()))
		request.Header.Set("Content-Type", media)
		record := httptest.NewRecorder()
		engine.ServeHTTP(record, request)
		var actual MapResult
		if err = json.Unmarshal(record.Body.Bytes(), &actual); err != nil {
			t.Fatal(err)
		}
		if actual.Form["role"] != "editor" || actual.Bare["name"] != "body" || actual.Query["name"] != "Ada" || actual.BareQuery["role"] != "reader" || !actual.FormPresent || !actual.QueryPresent {
			t.Fatal(actual)
		}
		var parsed spec.OpenAPI
		if err = json.Unmarshal(doc.JSON(), &parsed); err != nil {
			t.Fatal(err)
		}
		operation := parsed.Paths["/maps"].Post
		if len(operation.Parameters) != 2 {
			t.Fatal("lost dictionary query parameters")
		}
		for _, p := range operation.Parameters {
			if p.Value.In != "query" || p.Value.Style != "deepObject" || !p.Value.Explode.Value {
				t.Fatal(p)
			}
		}
		for _, entry := range operation.RequestBody.Value.Content {
			if entry.Value.Encoding["filter"].Style != "deepObject" || !entry.Value.Encoding["filter"].Explode.Value {
				t.Fatal("lost dictionary body encoding")
			}
		}
		input, err := contracttest.Compile(doc.JSON(), "/paths/~1maps/post/requestBody/content/application~1x-www-form-urlencoded/schema", contracttest.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err = input.JSON([]byte(`{"filter":{"role":"editor"},"bare":{"name":"body"}}`)); err != nil {
			t.Fatal(err)
		}
		if input.JSON([]byte(`{"filter":{"role":1}}`)) == nil || input.JSON([]byte(`{"filter":null}`)) == nil {
			t.Fatal("invalid dictionary representation was accepted")
		}
	}
}

// File saving produces only business-selected statuses, and compilation must not execute filesystem writes.
func TestRawFileSaving(t *testing.T) {
	bundle := rawRequestBundle(t)
	for _, invalidDestination := range []bool{false, true} {
		var document *openapi.Document
		for _, mount := range []bool{false, true} {
			destination := t.TempDir() + "/saved.bin"
			if invalidDestination {
				occupied := t.TempDir() + "/occupied"
				if err := os.WriteFile(occupied, []byte("occupied"), 0600); err != nil {
					t.Fatal(err)
				}
				destination = occupied + "/saved.bin"
			}
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("destination", destination) })
			engine.POST("/save", Save)
			if mount {
				var err error
				document, err = ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Save", Version: "1"}})
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(destination); err == nil {
				t.Fatal("documentation build wrote a file")
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("asset", "sample.bin")
			if err != nil {
				t.Fatal(err)
			}
			payload := []byte{0, 255, 1}
			if _, err = part.Write(payload); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("POST", "/save", bytes.NewReader(body.Bytes()))
			request.Header.Set("Content-Type", writer.FormDataContentType())
			record := httptest.NewRecorder()
			engine.ServeHTTP(record, request)
			if invalidDestination {
				if record.Code != 500 || record.Body.String() != "save failed" {
					t.Fatal(record.Code, record.Body)
				}
			} else {
				if record.Code != 204 || record.Body.Len() != 0 {
					t.Fatal(record.Code, record.Body)
				}
				saved, err := os.ReadFile(destination)
				if err != nil || !bytes.Equal(saved, payload) {
					t.Fatal("actual upload bytes changed", err)
				}
			}
		}
		var parsed spec.OpenAPI
		if err := json.Unmarshal(document.JSON(), &parsed); err != nil {
			t.Fatal(err)
		}
		op := parsed.Paths["/save"].Post
		if len(op.Parameters) != 0 || len(op.Responses) != 3 || op.Responses["204"].Value == nil || op.Responses["500"].Value == nil || op.Responses["400"].Value == nil {
			t.Fatal("filesystem outcomes or context source were misclassified")
		}
	}
}

// Unsupported dynamic reads and known-nil saves block only their selected routes, not unused candidates.
func TestRawReadDiagnostics(t *testing.T) {
	bundle := rawRequestBundle(t)
	for _, sample := range []struct {
		handler gin.HandlerFunc
		message string
	}{{Dynamic, "field name"}, {IgnoreFileError, "file pointer is nil"}} {
		engine := gin.New()
		engine.POST("/unsupported", sample.handler)
		_, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Unsupported", Version: "1"}})
		if err == nil || !strings.Contains(err.Error(), sample.message) {
			t.Fatalf("missing selected-route diagnostic: %v", err)
		}
		if len(engine.Routes()) != 1 {
			t.Fatal("failed documentation mount changed routes")
		}
	}
	engine := gin.New()
	engine.GET("/query", Query)
	if _, err := ginswagger.Mount(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Unrelated", Version: "1"}}); err != nil {
		t.Fatal("unused candidate diagnostics leaked", err)
	}
}
