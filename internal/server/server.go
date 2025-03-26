package server

import (
	_ "embed"
	"net/http"
	"strconv"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
)

//go:embed hmr.js
var hmrJS []byte

type Server struct {
	config *config.Config

	http *http.Server

	hmr *hotModuleReplacer
}

func New(c *config.Config) *Server {
	mux := http.NewServeMux()

	hmr := newHotModuleReplacer(module.Join(c.Codegen.OutDir, "pages"))
	hmr.Send(UpdateFileMessage("@nova/hmr.js", hmrJS))

	nodeModules := module.Join("node_modules", ".nova")
	mux.Handle("/@node_modules/", http.StripPrefix("/@node_modules", http.FileServer(http.Dir(nodeModules))))
	mux.HandleFunc("/@nova/hmr", hmr.serveNovaHMR)
	mux.Handle("/", hmr)

	httpServer := http.Server{
		Addr:    c.Server.Host + ":" + strconv.Itoa(int(c.Server.Port)),
		Handler: mux,
	}

	return &Server{
		config: c,
		http:   &httpServer,
		hmr:    hmr,
	}
}

func (s *Server) Send(msg *Message) {
	s.hmr.Send(msg)
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) Close() error {
	return s.http.Close()
}
