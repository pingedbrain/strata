// SPDX-License-Identifier: Apache-2.0
package check

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSARIF(t *testing.T) {
	fsys := fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# auth

### Requirement: Token expiry
Tokens expire.

#### Scenario: Expired
- **WHEN** old
- **THEN** reject
`)},
		"main.go": &fstest.MapFile{Data: []byte(`package main

// @spec auth/token-expiry #deadbeef
func Validate() {}

// @spec auth/missing
func Ghost() {}
`)},
	}
	rep, err := Run(fsys, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := rep.SARIF()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
				Level  string `json:"level"`
				Loc    []struct {
					PL struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Version != "2.1.0" || len(doc.Runs) != 1 {
		t.Fatal("bad sarif envelope")
	}
	rules := map[string]int{}
	for _, r := range doc.Runs[0].Results {
		rules[r.RuleID]++
	}
	if rules["dangling-marker"] != 1 || rules["stale-marker"] != 1 {
		t.Fatalf("rules: %v", rules)
	}
	// dangling location points at the marker line in main.go
	for _, r := range doc.Runs[0].Results {
		if r.RuleID == "dangling-marker" {
			if r.Loc[0].PL.ArtifactLocation.URI != "main.go" || r.Loc[0].PL.Region.StartLine != 6 {
				t.Fatalf("loc: %+v", r.Loc)
			}
		}
	}
	if !strings.Contains(string(b), `"$schema"`) {
		t.Fatal("sarif should declare its schema")
	}
}
