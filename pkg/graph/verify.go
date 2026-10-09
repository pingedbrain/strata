package graph

import (
	"sort"

	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/junit"
)

// ApplyTestResults merges JUnit test outcomes into the join: a
// requirement is Verified when a passing test plausibly covers a symbol
// bound to it (index.Covers), and Failing when any covering test failed.
// Test data is heuristic — it informs status, never the gate.
func (r *Result) ApplyTestResults(idx *index.Index, tests []junit.Case) {
	if idx == nil || len(tests) == 0 {
		return
	}
	pass := map[index.Symbol]bool{}
	fail := map[index.Symbol]bool{}
	for _, t := range tests {
		for _, syms := range idx.Files {
			for _, s := range syms {
				if !index.Covers(t.Name, s.Name) {
					continue
				}
				if t.Failed {
					fail[s] = true
				} else if !t.Skipped {
					pass[s] = true
				}
			}
		}
	}
	lookup := map[[2]string]index.Symbol{}
	for f, syms := range idx.Files {
		for _, s := range syms {
			lookup[[2]string{f, s.Name}] = s
		}
	}
	verified := map[string]bool{}
	failing := map[string]bool{}
	for _, e := range r.Edges {
		sym, ok := lookup[[2]string{e.File, e.Symbol}]
		if !ok {
			continue
		}
		if fail[sym] {
			failing[e.ReqID] = true
		} else if pass[sym] {
			verified[e.ReqID] = true
		}
	}
	for id := range failing {
		delete(verified, id) // a failing test trumps a passing one
	}
	r.Verified = sortedKeys(verified)
	r.Failing = sortedKeys(failing)
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
