// spec-excavate mines code→spec: proposes spec.md for repos that never
// had one, plus suggested @spec markers.
package main

import (
	"fmt"
	"os"
)

const usage = `spec-excavate — bootstrap specs for brownfield repos

usage: spec-excavate <command> [flags]

commands:
  scan              walk a repo and list candidate requirements
  propose           draft openspec-compatible spec.md
  suggest-markers   propose @spec annotations as a reviewable diff
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "scan", "propose", "suggest-markers":
		fmt.Fprintf(os.Stderr, "spec-excavate %s: not implemented yet\n", os.Args[1])
		os.Exit(1)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}
