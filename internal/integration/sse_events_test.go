package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ginswagger "github.com/openapi-golang/gin-swagger"
	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/gin-swagger/internal/integration/testdata/sseevents"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
	"github.com/openapi-golang/openapi/contracttest"
	"github.com/openapi-golang/openapi/spec"
)

// 静态生成真实 SSE 样本，并检查生成器没有改写业务源码。
// Generate real SSE fixtures statically and check that generation leaves business source unchanged.
func sseBundle(t *testing.T) openapi.Bundle {
	t.Helper()
	path := "testdata/sseevents/app.go"
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := core.Compile(context.Background(), core.Options{Load: core.LoadOptions{Dir: "testdata/sseevents"}, Frontends: []core.Frontend{front.Frontend()}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("generator rewrote SSE source")
	}
	return result.Bundle
}

// 记录手工确认的协议事件，不复用生产编码或 Schema 生成逻辑计算期望值。
// Record hand-checked protocol events without deriving expectations from production encoders or schemas.
func expectedEvent(name, data string) map[string]any {
	value := map[string]any{"data": data}
	if name != "" {
		value["event"] = name
	}
	return value
}

// 经真实 HTTP 服务验证 SSE 文档、实际帧和挂载前后业务行为一致。
// Verify SSE contracts, actual frames, and unchanged business behavior through real HTTP servers before/after mounting.
func TestSSEWireContract(t *testing.T) {
	bundle := sseBundle(t)
	for _, sample := range []struct {
		name    string
		handler gin.HandlerFunc
		status  int
		cache   string
		events  []map[string]any
	}{
		{"objects", sseevents.SSEObjects, 202, "no-cache", []map[string]any{expectedEvent("message", `{"Name":"Ada","Count":2}`), expectedEvent("count", `{"count":3}`)}},
		{"text", sseevents.SSEText, 200, "no-cache", []map[string]any{expectedEvent("text", "first\nsecond\\rline")}},
		{"whitespace", sseevents.SSEWhitespace, 200, "no-cache", []map[string]any{expectedEvent("name", "first\n second")}},
		{"metadata", sseevents.SSEMetadata, 201, "no-cache", []map[string]any{{"event": "notice\\nnext", "id": "id\\r", "retry": json.Number("1500"), "data": `{"Name":"Ada","Count":2}`}}},
		{"default", sseevents.SSEDefault, 200, "no-cache", []map[string]any{{"data": "hello"}}},
		{"event-pointer", sseevents.SSEEventPointer, 200, "no-cache", []map[string]any{{"data": "pointer"}}},
		{"invalid-id", sseevents.SSEInvalidID, 200, "no-cache", []map[string]any{{"data": "hello"}}},
		{"nil", sseevents.SSENil, 200, "no-cache", []map[string]any{expectedEvent("nil", "<nil>")}},
		{"nil-pointer", sseevents.SSENilPointer, 200, "no-cache", []map[string]any{expectedEvent("nil", "<nil>")}},
		{"boxed-nil", sseevents.SSEBoxedNil, 200, "no-cache", []map[string]any{expectedEvent("nil", "<nil>")}},
		{"nil-map", sseevents.SSENilMap, 200, "no-cache", []map[string]any{expectedEvent("map", "null")}},
		{"nil-slice", sseevents.SSENilSlice, 200, "no-cache", []map[string]any{expectedEvent("slice", "null")}},
		{"pointer", sseevents.SSEPointer, 200, "no-cache", []map[string]any{expectedEvent("pointer", `{"Name":"Ada","Count":2}`)}},
		{"array", sseevents.SSEArray, 200, "no-cache", []map[string]any{expectedEvent("array", "[1 2]")}},
		{"bytes", sseevents.SSEBytes, 200, "no-cache", []map[string]any{expectedEvent("bytes", "first\nsecond")}},
		{"named-bytes", sseevents.SSENamedBytes, 200, "no-cache", []map[string]any{expectedEvent("bytes", `"QUI="`)}},
		{"nil-bytes", sseevents.SSENilBytes, 200, "no-cache", []map[string]any{expectedEvent("bytes", "")}},
		{"dynamic", sseevents.SSEDynamicName, 200, "no-cache", []map[string]any{expectedEvent("dynamic", "hello")}},
		{"cache", sseevents.SSECache, 200, "private", []map[string]any{expectedEvent("cache", "hello")}},
		{"committed", sseevents.SSECommitted, 409, "", []map[string]any{expectedEvent("committed", "hello")}},
		{"late", sseevents.SSELateHeaders, 200, "no-cache", []map[string]any{expectedEvent("first", "one"), expectedEvent("second", "two")}},
		{"no-content", sseevents.SSENoContent, 204, "no-cache", nil},
		{"zero-event", sseevents.SSEZeroEvent, 200, "no-cache", []map[string]any{{"data": "<nil>"}}},
		{"converted-nil", sseevents.SSEConvertedNil, 200, "no-cache", []map[string]any{expectedEvent("nil", "<nil>")}},
		{"unknown-pointer", sseevents.SSEUnknownPointer, 200, "no-cache", []map[string]any{expectedEvent("pointer", `{"Name":"Global","Count":4}`)}},
		{"unknown-nil-pointer", sseevents.SSEUnknownNilPointer, 200, "no-cache", []map[string]any{expectedEvent("pointer", "<nil>")}},
		{"pointer-slice", sseevents.SSEPointerSlice, 200, "no-cache", []map[string]any{expectedEvent("slice", "null")}},
		{"formatted", sseevents.SSEFormatted, 200, "no-cache", []map[string]any{expectedEvent("formatted", "formatted text")}},
		{"invalid-utf8", sseevents.SSEInvalidUTF8, 200, "no-cache", []map[string]any{expectedEvent("utf8", "��")}},
		{"empty-metadata", sseevents.SSEEmptyMetadata, 200, "no-cache", []map[string]any{{"event": "", "id": "", "data": ""}}},
		{"dynamic-retry", sseevents.SSEDynamicRetry, 200, "no-cache", []map[string]any{{"retry": json.Number("1500"), "data": "retry"}}},
		{"large-retry", sseevents.SSELargeRetry, 200, "no-cache", []map[string]any{{"retry": json.Number("4294967295"), "data": "retry"}}},
		{"ignored-status", sseevents.SSEIgnoredStatus, 202, "no-cache", []map[string]any{{"data": "pending"}}},
		{"zero-status", sseevents.SSEZeroStatus, 200, "no-cache", []map[string]any{{"data": "default"}}},
		{"pending-no-content", sseevents.SSEPendingNoContent, 204, "no-cache", nil},
		{"not-modified", sseevents.SSENotModified, 304, "no-cache", nil},
	} {
		for _, method := range []string{"GET", "HEAD", "POST"} {
			t.Run(sample.name+"/"+method, func(t *testing.T) {
				before, after := gin.New(), gin.New()
				before.Handle(method, "/events", sample.handler)
				after.Handle(method, "/events", sample.handler)
				doc, err := ginswagger.Mount(after, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Events", Version: "1"}})
				if err != nil {
					t.Fatal(err)
				}
				var bodies [][]byte
				var headers []http.Header
				for _, engine := range []*gin.Engine{before, after} {
					server := httptest.NewServer(engine)
					request, err := http.NewRequest(method, server.URL+"/events?name=dynamic", nil)
					if err != nil {
						t.Fatal(err)
					}
					response, err := server.Client().Do(request)
					if err != nil {
						server.Close()
						t.Fatal(err)
					}
					body, err := io.ReadAll(response.Body)
					response.Body.Close()
					server.Close()
					if err != nil {
						t.Fatal(err)
					}
					if response.StatusCode != sample.status || response.Header.Get("Cache-Control") != sample.cache {
						t.Fatalf("wrong wire status/cache: %d %v", response.StatusCode, response.Header)
					}
					if sample.status != 304 && !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
						t.Fatalf("wrong wire media: %v", response.Header)
					}
					response.Header.Del("Date")
					bodies = append(bodies, body)
					headers = append(headers, response.Header)
				}
				if !bytes.Equal(bodies[0], bodies[1]) || !reflect.DeepEqual(headers[0], headers[1]) {
					t.Fatal("mount changed the event stream")
				}
				var document spec.OpenAPI
				if err = json.Unmarshal(doc.JSON(), &document); err != nil {
					t.Fatal(err)
				}
				op := httpOperation(document.Paths["/events"], method)
				response := op.Responses[strconv.Itoa(sample.status)].Value
				if response == nil || len(op.Responses) != 1 {
					t.Fatalf("incorrect statuses: %v", op.Responses)
				}
				if sample.cache != "" {
					header := response.Headers["Cache-Control"].Value
					if header == nil || header.Schema.Const.Value != sample.cache {
						t.Fatal("cache header constraint was lost")
					}
				} else if response.Headers["Cache-Control"].Value != nil {
					t.Fatal("late default cache header entered the contract")
				}
				if method == "HEAD" || sample.events == nil {
					if len(bodies[1]) != 0 || len(response.Content) != 0 {
						t.Fatal("bodyless response retained stream content")
					}
					return
				}
				media := response.Content["text/event-stream"].Value
				if media == nil || media.ItemSchema == nil || media.Schema != nil {
					t.Fatal("missing native itemSchema")
				}
				events, err := contracttest.ParseSSE(bytes.NewReader(bodies[1]), contracttest.Limits{})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(events, sample.events) {
					t.Fatalf("wrong parsed events:\ngot %#v\nwant %#v", events, sample.events)
				}
				pointer := "/paths/~1events/" + strings.ToLower(method) + "/responses/" + strconv.Itoa(sample.status) + "/content/text~1event-stream/itemSchema"
				validator, err := contracttest.Compile(doc.JSON(), pointer, contracttest.Options{AssertContent: true})
				if err != nil {
					t.Fatal(err)
				}
				if err = validator.SSE(bytes.NewReader(bodies[1]), contracttest.Limits{}); err != nil {
					t.Fatal(err)
				}
				wrong := map[string]any{}
				for k, v := range events[0] {
					wrong[k] = v
				}
				wrong["data"] = 7
				if validator.Value(wrong) == nil {
					t.Fatal("SSE data was not constrained to string")
				}
				if sample.name == "objects" || sample.name == "pointer" {
					wrong["data"] = `{"Name":7,"Count":"bad"}`
					if validator.Value(wrong) == nil {
						t.Fatal("JSON payload constraints were lost")
					}
					wrong["data"] = "null"
					if validator.Value(wrong) == nil {
						t.Fatal("non-nil object acquired a JSON null alternative")
					}
				}
				if sample.name == "dynamic-retry" {
					if err = validator.Value(map[string]any{"data": "retry"}); err != nil {
						t.Fatal("possibly absent retry became required")
					}
				}
				if sample.name == "dynamic" {
					if err = validator.Value(map[string]any{"data": "hello"}); err != nil {
						t.Fatal("possibly absent event name became required")
					}
				}
			})
		}
	}
}

// 不支持的 JSON 数据、Renderer 指针或正文混写仅阻断被选中的业务路由。
// Unsupported JSON data, renderer pointers, or mixed bodies block only selected business routes.
func TestSSEDiagnostics(t *testing.T) {
	bundle := sseBundle(t)
	engineForMedia := gin.New()
	engineForMedia.GET("/media", sseevents.SSECommittedWithoutMedia)
	server := httptest.NewServer(engineForMedia)
	wire, err := server.Client().Get(server.URL + "/media")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	body, err := io.ReadAll(wire.Body)
	wire.Body.Close()
	server.Close()
	if err != nil || !strings.HasPrefix(wire.Header.Get("Content-Type"), "text/plain") || !strings.Contains(string(body), "data:hello") {
		t.Fatalf("unexpected committed media: %v %q", wire.Header, body)
	}
	for _, handler := range []gin.HandlerFunc{sseevents.SSEMixed, sseevents.SSEUnsupported, sseevents.SSEWrongCommittedMedia, sseevents.SSENilRenderer, sseevents.SSECommittedWithoutMedia, sseevents.SSELateCommittedMedia} {
		engine := gin.New()
		engine.GET("/bad", handler)
		if _, err := ginswagger.Build(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Bad", Version: "1"}}); err == nil {
			t.Fatal("invalid SSE behavior was accepted")
		}
	}
	engine := gin.New()
	engine.GET("/good", sseevents.SSEText)
	if _, err := ginswagger.Build(engine, bundle, ginswagger.Config{OpenAPI: openapi.Config{Title: "Good", Version: "1"}}); err != nil {
		t.Fatal("unselected diagnostic escaped", err)
	}
}
