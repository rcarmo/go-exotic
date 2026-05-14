# go-exotic

`go-exotic` is a Go port/experiment inspired by [`exo`](https://github.com/exo-explore/exo), using [`go-pherence`](../go-pherence) as the local inference runtime.

The upstream reference is cloned separately at:

- `/workspace/references/exo`

## Goals

- Recreate exo-style local distributed inference in Go.
- Use `go-pherence` for local model metadata, tokenization, and full-model generation.
- Keep shard placement deterministic and heavily validated before enabling remote execution.
- Start with HTTP/LAN capability and placement surfaces, then simulate multi-node execution before true distributed inference.
- Keep CLI/LAN distributed generation disabled until local real-shard parity and HTTP transport guardrails are validated.

## Current status

Implemented:

- standalone Go module with local `go-pherence` dependency via `replace ../go-pherence`
- deterministic memory-weighted layer-shard planner
- placement plan validation and JSON export
- CLI commands:
  - `plan` — placement preview
  - `run` — local `go-pherence` generation smoke
  - `peers` — local capability inspection
  - `serve` — local HTTP API skeleton
- HTTP endpoints:
  - `GET /health`
  - `GET /capabilities`
  - `GET /placement/preview?layers=N&model=...`
  - `POST /shards/execute` — gated behind explicit server wiring; disabled by the default CLI server
- core packages for domain, placement, cluster, routing, runtime, protocol, and server surfaces
- in-memory peer registry with heartbeat and stale-peer eviction
- route construction with cancellation propagation
- in-process multi-node placement/routing integration test
- local in-process `go-pherence` layer-range executor over `ForwardLayer`
- simulator worker adapter for real local layer executors
- bounded flat f32 activation payload format and shard execution DTOs
- single-host real-shard hidden-state and one-token output parity gates
- disabled-by-default HTTP shard execution bridge with request IDs, context cancellation, timeouts, and httptest coverage

Deferred:

- enabling remote shard execution in the CLI/LAN path
- OpenAI-compatible generation endpoint
- model download/cache coordination
- richer tensor activation serialization beyond the initial flat f32 payload
- networked multi-worker generation

## Quick start

```bash
make check

go run ./cmd/go-exotic plan -layers 4
go run ./cmd/go-exotic plan -layers 4 -json
go run ./cmd/go-exotic peers -json
go run ./cmd/go-exotic run -model ../go-pherence/models/smollm2-135m -prompt "Hello" -tokens 1
go run ./cmd/go-exotic serve -addr 127.0.0.1:8089
```

Example API calls while `serve` is running:

```bash
curl http://127.0.0.1:8089/health
curl http://127.0.0.1:8089/capabilities
curl 'http://127.0.0.1:8089/placement/preview?layers=4&model=demo'
```

`serve` is quiet by default. Use `-verbose` for structured JSON diagnostics. `/shards/execute` remains disabled unless `serve` is started with the explicit local opt-in `-shard-model /path/to/go-pherence-model`.

## Architecture

See [`docs/architecture.md`](docs/architecture.md). The Phase 7–9 closeout is summarized in [`docs/real-shard-execution-closeout.md`](docs/real-shard-execution-closeout.md).

## exo reference

See [`docs/reference-exo.md`](docs/reference-exo.md) and [`docs/exo-reference-map.md`](docs/exo-reference-map.md). The reference checkout is deliberately kept outside this module and must not be vendored.

## Validation

Every code batch should run:

```bash
make check
```

Runtime behavior changes should also run the relevant smoke command (`plan`, `run`, `peers`, or `serve`/`curl`).

## Commit policy

Use Rui Carmo's Git identity and include:

```text
Signed-off-by: Rui Carmo <rui.carmo@gmail.com>
```

## Current execution boundary

`go-exotic` can now execute validated layer ranges in-process through `runtime.PherenceLayerExecutor`, which wraps `go-pherence/model.ForwardLayer`. `internal/sim.LayerExecutorWorker` adapts that executor to the simulator worker interface. Local real-shard hidden-state and one-token output parity are covered by fixture-backed tests. The HTTP shard endpoint and client adapter exist for tests, but the default CLI server still leaves remote execution disabled.

### Explicit local shard execution opt-in

The default server keeps remote shard execution disabled. For local development only, a model-backed shard worker can be installed explicitly:

```bash
go run ./cmd/go-exotic serve -addr 127.0.0.1:8089 -shard-model ../go-pherence/models/smollm2-135m
```

This wires `server.WithShardExecution` to a local `runtime.PherenceLayerExecutor` and `sim.LayerExecutorWorker`. It does not perform peer discovery, route construction, or end-to-end distributed generation.
