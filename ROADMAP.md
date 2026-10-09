# Roadmap — strata

Two binaries, one shared core. `spec-blame` verifies spec→code (coverage,
drift, blame); `spec-excavate` mines code→spec (brownfield SDD bootstrap).
Organized by extension seam — each item should be "add a file in the right
package" rather than "touch everything". `[size]` is a rough effort hint;
`[good-first-issue]` marks self-contained items needing no deep context.

Design rule (stolen from spec-trace, validated): **the CI gate only fails
on objectively broken things** — dangling refs, stale markers, unknown IDs.
Coverage levels are advisory by default; a noisy gate gets uninstalled in
a month.

## v0.1 MVP cut

Goal: `spec-blame check` as a useful CI gate on an openspec project, as a
single static binary. Everything marked **(MVP)** below.

### `pkg/reqgraph` — canonical IR

- ~~**(MVP)** `Requirement` type: ID, title, text, parent, source location,
  acceptance-criteria hash~~ ✅ — whitespace-normalized sha256[:6].
- ~~**(MVP)** openspec `spec.md` adapter~~ ✅ — `openspec/specs/*/*.md`,
  `### Requirement:`/`#### Scenario:` + WHEN/THEN bullets (bold and bare),
  IDs derived as `<capability>/<slug>`. Line-parse, no deps. `[size:M]`
- Markdown-with-IDs adapter — `## REQ-001`-style headings, generic format
  for repos that just name requirements. `[size:S]` `[good-first-issue]`
- Gherkin `.feature` adapter — Feature/Scenario Outline → requirements;
  step defs already link to tests for free. `[size:M]` `[good-first-issue]`
- spec-kit adapter — `spec.md`/`plan.md`/`tasks.md` layout. `[size:M]`
- GitHub issues adapter — `gh` CLI or REST; issues as requirements.
  `[size:M]` (post-MVP; needs auth story)

### `pkg/markers` — code annotations

- ~~**(MVP)** Marker grammar: `@spec <req-id> #<hash>`~~ ✅ — any comment
  style (`//`, `#`, `/*`, `--`), structured grep, `Scan`/`ScanFile`/`ScanDir`.
- ~~**(MVP)** Staleness~~ ✅ + ~~**(MVP)** Dangling refs~~ ✅ — in pkg/graph Join.
- Marker auto-fix: `spec-blame sync` rewrites stale hashes after spec
  edits are reviewed (spec-seal proved this UX works). `[size:S]`

### `pkg/index` — code side

- **(MVP)** File-level index only. blame annotates files, not symbols —
  good enough for a gate. `[size:S]`
- SCIP consumer — read `index.scip` produced by language indexers
  (scip-go, scip-python, scip-typescript) → symbol-level locations without
  writing a single parser. `[size:M]`
- tree-sitter via WASM (wazero, no CGO) — own grammars, symbol extraction
  as fallback when no SCIP indexer exists. `[size:L]`
- Dead-code radar — symbols reachable in index with no requirement link.
  Advisory only, never gate. `[size:M]`

### `pkg/graph` — the join

- ~~**(MVP)** Link resolution + coverage computation~~ ✅ — `Join` classifies
  edges (stale/dangling), uncovered reqs, coverage counts. Test-name
  conventions still open. `[size:M]`
- JUnit XML merge — test results turn `covered` into `verified`/`failing`
  (reqcov pattern — auditors want this, costs little). `[size:M]`
- BDD test linkage — pytest-bdd/cucumber step-def graphs feed coverage
  automatically. `[size:M]` `[good-first-issue]`

### `cmd/spec-blame` — verify binary

- ~~**(MVP)** `check`~~ ✅ — gate fails on dangling/stale only, coverage
  advisory (`--min-coverage` opt-in). SARIF output still open. `[size:M]`
- ~~**(MVP)** `coverage`~~ ✅ — % + per-requirement table. `[size:S]`
- ~~**(MVP)** `blame <file>`~~ ✅ — annotations in a file with ok/stale/
  dangling status + requirement titles.
- ~~**(MVP)** `map <req>` / `map <file>`~~ ✅ — bidirectional lookup,
  basename suffix matching. `[size:S]`
- `serve` — MCP server over the graph (`req_for_symbol`,
  `symbols_for_req`, `coverage`, `stale`) so sdd-apply/agents query the
  map while coding. `[size:M]`
- `tui` — bubbletea explorer: req→symbols→tests navigation, stale-refs
  fix flow. `[size:M]` (post-MVP, needs the graph stable)
- `badge` — emit shields.io endpoint JSON for README coverage badges.
  `[size:S]` `[good-first-issue]`

### `cmd/spec-excavate` — mine binary

- `scan` — walk repo: test names, exported API surface, README/docs,
  config → candidate requirement list. `[size:L]`
- `propose` — draft `openspec/specs/**/spec.md` from candidates; LLM via
  any OpenAI-compatible endpoint, **optional** — deterministic skeleton
  without it (req IDs + titles from API/test names). `[size:L]`
- `suggest-markers` — propose `@spec` annotations for existing symbols;
  writes a diff the human reviews. `[size:M]`
- Git-history mining — hot files + change coupling inform requirement
  boundaries. `[size:L]` (post-MVP)

### Distribution

- ~~**(MVP)** GitHub Action~~ ✅ — `action.yml` composite: setup-go +
  `go install` + `spec-blame check`. Needs a tag to pin `@vX`.
- ~~goreleaser pipeline~~ ✅ — `.goreleaser.yaml`, both binaries,
  CGO_ENABLED=0, tar.gz/zip + checksums. Needs a tag to fire.
- SARIF upload docs — results in GitHub code-scanning UI. `[size:S]`

## Explicitly out of scope

- Own spec format — the wedge is *your* specs, not ours.
- Editing specs — strata reads (blame) or proposes (excavate); authoring
  belongs to openspec/spec-kit/editors.
- Compliance exports (ReqIF, DO-178C matrices) — StrictDoc already owns
  that market; revisit only if asked.
- LLM in the gate — inference suggests links, never decides them.
