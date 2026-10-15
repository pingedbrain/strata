package main

import "testing"

func TestParseRepoArgsFormats(t *testing.T) {
	for _, f := range []string{"text", "sarif", "json"} {
		if _, err := parseRepoArgs([]string{"--format", f}); err != nil {
			t.Fatalf("--format %s rejected: %v", f, err)
		}
	}
	if _, err := parseRepoArgs([]string{"--format", "yaml"}); err == nil {
		t.Fatal("unknown --format must error non-zero, not fall back to text")
	}
}

func TestParseRepoArgsJUnitRepeatable(t *testing.T) {
	ra, err := parseRepoArgs([]string{"--junit", "a.xml", "--junit", "b.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ra.opts.JUnit) != 2 || ra.opts.JUnit[0] != "a.xml" || ra.opts.JUnit[1] != "b.xml" {
		t.Fatalf("repeated --junit should merge both files: %v", ra.opts.JUnit)
	}
}
