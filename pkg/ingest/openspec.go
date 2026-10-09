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

// openspecAdapter reads openspec/specs/<capability>/*.md.
// Requirements are "### Requirement: <name>"; scenarios are
// "#### Scenario: <name>" followed by WHEN/THEN/AND bullets.
// Requirement IDs are derived deterministically: <capability>/<slug>.
type openspecAdapter struct{}

func init() { Register(&openspecAdapter{}) }

var (
	reReq      = regexp.MustCompile(`^###\s+Requirement:\s*(.+?)\s*$`)
	reScenario = regexp.MustCompile(`^####\s+Scenario:\s*(.+?)\s*$`)
	reStep     = regexp.MustCompile(`^[-*]\s+(?:\*\*)?(?:WHEN|THEN|AND|GIVEN)(?:\*\*)?\s+(?::\s*)?(.+?)\s*$`)
)

func (*openspecAdapter) Name() string { return "openspec" }

func (*openspecAdapter) Detect(dir fs.FS) bool {
	m, err := fs.Glob(dir, "openspec/specs/*/*.md")
	return err == nil && len(m) > 0
}

func (*openspecAdapter) Ingest(dir fs.FS) (*reqgraph.Graph, error) {
	files, err := fs.Glob(dir, "openspec/specs/*/*.md")
	if err != nil {
		return nil, err
	}
	g := reqgraph.New()
	for _, f := range files {
		cap := path.Base(path.Dir(f))
		data, err := fs.ReadFile(dir, f)
		if err != nil {
			return nil, err
		}
		if err := parseSpecMD(g, cap, f, data); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func parseSpecMD(g *reqgraph.Graph, cap, file string, data []byte) error {
	var cur *reqgraph.Requirement
	var sc *reqgraph.Scenario
	line := 0

	flush := func() {
		if cur != nil && sc != nil {
			cur.Scenarios = append(cur.Scenarios, *sc)
		}
		sc = nil
	}

	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		line++
		text := s.Text()
		if m := reReq.FindStringSubmatch(text); m != nil {
			flush()
			cur = &reqgraph.Requirement{
				ID:     cap + "/" + reqgraph.Slug(m[1]),
				Title:  m[1],
				Source: file + ":" + strconv.Itoa(line),
			}
			if err := g.Add(cur); err != nil {
				return err
			}
			continue
		}
		if m := reScenario.FindStringSubmatch(text); m != nil {
			flush()
			if cur != nil {
				sc = &reqgraph.Scenario{Name: m[1]}
			}
			continue
		}
		if m := reStep.FindStringSubmatch(text); m != nil && sc != nil {
			sc.Steps = append(sc.Steps, m[1])
			continue
		}
		if cur != nil && sc == nil && strings.TrimSpace(text) != "" {
			if cur.Text != "" {
				cur.Text += " "
			}
			cur.Text += strings.TrimSpace(text)
		}
	}
	flush()
	return s.Err()
}
