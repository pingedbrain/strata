// Package junit parses JUnit XML test reports — the lingua franca of CI
// test output (pytest, go-junit-report, jest-junit, surefire).
package junit

import (
	"encoding/xml"
	"strings"
)

// Case is one executed test.
// @spec junit/case
type Case struct {
	Name      string // test name, last segment only
	Classname string // suite/class it ran in (often a file or package)
	Failed    bool   // <failure> or <error> child present
	Skipped   bool
}

// Parse reads a JUnit XML document, handling both <testsuites> and bare
// <testsuite> roots.
// @spec junit/parse
func Parse(data []byte) ([]Case, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var out []Case
	inCase := false
	var cur *Case
	for {
		tok, err := dec.Token()
		if err != nil {
			break // io.EOF or malformed tail
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "testcase":
				cur = &Case{}
				inCase = true
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "name":
						cur.Name = a.Value
					case "classname":
						cur.Classname = a.Value
					}
				}
			case "failure", "error":
				if inCase {
					cur.Failed = true
				}
			case "skipped":
				if inCase {
					cur.Skipped = true
				}
			}
		case xml.EndElement:
			if t.Name.Local == "testcase" && cur != nil {
				name := cur.Name
				if i := strings.LastIndexByte(name, '.'); i >= 0 {
					name = name[i+1:] // junit often emits qualified names
				}
				cur.Name = name
				out = append(out, *cur)
				cur, inCase = nil, false
			}
		}
	}
	return out, nil
}
