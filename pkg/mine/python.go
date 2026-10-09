package mine

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"

	"github.com/pingedbrain/strata/pkg/index"
)

type pyExtractor struct{}

var (
	pyDef   = regexp.MustCompile(`^(?:async )?def (\w+)`)
	pyClass = regexp.MustCompile(`^class (\w+)`)
	pyTest  = regexp.MustCompile(`^\s*(?:async )?def (test_\w+)`)
)

func (pyExtractor) match(p string) bool { return strings.HasSuffix(p, ".py") }

func (pyExtractor) isTestFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.Contains(p, "tests/")
}

func (pyExtractor) parse(data []byte, filename string) (exports, tests []index.Symbol) {
	testFile := pyExtractor{}.isTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if testFile {
			if m := pyTest.FindStringSubmatch(text); m != nil {
				tests = append(tests, index.Symbol{Name: m[1], Kind: "test", File: filename, Line: line})
			}
			continue
		}
		if m := pyDef.FindStringSubmatch(text); m != nil && !strings.HasPrefix(m[1], "_") {
			exports = append(exports, index.Symbol{Name: m[1], Kind: "func", File: filename, Line: line})
		} else if m := pyClass.FindStringSubmatch(text); m != nil && !strings.HasPrefix(m[1], "_") {
			exports = append(exports, index.Symbol{Name: m[1], Kind: "type", File: filename, Line: line})
		}
	}
	return exports, tests
}
