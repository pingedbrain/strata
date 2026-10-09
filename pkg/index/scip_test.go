package index

import (
	"testing"
)

// minimal protobuf wire encoder for test fixtures
func vint(v uint64) []byte {
	var b []byte
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}
func key(f, wt int) []byte { return vint(uint64(f<<3 | wt)) }
func vf(f int, v uint64) []byte {
	return append(key(f, 0), vint(v)...)
}
func bf(f int, data []byte) []byte {
	out := append(key(f, 2), vint(uint64(len(data)))...)
	return append(out, data...)
}
func sf(f int, s string) []byte { return bf(f, []byte(s)) }

func scipOcc(rng []int32, sym string, roles int) []byte {
	var occ []byte
	var packed []byte
	for _, r := range rng {
		packed = append(packed, vint(uint64(r))...)
	}
	occ = append(occ, bf(1, packed)...)
	occ = append(occ, sf(2, sym)...)
	occ = append(occ, vf(3, uint64(roles))...)
	return occ
}

func scipDoc(path, lang string, occs ...[]byte) []byte {
	var doc []byte
	doc = append(doc, sf(1, path)...)
	for _, o := range occs {
		doc = append(doc, bf(2, o)...)
	}
	doc = append(doc, sf(4, lang)...)
	return doc
}

func scipIndex(docs ...[]byte) []byte {
	var idx []byte
	for _, d := range docs {
		idx = append(idx, bf(2, d)...)
	}
	return idx
}

func TestLoadSCIP(t *testing.T) {
	data := scipIndex(scipDoc("x.go", "go",
		// definition: func Validate at line 2 (0-based 1)
		scipOcc([]int32{1, 0, 7}, "go . example.com/x v1 auth/Validate().", 0x1),
		// definition: method Client.Check at line 5
		scipOcc([]int32{4, 0, 5}, "go . example.com/x v1 auth/Client#Check().", 0x1),
		// unexported func
		scipOcc([]int32{7, 0, 6}, "go . example.com/x v1 auth/helper().", 0x1),
		// local symbol → skipped
		scipOcc([]int32{9, 0, 4}, "local 0", 0x1),
		// reference, not definition (roles=ReadAccess) → skipped
		scipOcc([]int32{1, 4, 12}, "go . example.com/x v1 auth/Validate().", 0x8),
		// test role
		scipOcc([]int32{11, 0, 12}, "go . example.com/x v1 auth/TestValidate().", 0x1|0x20),
		// struct field (term under type, exported) → skipped, not a declaration
		scipOcc([]int32{13, 0, 5}, "go . example.com/x v1 auth/Client#Host.", 0x1),
	), scipDoc("x_test.go", "go",
		// test func in _test.go file → Tests bucket even without Test role
		scipOcc([]int32{2, 0, 8}, "go . example.com/x v1 auth/TestOther().", 0x1),
	))
	idx, err := LoadSCIP(data)
	if err != nil {
		t.Fatal(err)
	}
	syms := idx.Files["x.go"]
	if len(syms) != 3 {
		t.Fatalf("want 3 defs, got %+v", syms)
	}
	byName := map[string]Symbol{}
	for _, s := range syms {
		byName[s.Name] = s
	}
	if v := byName["Validate"]; v.Line != 2 || v.Kind != "func" || !v.Exported {
		t.Fatalf("Validate: %+v", v)
	}
	if m := byName["Client.Check"]; m.Kind != "method" || !m.Exported {
		t.Fatalf("Client.Check: %+v", m)
	}
	if h := byName["helper"]; h.Exported {
		t.Fatal("helper should be unexported (go + lowercase)")
	}
	if _, isField := byName["Host"]; isField {
		t.Fatal("struct fields must not become symbols")
	}
	if len(idx.Tests) != 2 {
		t.Fatalf("tests: %+v", idx.Tests)
	}
}

func TestSCIPSymbolName(t *testing.T) {
	for in, want := range map[string]string{
		"go . example.com/x v1 auth/Validate().":     "Validate",
		"go . example.com/x v1 auth/Client#Check().": "Client.Check",
		"python . pkg v0 lib/User#":                  "User",
		"rust-analyzer cargo crate v1 module/term.":  "term",
		"local 42":                          "",
		"ts . mypkg v1 `weird name`/run().": "run",
	} {
		got, _ := scipSymbolName(in)
		if got != want {
			t.Errorf("%q → %q, want %q", in, got, want)
		}
	}
}

func TestOverlay(t *testing.T) {
	base := &Index{Files: map[string][]Symbol{
		"a.go": {{Name: "regexFunc"}},
		"b.go": {{Name: "keepMe"}},
	}}
	scip := &Index{Files: map[string][]Symbol{
		"a.go": {{Name: "scipFunc"}},
	}}
	base.Overlay(scip)
	if base.Files["a.go"][0].Name != "scipFunc" {
		t.Fatal("scip should win for covered files")
	}
	if base.Files["b.go"][0].Name != "keepMe" {
		t.Fatal("uncovered files untouched")
	}
}
