# Response rendering and integration checks

The Gin frontend recognizes calls by their full package and type identity and translates them through the public `openapi/compiler` SDK. Business handlers and route registration remain unchanged. The shared analyzer owns branching and write order; Gin call and renderer rules stay in this module.

| Calls | Derived representation |
| --- | --- |
| `JSON`, `IndentedJSON`, `AsciiJSON`, `PureJSON` | Core JSON projection of the actual payload |
| `String` | A formatted plain-text response |
| `Data` | Raw bytes with a statically known media type |
| `DataFromReader` | Raw bytes, known length or explicit negative length, and analyzable extra headers |
| `Render` with Gin `render.JSON`, `IndentedJSON`, `AsciiJSON`, `PureJSON`, `String`, `Data`, or `Reader` | The corresponding known renderer, including its concrete fields |
| `Redirect` | Method- and header-dependent HTML or no body, actual pending/committed status, and Location |
| HEAD routes | Core HTTP projection removes content while preserving metadata and GET contracts |
| `Status` | A pending status, with Gin's default 200 provided at entry |
| `AbortWithStatus`, `AbortWithError` | Immediate status commit; Abort does not return from the current Go function |
| `Header` | Case-insensitive replacement or deletion, retaining only pre-commit values |

Raw binary schemas carry `contentMediaType` without a JSON string type or Base64 constraint. They do not claim the bytes satisfy a logical JSON payload schema. Custom renderer formats, dynamic media types, unresolved reader length/headers or preserved pending status, SecureJSON prefixes, JSONP, and unrecognized writer methods remain diagnostics requiring centralized rules. A reader's duplicate extra-header names with different casing are rejected because map iteration cannot establish a single result.

Status 204 and 304 samples have no response content or reader-generated length/extra headers; an unused JSON payload is not projected. An AbortWithError call does not invent an automatic JSON error body. A later JSON write can retain an already committed error status. A second complete body write on the same path is not treated as an alternative response. HEAD/redirect behavior has the actual HTTP coverage described below. Interim-response sequences and file/range/stream matrices still need their full acceptance coverage. Interim statuses are diagnosed explicitly: a real HTTP test demonstrates that a Gin 103 followed by a nominal 201 can arrive as a final 200, so the frontend must not report 103 as its complete final contract.

`internal/integration` compiles a real source fixture, constructs equivalent Gin engines, mounts documentation on one, and compares actual status, commit-time headers, and body bytes. Text and JSON samples are validated by the independent contract engine. Binary bytes are compared directly and their OpenAPI representation is checked separately. The fixture source is compared before and after compilation. Unsupported routes are selected separately to prove that diagnostics neither disappear nor contaminate unrelated routes.

`internal/verify` checks the runtime dependency graph and exercises an independent consumer module through first generation, deterministic regeneration, freshness checking, real HTTP contract tests, a `-trimpath -ldflags='-s -w'` application build, and runtime document export. Set `GIN_SWAGGER_TEST_VERSION` to a synchronized remote version to reject adapter replacements; without it, development uses a temporary replacement of this adapter checkout. The core dependency always remains fixed to the version in go.mod, without a local replacement. This mode choice is reported by the test and must not be confused with cold-cache acceptance.

Run the module's `make dev` after downloading dependencies. The dedicated checks are also available through `go test ./internal/integration` and `go test ./internal/verify`. Actual stage results and remaining full-goal work are recorded in [verification.md](verification.md).

## Redirects and HEAD on the real HTTP wire

`Context.Redirect` follows Gin 1.12.0 and Go 1.27.1. Without an existing Content-Type, GET writes HTML and commits; HEAD sets HTML metadata without a body; other methods only set the pending status and Location. An existing Content-Type suppresses the generated HTML. A pending redirect can therefore be replaced by a subsequent response. A GET redirect followed by another body is mixed output and produces a diagnostic. Committed status and wire headers stay frozen, while subsequent header mutations remain visible to rendering decisions through the public core response snapshot.

Known destinations independent of the request path retain their exact normalized Location. Existing percent escapes, absolute URIs, root-relative path cleaning, trailing slashes, non-ASCII byte escaping, and HTTP header whitespace trimming follow the standard library. Relative or dynamic destinations retain a string header schema without claiming a fixed value. Invalid or unknown redirect statuses are diagnosed. Explicit `Render` with a custom or unrecognized redirect renderer still requires a centralized rule; this support does not treat arbitrary request-object mutation as proven method evidence.

The real-server matrix covers 72 combinations: six methods and twelve redirect cases, including 201, 301/302/303/304/307/308, prior committed 409, custom/deleted/late media headers, and absolute/relative/dynamic destinations. Mount is compared against an equivalent unmounted engine for status, headers, and body bytes. Separate tests verify continued POST output, mixed GET diagnostics, and GET/HEAD reuse of one handler. HEAD is tested through an actual HTTP server; ResponseRecorder alone does not establish wire-body suppression. Native response summary/description metadata and local shared responses are handled by the core.

See the [Go Redirect contract](https://pkg.go.dev/net/http#Redirect) and the fixed dependency source for framework-specific behavior. This increment leaves file/range/stream outputs, interim sequences, arbitrary custom renderers, full request mutation tracking, and the complete browser matrix in the active Goal.
