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

type StaticRouteHTML struct {
	Pattern  string
	Filename string
}

func (r *StaticRouteHTML) route() {}

type FuncRoute struct {
	Pattern string
	Func    string
	Recv    string
}

func (r *FuncRoute) route() {}

type StructRoute struct {
	Pattern string
}
