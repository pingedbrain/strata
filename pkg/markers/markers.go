// Package markers scans source files for requirement annotations:
//
//	@spec <req-id> #<hash>        (any comment style)
//	@implements <req-id>
//	@verifies <req-id>
//
// The scan is a structured grep — no language parsing. Comment syntax is
// irrelevant because the marker itself is the same in every language.
package markers

import (
	"os"
	"regexp"
)

// Kind distinguishes what a marker claims.
// @spec markers/kind
type Kind int

const (
	Spec       Kind = iota // @spec — this code implements the requirement
	Implements             // @implements — explicit impl claim
	Verifies               // @verifies — this test verifies it
)

// @spec markers/kind-string
func (k Kind) String() string {
	switch k {
	case Implements:
		return "implements"
	case Verifies:
		return "verifies"
	default:
		return "spec"
	}
}

var re = regexp.MustCompile(`@(spec|implements|verifies)\s+([A-Za-z0-9_/.-]+)(?:\s+#([0-9a-fA-F]{6}))?`)

// Marker is one annotation occurrence.
// @spec markers/marker
type Marker struct {
	Kind  Kind
	ReqID string
	Hash  string // optional bound hash, "" if absent
	File  string
	Line  int
}

// ScanFile extracts every marker in a file.
// @spec markers/scan-file
func ScanFile(path string) ([]Marker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Scan(f, path)
}

func kindOf(s string) Kind {
	switch s {
	case "implements":
		return Implements
	case "verifies":
		return Verifies
	default:
		return Spec
	}
}
