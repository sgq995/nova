package router

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/module"
)

func parseDirectives(cg *ast.CommentGroup) []string {
	patterns := []string{}
	for _, c := range cg.List {
		if strings.HasPrefix(c.Text, "//nova:route ") {
			pattern := strings.TrimPrefix(c.Text, "//nova:route ")
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

func parseRoutesFromGo(filename string) ([]FuncRoute, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	routes := []FuncRoute{}
	for _, decl := range f.Decls {
		var cg *ast.CommentGroup
		var name *ast.Ident
		var funcType *ast.FuncType
		var spec *ast.TypeSpec

		switch decl := decl.(type) {
		case *ast.FuncDecl:
			cg = decl.Doc
			name = decl.Name
			funcType = decl.Type

		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}

			cg = decl.Doc
			spec = decl.Specs[0].(*ast.TypeSpec)

		default:
			continue
		}

		if cg == nil {
			continue
		}

		parseDirectives(cg)
		if name != nil && funcType != nil {
			fmt.Printf("name: %+v\n", name)
			fmt.Printf("type: %+v\n", funcType)
		}
		if spec != nil {
			fmt.Printf("spec: %+v\n", spec)
		}
	}

	return routes, nil
}

func (r *Router) scanHttpDir() {
	root := module.Abs(r.config.Router.Http)
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" {
			return nil
		}

		parseRoutesFromGo(path)

		return nil
	})
}

func (r *Router) Scan() {
	r.scanHttpDir()
}
