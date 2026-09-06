# Require the exact development toolchain, then generate the example, test, and build.
# 开发入口要求精确工具链，生成示例后测试并构建。
GO ?= go
export GOTOOLCHAIN = local

.PHONY: dev
# Generate once without changing business handlers or routes.
# 单次生成不会改变业务 handler 或路由。
dev:
	@$(GO) version | grep -q '^go version go1.27.1 '
	$(GO) run ./cmd/gin-swagger generate --dir ./examples/basic --output ./internal/apidoc
	$(GO) test ./...
	$(GO) build ./...
