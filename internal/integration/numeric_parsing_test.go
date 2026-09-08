package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/go-devtools/gin-swagger"
	front "github.com/go-devtools/gin-swagger/compiler"
	"github.com/go-devtools/gin-swagger/internal/integration/testdata/protocolbounds"
	"github.com/go-devtools/openapi"
	core "github.com/go-devtools/openapi/compiler"
	"github.com/go-devtools/openapi/contracttest"
	"github.com/go-devtools/openapi/spec"
)

// Compile real source while checking that neither business code nor its static asset changes.
func protocolBoundaryBundle(t *testing.T, frontend core.Frontend, configuration map[string]json.RawMessage) *core.Result {
	t.Helper()
	paths := []string{"testdata/protocolbounds/app.go", "testdata/protocolbounds/report.txt"}
	before := map[string][]byte{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = raw
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/protocolbounds"}, Frontends: []core.Frontend{frontend}, Configuration: configuration, Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before[path], after) {
			t.Fatal("generation changed fixture", path)
		}
	}
	return result
}

// Distinguish checked parse failures from accepted error results without converting raw query text into numeric input.
func TestNumericParsingAndByteLength(t *testing.T) {
	result := protocolBoundaryBundle(t, front.Frontend(), nil)
	for _, sample := range []struct {
		name    string
		handler gin.HandlerFunc
		checked bool
		values  []struct {
			text    string
			parsed  int64
			invalid bool
		}
	}{
		{"checked-atoi", protocolbounds.CheckedAtoi, true, nativeNumbers()},
		{"ignored-atoi", protocolbounds.IgnoredAtoi, false, nativeNumbers()},
		{"checked-int16", protocolbounds.CheckedParseInt, true, smallNumbers()},
		{"ignored-int16", protocolbounds.IgnoredParseInt, false, smallNumbers()},
		{"byte-length", protocolbounds.ByteLength, true, []struct {
			text    string
			parsed  int64
			invalid bool
		}{{"ab", 2, true}, {"é", 2, true}, {"中", 3, false}, {"🙂", 4, false}, {"aé", 3, false}}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			before, after := gin.New(), gin.New()
			before.GET("/value", sample.handler)
			after.GET("/value", sample.handler)
			doc, err := ginswagger.Mount(after, result.Bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Numeric input", Version: "1"}})
			if err != nil {
				t.Fatal(err)
			}
			var parsed spec.OpenAPI
			if err := json.Unmarshal(doc.JSON(), &parsed); err != nil {
				t.Fatal(err)
			}
			op := parsed.Paths["/value"].Get
			if len(op.Parameters) != 1 || op.Parameters[0].Value.Name != "value" || (op.Responses["400"].Value != nil) != sample.checked || op.Responses["default"].Value != nil {
				t.Fatal("parsing lost response branches or raw parameter identity")
			}
			parameter := op.Parameters[0].Value.Schema
			if !reflect.DeepEqual(parameter.Type, spec.Types{"string"}) || parameter.MinLength.Present || parameter.Pattern != "" {
				t.Fatal("using a parsed value or byte length fabricated input validation")
			}
			input, err := contracttest.Compile(doc.JSON(), "/paths/~1value/get/parameters/0/schema", contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			output, err := contracttest.Compile(doc.JSON(), "/paths/~1value/get/responses/200/content/application~1json/schema", contracttest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range sample.values {
				var records []*httptest.ResponseRecorder
				for _, engine := range []*gin.Engine{before, after} {
					recorder := httptest.NewRecorder()
					engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/value?value="+url.QueryEscape(value.text), nil))
					records = append(records, recorder)
				}
				if records[0].Code != records[1].Code || !bytes.Equal(records[0].Body.Bytes(), records[1].Body.Bytes()) || !reflect.DeepEqual(records[0].Header(), records[1].Header()) {
					t.Fatal("mount changed numeric conversion")
				}
				if err := input.Value(value.text); err != nil {
					t.Fatal(err)
				}
				want := 200
				if sample.checked && value.invalid {
					want = 400
				}
				if records[1].Code != want {
					t.Fatalf("%q: status %d want %d", value.text, records[1].Code, want)
				}
				if want == 400 {
					if records[1].Body.Len() != 0 || len(op.Responses["400"].Value.Content) != 0 {
						t.Fatal("rejected parse acquired an invented body")
					}
					continue
				}
				var got protocolbounds.Parsed
				if err := json.Unmarshal(records[1].Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.Value != value.parsed {
					t.Fatalf("%q: value %d want %d", value.text, got.Value, value.parsed)
				}
				if err := output.JSON(records[1].Body.Bytes()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Specify hand-checked decimal syntax and signed native-width overflow cases for the active integer width.
func nativeNumbers() []struct {
	text    string
	parsed  int64
	invalid bool
} {
	return []struct {
		text    string
		parsed  int64
		invalid bool
	}{{"", 0, true}, {"bad", 0, true}, {" 7", 0, true}, {"+7", 7, false}, {"007", 7, false}, {"-8", -8, false}, {"32768", 32768, false}, {"9223372036854775808", int64(^uint(0) >> 1), true}}
}

// Specify signed sixteen-bit saturation independently of the native integer width.
func smallNumbers() []struct {
	text    string
	parsed  int64
	invalid bool
} {
	return []struct {
		text    string
		parsed  int64
		invalid bool
	}{{"", 0, true}, {"bad", 0, true}, {"+7", 7, false}, {"007", 7, false}, {"-8", -8, false}, {"32768", 32767, true}, {"-32769", -32768, true}}
}
