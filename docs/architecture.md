# go-exotic architecture

`go-exotic` ports the core ideas of upstream `exo` to Go while using `go-pherence` as the local inference runtime.

## Reference boundaries

- Upstream exo checkout: `/workspace/references/exo`
- Go runtime dependency: `/workspace/projects/go-pherence`
- This module must not vendor upstream exo; it is a reference for behavior and architecture.

## Initial package ownership

| Area | Package | Responsibility |
| --- | --- | --- |
| Domain model | `internal/exotic` | Devices, layer shards, domain validation |
| CLI | `cmd/go-exotic` | `plan`, `run`, `peers`, and `serve` entrypoints; distributed generation remains disabled |
| Runtime adapter | `internal/runtime` | Minimal `go-pherence` adapter for metadata, tokenization, local full-model generation, and future shard execution contract |
| Cluster membership | `internal/cluster` | Peer identity, local capability exchange, heartbeat refresh, in-memory registry, stale-peer eviction |
| Protocol/API | `internal/protocol`, `internal/server` | Initial DTOs and HTTP skeleton for health, capabilities, and placement previews; future shard execution/token streaming |
| Placement | `internal/placement` | Deterministic baseline planner, contiguous coverage validation, future exo-style policies |

## Porting model

1. Start local-only and deterministic.
2. Keep shard ranges explicit and validated.
3. Add `go-pherence` runtime calls behind a small interface.
4. Simulate multiple workers in-process before adding LAN transport.
5. Compare distributed output against local full-model `go-pherence` output before enabling networked generation.

## Transport decision

The initial cluster transport is HTTP over LAN. This is deliberately simpler than upstream exo routing/discovery internals and gives `go-exotic` a stable contract for health, capabilities, placement previews, and future shard execution before exploring richer transports. UDP/mDNS/gRPC/exotic transports are deferred until the in-process and HTTP paths are validated.

## Validation baseline

- `make check` must pass after every code batch.
- Runtime behavior changes need focused smoke commands.
- Follow-up commits include `Signed-off-by: Rui Carmo <rui.carmo@gmail.com>`.
