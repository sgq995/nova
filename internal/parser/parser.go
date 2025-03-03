package parser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func ParseImportsGo(filename string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	basepath := filepath.Dir(filename)

	imports := []string{}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//nova:template ") {
				data := strings.TrimPrefix(c.Text, "//nova:template ")
				files := strings.Split(data, " ")
				for _, f := range files {
					imports = append(imports, filepath.Clean(filepath.Join(basepath, f)))
				}
			}
		}
	}

	return imports, nil
}

func ParseImportsHTML(filename string) ([]string, error) {
	basepath := filepath.Dir(filename)

	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	imports := []string{}
	nodes := doc.Descendants()
	for n := range nodes {
		if n.Type != html.ElementNode {
			continue
		}

		switch n.DataAtom {
		case atom.Script, atom.Img:
			for _, attr := range n.Attr {
				if strings.ToLower(attr.Key) == "src" {
					src := filepath.Join(basepath, attr.Val)
					src = filepath.Clean(src)
					imports = append(imports, src)
					break
				}
			}

		case atom.Link:
			for _, attr := range n.Attr {
				if strings.ToLower(attr.Key) == "href" {
					href := filepath.Join(basepath, attr.Val)
					href = filepath.Clean(href)
					imports = append(imports, href)
					break
				}
			}
		}
	}

	return imports, nil
}

func ParseRouteHandlersGo(filename string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	handlers := []string{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}

		identifier := strings.ToUpper(fn.Name.Name)

		switch identifier {
		case "RENDER":
			handlers = append(handlers, fn.Name.Name)

		case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace:
			handlers = append(handlers, fn.Name.Name)
		}
	}

	return handlers, nil
}
