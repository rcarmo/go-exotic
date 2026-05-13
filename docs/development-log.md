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
