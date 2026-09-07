# Gin paths and route snapshots

The adapter reads methods, dynamic prefixes and final handlers from `Engine.Routes()`. It converts Gin path syntax and request-path settings before passing neutral routes to the core. Build and Mount do not initialize the business Engine, execute a request, inspect Gin internals or alter routing settings.

## Encoding

`UseEscapedPath` takes precedence over `UseRawPath`, matching Gin 1.12.0:

| Engine settings | Registered path | Documented path |
| --- | --- | --- |
| Default decoded `URL.Path` | `/users/张三` | `/users/%E5%BC%A0%E4%B8%89` |
| Default decoded `URL.Path` | `/percent/%2F` | `/percent/%252F` |
| `UseEscapedPath = true` | `/percent/%20` | `/percent/%20` |
| `UseRawPath = true` | `/percent/%2F` | `/percent/%2F` |
| `UseRawPath = true` | `/percent/%20` | `/percent/%2520` |

For decoded paths, percent signs, spaces, braces, query/fragment punctuation and UTF-8 bytes in static route segments are encoded as URL path literals. Gin parameters such as `:id` become `{id}`; parameter names are template identifiers, not static URL bytes. A static escaped colon such as `/jobs/run\:now` becomes `/jobs/run:now`.

Escaped mode preserves valid static percent escapes, including their case. Unescaped static bytes that OpenAPI cannot represent directly fail with `gin-swagger.path.escaped`; the adapter does not publish a different address that would miss the registered route.

Raw mode is conditional: Gin uses `URL.RawPath` only when it is nonempty, otherwise it uses decoded `URL.Path`. A noncanonical static escape such as `%2F`, `%2f` or `%41` keeps the raw branch active. Otherwise the document uses the canonical URL for the decoded fallback. If that fallback has encoded static literals and variable parameters, `x-gin-raw-path-note` explains that noncanonical parameter escapes can switch branches and prevent the static prefix from matching. For example, `/percent/%20/:id` is documented as `/percent/%2520/{id}`: a canonical `a%20b` value matches, while `a%2Fb` makes the whole raw prefix differ and returns 404. This conditional behavior is not expressible as an unconditional OpenAPI path template.

Nondefault modes expose `x-gin-path-mode` and the actual `x-gin-unescape-path-values` setting. Parameter decoding still follows Gin, including its use of query unescaping in the raw/escaped branch. Catch-all parameters retain `x-gin-catch-all` and the existing cross-slash compatibility note. These extensions describe framework behavior; they do not change routing, authorize reserved parameter values or guarantee every generated client's serialization.

## Initialization and escaped colons

Prefer Build or Mount after all business routes are registered and before `Run`, `RunListener` or the first `ServeHTTP`. Gin removes the escape marker from static colons during initialization. Its later public route list cannot distinguish a single `/jobs/run\:now` static registration from a `/jobs/run:now` wildcard registration.

If documentation must be built after initialization, capture the public snapshot first and retain it in configuration:

```go
router := newBusinessRouter()
cfg := ginswagger.Config{
    OpenAPI: openapi.Config{Title: "Service", Version: "1"},
    RegisteredRoutes: router.Routes(),
}
// Retain cfg before Gin initializes the router; pass it to later Build calls.
document, err := ginswagger.Build(router, apidoc.Bundle, cfg)
```

This is documentation configuration, not a replacement router. The saved slice must be the complete, unmodified `Engine.Routes()` result captured after registration and before initialization. Build compares its route count, methods, paths and final-handler evidence with the current Engine. Only Gin's static-colon unescaping is tolerated. Added routes, changed prefixes or changed evidence fail with `gin-swagger.routes.stale`; refresh a snapshot while original registration syntax is still available.

Saved original paths are also the keys used by `Include`, group filters, `Bindings` and request-media overrides. They remain stable across initialization. Mount's isolated preflight uses the same validated original paths, so static and dynamic colon routes can coexist without a false conflict.

Without saved evidence, duplicate method/path entries caused by collapsed static and dynamic colons fail with `gin-swagger.routes.ambiguous`. A single lost escape is not detectable through Gin's public API: callers must follow the startup timing above or provide the original snapshot. Capturing an already initialized router does not recover missing information. Function pointers only corroborate route evidence; they never establish the identity of captured closure values or receiver state, which still requires centralized `Bindings` when ambiguous.

Do not mutate the Engine, snapshot slice or configuration concurrently with startup Build/Mount. A snapshot captured before Mount intentionally becomes stale after the documentation routes are registered; serve the immutable returned document rather than rebuilding it per request. Tests exercise actual requests before and after initialization, duplicate collapsed paths with the same handler, stale snapshots, preflight mounting and conditional raw-path failures.

References: [Gin 1.12.0 path selection and route enumeration](https://github.com/gin-gonic/gin/blob/v1.12.0/gin.go), [OpenAPI 3.2 path templating](https://spec.openapis.org/oas/v3.2.0.html#path-templating).
