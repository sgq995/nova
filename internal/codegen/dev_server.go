package codegen

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/sgq995/nova/internal/module"
)

var mainDevServer string = `package main

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
	"strings"
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
		fmt.Printf("[main.go] handle %s\n", pattern)
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

func (hmr *hotModuleReplacer) bulk(payload map[string]any) (created []string, updated []string, deleted []string, routes []string) {
	messages := payload["messages"].([]any)	
	for _, msg := range messages {
		msg := msg.(map[string]any)
		typ := int(msg["type"].(float64))
		payload := msg["payload"].(map[string]any)

		switch typ {
		case ` + CreateFileType.Itoa() + `: // ` + CreateFileType.String() + `
			filename := hmr.createFile(payload)
			created = append(created, filename)

		case ` + UpdateFileType.Itoa() + `: // ` + UpdateFileType.String() + `
			filename := hmr.updateFile(payload)
			updated = append(updated, filename)
			
		case ` + DeleteFileType.Itoa() + `: // ` + DeleteFileType.String() + `
			filename := hmr.deleteFile(payload)
			deleted = append(deleted, filename)
			
		case ` + CreateRouteType.Itoa() + `: // ` + CreateRouteType.String() + `
			pattern := hmr.createRoute(payload)
			routes = append(routes, pattern)

		// TODO: udate route, just need to notify ServeNovaHMR
		// NOTE: DO NOT generate server mux, it's not needed because the pattern is
		//       registered already

		case ` + DeleteRouteType.Itoa() + `: // ` + DeleteRouteType.String() + `
			pattern := hmr.deleteRoute(payload)
			routes = append(routes, pattern)
		}
	}
	return
}

func (hmr *hotModuleReplacer) createFile(payload map[string]any) string {
	filename := payload["filename"].(string)
	contents, _ := base64.StdEncoding.DecodeString(payload["contents"].(string))
	hmr.fsys.update(filename, contents)
	return filename
}

func (hmr *hotModuleReplacer) updateFile(payload map[string]any) string {
	filename := payload["filename"].(string)
	contents, _ := base64.StdEncoding.DecodeString(payload["contents"].(string))
	hmr.fsys.update(filename, contents)
	return filename
}

func (hmr *hotModuleReplacer) deleteFile(payload map[string]any) string {
	filename := payload["filename"].(string)
	hmr.fsys.remove(filename)
	return filename
}

func (hmr *hotModuleReplacer) createRoute(payload map[string]any) string {
	pattern := payload["pattern"].(string)
	hmr.router.add(pattern)
	return pattern
}

func (hmr *hotModuleReplacer) deleteRoute(payload map[string]any) string {
	pattern := payload["pattern"].(string)
	hmr.router.remove(pattern)
	return pattern
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
		case ` + BulkType.Itoa() + `: // ` + BulkType.String() + `
			created, updated, deleted, routes := hmr.bulk(msg.Payload)
			
			createdNotification := "\"created\":[]"
			if len(created) > 0 {
				createdNotification = fmt.Sprintf("\"created\": [\"%s\"]", strings.Join(created, "\",\""))
			}

			updatedNotification := "\"updated\":[]"
			if len(updated) > 0 {
				updatedNotification = fmt.Sprintf("\"updated\": [\"%s\"]", strings.Join(updated, "\",\""))
			}

			deletedNotification := "\"deleted\":[]"
			if len(deleted) > 0 {
				deletedNotification = fmt.Sprintf("\"deleted\": [\"%s\"]", strings.Join(deleted, "\",\""))
			}

			data := "{" + createdNotification + "," + updatedNotification + "," + deletedNotification + "}"
			hmr.ps.notify(data)

			if len(routes) > 0 {
				hmr.generateServeMux()
			}

		case ` + CreateFileType.Itoa() + `: // ` + CreateFileType.String() + `
			filename := hmr.createFile(msg.Payload)
			data := fmt.Sprintf("{ \"created\": [\"%s\"] }", filename)
			hmr.ps.notify(data)

		case ` + UpdateFileType.Itoa() + `: // ` + UpdateFileType.String() + `
			filename := hmr.updateFile(msg.Payload)
			data := fmt.Sprintf("{ \"updated\": [\"%s\"] }", filename)
			hmr.ps.notify(data)

		case ` + DeleteFileType.Itoa() + `: // ` + DeleteFileType.String() + `
			filename := hmr.deleteFile(msg.Payload)
			data := fmt.Sprintf("{ \"deleted\": [\"%s\"] }", filename)
			hmr.ps.notify(data)

		case ` + CreateRouteType.Itoa() + `: // ` + CreateRouteType.String() + `
			hmr.createRoute(msg.Payload)
			hmr.generateServeMux()
			// TODO: notify ServeNovaHMR

		// TODO: udate route, just need to notify ServeNovaHMR
		// NOTE: DO NOT generate server mux, it's not needed because the pattern is
		//       registered already

		case ` + DeleteRouteType.Itoa() + `: // ` + DeleteRouteType.String() + `
			hmr.deleteRoute(msg.Payload)
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

		case data := <-ch:
			_, err := fmt.Fprintf(w, "event: change\ndata: %s\n\n", data)
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

var mainDevServerTempl *template.Template = newDevServerTemplate()

func newDevServerTemplate() *template.Template {
	mainTemplate := template.Must(template.New("main.go").Parse(mainDevServer))
	return mainTemplate
}

func (c *Codegen) GenerateDevelopmentServer() error {
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

	pagespath := module.Abs(c.config.Router.Pages)
	nodemodules := module.Join("node_modules", ".nova")

	err = mainDevServerTempl.Execute(file, map[string]any{
		"IsProd":      false,
		"Root":        pagespath,
		"OutDir":      outDir,
		"Host":        c.config.Server.Host,
		"Port":        c.config.Server.Port,
		"NodeModules": nodemodules,
	})
	if err != nil {
		return err
	}

	return nil
}
