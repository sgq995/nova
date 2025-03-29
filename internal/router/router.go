package router

import (
	"maps"

	"github.com/sgq995/nova/internal/config"
)

type Router struct {
	Routes map[string][]Route

	config *config.Config
}

func New(c *config.Config) *Router {
	return &Router{
		Routes: make(map[string][]Route),
		config: c,
	}
}

func (r *Router) ParseRoute(filename string) ([]Route, error) {
	routes, err := parseFile(r.config, filename)
	if err != nil {
		return nil, err
	}
	r.Routes[filename] = routes
	return routes, nil
}

func (r *Router) ParseRoutes(files []string) (map[string][]Route, error) {
	routesMap, err := parseFiles(r.config, files)
	if err != nil {
		return nil, err
	}
	maps.Copy(r.Routes, routesMap)
	return routesMap, nil
}
