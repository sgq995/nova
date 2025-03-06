package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
)

const mainTmpl string = `package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	{{range $alias, $package := .Imports}}
	{{$alias}} "{{$package}}"{{end}}
)

{{if not .IsProd}}
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
		log.Println("[main.go]", line)

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
{{end}}

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
	{{if not .IsProd}}
	ch := make(chan string)
	fsys := &esbuildFS{files: make(map[string][]byte)}
	go esbuildScanner(fsys, ch)
	mux.Handle("/", http.FileServerFS(fsys))
	mux.Handle("/@node_modules/", http.StripPrefix("/@node_modules", http.FileServer(http.Dir("/home/sebastian/Proyectos/Personal/nova/node_modules/.nova"))))
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
		"IsProd":      isProd,
		"Imports":     imports,
		"Root":        pagespath,
		"Handlers":    handlers,
		"Host":        c.Server.Host,
		"Port":        c.Server.Port,
		"NodeModules": module.Abs(filepath.Join("node_modules", ".nova")),
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
