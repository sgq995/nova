package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type pubSub struct {
	mu   sync.Mutex
	subs map[chan string]*sync.WaitGroup
}

func newPubSub() *pubSub {
	return &pubSub{
		subs: make(map[chan string]*sync.WaitGroup),
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

	ps.subs[sub] = &sync.WaitGroup{}
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

func (hmr *hotModuleReplacer) bulk(payload map[string]any) (created []string, updated []string, deleted []string, routes []string) {
	messages := payload["messages"].([]*Message)
	for _, msg := range messages {
		// msg := msg.(map[string]any)
		// typ := int(msg["type"].(float64))
		// payload := msg["payload"].(map[string]any)
		typ := msg.Type
		payload := msg.Payload

		switch typ {
		case CreateFileType: // CreateFileType.String()
			filename := hmr.createFile(payload)
			created = append(created, filename)

		case UpdateFileType: // ` + UpdateFileType.String() + `
			filename := hmr.updateFile(payload)
			updated = append(updated, filename)

		case DeleteFileType: // ` + DeleteFileType.String() + `
			filename := hmr.deleteFile(payload)
			deleted = append(deleted, filename)

		case CreateRouteType: // ` + CreateRouteType.String() + `
			pattern := hmr.createRoute(payload)
			routes = append(routes, pattern)

		// TODO: udate route, just need to notify ServeNovaHMR
		// NOTE: DO NOT generate server mux, it's not needed because the pattern is
		//       registered already

		case DeleteRouteType: // ` + DeleteRouteType.String() + `
			pattern := hmr.deleteRoute(payload)
			routes = append(routes, pattern)
		}
	}
	return
}

func (hmr *hotModuleReplacer) createFile(payload map[string]any) string {
	filename := payload["filename"].(string)
	// contents, _ := base64.StdEncoding.DecodeString(payload["contents"].(string))
	contents := payload["contents"].([]byte)
	hmr.fsys.update(filename, contents)
	return filename
}

func (hmr *hotModuleReplacer) updateFile(payload map[string]any) string {
	filename := payload["filename"].(string)
	// contents, _ := base64.StdEncoding.DecodeString(payload["contents"].(string))
	contents := payload["contents"].([]byte)
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

func (hmr *hotModuleReplacer) Send(msg *Message) {
	switch msg.Type {
	case BulkType: // ` + BulkType.String() + `
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

	case CreateFileType: // ` + CreateFileType.String() + `
		filename := hmr.createFile(msg.Payload)
		data := fmt.Sprintf("{ \"created\": [\"%s\"] }", filename)
		hmr.ps.notify(data)

	case UpdateFileType: // ` + UpdateFileType.String() + `
		filename := hmr.updateFile(msg.Payload)
		data := fmt.Sprintf("{ \"updated\": [\"%s\"] }", filename)
		hmr.ps.notify(data)

	case DeleteFileType: // ` + DeleteFileType.String() + `
		filename := hmr.deleteFile(msg.Payload)
		data := fmt.Sprintf("{ \"deleted\": [\"%s\"] }", filename)
		hmr.ps.notify(data)

	case CreateRouteType: // ` + CreateRouteType.String() + `
		hmr.createRoute(msg.Payload)
		hmr.generateServeMux()
		// TODO: notify ServeNovaHMR

	// TODO: udate route, just need to notify ServeNovaHMR
	// NOTE: DO NOT generate server mux, it's not needed because the pattern is
	//       registered already

	case DeleteRouteType: // ` + DeleteRouteType.String() + `
		hmr.deleteRoute(msg.Payload)
		hmr.generateServeMux()
		// TODO: notify ServeNovaHMR
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
