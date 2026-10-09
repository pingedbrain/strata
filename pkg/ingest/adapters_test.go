// SPDX-License-Identifier: Apache-2.0
package ingest

import (
	"testing"
	"testing/fstest"
)

func TestSpecKitAdapter(t *testing.T) {
	fsys := fstest.MapFS{
		"specs/001-auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth Feature

## Requirements

- **FR-001**: The system MUST reject expired tokens
- **FR-002**: The system MUST rotate refresh tokens
`)},
		"specs/002-billing/spec.md": &fstest.MapFile{Data: []byte(`# Billing

## Requirements

- **FR-001**: Invoices MUST be idempotent
`)},
	}
	a := &speckitAdapter{}
	if !a.Detect(fsys) {
		t.Fatal("should detect spec-kit")
	}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.IDs()) != 3 {
		t.Fatalf("ids: %v", g.IDs())
	}
	r := g.Requirements["001-auth/FR-001"]
	if r == nil || r.Title != "FR-001: The system MUST reject expired tokens" {
		t.Fatalf("%+v", r)
	}
	// hash covers the FR text → drift detectable
	if r.AcceptanceHash() == "" {
		t.Fatal("no acceptance hash")
	}
}

func TestGherkinAdapter(t *testing.T) {
	fsys := fstest.MapFS{
		"features/auth.feature": &fstest.MapFile{Data: []byte(`Feature: Authentication
  Users log in

  Scenario: Valid login
    Given valid credentials
    When login is posted
    Then a session is created

  Scenario Outline: Rate limiting
    Given <n> attempts
    Then blocked
`)},
	}
	a := &gherkinAdapter{}
	if !a.Detect(fsys) {
		t.Fatal("should detect gherkin")
	}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	r := g.Requirements["auth/valid-login"]
	if r == nil {
		t.Fatalf("ids: %v", g.IDs())
	}
	if len(r.Scenarios) != 1 || len(r.Scenarios[0].Steps) != 3 {
		t.Fatalf("scenarios: %+v", r.Scenarios)
	}
	if r.Scenarios[0].Steps[2] != "a session is created" {
		t.Fatalf("steps: %+v", r.Scenarios[0].Steps)
	}
	if g.Requirements["auth/rate-limiting"] == nil {
		t.Fatal("Scenario Outline should parse")
	}
}

func TestMdIDsAdapter(t *testing.T) {
	fsys := fstest.MapFS{
		"docs/requirements.md": &fstest.MapFile{Data: []byte(`# Spec

## REQ-042: Tokens expire
All tokens must expire.

### [AUTH-7] Refresh rotation
Refresh tokens rotate on use.
`)},
		"README.md": &fstest.MapFile{Data: []byte("# project\nno ids here\n")},
	}
	a := &mdidsAdapter{}
	if !a.Detect(fsys) {
		t.Fatal("should detect md-ids")
	}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if g.Requirements["REQ-042"] == nil || g.Requirements["AUTH-7"] == nil {
		t.Fatalf("ids: %v", g.IDs())
	}
	if got := g.Requirements["REQ-042"].Text; got != "All tokens must expire." {
		t.Fatalf("text: %q", got)
	}
}

func TestMdIDsDoesNotEatOpenSpec(t *testing.T) {
	// openspec dirs are owned by the openspec adapter — md-ids must
	// not double-count them even if headings look ID-ish
	fsys := fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# auth
### Requirement: Tokens
`)},
	}
	a := &mdidsAdapter{}
	if a.Detect(fsys) {
		t.Fatal("must not detect inside openspec/")
	}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.IDs()) != 0 {
		t.Fatalf("expected empty: %v", g.IDs())
	}
}

func TestLoadMergesFormats(t *testing.T) {
	fsys := fstest.MapFS{
		"openspec/specs/hello/spec.md": &fstest.MapFile{Data: []byte(`# hello
### Requirement: Greeting
Say hi.
`)},
		"docs/reqs.md": &fstest.MapFile{Data: []byte(`## REQ-1: A
x
## REQ-2: B
y
`)},
	}
	g, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.IDs()) != 3 {
		t.Fatalf("merged ids: %v", g.IDs())
	}
}
