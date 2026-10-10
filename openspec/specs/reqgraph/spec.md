# Reqgraph Specification

## Requirements

### Requirement: Requirement
Requirement is one spec requirement with a stable ID.

_Evidence: pkg/reqgraph/reqgraph.go:15_

### Requirement: Scenario
Scenario is one Given/When/Then-style acceptance case.

_Evidence: pkg/reqgraph/reqgraph.go:25_

#### Scenario: Apply Test Results Scenario Names
- **WHEN** TODO
- **THEN** TODO

### Requirement: Graph
Graph is the requirement side of the traceability map.

_Evidence: pkg/reqgraph/reqgraph.go:31_

### Requirement: New
TODO: describe the behavior this func must guarantee.

_Evidence: pkg/reqgraph/reqgraph.go:36_

### Requirement: Graph.Add
Add registers a requirement. Duplicate IDs are an error — requirement
identity must be unambiguous for markers to be trustworthy.

_Evidence: pkg/reqgraph/reqgraph.go:42_

### Requirement: Graph.IDs
IDs returns requirement IDs in insertion order.

_Evidence: pkg/reqgraph/reqgraph.go:55_

### Requirement: Slug
Slug converts a requirement title into the ID form markers reference:
"Token expiry" → "token-expiry". Ingest adapters and emit share this so
generated specs and code markers agree on IDs.

_Evidence: pkg/reqgraph/reqgraph.go:64_

### Requirement: Requirement.Acceptance Hash
AcceptanceHash returns the short hash bound into @spec markers
(the "#a3f2b1" suffix). Hashing normalized text — not raw bytes —
keeps the hash stable across whitespace-only edits.

_Evidence: pkg/reqgraph/reqgraph.go:71_

