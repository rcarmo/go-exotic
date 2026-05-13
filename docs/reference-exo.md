# exo reference notes

Reference checkout: `/workspace/references/exo`

Important upstream areas to study:

- `src/exo/master/` — request routing, placement, download coordination
- `src/exo/worker/` — worker lifecycle and inference runners
- `src/exo/worker/runner/llm_inference/` — batch generation and model execution paths
- `src/exo/worker/engines/` — backend-specific execution, especially MLX/tinygrad
- `src/exo/download/` — shard/model download coordination

Porting direction:

1. Keep `go-exotic` orchestration/runtime-neutral.
2. Use `go-pherence` for local model loading and layer execution.
3. Start with deterministic layer placement, then add discovery/network transport.
4. Keep shard boundaries explicit and validated before any remote execution.
