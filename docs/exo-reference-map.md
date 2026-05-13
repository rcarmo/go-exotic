# exo reference map

Reference checkout: `/workspace/references/exo`.

This note maps upstream exo concepts to the first Go-facing design for `go-exotic`. It is intentionally descriptive; upstream Python/Rust bindings are a reference, not vendored code.

## Upstream architecture summary

### Master request routing and placement

Relevant files:

- `src/exo/master/main.py`
- `src/exo/master/placement.py`
- `src/exo/master/placement_utils.py`
- `src/exo/routing/` and `src/exo/routing/*.py`

The upstream master owns user-facing request orchestration. It accepts API requests, creates sessions, determines which workers/runners should host model shards, coordinates shard/model downloads, and routes prompt/generation work to workers. Placement is modeled around runner/device capability and shard metadata, with helper code that can add/remove placements and cancel unnecessary downloads when placement changes.

Go direction:

- `internal/placement` will own deterministic and capability-aware layer placement.
- `internal/protocol` will carry placement previews and eventual execution routes.
- `internal/exotic` currently holds the bootstrap planner until the placement package is split.

### Worker lifecycle

Relevant files:

- `src/exo/worker/main.py`
- `src/exo/worker/runner/`
- `src/exo/worker/disaggregated/`

Workers expose available local execution resources, receive shard assignments, ensure required model shards are available, and run runner tasks. Upstream separates worker supervision from runner implementations so multiple engines/backends can be used under the same orchestration layer.

Go direction:

- `internal/cluster` will track node/peer membership and advertised capabilities.
- `internal/runtime` will expose `go-pherence` local runtime capabilities and execution.
- Worker lifecycle starts in-process before networked workers are enabled.

### Runner and batch generation

Relevant files:

- `src/exo/worker/runner/llm_inference/`
- `src/exo/worker/engines/`

Upstream runner code handles model execution and generation batching, delegating backend-specific tensor execution to engines such as MLX/tinygrad variants. The important portable idea is that orchestration should not depend on backend internals.

Go direction:

- Runtime calls stay behind an `internal/runtime` interface.
- The first implementation will call `go-pherence` for local full-model metadata/generation.
- Layer-shard execution is deferred until the runtime boundary can represent activations/KV state safely.

### Download and shard coordination

Relevant files:

- `src/exo/download/coordinator.py`
- `src/exo/download/shard_download.py`
- `src/exo/download/impl_shard_downloader.py`

Upstream coordinates downloads around model/shard metadata so workers obtain only what is needed for their assignments. It also cancels unnecessary downloads as placements change.

Go direction:

- Initial `go-exotic` will not implement download orchestration.
- Local paths and `go-pherence` fixtures are used first.
- A later model-store package can map placements to local files and download tasks.

### Networking and discovery surfaces

Relevant files:

- `src/exo/main.py`
- `src/exo/routing/`
- `src/exo/shared/types/`
- Rust/Python bindings imported as `exo_pyo3_bindings`

Upstream uses routing/connection messages and node identifiers to build a peer graph and route work. Some low-level networking/discovery behavior is delegated to packaged bindings rather than pure Python source.

Go direction:

- Start with HTTP/gRPC-style LAN transport rather than reproducing every exo transport detail.
- Keep peer identity, capability exchange, heartbeat, and routing DTOs explicit in `internal/protocol`.
- Simulate multi-worker execution in-process before enabling network transport.

## Go-facing glossary

| exo concept | Go term | Initial owner | Notes |
| --- | --- | --- | --- |
| node | peer/node | `internal/cluster` | Stable ID plus transport address and advertised capabilities. |
| device capability | capability | `internal/runtime`, `internal/cluster` | Backend, memory budget, layer support, runtime flags. |
| shard | layer range | `internal/exotic`, later `internal/placement` | Contiguous `[start,end]` transformer layer range assigned to a peer. |
| route / placement | placement plan | `internal/placement` | Maps model layers to peers and records why. |
| request/session | session | `internal/protocol` | Tracks prompt/generation lifecycle and cancellation. |
| token stream | token stream | `internal/protocol` | Ordered generated tokens and terminal/error state. |
| runner | runtime worker | `internal/runtime` | Local adapter over `go-pherence`, later shard executor. |
| download coordinator | model store | planned | Deferred until runtime/local path behavior is stable. |

## Divergences from upstream exo

- `go-exotic` uses `go-pherence` as the only initial runtime; upstream supports multiple Python/native engines.
- Initial placement is deterministic and local-data driven, not dynamic peer graph optimization.
- Network transport will start with simple LAN HTTP/gRPC semantics before reproducing any exotic discovery/routing behavior.
- Model download/shard cache coordination is deferred; local paths and fixtures are used first.
- Distributed execution stays disabled until single-host simulation can match full-model `go-pherence` output.
