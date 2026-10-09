package index

import (
	"regexp"
	"strings"
)

// BDD step patterns can carry parameters: pytest-bdd templates
// ("I have {n} cukes"), cucumber expressions, or regex literals
// ("^I have (\d+) cukes$"). BDDStepMatch reports whether a step-def
// pattern plausibly covers a scenario step: literal chunks must appear
// in order after normalization (alnum-only, lowercased).
func BDDStepMatch(pat, step string) bool {
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	chunks := bddParam.Split(pat, -1)
	s := norm(step)
	if len(chunks) == 1 {
		p := norm(pat)
		return p != "" && s != "" && (s == p || strings.Contains(s, p))
	}
	pos := 0
	for _, c := range chunks {
		c = norm(c)
		if c == "" {
			continue
		}
		i := strings.Index(s[pos:], c)
		if i < 0 {
			return false
		}
		pos += i + len(c)
	}
	return pos > 0 || norm(pat) != ""
}

// param placeholders inside step-def patterns
var bddParam = regexp.MustCompile(`\{[^}]*\}|\([^)]*\)|\\[dws]\+?|\.[*+]`)
