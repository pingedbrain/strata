// Package emit renders canonical artifacts: openspec-compatible spec.md
// skeletons from mined candidates, and marker insertion suggestions.
// Everything it writes is a draft meant for human review.
package emit

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/pingedbrain/strata/pkg/mine"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Spec renders an openspec-compatible spec.md draft for one capability.
func Spec(c mine.Capability) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Specification\n\n## Requirements\n\n", title(c.Name))
	for _, cand := range c.Candidates {
		fmt.Fprintf(&b, "### Requirement: %s\n", title(cand.Name))
		fmt.Fprintf(&b, "TODO: describe the behavior this %s must guarantee.\n\n", cand.Kind)
		fmt.Fprintf(&b, "_Evidence: %s:%d_\n\n", cand.Evidence.File, cand.Evidence.Line)
		for _, t := range cand.Tests {
			fmt.Fprintf(&b, "#### Scenario: %s\n", title(testTitle(t.Name)))
			b.WriteString("- **WHEN** TODO\n- **THEN** TODO\n\n")
		}
	}
	return b.String()
}

// MarkerSuggestion is a proposed @spec annotation for a symbol.
type MarkerSuggestion struct {
	File    string
	Line    int    // insert the comment before this line
	Comment string // full comment line, e.g. `// @spec auth/validate`
}

// SuggestMarkers proposes an @spec marker above each candidate, in the
// file's comment style.
func SuggestMarkers(caps []mine.Capability) []MarkerSuggestion {
	var out []MarkerSuggestion
	for _, c := range caps {
		for _, cand := range c.Candidates {
			id := c.Name + "/" + reqgraph.Slug(title(cand.Name))
			out = append(out, MarkerSuggestion{
				File:    cand.Evidence.File,
				Line:    cand.Evidence.Line,
				Comment: commentPrefix(cand.Evidence.File) + " @spec " + id,
			})
		}
	}
	return out
}

// title humanizes an identifier: "AcceptanceHash" → "Acceptance Hash",
// "forward_request" → "Forward Request".
func title(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	var b strings.Builder
	prevLower := false
	for _, r := range s {
		if unicode.IsUpper(r) && prevLower {
			b.WriteByte(' ')
		}
		prevLower = unicode.IsLower(r) || unicode.IsDigit(r)
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return s
	}
	return strings.ToUpper(out[:1]) + out[1:]
}

var testPrefix = regexp.MustCompile(`^(?:test_|Test|test)`)

func testTitle(name string) string {
	return testPrefix.ReplaceAllString(name, "")
}

func commentPrefix(file string) string {
	switch {
	case strings.HasSuffix(file, ".py"), strings.HasSuffix(file, ".sh"),
		strings.HasSuffix(file, ".rb"), strings.HasSuffix(file, ".yaml"),
		strings.HasSuffix(file, ".yml"), strings.HasSuffix(file, ".toml"):
		return "#"
	case strings.HasSuffix(file, ".sql"), strings.HasSuffix(file, ".lua"):
		return "--"
	default: // c-family, go, rust, ts...
		return "//"
	}
}
