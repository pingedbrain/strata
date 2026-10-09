# strata

**Requirements traceability for spec-driven development.**

Two binaries, one requirement graph:

- **`spec-blame`** — verifies spec→code: `git blame`, but every marked
  region answers "which requirement justifies me". CI gate for dangling
  refs, stale spec bindings, uncovered requirements.
- **`spec-excavate`** — mines code→spec: scans a brownfield repo and
  proposes the `spec.md` it should have had, plus suggested markers.

strata never invents a spec format — it ingests the ones you already use
(openspec, spec-kit, Gherkin, markdown-with-IDs).

> Status: scaffold. See [ROADMAP.md](ROADMAP.md) and
> [EXPLORATION.md](EXPLORATION.md).

## Build

```bash
go build ./cmd/spec-blame ./cmd/spec-excavate
```

## License

MIT
