package index

import (
	"bufio"
	"bytes"
	"path"
	"regexp"
	"strings"
)

// hclExtractor indexes Terraform/OpenTofu files (.tf, .hcl). The named
// entities are block declarations — `resource`, `data`, `module`,
// `variable`, `output`, `locals` — so a marker binds to the block that
// implements the requirement, not just the file. HCL comments are `#`,
// `//`, or `/* */`; doc text is collected from comment lines directly
// above the block, same convention as the other extractors.
type hclExtractor struct{}

var (
	// `resource "aws_s3_bucket" "logs" {` → "aws_s3_bucket.logs"
	// `data "aws_iam_policy_document" "x" {` → "data.aws_iam_policy_document.x"
	// `module "vpc" {` / `variable "region" {` / `output "arn" {` → keyword.name
	hclBlock  = regexp.MustCompile(`^\s*(resource|data)\s+"([^"]+)"\s+"([^"]+)"`)
	hclSingle = regexp.MustCompile(`^\s*(module|variable|output|provider|terraform)\s+"?([A-Za-z_]\w*)?"?`)
	hclLocals = regexp.MustCompile(`^\s*locals\s*\{`)
	hclCmt    = regexp.MustCompile(`^\s*(?:#|//)\s?(.*)`)
)

func (hclExtractor) Match(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	return ext == ".tf" || ext == ".tfvars" || ext == ".hcl"
}

func (hclExtractor) IsTestFile(string) bool { return false } // HCL has no test files in-repo

func (hclExtractor) Parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	var cmtBuf []string
	for sc.Scan() {
		line++
		text := sc.Text()
		trimmed := strings.TrimSpace(text)
		if m := hclCmt.FindStringSubmatch(text); m != nil || trimmed == "" {
			if m != nil {
				cmtBuf = append(cmtBuf, m[1])
			}
			continue
		}
		var sym *Symbol
		if m := hclBlock.FindStringSubmatch(text); m != nil {
			// Terraform addresses: `aws_s3_bucket.logs` for resources but
			// `data.aws_iam_policy_document.x` for data sources.
			name := m[2] + "." + m[3]
			if m[1] == "data" {
				name = "data." + name
			}
			sym = &Symbol{Name: name, Kind: m[1]}
		} else if m := hclSingle.FindStringSubmatch(text); m != nil && m[2] != "" {
			sym = &Symbol{Name: m[1] + "." + m[2], Kind: m[1]}
		} else if hclLocals.MatchString(text) {
			sym = &Symbol{Name: "locals", Kind: "locals"}
		}
		if sym != nil {
			sym.File, sym.Line, sym.Exported = filename, line, true
			sym.Doc = joinDoc(cmtBuf, "")
			exports = append(exports, *sym)
		}
		cmtBuf = cmtBuf[:0]
	}
	return exports, nil, nil
}
