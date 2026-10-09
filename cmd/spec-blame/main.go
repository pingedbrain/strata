// spec-blame verifies spec→code: coverage, drift, blame.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pingedbrain/strata/pkg/check"
	"github.com/pingedbrain/strata/pkg/markers"
)

const usage = `spec-blame — traceability gate for spec-driven development

usage: spec-blame <command> [flags]

commands:
  check     CI gate: dangling refs, stale markers, unknown IDs
  coverage  spec coverage report
  blame     annotate a file with the requirements that justify it
  map       bidirectional req↔code lookup
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
	case "blame", "map", "serve":
		err = fmt.Errorf("spec-blame %s: not implemented yet", os.Args[1])
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
