package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
)

const mainProdServer string = `package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	{{range $alias, $package := .Imports}}
	{{$alias}} "{{$package}}"{{end}}
)

//go:embed static
var staticFS embed.FS

//go:embed pages templates
var htmlFS embed.FS

var templatesFS fs.FS = must(fs.Sub(htmlFS, "templates"))

var pagesFS fs.FS = must(fs.Sub(htmlFS, "pages"))

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

{{template "renderHandler" .}}

func main() {
	mux := http.NewServeMux()
	{{- $global := . -}}
	{{range $filename, $handler := .Handlers}}
	// {{$filename}}
	{{with $render := $handler.Render}}mux.Handle("{{$render.Pattern}}", renderHandler("{{$render.Root}}", []string{ {{- range $render.Templates}}"{{.}}", {{end -}} }, {{$handler.Package}}.{{$render.Handler}})){{end}}
	{{range $handler.Rest}}mux.HandleFunc("{{.Pattern}}", {{$handler.Package}}.{{.Handler}}){{end}}
	{{end}}

	// nova
	mux.Handle("/static/", http.FileServerFS(staticFS))
	mux.Handle("/", http.FileServerFS(pagesFS))
	
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

var mainProdServerTempl *template.Template = newProdServerTemplate()

func newProdServerTemplate() *template.Template {
	mainTemplate := template.Must(template.New("main.go").Parse(mainProdServer))
	template.Must(mainTemplate.New("renderHandler").Parse(renderHandlerFunc))
	return mainTemplate
}

type routeHandler struct {
	Render  *router.RenderRouteGo
	Rest    []*router.RestRouteGo
	Package string
}

func (c *Codegen) GenerateProductionServer(files map[string][]router.Route) error {
	outDir := module.Abs(c.config.Codegen.OutDir)
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

	imports := map[string]string{}
	handlers := map[string]routeHandler{}
	for filename, routes := range files {
		if len(routes) == 0 {
			continue
		}

		if filepath.Ext(filename) != ".go" {
			continue
		}

		basepath := filepath.ToSlash(module.Rel(filepath.Dir(filename)))
		alias := strings.ReplaceAll(basepath, "/", "")
		pkg := path.Join(module.ModuleName(), basepath)
		imports[alias] = pkg

		handler := routeHandler{
			Package: alias,
		}

		for _, route := range routes {
			switch r := route.(type) {
			case *router.RenderRouteGo:
				handler.Render = r

			case *router.RestRouteGo:
				handler.Rest = append(handler.Rest, r)
			}
		}

		handlers[filename] = handler
	}

	err = mainProdServerTempl.Execute(file, map[string]any{
		"IsProd":   true,
		"Imports":  imports,
		"Handlers": handlers,
		"Host":     c.config.Server.Host,
		"Port":     c.config.Server.Port,
	})
	if err != nil {
		return err
	}

	return nil
}
