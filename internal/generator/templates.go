package generator

var mainFile string = `package main

{{range .Imports -}}
import {{.Alias}} "{{.Path}}"
{{end}}

type Router struct {
	*http.ServeMux
}

func NewRouter() *Router {
	return &Router{http.NewServeMux()}
}

{{if eq .Environment "prod"}}
//go:embed templates
var templatesFS embed.FS
{{end}}

func loadTemplate(root string, tmpls []string) *template.Template {
	if len(tmpls) == 0 {
		return nil
	}

	{{if eq .Environment "prod"}}
	sub, err := fs.Sub(templatesFS, path.Join("templates", root))
	if err != nil {
		panic(err)
	}
	{{else}}
	root = filepath.Join("src", "pages", root)
	filenames := []string{}
	for _, tmpl := range tmpls {
		filename := filepath.Join(root, tmpl)
		filenames = append(filenames, filename)
	}
	{{end}}

	t := template.New(tmpls[0])
	{{if eq .Environment "prod"}}
	t = template.Must(t.ParseFS(sub, tmpls...))
	{{else}}
	t = template.Must(t.ParseFiles(filenames...))
	{{end}}
	return t
}

func RenderHandler(root string, tmpls []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	{{if eq .Environment "prod"}}t := loadTemplate(root, tmpls){{end}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		{{if eq .Environment "dev"}}t := loadTemplate(root, tmpls){{end}}
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

{{range .Middlewares}}
{{.Decl}}
{{end}}

func main() {
	r := NewRouter()

	{{range .SSRRoutes -}}
	r.Handle("{{.Path}}", RenderHandler("{{ .Root }}", []string{ {{range .Templates}}"{{.}}", {{end}} }, {{.Handler}}))
	{{end}}

	{{range .APIRoutes -}}
	r.HandleFunc("{{.Path}}", {{.Handler}})
	{{end}}

	{{.Static}}

	var h http.Handler = r
	{{- range .Middlewares}}
	h = {{.Name}}(h{{range .Args -}} , {{. -}} {{end}})
	{{- end}}

	s := http.Server{
		Addr:    "{{.Host}}:{{.Port}}",
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

var staticMiddlewareDecl = `//go:embed static
var staticFS embed.FS

type fileServerResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *fileServerResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	if statusCode != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *fileServerResponseWriter) Write(content []byte) (int, error) {
	if w.statusCode != http.StatusNotFound {
		return w.ResponseWriter.Write(content)
	}
	return len(content), nil
}

func withFileServer(next http.Handler) http.Handler {
	fs := http.FileServerFS(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &fileServerResponseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		if rw.statusCode == http.StatusNotFound {
			clear(w.Header())
			fs.ServeHTTP(w, r)
		}
	})
}`
