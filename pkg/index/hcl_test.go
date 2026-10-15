package index

import (
	"testing"
	"testing/fstest"
)

func TestHclExtractor(t *testing.T) {
	fsys := fstest.MapFS{
		"main.tf": &fstest.MapFile{Data: []byte(`# Bucket holding scan artifacts.
# Encrypted at rest per policy.
resource "aws_s3_bucket" "artifacts" {
  bucket = "artifacts"
}

data "aws_iam_policy_document" "read" {
  statement {}
}

module "vpc" {
  source = "./vpc"
}

variable "region" {
  default = "us-east-2"
}

output "bucket_arn" {
  value = aws_s3_bucket.artifacts.arn
}

locals {
  env = "prod"
}
`)},
		"vars.tfvars": &fstest.MapFile{Data: []byte(`region = "us-east-2"
`)},
	}
	idx, err := Scan(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	syms := idx.Files["main.tf"]
	names := map[string]string{}
	for i := range syms {
		names[syms[i].Name] = syms[i].Kind
	}
	want := map[string]string{
		"aws_s3_bucket.artifacts":           "resource",
		"data.aws_iam_policy_document.read": "data",
		"module.vpc":                        "module",
		"variable.region":                   "variable",
		"output.bucket_arn":                 "output",
		"locals":                            "locals",
	}
	for name, kind := range want {
		if names[name] != kind {
			t.Errorf("want %s kind %q, got %q (all: %v)", name, kind, names[name], names)
		}
	}

	// Doc comments above a block attach to the symbol.
	var bucket *Symbol
	for i := range syms {
		if syms[i].Name == "aws_s3_bucket.artifacts" {
			bucket = &syms[i]
		}
	}
	if bucket == nil || bucket.Doc == "" {
		t.Fatalf("expected doc on aws_s3_bucket.artifacts, got %+v", bucket)
	}

	// .tfvars files match but produce no symbols (assignments, not blocks).
	if len(idx.Files["vars.tfvars"]) != 0 {
		t.Errorf("tfvars produced symbols: %v", idx.Files["vars.tfvars"])
	}
}
