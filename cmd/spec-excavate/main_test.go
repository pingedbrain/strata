package main

import "testing"

func TestParseOptsFormat(t *testing.T) {
	o, err := parseOpts(nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.format != "lines" {
		t.Fatalf("default format should be lines, got %q", o.format)
	}
	if _, err := parseOpts([]string{"--format", "diff"}); err != nil {
		t.Fatalf("--format diff rejected: %v", err)
	}
	if _, err := parseOpts([]string{"--format", "bogus"}); err == nil {
		t.Fatal("unknown --format must error")
	}
}
