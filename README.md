# gin-swagger

[Simplified Chinese](README.zh-cn.md)

Non-invasive Gin source contract generation and runtime integration for native OpenAPI 3.2.

This pre-1.0 SDK evolves between pinned versions. Use the public APIs and check the documented capability boundaries before depending on advanced behavior.

## Requirements

- Go 1.27.1 for development and verification.
- Gin v1.12.0 or the version pinned in go.mod.
- The module pins `github.com/openapi-golang/openapi` to `v0.0.0-20260907071604-72140a9a7490`, resolved from an actual remote commit.

## Architecture

Go source and real codec behavior define structure; ordinary comments provide business meaning. Documentation generation must not change business DTO tags, handler bodies, signatures, or existing route registration.

The core owns type projection, comments, neutral effects, Bundle and OpenAPI models. Framework adapters own framework call semantics, route syntax, handler evidence and mounting. Runtime document construction never imports the compiler or reads application source.

## Capability boundaries

- **Automatically derived:** types, supported wire representations and recognized source effects, as verified by implementation tests.
- **Explicitly declared:** semantic constraints and advanced contracts; these are not proof that the server enforces them.
- **Centrally adapted:** custom codecs and unsupported project helpers through explicit Go extension points.
- **Unresolved:** ambiguous or unsupported behavior must produce a diagnostic rather than a guessed response.

Future Fiber and Echo adapters are extension directions only. They are not products delivered or claimed as supported by this repository.

## Swagger UI examples and grouping

The [basic example](examples/basic) includes the following working routes. Resource writes are demonstrations and are not persisted.

| Method | Route | Behavior |
| --- | --- | --- |
| GET | `/examples/items/:id` | Read a resource; `missing` demonstrates 404. |
| POST | `/examples/items` | Create a sample resource and return 201. |
| PUT | `/examples/items/:id` | Replace the sample fields. |
| PATCH | `/examples/items/:id` | Apply non-null fields; `false` is an explicit update. |
| DELETE | `/examples/items/:id` | Return 204 without a response body. |
| GET | `/examples/legacy/items/:id` | A deprecated operation with a replacement link in its description. |

`Config.Groups` defines complete documents for the top-right **Select a definition** menu. Each group has a stable `ID`, display `Name`, and optional `Include(method, path)` predicate over the original Gin route path. A group always intersects the top-level `Config.Include` scope. `Config.DefaultGroup` selects the initial group. Mounting builds all documents before adding routes; requests read cached bytes from `/docs/groups/<ID>.json`. `/docs/openapi.json` remains the full configured overview.

Tags group operations *within* the selected document. Declare operation tags using ordinary `@openapi tags=[...]` comments, and configure descriptions and order through `OpenAPI.Tags`. `UI.Filter`, `UI.DocExpansion`, `UI.TagsSorter`, and `UI.OperationsSorter` control filtering, expansion and sorting. The example keeps the tag filter disabled and offers All endpoints, Users and resources, Types and enums, Authentication, and Legacy · Deprecated.

The authentication group contains one Bearer-protected operation. **Authorize** accepts `demo-token` without a `Bearer` prefix; this intentionally public demo value only protects the new example route. The original `/users` operation is unchanged. The enum example includes named requests for `admin/0`, `editor/1`, and `viewer/2`, plus the complete allowed values in Schema view.

The example project uses English OpenAPI comments, operation descriptions, document groups, and field descriptions. Enum labels include `"admin" - Administrator`, `"editor" - Editor`, and `0 - Pending`.

Enum descriptions come from ordinary comments on the typed constants. Generated `x-enum-descriptions` entries stay aligned with the standard `enum` array; the shared UI renders each as a value followed by its meaning. Missing descriptions do not invent a label.

The UI preserves a selected registered definition across refreshes and deep links. Other query-string overrides remain disabled. Submitting API requests is disabled by default; enable specific lower-case methods explicitly with `UI.SubmitMethods` when appropriate.

## Development checks

Use `GOWORK=off make dev`, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and `GOWORK=off go mod verify` with the pinned core and no local replace. The independent consumer test supports a fixed remote version through `GIN_SWAGGER_TEST_VERSION`.

For a new checkout, run `GOWORK=off go mod download` before `GOWORK=off make dev`. This keeps initial dependency downloads outside the generator's default one-minute budget. Direct CLI calls may select a longer budget with `--timeout=5m`.

## Response rendering

The frontend derives text, raw data, reader, explicit standard renderer, and immediate-status behavior from actual Gin calls. See the [response and verification guide](docs/responses.md) for supported cases and remaining boundaries.

## License

New project code is licensed under [MIT](LICENSE). Third-party assets retain their original licenses and notices.

Project-owned source comments, OpenAPI descriptions, diagnostics, CLI help, example text, and commit messages use English. Multilingual encoding tests preserve their actual input data. Upstream assets retain their original form; [README.zh-cn.md](README.zh-cn.md) provides corresponding Chinese documentation.

See the [request binding guide](docs/requests.md) for explicit JSON, Query, URI, Header, FormPost, Multipart, and centralized custom-decoder rules.

The request-binding guide also covers mandatory Bind calls, correlated error results, committed 400/413 responses, and finite method/media selection for automatic Bind/ShouldBind and explicit Form. Declare default or per-route request media centrally through Config; the settings resolve documentation scope without changing request handling.

Build target settings, dependency source, overlays, and declared custom mapping configuration participate in generation freshness. Compile calls with TypeMapper functions must provide named JSON inputs through Options.Configuration; configuration values are represented by digests in the Bundle.

Gin `Build` and `Mount` verify the executable build conditions against the generated Bundle. Known mismatches fail before mounting documentation routes; missing build metadata is retained as warnings in `Document.Report()`. This does not replace source freshness checks in CI.

Raw PostForm, array/dictionary getters, FormFile, and SaveUploadedFile now preserve their body/query locations, encodings, and business error branches. See the [request guide](docs/requests.md) for tested behavior and remaining boundaries.

HEAD and redirect response semantics are validated against real HTTP servers; see the [response guide](docs/responses.md).

Gin SSEvent and standard sse.Event renderers produce native event itemSchema, preserving actual text/JSON payloads and transmitted metadata. See the [response guide](docs/responses.md) for the real HTTP matrix and remaining streaming callback boundaries.

Stream callbacks and JSON Encoder writes to the Gin response writer are supported through the public core callback SDK, including explicit NDJSON framing and stable long-lived SSE. See the [response guide](docs/responses.md) for tested behavior and remaining boundaries.

## AI-assisted integration

Start with [llms.txt](llms.txt) for a compact documentation index and the [AI integration guide](docs/ai-integration.md) for actual commands, structured diagnostics, and public API boundaries. Generated JSON and provenance provide evidence for integration decisions.

Function comments can declare request and response types through the shared core. Local and fully qualified generic types retain actual Go identities; matching declarations must agree with derived schemas. Unknown behavior still requires a centralized rule. See [request declarations](docs/requests.md#explicit-request-and-response-types).

See the [performance guide](docs/performance.md) for reproducible 100/1000-route generation, startup Build, document-read, and allocation benchmarks.

See the [independent CI guide](docs/ci.md) for pinned tools, private module access, offline browser checks, actual platform jobs, and fixed remote-version consumption.

The [explanation commands](docs/ai-integration.md#explain-a-field-or-response) report field and response origins, actual projection rules, and declarations without claiming business enforcement.

The pinned core uses `spec.Optional[bool]` for optional standard boolean fields. Use `spec.Set(false)` to preserve explicit false and read `.Value` when testing a flag; see [native object migration](https://github.com/openapi-golang/openapi/blob/main/docs/native-objects.md).

The pinned core validates native HTTP object shapes and resolved parameter contexts, including Path Item inheritance, operation overrides, whole-query conflicts and Link operation identities across offline documents. `spec.Parameter.Name` preserves explicitly empty native query names. These checks do not alter Gin routes or business handlers; see the [HTTP validation boundaries](https://github.com/openapi-golang/openapi/blob/72140a9a7490829eea427274c3b317916cb2376d/docs/native-objects.md#http-objects-and-parameter-contexts).

The pinned core also validates native metadata field types, required-field presence, component names and license alternatives. See its [metadata rules](https://github.com/openapi-golang/openapi/blob/72140a9a7490829eea427274c3b317916cb2376d/docs/native-objects.md#document-metadata-and-component-names), including the explicit empty Request Body content policy.

Gin path encoding follows the actual Engine configuration. Build before Gin initialization, or retain `Config.RegisteredRoutes` from `Engine.Routes()` before initialization when escaped static colons are used. See [path encoding and route snapshots](docs/paths.md) for raw-path conditions, stale-snapshot diagnostics and mounting boundaries.

Closures, receiver methods and generic handlers use evidence-based matching or explicit centralized bindings. See [handler identity](docs/identity.md) for verified common contracts and ordinary, trimpath and stripped builds. Unknown generic payloads remain rejected.

Mounted documentation reuses the core native compatibility panel. It identifies omitted extension methods and tag metadata without rewriting the document. See [UI rendering and submission boundaries](https://github.com/openapi-golang/openapi/blob/72140a9a7490829eea427274c3b317916cb2376d/docs/swaggerui-compatibility.md).

The shared UI renders request/response stream item schemas separately from complete-body schemas, preserving finite NDJSON/SSE bytes. Whole-query parameters remain read-only because the pinned client omits their values during serialization; a structured browser diagnostic explains the limitation.
