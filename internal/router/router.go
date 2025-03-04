package router

import "github.com/sgq995/nova/internal/config"

type Router struct {
	Routes map[string][]Route
}

func NewRouter(config config.RouterConfig, files []string) (*Router, error) {
	routes, err := parse(config.Pages, files)
	if err != nil {
		return nil, err
	}
	return &Router{Routes: routes}, nil
}
