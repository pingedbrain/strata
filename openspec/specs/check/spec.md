# Check Specification

## Requirements

### Requirement: Report.SARIF
SARIF renders the gate failures as SARIF 2.1.0 — feed it to GitHub
code scanning via `upload-sarif` and findings land on the PR diff.

_Evidence: pkg/check/sarif.go:10_

### Requirement: Options
Options tunes the gate.

_Evidence: pkg/check/check.go:21_

### Requirement: Report
Report is the full gate result.

_Evidence: pkg/check/check.go:35_

### Requirement: Run
Run executes the gate over fsys (a repo root).

_Evidence: pkg/check/check.go:44_

#### Scenario: Run Gate
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Run No Specs
- **WHEN** TODO
- **THEN** TODO

### Requirement: Report.Gate Failures
GateFailures lists the objectively-broken findings. Empty = pass.

_Evidence: pkg/check/check.go:98_

### Requirement: Report.Stale Fixes
StaleFixes converts stale edges into marker rewrites for `sync`.

_Evidence: pkg/check/check.go:119_

### Requirement: Report.Impact
Impact lists requirement IDs affected by changing file: those bound
to symbols defined there, plus those whose implementation files
reference those symbols. The dependent direction needs SCIP
references — without --scip only the direct part applies. Symbol
matching is by name, so same-named symbols in different packages can
over-approximate; it's a drift hint, not a gate.

_Evidence: pkg/check/check.go:139_

