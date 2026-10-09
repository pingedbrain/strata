package markers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRehashUpdatesOnlyTargetLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.go")
	src := `// @spec auth/login #000000
func a() {}

// @spec auth/login #111111
func b() {}
`
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	fixes := []Fix{{File: "a.go", Line: 4, ReqID: "auth/login", NewHash: "abc123"}}
	if err := ApplyFixes(dir, fixes); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	want := `// @spec auth/login #000000
func a() {}

// @spec auth/login #abc123
func b() {}
`
	if string(got) != want {
		t.Fatalf("got:\n%s", got)
	}
}
