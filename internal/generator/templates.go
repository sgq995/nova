package generator

var mainFile string = `package main

import (
	"html/template"
	"log"	
	"net/http"
)

{{ range .Imports -}}
import {{ .Alias }} "{{ .Path }}"
{{ end }}

{{ range .Middlewares -}}
	{{- range .Imports }}
import {{ .Alias }} "{{ .Path }}"
	{{- end }}
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

{{ range .Middlewares }}
{{ .Decl }}
{{ end }}

func main() {
	r := NewRouter()

	{{ range .SSRRoutes -}}
	r.Handle("{{ .Path }}", RenderHandler({{ .Handler }}))
	{{ end }}

	{{ range .APIRoutes -}}
	r.HandleFunc("{{ .Path }}", {{ .Handler }})
	{{ end }}

	{{ .Static }}

	var h http.Handler = r
	{{- range .Middlewares }}
	h = {{ .Name }}(h{{ range .Args -}} , {{ . -}} {{ end }})
	{{- end }}

	s := http.Server{
		Addr:    "{{ .Host }}:{{ .Port }}",
		Handler: h,
	}
	log.Fatalln(s.ListenAndServe())
}
`

var proxyMiddlewareDecl = `type reverseProxyResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *reverseProxyResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	if statusCode != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *reverseProxyResponseWriter) Write(content []byte) (int, error) {
	if w.statusCode != http.StatusNotFound {
		return w.ResponseWriter.Write(content)
	}
	return len(content), nil
}

func withReverseProxy(next http.Handler, target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalln(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &reverseProxyResponseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		if rw.statusCode == http.StatusNotFound {
			clear(w.Header())
			proxy.ServeHTTP(w, r)
		}
	})
}`
