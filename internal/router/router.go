package router

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/parser"
)

type Route interface {
	Pattern() string
}

type routeImpl struct {
	method string
	path   string
}

func (r *routeImpl) Pattern() string {
	return r.method + " " + r.path
}

type Router struct {
	Routes map[string][]Route
}

func NewRouter(base string, files []string) (*Router, error) {
	routes, err := parse(base, files)
	if err != nil {
		return nil, err
	}
	return &Router{Routes: routes}, nil
}

func parseGoFile(base, filename string) ([]Route, error) {
	handlers, err := parser.ParseRouteHandlersGo(filename)
	if err != nil {
		return nil, err
	}

	basepath := module.Abs(base)

	routePath, _ := filepath.Rel(basepath, filepath.Dir(filename))
	routePath = filepath.ToSlash(routePath)
	routePath = path.Clean("/" + routePath)

	routes := []Route{}
	for _, h := range handlers {
		method := strings.ToUpper(h)
		if method == "RENDER" {
			method = http.MethodGet
		} else {
			routePath = path.Join("/api", routePath)
		}

		// TODO: config traling slashes
		if strings.HasSuffix(routePath, "/") {
			// non-root routes
			if len(routePath) > 1 {
				routes = append(routes, &routeImpl{
					method: method,
					path:   strings.TrimSuffix(routePath, "/"),
				})
			}

			routePath += "{$}"
		}

		routes = append(routes, &routeImpl{
			method: method,
			path:   routePath,
		})
	}

	return routes, nil
}

func parseJSFile(filename string) Route {
	return nil
}

func parseHTMLFile(filename string) Route {
	return nil
}

func parse(base string, files []string) (map[string][]Route, error) {
	routes := map[string][]Route{}
	for _, filename := range files {
		switch filepath.Ext(filename) {
		case ".go":
			goRoutes, err := parseGoFile(base, filename)
			if err != nil {
				return nil, err
			}
			routes[filename] = goRoutes

		case ".js":
			routes[filename] = []Route{parseJSFile(filename)}

		case ".html":
			routes[filename] = []Route{parseHTMLFile(filename)}
		}
	}
	return routes, nil
}
