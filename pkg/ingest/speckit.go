package ingest

import (
	"bufio"
	"bytes"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// speckitAdapter reads GitHub spec-kit repos: specs/<NNN-feature>/spec.md
// with functional requirements as "- **FR-001**: …" bullets.
// Requirement IDs: <feature-dir>/<FR-NNN>, e.g. "001-auth/FR-001".
// The FR text is stored as a single scenario so AcceptanceHash covers it
// (marker staleness works on spec-kit edits).
type speckitAdapter struct{}

func init() { Register(&speckitAdapter{}) }

var (
	reSpecKitFile = regexp.MustCompile(`^specs/[^/]+/spec\.md$`)
	reFR          = regexp.MustCompile(`(?m)^[-*]\s+\*\*([A-Z]+-\d+)\*\*:\s*(.+?)\s*$`)
)

func (*speckitAdapter) Name() string { return "spec-kit" }

func (a *speckitAdapter) files(dir fs.FS) []string {
	var out []string
	fs.WalkDir(dir, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && reSpecKitFile.MatchString(p) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func (a *speckitAdapter) Detect(dir fs.FS) bool {
	for _, f := range a.files(dir) {
		data, err := fs.ReadFile(dir, f)
		if err == nil && reFR.Match(data) {
			return true
		}
	}
	return false
}

func (a *speckitAdapter) Ingest(dir fs.FS) (*reqgraph.Graph, error) {
	g := reqgraph.New()
	for _, f := range a.files(dir) {
		data, err := fs.ReadFile(dir, f)
		if err != nil {
			return nil, err
		}
		cap := path.Base(path.Dir(f))
		line := 0
		s := bufio.NewScanner(bytes.NewReader(data))
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		for s.Scan() {
			line++
			m := reFR.FindStringSubmatch(s.Text())
			if m == nil {
				continue
			}
			text := strings.TrimSpace(m[2])
			err := g.Add(&reqgraph.Requirement{
				ID:     cap + "/" + m[1],
				Title:  m[1] + ": " + text,
				Source: f + ":" + strconv.Itoa(line),
				Scenarios: []reqgraph.Scenario{
					{Name: "spec", Steps: []string{text}},
				},
			})
			if err != nil {
				return nil, err
			}
		}
		if err := s.Err(); err != nil {
			return nil, err
		}
	}
	return g, nil
}
