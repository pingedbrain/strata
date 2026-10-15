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

`check` and `coverage` take `--format text|sarif|json` and a repeatable
`--junit` (merge several result files). The JSON report is a stable
schema for tooling:

```json
{
  "tool": "spec-blame", "version": 1, "formats": ["openspec"],
  "requirements": [{"id": "…", "title": "…", "source": "file:line",
                    "hash": "a3f2b1", "linked": true, "status": "verified"}],
  "edges":    [{"req_id": "…", "kind": "spec", "file": "…", "line": 3,
                "hash": "…", "symbol": "…", "stale": false}],
  "dangling": [{"req_id": "…", "kind": "spec", "file": "…", "line": 3}],
  "unlinked": [{"name": "…", "kind": "func", "file": "…", "line": 1}],
  "coverage": {"linked": 3, "total": 4, "fraction": 0.75},
  "verified": ["…"], "failing": ["…"],
  "gate": {"pass": false, "failures": ["stale: …"]}
}
```

## BDD: markers not required

Gherkin scenarios get verification links automatically: pytest-bdd
`@given/@when/@then` decorators, `scenarios("x.feature")` refs, and
cucumber.js `Given()/When()/Then()` defs whose text matches a scenario
step synthesize `verifies` edges — no `@spec` marker needed. Pass
`--junit results.xml` and scenario names (`test_valid_login`) mark the
requirement `verified`/`failing` (see `examples/multiformat`).

## Excavating specs from brownfield

```bash
spec-excavate scan    --root .            # candidates by capability
spec-excavate propose --root .            # openspec skeletons to stdout
spec-excavate propose --root . --write    # write openspec/specs/*/spec.md
spec-excavate suggest-markers --root .    # @spec annotations to review
spec-excavate suggest-markers --root . --format diff  # as a unified diff
spec-excavate suggest-markers --root . --write        # apply in place
```

Suggested markers carry the `#hash` pin of the requirement `propose`
emits, so links get drift detection from day one — apply both and
`check` passes immediately; edit the spec afterwards and the marker
goes stale.

Doc comments/docstrings become requirement descriptions automatically.
For symbols without docs, `propose` can ask an external assistant —
provider-agnostic, prompt on stdin, one sentence on stdout:

```bash
spec-excavate propose --root . --enrich-cmd "claude -p"
spec-excavate propose --root . --enrich-cmd "ollama run llama3"
spec-excavate propose --root . --dump-prompts   # feed prompts manually
spec-excavate history --root .                  # git hot files + coupling
```

## Ignore & impact

Specs are discovered anywhere under `--root`, so monorepos can keep
per-package `*/openspec/specs/` or `*/specs/` (spec-kit) trees — all are
merged into one graph, and a requirement ID appearing in two roots is an
error rather than a silent merge. `vendor/`, `node_modules/` and dot-dirs
are skipped during discovery.

`.strataignore` at the repo root excludes paths from specs ingestion and
marker/symbol scanning — one pattern per line: `dir/`, `*.ext`, or an
exact path. Markdown docs and `*.scip` artifacts are always code-side
excluded.

```bash
spec-blame impact --root . pkg/graph/graph.go   # reqs a refactor touches
```

`impact` uses symbol references when a SCIP index is present
(`--scip index.scip`, auto-detected at root): changing a file flags the
requirements bound to its symbols *and* requirements whose code calls
them.

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
