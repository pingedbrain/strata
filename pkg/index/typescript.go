package index

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"
)

type tsExtractor struct{}

var (
	tsExport = regexp.MustCompile(`^export\s+(?:async\s+)?(?:default\s+)?(?:function|class|const|let|interface|type|enum)\s+(\w+)`)
	tsTest   = regexp.MustCompile(`^\s*(?:it|test|describe)\(\s*['"]([^'"]+)`)
	// cucumber-js: Given("text", fn) or Given(/^regex$/, fn)
	tsStepDef = regexp.MustCompile(`(?:^|[^\w])(?:Given|When|Then|Step)\(\s*(?:['"]([^'"]+)['"]|/([^/]+)/)`)
)

func (tsExtractor) match(p string) bool {
	return strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx") ||
		strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".mjs") || strings.HasSuffix(p, ".jsx")
}

func (tsExtractor) isTestFile(p string) bool {
	base := path.Base(p)
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.Contains(p, "__tests__/")
}

func (tsExtractor) parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	testFile := tsExtractor{}.isTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if m := tsStepDef.FindStringSubmatch(text); m != nil {
			stepText := m[1]
			if stepText == "" {
				stepText = stripRegexAnchors(m[2]) // cucumber regex literal
			}
			if stepText != "" {
				stepdefs = append(stepdefs, Symbol{Name: stepText, Kind: "stepdef", File: filename, Line: line})
			}
		}
		if testFile {
			if m := tsTest.FindStringSubmatch(text); m != nil {
				tests = append(tests, Symbol{Name: m[1], Kind: "test", File: filename, Line: line})
			}
			continue
		}
		if m := tsExport.FindStringSubmatch(text); m != nil {
			exports = append(exports, Symbol{Name: m[1], Kind: "func", File: filename, Line: line, Exported: true})
		}
	}
	return exports, tests, stepdefs
}

// stripRegexAnchors turns a cucumber regex like "^I have (\\d+) cukes$"
// into matchable text "I have (\\d+) cukes" — good enough for the
// normalized containment check in index.Covers.
func stripRegexAnchors(s string) string {
	s = strings.TrimPrefix(s, "^")
	s = strings.TrimSuffix(s, "$")
	return s
}
