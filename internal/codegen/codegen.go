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

const mainTmpl string = `package main

import (
	"log"
	"html/template"
	"net/http"
	"os"
	{{range $alias, $package := .Imports}}
	{{$alias}} "{{$package}}"{{end}}
)

func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	{{if .IsProd -}}
	fs := os.DirFS(root)
	t := template.Must(template.ParseFS(fs, templates...)){{end}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		{{if not .IsProd -}}
		fs := os.DirFS("{{.Root}}" + root)
		t := template.Must(template.ParseFS(fs, templates...)){{end}}
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func main() {
	mux := http.NewServeMux()
	{{range $filename, $handler := .Handlers}}
	// {{$filename}}
	{{with $render := $handler.Render}}mux.Handle("{{$render.Pattern}}", renderHandler("{{$render.Root}}", []string{ {{- range $render.Templates}}"{{.}}", {{end -}} }, {{$handler.Package}}.{{$render.Handler}})){{end}}
	{{range $handler.Rest}}mux.HandleFunc("{{.Pattern}}", {{$handler.Package}}.{{.Handler}}){{end}}
	{{end}}
	s := http.Server{
		Addr:    "{{.Host}}:{{.Port}}",
		Handler: mux,
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

func generateMain(c *config.Config, routes map[string][]router.Route) error {
	t := template.Must(template.New("main.go").Parse(mainTmpl))

	outDir := filepath.Join(module.Root(), c.Codegen.OutDir)
	out := filepath.Join(outDir, "main.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	imports := map[string]string{}
	handlers := map[string]handler{}
	for filename, routes := range routes {
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

	isProd := os.Getenv("NOVA_ENV") == "production"
	pagespath := module.Abs(filepath.FromSlash(c.Router.Pages)) + "/"

	return t.Execute(file, map[string]any{
		"IsProd":   isProd,
		"Imports":  imports,
		"Root":     pagespath,
		"Handlers": handlers,
		"Host":     c.Server.Host,
		"Port":     c.Server.Port,
	})
}

type Codegen struct {
	config *config.Config
}

func NewCodegen(c *config.Config) *Codegen {
	return &Codegen{config: c}
}

func (c *Codegen) Generate(routes map[string][]router.Route) error {
	outDir := filepath.Join(module.Root(), c.config.Codegen.OutDir)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	err = generateMain(c.config, routes)
	if err != nil {
		return err
	}

	return nil
}
