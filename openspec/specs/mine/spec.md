# Mine Specification

## Requirements

### Requirement: Candidate
Candidate is a symbol that plausibly backs a requirement.

_Evidence: pkg/mine/mine.go:18_

### Requirement: Capability
Capability is a group of candidates, named after the directory.

_Evidence: pkg/mine/mine.go:27_

### Requirement: Scan
Scan walks fsys, extracts symbols, groups candidates by directory
capability, and links tests to candidates by normalized name.

_Evidence: pkg/mine/mine.go:34_

#### Scenario: Scan File All Comment Styles
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Groups Capabilities And Links Tests
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Python
- **WHEN** TODO
- **THEN** TODO

### Requirement: File Stats
FileStats summarizes how a file evolves in git history.

_Evidence: pkg/mine/history.go:12_

### Requirement: Coupling
Coupling is a pair of files that change together — a hint that they
may belong to the same requirement boundary even across directories.

_Evidence: pkg/mine/history.go:20_

### Requirement: Parse History
ParseHistory reads `git log --numstat --format=COMMIT` output. COMMIT
lines delimit commits; numstat lines are "added<TAB>deleted<TAB>path".
Binary files show "-" counts and contribute changes but no lines.

_Evidence: pkg/mine/history.go:28_

#### Scenario: Parse History
- **WHEN** TODO
- **THEN** TODO

### Requirement: Hot Files
HotFiles returns paths sorted by change count, descending.

_Evidence: pkg/mine/history.go:81_

