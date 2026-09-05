# gin-swagger

[简体中文](README.zh-cn.md)

Non-invasive Gin source contract generation and runtime integration for native OpenAPI 3.2.

This repository is under active implementation. The full acceptance target is recorded in [GOAL.md](GOAL.md); current evidence and remaining work are tracked in [status](docs/status.md) and [verification](docs/verification.md). It is not yet a completed or released product.

## Requirements

- Exactly Go 1.27.1 for minimum-version acceptance.
- Gin v1.12.0 for minimum-version acceptance.
- The module pins `github.com/openapi-golang/openapi` to `v0.0.0-20260905135839-1c9eeff38e45`, resolved from an actual remote commit.

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

## Verified module snapshot

`GOWORK=off make dev`, `go test -race ./...`, `go vet ./...`, and `go mod verify` pass with the pinned core downloaded into the module cache and no local `replace`. This verifies the current implementation against a real remote version; final cold-cache, CI, and full capability acceptance remain tracked in [verification](docs/verification.md).

## License

New project code is licensed under [MIT](LICENSE). Third-party assets retain their original licenses and notices.
