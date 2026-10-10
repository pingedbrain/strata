package index

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// genericExtractor is the last-resort symbol extractor for C-family and
// adjacent languages without a dedicated indexer or parser: Rust, C/C++,
// Java, C#, Kotlin, Swift, Ruby, PHP, Scala, Objective-C, Lua… It trades
// precision for coverage — markers still bind, but expect noise the
// dedicated extractors (and SCIP) avoid. A tree-sitter WASM plugin can
// replace it per-language via RegisterExtractor ordering.
type genericExtractor struct{}

var genericExts = map[string]bool{
	".rs": true, ".c": true, ".h": true, ".cc": true, ".cpp": true,
	".cxx": true, ".hpp": true, ".hh": true, ".cs": true, ".java": true,
	".kt": true, ".kts": true, ".swift": true, ".rb": true, ".php": true,
	".scala": true, ".m": true, ".mm": true, ".lua": true, ".pl": true,
	".groovy": true, ".dart": true, ".ex": true, ".exs": true,
}

var (
	// type-ish declarations: `pub struct Foo`, `public class Bar`,
	// `interface Baz`, `enum E`, `trait T`, `impl X`…
	genType = regexp.MustCompile(`^\s*(?:pub(?:lic)?(?:\s*\([^)]*\))?\s+|priv(?:ate)?\s+|prot(?:ected)?\s+|internal\s+|static\s+|final\s+|abstract\s+|sealed\s+|open\s+|export\s+|extern\s+)*(?:class|interface|enum|struct|record|trait|type|object|module|namespace|union)\s+([A-Za-z_]\w*)`)
	// named-function keywords: fn (rust), func, fun (kotlin), def
	// (ruby/python-ish), function (php/lua), sub (perl)
	genFunc = regexp.MustCompile(`^\s*(?:pub(?:lic)?(?:\s*\([^)]*\))?\s+|priv(?:ate)?\s+|prot(?:ected)?\s+|internal\s+|static\s+|final\s+|abstract\s+|extern\s+|inline\s+|async\s+|suspend\s+|virtual\s+|override\s+|export\s+)*(?:fn|func|fun|def|function|sub)\s+([A-Za-z_]\w*)`)
	// return-type style: `int add(`, `public void run(`, `const char* name(`
	genCFunc = regexp.MustCompile(`^\s*(?:pub(?:lic)?(?:\s*\([^)]*\))?\s+|priv(?:ate)?\s+|prot(?:ected)?\s+|internal\s+|static\s+|final\s+|abstract\s+|extern\s+|inline\s+|virtual\s+|override\s+|async\s+|suspend\s+)*[\w:*&<>\[\],]+[\s*&]+([A-Za-z_]\w*)\s*\(`)
	// never a declaration
	genNotDecl = map[string]bool{
		"if": true, "for": true, "while": true, "switch": true,
		"catch": true, "return": true, "sizeof": true, "do": true,
		"else": true, "case": true, "new": true, "throw": true,
	}
	// obvious visibility keywords
	genPublic = regexp.MustCompile(`^\s*(?:pub\b|public\b|export\b|def\b)`)
)

func (genericExtractor) Match(p string) bool {
	return genericExts[strings.ToLower(path.Ext(p))]
}

func (genericExtractor) IsTestFile(p string) bool {
	base := strings.ToLower(path.Base(p))
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test"+path.Ext(base)) ||
		strings.Contains(base, ".test.") || strings.Contains(base, "_spec.") ||
		strings.Contains(p, "tests/") || strings.Contains(p, "spec/")
}

func (genericExtractor) Parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	testFile := genericExtractor{}.IsTestFile(filename)
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	seen := map[string]int{} // name → count, dedup redeclarations (overloads)
	for sc.Scan() {
		line++
		text := sc.Text()
		var name, kind string
		if m := genType.FindStringSubmatch(text); m != nil {
			name, kind = m[1], "type"
		} else if m := genFunc.FindStringSubmatch(text); m != nil {
			name, kind = m[1], "func"
		} else if m := genCFunc.FindStringSubmatch(text); m != nil {
			name, kind = m[1], "func"
		}
		if name == "" || genNotDecl[name] {
			continue
		}
		seen[name]++
		if seen[name] > 1 {
			continue // overloads/declarations — first sighting wins
		}
		r, _ := utf8.DecodeRuneInString(name)
		exported := genPublic.MatchString(text) || unicode.IsUpper(r)
		s := Symbol{Name: name, Kind: kind, File: filename, Line: line, Exported: exported}
		if testFile {
			if strings.HasPrefix(strings.ToLower(name), "test") {
				tests = append(tests, Symbol{Name: name, Kind: "test", File: filename, Line: line})
			}
			continue
		}
		exports = append(exports, s)
	}
	return exports, tests, stepdefs
}
