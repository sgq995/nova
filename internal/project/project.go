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

type projectContextImpl struct {
	config *config.Config
}

func (p *projectContextImpl) Serve(ctx context.Context) (Server, error) {
	// dev

	// files := Scan(pages)
	// Link(files)
	scanner := newScanner()
	err := scanner.scan(p.config.Router.Pages)
	if err != nil {
		return nil, err
	}

	// TODO: esbuild

	r, err := router.NewRouter(p.config, scanner.pages)
	if err != nil {
		return nil, err
	}

	err = codegen.Generate(p.config, r)
	if err != nil {
		return nil, err
	}

	watcher := newWatcher(ctx, map[string]func(string){
		"*.go": func(filename string) {
			err := scanner.scan(p.config.Router.Pages)
			if err != nil {
				log.Println("[scanner]", err)
				return
			}
			r, err := router.NewRouter(p.config, scanner.pages)
			if err != nil {
				log.Println("[router]", err)
				return
			}
			err = codegen.Generate(p.config, r)
			if err != nil {
				log.Println("[codegen]", err)
			}
			log.Println("[reload]", filename)
		},
		"*.js": func(s string) {
			// TODO: esbuild
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

	return nil, nil
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
