# go-exotic architecture

`go-exotic` ports the core ideas of upstream `exo` to Go while using `go-pherence` as the local inference runtime.

## Reference boundaries

- Upstream exo checkout: `/workspace/references/exo`
- Go runtime dependency: `/workspace/projects/go-pherence`
- This module must not vendor upstream exo; it is a reference for behavior and architecture.

## Package ownership

| Area | Package | Responsibility |
| --- | --- | --- |
| Domain model | `internal/exotic` | Devices, layer shards, JSON shape, domain validation |
| Placement | `internal/placement` | Deterministic memory-weighted baseline planner, contiguous coverage validation, JSON placement export |
| Cluster membership | `internal/cluster` | Peer identity, local capability exchange, heartbeat refresh, in-memory registry, stale-peer eviction |
| Routing | `internal/router` | Validated route construction from shard plans, capability advertisements, and registry snapshots; route preview DTO construction; cancellation propagation |
| Runtime adapter | `internal/runtime` | Minimal `go-pherence` adapter for metadata, tokenization, local full-model generation, and future shard execution contract |
| Protocol/API | `internal/protocol`, `internal/server` | DTOs and HTTP surfaces for health, capabilities, placement previews, route previews, and gated shard execution |
| CLI | `cmd/go-exotic` | `plan`, `run`, `peers`, `routes`, and `serve`; distributed generation remains disabled by default |
| Simulation/integration | `internal/sim`, `internal/integration` | In-process routed shard execution harness plus multi-node placement/routing validation |

## Porting model

1. Start local-only and deterministic.
2. Keep shard ranges explicit and validated.
3. Add `go-pherence` runtime calls behind a small interface.
4. Simulate multi-worker orchestration in-process before enabling remote shard execution.
5. Compare distributed output against local full-model `go-pherence` output before enabling networked generation.

## Transport decision

The initial cluster transport is **HTTP over LAN**. This is deliberately simpler than upstream exo routing/discovery internals and gives `go-exotic` a stable contract for health, capabilities, placement previews, and future shard execution before exploring richer transports. UDP/mDNS/gRPC/exotic transports are deferred until the in-process and HTTP paths are validated.

Current HTTP endpoints:

- `GET /health`
- `GET /capabilities`
- `GET /placement/preview?layers=N&model=...`
- `GET /routes/preview?layers=N&model=...`
- `POST /shards/execute` only when explicitly wired; disabled by default in the CLI server

`serve` is quiet by default and emits structured JSON diagnostics only with `-verbose`. `serve -shard-model /path/to/model` installs a local-development shard worker but does not enable LAN distributed generation.

## Current cluster validation

Integration coverage runs multiple in-process peer capabilities through HTTP placement and route preview endpoints, validates route construction, and exercises the gated shard endpoint under `httptest`. Remote/LAN generation remains disabled by default.

## Runtime boundary

The current `internal/runtime.Adapter` supports:

- model metadata
- tokenization
- local full-model generation

`runtime.PherenceLayerExecutor` implements local layer-range execution over `go-pherence/model.ForwardLayer`. `protocol.ShardExecutionRequest` and `protocol.ShardExecutionResponse` carry session/request IDs, explicit layer ranges, positions, hidden size, and activation vectors. `protocol.ActivationPayload` defines the first transport serialization format: flat float32 little-endian bytes with hidden-size metadata.

## Validation baseline

- `make check` must pass after every code batch.
- Runtime behavior changes need focused smoke commands.
- Follow-up commits include `Signed-off-by: Rui Carmo <rui.carmo@gmail.com>`.

## Single-host simulation

`internal/sim` executes routed shard requests sequentially in one process. It can use synthetic workers, local `LayerExecutorWorker` instances, or explicit remote-worker maps backed by `cluster.RemoteShardWorker` in tests/orchestration. Default generation paths do not call the remote-worker helper.

## Synthetic token generation harness

`Simulator.GenerateTokens` runs a synthetic decode loop through routed in-process shard workers and maps the final activation to a token via a caller-provided projector. This validates orchestration, cancellation, and activation handoff through N shards before real `go-pherence` layer execution or numerical parity checks are attempted.

## Parity status

The current parity gate compares the `go-exotic` local runtime adapter against direct full-model `go-pherence` generation on the small SmolLM2 fixture when available. Synthetic shard simulation validates orchestration; local real-shard hidden-state and one-token output parity are covered by fixture-backed tests. Real LAN distributed numerical parity remains future work.

## Initial plan closeout

The initial development plan is complete through synthetic N-shard token flow and local adapter parity against direct `go-pherence` generation. The next implementation phase is to replace synthetic workers with real layer-shard execution and then compare distributed shard output against full-model `go-pherence` output.

## Real layer execution hook

Phase 7 starts with `go-pherence/model.ForwardLayer` as the smallest safe local API for layer-range execution. `runtime.PherenceLayerExecutor` wraps this hook behind `LayerRangeExecutor`, validates request/model/KV-cache shape, copies activation inputs/outputs, and checks cancellation before each layer. This remains in-process only; remote HTTP shard execution is still disabled.

## Simulator layer executor worker

`internal/sim.LayerExecutorWorker` adapts a `runtime.LayerRangeExecutor` to the simulator `Worker` interface and keeps float KV caches local to the worker. This allows Phase 8 to substitute real `go-pherence` layer execution for synthetic workers without adding network transport.

## Phase 7 audit status

Recent audits added guardrails around the real-shard scaffolding:

- `PherenceLayerExecutor` requires request `totalLayers` to match the loaded model layer count before validating/executing a shard range.
- `protocol.MaxActivationElements` bounds flat f32 activation payloads and shard request/response hidden sizes.
- `Simulator.Execute` verifies shard responses come from the route peer.
- `Simulator.GenerateTokens` rejects nil simulators and checks decode-position overflow.
- `NewSimulator` rejects typed-nil workers such as `WorkerFunc(nil)` and nil pointer workers.

These checks keep the local real-layer path safe while Phase 8 parity work is still pending.

## Real layer simulation status

A single-host integration test now routes a synthetic two-layer `go-pherence` model across two `LayerExecutorWorker` instances backed by `PherenceLayerExecutor`. This proves the real local `ForwardLayer` path can be driven through routing and simulation. Full numerical parity against direct generation is still pending.

## Phase 8 parity gate status

The integration suite now has a documented SmolLM2 fixture skip path: set `GO_EXOTIC_SMOLLM2_MODEL` or provide `../go-pherence/models/smollm2-135m`. When available, `TestRealShardHiddenStateMatchesSequentialForwardLayer` compares routed multi-shard simulator execution against direct sequential `go-pherence` `ForwardLayer` execution for one embedded token. One-token output parity now uses `go-pherence` `FinishCPUDecodeStep` to project both direct sequential and routed shard activations through the same final norm/LM-head path. Direct full-model adapter parity remains covered by a separate fixture-backed generation test.

## One-token output parity

`TestRealShardOneTokenOutputMatchesSequentialForwardLayer` verifies that routed multi-shard execution and direct sequential `ForwardLayer` execution produce the same final hidden state and greedy next token after `FinishCPUDecodeStep`. This closes the local Phase 8 output parity gate without enabling remote HTTP shard execution.

## HTTP shard execution surface

`/shards/execute` now exists as a disabled-by-default HTTP bridge. The default CLI server does not install a shard worker, so remote execution returns `503 shard execution disabled`. Tests can enable it with `server.WithShardExecution(worker, totalLayers)`. The wire format uses `protocol.ShardExecutionHTTPBridgeRequest` and `ShardExecutionHTTPBridgeResponse`, carrying activations as `ActivationPayload` flat f32 bytes rather than JSON float arrays. LAN execution remains gated behind future timeout/request-ID/client work.

## Remote shard client guardrails

`cluster.RemoteShardWorker` is the first HTTP client adapter for `/shards/execute`. It serializes requests with `ActivationPayload`, derives HTTP requests from the caller context, optionally wraps calls in a timeout, and sends `X-Go-Exotic-Request-ID`. The server rejects header/body request-ID mismatches, and the client rejects response request-ID mismatches. This is covered with `httptest`; LAN execution is still not enabled by the CLI.

## Phase 9 closeout status

Phase 9 has the first remote transport surface in place without enabling LAN distributed generation by default:

- `/shards/execute` is present but requires explicit `server.WithShardExecution` wiring.
- The default `go-exotic serve` command does not install a shard worker and returns `503 shard execution disabled`.
- `cluster.RemoteShardWorker` provides the client-side bridge with context cancellation, optional timeout, and request-ID propagation.
- HTTP transport tests run entirely under `httptest`; no LAN assumptions are required.

Remaining work is productization rather than transport scaffolding: LAN peer selection, routed generation orchestration, richer payload formats, and end-to-end networked generation.

## Explicit serve opt-in

`go-exotic serve` now accepts `-shard-model` as an explicit local-development opt-in. When omitted, `/shards/execute` remains disabled and returns `503`. When provided, the CLI loads the `go-pherence` model, builds a `PherenceLayerExecutor`, wraps it with `sim.LayerExecutorWorker`, and installs it with `server.WithShardExecution`. This is still a local shard worker surface, not LAN distributed generation.


## Capability-based route planning

`router.BuildRoutesFromCapabilities` turns live `protocol.Capability` advertisements into validated `router.Route` values by reconstructing cluster peers from capability metadata (`address`, `transport`) and applying the existing route validation. This is planning-only and does not execute remote generation.


## Route preview API

`GET /routes/preview?layers=N&model=...` builds a placement plan from advertised server capabilities, converts those capabilities into validated routes, and returns peer/address/transport/shard entries. This is a planning endpoint only; it does not call `/shards/execute` or enable distributed generation.


## Registry-backed route previews

`router.PreviewFromRegistry` builds a `protocol.RoutePreview` from the current in-memory `cluster.Registry` snapshot. It converts registered peers to capability DTOs, builds a placement plan, and returns route entries. The function is metadata-only: it does not contact peers and does not call the shard execution endpoint.


## Registry-backed HTTP route previews

Servers can be constructed with `server.WithRegistry(registry)` to have `/routes/preview` use the live in-memory peer registry snapshot instead of the static capability list. If no registry is configured, the endpoint preserves the static capability behavior. Registry-backed previews are still metadata-only and do not execute shards.


## CLI route preview

`go-exotic routes -layers N [-json] [-model ID]` exposes the same capability-based route planning as `/routes/preview` for local capabilities. The command emits route metadata only and does not invoke shard execution.


## Remote worker maps from routes

`sim.RemoteWorkersFromRoutes` turns validated `router.Route` values into a simulator worker map backed by `cluster.RemoteShardWorker`. This is an explicit orchestration helper for tests and future opt-in flows; the default generation paths do not call it.


## Documentation audit status

This document was refreshed after the route-preview, registry-backed planning, remote-worker-map, and HTTP parsing audit work. The current boundary is: planning and explicit test/orchestration helpers are implemented; default CLI/LAN distributed generation remains disabled.


## Web UI

The server serves a small dashboard at `/` plus bundled static assets under `/static/`. The UI source is TypeScript/TSX in `web/src`, typechecked with `tsc`, and built with Bun into `web/static/app.js` and `web/static/app.css`. Preact and D3 are bundled into the generated asset from Bun dependencies, and the dashboard currently shows peers, placement previews, route previews, execution-boundary status, error states, and copyable model setup helper commands. API fetch/types live in `web/src/api.ts`; lightweight localStorage helpers live in `web/src/storage.ts` for persisting planner controls.


## Model helper API

`GET /models/helpers?model=...&path=...` returns manual model fixture setup commands, model path presets, required file names, and read-only file presence status for the dashboard. It is intentionally non-mutating: it does not download models or write files.
