package generator

var mainFile string = `package main

import (
	"html/template"
	"log"	
	"net/http"
)

{{ range .Imports }}
import {{ .Alias }} "{{ .Path }}"
{{ end }}

{{ .Embed }}

type Router struct {
	*http.ServeMux
}

func NewRouter() *Router {
	return &Router{http.NewServeMux()}
}

func RenderHandler(render func(*template.Template, http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render(nil, w, r)
	})
}

func main() {
	r := NewRouter()

	{{ range .SSRRoutes }}
	r.Handle("{{ .Path }}", RenderHandler({{ .Handler }}))
	{{ end }}

	{{ range .APIRoutes }}
	r.HandleFunc("{{ .Path }}", {{ .Handler }})
	{{ end }}

	{{ .Static }}

	s := http.Server{
		Addr:    "{{ .Host }}:{{ .Port }}",
		Handler: r,
	}
	log.Fatalln(s.ListenAndServe())
}
`
