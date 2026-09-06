# AI-assisted Gin integration

Start with [llms.txt](../llms.txt), then read the [request](requests.md) or [response](responses.md) guide for the behavior being integrated. The compact index follows the [llms.txt proposal](https://llmstxt.org/). It does not add a model dependency or send application code to a service.

## Generation and runtime responsibilities

The compiler reads actual Go source and emits a static Bundle factory. At startup, `ginswagger.Mount(engine, apidoc.Bundle(), config)` links that Bundle to the existing Engine.Routes snapshot and caches documentation. Keep existing DTO tags, handler bodies, signatures, and route registration unchanged when integrating documentation.

Gin rules belong in this module's compiler frontend. Shared type projection, comments, budgets, neutral effects, and document models belong in the public core SDK. An adapter must not import core internal packages. Runtime Build and Mount must not import the compiler.

## Try the existing application

From this repository checkout:

```sh
GOWORK=off go mod download
GOWORK=off go run ./cmd/gin-swagger version
GOWORK=off go run ./cmd/gin-swagger generate --dir ./examples/basic --output ./internal/apidoc
GOWORK=off go run ./cmd/gin-swagger check --dir ./examples/basic --output ./internal/apidoc
GOWORK=off go run ./cmd/gin-swagger explain --dir ./examples/basic --symbol github.com/openapi-golang/gin-swagger/examples/basic.CreateUser
GOWORK=off make dev
```

Download dependencies before generation so dependency installation does not consume the default one-minute generation budget. Direct commands can use an explicit timeout such as --timeout=5m. The generated output belongs to the compiler; edit source comments or centralized configuration, then regenerate.

The [basic application](../examples/basic/main.go) demonstrates the generated import and one startup mount. For another application, set dir to its real package and preserve the same generate/check output directory. Keep the core version pinned in go.mod. Install the CLI at the application's fixed adapter version and record its version output, including the core version. Invoke the installed binary when native exit codes matter; go run wraps program failures.

## Structured outputs and exit codes

| Command | Evidence |
| --- | --- |
| version | JSON module, adapter/core versions, frontend identity, Go version, and format versions. |
| generate | JSON status, fingerprint, and candidate template count after writing the owned output. |
| check with dir/output | JSON status after complete regenerated-byte comparison; no generated-file write. |
| check with spec | JSON Report for a document; does not check source freshness. |
| explain with symbol | Handler: original top-level template fields plus `explanation`; projected field/type: structured sources, uses, declarations, and guidance. `--response` selects a handler response. |

Successful commands exit 0. Runtime, loading, generation, or document errors exit 1; invalid flag syntax exits 2. Help exits 0. Setup errors use a JSON stderr envelope with code, severity, and message; flag usage text is human-readable stderr. Capture streams separately and use the native process status. A successfully generated candidate set can contain unresolved templates; actual selected routes must still pass Build or Mount.

Use diagnostic codes and structured fields rather than matching translated prose. For a build-profile mismatch, regenerate under the application's actual target and tags. For gin-swagger.handler.ambiguous, inspect real handler identity and the Bundle index before supplying one centralized Config.Bindings mapping keyed by the original METHOD /Gin/path. Do not guess operation keys or wrap existing handlers to change their identity.

## Keep the application contract intact

Ordinary comments provide business meaning and explicit semantic constraints. Type names, field names, and wire structure come from source and actual codecs. A declaration is not evidence that the server enforces it. Unknown custom renderers, unresolved writers, and unsupported asynchronous behavior require explicit centralized rules.

Config.Groups creates complete documents for the definition selector; operation tags group routes within one document. Each group intersects Config.Include and cannot expand its scope. The example exposes only Bearer authorization, keeps tag filtering disabled, and disables request submission unless methods are explicitly enabled through UI.SubmitMethods.

Validate the existing routes before and after mounting and compare actual samples against generated contracts. The independent consumer test can select a remote fixed version with GIN_SWAGGER_TEST_VERSION; GOWORK must be off and replacements absent for that mode. A local workspace check does not establish independent module consumption.

## Explain a field or response

```sh
GOWORK=off go run ./cmd/gin-swagger explain --dir ./examples/basic --symbol github.com/openapi-golang/gin-swagger/examples/basic.User.Name
GOWORK=off go run ./cmd/gin-swagger explain --dir ./examples/basic --symbol github.com/openapi-golang/gin-swagger/examples/basic.CreateUser --response 201
```

A field explanation reports its original declaration, actual wire name, handler/media/status uses, applied codec and type rules, and semantic declarations. `implementation: "not-proven"` preserves the difference between a contract and verified business enforcement. Imported and generic origins use their original declaration identity. The source snapshot is captured during compilation; queries do not execute business handlers.

A handler query preserves the previous top-level `key`, `symbol`, `operation`, `source`, diagnostics, and variants. Additional details are under `explanation`. Response queries return the selected final response contracts with request conditions and components. Payload origins precede optional response wrapping; `transformed: true` identifies that distinction. Original handler diagnostics remain visible with their scope and centralized fixes.

`--max-explain-bytes` limits additional captured evidence; zero uses sixteen MiB. It does not limit the existing Bundle or total JSON response size. Capture only runs for explain and does not change generated Bundle data. Missing symbols, missing response statuses, evidence overflow, and output write errors return a structured failure with exit 1. A known source-root field excluded from all projected contracts returns empty uses with guidance, not a fabricated Schema.

The implementation uses the core public `Options.Explain` and `Result.Explain` APIs. For unknown helper effects, use centralized frontend rules; for custom wire behavior, use the public WireCodec or TypeMapper boundaries. Keep DTOs, handlers, and routes unchanged, and test enforcement with actual samples.
