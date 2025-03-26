package server

import (
	"net/http"
	"sync"

	"github.com/sgq995/nova/internal/logger"
)

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
	logger.Debugf("[server] add %s\n", pattern)
	mr.routes[pattern] = struct{}{}
}

func (mr *memRouter) remove(pattern string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	logger.Debugf("[server] remove %s\n", pattern)
	delete(mr.routes, pattern)
}

func (mr *memRouter) newServeMux(handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	for pattern := range mr.routes {
		logger.Debugf("[server] handle %s\n", pattern)
		mux.Handle(pattern, handler)
	}
	return mux
}
