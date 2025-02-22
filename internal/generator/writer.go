package generator

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"segoqu.com/nova/internal/project"
)

type Option struct {
	key   string
	value any
}

func OptionImport() *Option {
	return nil
}

func alias(r *RouteInfo) string {
	return strings.ReplaceAll(r.Package, "/", "")
}

type importStmt struct {
	Alias string
	Path  string
}

type route struct {
	Path    string
	Handler string
}

func defaultData() map[string]any {
	return map[string]any{
		"Imports": []importStmt{},

		"Embed": "",

		"SSRRoutes": []route{},
		"APIRoutes": []route{},

		"Static": "",

		"Host": "",
		"Port": "8080",
	}
}

func createData(options ...*Option) map[string]any {
	data := defaultData()
	for _, opt := range options {
		data[opt.key] = opt.value
	}
	return data
}

func importRoutes(data map[string]any, routes []RouteInfo) {
	var routeImports []importStmt

	for _, r := range routes {
		routeImports = append(routeImports, importStmt{
			Alias: alias(&r),
			Path:  path.Join(project.ModuleName(), r.Package),
		})
	}

	if _, ok := data["Imports"]; ok {
		imports := data["Imports"].([]importStmt)
		data["Imports"] = slices.Concat(routeImports, imports)
	} else {
		data["Imports"] = routeImports
	}
}

func registerRoutes(data map[string]any, routes []RouteInfo) {
	var ssrRoutes []route
	var apiRoutes []route

	for _, r := range routes {
		route := route{
			Path:    r.Method + " " + r.Path,
			Handler: alias(&r) + "." + r.Handler,
		}

		switch r.Kind {
		case KindRender:
			ssrRoutes = append(ssrRoutes, route)

		case KindRest:
			apiRoutes = append(apiRoutes, route)

		default:
			continue
		}
	}

	data["SSRRoutes"] = ssrRoutes
	data["APIRoutes"] = apiRoutes
}

func GenerateServer(routes []RouteInfo, options ...*Option) error {
	outDir := filepath.Join(project.Root(), ".nova")
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	out := filepath.Join(outDir, "main.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	data := createData(options...)
	importRoutes(data, routes)
	registerRoutes(data, routes)

	tmpl := template.Must(template.New("main").Parse(mainFile))
	return tmpl.Execute(file, data)
}
