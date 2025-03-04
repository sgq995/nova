package router

type Route interface {
	route()
}

type RenderRouteGo struct {
	Pattern   string
	Root      string
	Templates []string
	Handler   string
}

func (r *RenderRouteGo) route() {}

type RestRouteGo struct {
	Pattern string
	Handler string
}

func (r *RestRouteGo) route() {}
