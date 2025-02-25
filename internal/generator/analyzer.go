package generator

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"segoqu.com/nova/internal/project"
)

type Kind int

const (
	KindRender Kind = iota
	KindRest
)

type RouteInfo struct {
	Method    string
	Path      string
	Package   string
	Handler   string
	Kind      Kind
	Templates []string
}

func parseGoFile(filename, dir string) ([]RouteInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	filename = path.Join(filepath.SplitList(filename)...)

	pkg := strings.TrimPrefix(filename, project.Root()+"/")
	pkg = filepath.Dir(pkg)

	basename := strings.TrimPrefix(filename, dir+"/")
	basename = strings.TrimSuffix(basename, ".go")
	basename = strings.TrimSuffix(basename, "index")

	templates := []string{}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			fmt.Println(c.Text)
			if strings.HasPrefix(c.Text, "//nova:template ") {
				filenames := strings.TrimPrefix(c.Text, "//nova:template ")
				templates = slices.Concat(templates, strings.Split(filenames, " "))
			}
		}
	}

	var routes []RouteInfo
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.IsExported() {
			handler := fn.Name.Name
			identifier := strings.ToUpper(handler)

			var (
				method, route string
				kind          Kind
			)
			switch identifier {
			case "RENDER":
				method = http.MethodGet
				route = "/" + basename
				kind = KindRender

			case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace:
				method = identifier
				route = "/api/" + basename
				kind = KindRest

			default:
				// TODO: log not route found?
				continue
			}

			if strings.HasSuffix(route, "/") {
				route += "{$}"
			}

			log.Println(method, route)

			routes = append(routes, RouteInfo{
				Method:    method,
				Path:      route,
				Package:   pkg,
				Handler:   handler,
				Kind:      kind,
				Templates: templates,
			})
		}
	}

	return routes, nil
}

func FindRoutes(dir string) ([]RouteInfo, error) {
	var routes []RouteInfo
	var errs []error

	target := project.Abs(dir)

	log.Println("searching routes:", dir)
	errs = append(errs, filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".go" {
			goRoutes, err := parseGoFile(path, target)
			if err != nil {
				return err
			}
			routes = slices.Concat(routes, goRoutes)
		}

		return nil
	}))

	err := errors.Join(errs...)

	return routes, err
}
