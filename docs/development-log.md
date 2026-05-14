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

## Session 13: Cluster heartbeat registry

- Added in-memory peer registry for the initial HTTP/LAN cluster transport.
- Implemented peer upsert, heartbeat refresh, sorted peer snapshots, and stale-peer eviction.
- Added tests for validation, deterministic snapshots, copy semantics, heartbeat refresh, and stale eviction.

## Session 14: Request routing skeleton

- Added `internal/router` with route construction from a validated placement plan and current peer snapshot.
- Routing validates peers, rejects duplicate/missing peer IDs, keeps output ordered by layer range, and checks `context.Context` for cancellation before and during route construction.
- Added tests for deterministic route output, malformed route inputs, and cancellation propagation.

## Session 15: In-process multi-node integration test

- Added an in-process multi-node integration test using `httptest` and two local peer capabilities.
- The test exercises `/placement/preview`, validates the returned shard plan, and builds routes against the peer snapshot.
- This closes the first cluster transport phase without enabling remote shard execution.

## Session 16: Router audit

- Audited the route-building path after adding in-process integration coverage.
- Added explicit `no peers` and `peers fewer than shards` validation before per-shard routing.
- Extended malformed route tests to cover no-peer, fewer-peer, and missing-peer cases distinctly.

## Session 17: Documentation sweep

- Refreshed README, architecture, and exo reference docs after completing placement hardening, runtime adapter, CLI/API skeleton, cluster registry, routing, and in-process integration tests.
- Documented current implemented/deferred surfaces and quiet/verbose serve behavior.
- Reaffirmed that remote shard execution and OpenAI-compatible generation remain deferred.

## Session 18: Shard execution protocol

- Added protocol DTOs for `ShardExecutionRequest` and `ShardExecutionResponse` around explicit layer ranges, positions, hidden sizes, and activation vectors.
- Added validation and JSON round-trip tests for malformed shard execution requests/responses.
- Updated the runtime `LayerShardExecutor` hook to use the protocol DTOs while remote execution remains disabled.

## Session 19: Single-host simulation harness

- Added `internal/sim`, a sequential in-process multi-worker harness over routed shard execution requests.
- The simulator validates request/response metadata, propagates cancellation, copies activation outputs between routes, and keeps execution local-only.
- Added tests for successful two-shard execution, malformed simulator inputs, worker errors, and cancellation propagation.

## Session 20: Activation serialization format

- Added `protocol.ActivationPayload`, a flat float32 little-endian activation serialization format for the first shard execution protocol.
- Added round-trip JSON/base64 coverage and malformed payload validation for encoding, hidden size, and byte length.
- Kept the format deliberately simple pending batched/paged tensor execution.

## Session 21: Synthetic N-shard token generation harness

- Added `Simulator.GenerateTokens`, a synthetic decode loop that runs each step through routed shard execution and projects the final activation to a token.
- This exercises end-to-end token flow through N simulated shards without claiming numerical parity with `go-pherence` yet.
- Added tests for multi-step two-shard generation, malformed inputs, cancellation propagation, and projector errors.

## Session 22: Local go-pherence generation parity check

- Added a resource-safe integration test comparing `go-exotic`'s local runtime adapter output against direct full-model `go-pherence` generation on the SmolLM2 fixture when present.
- This closes the first parity gate for the local adapter. Distributed numerical parity remains a future gate once real shard-layer execution replaces the synthetic simulator.

## Session 23: Initial development plan closeout

- Re-ran `make check` after completing the initial go-exotic plan from project setup through Phase 6 simulation/parity scaffolding.
- Current implementation covers reference mapping, package boundaries, hardened placement, local go-pherence adapter, CLI/API skeleton, HTTP cluster surfaces, routing, in-process simulation, activation serialization, synthetic N-shard token flow, and local adapter parity against direct go-pherence generation.
- Remaining work moves into a follow-up implementation plan: replacing synthetic shard workers with real layer-shard execution, then validating distributed numerical parity.

## Session 24: Local go-pherence layer executor contract

- Inspected `go-pherence/model.ForwardLayer` as the smallest safe local hook for layer-range execution over float activations and caller-owned KV caches.
- Added `runtime.LayerRangeExecutor` and a local `PherenceLayerExecutor` wrapper that executes a validated shard range in-process with cancellation checks before each layer.
- Added validation tests for nil/invalid models, context cancellation, malformed hidden size, KV cache shape, and invalid zero-layer ranges.

## Session 25: Layer executor audit

- Audited the new local `PherenceLayerExecutor` for mismatches between request validation and loaded model state.
- Added an explicit check that shard request `totalLayers` matches the loaded model's layer count before validating/executing a range.
- Split regression coverage so layer-count mismatch and KV-cache shape validation are both exercised distinctly.

## Session 26: Simulator response audit

- Audited simulator shard response validation for route/peer mismatch edges.
- Added an explicit check that each shard response `PeerID` matches the route peer that executed the shard.
- Added regression coverage rejecting a worker response that claims a different peer ID.

## Session 27: Synthetic generation loop audit

- Audited `Simulator.GenerateTokens` for nil receiver and decode-position overflow edges.
- Added explicit nil simulator rejection before generation setup.
- Added checked position range validation so `base.Position + step` cannot overflow during multi-token synthetic generation.
- Added regression coverage for nil simulator and overflowing generation positions.

## Session 28: Activation payload bounds audit

- Audited protocol activation payloads and shard messages for unbounded hidden-size/payload acceptance.
- Added `MaxActivationElements` as a generous bound for the first flat f32 activation transport format.
- Applied the bound to activation payload construction/decoding and shard request/response validation.

## Session 29: Layer executor simulator worker

- Integrated the local layer-range execution contract into `internal/sim` via `LayerExecutorWorker`.
- Added focused tests for single-layer execution validation, multi-layer range validation, malformed activation rejection, nil worker/executor rejection, and cancellation propagation through the worker adapter.
- The tests use a fake layer executor; real `go-pherence` layer numerical parity remains a Phase 8 gate.

## Session 30: Simulator worker nil audit

- Audited simulator worker registration after adding `LayerExecutorWorker`.
- Fixed typed-nil worker handling: `WorkerFunc(nil)` and nil pointer workers inside the `Worker` interface are now rejected by `NewSimulator` instead of panicking later during execution.
- Added regression coverage for typed nil worker functions and nil pointer workers.

## Session 31: Documentation refresh after Phase 7 audits

- Refreshed project documentation after adding the local `go-pherence` layer-range executor, simulator worker adapter, bounded activation payloads, and simulator audit fixes.
- Documented current Phase 7 status: real local layer execution is available in-process through `PherenceLayerExecutor` and `LayerExecutorWorker`, while remote HTTP shard execution remains disabled until parity gates pass.
- Noted protocol and simulator guardrails: bounded flat f32 activation payloads, response peer identity validation, generation position overflow checks, and typed-nil worker rejection.

## Session 32: Single-host real layer executor simulation

- Added an in-process integration test that routes a synthetic two-layer `go-pherence` model across two simulated shard workers backed by `PherenceLayerExecutor`.
- The test validates that real local `ForwardLayer` execution can be driven through the `router` + `sim.LayerExecutorWorker` path without remote transport.
- This is a structural real-shard execution gate; numerical parity against full-model generation remains the next Phase 8 item.
