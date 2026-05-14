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
| Routing | `internal/router` | Validated route construction from shard plans and peer snapshots; cancellation propagation |
| Runtime adapter | `internal/runtime` | Minimal `go-pherence` adapter for metadata, tokenization, local full-model generation, and future shard execution contract |
| Protocol/API | `internal/protocol`, `internal/server` | DTOs and HTTP skeleton for health, capabilities, and placement previews |
| CLI | `cmd/go-exotic` | `plan`, `run`, `peers`, and `serve`; distributed generation remains disabled |
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

`serve` is quiet by default and emits structured JSON diagnostics only with `-verbose`.

## Current cluster validation

The first integration test runs multiple in-process peer capabilities through the HTTP placement preview endpoint and validates route construction. Remote shard execution remains disabled.

## Runtime boundary

The current `internal/runtime.Adapter` supports:

- model metadata
- tokenization
- local full-model generation

The future `LayerShardExecutor` shape uses `protocol.ShardExecutionRequest` and `protocol.ShardExecutionResponse`, which carry session/request IDs, explicit layer ranges, positions, hidden size, and activation vectors. `protocol.ActivationPayload` defines the first transport serialization format: flat float32 little-endian bytes with hidden-size metadata. The executor remains intentionally not implemented until KV/state ownership and multi-worker generation are tested.

## Validation baseline

- `make check` must pass after every code batch.
- Runtime behavior changes need focused smoke commands.
- Follow-up commits include `Signed-off-by: Rui Carmo <rui.carmo@gmail.com>`.

## Single-host simulation

`internal/sim` executes routed shard requests sequentially in one process. It is the first distributed-inference harness and deliberately uses `protocol.ShardExecutionRequest`/`Response` without any remote transport. Real layer execution and tensor serialization are still pending.

## Synthetic token generation harness

`Simulator.GenerateTokens` runs a synthetic decode loop through routed in-process shard workers and maps the final activation to a token via a caller-provided projector. This validates orchestration, cancellation, and activation handoff through N shards before real `go-pherence` layer execution or numerical parity checks are attempted.
