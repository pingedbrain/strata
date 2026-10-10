# Markers Specification

## Requirements

### Requirement: Kind
Kind distinguishes what a marker claims.

_Evidence: pkg/markers/markers.go:17_

### Requirement: Kind.String
TODO: describe the behavior this method must guarantee.

_Evidence: pkg/markers/markers.go:25_

### Requirement: Marker
Marker is one annotation occurrence.

_Evidence: pkg/markers/markers.go:39_

#### Scenario: Suggest Markers
- **WHEN** TODO
- **THEN** TODO

### Requirement: Scan File
ScanFile extracts every marker in a file.

_Evidence: pkg/markers/markers.go:48_

#### Scenario: Scan File All Comment Styles
- **WHEN** TODO
- **THEN** TODO

### Requirement: Scan
Scan reads markers from any stream. name is recorded on each Marker.
Files with lines beyond 1 MiB (lockfiles, minified bundles, JSON
captures) can't host markers — they're skipped, not fatal.

_Evidence: pkg/markers/scan.go:12_

#### Scenario: Scan File All Comment Styles
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Groups Capabilities And Links Tests
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Python
- **WHEN** TODO
- **THEN** TODO

### Requirement: Scan Dir
ScanDir walks fsys and scans every file not excluded by skip.
skip(path, isDir) returning true prunes dirs and ignores files.

_Evidence: pkg/markers/scan.go:37_

### Requirement: Fix
Fix rewrites the bound hash of one marker occurrence.

_Evidence: pkg/markers/rehash.go:12_

#### Scenario: TUIFix Stale
- **WHEN** TODO
- **THEN** TODO

### Requirement: Apply Fixes
ApplyFixes rewrites marker hashes in place. Only the marker matching
(file, line, reqID) is touched; other occurrences of the same ID are
left alone.

_Evidence: pkg/markers/rehash.go:22_

