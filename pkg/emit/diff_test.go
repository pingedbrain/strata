package emit

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestDiffMarkers(t *testing.T) {
	fsys := fstest.MapFS{
		"auth/validate.go": &fstest.MapFile{Data: []byte(`package auth

func Validate() {}
func Other() {}
func Third() {}
func Fourth() {}
func Fifth() {}
func Sixth() {}
func Seventh() {}
func Eighth() {}
`)},
	}
	sugs := []MarkerSuggestion{
		{File: "auth/validate.go", Line: 3, Comment: "// @spec auth/validate #abc123"},
		{File: "auth/validate.go", Line: 10, Comment: "// @spec auth/seventh #def456"},
	}
	out, err := DiffMarkers(fsys, sugs)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--- a/auth/validate.go",
		"+++ b/auth/validate.go",
		"@@",
		"+// @spec auth/validate #abc123",
		"+// @spec auth/seventh #def456",
		" func Validate() {}",
		" package auth",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("diff missing %q:\n%s", want, out)
		}
	}
	// insertions 7 lines apart exceed the context window → two hunks
	if strings.Count(out, "@@") != 4 {
		t.Fatalf("want 2 hunks (@@ appears twice per header):\n%s", out)
	}
}

func TestDiffMarkersMergesCloseInsertions(t *testing.T) {
	fsys := fstest.MapFS{
		"a.py": &fstest.MapFile{Data: []byte("def one():\n    pass\n\ndef two():\n    pass\n")},
	}
	sugs := []MarkerSuggestion{
		{File: "a.py", Line: 1, Comment: "# @spec x/one #111aaa"},
		{File: "a.py", Line: 4, Comment: "# @spec x/two #222bbb"},
	}
	out, err := DiffMarkers(fsys, sugs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "@@") != 2 {
		t.Fatalf("close insertions should merge into one hunk:\n%s", out)
	}
	if !strings.Contains(out, "+# @spec x/one #111aaa") ||
		!strings.Contains(out, "+# @spec x/two #222bbb") {
		t.Fatalf("inserted lines missing:\n%s", out)
	}
}
