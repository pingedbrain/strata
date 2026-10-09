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
	pyDef   = regexp.MustCompile(`^(?:async )?def (\w+)`)
	pyClass = regexp.MustCompile(`^class (\w+)`)
	pyTest  = regexp.MustCompile(`^\s*(?:async )?def (test_\w+)`)
	// pytest-bdd: @given("..."), @when(...), @then(parsers.parse("..."))
	pyStepDef = regexp.MustCompile(`^\s*@(?:given|when|then|step)\s*\(.*['"]([^'"]+)['"]`)
	// pytest-bdd: scenarios("features/login.feature") binds a whole feature
	pyScenarios = regexp.MustCompile(`\bscenarios\(\s*['"]([^'"]+\.feature)['"]`)
)

func (pyExtractor) match(p string) bool { return strings.HasSuffix(p, ".py") }

func (pyExtractor) isTestFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.Contains(p, "tests/")
}

func (pyExtractor) parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	testFile := pyExtractor{}.isTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
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
		if m := pyDef.FindStringSubmatch(text); m != nil {
			exports = append(exports, Symbol{Name: m[1], Kind: "func", File: filename, Line: line, Exported: !strings.HasPrefix(m[1], "_")})
		} else if m := pyClass.FindStringSubmatch(text); m != nil {
			exports = append(exports, Symbol{Name: m[1], Kind: "type", File: filename, Line: line, Exported: !strings.HasPrefix(m[1], "_")})
		}
	}
	return exports, tests, stepdefs
}
