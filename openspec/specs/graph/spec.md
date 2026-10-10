# Graph Specification

## Requirements

### Requirement: Edge
Edge is a resolved link between a marker and a requirement.

_Evidence: pkg/graph/graph.go:15_

#### Scenario: Join Classifies Edges
- **WHEN** TODO
- **THEN** TODO

### Requirement: Result
Result is the joined traceability picture.

_Evidence: pkg/graph/graph.go:22_

#### Scenario: Apply Test Results
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Apply Test Results Scenario Names
- **WHEN** TODO
- **THEN** TODO

### Requirement: Join
Join resolves every marker against the requirement graph. idx may be
nil (file-level mode); when set, each marker binds to the symbol
declared directly below it and unlinked symbols surface in Result.
Impl-links are Spec/Implements kinds; Verifies counts separately.

_Evidence: pkg/graph/graph.go:35_

#### Scenario: Join Classifies Edges
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Join Resolves Symbols
- **WHEN** TODO
- **THEN** TODO

### Requirement: Result.Coverage
Coverage returns implemented / total requirements.

_Evidence: pkg/graph/graph.go:86_

### Requirement: Result.Edges In
EdgesIn returns resolved links whose marker sits in file.

_Evidence: pkg/graph/graph.go:97_

### Requirement: Result.Dangling In
DanglingIn returns unresolved markers sitting in file.

_Evidence: pkg/graph/graph.go:108_

### Requirement: Result.Edges For
EdgesFor returns resolved links pointing at requirement id.

_Evidence: pkg/graph/graph.go:119_

### Requirement: Result.Apply Test Results
ApplyTestResults merges JUnit test outcomes into the join: a
requirement is Verified when a passing test plausibly covers it —
either via a bound symbol (index.Covers on test name) or directly via
a scenario name (pytest-bdd emits test names from scenario titles).
Failing trumps verified. Test data is heuristic — it informs status,
never the gate.

_Evidence: pkg/graph/verify.go:17_

