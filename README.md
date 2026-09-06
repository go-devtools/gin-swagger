# gin-swagger

[Simplified Chinese](README.zh-cn.md)

Non-invasive Gin source contract generation and runtime integration for native OpenAPI 3.2.

This pre-1.0 SDK evolves between pinned versions. Use the public APIs and check the documented capability boundaries before depending on advanced behavior.

## Requirements

- Go 1.27.1 for development and verification.
- Gin v1.12.0 or the version pinned in go.mod.
- The module pins `github.com/openapi-golang/openapi` to `v0.0.0-20260906035319-560c9adbb5be`, resolved from an actual remote commit.

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

Project-owned code comments, diagnostics, CLI help, and example text use English. Multilingual encoding tests retain their input values through escaped literals. Upstream assets retain their original form. Commit messages use English; [README.zh-cn.md](README.zh-cn.md) provides corresponding Chinese documentation.

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
