# Request binding

The frontend recognizes actual Gin methods and explicit exported binders by full package and go/types identity. It emits shared compiler effects and supplies `BindingCodec`; annotation parsing, enum handling, Schema construction, component identity, and parameter expansion remain in the core. Application DTOs and handlers need no new tags or wrappers.

| Gin call | Derived request contract |
| --- | --- |
| `ShouldBindJSON`, `ShouldBindBodyWithJSON` | Standard JSON body projection |
| `ShouldBindQuery` | Named query parameters using repeated-value form serialization |
| `ShouldBindUri` | Required parameters matched by their actual URI field names |
| `ShouldBindHeader` | Named, case-normalized scalar headers |
| `ShouldBindWith(..., binding.JSON)` | Explicit JSON body |
| `ShouldBindBodyWith(..., binding.JSON)` | Explicit cached JSON body |
| `ShouldBindWith(..., binding.Query)` | Query parameters, including supported local aliases of this binder |
| `ShouldBindWith(..., binding.Header)` | Scalar header parameters |
| `ShouldBindWith(..., binding.FormPost)` | URL-encoded body fields without query fallback |
| `ShouldBindWith(..., binding.FormMultipart)` | Multipart text fields and uploaded files |

`BindingCodec{Mode: ...}` is public for custom generation entry points. Valid modes are `query`, `uri`, `header`, `form-post`, and `multipart`. Its field selection follows Gin rather than encoding/json: existing simple form/uri/header names are read when present; no tag is required. Exported anonymous struct children are traversed, and an anonymously embedded time.Time does not invent a bindable Time parameter. Conflicting field names are diagnosed instead of applying JSON promotion precedence.

Text input pointers express absence through an optional parameter/property, not a JSON null value. Slices and fixed arrays are collections; byte slices remain byte lists rather than Base64. Named enums and field constraints from ordinary comments survive projection. A time.Time text field uses RFC3339 date-time, and time.Duration uses Go duration text. The same DTO can independently retain JSON response behavior: for example, a query byte list can produce a Base64 JSON response field.

Multipart FileHeader values describe raw uploaded content with contentMediaType and no JSON string/Base64 constraint. A list of file headers becomes an array of files, not an array of Go metadata objects. Tests send real multipart bodies and separately verify actual response bytes and generated input representation.

## Declarations and binding errors

Declared required/minLength/enum constraints define the client contract; they do not prove that Gin or business code enforces those declarations. Tests distinguish real binder rejection from independent contract validation. The example invalid numeric, array-length, header, URI, form, and malformed JSON requests keep their actual 400 response branches after mounting documentation.

The compiler does not yet infer body-level required from all binder error and continuation paths. Field-level required remains separate. Raw form/file getters are covered below; the complete decoder/tag matrix still requires additional implementation. Automatic selection and binding.Form now use the finite conditions described below.

## Centralized custom decoding

Custom UnmarshalParam methods produce a diagnostic until an explicit TypeMapper describes their input representation. Register that mapper through `core.Options.Mappers` alongside `gincompiler.Frontend()`. TypeMapper rules run before BindingCodec's type rules. The integration test `TestCustomBindingMapper` uses real source and an actual custom decoder to prove this extension works without changing its DTO or handler.

Default repeated header or URI collections cannot be mislabeled as comma-separated OpenAPI parameters. Nested named struct fields with JSON-text versus flattened fallback behavior, recursive embedded structures, dynamic map/interface inputs, and unmodeled tag options likewise retain diagnostics. Existing binding/validate tags are not repurposed as documentation metadata. XML/YAML/TOML/ProtoBuf and other distinct codecs require their own actual wire rules.

## Verification

The dedicated integration package compares equivalent engines before and after mounting and validates actual responses through an independent Schema engine. Parameter schemas validate decoded values; their real text serialization is exercised through HTTP requests. Multipart byte representation is checked separately from JSON instance validation. Compiler input files are checked for byte equality before and after generation.

The independent-consumer fixture now includes query binding, repeated byte values, malformed numeric input, first generation, and a stripped application build. Fixed remote validation must explicitly set `GIN_SWAGGER_TEST_VERSION` or install a fixed remote CLI. A local replacement validates development changes; a fixed remote version validates independent module consumption.

## Mandatory binding and committed errors

`BindJSON`, `BindQuery`, `BindHeader`, `BindUri`, and supported `MustBindWith` binders now use the core's public CallOutcomes API. Successful and failed calls return distinct nil/non-nil alternatives. The adapter emits Gin-specific commits; the core owns subsequent Go control flow. The corresponding explicit `ShouldBind` methods also expose correlated error results without implicit response commits.

For the fixed Gin v1.12.0 standard JSON profile, ordinary mandatory-binding errors commit 400 and a propagated http.MaxBytesError commits 413. Body binders can encounter a request-body limit; custom text decoders can also return that error type. BindUri itself always commits 400 on failure. These are possible failure paths, not a requirement for the adapter to install a body limit or change existing middleware.

Abort does not return from the current handler. Ignoring the binding error and writing JSON afterward keeps the committed error status. Trying to write 422 on the error branch cannot override 400 or 413. Returning immediately leaves the failure response bodyless. Integration tests issue real valid, malformed, and explicitly size-limited requests and compare status, headers, and body before and after mounting. They independently validate emitted JSON bodies and assert the absence of invented status branches.

The implementation does not generalize the default codec's MaxBytesError behavior to unverified sonic/go-json build profiles. Automatic Bind/ShouldBind selection now uses explicit finite conditions rather than implicit JSON assumptions.

## Automatic selection and explicit Form

ShouldBind, Bind, and explicitly identified binding.Form now preserve finite method/media conditions in the generated Bundle. Source analysis projects every applicable case and its success/error continuation. Runtime linking uses Engine.Routes() for methods and centralized documentation settings for the intended media scope; no handler, DTO tag, or route registration needs modification.

```go
// Declare default automatic-binding media and centrally override the upload route.
cfg := ginswagger.Config{
    OpenAPI: openapi.Config{Title: "Service", Version: "1"},
    DefaultRequestMediaTypes: []string{"application/json"},
    RequestMediaTypes: map[string][]string{
        "POST /uploads": {"multipart/form-data"},
    },
}
```

RequestMediaTypes keys use the original Gin method and path, before path normalization. A present route entry overrides the default, including an empty entry that leaves that route unresolved. Use a single empty string to declare absence of Content-Type. Values are exact base selector strings without charset/boundary parameters or wildcards. These settings resolve documentation conditions; they do not install request filters or alter explicit binder behavior.

| Actual selection | Input sources |
| --- | --- |
| Automatic GET with non-multipart media | Form query fields; even a JSON/XML body is not selected as that codec |
| Automatic non-GET with application/json | JSON request body |
| Automatic non-GET with multipart/form-data | Multipart body fields and supported uploads, without query fallback |
| Form with POST/PUT/PATCH and URL-encoded media | Body fields followed by query fallback; repeated values preserve body-first order |
| Form with other methods and URL-encoded media | Query fields |
| Form with multipart media, including GET | Query fields followed by multipart text values; this is distinct from FormMultipart |
| Form with other/absent media | Query fields |

Automatic non-GET XML/YAML/TOML/ProtoBuf/MsgPack/BSON selections retain codec diagnostics until their actual wire rules are supplied. Those diagnostics do not block a selected JSON case or GET's Form behavior. Unknown media scope fails during Build/Mount instead of defaulting to JSON. Multiple media such as JSON and Multipart are supported when their parameter and presence contracts can be merged accurately.

For mixed Form sources, generated query parameters and body properties describe the same logical fields. Source rules record body-before-query or query-before-body precedence. A field-required declaration across these alternative locations is diagnosed because OpenAPI cannot simply require that field in both places. A centralized contract rule must resolve that ambiguity; the generator does not discard the declaration.

The real-request matrix covers 17 successful cases across GET, POST, PUT, PATCH, DELETE, and QUERY, including repeated-value precedence and explicit Form. Eight additional cases cover business 422, mandatory 400/413, and a GET body limit that must not invent an unread-body 413. Independent validators check response bodies, numeric request representation, non-null form arrays, and repeated query serialization. Default and per-route configuration, unresolved media, unsupported selected codecs, compatible multi-media output, and cross-location presence ambiguity have boundary tests.

## Custom mapping configuration and freshness

When passing Mappers to core.Options, also pass stable, named JSON settings in Configuration. These settings identify captured configuration that cannot be reconstructed from callback addresses. The integration example declares `{"version":1,"minLength":2}` for its custom-parameter rule; no DTO or handler changes are needed. The core includes the settings in freshness checks and serializes only their digest, not their original values. Update the declaration when mapping or external codec configuration changes.

The core's build-input profile records the actual Go target, CGO/experiment/architecture selectors, selected module and local workspace inputs, overlay snapshots, embedded and inactive source, and build-parameter digests. GoFlag values are not exposed in the generated Bundle. The generator uses read-only module mode and rejects executable packages drivers or tool wrappers; provide source overlays through the public LoadOptions.Overlay API.

## Raw getters and file saving

PostForm, DefaultPostForm, GetPostForm, PostFormArray/GetPostFormArray, and PostFormMap/GetPostFormMap now emit body-field effects. URL-encoded bodies are read for POST/PUT/PATCH; multipart text fields are read for all methods. These getters read PostForm rather than the combined Form map, so query parameters do not become fallback body values. Supported media can be derived directly for these explicit getters; no additional media setting is required.

Repeated values use array schemas and form/explode encoding. Bracket dictionaries use deepObject for URL-encoded/multipart body fields and QueryMap/GetQueryMap parameters; only string-valued dictionaries are modeled, without claiming arbitrary nested-object syntax. QueryArray/GetQueryArray retain repeated-value serialization. Explicit wire schemas prevent JSON null or Base64 conventions from leaking into text inputs. A constant DefaultQuery/DefaultPostForm fallback becomes a documentation default without changing the handler or making the field mandatory.

FormFile describes generic raw upload content rather than FileHeader metadata or Base64 JSON. It returns the first matching file; nil/non-nil file and error results remain correlated with business branches. It does not implicitly commit 400 or 413. SaveUploadedFile is modeled as filesystem work returning success/error, with HTTP responses derived only from subsequent business code. GetString reads application context without inventing a request parameter. Known nil file pointers and dynamic getter field names remain diagnostics, scoped to selected routes.

Ten real form cases cover five HTTP methods and two media types, including empty strings, defaults, repeated values, and query isolation. Further tests cover repeated/missing query values, both dictionary encodings, binary uploads and missing files, and saving bytes before and after documentation mounting. Saving failures use an occupied regular-file parent, since Gin intentionally creates missing parent directories. Tests verify that compilation and mounting do not write uploaded files. Independent validators check decoded inputs and actual responses; wire behavior is checked through real HTTP requests.

Core composition preserves multiple fields and whole-body constraints, while field required stays separate from body required. Full inference of empty-body rejection from arbitrary error paths, arbitrary MultipartForm map access, all file I/O helpers, all decoder/tag variants, and complete browser Try it out coverage remain work in progress. Read the [OpenAPI 3.2 encoding rules](https://spec.openapis.org/oas/v3.2.0.html#encoding-object) when adding custom serialization.

## Explicit request and response types

The shared core now resolves function-level request and response declarations against actual Go types. Gin uses the same implementation through its pinned core version:

```go
// Submit a request using the existing handler.
// @openapi request mediaType="application/json" type="Request" required
// @openapi response status=201 mediaType="application/json" type="Request"
// @openapi response status="default" mediaType="application/json" type="Request"
func Create(c *gin.Context) {
    // Keep the application's existing binding and response logic here.
}
```

Unqualified type names resolve in the handler package. Fully qualified generic references can name other explicitly loaded packages without adding unused imports. The standard CLI loads application source packages; an annotation does not initiate extra module downloads. Central generation tools can use the public `Project.TypeIn` API.

Matching declared and derived wire schemas must agree. Declaring body presence as required is a client constraint, not proof of actual empty-body rejection. Derived error responses remain present, and unknown status/helper diagnostics still prevent selected routes from building. Invalid unused candidates remain isolated. Bodyless responses support `@openapi response status=204`.

The independent consumer verifies the declaration path using real valid/malformed JSON requests, a bodyless DELETE response, preserved provenance, and independent Schema sample validation. It separately selects a conflicting type declaration and an unknown dynamic-status handler to confirm failure. No adapter access to core internal packages is required.

See the core's [request and response declaration reference](https://github.com/openapi-golang/openapi/blob/main/docs/request-response-declarations.md) for exact syntax, scoped type expressions, codec reuse, conservative conflict checks, and structured diagnostics.
