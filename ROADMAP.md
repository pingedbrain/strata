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
- ~~Markdown-with-IDs adapter~~ ✅ — `## REQ-001:` / `### [AUTH-7]`
  headings in any md outside spec dirs; IDs verbatim, ≥2 per file to
  detect.
- ~~Gherkin `.feature` adapter~~ ✅ — Scenario/Scenario Outline →
  requirement, steps → acceptance. BDD step-def→test linkage still open.
- ~~spec-kit adapter~~ ✅ — `specs/<NNN-feature>/spec.md` `**FR-NNN**`
  bullets → `<feature>/FR-NNN`; FR text rides in a scenario so hashes
  catch drift.
- GitHub issues adapter — `gh` CLI or REST; issues as requirements.
  `[size:M]` (post-MVP; needs auth story)

### `pkg/markers` — code annotations

- ~~**(MVP)** Marker grammar: `@spec <req-id> #<hash>`~~ ✅ — any comment
  style (`//`, `#`, `/*`, `--`), structured grep, `Scan`/`ScanFile`/`ScanDir`.
- ~~**(MVP)** Staleness~~ ✅ + ~~**(MVP)** Dangling refs~~ ✅ — in pkg/graph Join.
- ~~Marker auto-fix: `spec-blame sync`~~ ✅ — rewrites stale bound hashes
  by (file, line, reqID); leaves same-ID markers elsewhere untouched.

### `pkg/index` — code side

- ~~**(MVP)** File-level index only~~ ✅ superseded — `index.Scan` now
  extracts symbols natively (go AST, py/ts regex), `SymbolBelow` binds a
  marker to the declaration under it. File-level remains the fallback for
  unindexed languages.
- ~~Symbol extraction seam~~ ✅ — `pkg/index` owns `Symbol{Name,Kind,File,
  Line,Exported}`; `pkg/mine` consumes it. SCIP ingestion can now replace
  the native extractors behind the same `Index` shape.
- ~~SCIP consumer~~ ✅ — `pkg/index.LoadSCIP` decodes `index.scip`
  (protobuf wire, zero deps), overlays onto native extraction.
  Auto-detects `index.scip` at root or `--scip <path>`. Validated
  against real scip-go output.
- SCIP refs/relationships — we only consume definitions today;
  references could power usage-based drift hints. `[size:M]`
- tree-sitter via WASM (wazero, no CGO) — own grammars, symbol extraction
  as fallback when no SCIP indexer exists. `[size:L]`
- ~~Dead-code radar~~ ✅ — exported symbols with no marker surface as
  `Result.Unlinked` and in `blame`. Advisory only, never gate. `[size:M]`

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
- ~~**(MVP)** `blame <file>`~~ ✅ — symbol-level: shows the declaration
  each marker binds to, plus unlinked symbols in the file.
- ~~**(MVP)** `map <req>` / `map <file>`~~ ✅ — bidirectional lookup,
  basename suffix matching. `[size:S]`
- ~~`serve`~~ ✅ — MCP stdio server, hand-rolled JSON-RPC (zero deps):
  `strata_reqs_for_file`, `strata_files_for_req`, `strata_coverage`,
  `strata_stale`, `strata_dangling`. Fresh graph per call.
- ~~`serve` symbol-level tools~~ ✅ — `strata_req_for_symbol`,
  `strata_symbols_for_req`, `strata_unlinked` over the index.
- ~~`tui`~~ ✅ — bubbletea explorer: req list (uncovered first) + detail
  pane with scenarios and bound symbols, stale/dangling/unlinked stats,
  `f` fixes stale markers in place (sync without leaving the UI).
- ~~`badge`~~ ✅ — `spec-blame badge` emits shields.io endpoint JSON.
- ~~SARIF~~ ✅ — `check --format sarif` emits SARIF 2.1.0 for GitHub
  code scanning (dangling=error, stale=warning, unlinked=note).
- ~~JUnit merge~~ ✅ — `check|coverage --junit results.xml` marks
  requirements verified/failing via test-name→symbol matching.

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
