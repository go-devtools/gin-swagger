# Handler identity and route bindings

`OperationKey` identifies a source contract. Gin's final handler name and code address are runtime evidence. The public OpenAPI `operationId` identifies an endpoint and defaults to its method and normalized path; renaming a function does not change that default ID.

Ordinary named functions, unexported handlers and aliases of those functions match by complete runtime symbol. A shared code address does not distinguish closure captures or receiver instances. The linker does not strip `.func1`, `-fm` or generic suffixes to manufacture a match. Ambiguous selected routes fail with `gin-swagger.handler.ambiguous`.

## Centralized bindings

Inspect `bundle.Index()` and select the exact existing template key. Add `Config.Bindings["GET /original/gin/path"] = entry.Key` once in documentation configuration. Preserve the existing handler and route registration. Bindings use the original Gin path before OpenAPI normalization and intersect the same `Include` scope as automatic matching.

A binding asserts which template applies; it does not prove the template covers every captured value or remove its diagnostics. Value methods retain state copied at registration, while pointer methods may observe later state changes. If these differences only affect values within one proven wire shape, an explicitly selected common template can describe both. State-dependent status codes or unresolved payload types must still be analyzed or declared through a project rule. A binding alone cannot repair them.

## Constructors and generic handlers

A function returning `gin.HandlerFunc` is a constructor, not an executing request handler. A project frontend can explicitly nominate a known constructor through `Frontend.Match`. Its `Entry` rule must not add Gin's implicit empty response to the constructor itself. A response annotation on that nominated constructor remains **declared** in the report; compilation does not execute the constructor or its returned closure. Validate actual instances against that declaration before trusting a shared contract.

Generic runtime names do not specialize source templates. A common text response can cover several instantiations when explicitly bound and verified. An unbound payload such as `var value T; c.JSON(200, value)` remains unresolved; its zero value is not automatically `null`. Concrete pointer, slice and map zero values retain their normal nil semantics. Do not infer type arguments from runtime-name spelling.

## Verified boundary

The real-source integration and independent module consumers exercise direct functions, aliases, private handlers, two closures sharing code with different captures, value and pointer receivers, and `int`/`string` instantiations. They compare HTTP status, headers and bodies before and after mounting, then independently validate positive and negative payloads. Unknown receiver statuses and generic payloads remain rejected after explicit binding; unused candidates stay outside the document.

The complete matrix also runs in ordinary, `-trimpath`, `-ldflags='-s -w'`, and combined binaries. Resulting document bytes, component identities and operation IDs must be identical. These guarantees concern the tested common contracts and explicit bindings; they do not establish arbitrary closure-state identity or complete middleware-chain discovery.
