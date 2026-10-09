package mine

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"

	"github.com/pingedbrain/strata/pkg/index"
)

type tsExtractor struct{}

var (
	tsExport = regexp.MustCompile(`^export\s+(?:async\s+)?(?:default\s+)?(?:function|class|const|let|interface|type|enum)\s+(\w+)`)
	tsTest   = regexp.MustCompile(`^\s*(?:it|test|describe)\(\s*['"]([^'"]+)`)
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

func (tsExtractor) parse(data []byte, filename string) (exports, tests []index.Symbol) {
	testFile := tsExtractor{}.isTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if testFile {
			if m := tsTest.FindStringSubmatch(text); m != nil {
				tests = append(tests, index.Symbol{Name: m[1], Kind: "test", File: filename, Line: line})
			}
			continue
		}
		if m := tsExport.FindStringSubmatch(text); m != nil {
			exports = append(exports, index.Symbol{Name: m[1], Kind: "func", File: filename, Line: line})
		}
	}
	return exports, tests
}
