# Emit Specification

## Requirements

### Requirement: Badge
Badge renders a shields.io endpoint JSON payload for requirement
coverage — drop it behind an endpoint or commit it as a badge.json
and reference shields.io/endpoint?url=…

_Evidence: pkg/emit/badge.go:11_

#### Scenario: Badge
- **WHEN** TODO
- **THEN** TODO

### Requirement: Req Title
ReqTitle returns the requirement title for candidate i — the same
disambiguation Spec applies.

_Evidence: pkg/emit/emit.go:40_

### Requirement: Spec
Spec renders an openspec-compatible spec.md draft for one capability.
Descriptions come from the symbol's doc comment when present; when
absent and enrich is non-nil it is invoked with EnrichPrompt's prompt
and its (trimmed) output becomes the description — the intended hook
for `spec-excavate propose --enrich-cmd`. Otherwise a TODO stands in.

_Evidence: pkg/emit/emit.go:52_

#### Scenario: Run No Specs
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Spec Renders Open Spec Shape
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Spec Uses Doc Then Enrich
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Spec Kit Adapter
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Md IDs Does Not Eat Open Spec
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Openspec Detect
- **WHEN** TODO
- **THEN** TODO

#### Scenario: Openspec Ingest
- **WHEN** TODO
- **THEN** TODO

### Requirement: Enrich Prompt
EnrichPrompt builds the question an external assistant should answer
to write a requirement description. Provider-agnostic: the caller
pipes it to whatever command the user configured.

_Evidence: pkg/emit/emit.go:81_

### Requirement: Marker Suggestion
MarkerSuggestion is a proposed @spec annotation for a symbol.

_Evidence: pkg/emit/emit.go:98_

### Requirement: Suggest Markers
SuggestMarkers proposes an @spec marker above each candidate, in the
file's comment style.

_Evidence: pkg/emit/emit.go:106_

#### Scenario: Suggest Markers
- **WHEN** TODO
- **THEN** TODO

