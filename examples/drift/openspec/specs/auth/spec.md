# Auth Specification

## Requirements

### Requirement: Token expiry
Tokens expire after 1 hour (edited — was 24h when code was annotated).

#### Scenario: Expired rejected
- **WHEN** token is old
- **THEN** reject it
