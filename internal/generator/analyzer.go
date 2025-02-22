package generator

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Kind int

const (
	KindRender Kind = iota
	KindRest
)

type RouteInfo struct {
	Method  string
	Path    string
	Package string
	Handler string
	Kind    Kind
}

var root string = must(projectRoot())

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func projectRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var goMod string
	for len(root) > 1 {
		goMod = filepath.Join(root, "go.mod")
		exists, err := fileExists(goMod)
		if err != nil {
			return "", err
		}

		if exists {
			return root, nil
		}

		root = filepath.Dir(root)
	}

	return "", nil
}

func parseGoFile(filename, dir string) ([]RouteInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	pkg := strings.TrimPrefix(filename, root+"/")
	pkg = filepath.Dir(pkg)

	basename := strings.TrimPrefix(filename, dir+"/")
	basename = strings.TrimSuffix(basename, ".go")
	basename = strings.TrimSuffix(basename, "index")

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
				Method:  method,
				Path:    route,
				Package: pkg,
				Handler: handler,
				Kind:    kind,
			})
		}
	}

	return routes, nil
}

func FindRoutes(dir string) ([]RouteInfo, error) {
	var routes []RouteInfo
	var errs []error

	target := dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, dir)
	}

	log.Println("Search for routes at", dir)
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
