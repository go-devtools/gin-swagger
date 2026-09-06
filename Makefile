# Require the exact development toolchain, then generate the example, test, and build.
GO ?= go
export GOTOOLCHAIN = local

.PHONY: dev
# Generate once without changing business handlers or routes.
dev:
	@$(GO) version | grep -q '^go version go1.27.1 '
	$(GO) run ./cmd/gin-swagger generate --dir ./examples/basic --output ./internal/apidoc
	$(GO) test ./...
	$(GO) build ./...
