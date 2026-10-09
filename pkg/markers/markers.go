// Package markers scans source files for requirement annotations:
//
//	@spec REQ-04 #a3f2b1          (any comment style)
//	@implements AUTH-01
//	@verifies AUTH-01
//
// The scan is a structured grep — no language parsing. Comment syntax is
// irrelevant because the marker itself is the same in every language.
package markers

import (
	"bufio"
	"os"
	"regexp"
)

// Kind distinguishes what a marker claims.
type Kind int

const (
	Spec       Kind = iota // @spec — this code implements the requirement
	Implements             // @implements — explicit impl claim
	Verifies               // @verifies — this test verifies it
)

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
type Marker struct {
	Kind  Kind
	ReqID string
	Hash  string // optional bound hash, "" if absent
	File  string
	Line  int
}

// ScanFile extracts every marker in a file.
func ScanFile(path string) ([]Marker, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Marker
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		for _, m := range re.FindAllStringSubmatch(sc.Text(), -1) {
			out = append(out, Marker{
				Kind:  kindOf(m[1]),
				ReqID: m[2],
				Hash:  m[3],
				File:  path,
				Line:  line,
			})
		}
	}
	return out, sc.Err()
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
