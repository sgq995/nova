package project

import (
	"context"

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
	fs, err := scan(p.config.Pages)
	if err != nil {
		return nil, err
	}

	_, err = router.NewRouter(p.config.Pages, fs.pages)
	if err != nil {
		return nil, err
	}

	// proxyUrl := BundlerServe(files)
	// Watch()
	// -> files := Scan(pages)
	// -> Link(files)
	// -> Bundle(files)
	// -> router := CreateRouter(files, WithProxy(proxyUrl))
	// -> Execute(mainTemplate, router)
	// -> Serve()

	// p.scan(p.config.Pages)

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
