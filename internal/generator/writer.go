package generator

import (
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/module"
)

type Option struct {
	key   string
	value any
}

func OptionImport(alias, path string) *Option {
	return &Option{
		key: "Imports",
		value: importStmt{
			Alias: alias,
			Path:  path,
		},
	}
}

func OptionProxy(addr string) *Option {
	return &Option{
		key: "Middlewares",
		value: middleware{
			Imports: []importStmt{
				{Path: "net/http/httputil"},
				{Path: "net/url"},
			},
			Decl: proxyMiddlewareDecl,
			Name: "withReverseProxy",
			Args: []any{`"http://` + addr + `"`},
		},
	}
}

func OptionStatic() *Option {
	return &Option{
		key: "Middlewares",
		value: middleware{
			Imports: []importStmt{
				{Path: "embed"},
			},
			Decl: staticMiddlewareDecl,
			Name: "withFileServer",
			Args: []any{},
		},
	}
}

type Environment int

const (
	Dev Environment = iota
	Prod
)

func OptionEnvironment(env Environment) *Option {
	environment := "dev"
	if env == Prod {
		environment = "prod"
	}

	return &Option{
		key:   "Environment",
		value: environment,
	}
}

func alias(r *RouteInfo) string {
	return strings.ReplaceAll(r.Package, "/", "")
}

type importStmt struct {
	Alias string
	Path  string
}

type route struct {
	Path      string
	Handler   string
	Root      string
	Templates []string
}

type middleware struct {
	Imports []importStmt
	Decl    string
	Name    string
	Args    []any
}

func defaultData() map[string]any {
	return map[string]any{
		"Imports": []importStmt{
			{Path: "html/template"},
			{Path: "log"},
			{Path: "net/http"},
		},

		"Environment": "",

		"SSRRoutes": []route{},
		"APIRoutes": []route{},

		"Static": "",

		"Middlewares": []middleware{},

		"Host": "",
		"Port": "8080",
	}
}

func createData(options ...*Option) map[string]any {
	data := defaultData()
	for _, opt := range options {
		switch old := data[opt.key].(type) {
		case []middleware:
			if v, ok := opt.value.(middleware); ok {
				data[opt.key] = append(old, v)
			}

		case []any:
			if v, ok := opt.value.([]any); ok {
				data[opt.key] = slices.Concat(old, v)
			} else {
				data[opt.key] = append(old, opt.value)
			}

		default:
			data[opt.key] = opt.value
		}
	}
	return data
}

func importRoutes(data map[string]any, routes []RouteInfo) {
	var routeImports []importStmt

	for _, r := range routes {
		routeImports = append(routeImports, importStmt{
			Alias: alias(&r),
			Path:  path.Join(module.ModuleName(), r.Package),
		})
	}

	if _, ok := data["Imports"]; ok {
		imports := data["Imports"].([]importStmt)
		data["Imports"] = slices.Concat(routeImports, imports)
	} else {
		data["Imports"] = routeImports
	}
}

func processImports(data map[string]any) {
	imports := data["Imports"].([]importStmt)

	if raw, exists := data["Environment"]; exists {
		environment := raw.(string)
		if environment == "prod" {
			imports = append(imports,
				importStmt{Path: "embed"},
				importStmt{Path: "io/fs"},
				importStmt{Path: "path"},
			)
		}
		if environment == "dev" {
			imports = append(imports, importStmt{Path: "path/filepath"})
		}
	}

	if raw, exists := data["Middlewares"]; exists {
		middlewares := raw.([]middleware)
		for _, m := range middlewares {
			imports = append(imports, m.Imports...)
		}
	}

	set := make(map[string]importStmt)
	for _, imprt := range imports {
		set[imprt.Path] = imprt
	}

	data["Imports"] = slices.Collect(maps.Values(set))
}

func registerRoutes(data map[string]any, routes []RouteInfo) {
	var ssrRoutes []route
	var apiRoutes []route

	for _, r := range routes {
		route := route{
			Path:      r.Method + " " + r.Path,
			Handler:   alias(&r) + "." + r.Handler,
			Root:      r.Root,
			Templates: r.Templates,
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
	outDir := filepath.Join(module.Root(), ".nova")
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
	processImports(data)
	registerRoutes(data, routes)

	tmpl := template.Must(template.New("main").Parse(mainFile))
	return tmpl.Execute(file, data)
}
