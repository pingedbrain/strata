// Package reqgraph defines the canonical requirement IR shared by
// spec-blame and spec-excavate. Adapters in pkg/ingest produce this
// graph from whatever spec format a repo already uses; strata never
// defines its own spec format.
package reqgraph

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// Requirement is one spec requirement with a stable ID.
type Requirement struct {
	ID        string // e.g. "AUTH-04", "auth/login-retry"
	Title     string
	Text      string     // normative body
	Parent    string     // parent requirement ID, if any
	Source    string     // file:line or URL where the requirement lives
	Scenarios []Scenario // acceptance criteria
}

// Scenario is one Given/When/Then-style acceptance case.
type Scenario struct {
	Name  string
	Steps []string
}

// Graph is the requirement side of the traceability map.
type Graph struct {
	Requirements map[string]*Requirement
	order        []string // insertion order, for deterministic output
}

func New() *Graph {
	return &Graph{Requirements: map[string]*Requirement{}}
}

// Add registers a requirement. Duplicate IDs are an error — requirement
// identity must be unambiguous for markers to be trustworthy.
func (g *Graph) Add(r *Requirement) error {
	if r.ID == "" {
		return fmt.Errorf("requirement with empty ID at %s", r.Source)
	}
	if _, dup := g.Requirements[r.ID]; dup {
		return fmt.Errorf("duplicate requirement ID %q", r.ID)
	}
	g.Requirements[r.ID] = r
	g.order = append(g.order, r.ID)
	return nil
}

// IDs returns requirement IDs in insertion order.
func (g *Graph) IDs() []string { return g.order }

var ws = regexp.MustCompile(`\s+`)

// AcceptanceHash returns the short hash bound into @spec markers
// (the "#a3f2b1" suffix). Hashing normalized text — not raw bytes —
// keeps the hash stable across whitespace-only edits.
func (r *Requirement) AcceptanceHash() string {
	var b strings.Builder
	b.WriteString(r.Title)
	for _, s := range r.Scenarios {
		b.WriteString(s.Name)
		for _, step := range s.Steps {
			b.WriteString(step)
		}
	}
	sum := sha256.Sum256([]byte(ws.ReplaceAllString(b.String(), " ")))
	return fmt.Sprintf("%x", sum)[:6]
}
