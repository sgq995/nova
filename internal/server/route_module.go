package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type request struct {
	Method string `json:"method"`

	RawURL string `json:"rawUrl"`

	Proto      string `json:"proto"`
	ProtoMajor int    `json:"protoMajor"`
	ProtoMinor int    `json:"protoMinor"`

	Header http.Header `json:"headers"`

	ContentLength int64 `json:"contentLength"`

	Host string `json:"host"`

	RemoteAddr string `json:"remoteAddr"`

	RequestURI string `json:"requestUri"`

	Pattern string `json:"pattern"`
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
	Headers    http.Header `json:"headers"`
	StatusCode int         `json:"statusCode"`
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
