package ingest

import (
	"strings"
	"testing"
	"testing/fstest"
)

// spec trees rooted below --root (monorepo layout): every
// */openspec/specs/<cap>/*.md and */specs/<feature>/spec.md counts.
var multiRootFS = fstest.MapFS{
	"a/openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth
### Requirement: Token expiry
Tokens expire.
`)},
	"b/openspec/specs/billing/spec.md": &fstest.MapFile{Data: []byte(`# Billing
### Requirement: Idempotent invoices
Invoices are idempotent.
`)},
	// dependency trees never host specs
	"vendor/lib/openspec/specs/vend/spec.md": &fstest.MapFile{Data: []byte(`# V
### Requirement: Nope
x
`)},
	"node_modules/lib/openspec/specs/nm/spec.md": &fstest.MapFile{Data: []byte(`# N
### Requirement: Nope
x
`)},
	".hidden/openspec/specs/h/spec.md": &fstest.MapFile{Data: []byte(`# H
### Requirement: Nope
x
`)},
}

func TestOpenspecMultiRoot(t *testing.T) {
	a := &openspecAdapter{}
	if !a.Detect(multiRootFS) {
		t.Fatal("should detect nested openspec/specs trees")
	}
	g, err := a.Ingest(multiRootFS)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"auth/token-expiry", "billing/idempotent-invoices"} {
		if g.Requirements[id] == nil {
			t.Fatalf("missing %s: %v", id, g.IDs())
		}
	}
	if len(g.IDs()) != 2 {
		t.Fatalf("vendor/node_modules/.hidden trees must be skipped: %v", g.IDs())
	}
	if got := g.Requirements["auth/token-expiry"].Source; !strings.HasPrefix(got, "a/openspec/") {
		t.Fatalf("source should keep the nested path: %q", got)
	}
}

func TestSpecKitMultiRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"svc/specs/001-pay/spec.md": &fstest.MapFile{Data: []byte(`# Pay
- **FR-001**: Payments MUST be idempotent
`)},
		// an openspec spec with FR-looking bullets is openspec's, not
		// spec-kit's — no double ownership
		"a/openspec/specs/x/spec.md": &fstest.MapFile{Data: []byte(`# X
- **FR-009**: not a spec-kit spec
`)},
		"vendor/pkgs/specs/002-v/spec.md": &fstest.MapFile{Data: []byte(`# V
- **FR-001**: vendored
`)},
	}
	a := &speckitAdapter{}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if g.Requirements["001-pay/FR-001"] == nil {
		t.Fatalf("nested spec-kit spec not discovered: %v", g.IDs())
	}
	if g.Requirements["x/FR-009"] != nil {
		t.Fatalf("openspec tree must not be claimed by spec-kit: %v", g.IDs())
	}
	if len(g.IDs()) != 1 {
		t.Fatalf("want only 001-pay/FR-001, got %v", g.IDs())
	}
}

func TestMultiRootDuplicateID(t *testing.T) {
	// same capability + requirement title in two roots → same derived
	// ID; merging must fail loudly, not silently pick one
	fsys := fstest.MapFS{
		"a/openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# A
### Requirement: Token expiry
one
`)},
		"b/openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# B
### Requirement: Token expiry
two
`)},
	}
	if _, err := Load(fsys); err == nil {
		t.Fatal("duplicate requirement ID across roots must error")
	}
}

func TestMdIDSIgnoresNestedSpecTrees(t *testing.T) {
	// ID-looking headings inside ANY openspec/ or specs/ tree belong to
	// those adapters, wherever the tree sits
	fsys := fstest.MapFS{
		"a/openspec/changes/x/proposal.md": &fstest.MapFile{Data: []byte(`## REQ-1: A
x
## REQ-2: B
y
`)},
		"svc/specs/001-pay/notes.md": &fstest.MapFile{Data: []byte(`## REQ-3: C
x
## REQ-4: D
y
`)},
		"docs/reqs.md": &fstest.MapFile{Data: []byte(`## REQ-9: Real
x
## REQ-10: Also real
y
`)},
	}
	a := &mdidsAdapter{}
	g, err := a.Ingest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if g.Requirements["REQ-9"] == nil || len(g.IDs()) != 2 {
		t.Fatalf("want only docs/reqs.md ids, got %v", g.IDs())
	}
}
