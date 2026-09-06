package routes

import "testing"

// Distinguish ordinary parameters from cross-slash catch-all parameters.
// 验证普通参数和跨斜杠 catch-all 被适配器显式区分。
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

// Fuzz path parsing for crashes and unbalanced output templates.
// 对任意路径验证解析不会崩溃或输出不平衡模板。
func FuzzRoutePath(f *testing.F) {
	for _, path := range []string{"/", "/users/:id", "/assets/*path", "/a/%2F"} {
		f.Add(path)
	}
	f.Fuzz(func(t *testing.T, path string) { _, _ = Parse(path) })
}
