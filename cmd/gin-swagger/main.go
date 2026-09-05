// 组合公开 Gin 前端与核心编译器，不复制注释或 Schema 算法。
// Compose the public Gin frontend and core compiler without copying shared algorithms.
package main

import (
	"context"
	"encoding/json"
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

// 响应中断并将失败转换为可自动化处理的退出码。
// Handle interruption and expose automation-friendly exit codes.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// 分派生成、检查、来源解释与版本命令。
// Dispatch generation, validation, provenance, and version commands.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		_ = json.NewEncoder(stderr).Encode(map[string]any{"code": "gin-swagger.cli.failed", "severity": "error", "message": err.Error()})
		return 1
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(stdout, "gin-swagger generate --dir . --output ./internal/apidoc\ngin-swagger check --dir . --output ./internal/apidoc\ngin-swagger check --spec openapi.json\ngin-swagger explain --dir . --symbol module/pkg.Handler\ngin-swagger version")
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
		return fail(fmt.Errorf("未知命令 %s", args[0]))
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", ".", "业务源码目录")
	output := flags.String("output", "./internal/apidoc", "相对业务目录的输出位置")
	packageName := flags.String("package", "apidoc", "生成文件包名")
	specFile := flags.String("spec", "", "独立规范校验文件")
	symbol := flags.String("symbol", "", "需要解释的完整符号")
	tags := flags.String("tags", "", "实际构建 tags")
	maxDepth := flags.Int("max-depth", 12, "最大跨函数深度")
	maxPaths := flags.Int("max-paths", 128, "最大路径数量")
	maxCalls := flags.Int("max-calls", 10000, "最大调用数量")
	timeout := flags.Duration("timeout", time.Minute, "最大生成耗时")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *specFile != "" {
		if args[0] != "check" {
			return fail(fmt.Errorf("--spec 仅适用于 check"))
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
		return fail(fmt.Errorf("资源预算必须为正"))
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
	// 首次仅引导本工具拥有的生成文件；任何失败都恢复缺失状态。
	// Bootstrap only owned generated output and restore absence on failure.
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
	options := core.Options{Load: core.LoadOptions{Dir: absolute, Patterns: []string{"./..."}}, Frontends: []core.Frontend{front.Frontend()}, MaxDepth: *maxDepth, MaxPaths: *maxPaths, MaxCalls: *maxCalls}
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
		found := false
		for _, template := range result.Bundle.Index() {
			if template.Symbol == *symbol {
				found = true
				_ = json.NewEncoder(stdout).Encode(template)
			}
		}
		if !found {
			return fail(fmt.Errorf("没有匹配符号 %s", *symbol))
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

// 为缺失的生成包创建可恢复的类型检查引导文件，不运行或注册空契约。
// Create a temporary type-checking package without executing or registering an empty contract.
func bootstrap(dir, packageName string) (func(), error) {
	path := filepath.Join(dir, "zz_openapi.gen.go")
	if _, err := os.Stat(path); err == nil {
		return func() {}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if packageName == "" || strings.ContainsAny(packageName, " \n\r\t./\\\";") {
		return nil, fmt.Errorf("非法生成包名")
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
	content := "// Code generated by openapi/compiler. DO NOT EDIT.\n// 仅用于首次静态类型检查的可恢复引导。\n// Recoverable bootstrap used only for the first static type check.\npackage " + packageName + "\nimport \"github.com/openapi-golang/openapi\"\n// 实际生成完成前不会执行此工厂。\n// This factory is not executed before actual generation completes.\nfunc Bundle() openapi.Bundle { return openapi.Bundle{} }\n"
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
