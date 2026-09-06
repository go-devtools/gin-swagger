package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Help must succeed without loading an application or generating files.
// 帮助命令应成功返回，且不加载应用或生成文件。
func TestSubcommandHelp(t *testing.T) {
	for _, command := range []string{"generate", "check", "explain"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), []string{command, "--help"}, &stdout, &stderr)
			if code != 0 || !strings.Contains(stderr.String(), "Usage of "+command) || stdout.Len() != 0 {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
		})
	}
}

// Reject stray positional arguments before they can trigger source loading or generation.
// 在触发源码加载或生成前拒绝多余的位置参数。
func TestRejectPositionalArguments(t *testing.T) {
	for _, command := range []string{"generate", "check", "explain"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), []string{command, "--dir", t.TempDir(), "unexpected"}, &stdout, &stderr)
			var failure struct{ Code, Severity, Message string }
			if err := json.Unmarshal(stderr.Bytes(), &failure); err != nil {
				t.Fatal(err, stderr.String())
			}
			if code != 1 || stdout.Len() != 0 || failure.Code != "gin-swagger.cli.failed" || failure.Severity != "error" || failure.Message != command+" does not accept positional arguments" {
				t.Fatalf("code=%d stdout=%s failure=%+v", code, stdout.String(), failure)
			}
		})
	}
}
