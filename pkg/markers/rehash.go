package markers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Fix rewrites the bound hash of one marker occurrence.
// @spec markers/fix
type Fix struct {
	File    string
	Line    int
	ReqID   string
	NewHash string
}

// ApplyFixes rewrites marker hashes in place. Only the marker matching
// (file, line, reqID) is touched; other occurrences of the same ID are
// left alone.
// @spec markers/apply-fixes
func ApplyFixes(root string, fixes []Fix) error {
	byFile := map[string][]Fix{}
	var order []string
	for _, f := range fixes {
		if _, seen := byFile[f.File]; !seen {
			order = append(order, f.File)
		}
		byFile[f.File] = append(byFile[f.File], f)
	}
	for _, file := range order {
		p := filepath.Join(root, file)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		for _, f := range byFile[file] {
			idx := f.Line - 1
			if idx < 0 || idx >= len(lines) {
				return fmt.Errorf("%s:%d out of range", f.File, f.Line)
			}
			re := regexp.MustCompile(`(@(?:spec|implements|verifies)\s+` +
				regexp.QuoteMeta(f.ReqID) + `)(\s+#[0-9a-fA-F]{6})?`)
			lines[idx] = re.ReplaceAllString(lines[idx], "${1} #"+f.NewHash)
		}
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return err
		}
	}
	return nil
}
