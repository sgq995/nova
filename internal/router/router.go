package router

import "github.com/sgq995/nova/internal/config"

type Router struct {
	config *config.Config
}

func NewRouter(c *config.Config) *Router {
	return &Router{config: c}
}

func (r *Router) ParseRoutes(files []string) (map[string][]Route, error) {
	routes, err := parse(r.config, files)
	if err != nil {
		return nil, err
	}
	return routes, nil
}
