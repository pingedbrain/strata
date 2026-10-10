package graph

import (
	"sort"

	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/junit"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// ApplyTestResults merges JUnit test outcomes into the join: a
// requirement is Verified when a passing test plausibly covers it —
// either via a bound symbol (index.Covers on test name) or directly via
// a scenario name (pytest-bdd emits test names from scenario titles).
// Failing trumps verified. Test data is heuristic — it informs status,
// never the gate.
// @spec graph/result-apply-test-results
func (r *Result) ApplyTestResults(g *reqgraph.Graph, idx *index.Index, tests []junit.Case) {
	if len(tests) == 0 {
		return
	}
	pass := map[index.Symbol]bool{}
	fail := map[index.Symbol]bool{}
	reqPass := map[string]bool{} // scenario-name direct matches
	reqFail := map[string]bool{}
	for _, t := range tests {
		if idx != nil {
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
		// scenario names: test_foo_bar ↔ scenario "Foo bar".
		// "spec" is the synthetic scenario the spec-kit adapter emits —
		// too generic to match (every test_spec_* would cover it).
		for _, id := range g.IDs() {
			for _, sc := range g.Requirements[id].Scenarios {
				if sc.Name == "spec" {
					continue
				}
				if index.Covers(t.Name, sc.Name) || index.BDDStepMatch(t.Name, sc.Name) {
					if t.Failed {
						reqFail[id] = true
					} else if !t.Skipped {
						reqPass[id] = true
					}
					break
				}
			}
		}
	}
	var lookup map[[2]string]index.Symbol
	if idx != nil {
		lookup = map[[2]string]index.Symbol{}
		for f, syms := range idx.Files {
			for _, s := range syms {
				lookup[[2]string{f, s.Name}] = s
			}
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
	for id := range reqPass {
		verified[id] = true
	}
	for id := range reqFail {
		failing[id] = true
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
