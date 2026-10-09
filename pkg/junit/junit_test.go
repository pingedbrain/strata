// SPDX-License-Identifier: Apache-2.0
package junit

import "testing"

func TestParseSuites(t *testing.T) {
	data := []byte(`<testsuites tests="3" failures="1">
  <testsuite name="pkg">
    <testcase classname="pkg.TestX" name="TestValidate" time="0.01"/>
    <testcase classname="pkg.TestX" name="TestReset" time="0.02"><failure>boom</failure></testcase>
    <testcase classname="pkg.TestX" name="test_skip.me" time="0"><skipped/></testcase>
  </testsuite>
</testsuites>`)
	cases, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("want 3 cases, got %d", len(cases))
	}
	if cases[0].Name != "TestValidate" || cases[0].Failed || cases[0].Skipped {
		t.Fatalf("case0: %+v", cases[0])
	}
	if !cases[1].Failed {
		t.Fatalf("case1 should be failed: %+v", cases[1])
	}
	if cases[2].Name != "me" || !cases[2].Skipped { // qualified name → last segment
		t.Fatalf("case2: %+v", cases[2])
	}
}

func TestParseBareSuite(t *testing.T) {
	data := []byte(`<testsuite name="x"><testcase name="TestA"/></testsuite>`)
	cases, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Name != "TestA" {
		t.Fatalf("%+v", cases)
	}
}
