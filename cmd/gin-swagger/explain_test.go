package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Exercise public explanations against the real example without modifying its business sources.
func TestExplainBasicCLI(t *testing.T) {
	prefix := "github.com/go-devtools/gin-swagger/examples/basic."
	for _, tc := range []struct {
		name string
		args []string
		kind string
		ok   bool
	}{
		{"field", []string{"explain", "--symbol", prefix + "User.Name"}, "field", true},
		{"handler", []string{"explain", "--symbol", prefix + "CreateUser"}, "operation", true},
		{"response", []string{"explain", "--symbol", prefix + "CreateUser", "--response", "201"}, "response", true},
		{"unknown field", []string{"explain", "--symbol", prefix + "User.Missing"}, "", false},
		{"missing response", []string{"explain", "--symbol", prefix + "CreateUser", "--response", "799"}, "", false},
		{"response needs handler", []string{"explain", "--symbol", prefix + "User.Name", "--response", "201"}, "", false},
		{"response only for explain", []string{"check", "--response", "201"}, "", false},
		{"budget", []string{"explain", "--symbol", prefix + "CreateUser", "--max-explain-bytes", "64"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output, errs bytes.Buffer
			args := append(append([]string{}, tc.args...), "--dir", "../../examples/basic")
			code := run(context.Background(), args, &output, &errs)
			if !tc.ok {
				if code == 0 || !json.Valid(errs.Bytes()) {
					t.Fatalf("expected structured failure, got %d %s %s", code, output.String(), errs.String())
				}
				return
			}
			if code != 0 {
				t.Fatalf("explain failed: %d %s", code, errs.String())
			}
			var value map[string]json.RawMessage
			if err := json.Unmarshal(output.Bytes(), &value); err != nil {
				t.Fatal(err)
			}
			if tc.name == "handler" {
				for _, key := range []string{"key", "symbol", "operation", "source"} {
					if _, ok := value[key]; !ok {
						t.Fatalf("legacy template field %s removed", key)
					}
				}
				raw := value["explanation"]
				if err := json.Unmarshal(raw, &value); err != nil {
					t.Fatal(err)
				}
			}
			var kind string
			if err := json.Unmarshal(value["kind"], &kind); err != nil || kind != tc.kind {
				t.Fatalf("kind: %s %v", output.String(), err)
			}
			if !bytes.Contains(output.Bytes(), []byte(`"not-proven"`)) {
				t.Fatalf("missing declaration boundary: %s", output.String())
			}
			if tc.name == "field" && (!bytes.Contains(value["uses"], []byte(`"wireName":"Name"`)) || !bytes.Contains(value["uses"], []byte(`"status":"201"`))) {
				t.Fatalf("field wire use: %s", output.String())
			}
		})
	}
}

// Simulate an output stream error without external files or processes.
type failedExplainWriter struct{}

// Return a stable write failure for CLI error propagation.
func (failedExplainWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

// Report JSON encoding failures instead of claiming successful explanation output.
func TestExplainOutputFailure(t *testing.T) {
	var errs bytes.Buffer
	code := run(context.Background(), []string{"explain", "--dir", "../../examples/basic", "--symbol", "github.com/go-devtools/gin-swagger/examples/basic.CreateUser"}, failedExplainWriter{}, &errs)
	if code != 1 || !bytes.Contains(errs.Bytes(), []byte("output unavailable")) {
		t.Fatalf("write failure: %d %s", code, errs.String())
	}
}
