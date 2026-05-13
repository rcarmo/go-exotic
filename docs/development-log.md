# go-exotic development log

## Session 1: Initial scaffold

- Cloned upstream exo as a separate read-only reference at `/workspace/references/exo`.
- Created standalone Go module `github.com/rcarmo/go-exotic` with a local `replace` to `../go-pherence`.
- Added deterministic baseline layer shard planning plus a minimal CLI.

## Session 2: Module pattern alignment

- Added a `go-pherence`-style Makefile with workspace temp dirs and `fmt`, `test`, `vet`, `check`, and `run` targets.
- Added `.gitignore`, architecture notes, and expanded README structure.
- Established signed-off commit policy for follow-up commits.

## Session 3: exo reference map

- Read upstream exo master, worker, runner, download, routing, and shared type surfaces under `/workspace/references/exo/src/exo`.
- Added `docs/exo-reference-map.md` summarizing upstream architecture, Go-facing glossary, and deliberate divergences for `go-pherence` integration.

## Session 4: Core package skeletons and placement split

- Added initial Phase 2 package skeletons: `internal/cluster`, `internal/placement`, `internal/runtime`, and `internal/protocol`.
- Kept `internal/exotic` focused on domain types (`Device`, `Shard`) and validation.
- Moved layer-shard planning into `internal/placement`, added duplicate device rejection and exact contiguous coverage validation.
- Added package tests for placement malformed inputs and domain shard validation.

## Session 5: Placement planner hardening

- Hardened the memory-weighted shard planner for edge cases: too many devices for layers, NaN/Inf/negative memory, duplicate device IDs, and total memory overflow.
- Added exact plan validation into planner output and table-driven malformed input tests.
- Added serializable `placement.Plan` plus stable JSON export and `go-exotic -json` CLI output.

## Session 6: Minimal go-pherence runtime adapter

- Expanded `internal/runtime` with metadata, tokenization, generation, and future shard execution contracts.
- Added `PherenceAdapter`, a local-only adapter over `go-pherence/model.LoadLlama`, tokenizer encode, and full-model `Generate`.
- Added validation tests for empty paths, negative token budgets, cancelled contexts, and interface conformance.

## Session 7: Local runtime smoke command

- Added CLI subcommands: `plan`, `run`, plus explicit disabled placeholders for `serve` and `peers`.
- Added `go-exotic run` as a local-only `go-pherence` runtime smoke command.
- Fixed the runtime adapter to load `tokenizer.json` explicitly instead of assuming `LoadLlama` populates `Tok`.
- Smoke passed with `../go-pherence/models/smollm2-135m` and a one-token budget.
