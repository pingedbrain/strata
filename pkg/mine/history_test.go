package mine

import (
	"strings"
	"testing"
)

func TestParseHistory(t *testing.T) {
	log := `COMMIT
3	1	pkg/graph/graph.go
2	0	pkg/graph/verify.go
-	-	blob.bin

COMMIT
1	0	pkg/graph/graph.go
5	2	pkg/check/check.go

COMMIT
0	1	pkg/graph/graph.go
1	0	pkg/graph/verify.go

`
	files, pairs := ParseHistory(strings.NewReader(log))
	if got := files["pkg/graph/graph.go"].Changes; got != 3 {
		t.Fatalf("graph.go changes: %d", got)
	}
	if got := files["pkg/graph/graph.go"].Added; got != 4 {
		t.Fatalf("graph.go added: %d", got)
	}
	if files["blob.bin"].Added != 0 {
		t.Fatal("binary file should contribute changes but no line counts")
	}
	// graph.go↔verify.go co-change twice; graph.go↔check.go once (below min)
	var top Coupling
	for _, p := range pairs {
		if p.A == "pkg/graph/graph.go" && p.B == "pkg/graph/verify.go" {
			top = p
		}
		if p.B == "pkg/check/check.go" {
			t.Fatalf("single co-change should be filtered: %+v", p)
		}
	}
	if top.Together != 2 {
		t.Fatalf("want coupling 2, got %+v (all: %+v)", top, pairs)
	}
	if HotFiles(files)[0] != "pkg/graph/graph.go" {
		t.Fatal("hot file ordering wrong")
	}
}
