# AGENTS.md

Guidance for AI coding agents working in this repository.

## What this project is

strata is a requirements-traceability toolkit: `spec-blame` verifies that
code implements what specs claim (coverage, drift, blame), `spec-excavate`
bootstraps specs for repos that never had one. Both share `pkg/reqgraph`,
the canonical requirement IR.

## Non-negotiable invariants

- **strata never defines its own spec format.** Adapters in `pkg/ingest`
  read the formats repos already use (openspec, spec-kit, Gherkin,
  markdown-with-IDs). A new format is a new adapter file, not a schema
  change to the IR.
- **The gate only fails on objectively broken things**: dangling refs,
  stale hashes, unknown IDs. Coverage is advisory by default — a noisy
  gate gets uninstalled.
- **Links are decided by explicit markers, never by inference.** LLM-based
  suggestions (excavate) produce reviewable diffs; they never feed `check`.
- **Zero third-party deps in the core.** pkg/* is stdlib-only. New deps
  need a reason in the PR description; parsing libraries (goldmark, SCIP)
  land only where the roadmap calls for them.

## Working

```bash
go test ./...     # full suite
go vet ./...      # lint gate — CI enforces this
go build ./cmd/spec-blame ./cmd/spec-excavate
```

CI runs tests on Go 1.24–1.26 and blocks merge.

See ROADMAP.md for the seam-organized extension plan and EXPLORATION.md
for the market/context research behind the design.
