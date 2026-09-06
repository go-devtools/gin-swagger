// Compose the public Gin frontend and core compiler without copying shared algorithms.
// 组合公开 Gin 前端与核心编译器，不复制注释或 Schema 算法。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	front "github.com/openapi-golang/gin-swagger/compiler"
	"github.com/openapi-golang/openapi"
	core "github.com/openapi-golang/openapi/compiler"
)

// Handle interruption and expose automation-friendly exit codes.
// 响应中断并将失败转换为可自动化处理的退出码。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// Dispatch generation, validation, provenance, and version commands.
// 分派生成、检查、来源解释与版本命令。
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		_ = json.NewEncoder(stderr).Encode(map[string]any{"code": "gin-swagger.cli.failed", "severity": "error", "message": err.Error()})
		return 1
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(stdout, "gin-swagger generate --dir . --output ./internal/apidoc\ngin-swagger check --dir . --output ./internal/apidoc\ngin-swagger check --spec openapi.json\ngin-swagger explain --dir . --symbol module/pkg.Handler\ngin-swagger explain --dir . --symbol module/pkg.DTO.Field\ngin-swagger explain --dir . --symbol module/pkg.Handler --response 201\ngin-swagger version")
		return 0
	}
	if args[0] == "version" {
		version, coreVersion, revision := "unversioned", "unversioned", ""
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
			for _, dep := range info.Deps {
				if dep.Path == "github.com/openapi-golang/openapi" {
					coreVersion = dep.Version
				}
			}
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					revision = setting.Value
				}
			}
		}
		_ = json.NewEncoder(stdout).Encode(map[string]any{"module": "github.com/openapi-golang/gin-swagger", "version": version, "core": coreVersion, "frontend": front.Frontend().Name, "revision": revision, "go": runtime.Version(), "bundle": openapi.BundleFormatVersion, "openapi": "3.2.0"})
		return 0
	}
	if args[0] != "generate" && args[0] != "check" && args[0] != "explain" {
		return fail(fmt.Errorf("unknown command %s", args[0]))
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", ".", "Application source directory")
	output := flags.String("output", "./internal/apidoc", "Output location relative to the application directory")
	packageName := flags.String("package", "apidoc", "Generated file package name")
	specFile := flags.String("spec", "", "Independent specification validation file")
	symbol := flags.String("symbol", "", "Fully qualified symbol to explain")
	response := flags.String("response", "", "Exact response status for the selected handler (explain only)")
	maxExplainBytes := flags.Int("max-explain-bytes", 0, "Evidence byte limit for explain; zero selects sixteen MiB")
	tags := flags.String("tags", "", "Actual build tags")
	maxDepth := flags.Int("max-depth", 12, "Maximum cross-function depth")
	maxPaths := flags.Int("max-paths", 128, "Maximum path count")
	maxCalls := flags.Int("max-calls", 10000, "Maximum call count")
	timeout := flags.Duration("timeout", time.Minute, "Maximum generation duration")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		return fail(fmt.Errorf("%s does not accept positional arguments", args[0]))
	}
	if *response != "" && args[0] != "explain" {
		return fail(fmt.Errorf("--response is only supported by explain"))
	}
	if args[0] == "explain" && strings.TrimSpace(*symbol) == "" {
		return fail(fmt.Errorf("explain requires --symbol with a fully qualified Go symbol"))
	}
	if *maxExplainBytes < 0 {
		return fail(fmt.Errorf("--max-explain-bytes must not be negative"))
	}
	if *specFile != "" {
		if args[0] != "check" {
			return fail(fmt.Errorf("--spec is only supported by check"))
		}
		raw, err := os.ReadFile(*specFile)
		if err != nil {
			return fail(err)
		}
		report := openapi.Check(raw)
		_ = json.NewEncoder(stdout).Encode(report)
		if report.HasErrors() {
			return 1
		}
		return 0
	}
	if *maxDepth < 1 || *maxPaths < 1 || *maxCalls < 1 || *timeout <= 0 {
		return fail(fmt.Errorf("resource budgets must be positive"))
	}
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	absolute, err := filepath.Abs(*dir)
	if err != nil {
		return fail(err)
	}
	target := *output
	if !filepath.IsAbs(target) {
		target = filepath.Join(absolute, target)
	}
	// Bootstrap only owned generated output and restore absence on failure.
	// 首次仅引导本工具拥有的生成文件；任何失败都恢复缺失状态。
	cleanup := func() {}
	if args[0] == "generate" {
		cleanup, err = bootstrap(target, *packageName)
		if err != nil {
			return fail(err)
		}
	}
	success := false
	defer func() {
		if !success {
			cleanup()
		}
	}()
	options := core.Options{Load: core.LoadOptions{Dir: absolute, Patterns: []string{"./..."}}, Frontends: []core.Frontend{front.Frontend()}, MaxDepth: *maxDepth, MaxPaths: *maxPaths, MaxCalls: *maxCalls, Explain: args[0] == "explain", MaxExplainBytes: *maxExplainBytes}
	if *tags != "" {
		options.Load.BuildFlags = []string{"-tags=" + *tags}
	}
	result, err := core.Compile(ctx, options)
	if err != nil {
		return fail(err)
	}
	write := core.WriteOptions{Dir: target, Package: *packageName}
	switch args[0] {
	case "generate":
		err = result.Write(write)
	case "check":
		err = result.Check(write)
	case "explain":
		var explanation core.Explanation
		explanation, err = result.Explain(core.ExplainQuery{Symbol: *symbol, Response: *response})
		if err != nil {
			break
		}
		if explanation.Operation != nil {
			// Preserve the existing top-level template contract for handler consumers.
			// 为 handler 输出的既有使用方保留顶层模板契约。
			template := *explanation.Operation
			explanation.Operation = nil
			err = json.NewEncoder(stdout).Encode(struct {
				openapi.Template
				Explanation core.Explanation `json:"explanation"`
			}{Template: template, Explanation: explanation})
		} else {
			err = json.NewEncoder(stdout).Encode(explanation)
		}
	}
	if err != nil {
		return fail(err)
	}
	success = true
	if args[0] != "explain" {
		_ = json.NewEncoder(stdout).Encode(map[string]any{"status": "ok", "fingerprint": result.Bundle.Snapshot().Fingerprint, "templates": len(result.Bundle.Index())})
	}
	return 0
}

// Create a temporary type-checking package without executing or registering an empty contract.
// 为缺失的生成包创建可恢复的类型检查引导文件，不运行或注册空契约。
func bootstrap(dir, packageName string) (func(), error) {
	path := filepath.Join(dir, "zz_openapi.gen.go")
	if _, err := os.Stat(path); err == nil {
		return func() {}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if packageName == "" || strings.ContainsAny(packageName, " \n\r\t./\\\";") {
		return nil, fmt.Errorf("invalid generated package name")
	}
	var created []string
	for current := dir; current != filepath.Dir(current); current = filepath.Dir(current) {
		if _, err := os.Stat(current); err == nil {
			break
		}
		created = append(created, current)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	content := "// Code generated by openapi/compiler. DO NOT EDIT.\n// Recoverable bootstrap used only for the first static type check.\n// 可恢复的占位源码，仅供首次静态类型检查使用。\npackage " + packageName + "\nimport \"github.com/openapi-golang/openapi\"\n// This factory is not executed before actual generation completes.\n// 实际生成完成前不会执行此工厂。\nfunc Bundle() openapi.Bundle { return openapi.Bundle{} }\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}
	return func() {
		_ = os.Remove(path)
		for _, directory := range created {
			_ = os.Remove(directory)
		}
	}, nil
}
