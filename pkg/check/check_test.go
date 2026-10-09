package check

import (
	"testing"
	"testing/fstest"
)

var fixture = fstest.MapFS{
	"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth

## Requirements

### Requirement: Token expiry
Tokens expire.

#### Scenario: Expired rejected
- **WHEN** token is old
- **THEN** reject it
`)},
	"auth/handler.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/token-expiry
func Validate() {}
`)},
	"auth/stale.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/token-expiry #000000
func Old() {}
`)},
	"auth/ghost.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/does-not-exist
func Ghost() {}
`)},
	"openspec/changes/whatever/notes.md": &fstest.MapFile{Data: []byte(
		"// @spec auth/token-expiry — spec dirs must not be scanned as code\n")},
}

func TestRunGate(t *testing.T) {
	rep, err := Run(fixture, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fails := rep.GateFailures()
	if len(fails) != 2 {
		t.Fatalf("want 2 gate failures (stale + dangling), got %v", fails)
	}
	cov, tot := rep.Result.Coverage()
	if cov != 1 || tot != 1 {
		t.Fatalf("coverage %d/%d, want 1/1", cov, tot)
	}
}

func TestRunNoSpecs(t *testing.T) {
	_, err := Run(fstest.MapFS{"main.go": {}}, Options{})
	if err == nil {
		t.Fatal("repo without specs must error")
	}
}
