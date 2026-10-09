// SPDX-License-Identifier: Apache-2.0
package emit

import (
	"encoding/json"
	"testing"
)

func TestBadge(t *testing.T) {
	var doc struct {
		SchemaVersion int    `json:"schemaVersion"`
		Label         string `json:"label"`
		Message       string `json:"message"`
		Color         string `json:"color"`
	}
	if err := json.Unmarshal(Badge("spec coverage", 9, 10), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || doc.Label != "spec coverage" ||
		doc.Message != "90% (9/10)" || doc.Color != "brightgreen" {
		t.Fatalf("%+v", doc)
	}
	json.Unmarshal(Badge("x", 0, 4), &doc)
	if doc.Color != "red" {
		t.Fatalf("0%% should be red: %+v", doc)
	}
	json.Unmarshal(Badge("x", 0, 0), &doc)
	if doc.Color != "red" || doc.Message != "0% (0/0)" {
		t.Fatalf("empty repo: %+v", doc)
	}
}
