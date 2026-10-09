package markers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFileAllCommentStyles(t *testing.T) {
	src := `// @spec AUTH-01 #a3f2b1
# @implements AUTH-02
/* @verifies AUTH-03 */
-- @spec DB-09
def f():  # @spec PY-1
`
	p := filepath.Join(t.TempDir(), "mixed.src")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ms, err := ScanFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 5 {
		t.Fatalf("want 5 markers, got %d: %+v", len(ms), ms)
	}
	if ms[0].Hash != "a3f2b1" {
		t.Fatalf("hash not captured: %+v", ms[0])
	}
	if ms[1].Kind != Implements || ms[2].Kind != Verifies {
		t.Fatalf("kinds wrong: %+v", ms)
	}
	if ms[3].ReqID != "DB-09" || ms[3].Line != 4 {
		t.Fatalf("sql-style comment marker missed: %+v", ms[3])
	}
}
