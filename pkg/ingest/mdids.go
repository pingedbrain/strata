package ingest

import (
	"bufio"
	"bytes"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// mdidsAdapter reads plain markdown specs where requirements carry
// explicit stable IDs in headings:
//
//	## REQ-042: Tokens expire
//	### [AUTH-7] Refresh tokens rotate
//
// IDs are taken verbatim — markers reference "REQ-042" directly.
// Two requirement headings in one file are the detection threshold.
type mdidsAdapter struct{}

func init() { Register(&mdidsAdapter{}) }

var reReqID = regexp.MustCompile(`^#{1,4}\s+(?:\[([A-Z][A-Z0-9_-]+)\]|([A-Z][A-Z0-9_-]+):)\s*(.+?)\s*$`)

func (*mdidsAdapter) Name() string { return "md-ids" }

func (a *mdidsAdapter) files(dir fs.FS) []string {
	var out []string
	fs.WalkDir(dir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		if strings.HasPrefix(p, "openspec/") || strings.HasPrefix(p, "specs/") {
			return nil // other adapters own those trees
		}
		out = append(out, p)
		return nil
	})
	return out
}

// parse finds ID-carrying headings in one file.
func parseMdIDs(g *reqgraph.Graph, file string, data []byte) (int, error) {
	count := 0
	var cur *reqgraph.Requirement
	line := 0
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		line++
		text := s.Text()
		m := reReqID.FindStringSubmatch(text)
		if m == nil {
			if cur != nil && strings.TrimSpace(text) != "" && !strings.HasPrefix(text, "#") {
				if cur.Text != "" {
					cur.Text += " "
				}
				cur.Text += strings.TrimSpace(text)
			}
			continue
		}
		id := m[1]
		if id == "" {
			id = m[2]
		}
		cur = &reqgraph.Requirement{
			ID:     id,
			Title:  m[3],
			Source: file + ":" + strconv.Itoa(line),
		}
		if err := g.Add(cur); err != nil {
			return count, err
		}
		count++
	}
	return count, s.Err()
}

func (a *mdidsAdapter) Detect(dir fs.FS) bool {
	for _, f := range a.files(dir) {
		data, err := fs.ReadFile(dir, f)
		if err != nil {
			continue
		}
		g := reqgraph.New()
		if n, err := parseMdIDs(g, f, data); err == nil && n >= 2 {
			return true
		}
	}
	return false
}

func (a *mdidsAdapter) Ingest(dir fs.FS) (*reqgraph.Graph, error) {
	g := reqgraph.New()
	for _, f := range a.files(dir) {
		data, err := fs.ReadFile(dir, f)
		if err != nil {
			return nil, err
		}
		if _, err := parseMdIDs(g, f, data); err != nil {
			return nil, err
		}
	}
	return g, nil
}
