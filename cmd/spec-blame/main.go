// spec-blame verifies spec→code: coverage, drift, blame.
package main

import (
	"fmt"
	"os"
)

const usage = `spec-blame — traceability gate for spec-driven development

usage: spec-blame <command> [flags]

commands:
  check     CI gate: dangling refs, stale markers, unknown IDs
  coverage  spec coverage report
  blame     annotate a file with the requirements that justify it
  map       bidirectional req↔code lookup
  serve     MCP server over the requirement graph
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "check", "coverage", "blame", "map", "serve":
		fmt.Fprintf(os.Stderr, "spec-blame %s: not implemented yet\n", os.Args[1])
		os.Exit(1)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}
