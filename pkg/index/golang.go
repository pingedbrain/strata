package index

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type goExtractor struct{}

func (goExtractor) Match(p string) bool { return strings.HasSuffix(p, ".go") }

func (goExtractor) IsTestFile(p string) bool {
	return strings.HasSuffix(p, "_test.go")
}

func (goExtractor) Parse(data []byte, filename string) (exports, tests, stepdefs []Symbol) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, data, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return nil, nil, nil
	}
	testFile := strings.HasSuffix(filename, "_test.go")
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			pos := fset.Position(d.Pos())
			if testFile {
				if strings.HasPrefix(d.Name.Name, "Test") {
					tests = append(tests, Symbol{Name: d.Name.Name, Kind: "test", File: filename, Line: pos.Line})
				}
				continue
			}
			kind, name := "func", d.Name.Name
			exported := ast.IsExported(d.Name.Name)
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := recvName(d.Recv.List[0].Type)
				if recv == "" {
					continue // can't name the receiver
				}
				kind = "method"
				name = recv + "." + name
				exported = ast.IsExported(recv) && ast.IsExported(d.Name.Name)
			}
			exports = append(exports, Symbol{Name: name, Kind: kind, File: filename, Line: pos.Line, Exported: exported, Doc: docText(d.Doc)})
		case *ast.GenDecl:
			if testFile || d.Tok != token.TYPE {
				continue
			}
			for _, sp := range d.Specs {
				ts := sp.(*ast.TypeSpec)
				pos := fset.Position(ts.Pos())
				doc := ts.Doc
				if doc == nil {
					doc = d.Doc // doc comment sits on the `type (...)` decl
				}
				exports = append(exports, Symbol{Name: ts.Name.Name, Kind: "type", File: filename, Line: pos.Line, Exported: ast.IsExported(ts.Name.Name), Doc: docText(doc)})
			}
		}
	}
	return exports, tests, nil
}

func docText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(g.Text())
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
