package router

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/parser"
)

func parseGoFile(base, filename string) ([]Route, error) {
	handlers, err := parser.ParseRouteHandlersGo(filename)
	if err != nil {
		return nil, err
	}

	basepath := module.Abs(filepath.FromSlash(base))

	routePath, _ := filepath.Rel(basepath, filepath.Dir(filename))
	routePath = filepath.ToSlash(routePath)
	routePath = path.Clean("/" + routePath)
	// TODO: config traling slashes
	if strings.HasSuffix(routePath, "/") {
		routePath += "{$}"
	}

	routes := []Route{}
	for _, h := range handlers {
		method := strings.ToUpper(h)
		switch method {
		case "RENDER":
			routes = append(routes, &RenderRouteGo{
				Pattern:   "GET " + routePath,
				Root:      routePath,
				Templates: []string{},
				Handler:   h,
			})

		case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace:
			// TODO: config api base
			routePath = path.Join("/api", routePath)

			routes = append(routes, &RestRouteGo{
				Pattern: method + " " + routePath,
				Handler: h,
			})
		}
	}

	return routes, nil
}

func parseJSFile(filename string) Route {
	// TODO: handle JS ssr if configured
	return nil
}

func parseHTMLFile(filename string) Route {
	// TODO: handle static non-template html
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
