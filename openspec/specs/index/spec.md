# Index Specification

## Requirements

### Requirement: Load SCIP
LoadSCIP reads a SCIP index (protobuf wire format) produced by
language indexers like scip-go or scip-python and converts symbol
definitions into an Index. Hand-rolled wire decoding — protobuf
fields we don't need are skipped, so unknown/newer fields are fine.

_Evidence: pkg/index/scip.go:14_

#### Scenario: Load SCIP
- **WHEN** TODO
- **THEN** TODO

### Requirement: Index.Overlay
Overlay merges richer symbol data (typically SCIP) into i: files the
other index covers are replaced outright, others are left alone.
SCIP data wins over regex extraction for the same file.

_Evidence: pkg/index/scip.go:31_

### Requirement: Symbol
Symbol is a named code entity with an exact location.

_Evidence: pkg/index/index.go:13_

#### Scenario: Join Resolves Symbols
- **WHEN** TODO
- **THEN** TODO

#### Scenario: SCIPSymbol Name
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Symbol Tools
- **WHEN** TODO
- **THEN** TODO

### Requirement: Index
Index maps files to their symbols, plus test symbols and BDD
step-definition texts (pytest-bdd decorators, cucumber Given()).

_Evidence: pkg/index/index.go:24_

### Requirement: Extractor
Extractor pulls exported symbols, test names, and BDD step-definition
texts from one language family.

_Evidence: pkg/index/index.go:35_

#### Scenario: Generic Extractor
- **WHEN** TODO
- **THEN** TODO

### Requirement: Register Extractor
RegisterExtractor appends a language extractor — the seam where a
tree-sitter WASM or bespoke plugin lands. Extractors are tried in
registration order; the built-ins (go, python, typescript, generic
c-family) are registered first.

_Evidence: pkg/index/index.go:45_

### Requirement: Scan
Scan walks fsys and extracts symbols for every recognized source file
not skipped. skip(path, isDir)=true prunes dirs and ignores files;
nil scans everything.

_Evidence: pkg/index/index.go:52_

#### Scenario: Scan File All Comment Styles
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Groups Capabilities And Links Tests
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Scan Python
- **WHEN** TODO
- **THEN** TODO

### Requirement: Index.Symbol Below
SymbolBelow returns the first declared symbol after line — the symbol a
comment marker annotates.

_Evidence: pkg/index/index.go:91_

### Requirement: File Match
FileMatch accepts exact paths or basename suffixes ("main.go" matches
"cmd/sub/main.go").

_Evidence: pkg/index/index.go:104_

### Requirement: Covers
Covers reports whether a test name plausibly exercises a symbol:
normalized containment, case/underscore/space-insensitive.
"TestValidateToken" covers "ValidateToken" and "validate_token".

_Evidence: pkg/index/index.go:116_

### Requirement: Index.Files Matching
FilesMatching returns index keys matching a path or basename.

_Evidence: pkg/index/index.go:128_

### Requirement: BDDStep Match
BDD step patterns can carry parameters: pytest-bdd templates
("I have {n} cukes"), cucumber expressions, or regex literals
("^I have (\d+) cukes$"). BDDStepMatch reports whether a step-def
pattern plausibly covers a scenario step: literal chunks must appear
in order after normalization (alnum-only, lowercased).

_Evidence: pkg/index/bdd.go:13_

