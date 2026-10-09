// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
	"testing/fstest"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pingedbrain/strata/pkg/check"
)

func testReport(t *testing.T) *check.Report {
	fsys := fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# auth

### Requirement: Token expiry
Tokens expire.

#### Scenario: Expired token
- **WHEN** a token is expired
- **THEN** requests are rejected

### Requirement: Orphaned
Nothing implements this.
`)},
		"main.go": &fstest.MapFile{Data: []byte(`package main

// @spec auth/token-expiry
func Validate() {}

func Other() {}
`)},
	}
	rep, err := check.Run(fsys, check.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Graph.Requirements["auth/token-expiry"] == nil {
		t.Fatal("test fixture broken: no requirements parsed")
	}
	return rep
}

func TestTUIModelRows(t *testing.T) {
	m := newTUIModel(testReport(t))
	if len(m.rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(m.rows))
	}
	// uncovered requirement sorts first (✗ > ✓)
	if m.rows[0].status != '✗' || m.rows[0].req.ID != "auth/orphaned" {
		t.Fatalf("uncovered should sort first: %+v", m.rows[0])
	}
	if m.rows[1].status != '✓' {
		t.Fatalf("linked req status: %+v", m.rows[1])
	}
	if len(m.rows[1].edges) != 1 || m.rows[1].edges[0].Symbol != "Validate" {
		t.Fatalf("edge should carry symbol: %+v", m.rows[1].edges)
	}
}

func TestTUIView(t *testing.T) {
	m := newTUIModel(testReport(t))
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = m2.(tuiModel)
	out := m.View()
	for _, want := range []string{"requirements", "auth/orphaned", "auth/token-expiry", "unlinked"} {
		if !strings.Contains(stripANSI(out), want) {
			t.Errorf("view missing %q\n%s", want, out)
		}
	}
	// navigate to the linked req: detail shows the bound symbol
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(tuiModel)
	if d := stripANSI(m.detail()); !strings.Contains(d, "Validate") {
		t.Errorf("detail should show bound symbol\n%s", d)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
		}
		if !inEsc {
			b.WriteRune(r)
		}
		if inEsc && (r == 'm' || r == 'K' || r == 'J' || r == 'H') {
			inEsc = false
		}
	}
	return b.String()
}
