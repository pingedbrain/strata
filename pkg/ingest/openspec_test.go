package ingest

import (
	"testing"
	"testing/fstest"
)

const authSpec = `# Auth Specification

## Purpose

Session management for the API.

## Requirements

### Requirement: Token expiry
Session tokens SHALL expire after 24 hours of inactivity.

#### Scenario: Expired token rejected
- **WHEN** a request presents a token issued more than 24h ago
- **THEN** the request is rejected with 401

#### Scenario: Refresh extends window
- **WHEN** a token is refreshed inside the window
- **THEN** expiry is extended another 24h

### Requirement: Login retry backoff
Failed logins SHALL back off exponentially.

#### Scenario: Backoff applied
- WHEN the same account fails 3 times
- THEN subsequent attempts are delayed
`

func openspecFS() fstest.MapFS {
	return fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(authSpec)},
	}
}

func TestOpenspecDetect(t *testing.T) {
	if !(&openspecAdapter{}).Detect(openspecFS()) {
		t.Fatal("should detect openspec/specs/")
	}
	if (&openspecAdapter{}).Detect(fstest.MapFS{"README.md": {}}) {
		t.Fatal("must not detect a repo without openspec/specs/")
	}
}

func TestOpenspecIngest(t *testing.T) {
	g, err := (&openspecAdapter{}).Ingest(openspecFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(g.IDs()) != 2 {
		t.Fatalf("want 2 requirements, got %v", g.IDs())
	}
	r := g.Requirements["auth/token-expiry"]
	if r == nil {
		t.Fatalf("expected derived ID auth/token-expiry, have %v", g.IDs())
	}
	if len(r.Scenarios) != 2 {
		t.Fatalf("want 2 scenarios, got %+v", r.Scenarios)
	}
	if r.Scenarios[0].Name != "Expired token rejected" {
		t.Fatalf("scenario name: %q", r.Scenarios[0].Name)
	}
	if r.Scenarios[0].Steps[0] != "a request presents a token issued more than 24h ago" {
		t.Fatalf("WHEN step parsed wrong: %q", r.Scenarios[0].Steps[0])
	}
	// both - **WHEN** and bare - WHEN forms must parse
	if len(r.Scenarios[1].Steps) != 2 {
		t.Fatalf("bare WHEN/THEN steps not parsed: %+v", r.Scenarios[1].Steps)
	}
	// second requirement's scenarios must not bleed into the first
	if len(g.Requirements["auth/login-retry-backoff"].Scenarios) != 1 {
		t.Fatal("scenario bleeding across requirements")
	}
}
