# exo reference notes

Reference checkout: `/workspace/references/exo`

Important upstream areas:

- `src/exo/master/` — request routing, placement, download coordination
- `src/exo/worker/` — worker lifecycle and inference runners
- `src/exo/worker/runner/llm_inference/` — batch generation and model execution paths
- `src/exo/worker/engines/` — backend-specific execution, especially MLX/tinygrad
- `src/exo/download/` — shard/model download coordination
- `src/exo/routing/` and `src/exo/shared/types/` — peer/routing message shapes

Current porting direction:

1. Keep `go-exotic` orchestration/runtime-neutral.
2. Use `go-pherence` for local model loading, tokenization, and full-model generation.
3. Start with deterministic layer placement, then add capability exchange and HTTP placement previews.
4. Validate route construction and cancellation before any remote shard execution.
5. Keep shard boundaries explicit and validated before distributed inference.

Do not vendor upstream exo into `go-exotic`; treat it as a read-only reference checkout.


## Current go-exotic divergence update

Unlike upstream exo's backend runners, the current Go port has a narrow in-process layer execution hook over `go-pherence/model.ForwardLayer`. This deliberately avoids remote execution and download orchestration until local real-shard simulation can be compared against full-model `go-pherence` output.
