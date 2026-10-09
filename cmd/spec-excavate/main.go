// spec-excavate mines code→spec: proposes spec.md for repos that never
// had one, plus suggested @spec markers.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pingedbrain/strata/pkg/emit"
	"github.com/pingedbrain/strata/pkg/mine"
)

const usage = `spec-excavate — bootstrap specs for brownfield repos

usage: spec-excavate <command> [flags]

commands:
  scan              walk a repo and list candidate requirements
  propose           draft openspec-compatible spec.md (--write to save)
  suggest-markers   propose @spec annotations (--write to apply)

common flags:
  --root      repo root (default ".")
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "scan":
		err = runScan(os.Args[2:])
	case "propose":
		err = runPropose(os.Args[2:])
	case "suggest-markers":
		err = runSuggestMarkers(os.Args[2:])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func repo(args []string) (root string, caps []mine.Capability, write bool, err error) {
	fs := flag.NewFlagSet("spec-excavate", flag.ContinueOnError)
	rootF := fs.String("root", ".", "repo root")
	writeF := fs.Bool("write", false, "apply changes (propose/suggest-markers)")
	if err := fs.Parse(args); err != nil {
		return "", nil, false, err
	}
	caps, err = mine.Scan(os.DirFS(*rootF))
	return *rootF, caps, *writeF, err
}

func runScan(args []string) error {
	_, caps, _, err := repo(args)
	if err != nil {
		return err
	}
	total := 0
	for _, c := range caps {
		fmt.Printf("%s (%d candidates)\n", c.Name, len(c.Candidates))
		for _, cand := range c.Candidates {
			tests := ""
			if len(cand.Tests) > 0 {
				tests = fmt.Sprintf("  ← %d test(s)", len(cand.Tests))
			}
			fmt.Printf("  %-30s %-6s %s:%d%s\n", cand.Name, cand.Kind, cand.Evidence.File, cand.Evidence.Line, tests)
			total++
		}
	}
	fmt.Printf("\n%d candidates across %d capabilities\n", total, len(caps))
	return nil
}

func runPropose(args []string) error {
	root, caps, write, err := repo(args)
	if err != nil {
		return err
	}
	for _, c := range caps {
		spec := emit.Spec(c)
		if !write {
			fmt.Printf("=== openspec/specs/%s/spec.md ===\n%s\n", c.Name, spec)
			continue
		}
		p := filepath.Join(root, "openspec", "specs", c.Name, "spec.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(spec), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", p)
	}
	return nil
}

func runSuggestMarkers(args []string) error {
	root, caps, write, err := repo(args)
	if err != nil {
		return err
	}
	sugs := emit.SuggestMarkers(caps)
	if !write {
		for _, s := range sugs {
			fmt.Printf("%s:%d → insert: %s\n", s.File, s.Line, s.Comment)
		}
		return nil
	}
	// group by file, insert bottom-up so line numbers stay valid
	byFile := map[string][]emit.MarkerSuggestion{}
	var order []string
	for _, s := range sugs {
		if _, seen := byFile[s.File]; !seen {
			order = append(order, s.File)
		}
		byFile[s.File] = append(byFile[s.File], s)
	}
	for _, f := range order {
		p := filepath.Join(root, f)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		inserts := byFile[f]
		sort.Slice(inserts, func(i, j int) bool { return inserts[i].Line > inserts[j].Line })
		for _, s := range inserts {
			idx := s.Line - 1
			if idx < 0 || idx > len(lines) {
				continue
			}
			lines = append(lines[:idx], append([]string{s.Comment}, lines[idx:]...)...)
		}
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return err
		}
		fmt.Println("annotated", f)
	}
	return nil
}
