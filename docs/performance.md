# Performance measurement

Run the benchmark from the module root with Go 1.27.1, Gin v1.12.0, and the exact core version pinned in `go.mod`:

```sh
GOWORK=off go mod download
GOWORK=off go test ./compiler -run '^$' -bench '^BenchmarkGinScale$' -benchtime=500ms -count=3 -benchmem
```

Dependencies must already be available in the configured module cache. Fixture loading sets `GOWORK=off` and `GOPROXY=off`. Record the actual toolchain, platform, CPU, module versions, cache state, command, and every repeated sample with published results.

| Stage | Work inside the measured loop |
| --- | --- |
| `Generate` | Load and analyze 100 or 1000 distinct ordinary Gin handlers through the public compiler SDK and Gin frontend, project their request and response contracts, then format and atomically write generated Go. |
| `CoreBuildSharedHandler` | Link a precompiled Bundle and prebuilt neutral route snapshot through `openapi.Build`. |
| `GinBuildSharedHandler` | Read actual `Engine.Routes`, normalize paths, automatically match the compiled handler symbol, and call core Build. |
| `Read` | Return a defensive copy of cached JSON through `Document.JSON`. |

Generation uses copies of the actual `internal/benchfixture` handler in a temporary module with the repository's dependency versions. Each handler binds a tag-free JSON request and returns typed JSON for success or failure. Setup verifies that compilation produced exactly 100 or 1000 distinct templates.

The runtime workload separately registers 100 or 1000 actual POST routes against one ordinary compiled handler. It uses automatic symbol matching, with no explicit binding map. Both build measurements use the same Bundle, the same routes, and runtime-build verification. Before timing, setup checks that the core and Gin document bytes are identical and that all registered routes appear in the document. Route registration, fixture compilation, and baseline checks are outside the measured build loops.

The paired build stages help isolate adapter work, but each includes core construction, validation, and serialization. A subtraction of noisy timings is not a precise standalone adapter latency. This shared-handler runtime workload also differs from the core repository's distinct-handler workload. Generation and runtime scenarios deliberately exercise different counts of handler templates.

Generation excludes CLI process startup and compilation of the generated consumer application. `Read` includes its owned copy, but excludes HTTP transport and UI rendering. Handler execution is not timed, and no business-request throughput or zero-overhead claim follows from these measurements.

`B/op` and `allocs/op` measure cumulative Go allocations in the benchmark process. They exclude allocations in external Go tool processes and do not represent retained memory or peak RSS. Repeated samples normally use warm build and filesystem caches. Cold-cache memory and startup costs require a separate experiment.
