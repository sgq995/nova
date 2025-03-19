package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/module"
)

const mainModule string = `package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	
	{{.Alias}} "{{.Package}}"
)

type request struct {
	Method string ` + "`json:\"method\"`" + `

	RawURL string ` + "`json:\"rawUrl\"`" + `

	Proto      string ` + "`json:\"proto\"`" + `
	ProtoMajor int    ` + "`json:\"protoMajor\"`" + `
	ProtoMinor int    ` + "`json:\"protoMinor\"`" + `

	Header http.Header ` + "`json:\"headers\"`" + `

	// Can be Stdin
	// Body []byte ` + "`json:\"body\"`" + `

	ContentLength int64 ` + "`json:\"contentLength\"`" + `

	Host string ` + "`json:\"host\"`" + `

	// TODO:
	// Trailer http.Header

	RemoteAddr string ` + "`json:\"remoteAddr\"`" + `

	RequestURI string ` + "`json:\"requestUri\"`" + `

	Pattern string ` + "`json:\"pattern\"`" + `
}

func transformRequest(r *request) (*http.Request, error) {
	u, err := url.Parse(r.RawURL)
	if err != nil {
		return nil, err
	}
	return &http.Request{
		Method: r.Method,

		URL: u,

		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,

		Header: r.Header,

		Body: os.Stdin,

		ContentLength: r.ContentLength,

		Host: r.Host,

		RemoteAddr: r.RemoteAddr,

		RequestURI: r.RequestURI,

		Pattern: r.Pattern,
	}, nil
}

type responseWriter struct {
	wroteHeader bool

	Headers    http.Header ` + "`json:\"headers\"`" + `
	StatusCode int         ` + "`json:\"statusCode\"`" + `
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		Headers: make(http.Header),
	}
}

func (w *responseWriter) Header() http.Header {
	return w.Headers
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return os.Stdout.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.StatusCode = statusCode
	json.NewEncoder(os.Stdout).Encode(w)
}

{{template "renderHandler" .}}

func main() {
	var jsonReq request
	json.NewDecoder(os.Stdin).Decode(&jsonReq)
	r, err := transformRequest(&jsonReq)
	if err != nil {
		panic(err)
	}

	w := newResponseWriter()

	mux := http.NewServeMux()
	{{range $handler := .Handlers}}
	{{with $render := .Render}}mux.Handle("{{$render.Pattern}}", renderHandler("{{$render.Root}}", []string{ {{- range $render.Templates}}"{{.}}", {{end -}} }, {{$handler.Package}}.{{$render.Handler}})){{end}}
	{{range .Rest}}mux.HandleFunc("{{.Pattern}}", {{$handler.Package}}.{{.Handler}}){{end}}
	{{end}}

	mux.ServeHTTP(w, r)
}
`

var mainModuleTmpl *template.Template = generateHMRTemplate()

func generateHMRTemplate() *template.Template {
	hmrTemplate := template.Must(template.New("main.go").Parse(mainModule))
	template.Must(hmrTemplate.New("renderHandler").Parse(renderHandlerFuncTmpl))
	return hmrTemplate
}

func (c *Codegen) GenerateModule(filename string) error {
	pagespath := module.Abs(c.config.Router.Pages)

	basepath := filepath.ToSlash(module.Rel(filepath.Dir(filename)))
	alias := strings.ReplaceAll(basepath, "/", "")
	pkg := path.Join(module.ModuleName(), basepath)

	os.MkdirAll(filepath.Dir(filename), 0755)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = mainModuleTmpl.Execute(file, map[string]any{
		"Alias":   alias,
		"Package": pkg,
		"Root":    pagespath,
	})
	if err != nil {
		return err
	}

	return nil
}
