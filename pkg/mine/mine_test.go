package mine

import (
	"testing"
	"testing/fstest"
)

var goRepo = fstest.MapFS{
	"pkg/auth/token.go": &fstest.MapFile{Data: []byte(`package auth

// Validate checks a token.
func Validate(t string) bool { return t != "" }

func helper() {}

type Session struct{ ID string }
`)},
	"pkg/auth/token_test.go": &fstest.MapFile{Data: []byte(`package auth

func TestValidateRejectsEmpty(t *testing.T) {}
func TestUnrelated(t *testing.T) {}
`)},
	"cmd/tool/main.go": &fstest.MapFile{Data: []byte(`package main
func main() {}
`)},
	"README.md": &fstest.MapFile{Data: []byte("# x\n")},
}

func TestScanGroupsCapabilitiesAndLinksTests(t *testing.T) {
	caps, err := Scan(goRepo)
	if err != nil {
		t.Fatal(err)
	}
	var auth *Capability
	for i := range caps {
		if caps[i].Name == "auth" {
			auth = &caps[i]
		}
	}
	if auth == nil {
		t.Fatalf("auth capability missing: %+v", caps)
	}
	var validate *Candidate
	for i := range auth.Candidates {
		if auth.Candidates[i].Name == "Validate" {
			validate = &auth.Candidates[i]
		}
	}
	if validate == nil {
		t.Fatalf("Validate not found: %+v", auth.Candidates)
	}
	if len(validate.Tests) != 1 || validate.Tests[0].Name != "TestValidateRejectsEmpty" {
		t.Fatalf("test linkage wrong: %+v", validate.Tests)
	}
	// helper() unexported, Session exported → 2 candidates in auth
	if len(auth.Candidates) != 2 {
		t.Fatalf("want 2 candidates (Validate, Session), got %+v", auth.Candidates)
	}
}

var pyRepo = fstest.MapFS{
	"src/proxy/forward.py": &fstest.MapFile{Data: []byte(`def forward_request(req): pass
def _hidden(): pass
class Relay: pass
`)},
	"tests/test_forward.py": &fstest.MapFile{Data: []byte(`def test_forward_request_retries(): pass
`)},
}

func TestScanPython(t *testing.T) {
	caps, err := Scan(pyRepo)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*Candidate{}
	for _, c := range caps {
		for i := range c.Candidates {
			found[c.Candidates[i].Name] = &c.Candidates[i]
		}
	}
	if found["forward_request"] == nil || found["Relay"] == nil {
		t.Fatalf("python exports missing: %+v", caps)
	}
	if found["_hidden"] != nil {
		t.Fatal("private symbol leaked")
	}
	if len(found["forward_request"].Tests) != 1 {
		t.Fatalf("python test not linked: %+v", found["forward_request"])
	}
}
