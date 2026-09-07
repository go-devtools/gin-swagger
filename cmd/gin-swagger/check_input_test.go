package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Validate the actual spec command without loading source or changing generated files.
func TestSpecInputBudgetAndCancellation(t *testing.T) {
	raw := []byte(`{"openapi":"3.2.0","info":{"title":"Input","version":"1"},"paths":{}}`)
	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name    string
		ctx     context.Context
		flags   []string
		message string
	}{
		{"canceled", canceled, nil, "context canceled"},
		{"timeout", context.Background(), []string{"--timeout=1ns"}, "context deadline exceeded"},
		{"byte limit", context.Background(), []string{"--max-bytes=8"}, "byte budget"},
		{"invalid timeout", context.Background(), []string{"--timeout=0"}, "positive"},
		{"invalid byte limit", context.Background(), []string{"--max-bytes=0"}, "positive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errout bytes.Buffer
			args := append([]string{"check", "--spec", path}, tc.flags...)
			code := run(tc.ctx, args, &out, &errout)
			var failure struct{ Code, Severity, Message string }
			if err := json.Unmarshal(errout.Bytes(), &failure); err != nil {
				t.Fatalf("non-JSON failure: %v; %s", err, errout.String())
			}
			if code != 1 || out.Len() != 0 || failure.Code != "gin-swagger.cli.failed" || failure.Severity != "error" || !strings.Contains(failure.Message, tc.message) {
				t.Fatalf("code=%d stdout=%s failure=%+v", code, out.String(), failure)
			}
		})
	}
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"check", "--spec", path, "--max-bytes=" + strconv.Itoa(len(raw))}, &out, &errout); code != 0 {
		t.Fatalf("exact byte boundary rejected: %d %s %s", code, out.String(), errout.String())
	}
	var report struct {
		Diagnostics []json.RawMessage `json:"diagnostics"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || len(report.Diagnostics) != 0 {
		t.Fatal(err, out.String())
	}
}

// Reject non-regular input before attempting a read and avoid treating source flags as spec budgets.
func TestSpecInputKindsAndFlagScope(t *testing.T) {
	for _, path := range []string{t.TempDir(), os.DevNull} {
		var out, errout bytes.Buffer
		code := run(context.Background(), []string{"check", "--spec", path}, &out, &errout)
		if code != 1 || !strings.Contains(errout.String(), "only regular files") {
			t.Fatalf("%s: %d %s", path, code, errout.String())
		}
	}
	for _, command := range []string{"generate", "check", "explain"} {
		var out, errout bytes.Buffer
		args := []string{command, "--dir", t.TempDir(), "--max-bytes=8"}
		if command == "explain" {
			args = append(args, "--symbol=example.test.Handler")
		}
		if code := run(context.Background(), args, &out, &errout); code != 1 || !strings.Contains(errout.String(), "--max-bytes requires check --spec") {
			t.Fatalf("%s: %d %s", command, code, errout.String())
		}
	}
}
