// spec-blame verifies spec→code: coverage, drift, blame.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pingedbrain/strata/pkg/check"
	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/mcp"
)

const usage = `spec-blame — traceability gate for spec-driven development

usage: spec-blame <command> [flags]

commands:
  check     CI gate: dangling refs, stale markers, unknown IDs
  coverage  spec coverage report
  blame     annotate a file with the requirements that justify it
  map       bidirectional req↔code lookup
  sync      rewrite stale bound hashes after reviewed spec edits
  serve     MCP server over the requirement graph

common flags:
  --root            repo root (default ".")
  --min-coverage    fail check below this fraction, e.g. 0.8 (default 0 = advisory)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "check":
		err = runCheck(os.Args[2:])
	case "coverage":
		err = runCoverage(os.Args[2:])
	case "blame":
		err = runBlame(os.Args[2:])
	case "map":
		err = runMap(os.Args[2:])
	case "sync":
		err = runSync(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadRepo(args []string) (*check.Report, string, error) {
	fs := flag.NewFlagSet("spec-blame", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	minCov := fs.Float64("min-coverage", 0, "minimum coverage fraction")
	if err := fs.Parse(args); err != nil {
		return nil, "", err
	}
	rep, err := check.Run(os.DirFS(*root), check.Options{MinCoverage: *minCov})
	return rep, *root, err
}

func runCheck(args []string) error {
	rep, _, err := loadRepo(args)
	if err != nil {
		return err
	}
	fmt.Printf("%s, %d requirements, %d links\n",
		fmt.Sprintf("specs: %v", rep.Formats), len(rep.Graph.IDs()), len(rep.Result.Edges))
	fails := rep.GateFailures()
	for _, f := range fails {
		fmt.Println("✗", f)
	}
	cov, tot := rep.Result.Coverage()
	fmt.Printf("coverage: %d/%d requirements linked\n", cov, tot)
	if len(fails) > 0 {
		return fmt.Errorf("gate failed")
	}
	return nil
}

func runCoverage(args []string) error {
	rep, _, err := loadRepo(args)
	if err != nil {
		return err
	}
	linked := map[string][]markers.Marker{}
	for _, e := range rep.Result.Edges {
		if e.Kind != markers.Verifies {
			linked[e.ReqID] = append(linked[e.ReqID], e.Marker)
		}
	}
	for _, id := range rep.Graph.IDs() {
		r := rep.Graph.Requirements[id]
		if ms := linked[id]; len(ms) > 0 {
			fmt.Printf("✓ %-32s %d link(s): %s\n", id, len(ms), ms[0].File)
		} else {
			fmt.Printf("✗ %-32s uncovered — %s\n", id, r.Title)
		}
	}
	cov, tot := rep.Result.Coverage()
	fmt.Printf("\ncoverage: %d/%d requirements linked\n", cov, tot)
	return nil
}

func runBlame(args []string) error {
	fs := flag.NewFlagSet("blame", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: spec-blame blame [--root dir] <file>")
	}
	rep, err := check.Run(os.DirFS(*root), check.Options{})
	if err != nil {
		return err
	}
	file := fs.Arg(0)
	found := false
	for _, e := range rep.Result.EdgesIn(file) {
		found = true
		r := rep.Graph.Requirements[e.ReqID]
		status := "ok"
		if e.Stale {
			status = "stale"
		}
		fmt.Printf("%-8s %s:%d → %s (%s)\n         %s\n", status, e.File, e.Line, e.ReqID, e.Kind, r.Title)
	}
	for _, m := range rep.Result.DanglingIn(file) {
		found = true
		fmt.Printf("%-8s %s:%d → %s (%s)\n         no such requirement\n", "dangling", m.File, m.Line, m.ReqID, m.Kind)
	}
	if !found {
		fmt.Println("no markers in", file)
	}
	return nil
}

func runMap(args []string) error {
	fs := flag.NewFlagSet("map", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: spec-blame map [--root dir] <req-id|file>")
	}
	rep, err := check.Run(os.DirFS(*root), check.Options{})
	if err != nil {
		return err
	}
	arg := fs.Arg(0)
	if r, ok := rep.Graph.Requirements[arg]; ok {
		fmt.Printf("%s — %s\n  %s\n", r.ID, r.Title, r.Source)
		for _, e := range rep.Result.EdgesFor(arg) {
			stale := ""
			if e.Stale {
				stale = "  [stale]"
			}
			fmt.Printf("  %s:%d (%s)%s\n", e.File, e.Line, e.Kind, stale)
		}
		return nil
	}
	// file → requirements
	edges := rep.Result.EdgesIn(arg)
	if len(edges) == 0 && len(rep.Result.DanglingIn(arg)) == 0 {
		return fmt.Errorf("no requirement or file matches %q", arg)
	}
	fmt.Println(arg, "implements:")
	for _, e := range edges {
		r := rep.Graph.Requirements[e.ReqID]
		fmt.Printf("  %s — %s\n", e.ReqID, r.Title)
	}
	for _, m := range rep.Result.DanglingIn(arg) {
		fmt.Printf("  %s (dangling)\n", m.ReqID)
	}
	return nil
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rep, err := check.Run(os.DirFS(*root), check.Options{})
	if err != nil {
		return err
	}
	fixes := rep.StaleFixes()
	if len(fixes) == 0 {
		fmt.Println("no stale markers")
		return nil
	}
	if err := markers.ApplyFixes(*root, fixes); err != nil {
		return err
	}
	for _, f := range fixes {
		fmt.Printf("synced %s:%d → %s #%s\n", f.File, f.Line, f.ReqID, f.NewHash)
	}
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	srv := &mcp.Server{Root: os.DirFS(*root)}
	return srv.Serve(os.Stdin, os.Stdout)
}
