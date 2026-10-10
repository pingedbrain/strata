package index

import (
	"encoding/binary"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LoadSCIP reads a SCIP index (protobuf wire format) produced by
// language indexers like scip-go or scip-python and converts symbol
// definitions into an Index. Hand-rolled wire decoding — protobuf
// fields we don't need are skipped, so unknown/newer fields are fine.
// @spec index/load-scip
func LoadSCIP(data []byte) (*Index, error) {
	idx := &Index{Files: map[string][]Symbol{}}
	var err error
	decodeFields(data, func(f, wt int, num uint64, raw []byte) {
		if f == 2 && wt == 2 { // Index.documents
			loadSCIPDocument(idx, raw)
		}
	})
	if err != nil {
		return nil, err
	}
	return idx, nil
}

// Overlay merges richer symbol data (typically SCIP) into i: files the
// other index covers are replaced outright, others are left alone.
// SCIP data wins over regex extraction for the same file.
// @spec index/index-overlay
func (i *Index) Overlay(other *Index) {
	if other == nil {
		return
	}
	for f, syms := range other.Files {
		i.Files[f] = syms
	}
	i.Tests = append(i.Tests, other.Tests...)
	if len(other.Refs) > 0 && i.Refs == nil {
		i.Refs = map[string][]Symbol{}
	}
	for f, refs := range other.Refs {
		i.Refs[f] = refs
	}
}

func loadSCIPDocument(idx *Index, data []byte) {
	var path, language string
	var occs [][]byte
	decodeFields(data, func(f, wt int, num uint64, raw []byte) {
		switch f {
		case 1: // relative_path
			path = string(raw)
		case 2: // occurrences
			occs = append(occs, raw)
		case 4: // language
			language = string(raw)
		}
	})
	if path == "" {
		return
	}
	for _, o := range occs {
		sym, roles, line := scipOccurrence(o)
		name, kind := scipSymbolName(sym)
		if name == "" {
			continue // locals and unparsable symbols don't map to declarations
		}
		if roles&0x1 == 0 { // reference, not a definition
			if idx.Refs == nil {
				idx.Refs = map[string][]Symbol{}
			}
			idx.Refs[path] = append(idx.Refs[path], Symbol{
				Name: name, Kind: "ref", File: path, Line: line + 1,
			})
			continue
		}
		s := Symbol{
			Name: name, Kind: kind, File: path, Line: line + 1, // scip is 0-based
			Exported: scipExported(name, language),
		}
		if roles&0x20 != 0 || (isSCIPTestFile(path) && isSCIPTestName(name)) {
			idx.Tests = append(idx.Tests, s)
		} else {
			idx.Files[path] = append(idx.Files[path], s)
		}
	}
}

// isSCIPTestFile mirrors the per-language test-file conventions of the
// native extractors so SCIP data lands in the same buckets.
func isSCIPTestFile(p string) bool {
	base := p
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		base = p[i+1:]
	}
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "_test.py") || strings.HasSuffix(base, "_test.ts") ||
		strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".spec.ts") ||
		strings.HasPrefix(base, "test_")
}

func isSCIPTestName(name string) bool {
	return strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "test_")
}

// scipOccurrence extracts (symbol, roles, startLine) from an Occurrence.
func scipOccurrence(data []byte) (sym string, roles int, line int) {
	decodeFields(data, func(f, wt int, num uint64, raw []byte) {
		switch f {
		case 1: // deprecated packed int32 range
			if wt == 2 && len(raw) > 0 {
				l, _ := binary.Uvarint(raw)
				line = int(l)
			}
		case 2:
			sym = string(raw)
		case 3:
			roles = int(num)
		case 8: // single_line_range{line=1}
			decodeFields(raw, func(f2, wt2 int, num2 uint64, _ []byte) {
				if f2 == 1 {
					line = int(num2)
				}
			})
		case 9: // multi_line_range{start_line=1}
			decodeFields(raw, func(f2, wt2 int, num2 uint64, _ []byte) {
				if f2 == 1 {
					line = int(num2)
				}
			})
		}
	})
	return
}

// scipSymbolName extracts a display name + kind from a SCIP symbol
// string: "scheme manager name version desc1 desc2…". Descriptors:
// name/ = namespace, name# = type, name. = term, name(). = method,
// name! = macro, name: = meta.
func scipSymbolName(sym string) (name, kind string) {
	if strings.HasPrefix(sym, "local ") {
		return "", ""
	}
	// descriptors live after scheme + package(manager name version)
	parts := strings.SplitN(sym, " ", 5)
	if len(parts) < 5 {
		return "", ""
	}
	desc := parts[4]
	var typeName, last, lastKind string
	for i := 0; i < len(desc); {
		// read one descriptor name (may be `backtick escaped`)
		start := i
		if desc[i] == '`' {
			i++
			for i < len(desc) && desc[i] != '`' {
				i++
			}
			i++ // closing backtick
		} else {
			for i < len(desc) && !strings.ContainsRune("/#.(:![", rune(desc[i])) {
				i++
			}
		}
		if i >= len(desc) {
			break
		}
		n := desc[start:i]
		switch desc[i] {
		case '#':
			typeName, last, lastKind = n, n, "type"
			i++
		case '.':
			// term under a type = struct field/member — not a
			// declaration markers should bind to
			if typeName != "" {
				return "", ""
			}
			last, lastKind = n, "var"
			i++
		case '/':
			i++ // namespace — dropped from display
		case '(':
			// method: name(disambiguator). — consume "…)."
			j := strings.Index(desc[i:], ").")
			if j < 0 {
				return "", ""
			}
			i += j + 2
			last, lastKind = n, "func"
			if typeName != "" {
				last, lastKind = typeName+"."+n, "method"
			}
		case '!', ':', '[':
			i++
			for i < len(desc) && !strings.ContainsRune("/#.(:!", rune(desc[i])) {
				i++
			}
		default:
			return "", ""
		}
	}
	return last, lastKind
}

// scipExported guesses public visibility per-language: Go = uppercase,
// most others = no leading underscore.
func scipExported(name, language string) bool {
	base := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		base = name[i+1:]
	}
	if language == "go" || language == "Go" {
		r, _ := utf8.DecodeRuneInString(base)
		return unicode.IsUpper(r)
	}
	return !strings.HasPrefix(base, "_")
}

// decodeFields walks protobuf wire fields. cb receives (field number,
// wire type, varint value, raw bytes for length-delimited fields).
// Unknown wire types abort silently — partial data beats none.
func decodeFields(b []byte, cb func(field, wireType int, num uint64, raw []byte)) {
	for i := 0; i < len(b); {
		key, n := binary.Uvarint(b[i:])
		if n <= 0 {
			return
		}
		i += n
		f, wt := int(key>>3), int(key&7)
		switch wt {
		case 0:
			v, m := binary.Uvarint(b[i:])
			if m <= 0 {
				return
			}
			i += m
			cb(f, wt, v, nil)
		case 1:
			if i+8 > len(b) {
				return
			}
			cb(f, wt, 0, b[i:i+8])
			i += 8
		case 2:
			l, m := binary.Uvarint(b[i:])
			if m <= 0 || i+m+int(l) > len(b) {
				return
			}
			cb(f, wt, 0, b[i+m:i+m+int(l)])
			i += m + int(l)
		case 5:
			if i+4 > len(b) {
				return
			}
			cb(f, wt, 0, b[i:i+4])
			i += 4
		default:
			return
		}
	}
}
