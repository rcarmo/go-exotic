# Real shard execution closeout

Status: complete for the current Phase 7–9 plan.

## Completed gates

- Local `go-pherence` layer-range execution via `runtime.PherenceLayerExecutor` over `model.ForwardLayer`.
- Simulator integration through `sim.LayerExecutorWorker`.
- Single-host multi-shard execution using real local layer executor workers.
- Fixture-backed hidden-state and one-token output parity against direct sequential `go-pherence` layer execution.
- Documented fixture skip path via `GO_EXOTIC_SMOLLM2_MODEL` or `../go-pherence/models/smollm2-135m`.
- Gated HTTP shard endpoint, disabled by default in the CLI server.
- HTTP activation serialization using `protocol.ActivationPayload` flat f32 payloads.
- Remote worker client adapter with request ID propagation, context cancellation, and timeout support.
- Route preview helpers for static capabilities, live registries, and CLI output.
- Explicit remote-worker-map helper for tests/orchestration from planned routes.
- `httptest` coverage for the remote worker path before any LAN enablement.

## Guardrails added during closeout

- Layer executor validates caller layer count against loaded model layer count.
- Activation payloads and shard messages enforce `protocol.MaxActivationElements`.
- Simulator validates response peer identity.
- Synthetic generation checks nil receivers and decode-position overflow.
- Simulator rejects typed-nil workers.
- Server rejects request ID header/body mismatches.
- Remote worker rejects response request ID mismatches.
- Server rejects trailing JSON values in shard execution requests.
- CLI-advertised route addresses normalize hostless/wildcard listen addresses to localhost.

## Current execution boundary

The default CLI server still does not enable remote shard execution. `/shards/execute` returns `503 shard execution disabled` unless a caller explicitly constructs a server with `server.WithShardExecution` or starts `go-exotic serve` with the local-development `-shard-model` opt-in.

This keeps LAN distributed generation disabled while preserving tested transport scaffolding for future work.

## Follow-up work

- Local CLI opt-in wiring is available through `go-exotic serve -shard-model /path/to/model`.
- LAN peer selection from live/discovered capability registries. Route planning from explicit registries is available, but not wired to LAN discovery.
- End-to-end networked generation smoke tests across real hosts.
- Richer activation/tensor payload formats beyond flat f32.
- OpenAI-compatible generation endpoint once distributed generation is productized.
