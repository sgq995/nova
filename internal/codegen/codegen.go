package codegen

import (
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
)

const routerTmpl string = `package router

import (
	"html/template"
	"net/http"
	"os"
	{{range $alias, $package := .Imports}}
	{{$alias}} "{{$package}}"{{end}}
)

func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	fs := os.DirFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := template.Must(template.ParseFS(fs, templates...))
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	{{range $filename, $handler := .Handlers}}
	// {{$filename}}
	{{with $render := $handler.Render}}mux.Handle("{{$render.Pattern}}", renderHandler("{{$render.Root}}", []string{ {{- range $render.Templates}}"{{.}}", {{end -}} }, {{$handler.Package}}.{{$render.Handler}})){{end}}
	{{range $handler.Rest}}mux.HandleFunc("{{.Pattern}}", {{$handler.Package}}.{{.Handler}}){{end}}
	{{end}}
	return mux
}
`

const mainTmpl string = `package main

import (
	"log"
	"net/http"

	"{{.Module}}/{{.Router}}"
)

func main() {
	r := router.NewRouter()

	s := http.Server{
		Addr:    "{{.Host}}:{{.Port}}",
		Handler: r,
	}
	err := s.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}
`

type handler struct {
	Render  *router.RenderRouteGo
	Rest    []*router.RestRouteGo
	Package string
}

func generateRouter(basepath string, r *router.Router) error {
	t := template.Must(template.New("router.go").Parse(routerTmpl))

	outDir := filepath.Join(basepath, "router")
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	out := filepath.Join(basepath, "router", "router.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	imports := map[string]string{}
	handlers := map[string]handler{}
	for filename, routes := range r.Routes {
		base, _ := filepath.Rel(module.Root(), filepath.Dir(filename))
		base = filepath.ToSlash(base)
		alias := strings.ReplaceAll(base, "/", "")
		pkg := path.Join(module.ModuleName(), base)
		imports[alias] = pkg

		h := handler{
			Rest:    make([]*router.RestRouteGo, 0),
			Package: alias,
		}

		for _, route := range routes {
			switch v := route.(type) {
			case *router.RenderRouteGo:
				h.Render = v

			case *router.RestRouteGo:
				h.Rest = append(h.Rest, v)
			}
		}

		handlers[filename] = h
	}

	return t.Execute(file, map[string]any{
		"Imports":  imports,
		"Handlers": handlers,
	})
}

func generateMain(basepath string) error {
	t := template.Must(template.New("main.go").Parse(mainTmpl))

	out := filepath.Join(basepath, "main.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, map[string]any{
		"Module": module.ModuleName(),
		"Router": path.Join(basepath, "router"),
		"Host":   "",
		"Port":   "8080",
	})
}

func Generate(config config.CodegenConfig, router *router.Router) error {
	outDir := filepath.Join(module.Root(), ".nova")
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	err = generateRouter(outDir, router)
	if err != nil {
		return err
	}

	err = generateMain(outDir)
	if err != nil {
		return err
	}

	return nil
}
