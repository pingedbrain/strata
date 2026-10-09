# strata

**Requirements traceability for spec-driven development.**

Two binaries, one requirement graph:

- **`spec-blame`** — verifies spec→code: `git blame`, but every marked
  *symbol* answers "which requirement justifies me". CI gate for dangling
  refs, stale spec bindings, uncovered requirements — plus a Bubbletea
  TUI and an MCP server for agents.
- **`spec-excavate`** — mines code→spec: scans a brownfield repo and
  proposes the `spec.md` it should have had, plus suggested markers.

strata never invents a spec format — it ingests the ones you already use
(openspec, spec-kit, Gherkin `.feature`, markdown-with-IDs; see
`examples/multiformat` for a mixed repo).

> Status: early. `spec-blame check|coverage|blame|map|sync|serve|badge|tui`
> works today — symbol-level on Go, Python and TypeScript (plus real
> SCIP indexes via `--scip`), SARIF + JUnit output for CI.
> See [ROADMAP.md](ROADMAP.md) and [EXPLORATION.md](EXPLORATION.md).

## Try it

```bash
go run ./cmd/spec-blame check --root examples/quickstart    # passes
go run ./cmd/spec-blame check --root examples/drift         # fails: stale + dangling
go run ./cmd/spec-blame blame --root examples/drift main.go # symbol-level
go run ./cmd/spec-blame map   --root examples/drift auth/token-expiry
go run ./cmd/spec-blame tui   --root examples/drift         # interactive
go run ./cmd/spec-blame check --root examples/multiformat --junit junit.xml
```

## BDD: markers not required

Gherkin scenarios get verification links automatically: pytest-bdd
`@given/@when/@then` decorators, `scenarios("x.feature")` refs, and
cucumber.js `Given()/When()/Then()` defs whose text matches a scenario
step synthesize `verifies` edges — no `@spec` marker needed. Pass
`--junit results.xml` and scenario names (`test_valid_login`) mark the
requirement `verified`/`failing` (see `examples/multiformat`).

Mark code with a comment in any comment style:

```go
// @spec auth/token-expiry #a3f2b1
func Validate() {}
```

The `#a3f2b1` is a hash of the acceptance criteria — when the spec
changes, the marker goes stale and `check` fails. Omit it to link without
drift detection.

## GitHub Action

```yaml
- uses: pingedbrain/strata@main   # tag pin recommended once released
```

## Build

```bash
go build ./cmd/spec-blame ./cmd/spec-excavate
```

## License

MIT
