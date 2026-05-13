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
| CLI | `cmd/go-exotic` | Human-facing entrypoints and smoke commands |
| Runtime adapter | `internal/runtime` | Minimal interface for future `go-pherence` model loading/generation adapter |
| Cluster membership | `internal/cluster` | Peer identity skeleton, future discovery, heartbeat, stale-peer eviction |
| Protocol | `internal/protocol` | Initial DTOs for capabilities and placement previews; future shard execution/token streaming |
| Placement | `internal/placement` | Deterministic baseline planner, contiguous coverage validation, future exo-style policies |

## Porting model

1. Start local-only and deterministic.
2. Keep shard ranges explicit and validated.
3. Add `go-pherence` runtime calls behind a small interface.
4. Simulate multiple workers in-process before adding LAN transport.
5. Compare distributed output against local full-model `go-pherence` output before enabling networked generation.

## Validation baseline

- `make check` must pass after every code batch.
- Runtime behavior changes need focused smoke commands.
- Follow-up commits include `Signed-off-by: Rui Carmo <rui.carmo@gmail.com>`.
