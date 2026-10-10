# Ingest Specification

## Requirements

### Requirement: Adapter
Adapter reads one spec format into the canonical graph.

_Evidence: pkg/ingest/ingest.go:13_

#### Scenario: Spec Kit Adapter
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Gherkin Adapter
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Md IDs Adapter
- **WHEN** TODO
- **THEN** TODO

### Requirement: Register
Register adds an adapter; called from each adapter file's init.

_Evidence: pkg/ingest/ingest.go:25_

### Requirement: Detect All
DetectAll returns every adapter whose format is present under dir.

_Evidence: pkg/ingest/ingest.go:28_

### Requirement: Load
Load runs every detected adapter and merges the graphs. Duplicate
requirement IDs across formats surface as errors via Graph.Add.

_Evidence: pkg/ingest/ingest.go:40_

#### Scenario: Load SCIP
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Load Merges Formats
- **WHEN** TODO
- **THEN** TODO

