package router

import (
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"

	"github.com/sgq995/nova/internal/module"
)

type Route interface {
	Pattern() string
}

type Router struct {
	Routes []Route
}

func NewRouter(files []string) (*Router, error) {
	routes, err := parse(files)
	if err != nil {
		return nil, err
	}
	return &Router{Routes: routes}, nil
}

func parseGoFile(filename string) ([]Route, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	basepath, _ := filepath.Rel(module.Root(), filename)

	routePath := path.Dir(basepath)

	routes := []Route{}
	return routes, nil
}

func parseJSFile(filename string) Route {
	return nil
}

func parseHTMLFile(filename string) Route {
	return nil
}

func parse(files []string) ([]Route, error) {
	routes := []Route{}
	for _, filename := range files {
		switch filepath.Ext(filename) {
		case ".go":
			goRoutes, err := parseGoFile(filename)
			if err != nil {
				return nil, err
			}
			routes = slices.Concat(routes, goRoutes)

		case ".js":
			routes = append(routes, parseJSFile(filename))

		case ".html":
			routes = append(routes, parseHTMLFile(filename))
		}
	}
	return routes, nil
}
