# go-exotic

`go-exotic` is a Go port/experiment inspired by [`exo`](https://github.com/exo-explore/exo), using [`go-pherence`](../go-pherence) as the local inference runtime.

The upstream reference has been cloned separately at:

- `/workspace/references/exo`

## Goals

- Recreate exo-style local distributed inference in Go.
- Use `go-pherence` for local model loading and execution.
- Start with deterministic, testable shard placement before adding peer discovery or networked execution.
- Keep distributed generation disabled until local and simulated multi-node paths match full-model `go-pherence` output.

## Current status

Implemented:

- standalone Go module
- local `go-pherence` dependency via `replace ../go-pherence`
- deterministic memory-weighted layer-shard planner
- minimal CLI for placement preview
- initial docs/tests/Makefile following `go-pherence` module patterns
- local-only `go-pherence` runtime adapter interface for metadata/tokenization/generation
- local runtime smoke command: `go-exotic run`

Planned next:

- exo reference map and glossary
- hardened placement package
- `go-pherence` runtime adapter
- local generation smoke command
- in-process multi-worker simulation

## Quick start

```bash
make check
make run LAYERS=4
```

Equivalent direct command:

```bash
go run ./cmd/go-exotic plan -layers 4
go run ./cmd/go-exotic plan -layers 4 -json
go run ./cmd/go-exotic run -model ../go-pherence/models/smollm2-135m -prompt "Hello" -tokens 1
```

## Architecture

See [`docs/architecture.md`](docs/architecture.md).

## exo reference

See [`docs/reference-exo.md`](docs/reference-exo.md) and [`docs/exo-reference-map.md`](docs/exo-reference-map.md). The reference checkout is deliberately kept outside this module.

## Commit policy

Use Rui Carmo's Git identity and include:

```text
Signed-off-by: Rui Carmo <rui.carmo@gmail.com>
```
