package index

import (
	"testing"
	"testing/fstest"
)

func TestGenericExtractor(t *testing.T) {
	fsys := fstest.MapFS{
		"lib.rs": &fstest.MapFile{Data: []byte(`pub fn connect(host: &str) -> Conn {
    if let Ok(x) = try_it() {
        return x;
    }
}

fn helper() {}

pub struct Session {
    id: u64,
}
`)},
		"math.c": &fstest.MapFile{Data: []byte(`int add(int a, int b) {
    return a + b;
}

static int clamp(int v) {
    return v;
}
`)},
		"test_check.rb": &fstest.MapFile{Data: []byte(`def test_add_works
end
`)},
	}
	idx, err := Scan(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	rs := idx.Files["lib.rs"]
	var connect, helper, session *Symbol
	for i := range rs {
		switch rs[i].Name {
		case "connect":
			connect = &rs[i]
		case "helper":
			helper = &rs[i]
		case "Session":
			session = &rs[i]
		}
	}
	if connect == nil || !connect.Exported {
		t.Fatalf("connect should be exported fn: %+v", rs)
	}
	if helper == nil || helper.Exported {
		t.Fatalf("helper should be private fn: %+v", rs)
	}
	if session == nil || session.Kind != "type" {
		t.Fatalf("Session should be a type: %+v", rs)
	}
	c := idx.Files["math.c"]
	if len(c) != 2 || c[0].Name != "add" || c[1].Name != "clamp" {
		t.Fatalf("c file: %+v", c)
	}
	if len(idx.Tests) != 1 || idx.Tests[0].Name != "test_add_works" {
		t.Fatalf("tests: %+v", idx.Tests)
	}
}
