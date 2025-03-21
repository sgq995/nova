package codegen

import (
	"text/template"
)

const renderHandlerFuncTmpl string = `
func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	{{- if .IsProd}}
	sub := must(fs.Sub(templatesFS, root))
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
`

const mainProdTmpl string = `package main

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

const mainDevTmpl string = `package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type memFS struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMemFS() *memFS {
	return &memFS{
		files: map[string][]byte{},
	}
}

func (fsys *memFS) update(filename string, contents []byte) {
	fsys.mu.Lock()
	defer fsys.mu.Unlock()
	fmt.Printf("[main.go] update %s\n", filename)
	fsys.files[filename] = contents
}

func (fsys *memFS) remove(filename string) {
	fsys.mu.Lock()
	defer fsys.mu.Unlock()
	fmt.Printf("[main.go] remove %s\n", filename)
	delete(fsys.files, filename)
}

func (fsys *memFS) Open(name string) (fs.File, error) {
	if f, exists := fsys.files[name]; exists {
		return &memFile{
			name:     name,
			size:     int64(len(f)),
			contents: f,
		}, nil
	}
	return nil, fs.ErrNotExist
}

type memFile struct {
	name     string
	size     int64
	contents []byte
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return &memFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *memFile) Read(out []byte) (int, error) {
	n := copy(out, f.contents)
	f.contents = f.contents[n:]
	return n, nil
}

func (f *memFile) Close() error {
	return nil
}

type memFileInfo struct {
	name string
	size int64
}

func (fi *memFileInfo) Name() string {
	return fi.name
}

func (fi *memFileInfo) Size() int64 {
	return fi.size
}

func (fi *memFileInfo) Mode() fs.FileMode {
	return 0755
}

func (fi *memFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *memFileInfo) IsDir() bool {
	return false
}

func (fi *memFileInfo) Sys() any {
	return nil
}

type memRouter struct {
	mu     sync.Mutex
	routes map[string]struct{}
}

func newMemRouter() *memRouter {
	return &memRouter{
		routes: make(map[string]struct{}),
	}
}

func (mr *memRouter) add(pattern string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	fmt.Printf("[main.go] add %s\n", pattern)
	mr.routes[pattern] = struct{}{}
}

func (mr *memRouter) remove(pattern string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	fmt.Printf("[main.go] remove %s\n", pattern)
	delete(mr.routes, pattern)
}

func (mr *memRouter) newServeMux(handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	for pattern := range mr.routes {
		mux.Handle(pattern, handler)
	}
	return mux
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

type routeModule struct {
	pagespath string
}

func newRouteModule(pagespath string) *routeModule {
	return &routeModule{
		pagespath: pagespath,
	}
}

func (rm *routeModule) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	upath = strings.TrimPrefix(upath, "/api")
	filename := filepath.Join(rm.pagespath, filepath.FromSlash(upath), "main.go")

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.CommandContext(r.Context(), "go", "run", filename)
	cmd.Stdout = stdoutWriter
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(r.Context())

	controller := http.NewResponseController(w)

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
				controller.Flush()
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
}

type pubSub struct {
	mu   sync.Mutex
	subs map[chan string]sync.WaitGroup
}

func newPubSub() *pubSub {
	return &pubSub{
		subs: make(map[chan string]sync.WaitGroup),
	}
}

func (ps *pubSub) notify(filename string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for sub, wg := range ps.subs {
		wg.Add(1)
		go func() {
			sub <- filename
			wg.Done()
		}()
	}
}

func (ps *pubSub) subscribe(sub chan string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if wg, exists := ps.subs[sub]; exists {
		wg.Wait()
	}

	ps.subs[sub] = sync.WaitGroup{}
}

func (ps *pubSub) unsubscribe(sub chan string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if wg, exists := ps.subs[sub]; exists {
		wg.Add(1)
		go func() {
			loop := true
			for loop {
				select {
				case <-sub:
					// we might need to delay a little bit here

				default:
					loop = false
				}
			}
			wg.Done()
		}()
		wg.Wait()
		delete(ps.subs, sub)
		close(sub)
	}
}

type hotModuleReplacer struct {
	fsys        *memFS
	router      *memRouter
	routeModule *routeModule

	ps *pubSub

	mu  sync.Mutex
	mux *http.ServeMux
}

func newHotModuleReplacer(pagespath string) *hotModuleReplacer {
	return &hotModuleReplacer{
		fsys:        newMemFS(),
		router:      newMemRouter(),
		routeModule: newRouteModule(pagespath),
		ps:          newPubSub(),
		mux:         http.NewServeMux(),
	}
}

func (hmr *hotModuleReplacer) generateServeMux() {
	hmr.mu.Lock()
	mux := hmr.router.newServeMux(hmr.routeModule)
	mux.Handle("/", http.FileServerFS(hmr.fsys))
	hmr.mux = mux
	hmr.mu.Unlock()
}

type Message struct {
	Type    int            ` + "`json:\"type\"`" + `
	Payload map[string]any ` + "`json:\"payload\"`" + `
}

func (hmr *hotModuleReplacer) read(r io.Reader) {
	decoder := json.NewDecoder(r)
	var msg Message
	for {
		err := decoder.Decode(&msg)
		if err != nil {
			// TODO: recover
			panic(err)
		}

		switch msg.Type {
		case 0: // UpdateFileType
			filename := msg.Payload["filename"].(string)
			contents, _ := base64.StdEncoding.DecodeString(msg.Payload["contents"].(string))
			hmr.fsys.update(filename, contents)
			hmr.generateServeMux()
			hmr.ps.notify(filename)

		case 1: // DeleteFileType
			filename := msg.Payload["filename"].(string)
			hmr.fsys.remove(filename)
			hmr.generateServeMux()
			hmr.ps.notify(filename)

		case 2: // CreateRouteType
			pattern := msg.Payload["pattern"].(string)
			hmr.router.add(pattern)
			hmr.generateServeMux()
			// TODO: notify ServeNovaHMR

		// TODO: udate route, just need to notify ServeNovaHMR
		// NOTE: DO NOT generate server mux, it's not needed because the pattern is
		//       registered already

		case 3: // DeleteRouteType
			pattern := msg.Payload["pattern"].(string)
			hmr.router.remove(pattern)
			hmr.generateServeMux()
			// TODO: notify ServeNovaHMR
		}
	}
}

func (hmr *hotModuleReplacer) serveNovaHMR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string)
	hmr.ps.subscribe(ch)
	defer hmr.ps.unsubscribe(ch)

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
}

func (hmr *hotModuleReplacer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hmr.mu.Lock()
	mux := hmr.mux
	hmr.mu.Unlock()

	mux.ServeHTTP(w, r)
}

func main() {
	mux := http.NewServeMux()

	hmr := newHotModuleReplacer(filepath.Join("{{.OutDir}}", "pages"))
	go hmr.read(os.Stdin)

	mux.Handle("/@node_modules/", http.StripPrefix("/@node_modules", http.FileServer(http.Dir("{{.NodeModules}}"))))
	mux.HandleFunc("/@nova/hmr", hmr.serveNovaHMR)
	mux.Handle("/", hmr)

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

func generateProdServerTemplate() *template.Template {
	mainTemplate := template.Must(template.New("main.go").Parse(mainProdTmpl))
	template.Must(mainTemplate.New("renderHandler").Parse(renderHandlerFuncTmpl))
	return mainTemplate
}

func generateDevServerTemplate() *template.Template {
	mainTemplate := template.Must(template.New("main.go").Parse(mainDevTmpl))
	return mainTemplate
}
