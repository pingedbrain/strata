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

// gherkinAdapter reads .feature files: each Scenario becomes a
// requirement (scenarios ARE the spec in BDD), its steps become the
// acceptance criteria. IDs: <feature-file-slug>/<scenario-slug>.
type gherkinAdapter struct{}

func init() { Register(&gherkinAdapter{}) }

var (
	reFeature         = regexp.MustCompile(`^\s*Feature:\s*(.+?)\s*$`)
	reGherkinScenario = regexp.MustCompile(`^\s*Scenario(?:\s+Outline| Template)?:\s*(.+?)\s*$`)
	reStepG           = regexp.MustCompile(`^\s*(?:Given|When|Then|And|But|\*)\s+(.+?)\s*$`)
)

func (*gherkinAdapter) Name() string { return "gherkin" }

func (a *gherkinAdapter) files(dir fs.FS) []string {
	var out []string
	fs.WalkDir(dir, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".feature") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func (a *gherkinAdapter) Detect(dir fs.FS) bool { return len(a.files(dir)) > 0 }

func (a *gherkinAdapter) Ingest(dir fs.FS) (*reqgraph.Graph, error) {
	g := reqgraph.New()
	for _, f := range a.files(dir) {
		data, err := fs.ReadFile(dir, f)
		if err != nil {
			return nil, err
		}
		if err := parseFeature(g, f, data); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func parseFeature(g *reqgraph.Graph, file string, data []byte) error {
	cap := reqgraph.Slug(strings.TrimSuffix(path.Base(file), ".feature"))
	var featTitle string
	var cur *reqgraph.Requirement
	line := 0
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		line++
		text := s.Text()
		if m := reFeature.FindStringSubmatch(text); m != nil {
			featTitle = m[1]
			continue
		}
		if m := reGherkinScenario.FindStringSubmatch(text); m != nil {
			cur = &reqgraph.Requirement{
				ID:     cap + "/" + reqgraph.Slug(m[1]),
				Title:  m[1],
				Source: file + ":" + strconv.Itoa(line),
			}
			if featTitle != "" {
				cur.Text = "Feature: " + featTitle
			}
			cur.Scenarios = append(cur.Scenarios, reqgraph.Scenario{Name: m[1]})
			if err := g.Add(cur); err != nil {
				return err
			}
			continue
		}
		if m := reStepG.FindStringSubmatch(text); m != nil && cur != nil {
			last := &cur.Scenarios[len(cur.Scenarios)-1]
			last.Steps = append(last.Steps, m[1])
		}
	}
	return s.Err()
}
