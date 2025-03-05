package router

import "github.com/sgq995/nova/internal/config"

type Router struct {
	Routes map[string][]Route
}

func NewRouter(c *config.Config, files []string) (*Router, error) {
	routes, err := parse(c, files)
	if err != nil {
		return nil, err
	}
	return &Router{Routes: routes}, nil
}
