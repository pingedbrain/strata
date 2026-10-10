package mine

import (
	"bufio"
	"io"
	"sort"
	"strconv"
	"strings"
)

// FileStats summarizes how a file evolves in git history.
// @spec mine/file-stats
type FileStats struct {
	Changes int // commits touching the file
	Added   int
	Deleted int
}

// Coupling is a pair of files that change together — a hint that they
// may belong to the same requirement boundary even across directories.
// @spec mine/coupling
type Coupling struct {
	A, B     string
	Together int
}

// ParseHistory reads `git log --numstat --format=COMMIT` output. COMMIT
// lines delimit commits; numstat lines are "added<TAB>deleted<TAB>path".
// Binary files show "-" counts and contribute changes but no lines.
// @spec mine/parse-history
func ParseHistory(r io.Reader) (files map[string]*FileStats, pairs []Coupling) {
	files = map[string]*FileStats{}
	together := map[[2]string]int{}
	var commitFiles []string
	flush := func() {
		for i := 0; i < len(commitFiles); i++ {
			for j := i + 1; j < len(commitFiles); j++ {
				a, b := commitFiles[i], commitFiles[j]
				if a > b {
					a, b = b, a
				}
				together[[2]string{a, b}]++
			}
		}
		commitFiles = commitFiles[:0]
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		t := sc.Text()
		if t == "COMMIT" {
			flush()
			continue
		}
		f := strings.SplitN(t, "\t", 3)
		if len(f) != 3 || f[2] == "" {
			continue
		}
		st := files[f[2]]
		if st == nil {
			st = &FileStats{}
			files[f[2]] = st
		}
		st.Changes++
		if n, err := strconv.Atoi(f[0]); err == nil {
			st.Added += n
		}
		if n, err := strconv.Atoi(f[1]); err == nil {
			st.Deleted += n
		}
		commitFiles = append(commitFiles, f[2])
	}
	flush()
	for k, n := range together {
		if n >= 2 {
			pairs = append(pairs, Coupling{A: k[0], B: k[1], Together: n})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Together > pairs[j].Together })
	return files, pairs
}

// HotFiles returns paths sorted by change count, descending.
// @spec mine/hot-files
func HotFiles(files map[string]*FileStats) []string {
	out := make([]string, 0, len(files))
	for f := range files {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return files[out[i]].Changes > files[out[j]].Changes })
	return out
}
