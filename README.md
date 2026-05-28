# go-exotic

`go-exotic` is a Go experiment inspired by [`exo`](https://github.com/exo-explore/exo). It explores deterministic layer placement, shard routing, and local inference execution using [`go-pherence`](https://github.com/rcarmo/go-pherence) as the runtime.

## Features

- Deterministic memory-weighted layer-shard planning.
- Capability and route preview APIs for planning distributed inference layouts.
- Local `go-pherence` runtime adapter for model metadata, tokenization, generation smoke tests, and layer-range execution.
- In-process simulator for validating shard orchestration before networked execution.
- Gated HTTP shard execution endpoint, disabled by default.
- Bun/TypeScript dashboard with bundled Preact and D3.
- Read-only model helper APIs for local fixture discovery and setup commands.

## Quick start

```bash
make check

go run ./cmd/go-exotic plan -layers 4
go run ./cmd/go-exotic plan -layers 4 -json
go run ./cmd/go-exotic peers -json
go run ./cmd/go-exotic routes -layers 4 -json
go run ./cmd/go-exotic run -model ../go-pherence/models/smollm2-135m -prompt "Hello" -tokens 1
go run ./cmd/go-exotic serve -addr 127.0.0.1:8089
```

Open the dashboard at:

```text
http://127.0.0.1:8089/
```

`serve` is quiet by default. Use `-verbose` for structured JSON diagnostics.

## CLI

```bash
go run ./cmd/go-exotic plan -layers 4 [-json]
go run ./cmd/go-exotic peers [-json]
go run ./cmd/go-exotic routes -layers 4 [-model demo] [-json]
go run ./cmd/go-exotic run -model /path/to/model -prompt "Hello" -tokens 1
go run ./cmd/go-exotic serve -addr 127.0.0.1:8089
```

`routes` is planning-only and does not execute shards.

## HTTP API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | Dashboard UI |
| `GET` | `/static/*` | Dashboard static assets |
| `GET` | `/health` | Health check |
| `GET` | `/ui/status` | Server, API, web bundle, and execution-boundary status |
| `GET` | `/capabilities` | Advertised peer capabilities |
| `GET` | `/placement/preview?layers=N&model=...` | Deterministic shard placement preview |
| `GET` | `/routes/preview?layers=N&model=...` | Planning-only route preview |
| `GET` | `/models/local?root=...&limit=N` | Bounded read-only local model inventory |
| `GET` | `/models/helpers?model=...&path=...` | Labeled helper commands and required-file status |
| `POST` | `/shards/execute` | Shard execution endpoint, gated and disabled by default |

Example API calls:

```bash
curl http://127.0.0.1:8089/health
curl http://127.0.0.1:8089/capabilities
curl 'http://127.0.0.1:8089/placement/preview?layers=4&model=demo'
curl 'http://127.0.0.1:8089/routes/preview?layers=4&model=demo'
```

## Execution boundary

Remote/LAN shard execution is not enabled by default. The default server returns `503 shard execution disabled` for `/shards/execute`.

For local development, a model-backed shard worker can be installed explicitly:

```bash
go run ./cmd/go-exotic serve -addr 127.0.0.1:8089 -shard-model ../go-pherence/models/smollm2-135m
```

This wires the shard endpoint to a local `go-pherence` model through `runtime.PherenceLayerExecutor`. It does not enable peer discovery or end-to-end networked distributed generation.

## Dashboard development

The dashboard source lives under `web/src` and is built with Bun:

```bash
bun install
bun run typecheck:web
bun run build:web
```

The generated `web/static/app.js` bundle includes the dashboard dependencies used by the Go server.

## Validation

```bash
make check
```

`make check` formats Go code, typechecks and rebuilds the dashboard, runs Go tests, runs `go vet`, and checks whitespace with `git diff --check`.

## Documentation

- [Architecture](docs/architecture.md)
- [exo reference notes](docs/reference-exo.md)
- [exo reference map](docs/exo-reference-map.md)
- [Real shard execution closeout](docs/real-shard-execution-closeout.md)
