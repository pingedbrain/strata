// Package emit renders canonical artifacts: openspec-compatible spec.md
// skeletons from mined candidates, and marker insertion suggestions.
// Everything it writes is a draft meant for human review.
package emit

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/pingedbrain/strata/pkg/mine"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// reqTitles returns a unique requirement title per candidate. Same-name
// symbols inside a capability (e.g. one `apply` per module) get the
// source file stem prepended: apply in latency.go → "Latency Apply" →
// ID "effects/latency-apply". Ingest derives IDs as cap + "/" + Slug(title),
// so titles and markers agree through this single source of truth.
func reqTitles(c mine.Capability) []string {
	used := map[string]bool{}
	titles := make([]string, len(c.Candidates))
	for i, cand := range c.Candidates {
		t := title(cand.Name)
		if used[reqgraph.Slug(t)] {
			stem := path.Base(cand.Evidence.File)
			stem = strings.TrimSuffix(stem, path.Ext(stem))
			t = title(stem + " " + cand.Name)
		}
		used[reqgraph.Slug(t)] = true
		titles[i] = t
	}
	return titles
}

// reqID is the full marker-facing ID for a candidate title.
func reqID(capName, title string) string {
	return capName + "/" + reqgraph.Slug(title)
}

// Spec renders an openspec-compatible spec.md draft for one capability.
func Spec(c mine.Capability) string {
	titles := reqTitles(c)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Specification\n\n## Requirements\n\n", title(c.Name))
	for i, cand := range c.Candidates {
		kind := cand.Kind
		if kind == "" {
			kind = "symbol"
		}
		fmt.Fprintf(&b, "### Requirement: %s\n", titles[i])
		fmt.Fprintf(&b, "TODO: describe the behavior this %s must guarantee.\n\n", kind)
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
		titles := reqTitles(c)
		for i, cand := range c.Candidates {
			out = append(out, MarkerSuggestion{
				File:    cand.Evidence.File,
				Line:    cand.Evidence.Line,
				Comment: commentPrefix(cand.Evidence.File) + " @spec " + reqID(c.Name, titles[i]),
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
	words := strings.Split(out, " ")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
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
