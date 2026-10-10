package index

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"
)

type pyExtractor struct{}

var (
	pyDef    = regexp.MustCompile(`^(?:async )?def (\w+)`)
	pyClass  = regexp.MustCompile(`^class (\w+)`)
	pyTest   = regexp.MustCompile(`^\s*(?:async )?def (test_\w+)`)
	pyDocStr = regexp.MustCompile(`^\s*(?:"""(.*?)"""|'''(.*?)'''|"(.*?)"|'(.*?)')`)
	pyCmt    = regexp.MustCompile(`^\s*#\s?(.*)`)
	// pytest-bdd: @given("..."), @when(...), @then(parsers.parse("..."))
	pyStepDef = regexp.MustCompile(`^\s*@(?:given|when|then|step)\s*\(.*['"]([^'"]+)['"]`)
	// pytest-bdd: scenarios("features/login.feature") binds a whole feature
	pyScenarios = regexp.MustCompile(`\bscenarios\(\s*['"]([^'"]+\.feature)['"]`)
)

func (pyExtractor) Match(p string) bool { return strings.HasSuffix(p, ".py") }

func (pyExtractor) IsTestFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.Contains(p, "tests/")
}

func (pyExtractor) Parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	testFile := pyExtractor{}.IsTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	var cmtBuf []string // consecutive comment lines directly above a decl
	awaitDoc := -1      // index into exports of a decl awaiting its docstring
	for sc.Scan() {
		line++
		text := sc.Text()
		if m := pyStepDef.FindStringSubmatch(text); m != nil {
			stepdefs = append(stepdefs, Symbol{Name: m[1], Kind: "stepdef", File: filename, Line: line})
		}
		if m := pyScenarios.FindStringSubmatch(text); m != nil {
			stepdefs = append(stepdefs, Symbol{Name: m[1], Kind: "scenarios-ref", File: filename, Line: line})
		}
		if testFile {
			if m := pyTest.FindStringSubmatch(text); m != nil {
				tests = append(tests, Symbol{Name: m[1], Kind: "test", File: filename, Line: line})
			}
			continue
		}
		// docstring on the line(s) right after a def/class
		if awaitDoc >= 0 {
			if m := pyDocStr.FindStringSubmatch(text); m != nil {
				for _, g := range m[1:] {
					if g != "" {
						exports[awaitDoc].Doc = joinDoc(cmtBuf, g)
						break
					}
				}
				awaitDoc = -1
				cmtBuf = cmtBuf[:0]
				continue
			}
			if strings.TrimSpace(text) == "" || strings.HasPrefix(text, " ") || strings.HasPrefix(text, "\t") {
				continue // blank or body line — docstring may still come
			}
			awaitDoc = -1
		}
		if m := pyDef.FindStringSubmatch(text); m != nil {
			exports = append(exports, Symbol{Name: m[1], Kind: "func", File: filename, Line: line, Exported: !strings.HasPrefix(m[1], "_"), Doc: joinDoc(cmtBuf, "")})
			cmtBuf = cmtBuf[:0]
			awaitDoc = len(exports) - 1
		} else if m := pyClass.FindStringSubmatch(text); m != nil {
			exports = append(exports, Symbol{Name: m[1], Kind: "type", File: filename, Line: line, Exported: !strings.HasPrefix(m[1], "_"), Doc: joinDoc(cmtBuf, "")})
			cmtBuf = cmtBuf[:0]
			awaitDoc = len(exports) - 1
		} else if m := pyCmt.FindStringSubmatch(text); m != nil {
			cmtBuf = append(cmtBuf, m[1])
		} else if strings.TrimSpace(text) != "" {
			cmtBuf = cmtBuf[:0] // non-comment line breaks the comment block
		}
	}
	return exports, tests, stepdefs
}

// joinDoc merges leading comment lines with a docstring line.
func joinDoc(comments []string, docstr string) string {
	parts := append([]string{}, comments...)
	if docstr != "" {
		parts = append(parts, strings.TrimSpace(docstr))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
