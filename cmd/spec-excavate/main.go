// spec-excavate mines code→spec: proposes spec.md for repos that never
// had one, plus suggested spec markers.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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
  suggest-markers   propose spec annotations (--write to apply)
  history           git hot files + change coupling (--max-commits 500)

common flags:
  --root      repo root (default ".")

propose flags:
  --write         write spec.md files under openspec/specs/
  --enrich-cmd    external command that fills requirement descriptions:
                  prompt goes to stdin, one-sentence reply on stdout
                  (e.g. "claude -p", "gh models run", "ollama run llama3")
  --dump-prompts  print the enrich prompts instead of running a command
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
	case "history":
		err = runHistory(os.Args[2:])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type opts struct {
	root        string
	write       bool
	enrichCmd   string
	dumpPrompts bool
}

func repo(args []string) (opts, []mine.Capability, error) {
	fs := flag.NewFlagSet("spec-excavate", flag.ContinueOnError)
	o := opts{}
	fs.StringVar(&o.root, "root", ".", "repo root")
	fs.BoolVar(&o.write, "write", false, "apply changes (propose/suggest-markers)")
	fs.StringVar(&o.enrichCmd, "enrich-cmd", "", "external LLM command for descriptions")
	fs.BoolVar(&o.dumpPrompts, "dump-prompts", false, "print enrich prompts and exit")
	if err := fs.Parse(args); err != nil {
		return o, nil, err
	}
	caps, err := mine.Scan(os.DirFS(o.root))
	return o, caps, err
}

func runScan(args []string) error {
	_, caps, err := repo(args)
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
	o, caps, err := repo(args)
	if err != nil {
		return err
	}
	if o.dumpPrompts {
		for _, c := range caps {
			for i, cand := range c.Candidates {
				if cand.Doc != "" {
					continue // doc comment already fills the description
				}
				fmt.Printf("=== %s → %s ===\n%s\n\n", c.Name, cand.Name,
					emit.EnrichPrompt(c.Name, emit.ReqTitle(c, i), cand))
			}
		}
		return nil
	}
	var enrich func(string) string
	if o.enrichCmd != "" {
		enrich = func(prompt string) string {
			parts := strings.Fields(o.enrichCmd)
			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Dir = o.root
			cmd.Stdin = strings.NewReader(prompt)
			out, err := cmd.Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "enrich-cmd failed: %v\n", err)
				return ""
			}
			return strings.TrimSpace(string(out))
		}
	}
	for _, c := range caps {
		spec := emit.Spec(c, enrich)
		if !o.write {
			fmt.Printf("=== openspec/specs/%s/spec.md ===\n%s\n", c.Name, spec)
			continue
		}
		p := filepath.Join(o.root, "openspec", "specs", c.Name, "spec.md")
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
	o, caps, err := repo(args)
	if err != nil {
		return err
	}
	sugs := emit.SuggestMarkers(caps)
	if !o.write {
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
		p := filepath.Join(o.root, f)
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

func runHistory(args []string) error {
	fset := flag.NewFlagSet("history", flag.ContinueOnError)
	root := fset.String("root", ".", "repo root")
	max := fset.Int("max-commits", 500, "git log window")
	top := fset.Int("top", 15, "rows per table")
	if err := fset.Parse(args); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", *root, "log",
		fmt.Sprintf("--max-count=%d", *max), "--numstat", "--format=format:COMMIT")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	files, pairs := mine.ParseHistory(strings.NewReader(string(out)))
	hot := mine.HotFiles(files)
	fmt.Printf("hot files (by commits touched, last %d commits):\n", *max)
	for i, f := range hot {
		if i >= *top {
			break
		}
		st := files[f]
		fmt.Printf("  %4d  +%-5d -%-5d %s\n", st.Changes, st.Added, st.Deleted, f)
	}
	fmt.Println("\nchange coupling (files that change together):")
	shown := 0
	for _, p := range pairs {
		if shown >= *top {
			break
		}
		fmt.Printf("  %4d  %s ↔ %s\n", p.Together, p.A, p.B)
		shown++
	}
	if shown == 0 {
		fmt.Println("  (none — files change independently)")
	}
	return nil
}
