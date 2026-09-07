package routes

import (
	"net/url"
	"strings"
	"testing"
)

// Distinguish ordinary parameters from cross-slash catch-all parameters.
func TestRoutePath(t *testing.T) {
	for _, tt := range []struct {
		input, path string
		catch       bool
	}{{"/users/:id", "/users/{id}", false}, {"/assets/*filepath", "/assets/{filepath}", true}, {"/health", "/health", false}} {
		p, err := Parse(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if p.Path != tt.path || p.CatchAll != tt.catch {
			t.Fatalf("incorrect conversion of %s: %+v", tt.input, p)
		}
	}
	for _, bad := range []string{"", "users/:id", "/users/:", "/a/*file/more", "/a/:id/:id"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("incorrectly accepted %q", bad)
		}
	}
}

// Check that successful conversions produce request URLs with no accidental query or fragment.
func FuzzRoutePath(f *testing.F) {
	for _, path := range []string{"/", "/users/:id", "/assets/*path", "/a/%2F", "/张三/:id", "/a\\:b", "/?#{}%"} {
		f.Add(path)
	}
	f.Fuzz(func(t *testing.T, path string) {
		for _, profile := range []Profile{{}, {UseRawPath: true}, {UseEscapedPath: true}} {
			result, err := ParseWithProfile(path, profile)
			if err != nil {
				continue
			}
			wire := result.Path
			for _, name := range result.Parameters {
				wire = strings.ReplaceAll(wire, "{"+name+"}", "sample")
			}
			parsed, err := url.ParseRequestURI(wire)
			if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(wire, "{}?#") {
				t.Fatalf("path %q produced a non-path URL %q: %v", path, wire, err)
			}
		}
	})
}
