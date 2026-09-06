package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// Configure groups and security at startup while keeping the existing user endpoint unauthenticated.
func TestExampleGroupsAndSecurity(t *testing.T) {
	engine, doc, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	var schema spec.OpenAPI
	if err = json.Unmarshal(doc.JSON(), &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Components.SecuritySchemes) != 1 {
		t.Fatal("authentication example must retain only Bearer")
	}
	bearer := schema.Components.SecuritySchemes["BearerAuth"].Value
	if bearer == nil || bearer.Type != "http" || bearer.Scheme != "bearer" {
		t.Fatal("standard Bearer authentication scheme is missing")
	}
	if schema.Paths["/examples/auth/api-key"] != nil {
		t.Fatal("document still contains the removed API key demonstration")
	}
	removed := httptest.NewRecorder()
	engine.ServeHTTP(removed, httptest.NewRequest("GET", "/examples/auth/api-key", nil))
	if removed.Code != 404 {
		t.Fatal("API key demonstration route was not removed")
	}
	var tags []string
	for _, tag := range schema.Tags {
		tags = append(tags, tag.Name)
		if tag.Description == "" {
			t.Fatal("group description is missing")
		}
	}
	if !reflect.DeepEqual(tags, []string{"Users", "HTTP methods", "Legacy", "Enums", "Field types", "Authentication"}) {
		t.Fatalf("group order: %v", tags)
	}
	if schema.Security.Present || schema.Paths["/users"].Post.Security.Present {
		t.Fatal("new demonstration changed authentication on the original user operation")
	}
	for _, item := range []struct{ path, scheme, header, value string }{
		{"/examples/auth/bearer", "BearerAuth", "Authorization", "Bearer demo-token"},
	} {
		op := schema.Paths[item.path].Get
		if !op.Security.Present || len(op.Security.Value) != 1 {
			t.Fatal("protected operation has no security declaration")
		}
		if _, ok := op.Security.Value[0][item.scheme]; !ok {
			t.Fatal("lock icon references the wrong security scheme")
		}
		for _, credential := range []string{"", "wrong", item.value} {
			expected := 401
			if credential == item.value {
				expected = 200
			}
			req := httptest.NewRequest("GET", item.path, nil)
			req.Header.Set(item.header, credential)
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)
			if rr.Code != expected {
				t.Fatalf("%s status %d, expected %d", item.path, rr.Code, expected)
			}
			pointer := fmt.Sprintf("/paths/%s/get/responses/%d/content/application~1json/schema", escapePointer(item.path), expected)
			v, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if err = v.JSON(rr.Body.Bytes()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// Escape one document-pointer path key without interpreting route separators as schema levels.
func escapePointer(value string) string {
	var out []byte
	for _, c := range []byte(value) {
		if c == '~' {
			out = append(out, '~', '0')
		} else if c == '/' {
			out = append(out, '~', '1')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

// Keep named examples consistent with actual enum validation, including zero and default states.
func TestEnumRequestExamples(t *testing.T) {
	engine, doc, err := Router()
	if err != nil {
		t.Fatal(err)
	}
	validator, err := contracttest.Compile(doc.JSON(), "/paths/~1examples~1enums/post/requestBody/content/application~1json/schema", contracttest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var schema spec.OpenAPI
	if err = json.Unmarshal(doc.JSON(), &schema); err != nil {
		t.Fatal(err)
	}
	examples := schema.Paths["/examples/enums"].Post.RequestBody.Value.Content["application/json"].Value.Examples
	if len(examples) != 3 {
		t.Fatalf("named example count: %d", len(examples))
	}
	for _, example := range examples {
		raw, err := json.Marshal(example.Value.Value.Value)
		if err != nil {
			t.Fatal(err)
		}
		if err = validator.JSON(raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, sample := range []struct {
		body   string
		status int
	}{
		{`{"Role":"admin","State":0}`, 200}, {`{"Role":"editor","State":1}`, 200}, {`{"Role":"viewer","State":2}`, 200},
		{`{"Role":"admin"}`, 200}, {`{"Role":"unknown","State":0}`, 400}, {`{"Role":"admin","State":3}`, 400}, {`{"State":0}`, 400}, {`{"Role":"admin","State":"pending"}`, 400},
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/examples/enums", bytes.NewBufferString(sample.body))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(rr, req)
		if rr.Code != sample.status {
			t.Fatalf("%s returned %d: %s", sample.body, rr.Code, rr.Body.String())
		}
		if accepted := validator.JSON([]byte(sample.body)) == nil; accepted != (sample.status == 200) {
			t.Fatalf("request differs from the enum Schema: %s", sample.body)
		}
	}
}
