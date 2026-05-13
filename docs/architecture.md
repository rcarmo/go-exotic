# go-exotic architecture

`go-exotic` ports the core ideas of upstream `exo` to Go while using `go-pherence` as the local inference runtime.

## Reference boundaries

- Upstream exo checkout: `/workspace/references/exo`
- Go runtime dependency: `/workspace/projects/go-pherence`
- This module must not vendor upstream exo; it is a reference for behavior and architecture.

## Initial package ownership

| Area | Package | Responsibility |
| --- | --- | --- |
| Domain model | `internal/exotic` | Devices, layer shards, deterministic placement baseline |
| CLI | `cmd/go-exotic` | Human-facing entrypoints and smoke commands |
| Runtime adapter | `internal/runtime` _(planned)_ | `go-pherence` model loading/generation adapter |
| Cluster membership | `internal/cluster` _(planned)_ | Peer identity, discovery, heartbeat, stale-peer eviction |
| Protocol | `internal/protocol` _(planned)_ | Wire DTOs for capabilities, shard execution, token streaming |
| Placement | `internal/placement` _(planned)_ | exo-style placement policies beyond the baseline planner |

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
