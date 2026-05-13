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

## Session 8: Runtime adapter audit

- Audited the new local `go-pherence` adapter for malformed path and metadata edges.
- Added model path trimming/rejection for whitespace-only paths across metadata, tokenization, generation, model loading, and tokenizer loading.
- Added metadata validation for model path, layer count, hidden size, and vocab size before exposing runtime metadata to placement/CLI callers.

## Session 9: CLI and HTTP API skeleton

- Added `peers` CLI for local capability inspection, with text and JSON output.
- Added `serve` CLI with an HTTP skeleton: `/health`, `/capabilities`, and `/placement/preview`.
- Kept distributed generation disabled; `serve` only exposes local capability and placement preview surfaces.
- Added HTTP handler tests and smoke-tested health/placement endpoints with a local server.

## Session 10: Quiet structured serve logging

- Audited CLI/server output and made `serve` quiet by default.
- Added `serve -verbose` for structured JSON diagnostics via `log/slog`.
- Verified default serve mode emits no stderr startup diagnostics while `/health` remains available.


## Session 11: Initial cluster transport decision

- Chose HTTP over LAN as the initial cluster transport before reproducing upstream exo's richer discovery/routing surfaces.
- Added peer identity/capability skeleton in `internal/cluster` with validation for peer ID, HTTP address, transport, and device capability.
- Added tests for local peer capability exchange and malformed peer rejection.

## Session 12: Cluster/server audit

- Tightened peer validation to require HTTP addresses, reject NaN/Inf memory, and normalize local peer addresses.
- Added peer validation tests for HTTPS/unsupported transport and malformed memory values.
- Fixed server capability handling to deep-copy metadata maps on input/output so callers cannot mutate server state through aliases.
