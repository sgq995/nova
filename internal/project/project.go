package project

import (
	"context"
	"log"

	"github.com/sgq995/nova/internal/codegen"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/router"
)

type Server interface {
	Dispose()
}

type ProjectContext interface {
	Serve(ctx context.Context) (Server, error)

	Build() error
}

func Context(c config.Config) (ProjectContext, error) {
	// Scan should Link as those are correlated
	// files := Scan(pages)
	// Link(files)

	// CreateRouter and Exec should be merged into Generate
	// router := CreateRouter(files, WithProxy(proxyUrl))
	// Execute(mainTemplate, router)

	return &projectContextImpl{config: &c}, nil
}

type serverImpl struct {
	scanner *scanner
	router  *router.Router
	codegen *codegen.Codegen
	runner  *runner
}

func (s *serverImpl) Dispose() {
	s.runner.stop()
}

type projectContextImpl struct {
	config *config.Config
}

func (p *projectContextImpl) Serve(ctx context.Context) (Server, error) {
	scanner := newScanner(p.config)
	// TODO: esbuild
	router := router.NewRouter(p.config)
	codegen := codegen.NewCodegen(p.config)
	runner := newRunner(p.config)

	// err := scanner.scan()
	// if err != nil {
	// 	return nil, err
	// }
	// Generate(p.config, router.ParseRoutes(scanner.pages))
	// if err != nil {
	// 	return nil, err
	// }

	watcher := newWatcher(ctx, map[string]func(string){
		"*.go": func(filename string) {
			runner.stop()
			runner.create()

			err := scanner.scan()
			if err != nil {
				log.Println("[scanner]", err)
				return
			}
			routes, err := router.ParseRoutes(scanner.pages)
			if err != nil {
				log.Println("[router]", err)
				return
			}
			err = codegen.Generate(routes)
			if err != nil {
				log.Println("[codegen]", err)
			}
			log.Println("[reload]", filename)

			runner.start()
		},
		"*.js": func(s string) {
			// TODO: esbuild
			// TODO: runner.update
		},
	})

	go watcher.watch(p.config.Router.Pages)

	// proxyUrl := BundlerServe(files)
	// Watch()
	// -> files := Scan(pages)
	// -> Link(files)
	// -> Bundle(files)
	// -> router := CreateRouter(files, WithProxy(proxyUrl))
	// -> Execute(mainTemplate, router)
	// -> Serve()

	return &serverImpl{
		scanner: scanner,
		router:  router,
		codegen: codegen,
		runner:  runner,
	}, nil
}

func (*projectContextImpl) Build() error {
	// build
	// files := Scan(c.Pages)
	// link with file handlers,
	// html will search for script and link,
	// js and css are automatically link by esbuild
	// assets will only be referenced
	// Link(files)
	// Bundle()
	// CreateRouter()
	// Execute(mainTemplate)
	// Build()

	return nil
}
