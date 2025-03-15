package codegen

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
	"github.com/sgq995/nova/internal/utils"
)

const mainTmpl string = `package main

import (
	"io/fs"
	"log"
	"net/http"
{{- if .IsProd}}
	"html/template"
	"embed"
	"path"
{{- else}}
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
{{end}}
{{if .IsProd -}}
	{{range $alias, $package := .Imports}}
	{{$alias}} "{{$package}}"{{end}}
{{end -}}
)

{{if .IsProd}}
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

func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	{{if .IsProd -}}
	sub, err := fs.Sub(templatesFS, root)
	if err != nil {
		panic(err)
	}
	t := template.Must(template.ParseFS(sub, templates...))
	{{- end}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		{{if not .IsProd -}}
		fs := os.DirFS(filepath.Join("{{.Root}}", root))
		t := template.Must(template.ParseFS(fs, templates...)){{end}}
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
{{else}}
type esbuildFS struct {
	files map[string][]byte
}

func (fsys *esbuildFS) Open(name string) (fs.File, error) {
	if f, exists := fsys.files[name]; exists {
		return &esbuildFile{
			name:     name,
			size:     int64(len(f)),
			contents: f,
		}, nil
	}
	return nil, fs.ErrNotExist
}

type esbuildFile struct {
	name     string
	size     int64
	contents []byte
}

func (f *esbuildFile) Stat() (fs.FileInfo, error) {
	return &esbuildFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *esbuildFile) Read(out []byte) (int, error) {
	n := copy(out, f.contents)
	f.contents = f.contents[n:]
	return n, nil
}

func (f *esbuildFile) Close() error {
	return nil
}

type esbuildFileInfo struct {
	name string
	size int64
}

func (fi *esbuildFileInfo) Name() string {
	return fi.name
}

func (fi *esbuildFileInfo) Size() int64 {
	return fi.size
}

func (fi *esbuildFileInfo) Mode() fs.FileMode {
	return 0755
}

func (fi *esbuildFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *esbuildFileInfo) IsDir() bool {
	return false
}

func (fi *esbuildFileInfo) Sys() any {
	return nil
}

func esbuildScanner(fsys *esbuildFS, ch chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		line = strings.TrimSpace(line)

		parts := strings.Split(line, " ")
		if len(parts) != 3 {
			panic("[main.go] wrong format " + line)
		}

		command := parts[0]
		filename := parts[1]
		length, err := strconv.Atoi(parts[2])
		if err != nil {
			panic(err)
		}

		contents := make([]byte, length)
		_, err = io.ReadFull(reader, contents)
		if err != nil {
			panic(err)
		}
		fmt.Println("[main.go]", line)

		switch command {
		case "UPDATE":
			fsys.files[filename] = contents
			select {
			case ch <- filename:

			default:
			}
		}
	}
}

func hmr(ch chan string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		rc := http.NewResponseController(w)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return

			case filename := <-ch:
				_, err := fmt.Fprintf(w, "event: change\ndata: { \"updated\": [\"%s\"] }\n\n", filename)
				if err != nil {
					return
				}
				err = rc.Flush()
				if err != nil {
					return
				}
			}
		}
	})
}

type request struct {
	Method string ` + "`json:\"method\"`" + `

	RawURL string ` + "`json:\"rawUrl\"`" + `

	Proto      string ` + "`json:\"proto\"`" + `
	ProtoMajor int    ` + "`json:\"protoMajor\"`" + `
	ProtoMinor int    ` + "`json:\"protoMinor\"`" + `

	Header http.Header ` + "`json:\"headers\"`" + `

	ContentLength int64 ` + "`json:\"contentLength\"`" + `

	Host string ` + "`json:\"host\"`" + `

	RemoteAddr string ` + "`json:\"remoteAddr\"`" + `

	RequestURI string ` + "`json:\"requestUri\"`" + `

	Pattern string ` + "`json:\"pattern\"`" + `
}

func transformRequest(r *http.Request) *request {
	return &request{
		Method: r.Method,

		RawURL: r.URL.String(),

		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,

		Header: r.Header,

		ContentLength: r.ContentLength,

		Host: r.Host,

		RemoteAddr: r.RemoteAddr,

		RequestURI: r.RequestURI,

		Pattern: r.Pattern,
	}
}

type responseWriter struct {
	Headers    http.Header ` + "`json:\"headers\"`" + `
	StatusCode int         ` + "`json:\"statusCode\"`" + `
}

func hmrModule(path string) http.Handler {
	basepath := filepath.Join("{{.OutDir}}", "pages", filepath.FromSlash(path))
	filename := filepath.Join(basepath, "main.go")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// var stdout bytes.Buffer
		stdoutReader, stdoutWriter, err := os.Pipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cmd := exec.CommandContext(r.Context(), "go", "run", filename)
		// cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		// cmd.Stdout = &stdout
		cmd.Stdout = stdoutWriter

		stdin, err := cmd.StdinPipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(r.Context())

		wg.Add(1)
		go func() {
			defer wg.Done()

			request := transformRequest(r)
			encoder := json.NewEncoder(stdin)
			if err := encoder.Encode(request); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			_, err := io.Copy(stdin, r.Body)
			if err != nil {
				return
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			var header bytes.Buffer
			var msg []byte
			buf := make([]byte, 1024)

			for {
				n, err := stdoutReader.Read(buf)
				if err != nil && !errors.Is(err, io.EOF) {
					return
				}
				if n > 0 {
					msg = buf[:n]
					if msg[0] != '{' {
						http.Error(w, "bad data: "+string(msg), http.StatusInternalServerError)
						return
					}
					var count int
					for _, b := range msg {
						header.WriteByte(b)

						if b == '{' {
							count++
						} else if b == '}' {
							count--
						} else if b == '\n' && count == 0 {
							break
						}
					}

					break
				}

				time.Sleep(time.Millisecond)
			}

			body := msg[header.Len():]

			var response responseWriter
			decoder := json.NewDecoder(&header)
			err := decoder.Decode(&response)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			maps.Copy(w.Header(), response.Headers)

			if response.StatusCode != 0 {
				w.WriteHeader(response.StatusCode)
			}

			if len(body) > 0 {
				w.Write(body)
			}

			for {
				n, err := stdoutReader.Read(buf)
				if err != nil && !errors.Is(err, io.EOF) {
					return
				}
				if n > 0 {
					w.Write(buf[:n])
				}

				select {
				case <-ctx.Done():
					return

				default:
					time.Sleep(time.Millisecond)
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			err := cmd.Start()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cmd.Wait()

			stdoutWriter.Close()
			stdoutReader.Close()

			cancel()
		}()

		wg.Wait()
	})
}
{{end}}

func main() {
	mux := http.NewServeMux()
	{{- $global := . -}}
	{{range $filename, $handler := .Handlers}}
	// {{$filename}}
	{{if $global.IsProd -}}
	{{with $render := $handler.Render}}mux.Handle("{{$render.Pattern}}", renderHandler("{{$render.Root}}", []string{ {{- range $render.Templates}}"{{.}}", {{end -}} }, {{$handler.Package}}.{{$render.Handler}})){{end}}
	{{range $handler.Rest}}mux.HandleFunc("{{.Pattern}}", {{$handler.Package}}.{{.Handler}}){{end}}
	{{- else -}}
	{{with $render := $handler.Render}}mux.Handle("{{$render.Pattern}}", hmrModule("{{$handler.Path}}")){{end}}
	{{range $handler.Rest}}mux.Handle("{{.Pattern}}", hmrModule("{{$handler.Path}}")){{end}}
	{{- end -}}
	{{end}}

	// nova
	{{- if .IsProd}}
	mux.Handle("/static/", http.FileServerFS(staticFS))
	mux.Handle("/", http.FileServerFS(pagesFS))
	{{else}}
	ch := make(chan string, 16)
	fsys := &esbuildFS{files: make(map[string][]byte)}
	go esbuildScanner(fsys, ch)
	mux.Handle("/", http.FileServerFS(fsys))
	mux.Handle("/@node_modules/", http.StripPrefix("/@node_modules", http.FileServer(http.Dir("{{.NodeModules}}"))))
	mux.Handle("/@nova/hmr", hmr(ch))
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

const mainHMRTmpl string = `package main

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

func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs := os.DirFS(filepath.Join("{{.Root}}", root))
		t := template.Must(template.ParseFS(fs, templates...))
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

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

type handler struct {
	Render  *router.RenderRouteGo
	Rest    []*router.RestRouteGo
	Package string
	Path    string
}

func generateMain(c *config.Config, files map[string][]router.Route) error {
	imports := map[string]string{}
	handlers := map[string]handler{}
	pages := filepath.Join(module.Root(), c.Router.Pages)
	for filename, routes := range files {
		if len(routes) == 0 {
			continue
		}

		if filepath.Ext(filename) != ".go" {
			continue
		}

		base, _ := filepath.Rel(module.Root(), filepath.Dir(filename))
		base = filepath.ToSlash(base)
		alias := strings.ReplaceAll(base, "/", "")
		pkg := path.Join(module.ModuleName(), base)
		imports[alias] = pkg
		handlerPath, _ := filepath.Rel(pages, filename)
		handlerPath = filepath.Dir(handlerPath)

		h := handler{
			Rest:    make([]*router.RestRouteGo, 0),
			Package: alias,
			Path:    handlerPath,
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
	pagespath := module.Abs(filepath.FromSlash(c.Router.Pages))
	outDir := filepath.Join(module.Root(), c.Codegen.OutDir)

	errs := []error{}
	if !isProd {
		hmrFiles := map[string][]handler{}
		aliases := map[string]string{}
		packages := map[string]string{}
		for filename, handler := range handlers {
			base := utils.Must(filepath.Rel(module.Root(), filepath.Dir(filename)))
			base = filepath.ToSlash(base)
			alias := strings.ReplaceAll(base, "/", "")
			pkg := path.Join(module.ModuleName(), base)
			imports[alias] = pkg

			basepath := utils.Must(filepath.Rel(pagespath, filepath.Dir(filename)))
			mainfile := filepath.Join(outDir, "pages", basepath, "main.go")

			aliases[mainfile] = alias
			packages[mainfile] = pkg

			handlers := hmrFiles[mainfile]
			handlers = append(handlers, handler)
			hmrFiles[mainfile] = handlers
		}

		t := template.Must(template.New("main.go").Parse(mainHMRTmpl))
		for filename, handlers := range hmrFiles {
			os.MkdirAll(filepath.Dir(filename), 0755)
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer file.Close()

			errs = append(errs, t.Execute(file, map[string]any{
				"Alias":    aliases[filename],
				"Package":  packages[filename],
				"Root":     pagespath,
				"Handlers": handlers,
			}))
		}
	}

	out := filepath.Join(outDir, "main.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.New("main.go").Parse(mainTmpl))
	errs = append(errs, t.Execute(file, map[string]any{
		"IsProd":      isProd,
		"Imports":     imports,
		"Root":        pagespath,
		"OutDir":      outDir,
		"Handlers":    handlers,
		"Host":        c.Server.Host,
		"Port":        c.Server.Port,
		"NodeModules": module.Abs(filepath.Join("node_modules", ".nova")),
	}))

	return errors.Join(errs...)
}

type Codegen struct {
	config *config.Config
}

func NewCodegen(c *config.Config) *Codegen {
	return &Codegen{config: c}
}

func (c *Codegen) Generate(files map[string][]router.Route) error {
	outDir := filepath.Join(module.Root(), c.config.Codegen.OutDir)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	err = generateMain(c.config, files)
	if err != nil {
		return err
	}

	return nil
}
